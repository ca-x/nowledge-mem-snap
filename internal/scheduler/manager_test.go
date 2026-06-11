package scheduler

import (
	"testing"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
)

func TestSnapshotFromConfigReportsTaskStates(t *testing.T) {
	nowLocal := time.Local
	time.Local = time.UTC
	defer func() { time.Local = nowLocal }()

	cfg := config.Config{
		Schedules: []config.ScheduleConfig{
			{Key: "daily", Enabled: true, Type: "daily", Time: "23:59"},
			{Key: "off", Enabled: false, Type: "daily", Time: "23:59"},
		},
		Tasks: []config.TaskConfig{
			{Key: "scheduled", Enabled: true, ScheduleKey: "daily"},
			{Key: "task-off", Enabled: false, ScheduleKey: "daily"},
			{Key: "schedule-off", Enabled: true, ScheduleKey: "off"},
			{Key: "missing", Enabled: true, ScheduleKey: "missing"},
		},
	}

	snapshot := snapshotFromConfig(cfg)
	if got := snapshot["scheduled"]; got.Status != config.TaskRuntimeStatusScheduled || got.NextRunAt == nil {
		t.Fatalf("scheduled runtime = %#v", got)
	}
	if got := snapshot["task-off"]; got.Status != config.TaskRuntimeStatusDisabled {
		t.Fatalf("disabled task runtime = %#v", got)
	}
	if got := snapshot["schedule-off"]; got.Status != config.TaskRuntimeStatusScheduleDisabled {
		t.Fatalf("disabled schedule runtime = %#v", got)
	}
	if got := snapshot["missing"]; got.Status != config.TaskRuntimeStatusMissingSchedule {
		t.Fatalf("missing schedule runtime = %#v", got)
	}
}
