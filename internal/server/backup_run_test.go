package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

func TestRunBackupRejectsTargetOutsideTask(t *testing.T) {
	srv := newRestoreAPITestServer(t, backupRunTestConfig())

	req := newAuthenticatedJSONRequest(t, srv, http.MethodPost, "/api/backup/run", `{"task_key":"task","target_keys":["target-b"]}`)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("not configured for task")) {
		t.Fatalf("body = %s, want target selection error", rr.Body.String())
	}
}

func TestRunBackupRejectsEmptyTargetSelection(t *testing.T) {
	srv := newRestoreAPITestServer(t, backupRunTestConfig())

	req := newAuthenticatedJSONRequest(t, srv, http.MethodPost, "/api/backup/run", `{"task_key":"task","target_keys":[]}`)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if !bytes.Contains(rr.Body.Bytes(), []byte("at least one target")) {
		t.Fatalf("body = %s, want empty selection error", rr.Body.String())
	}
}

func backupRunTestConfig() config.Config {
	cfg := config.Default()
	cfg.Targets = []config.TargetConfig{
		backupRunTestTarget("target-a"),
		backupRunTestTarget("target-b"),
	}
	cfg.Tasks = []config.TaskConfig{{
		Key:               "task",
		Name:              "Task",
		Enabled:           true,
		SourceKey:         config.DefaultSourceKey,
		ScheduleKey:       config.DefaultDailyScheduleKey,
		TargetKeys:        []string{"target-a"},
		ExportOptionKey:   config.DefaultExportOptionKey,
		BackupStrategyKey: config.DefaultBackupStrategyKey,
		ObjectPrefix:      "backups/{task}/{timestamp}",
	}}
	return cfg
}

func backupRunTestTarget(key string) config.TargetConfig {
	return config.TargetConfig{
		Key:     key,
		Name:    key,
		Enabled: true,
		Type:    "s3",
		S3: config.S3Config{
			EndpointURL: "http://127.0.0.1:9000",
			Region:      "auto",
			BucketName:  "bucket",
			AccessKeyID: "access",
		},
	}
}
