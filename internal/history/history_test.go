package history

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	mem "github.com/lib-x/nowledgemem-go"

	"github.com/ca-x/nowledge-mem-snap/internal/persist"
)

func TestListNormalizesLegacyNullTargets(t *testing.T) {
	client, err := persist.OpenClient(t.TempDir())
	if err != nil {
		t.Fatalf("OpenClient: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	})

	startedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	payload := fmt.Sprintf(`{"id":"legacy","task_key":"task","task_name":"Task","source_key":"source","status":"success","object_name":"backup.zip","encrypted":false,"size_bytes":42,"started_at":"%s","targets":null}`, startedAt.Format(time.RFC3339))
	if err := client.RunRecord.Create().
		SetTenant("tenant").
		SetRunID("legacy").
		SetPayload(payload).
		SetStartedAt(startedAt).
		Exec(context.Background()); err != nil {
		t.Fatalf("create run record: %v", err)
	}

	runs, err := NewStore(client, "tenant", 100).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("len(runs) = %d, want 1", len(runs))
	}
	if runs[0].Targets == nil {
		t.Fatal("Targets is nil, want empty slice")
	}
	if len(runs[0].Targets) != 0 {
		t.Fatalf("len(Targets) = %d, want 0", len(runs[0].Targets))
	}
}

func TestAppendPersistsSanitizedFailureDetails(t *testing.T) {
	client, err := persist.OpenClient(t.TempDir())
	if err != nil {
		t.Fatalf("OpenClient: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	})

	startedAt := time.Now().UTC().Truncate(time.Second)
	store := NewStore(client, "tenant", 100)
	err = store.Append(Run{
		ID:        "failed-run",
		TaskKey:   "task",
		TaskName:  "Task",
		SourceKey: "source",
		Status:    "failed",
		Error:     "download nowledge mem export: Post \"https://username-only@bad.example/export?nmem_api_key=top-secret\"\nAuthorization: Digest username=\"backup\", response=\"digest-secret\"\nCookie: theme=dark; sessionid=cookie-secret\n{\"api_key\":\"json-secret\"} API Key=spaced-secret password=\"quoted-secret\" secret_access_key=punctuation/+==",
		StartedAt: startedAt,
		Targets: []TargetResult{{
			TargetKey:  "target",
			TargetName: "Target",
			Status:     "failed",
			Error:      "upload failed: password=storage-secret",
		}},
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	runs, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("len(runs) = %d, want 1", len(runs))
	}
	if !strings.Contains(runs[0].Error, "bad.example") || !strings.Contains(runs[0].Error, "[REDACTED]") {
		t.Fatalf("run error = %q, want useful sanitized detail", runs[0].Error)
	}
	for _, secret := range []string{"username-only", "top-secret", "digest-secret", "cookie-secret", "json-secret", "spaced-secret", "quoted-secret", "punctuation/+==", "storage-secret"} {
		if strings.Contains(runs[0].Error, secret) || strings.Contains(runs[0].Targets[0].Error, secret) {
			t.Fatalf("persisted failure details contain secret %q", secret)
		}
	}
}

func TestErrorDetailOmitsUntrustedAPIBody(t *testing.T) {
	err := fmt.Errorf("nowledge mem checkpoint: %w", &mem.APIError{
		StatusCode: 502,
		Body:       `{"api_key":"body-secret","detail":"proxy failed"}`,
	})
	detail := ErrorDetail(err)
	if !strings.Contains(detail, "nowledge mem checkpoint") || !strings.Contains(detail, "status 502") {
		t.Fatalf("detail = %q, want stage and status", detail)
	}
	if strings.Contains(detail, "body-secret") || strings.Contains(detail, "proxy failed") {
		t.Fatalf("detail = %q, contains untrusted API body", detail)
	}
}

func TestListSanitizesLegacyFailureDetails(t *testing.T) {
	client, err := persist.OpenClient(t.TempDir())
	if err != nil {
		t.Fatalf("OpenClient: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Fatalf("close client: %v", err)
		}
	})

	startedAt := time.Now().UTC().Truncate(time.Second)
	payload := fmt.Sprintf(`{"id":"legacy-failure","task_key":"task","task_name":"Task","source_key":"source","status":"failed","error":"api_key=legacy-secret","started_at":"%s","targets":[]}`, startedAt.Format(time.RFC3339))
	if err := client.RunRecord.Create().
		SetTenant("tenant").
		SetRunID("legacy-failure").
		SetPayload(payload).
		SetStartedAt(startedAt).
		Exec(context.Background()); err != nil {
		t.Fatalf("create run record: %v", err)
	}

	runs, err := NewStore(client, "tenant", 100).List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(runs) != 1 || strings.Contains(runs[0].Error, "legacy-secret") || !strings.Contains(runs[0].Error, "[REDACTED]") {
		t.Fatalf("runs = %#v, want sanitized legacy failure", runs)
	}
}
