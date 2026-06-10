package scheduler

import (
	"context"
	"log/slog"
	"sync"

	"github.com/lib-x/nowledge-mem-snap/internal/backup"
	"github.com/lib-x/nowledge-mem-snap/internal/config"
	"github.com/lib-x/nowledge-mem-snap/internal/history"
)

type Manager struct {
	ctx        context.Context
	store      *config.Store
	logger     *slog.Logger
	mu         sync.Mutex
	schedulers map[string]*Scheduler
}

func NewManager(ctx context.Context, store *config.Store, logger *slog.Logger) *Manager {
	return &Manager{
		ctx:        ctx,
		store:      store,
		logger:     logger,
		schedulers: make(map[string]*Scheduler),
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
	old := m.schedulers[tenant]
	delete(m.schedulers, tenant)
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
	sched := New(cfg, runner, m.logger, func(taskKey string) error {
		return m.disableTask(tenant, taskKey)
	})
	sched.Start(m.ctx)

	m.mu.Lock()
	m.schedulers[tenant] = sched
	m.mu.Unlock()
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
	schedulers := make([]*Scheduler, 0, len(m.schedulers))
	for tenant, sched := range m.schedulers {
		delete(m.schedulers, tenant)
		schedulers = append(schedulers, sched)
	}
	m.mu.Unlock()
	for _, sched := range schedulers {
		sched.Stop()
	}
}
