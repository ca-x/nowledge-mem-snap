package restore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"

	"github.com/ca-x/nowledge-mem-snap/internal/archive"
	"github.com/ca-x/nowledge-mem-snap/internal/config"
	"github.com/ca-x/nowledge-mem-snap/internal/storage"
)

func TestListObjectsRejectsEmptyPrefix(t *testing.T) {
	manager := newTestManager(afero.NewMemMapFs())
	_, err := manager.ListObjects(context.Background(), testTarget(), "")
	if err == nil {
		t.Fatal("ListObjects accepted empty prefix")
	}
}

func TestStartRejectsNonImportableObject(t *testing.T) {
	manager := newTestManager(afero.NewMemMapFs())
	_, err := manager.Start(context.Background(), StartRequest{
		Target:      testTarget(),
		Destination: testSource("http://127.0.0.1:14242"),
		ObjectName:  "backups/readme.txt",
	})
	if err == nil {
		t.Fatal("Start accepted non-importable object")
	}
}

func TestStartEncryptedObjectRequiresPassword(t *testing.T) {
	manager := newTestManager(afero.NewMemMapFs())
	_, err := manager.Start(context.Background(), StartRequest{
		Target:      testTarget(),
		Destination: testSource("http://127.0.0.1:14242"),
		ObjectName:  "backups/export.zip.aes.json",
	})
	if err == nil {
		t.Fatal("Start accepted encrypted object without password")
	}
}

func TestPlainZIPRestoreUploadsAndCompletes(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "backups/export.zip", []byte("zip-data"), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	server := newImportServer(t, importServerExpect{
		filename: "export.zip",
		body:     "zip-data",
		mode:     "append",
		fields: map[string]string{
			"include_memories":     "true",
			"include_source_files": "false",
		},
	})
	defer server.Close()

	manager := newTestManager(fs)
	job, err := manager.Start(context.Background(), StartRequest{
		Target:      testTarget(),
		Destination: testSource(server.URL),
		ObjectName:  "backups/export.zip",
		ImportOptions: ImportOptions{
			Mode:               "append",
			IncludeMemories:    boolPtr(true),
			IncludeSourceFiles: boolPtr(false),
		},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	got := waitJob(t, manager, job.ID)
	if got.State != StateCompleted {
		t.Fatalf("state = %s, error = %s", got.State, got.Error)
	}
	if got.SizeBytes != int64(len("zip-data")) {
		t.Fatalf("size_bytes = %d, want %d", got.SizeBytes, len("zip-data"))
	}
	if got.MemJobID != "job-1" {
		t.Fatalf("mem_job_id = %q, want job-1", got.MemJobID)
	}
	if got.Imported != 7 || got.Skipped != 1 || got.Failed != 0 {
		t.Fatalf("import counters = %d/%d/%d", got.Imported, got.Skipped, got.Failed)
	}
}

func TestEncryptedRestoreDecryptsBeforeUpload(t *testing.T) {
	fs := afero.NewMemMapFs()
	encrypted, err := archive.Encrypt([]byte("zip-data"), "secret", archive.Metadata{TaskKey: "task"})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if err := afero.WriteFile(fs, "backups/export.zip.aes.json", encrypted, 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	server := newImportServer(t, importServerExpect{
		filename: "export.zip",
		body:     "zip-data",
	})
	defer server.Close()

	manager := newTestManager(fs)
	job, err := manager.Start(context.Background(), StartRequest{
		Target:             testTarget(),
		Destination:        testSource(server.URL),
		ObjectName:         "backups/export.zip.aes.json",
		EncryptionPassword: "secret",
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	got := waitJob(t, manager, job.ID)
	if got.State != StateCompleted {
		t.Fatalf("state = %s, error = %s", got.State, got.Error)
	}
	if !got.Encrypted {
		t.Fatal("job did not record encrypted source object")
	}
}

func TestRestoreWritesOperationLogsWithoutSecrets(t *testing.T) {
	fs := afero.NewMemMapFs()
	encrypted, err := archive.Encrypt([]byte("zip-data"), "secret", archive.Metadata{TaskKey: "task"})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if err := afero.WriteFile(fs, "backups/export.zip.aes.json", encrypted, 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	server := newImportServer(t, importServerExpect{
		filename: "export.zip",
		body:     "zip-data",
		mode:     "merge",
	})
	defer server.Close()

	var logs bytes.Buffer
	manager := newTestManagerWithLogger(fs, slog.New(slog.NewJSONHandler(&logs, nil)))
	job, err := manager.Start(context.Background(), StartRequest{
		Tenant:             "tenant-1",
		Target:             testTarget(),
		Destination:        testSource(server.URL),
		ObjectName:         " backups/export.zip.aes.json ",
		EncryptionPassword: "secret",
		ImportOptions: ImportOptions{
			Mode: "merge",
		},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	got := waitJob(t, manager, job.ID)
	if got.State != StateCompleted {
		t.Fatalf("state = %s, error = %s", got.State, got.Error)
	}
	output := logs.String()
	for _, want := range []string{
		"restore job started",
		"restore download started",
		"restore decrypt started",
		"restore upload started",
		"restore import started",
		"restore job completed",
		"tenant-1",
		"target",
		"source",
		"backups/export.zip.aes.json",
		"job-1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("restore logs do not contain %q:\n%s", want, output)
		}
	}
	for _, secret := range []string{"secret", "test-key"} {
		if strings.Contains(output, secret) {
			t.Fatalf("restore logs leaked %q:\n%s", secret, output)
		}
	}
}

type importServerExpect struct {
	filename string
	body     string
	mode     string
	fields   map[string]string
}

func newImportServer(t *testing.T, expect importServerExpect) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/data/import/upload":
			if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
				t.Fatalf("Authorization = %q, want bearer API key", got)
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("ParseMultipartForm: %v", err)
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("FormFile: %v", err)
			}
			body, err := io.ReadAll(file)
			if closeErr := file.Close(); err == nil {
				err = closeErr
			}
			if err != nil {
				t.Fatalf("read file: %v", err)
			}
			if header.Filename != expect.filename {
				t.Fatalf("filename = %q, want %q", header.Filename, expect.filename)
			}
			if string(body) != expect.body {
				t.Fatalf("uploaded body = %q, want %q", body, expect.body)
			}
			if got := r.FormValue("mode"); got != expect.mode {
				t.Fatalf("mode = %q, want %q", got, expect.mode)
			}
			for key, want := range expect.fields {
				if got := r.FormValue(key); got != want {
					t.Fatalf("%s = %q, want %q", key, got, want)
				}
			}
			writeTestJSON(t, w, map[string]any{"job_id": "job-1", "status": "queued", "message": "uploaded"})
		case r.Method == http.MethodGet && r.URL.Path == "/data/import/status/job-1":
			writeTestJSON(t, w, map[string]any{
				"job_id":   "job-1",
				"status":   "completed",
				"progress": 1,
				"imported": 7,
				"skipped":  1,
				"failed":   0,
				"message":  "done",
			})
		default:
			http.NotFound(w, r)
		}
	}))
}

func writeTestJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("encode JSON: %v", err)
	}
}

func waitJob(t *testing.T, manager *Manager, id string) *Job {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case <-deadline:
			job, _ := manager.Get(id)
			t.Fatalf("timed out waiting for job %s, last state %#v", id, job)
		default:
		}
		job, ok := manager.Get(id)
		if !ok {
			t.Fatalf("job %s was not found", id)
		}
		switch job.State {
		case StateCompleted, StateFailed, StateCancelled:
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func newTestManager(fs afero.Fs) *Manager {
	return newTestManagerWithLogger(fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func newTestManagerWithLogger(fs afero.Fs, logger *slog.Logger) *Manager {
	manager := NewManager(logger)
	manager.pollInterval = 10 * time.Millisecond
	manager.openTarget = func(_ context.Context, target config.TargetConfig) (storage.Target, error) {
		if !target.Enabled {
			return storage.Target{}, fmt.Errorf("target %q is disabled", target.Key)
		}
		return storage.Target{Key: target.Key, Name: target.Name, Fs: fs}, nil
	}
	return manager
}

func testTarget() config.TargetConfig {
	return config.TargetConfig{
		Key:     "target",
		Name:    "Target",
		Enabled: true,
		Type:    "s3",
	}
}

func testSource(apiURL string) config.SourceConfig {
	return config.SourceConfig{
		Key:     "source",
		Name:    "Source",
		Enabled: true,
		Type:    "nowledgemem_api",
		NowledgeMem: config.NowledgeConfig{
			APIURL:  strings.TrimRight(apiURL, "/"),
			APIKey:  "test-key",
			Timeout: time.Second,
		},
	}
}

func boolPtr(v bool) *bool {
	return &v
}
