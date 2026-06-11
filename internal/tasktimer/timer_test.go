package tasktimer

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

func TestReplaceAllSnapshotIncludesScheduledAndDisabledTasks(t *testing.T) {
	timer := New(func(context.Context, config.TaskConfig) error { return nil }, nil, nil)
	timer.Start(context.Background())
	defer timer.Stop()

	timer.ReplaceAll([]Entry{
		{
			Task:        config.TaskConfig{Key: "scheduled", Enabled: true},
			Schedule:    config.ScheduleConfig{Key: "daily", Enabled: true, Type: "daily", Time: "23:59"},
			HasSchedule: true,
		},
		{
			Task:        config.TaskConfig{Key: "disabled", Enabled: false},
			Schedule:    config.ScheduleConfig{Key: "daily", Enabled: true, Type: "daily", Time: "23:59"},
			HasSchedule: true,
		},
	})

	snapshot := timer.Snapshot()
	if got := snapshot["scheduled"]; got.Status != config.TaskRuntimeStatusScheduled || got.NextRunAt == nil {
		t.Fatalf("scheduled runtime = %#v", got)
	}
	if got := snapshot["disabled"]; got.Status != config.TaskRuntimeStatusDisabled || got.NextRunAt != nil {
		t.Fatalf("disabled runtime = %#v", got)
	}
}

func TestUpsertAndRemoveUpdateSnapshot(t *testing.T) {
	timer := New(func(context.Context, config.TaskConfig) error { return nil }, nil, nil)
	timer.Start(context.Background())
	defer timer.Stop()

	timer.Upsert(Entry{
		Task:        config.TaskConfig{Key: "added", Enabled: true},
		Schedule:    config.ScheduleConfig{Key: "daily", Enabled: true, Type: "daily", Time: "23:59"},
		HasSchedule: true,
	})
	if got := timer.Snapshot()["added"]; got.Status != config.TaskRuntimeStatusScheduled || got.NextRunAt == nil {
		t.Fatalf("upserted runtime = %#v", got)
	}

	timer.Remove("added")
	if _, ok := timer.Snapshot()["added"]; ok {
		t.Fatal("removed task still present in snapshot")
	}
}

func TestTimeWheelDiagnosticsUseTaskLogger(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	timer := New(func(context.Context, config.TaskConfig) error { return nil }, logger, nil)

	timer.Start(context.Background())
	timer.Stop()

	got := logs.String()
	if !strings.Contains(got, `"msg":"timewheel: stopped"`) || !strings.Contains(got, `"component":"timewheel"`) {
		t.Fatalf("timewheel stop log not recorded through task logger: %s", got)
	}
}

func TestDueOnceTaskRunsAndDisables(t *testing.T) {
	ran := make(chan string, 1)
	disabled := make(chan string, 1)
	timer := New(func(_ context.Context, task config.TaskConfig) error {
		ran <- task.Key
		return nil
	}, nil, func(taskKey string) error {
		disabled <- taskKey
		return nil
	})
	timer.Start(context.Background())
	defer timer.Stop()

	timer.ReplaceAll([]Entry{{
		Task:        config.TaskConfig{Key: "once", Enabled: true},
		Schedule:    config.ScheduleConfig{Key: "once", Enabled: true, Type: "once", RunAt: "2000-01-01T00:00"},
		HasSchedule: true,
	}})

	select {
	case key := <-ran:
		if key != "once" {
			t.Fatalf("ran task = %q", key)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task run")
	}
	select {
	case key := <-disabled:
		if key != "once" {
			t.Fatalf("disabled task = %q", key)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for once disable callback")
	}
	if got := timer.Snapshot()["once"]; got.Status != config.TaskRuntimeStatusDisabled || got.NextRunAt != nil {
		t.Fatalf("once runtime after run = %#v", got)
	}
}

func TestRemoveCancelsRunningTask(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	disabled := make(chan string, 1)
	timer := New(func(ctx context.Context, _ config.TaskConfig) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}, nil, func(taskKey string) error {
		disabled <- taskKey
		return nil
	})
	timer.Start(context.Background())
	defer timer.Stop()

	timer.ReplaceAll([]Entry{{
		Task:        config.TaskConfig{Key: "running", Enabled: true},
		Schedule:    config.ScheduleConfig{Key: "once", Enabled: true, Type: "once", RunAt: "2000-01-01T00:00"},
		HasSchedule: true,
	}})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task start")
	}
	timer.Remove("running")
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task cancellation")
	}
	if _, ok := timer.Snapshot()["running"]; ok {
		t.Fatal("removed task still present in snapshot")
	}
	select {
	case key := <-disabled:
		t.Fatalf("removed once task should not be disabled after stale completion, got %q", key)
	default:
	}
}

func TestUpsertCancelsRunningTask(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	timer := New(func(ctx context.Context, _ config.TaskConfig) error {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}, nil, nil)
	timer.Start(context.Background())
	defer timer.Stop()

	timer.ReplaceAll([]Entry{{
		Task:        config.TaskConfig{Key: "running", Enabled: true},
		Schedule:    config.ScheduleConfig{Key: "once", Enabled: true, Type: "once", RunAt: "2000-01-01T00:00"},
		HasSchedule: true,
	}})

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task start")
	}
	timer.Upsert(Entry{
		Task:        config.TaskConfig{Key: "running", Enabled: true},
		Schedule:    config.ScheduleConfig{Key: "daily", Enabled: true, Type: "daily", Time: "23:59"},
		HasSchedule: true,
	})
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task cancellation")
	}
	if got := timer.Snapshot()["running"]; got.Status != config.TaskRuntimeStatusScheduled || got.NextRunAt == nil {
		t.Fatalf("upserted task runtime = %#v", got)
	}
}

func TestSnapshotReportsScheduleProblems(t *testing.T) {
	timer := New(func(context.Context, config.TaskConfig) error { return nil }, nil, nil)
	timer.Start(context.Background())
	defer timer.Stop()

	timer.ReplaceAll([]Entry{
		{
			Task:        config.TaskConfig{Key: "missing", Enabled: true, ScheduleKey: "missing"},
			HasSchedule: false,
		},
		{
			Task:        config.TaskConfig{Key: "schedule-off", Enabled: true, ScheduleKey: "off"},
			Schedule:    config.ScheduleConfig{Key: "off", Enabled: false, Type: "daily", Time: "23:59"},
			HasSchedule: true,
		},
		{
			Task:        config.TaskConfig{Key: "invalid", Enabled: true, ScheduleKey: "bad"},
			Schedule:    config.ScheduleConfig{Key: "bad", Enabled: true, Type: "daily", Time: "bad"},
			HasSchedule: true,
		},
	})

	snapshot := timer.Snapshot()
	if got := snapshot["missing"]; got.Status != config.TaskRuntimeStatusMissingSchedule {
		t.Fatalf("missing runtime = %#v", got)
	}
	if got := snapshot["schedule-off"]; got.Status != config.TaskRuntimeStatusScheduleDisabled {
		t.Fatalf("schedule-off runtime = %#v", got)
	}
	if got := snapshot["invalid"]; got.Status != config.TaskRuntimeStatusInvalidSchedule {
		t.Fatalf("invalid runtime = %#v", got)
	}
}

func TestOnceDisableFailureStillDisablesRuntime(t *testing.T) {
	ran := make(chan struct{}, 1)
	timer := New(func(context.Context, config.TaskConfig) error {
		ran <- struct{}{}
		return nil
	}, nil, func(string) error {
		return context.Canceled
	})
	timer.Start(context.Background())
	defer timer.Stop()

	timer.ReplaceAll([]Entry{{
		Task:        config.TaskConfig{Key: "once", Enabled: true},
		Schedule:    config.ScheduleConfig{Key: "once", Enabled: true, Type: "once", RunAt: "2000-01-01T00:00"},
		HasSchedule: true,
	}})

	select {
	case <-ran:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task run")
	}
	deadline := time.After(time.Second)
	for {
		if got := timer.Snapshot()["once"]; got.Status == config.TaskRuntimeStatusDisabled && got.NextRunAt == nil {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("once runtime after failed disable = %#v", timer.Snapshot()["once"])
		default:
			time.Sleep(time.Millisecond)
		}
	}
}
