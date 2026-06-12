package history

import (
	"context"
	"testing"
	"time"

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

	startedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	payload := `{"id":"legacy","task_key":"task","task_name":"Task","source_key":"source","status":"success","object_name":"backup.zip","encrypted":false,"size_bytes":42,"started_at":"2026-01-02T03:04:05Z","targets":null}`
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
