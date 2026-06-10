package tasktimer

import (
	"context"
	"testing"
	"time"

	"github.com/lib-x/nowledge-mem-snap/internal/config"
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
	timer.Remove("running")
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for task cancellation")
	}
	if _, ok := timer.Snapshot()["running"]; ok {
		t.Fatal("removed task still present in snapshot")
	}
}
