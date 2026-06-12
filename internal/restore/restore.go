package restore

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path"
	"strings"
	"sync"
	"time"

	mem "github.com/lib-x/nowledgemem-go"

	"github.com/ca-x/nowledge-mem-snap/internal/archive"
	"github.com/ca-x/nowledge-mem-snap/internal/config"
	"github.com/ca-x/nowledge-mem-snap/internal/storage"
)

const (
	StateQueued      = "queued"
	StateDownloading = "downloading"
	StateDecrypting  = "decrypting"
	StateUploading   = "uploading"
	StateImporting   = "importing"
	StateCompleted   = "completed"
	StateFailed      = "failed"
	StateCancelled   = "cancelled"
)

type ImportOptions struct {
	Mode                        string `json:"mode,omitempty"`
	IncludeMemories             *bool  `json:"include_memories,omitempty"`
	IncludeThreads              *bool  `json:"include_threads,omitempty"`
	IncludeMessages             *bool  `json:"include_messages,omitempty"`
	IncludeEntities             *bool  `json:"include_entities,omitempty"`
	IncludeLabels               *bool  `json:"include_labels,omitempty"`
	IncludeSources              *bool  `json:"include_sources,omitempty"`
	IncludeCommunities          *bool  `json:"include_communities,omitempty"`
	IncludeSkills               *bool  `json:"include_skills,omitempty"`
	IncludeEdges                *bool  `json:"include_edges,omitempty"`
	IncludeWorkingMemory        *bool  `json:"include_working_memory,omitempty"`
	IncludeWorkingMemoryArchive *bool  `json:"include_working_memory_archive,omitempty"`
	IncludeSourceFiles          *bool  `json:"include_source_files,omitempty"`
}

type StartRequest struct {
	Tenant             string
	Target             config.TargetConfig
	Destination        config.SourceConfig
	ObjectName         string
	EncryptionPassword string
	ImportOptions      ImportOptions
}

type Job struct {
	Tenant               string     `json:"-"`
	ID                   string     `json:"id"`
	State                string     `json:"state"`
	Stage                string     `json:"stage"`
	TargetKey            string     `json:"target_key"`
	ObjectName           string     `json:"object_name"`
	DestinationSourceKey string     `json:"destination_source_key"`
	Encrypted            bool       `json:"encrypted"`
	SizeBytes            int64      `json:"size_bytes"`
	MemJobID             string     `json:"mem_job_id,omitempty"`
	Progress             float64    `json:"progress"`
	Imported             int        `json:"imported"`
	Skipped              int        `json:"skipped"`
	Failed               int        `json:"failed"`
	Message              string     `json:"message,omitempty"`
	Error                string     `json:"error,omitempty"`
	StartedAt            time.Time  `json:"started_at"`
	FinishedAt           *time.Time `json:"finished_at,omitempty"`
}

type Manager struct {
	mu           sync.RWMutex
	jobs         map[string]*Job
	logger       *slog.Logger
	openTarget   func(context.Context, config.TargetConfig) (storage.Target, error)
	pollInterval time.Duration
	now          func() time.Time
}

func NewManager(logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}
	factory := storage.NewFactory()
	return &Manager{
		jobs:         make(map[string]*Job),
		logger:       logger,
		openTarget:   factory.Target,
		pollInterval: 2 * time.Second,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

func (m *Manager) ListObjects(ctx context.Context, target config.TargetConfig, prefix string) ([]storage.BackupObject, error) {
	remoteTarget, err := m.openTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	defer func() { _ = remoteTarget.Close() }()
	return storage.ListBackupObjects(ctx, remoteTarget, prefix)
}

func (m *Manager) Start(ctx context.Context, req StartRequest) (*Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateStartRequest(req); err != nil {
		return nil, err
	}
	id, err := newJobID()
	if err != nil {
		return nil, err
	}
	job := &Job{
		ID:                   id,
		Tenant:               strings.TrimSpace(req.Tenant),
		State:                StateQueued,
		Stage:                StateQueued,
		TargetKey:            req.Target.Key,
		ObjectName:           strings.TrimSpace(req.ObjectName),
		DestinationSourceKey: req.Destination.Key,
		Encrypted:            isEncryptedObject(req.ObjectName),
		StartedAt:            m.now(),
	}
	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	started := cloneJob(job)
	go m.run(job.ID, req)
	return started, nil
}

func (m *Manager) Get(id string) (*Job, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	job, ok := m.jobs[id]
	if !ok {
		return nil, false
	}
	return cloneJob(job), true
}

func (m *Manager) run(jobID string, req StartRequest) {
	ctx := context.Background()
	objectName := strings.TrimSpace(req.ObjectName)
	jobLogger := m.restoreLogger(jobID, req)
	defer func() {
		if r := recover(); r != nil {
			m.failWithLogger(jobID, fmt.Errorf("restore job panicked: %v", r), jobLogger)
		}
	}()

	jobLogger.Info("restore job started", "stage", StateQueued, "import_mode", strings.TrimSpace(req.ImportOptions.Mode))
	m.setState(jobID, StateDownloading, "")
	jobLogger.Info("restore download started", "stage", StateDownloading)
	remoteTarget, err := m.openTarget(ctx, req.Target)
	if err != nil {
		m.failWithLogger(jobID, err, jobLogger)
		return
	}
	defer func() {
		if closeErr := remoteTarget.Close(); closeErr != nil {
			jobLogger.Warn("restore target close failed", "error", closeErr)
		}
	}()
	data, err := storage.Read(ctx, remoteTarget, objectName)
	if err != nil {
		m.failWithLogger(jobID, err, jobLogger)
		return
	}
	m.update(jobID, func(job *Job) {
		job.SizeBytes = int64(len(data))
	})
	jobLogger.Info("restore download finished", "stage", StateDownloading, "bytes", len(data))

	if isEncryptedObject(objectName) {
		m.setState(jobID, StateDecrypting, "")
		jobLogger.Info("restore decrypt started", "stage", StateDecrypting)
		plain, _, err := archive.Decrypt(data, req.EncryptionPassword)
		if err != nil {
			m.failWithLogger(jobID, err, jobLogger)
			return
		}
		data = plain
		jobLogger.Info("restore decrypt finished", "stage", StateDecrypting, "bytes", len(data))
	}

	m.setState(jobID, StateUploading, "")
	jobLogger.Info("restore upload started", "stage", StateUploading)
	client := newClient(req.Destination)
	defer client.Close()
	uploadResp, err := client.Data.UploadImport(ctx, uploadRequest(data, objectName, req.ImportOptions))
	if err != nil {
		m.failWithLogger(jobID, err, jobLogger)
		return
	}
	m.update(jobID, func(job *Job) {
		job.MemJobID = uploadResp.JobID
		job.Message = uploadResp.Message
	})
	jobLogger.Info("restore upload finished", "stage", StateUploading, "mem_job", uploadResp.JobID, "message", uploadResp.Message)

	if uploadResp.JobID == "" {
		if isCompletedStatus(uploadResp.Status) {
			m.complete(jobID, nil)
			jobLogger.Info("restore job completed", "stage", StateCompleted)
			return
		}
		m.failWithLogger(jobID, fmt.Errorf("nowledge mem import did not return job id"), jobLogger)
		return
	}

	m.setState(jobID, StateImporting, uploadResp.Message)
	jobLogger.Info("restore import started", "stage", StateImporting, "mem_job", uploadResp.JobID)
	m.pollImport(ctx, jobID, client, uploadResp.JobID, jobLogger)
}

func (m *Manager) pollImport(ctx context.Context, jobID string, client *mem.Client, memJobID string, logger *slog.Logger) {
	interval := m.pollInterval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			m.cancelWithLogger(jobID, ctx.Err().Error(), logger)
			return
		case <-timer.C:
		}

		status, err := client.Data.ImportStatus(ctx, memJobID)
		if err != nil {
			m.failWithLogger(jobID, err, logger)
			return
		}
		m.update(jobID, func(job *Job) {
			job.Progress = clampProgress(status.Progress)
			job.Imported = status.Imported
			job.Skipped = status.Skipped
			job.Failed = status.Failed
			job.Message = status.Message
			job.MemJobID = status.JobID
			if job.MemJobID == "" {
				job.MemJobID = memJobID
			}
		})
		if isCompletedStatus(status.Status) {
			m.complete(jobID, status)
			logger.Info("restore job completed",
				"stage", StateCompleted,
				"mem_job", status.JobID,
				"progress", clampProgress(status.Progress),
				"imported", status.Imported,
				"skipped", status.Skipped,
				"failed", status.Failed,
				"message", status.Message,
			)
			return
		}
		if isFailedStatus(status.Status) {
			message := strings.TrimSpace(status.Message)
			if message == "" {
				message = fmt.Sprintf("nowledge mem import ended with status %q", status.Status)
			}
			m.failWithLogger(jobID, fmt.Errorf("%s", message), logger)
			return
		}
		timer.Reset(interval)
	}
}

func (m *Manager) setState(jobID, state, message string) {
	m.update(jobID, func(job *Job) {
		job.State = state
		job.Stage = state
		if message != "" {
			job.Message = message
		}
	})
}

func (m *Manager) complete(jobID string, status *mem.DataImportStatus) {
	now := m.now()
	m.update(jobID, func(job *Job) {
		job.State = StateCompleted
		job.Stage = StateCompleted
		job.Progress = 1
		job.Error = ""
		job.FinishedAt = &now
		if status != nil {
			job.Imported = status.Imported
			job.Skipped = status.Skipped
			job.Failed = status.Failed
			job.Message = status.Message
			if status.JobID != "" {
				job.MemJobID = status.JobID
			}
		}
	})
}

func (m *Manager) fail(jobID string, err error) {
	m.failWithLogger(jobID, err, nil)
}

func (m *Manager) failWithLogger(jobID string, err error, logger *slog.Logger) {
	if err == nil {
		err = fmt.Errorf("restore failed")
	}
	now := m.now()
	failedStage := ""
	m.update(jobID, func(job *Job) {
		failedStage = job.Stage
		job.State = StateFailed
		job.Stage = StateFailed
		job.Error = err.Error()
		job.FinishedAt = &now
	})
	if logger == nil {
		logger = m.restoreLoggerForJob(jobID)
	}
	logger.Warn("restore job failed", "stage", StateFailed, "failed_stage", failedStage, "error", err)
}

func (m *Manager) cancel(jobID, message string) {
	m.cancelWithLogger(jobID, message, nil)
}

func (m *Manager) cancelWithLogger(jobID, message string, logger *slog.Logger) {
	now := m.now()
	cancelledStage := ""
	m.update(jobID, func(job *Job) {
		cancelledStage = job.Stage
		job.State = StateCancelled
		job.Stage = StateCancelled
		job.Message = message
		job.FinishedAt = &now
	})
	if logger == nil {
		logger = m.restoreLoggerForJob(jobID)
	}
	logger.Warn("restore job cancelled", "stage", StateCancelled, "cancelled_stage", cancelledStage, "message", message)
}

func (m *Manager) update(jobID string, fn func(*Job)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	job, ok := m.jobs[jobID]
	if !ok {
		return
	}
	fn(job)
}

func (m *Manager) restoreLogger(jobID string, req StartRequest) *slog.Logger {
	objectName := strings.TrimSpace(req.ObjectName)
	return m.logger.With(
		"tenant", strings.TrimSpace(req.Tenant),
		"job", jobID,
		"target", req.Target.Key,
		"target_type", req.Target.Type,
		"destination_source", req.Destination.Key,
		"object", objectName,
		"encrypted", isEncryptedObject(objectName),
	)
}

func (m *Manager) restoreLoggerForJob(jobID string) *slog.Logger {
	job, ok := m.Get(jobID)
	if !ok {
		return m.logger.With("job", jobID)
	}
	return m.logger.With(
		"tenant", job.Tenant,
		"job", job.ID,
		"target", job.TargetKey,
		"destination_source", job.DestinationSourceKey,
		"object", job.ObjectName,
		"encrypted", job.Encrypted,
	)
}

func validateStartRequest(req StartRequest) error {
	if !req.Target.Enabled {
		return fmt.Errorf("target %q is disabled", req.Target.Key)
	}
	if !req.Destination.Enabled {
		return fmt.Errorf("destination source %q is disabled", req.Destination.Key)
	}
	if req.Destination.Type != "nowledgemem_api" {
		return fmt.Errorf("destination source %q must be nowledgemem_api", req.Destination.Key)
	}
	objectName := strings.TrimSpace(req.ObjectName)
	if objectName == "" {
		return fmt.Errorf("object_name is required")
	}
	if !isImportableObject(objectName) {
		return fmt.Errorf("object_name must end with %s or %s", archive.PlainExtension, archive.EncryptedExtension)
	}
	if isEncryptedObject(objectName) && strings.TrimSpace(req.EncryptionPassword) == "" {
		return fmt.Errorf("encryption_password is required for encrypted objects")
	}
	mode := strings.TrimSpace(req.ImportOptions.Mode)
	if len(mode) > 64 {
		return fmt.Errorf("import mode is too long")
	}
	if strings.ContainsAny(mode, "\x00\r\n\t") {
		return fmt.Errorf("import mode contains invalid characters")
	}
	if strings.TrimSpace(req.Destination.NowledgeMem.APIURL) == "" {
		return fmt.Errorf("destination nowledge_mem api_url is required")
	}
	return nil
}

func newClient(source config.SourceConfig) *mem.Client {
	timeout := source.NowledgeMem.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	opts := []mem.Option{
		mem.WithBaseURL(source.NowledgeMem.APIURL),
		mem.WithTimeout(timeout),
	}
	if source.NowledgeMem.APIKey != "" {
		opts = append(opts, mem.WithAPIKey(source.NowledgeMem.APIKey))
	}
	return mem.NewClient(opts...)
}

func uploadRequest(data []byte, objectName string, opts ImportOptions) *mem.UploadImportRequest {
	return &mem.UploadImportRequest{
		File:                        bytes.NewReader(data),
		Filename:                    importFilename(objectName),
		Mode:                        strings.TrimSpace(opts.Mode),
		IncludeMemories:             opts.IncludeMemories,
		IncludeThreads:              opts.IncludeThreads,
		IncludeMessages:             opts.IncludeMessages,
		IncludeEntities:             opts.IncludeEntities,
		IncludeLabels:               opts.IncludeLabels,
		IncludeSources:              opts.IncludeSources,
		IncludeCommunities:          opts.IncludeCommunities,
		IncludeSkills:               opts.IncludeSkills,
		IncludeEdges:                opts.IncludeEdges,
		IncludeWorkingMemory:        opts.IncludeWorkingMemory,
		IncludeWorkingMemoryArchive: opts.IncludeWorkingMemoryArchive,
		IncludeSourceFiles:          opts.IncludeSourceFiles,
	}
}

func importFilename(objectName string) string {
	name := path.Base(strings.TrimSpace(objectName))
	if strings.HasSuffix(name, archive.EncryptedExtension) {
		name = strings.TrimSuffix(name, ".aes.json")
	}
	if name == "." || name == "/" || name == "" {
		return "import.zip"
	}
	if !strings.HasSuffix(name, archive.PlainExtension) {
		return name + archive.PlainExtension
	}
	return name
}

func isImportableObject(name string) bool {
	return strings.HasSuffix(name, archive.PlainExtension) || strings.HasSuffix(name, archive.EncryptedExtension)
}

func isEncryptedObject(name string) bool {
	return strings.HasSuffix(name, archive.EncryptedExtension)
}

func isCompletedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "complete", "succeeded", "success", "done":
		return true
	default:
		return false
	}
}

func isFailedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "error", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func clampProgress(progress float64) float64 {
	if progress < 0 {
		return 0
	}
	if progress > 1 {
		return 1
	}
	return progress
}

func cloneJob(job *Job) *Job {
	if job == nil {
		return nil
	}
	copy := *job
	if job.FinishedAt != nil {
		finishedAt := *job.FinishedAt
		copy.FinishedAt = &finishedAt
	}
	return &copy
}

func newJobID() (string, error) {
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate restore job id: %w", err)
	}
	return "restore-" + time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(random[:]), nil
}
