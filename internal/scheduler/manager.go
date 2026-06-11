package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/backup"
	"github.com/ca-x/nowledge-mem-snap/internal/config"
	"github.com/ca-x/nowledge-mem-snap/internal/history"
	"github.com/ca-x/nowledge-mem-snap/internal/schedulecalc"
	"github.com/ca-x/nowledge-mem-snap/internal/tasktimer"
)

type Manager struct {
	ctx    context.Context
	store  *config.Store
	logger *slog.Logger
	mu     sync.Mutex
	timers map[string]*tasktimer.Timer
}

func NewManager(ctx context.Context, store *config.Store, logger *slog.Logger) *Manager {
	return &Manager{
		ctx:    ctx,
		store:  store,
		logger: logger,
		timers: make(map[string]*tasktimer.Timer),
	}
}

func (m *Manager) StartAll() {
	tenants, err := m.store.ListUsers()
	if err != nil {
		m.logger.Warn("failed to list tenants for scheduler", "error", err)
		return
	}
	for _, tenant := range tenants {
		m.Reload(tenant)
	}
}

func (m *Manager) Reload(tenant string) {
	tenant = config.TenantKey(tenant)
	if tenant == "" {
		return
	}
	m.mu.Lock()
	old := m.timers[tenant]
	delete(m.timers, tenant)
	m.mu.Unlock()
	if old != nil {
		old.Stop()
	}

	cfg, err := m.store.LoadUser(tenant)
	if err != nil {
		m.logger.Warn("failed to load tenant scheduler config", "tenant", tenant, "error", err)
		return
	}
	historyStore := history.NewStoreWithRetention(m.store.Client(), tenant, cfg.HistoryLimit, cfg.HistoryRetentionDays)
	runner := backup.NewRunner(cfg, historyStore, m.logger)
	timer := tasktimer.New(func(ctx context.Context, task config.TaskConfig) error {
		runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
		run, err := runner.RunScheduledTask(runCtx, task)
		if err != nil {
			m.logger.Warn("scheduled backup failed", "tenant", tenant, "task", task.Key, "run", run.ID, "error", err)
			return err
		}
		m.logger.Info("scheduled backup finished", "tenant", tenant, "task", task.Key, "run", run.ID, "status", run.Status)
		return nil
	}, m.logger, func(taskKey string) error {
		return m.disableTask(tenant, taskKey)
	})
	timer.Start(m.ctx)
	timer.ReplaceAll(entries(cfg))

	m.mu.Lock()
	m.timers[tenant] = timer
	m.mu.Unlock()
}

func (m *Manager) Snapshot(tenant string) map[string]config.TaskRuntime {
	tenant = config.TenantKey(tenant)
	if tenant == "" {
		return nil
	}
	m.mu.Lock()
	timer := m.timers[tenant]
	m.mu.Unlock()
	if timer != nil {
		return timer.Snapshot()
	}
	cfg, err := m.store.LoadUser(tenant)
	if err != nil {
		m.logger.Warn("failed to load tenant scheduler snapshot config", "tenant", tenant, "error", err)
		return nil
	}
	return snapshotFromConfig(cfg)
}

func (m *Manager) disableTask(tenant string, taskKey string) error {
	cfg, err := m.store.LoadUser(tenant)
	if err != nil {
		return err
	}
	for i := range cfg.Tasks {
		if cfg.Tasks[i].Key != taskKey {
			continue
		}
		if !cfg.Tasks[i].Enabled {
			return nil
		}
		cfg.Tasks[i].Enabled = false
		return m.store.SaveUser(tenant, cfg)
	}
	return nil
}

func (m *Manager) Stop() {
	m.mu.Lock()
	timers := make([]*tasktimer.Timer, 0, len(m.timers))
	for tenant, timer := range m.timers {
		delete(m.timers, tenant)
		timers = append(timers, timer)
	}
	m.mu.Unlock()
	for _, timer := range timers {
		timer.Stop()
	}
}

func entries(cfg config.Config) []tasktimer.Entry {
	out := make([]tasktimer.Entry, 0, len(cfg.Tasks))
	for _, task := range cfg.Tasks {
		schedule, ok := cfg.Schedule(task.ScheduleKey)
		out = append(out, tasktimer.Entry{
			Task:        task,
			Schedule:    schedule,
			HasSchedule: ok,
		})
	}
	return out
}

func snapshotFromConfig(cfg config.Config) map[string]config.TaskRuntime {
	now := time.Now().In(time.Local)
	out := make(map[string]config.TaskRuntime, len(cfg.Tasks))
	for _, task := range cfg.Tasks {
		schedule, ok := cfg.Schedule(task.ScheduleKey)
		out[task.Key] = schedulecalc.Runtime(now, task, schedule, ok)
	}
	return out
}
