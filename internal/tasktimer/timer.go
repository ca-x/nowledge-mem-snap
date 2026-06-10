package tasktimer

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/lib-x/nowledge-mem-snap/internal/config"
	"github.com/lib-x/nowledge-mem-snap/internal/schedulecalc"
)

type RunFunc func(context.Context, config.TaskConfig) error

type Entry struct {
	Task        config.TaskConfig
	Schedule    config.ScheduleConfig
	HasSchedule bool
}

type Timer struct {
	run        RunFunc
	logger     *slog.Logger
	onOnceDone func(taskKey string) error

	commands chan command
	done     chan struct{}
	cancel   context.CancelFunc

	startOnce sync.Once
	stopOnce  sync.Once
	runWG     sync.WaitGroup
}

func New(run RunFunc, logger *slog.Logger, onOnceDone func(taskKey string) error) *Timer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Timer{
		run:        run,
		logger:     logger,
		onOnceDone: onOnceDone,
		commands:   make(chan command, 64),
		done:       make(chan struct{}),
	}
}

func (t *Timer) Start(parent context.Context) {
	t.startOnce.Do(func() {
		if parent == nil {
			parent = context.Background()
		}
		ctx, cancel := context.WithCancel(parent)
		t.cancel = cancel
		go t.loop(ctx)
	})
}

func (t *Timer) Stop() {
	t.stopOnce.Do(func() {
		if t.cancel != nil {
			t.cancel()
		}
		<-t.done
		t.runWG.Wait()
	})
}

func (t *Timer) ReplaceAll(entries []Entry) {
	t.send(command{kind: commandReplaceAll, entries: cloneEntries(entries)})
}

func (t *Timer) Upsert(entry Entry) {
	t.send(command{kind: commandUpsert, entry: entry})
}

func (t *Timer) Remove(taskKey string) {
	if taskKey == "" {
		return
	}
	t.send(command{kind: commandRemove, key: taskKey})
}

func (t *Timer) Snapshot() map[string]config.TaskRuntime {
	reply := make(chan map[string]config.TaskRuntime, 1)
	if !t.send(command{kind: commandSnapshot, reply: reply}) {
		return nil
	}
	select {
	case snapshot := <-reply:
		return snapshot
	case <-t.done:
		return nil
	}
}

func (t *Timer) send(cmd command) bool {
	select {
	case t.commands <- cmd:
		return true
	case <-t.done:
		return false
	}
}

func (t *Timer) finish(taskKey string, generation uint64, err error) {
	t.send(command{kind: commandFinished, key: taskKey, generation: generation, err: err})
}

type commandKind int

const (
	commandReplaceAll commandKind = iota
	commandUpsert
	commandRemove
	commandSnapshot
	commandFinished
)

type command struct {
	kind       commandKind
	entries    []Entry
	entry      Entry
	key        string
	generation uint64
	err        error
	reply      chan map[string]config.TaskRuntime
}

type taskState struct {
	entry      Entry
	status     string
	nextRunAt  *time.Time
	generation uint64
	running    bool
	cancel     context.CancelFunc
}

type loopState struct {
	states map[string]*taskState
	queue  scheduleHeap
	nextID uint64
}

func (t *Timer) loop(ctx context.Context) {
	state := loopState{states: make(map[string]*taskState)}
	state.queue.init()
	timer := time.NewTimer(time.Hour)
	stopTimer(timer)
	var timerC <-chan time.Time

	resetTimer := func() {
		stopTimer(timer)
		timerC = nil
		if due, ok := state.nextDue(); ok {
			delay := time.Until(due)
			if delay < 0 {
				delay = 0
			}
			timer.Reset(delay)
			timerC = timer.C
		}
	}

	for {
		select {
		case <-ctx.Done():
			stopTimer(timer)
			state.cancelAll()
			close(t.done)
			return
		case cmd := <-t.commands:
			t.handle(&state, cmd)
			state.runDue(ctx, t, time.Now().In(time.Local))
			resetTimer()
		case <-timerC:
			state.runDue(ctx, t, time.Now().In(time.Local))
			resetTimer()
		}
	}
}

func (t *Timer) handle(state *loopState, cmd command) {
	switch cmd.kind {
	case commandReplaceAll:
		state.replaceAll(cmd.entries)
	case commandUpsert:
		state.upsert(cmd.entry)
	case commandRemove:
		state.remove(cmd.key)
	case commandSnapshot:
		cmd.reply <- state.snapshot()
	case commandFinished:
		state.finish(t, cmd.key, cmd.generation, cmd.err)
	}
}

func (s *loopState) replaceAll(entries []Entry) {
	s.cancelAll()
	s.states = make(map[string]*taskState, len(entries))
	s.queue = scheduleHeap{}
	s.queue.init()
	now := time.Now().In(time.Local)
	for _, entry := range entries {
		st := &taskState{entry: entry, generation: s.nextGeneration()}
		s.plan(now, st)
		s.states[entry.Task.Key] = st
	}
}

func (s *loopState) upsert(entry Entry) {
	now := time.Now().In(time.Local)
	if old := s.states[entry.Task.Key]; old != nil {
		old.cancelRun()
	}
	st := &taskState{entry: entry, generation: s.nextGeneration()}
	s.plan(now, st)
	s.states[entry.Task.Key] = st
}

func (s *loopState) remove(taskKey string) {
	st := s.states[taskKey]
	if st != nil {
		st.cancelRun()
	}
	delete(s.states, taskKey)
}

func (s *loopState) finish(t *Timer, taskKey string, generation uint64, err error) {
	st := s.states[taskKey]
	if st == nil || st.generation != generation {
		return
	}
	st.running = false
	st.cancel = nil
	st.generation = s.nextGeneration()
	if err != nil {
		t.logger.Warn("scheduled task run finished with error", "task", taskKey, "error", err)
	}
	if st.entry.Schedule.Type == "once" {
		if t.onOnceDone != nil {
			if err := t.onOnceDone(taskKey); err != nil {
				t.logger.Warn("failed to disable one-time task", "task", taskKey, "schedule", st.entry.Schedule.Key, "error", err)
			} else {
				t.logger.Info("one-time task disabled", "task", taskKey, "schedule", st.entry.Schedule.Key)
			}
		}
		st.entry.Task.Enabled = false
		st.status = config.TaskRuntimeStatusDisabled
		st.nextRunAt = nil
		return
	}
	s.plan(time.Now().In(time.Local), st)
}

func (s *loopState) plan(now time.Time, st *taskState) {
	runtime := schedulecalc.Runtime(now, st.entry.Task, st.entry.Schedule, st.entry.HasSchedule)
	st.status = runtime.Status
	st.nextRunAt = cloneTime(runtime.NextRunAt)
	if runtime.NextRunAt != nil {
		s.queue.push(scheduleItem{key: st.entry.Task.Key, due: *runtime.NextRunAt, generation: st.generation})
	}
}

func (s *loopState) runDue(ctx context.Context, t *Timer, now time.Time) {
	for {
		due, ok := s.nextDue()
		if !ok || due.After(now) {
			return
		}
		item := s.queue.pop()
		st := s.states[item.key]
		if st == nil || st.generation != item.generation || st.running {
			continue
		}
		st.running = true
		st.status = config.TaskRuntimeStatusRunning
		st.nextRunAt = nil
		st.generation = s.nextGeneration()
		runGeneration := st.generation
		runCtx, cancel := context.WithCancel(ctx)
		st.cancel = cancel
		task := st.entry.Task
		t.runWG.Add(1)
		go func() {
			defer t.runWG.Done()
			err := t.run(runCtx, task)
			cancel()
			t.finish(task.Key, runGeneration, err)
		}()
	}
}

func (s *loopState) nextDue() (time.Time, bool) {
	for s.queue.Len() > 0 {
		item := s.queue[0]
		st := s.states[item.key]
		if st != nil && st.generation == item.generation && !st.running {
			return item.due, true
		}
		s.queue.pop()
	}
	return time.Time{}, false
}

func (s *loopState) snapshot() map[string]config.TaskRuntime {
	out := make(map[string]config.TaskRuntime, len(s.states))
	for key, st := range s.states {
		out[key] = config.TaskRuntime{
			Status:    st.status,
			NextRunAt: cloneTime(st.nextRunAt),
		}
	}
	return out
}

func (s *loopState) cancelAll() {
	for _, st := range s.states {
		st.cancelRun()
	}
}

func (s *loopState) nextGeneration() uint64 {
	s.nextID++
	return s.nextID
}

func (st *taskState) cancelRun() {
	if st.cancel != nil {
		st.cancel()
		st.cancel = nil
	}
	st.running = false
}

func stopTimer(timer *time.Timer) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
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
