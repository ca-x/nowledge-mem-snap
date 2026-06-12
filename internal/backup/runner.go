package backup

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/archive"
	"github.com/ca-x/nowledge-mem-snap/internal/config"
	"github.com/ca-x/nowledge-mem-snap/internal/history"
	"github.com/ca-x/nowledge-mem-snap/internal/source"
	"github.com/ca-x/nowledge-mem-snap/internal/storage"
)

type Runner struct {
	cfg      config.Config
	exporter *source.Exporter
	factory  *storage.Factory
	history  *history.Store
	logger   *slog.Logger
}

func NewRunner(cfg config.Config, historyStore *history.Store, logger *slog.Logger) *Runner {
	return &Runner{
		cfg:      cfg,
		exporter: source.NewExporter(),
		factory:  storage.NewFactory(),
		history:  historyStore,
		logger:   logger,
	}
}

func (r *Runner) RunTask(ctx context.Context, taskKey string) (history.Run, error) {
	task, ok := r.cfg.Task(taskKey)
	if !ok {
		return history.Run{}, fmt.Errorf("task %q was not found", taskKey)
	}
	return r.run(ctx, task, "manual")
}

func (r *Runner) RunScheduledTask(ctx context.Context, task config.TaskConfig) (history.Run, error) {
	return r.run(ctx, task, "schedule")
}

func (r *Runner) run(ctx context.Context, task config.TaskConfig, trigger string) (history.Run, error) {
	resolvedTask, err := r.cfg.ResolveTask(task)
	if err != nil {
		return history.Run{}, err
	}
	task = resolvedTask
	start := time.Now().UTC()
	run := history.Run{
		ID:        newRunID(),
		TaskKey:   task.Key,
		TaskName:  task.Name,
		SourceKey: task.SourceKey,
		Status:    "running",
		StartedAt: start,
	}
	runLogger := r.logger.With("tenant", r.history.Tenant(), "run", run.ID, "task", task.Key, "source", task.SourceKey, "trigger", trigger)
	runLogger.Info("backup run started")
	if !task.Enabled && trigger == "schedule" {
		run.Status = "skipped"
		run.Error = "task is disabled"
		now := time.Now().UTC()
		run.FinishedAt = &now
		_ = r.history.Append(run)
		runLogger.Info("backup run skipped", "reason", run.Error)
		return run, nil
	}

	runLogger.Info("backup export started")
	snap, err := r.exporter.Export(ctx, r.cfg, task)
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		now := time.Now().UTC()
		run.FinishedAt = &now
		_ = r.history.Append(run)
		runLogger.Error("backup export failed", "error", err)
		return run, err
	}
	runLogger.Info("backup export finished", "bytes", snap.SizeBytes, "items", snap.ItemCount)

	password := ""
	if task.Encryption.Enabled {
		password = task.Encryption.Password
		if password == "" {
			password = getenv(task.Encryption.PasswordEnv)
		}
	}
	runLogger.Info("backup packaging started", "encrypted", task.Encryption.Enabled)
	artifact, err := archive.Build(snap.Data, archive.BuildOptions{
		Prefix:             task.ObjectPrefix,
		TaskKey:            task.Key,
		TaskName:           task.Name,
		ItemCount:          snap.ItemCount,
		CreatedAt:          start,
		Encrypt:            task.Encryption.Enabled,
		EncryptionPassword: password,
	})
	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()
		now := time.Now().UTC()
		run.FinishedAt = &now
		_ = r.history.Append(run)
		runLogger.Error("backup packaging failed", "encrypted", task.Encryption.Enabled, "error", err)
		return run, err
	}

	run.ObjectName = artifact.Name
	run.Encrypted = artifact.Encrypted
	run.SizeBytes = artifact.SizeBytes
	runLogger.Info("backup packaging finished", "object", artifact.Name, "bytes", artifact.SizeBytes, "encrypted", artifact.Encrypted)

	targets := r.targetsForTask(task)
	if len(targets) == 0 {
		run.Status = "failed"
		run.Error = "no enabled targets"
		now := time.Now().UTC()
		run.FinishedAt = &now
		_ = r.history.Append(run)
		runLogger.Error("backup run failed", "error", run.Error)
		return run, fmt.Errorf("task %q has no enabled targets", task.Key)
	}

	results := make([]history.TargetResult, len(targets))
	var wg sync.WaitGroup
	for i, targetCfg := range targets {
		i := i
		targetCfg := targetCfg
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := history.TargetResult{
				TargetKey:  targetCfg.Key,
				TargetName: targetCfg.Name,
				Status:     "running",
			}
			targetLogger := runLogger.With("target", targetCfg.Key, "target_name", targetCfg.Name)
			targetLogger.Info("backup upload started", "object", artifact.Name, "bytes", artifact.SizeBytes)
			target, err := r.factory.Target(ctx, targetCfg)
			if err == nil {
				defer func() {
					if closeErr := target.Close(); closeErr != nil {
						targetLogger.Warn("backup target close failed", "error", closeErr)
					}
				}()
				result.Bytes, err = storage.Write(ctx, target, artifact.Name, artifact.Data)
			}
			if err == nil {
				targetLogger.Info("backup upload finished", "object", artifact.Name, "bytes", result.Bytes)
				if task.Retention.Mode != "" && task.Retention.Mode != "none" {
					targetLogger.Info("backup retention started", "mode", task.Retention.Mode)
					result.RetentionDeleted, err = storage.ApplyRetention(ctx, target, task, time.Now().In(time.Local))
					if err != nil {
						targetLogger.Warn("backup retention failed", "mode", task.Retention.Mode, "error", err)
					} else {
						targetLogger.Info("backup retention finished", "deleted", result.RetentionDeleted)
					}
				}
			}
			result.FinishedAt = time.Now().UTC()
			if err != nil {
				result.Status = "failed"
				result.Error = err.Error()
				targetLogger.Error("backup target failed", "object", artifact.Name, "error", err)
			} else {
				result.Status = "success"
			}
			results[i] = result
		}()
	}
	wg.Wait()

	failures := 0
	for _, result := range results {
		if result.Status != "success" {
			failures++
		}
	}
	run.Targets = results
	if failures == len(results) {
		run.Status = "failed"
		run.Error = "all targets failed"
	} else if failures > 0 {
		run.Status = "partial"
	} else {
		run.Status = "success"
	}
	now := time.Now().UTC()
	run.FinishedAt = &now
	if err := r.history.Append(run); err != nil {
		runLogger.Warn("failed to append run history", "error", err)
	}
	runLogger.Info("backup run finished",
		"status", run.Status,
		"object", run.ObjectName,
		"bytes", run.SizeBytes,
		"targets", len(run.Targets),
	)
	if run.Status == "failed" {
		return run, fmt.Errorf("backup failed: %s", run.Error)
	}
	return run, nil
}

func (r *Runner) targetsForTask(task config.TaskConfig) []config.TargetConfig {
	targets := make([]config.TargetConfig, 0, len(task.TargetKeys))
	for _, key := range task.TargetKeys {
		target, ok := r.cfg.Target(key)
		if ok && target.Enabled {
			targets = append(targets, target)
		}
	}
	return targets
}

func newRunID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return time.Now().UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(b[:])
}

func getenv(key string) string {
	if key == "" {
		return ""
	}
	return os.Getenv(key)
}
