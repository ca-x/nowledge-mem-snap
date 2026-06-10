package schedulecalc

import (
	"time"

	"github.com/lib-x/nowledge-mem-snap/internal/config"
)

func Runtime(now time.Time, task config.TaskConfig, schedule config.ScheduleConfig, hasSchedule bool) config.TaskRuntime {
	if !task.Enabled {
		return config.TaskRuntime{Status: config.TaskRuntimeStatusDisabled}
	}
	if !hasSchedule {
		return config.TaskRuntime{Status: config.TaskRuntimeStatusMissingSchedule}
	}
	if !schedule.Enabled {
		return config.TaskRuntime{Status: config.TaskRuntimeStatusScheduleDisabled}
	}
	next, ok := NextRun(now, schedule)
	if !ok {
		return config.TaskRuntime{Status: config.TaskRuntimeStatusInvalidSchedule}
	}
	return config.TaskRuntime{Status: config.TaskRuntimeStatusScheduled, NextRunAt: &next}
}

func NextRun(now time.Time, schedule config.ScheduleConfig) (time.Time, bool) {
	switch schedule.Type {
	case "once":
		runAt, err := config.ParseScheduleRunAt(schedule.RunAt, now.Location())
		if err != nil {
			return time.Time{}, false
		}
		if runAt.Before(now) {
			return now, true
		}
		return runAt, true
	case "weekly":
		hour, minute, err := config.ParseClock(schedule.Time)
		if err != nil {
			return time.Time{}, false
		}
		weekday, err := config.ParseWeekday(schedule.Weekday)
		if err != nil {
			return time.Time{}, false
		}
		base := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		days := (int(weekday) - int(base.Weekday()) + 7) % 7
		next := base.AddDate(0, 0, days)
		if !next.After(now) {
			next = next.AddDate(0, 0, 7)
		}
		return next, true
	case "daily":
		hour, minute, err := config.ParseClock(schedule.Time)
		if err != nil {
			return time.Time{}, false
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		return next, true
	default:
		return time.Time{}, false
	}
}
