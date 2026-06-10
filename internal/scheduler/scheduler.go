package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/lib-x/nowledge-mem-snap/internal/backup"
	"github.com/lib-x/nowledge-mem-snap/internal/config"
)

type Scheduler struct {
	cfg        config.Config
	runner     *backup.Runner
	logger     *slog.Logger
	onOnceDone func(taskKey string) error
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func New(cfg config.Config, runner *backup.Runner, logger *slog.Logger, onOnceDone func(taskKey string) error) *Scheduler {
	return &Scheduler{cfg: cfg, runner: runner, logger: logger, onOnceDone: onOnceDone}
}

func (s *Scheduler) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	s.cancel = cancel
	for _, task := range s.cfg.Tasks {
		if !task.Enabled {
			continue
		}
		schedule, ok := s.cfg.Schedule(task.ScheduleKey)
		if !ok || !schedule.Enabled {
			continue
		}
		localTask := task
		localSchedule := schedule
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.loop(ctx, localTask, localSchedule)
		}()
	}
}

func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

func (s *Scheduler) loop(ctx context.Context, task config.TaskConfig, schedule config.ScheduleConfig) {
	for {
		next, ok := nextRun(time.Now().In(time.Local), schedule)
		if !ok {
			s.logger.Warn("scheduled backup skipped", "task", task.Key, "schedule", schedule.Key, "type", schedule.Type, "reason", "invalid schedule")
			return
		}
		s.logger.Info("scheduled backup waiting", "task", task.Key, "schedule", schedule.Key, "type", schedule.Type, "next", next, "timezone", next.Location().String())
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
			run, err := s.runner.RunScheduledTask(runCtx, task)
			cancel()
			if err != nil {
				s.logger.Warn("scheduled backup failed", "task", task.Key, "run", run.ID, "error", err)
			} else {
				s.logger.Info("scheduled backup finished", "task", task.Key, "run", run.ID, "status", run.Status)
			}
			if schedule.Type == "once" {
				if s.onOnceDone != nil {
					if err := s.onOnceDone(task.Key); err != nil {
						s.logger.Warn("failed to disable one-time task", "task", task.Key, "schedule", schedule.Key, "error", err)
					} else {
						s.logger.Info("one-time task disabled", "task", task.Key, "schedule", schedule.Key)
					}
				}
				return
			}
		}
	}
}

func nextRun(now time.Time, schedule config.ScheduleConfig) (time.Time, bool) {
	hour, minute, _ := config.ParseClock(schedule.Time)
	base := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
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
		weekday, _ := config.ParseWeekday(schedule.Weekday)
		days := (int(weekday) - int(base.Weekday()) + 7) % 7
		next := base.AddDate(0, 0, days)
		if !next.After(now) {
			next = next.AddDate(0, 0, 7)
		}
		return next, true
	default:
		if !base.After(now) {
			base = base.AddDate(0, 0, 1)
		}
		return base, true
	}
}
