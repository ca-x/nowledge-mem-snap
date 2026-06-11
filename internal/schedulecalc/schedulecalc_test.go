package schedulecalc

import (
	"testing"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

func TestNextRunOnceUsesCurrentLocation(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	got, ok := NextRun(now, config.ScheduleConfig{
		Type:  "once",
		RunAt: "2026-06-10T13:30",
	})
	if !ok {
		t.Fatal("NextRun returned ok=false")
	}
	want := time.Date(2026, 6, 10, 13, 30, 0, 0, loc)
	if !got.Equal(want) || got.Location() != loc {
		t.Fatalf("NextRun = %v (%s), want %v (%s)", got, got.Location(), want, want.Location())
	}
}

func TestNextRunOncePastTimeRunsImmediately(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	got, ok := NextRun(now, config.ScheduleConfig{
		Type:  "once",
		RunAt: "2026-06-10T11:30",
	})
	if !ok {
		t.Fatal("NextRun returned ok=false")
	}
	if !got.Equal(now) || got.Location() != loc {
		t.Fatalf("NextRun = %v (%s), want %v (%s)", got, got.Location(), now, now.Location())
	}
}

func TestNextRunDailyUsesCurrentLocation(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	got, ok := NextRun(now, config.ScheduleConfig{
		Type: "daily",
		Time: "13:30",
	})
	if !ok {
		t.Fatal("NextRun returned ok=false")
	}
	want := time.Date(2026, 6, 10, 13, 30, 0, 0, loc)
	if !got.Equal(want) || got.Location() != loc {
		t.Fatalf("NextRun = %v (%s), want %v (%s)", got, got.Location(), want, want.Location())
	}
}

func TestNextRunWeeklyUsesCurrentLocation(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, loc)
	got, ok := NextRun(now, config.ScheduleConfig{
		Type:    "weekly",
		Time:    "03:00",
		Weekday: "thursday",
	})
	if !ok {
		t.Fatal("NextRun returned ok=false")
	}
	want := time.Date(2026, 6, 11, 3, 0, 0, 0, loc)
	if !got.Equal(want) || got.Location() != loc {
		t.Fatalf("NextRun = %v (%s), want %v (%s)", got, got.Location(), want, want.Location())
	}
}

func TestRuntimeStatuses(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	task := config.TaskConfig{Key: "task", Enabled: true, ScheduleKey: "schedule"}
	schedule := config.ScheduleConfig{Key: "schedule", Enabled: true, Type: "daily", Time: "13:00"}

	if got := Runtime(now, config.TaskConfig{Enabled: false}, schedule, true); got.Status != config.TaskRuntimeStatusDisabled {
		t.Fatalf("disabled task status = %q", got.Status)
	}
	if got := Runtime(now, task, config.ScheduleConfig{}, false); got.Status != config.TaskRuntimeStatusMissingSchedule {
		t.Fatalf("missing schedule status = %q", got.Status)
	}
	disabledSchedule := schedule
	disabledSchedule.Enabled = false
	if got := Runtime(now, task, disabledSchedule, true); got.Status != config.TaskRuntimeStatusScheduleDisabled {
		t.Fatalf("disabled schedule status = %q", got.Status)
	}
	invalidSchedule := schedule
	invalidSchedule.Time = "bad"
	if got := Runtime(now, task, invalidSchedule, true); got.Status != config.TaskRuntimeStatusInvalidSchedule {
		t.Fatalf("invalid schedule status = %q", got.Status)
	}
	if got := Runtime(now, task, schedule, true); got.Status != config.TaskRuntimeStatusScheduled || got.NextRunAt == nil {
		t.Fatalf("scheduled runtime = %#v", got)
	}
}
