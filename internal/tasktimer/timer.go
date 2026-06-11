package tasktimer

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/ca-x/nowledge-mem-snap/internal/config"
	"github.com/ca-x/nowledge-mem-snap/internal/schedulecalc"
	"github.com/lib-x/timewheel/scheduler"
)

const (
	wheelInterval = 250 * time.Millisecond
	wheelSlots    = 240
)

var errInvalidSchedule = errors.New("invalid schedule")

type RunFunc func(context.Context, config.TaskConfig) error

type Entry struct {
	Task        config.TaskConfig
	Schedule    config.ScheduleConfig
	HasSchedule bool
}

type Timer struct {
	logger     *slog.Logger
	onOnceDone func(taskKey string) error

	scheduler *scheduler.Scheduler[string, scheduledEntry]

	mu      sync.RWMutex
	entries map[string]scheduledEntry
	nextID  uint64

	startOnce sync.Once
	stopOnce  sync.Once
}

type scheduledEntry struct {
	Entry
	version uint64
}

func New(run RunFunc, logger *slog.Logger, onOnceDone func(taskKey string) error) *Timer {
	if logger == nil {
		logger = slog.Default()
	}
	t := &Timer{
		logger:     logger,
		onOnceDone: onOnceDone,
		entries:    make(map[string]scheduledEntry),
	}
	s, err := scheduler.NewScheduler[string, scheduledEntry](
		scheduler.Options[string, scheduledEntry]{
			Next:             t.nextRun,
			Run:              runEntry(run),
			OnFinish:         t.finish,
			ReschedulePolicy: scheduler.RescheduleAfterFinish,
		},
		scheduler.WithWheel(wheelInterval, wheelSlots),
		scheduler.WithCancelRunningOnRemove(true),
		scheduler.WithCancelRunningOnReplace(true),
		scheduler.WithWaitRunningOnClose(true),
	)
	if err != nil {
		panic(err)
	}
	t.scheduler = s
	return t
}

func (t *Timer) Start(parent context.Context) {
	t.startOnce.Do(func() {
		if parent == nil {
			parent = context.Background()
		}
		if err := t.scheduler.Start(parent); err != nil {
			t.logger.Warn("failed to start task timer", "error", err)
		}
	})
}

func (t *Timer) Stop() {
	t.stopOnce.Do(func() {
		if err := t.scheduler.Close(); err != nil {
			t.logger.Warn("failed to stop task timer", "error", err)
		}
	})
}

func (t *Timer) ReplaceAll(entries []Entry) {
	items := make([]scheduler.Item[string, scheduledEntry], 0, len(entries))
	next := make(map[string]scheduledEntry, len(entries))

	t.mu.Lock()
	for _, entry := range cloneEntries(entries) {
		data := scheduledEntry{Entry: entry, version: t.nextVersionLocked()}
		next[entry.Task.Key] = data
		items = append(items, scheduler.Item[string, scheduledEntry]{
			Key:  entry.Task.Key,
			Data: data,
		})
	}
	t.entries = next
	t.mu.Unlock()

	if err := t.scheduler.ReplaceAll(items); err != nil {
		t.logger.Warn("failed to replace scheduled tasks", "error", err)
	}
}

func (t *Timer) Upsert(entry Entry) {
	data := scheduledEntry{Entry: entry}

	t.mu.Lock()
	data.version = t.nextVersionLocked()
	t.entries[entry.Task.Key] = data
	t.mu.Unlock()

	if err := t.scheduler.Upsert(scheduler.Item[string, scheduledEntry]{
		Key:  entry.Task.Key,
		Data: data,
	}); err != nil {
		t.logger.Warn("failed to upsert scheduled task", "task", entry.Task.Key, "error", err)
	}
}

func (t *Timer) Remove(taskKey string) {
	if taskKey == "" {
		return
	}

	t.mu.Lock()
	delete(t.entries, taskKey)
	t.mu.Unlock()

	if err := t.scheduler.Remove(taskKey); err != nil {
		t.logger.Warn("failed to remove scheduled task", "task", taskKey, "error", err)
	}
}

func (t *Timer) Snapshot() map[string]config.TaskRuntime {
	snapshot := t.scheduler.Snapshot()
	if snapshot == nil {
		return nil
	}

	now := time.Now().In(time.Local)
	out := make(map[string]config.TaskRuntime, len(snapshot))

	t.mu.RLock()
	defer t.mu.RUnlock()

	for key, runtime := range snapshot {
		entry, ok := t.entries[key]
		if !ok {
			continue
		}
		out[key] = toTaskRuntime(now, entry.Entry, runtime)
	}
	return out
}

func (t *Timer) nextRun(now time.Time, _ string, entry scheduledEntry) (time.Time, bool, error) {
	runtime := schedulecalc.Runtime(now.In(time.Local), entry.Task, entry.Schedule, entry.HasSchedule)
	switch runtime.Status {
	case config.TaskRuntimeStatusScheduled:
		if runtime.NextRunAt == nil {
			return time.Time{}, false, errInvalidSchedule
		}
		return *runtime.NextRunAt, true, nil
	case config.TaskRuntimeStatusInvalidSchedule:
		return time.Time{}, false, errInvalidSchedule
	default:
		return time.Time{}, false, nil
	}
}

func (t *Timer) finish(taskKey string, entry scheduledEntry, runErr error) {
	if runErr != nil {
		t.logger.Warn("scheduled task run finished with error", "task", taskKey, "error", runErr)
	}
	if entry.Schedule.Type != "once" {
		return
	}
	if !t.isCurrent(taskKey, entry.version) {
		return
	}

	entry.Task.Enabled = false

	t.mu.Lock()
	if current, ok := t.entries[taskKey]; !ok || current.version != entry.version {
		t.mu.Unlock()
		return
	}
	entry.version = t.nextVersionLocked()
	t.entries[taskKey] = entry
	t.mu.Unlock()

	if err := t.scheduler.Upsert(scheduler.Item[string, scheduledEntry]{
		Key:  taskKey,
		Data: entry,
	}); err != nil {
		t.logger.Warn("failed to update one-time task runtime", "task", taskKey, "error", err)
	}

	if t.onOnceDone != nil {
		if err := t.onOnceDone(taskKey); err != nil {
			t.logger.Warn("failed to disable one-time task", "task", taskKey, "schedule", entry.Schedule.Key, "error", err)
		} else {
			t.logger.Info("one-time task disabled", "task", taskKey, "schedule", entry.Schedule.Key)
		}
	}
}

func (t *Timer) isCurrent(taskKey string, version uint64) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	current, ok := t.entries[taskKey]
	return ok && current.version == version
}

func (t *Timer) nextVersionLocked() uint64 {
	t.nextID++
	return t.nextID
}

func runEntry(run RunFunc) scheduler.RunFunc[string, scheduledEntry] {
	return func(ctx context.Context, _ string, entry scheduledEntry) error {
		if run == nil {
			return nil
		}
		return run(ctx, entry.Task)
	}
}

func toTaskRuntime(now time.Time, entry Entry, runtime scheduler.Runtime) config.TaskRuntime {
	switch runtime.State {
	case scheduler.StateRunning:
		return config.TaskRuntime{Status: config.TaskRuntimeStatusRunning}
	case scheduler.StatePending:
		return config.TaskRuntime{
			Status:    config.TaskRuntimeStatusScheduled,
			NextRunAt: cloneTime(runtime.NextRunAt),
		}
	default:
		return schedulecalc.Runtime(now, entry.Task, entry.Schedule, entry.HasSchedule)
	}
}

func cloneEntries(entries []Entry) []Entry {
	out := make([]Entry, len(entries))
	copy(out, entries)
	return out
}

func cloneTime(in *time.Time) *time.Time {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
