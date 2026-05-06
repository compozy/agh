package scheduler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestRunOnceWakesOnlyEligibleIdleSessions(t *testing.T) {
	base := time.Date(2026, 4, 26, 9, 0, 0, 0, time.UTC)
	source := &fakeTaskSource{
		pending: []RunSnapshot{workSnapshot("task-1", "run-1", taskpkg.ScopeWorkspace, "ws-1", []string{"go"}, base)},
		active: []taskpkg.Run{{
			ID:        "active-1",
			Status:    taskpkg.TaskRunStatusRunning,
			SessionID: "sess-busy",
		}},
	}
	sessions := &fakeSessionSource{sessions: []SessionSnapshot{
		sessionSnapshot("sess-busy", "ws-1", "active", false, []string{"go"}, base.Add(time.Second)),
		sessionSnapshot("sess-prompting", "ws-1", "active", true, []string{"go"}, base.Add(2*time.Second)),
		sessionSnapshot("sess-wrong-workspace", "ws-2", "active", false, []string{"go"}, base.Add(3*time.Second)),
		sessionSnapshot("sess-missing-capability", "ws-1", "active", false, []string{"docs"}, base.Add(4*time.Second)),
		sessionSnapshot("sess-stopped", "ws-1", "stopped", false, []string{"go"}, base.Add(5*time.Second)),
		sessionSnapshot("sess-idle", "ws-1", "active", false, []string{"go", "sqlite"}, base.Add(6*time.Second)),
	}}
	waker := &fakeWaker{}
	scheduler := newTestScheduler(t, source, sessions, waker, WithClock(clockwork.NewFakeClockAt(base)))

	result, err := scheduler.RunOnce(testutil.Context(t))
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.WakeAttempts != 1 || result.WakeSucceeded != 1 {
		t.Fatalf("wake result = %#v, want one successful wake", result)
	}
	targets := waker.targetsSnapshot()
	if got, want := len(targets), 1; got != want {
		t.Fatalf("wake targets = %d, want %d", got, want)
	}
	if got, want := targets[0].Session.ID, "sess-idle"; got != want {
		t.Fatalf("woken session = %q, want %q", got, want)
	}
	if got, want := targets[0].Work.Run.ID, "run-1"; got != want {
		t.Fatalf("woken run = %q, want %q", got, want)
	}
}

func TestRunOnceRecordsNoMatchWithoutWakeMutation(t *testing.T) {
	base := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)
	source := &fakeTaskSource{
		pending: []RunSnapshot{workSnapshot("task-1", "run-1", taskpkg.ScopeWorkspace, "ws-1", []string{"go"}, base)},
	}
	sessions := &fakeSessionSource{sessions: []SessionSnapshot{
		sessionSnapshot("sess-docs", "ws-1", "active", false, []string{"docs"}, base),
	}}
	waker := &fakeWaker{}
	scheduler := newTestScheduler(t, source, sessions, waker, WithClock(clockwork.NewFakeClockAt(base)))

	result, err := scheduler.RunOnce(testutil.Context(t))
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.NoMatchRuns != 1 {
		t.Fatalf("NoMatchRuns = %d, want 1", result.NoMatchRuns)
	}
	if got, want := result.NoMatchRunIDs, []string{"run-1"}; !slices.Equal(got, want) {
		t.Fatalf("NoMatchRunIDs = %v, want %v", got, want)
	}
	if got := len(waker.targetsSnapshot()); got != 0 {
		t.Fatalf("wake targets = %d, want 0", got)
	}
	stats := scheduler.Stats()
	if stats.NoMatchRuns != 1 || stats.WakeAttempts != 0 {
		t.Fatalf("stats = %#v, want one no-match and no wake attempts", stats)
	}
}

func TestRunOnceRequiresTaskOwnerMatch(t *testing.T) {
	t.Parallel()

	t.Run("Should wake only the session matching a pool-owned task agent", func(t *testing.T) {
		t.Parallel()

		base := time.Date(2026, 5, 6, 10, 15, 0, 0, time.UTC)
		work := workSnapshot("task-owner", "run-owner", taskpkg.ScopeWorkspace, "ws-1", nil, base)
		work.Task.Owner = &taskpkg.Ownership{Kind: taskpkg.OwnerKindPool, Ref: "frontend-engineer-agent"}
		work.Run.CoordinationChannelID = "design-review"
		wrongOwner := sessionSnapshot("sess-analytics", "ws-1", "active", false, nil, base)
		wrongOwner.AgentName = "analytics-engineer-agent"
		wrongOwner.Channel = "design-review"
		matchingOwner := sessionSnapshot("sess-frontend", "ws-1", "active", false, nil, base.Add(time.Second))
		matchingOwner.AgentName = "frontend-engineer-agent"
		matchingOwner.Channel = "design-review"
		source := &fakeTaskSource{pending: []RunSnapshot{work}}
		sessions := &fakeSessionSource{sessions: []SessionSnapshot{wrongOwner, matchingOwner}}
		waker := &fakeWaker{}
		scheduler := newTestScheduler(t, source, sessions, waker, WithClock(clockwork.NewFakeClockAt(base)))

		result, err := scheduler.RunOnce(testutil.Context(t))
		if err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
		if result.WakeSucceeded != 1 {
			t.Fatalf("WakeSucceeded = %d, want 1 (result %#v)", result.WakeSucceeded, result)
		}
		targets := waker.targetsSnapshot()
		if got, want := len(targets), 1; got != want {
			t.Fatalf("wake targets = %d, want %d", got, want)
		}
		if got, want := targets[0].Session.ID, "sess-frontend"; got != want {
			t.Fatalf("woken session = %q, want %q", got, want)
		}
	})
}

func TestRunOnceDelegatesExpiredLeaseRecovery(t *testing.T) {
	base := time.Date(2026, 4, 26, 11, 0, 0, 0, time.UTC)
	source := &fakeTaskSource{
		recovered: []taskpkg.ExpiredLeaseRecoveryResult{{
			Run:               taskpkg.Run{ID: "run-recovered", Status: taskpkg.TaskRunStatusQueued},
			PreviousSessionID: "sess-old",
			Reason:            "scheduler_sweep",
		}},
	}
	scheduler := newTestScheduler(
		t,
		source,
		&fakeSessionSource{},
		&fakeWaker{},
		WithClock(clockwork.NewFakeClockAt(base)),
		WithSweepReason("scheduler_sweep"),
		WithSweepLimit(7),
	)

	result, err := scheduler.RunOnce(testutil.Context(t))
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.RecoveredLeases != 1 {
		t.Fatalf("RecoveredLeases = %d, want 1", result.RecoveredLeases)
	}
	calls := source.recoveryCallsSnapshot()
	if got, want := len(calls), 1; got != want {
		t.Fatalf("recovery calls = %d, want %d", got, want)
	}
	if calls[0].Reason != "scheduler_sweep" || calls[0].Limit != 7 || !calls[0].Now.Equal(base) {
		t.Fatalf("recovery request = %#v, want configured reason, limit, and clock time", calls[0])
	}
	if got := source.actorsSnapshot()[0].Actor.Kind; got != taskpkg.ActorKindDaemon {
		t.Fatalf("recovery actor kind = %q, want daemon", got)
	}
}

func TestRunOnceReturnsRecoveryAndWakeErrors(t *testing.T) {
	base := time.Date(2026, 4, 26, 11, 30, 0, 0, time.UTC)
	source := &fakeTaskSource{
		pending:    []RunSnapshot{workSnapshot("task-1", "run-1", taskpkg.ScopeWorkspace, "ws-1", nil, base)},
		recoverErr: errors.New("recover failed"),
	}
	waker := &fakeWaker{err: errors.New("wake failed")}
	scheduler := newTestScheduler(
		t,
		source,
		&fakeSessionSource{sessions: []SessionSnapshot{
			sessionSnapshot("sess-1", "ws-1", "active", false, nil, base),
		}},
		waker,
		WithClock(clockwork.NewFakeClockAt(base)),
		WithWakeReason("wake-test"),
	)

	result, err := scheduler.RunOnce(testutil.Context(t))
	if err == nil {
		t.Fatal("RunOnce() error = nil, want joined recovery and wake errors")
	}
	if result.WakeAttempts != 1 || result.WakeFailed != 1 {
		t.Fatalf("wake result = %#v, want one failed wake", result)
	}
	stats := scheduler.Stats()
	if stats.RecoveryErrors != 1 || stats.WakeFailed != 1 {
		t.Fatalf("stats = %#v, want one recovery error and one wake failure", stats)
	}
	if stats.LastRecoveryError == "" || stats.LastWakeError == "" {
		t.Fatalf("stats = %#v, want last error fields recorded", stats)
	}
}

func TestRebuildClearsEphemeralWakeState(t *testing.T) {
	base := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	source := &fakeTaskSource{
		pending: []RunSnapshot{workSnapshot("task-1", "run-1", taskpkg.ScopeWorkspace, "ws-1", nil, base)},
	}
	sessions := &fakeSessionSource{sessions: []SessionSnapshot{
		sessionSnapshot("sess-1", "ws-1", "active", false, nil, base),
	}}
	waker := &fakeWaker{}
	scheduler := newTestScheduler(t, source, sessions, waker, WithClock(clockwork.NewFakeClockAt(base)))

	if _, err := scheduler.RunOnce(testutil.Context(t)); err != nil {
		t.Fatalf("first RunOnce() error = %v", err)
	}
	if _, err := scheduler.RunOnce(testutil.Context(t)); err != nil {
		t.Fatalf("second RunOnce() error = %v", err)
	}
	if got, want := len(waker.targetsSnapshot()), 1; got != want {
		t.Fatalf("wakes before rebuild = %d, want %d", got, want)
	}

	rebuild, err := scheduler.Rebuild(testutil.Context(t))
	if err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}
	if rebuild.ClearedWakeKeys != 1 {
		t.Fatalf("ClearedWakeKeys = %d, want 1", rebuild.ClearedWakeKeys)
	}
	if _, err := scheduler.RunOnce(testutil.Context(t)); err != nil {
		t.Fatalf("third RunOnce() error = %v", err)
	}
	if got, want := len(waker.targetsSnapshot()), 2; got != want {
		t.Fatalf("wakes after rebuild = %d, want %d", got, want)
	}
}

func TestWakeCooldownExpiresAndAllowsRepeatNotification(t *testing.T) {
	base := time.Date(2026, 4, 26, 12, 30, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(base)
	source := &fakeTaskSource{
		pending: []RunSnapshot{workSnapshot("task-1", "run-1", taskpkg.ScopeWorkspace, "ws-1", nil, base)},
	}
	sessions := &fakeSessionSource{sessions: []SessionSnapshot{
		sessionSnapshot("sess-1", "ws-1", "active", false, nil, base),
	}}
	waker := &fakeWaker{}
	scheduler := newTestScheduler(
		t,
		source,
		sessions,
		waker,
		WithClock(clock),
		WithWakeCooldown(time.Minute),
	)

	if _, err := scheduler.RunOnce(testutil.Context(t)); err != nil {
		t.Fatalf("first RunOnce() error = %v", err)
	}
	if _, err := scheduler.RunOnce(testutil.Context(t)); err != nil {
		t.Fatalf("second RunOnce() error = %v", err)
	}
	if got, want := len(waker.targetsSnapshot()), 1; got != want {
		t.Fatalf("wakes before cooldown expiry = %d, want %d", got, want)
	}

	clock.Advance(time.Minute)
	if _, err := scheduler.RunOnce(testutil.Context(t)); err != nil {
		t.Fatalf("third RunOnce() error = %v", err)
	}
	if got, want := len(waker.targetsSnapshot()), 2; got != want {
		t.Fatalf("wakes after cooldown expiry = %d, want %d", got, want)
	}
}

func TestRunOnceOrdersGlobalWakeTargetsByPriorityAndSession(t *testing.T) {
	base := time.Date(2026, 4, 26, 12, 45, 0, 0, time.UTC)
	low := workSnapshot("task-low", "run-low", taskpkg.ScopeGlobal, "", nil, base)
	low.Task.Priority = taskpkg.PriorityLow
	urgent := workSnapshot("task-urgent", "run-urgent", taskpkg.ScopeGlobal, "", nil, base.Add(time.Second))
	urgent.Task.Priority = taskpkg.PriorityUrgent
	source := &fakeTaskSource{pending: []RunSnapshot{low, urgent}}
	sessions := &fakeSessionSource{sessions: []SessionSnapshot{
		sessionSnapshot("sess-b", "", "active", false, nil, base),
		sessionSnapshot("sess-a", "", "active", false, nil, base),
	}}
	waker := &fakeWaker{}
	scheduler := newTestScheduler(t, source, sessions, waker, WithClock(clockwork.NewFakeClockAt(base)))

	result, err := scheduler.RunOnce(testutil.Context(t))
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.WakeSucceeded != 2 {
		t.Fatalf("WakeSucceeded = %d, want 2", result.WakeSucceeded)
	}
	targets := waker.targetsSnapshot()
	gotRuns := []string{targets[0].Work.Run.ID, targets[1].Work.Run.ID}
	if want := []string{"run-urgent", "run-low"}; !slices.Equal(gotRuns, want) {
		t.Fatalf("wake run order = %v, want %v", gotRuns, want)
	}
	gotSessions := []string{targets[0].Session.ID, targets[1].Session.ID}
	if want := []string{"sess-a", "sess-b"}; !slices.Equal(gotSessions, want) {
		t.Fatalf("wake session order = %v, want %v", gotSessions, want)
	}
}

func TestRunOnceDispatchesSelectedTargetsAsBatch(t *testing.T) {
	t.Run("Should dispatch selected targets through batch waker", func(t *testing.T) {
		base := time.Date(2026, 4, 26, 12, 50, 0, 0, time.UTC)
		source := &fakeTaskSource{pending: []RunSnapshot{
			workSnapshot("task-1", "run-1", taskpkg.ScopeWorkspace, "ws-1", nil, base),
			workSnapshot("task-2", "run-2", taskpkg.ScopeWorkspace, "ws-1", nil, base.Add(time.Second)),
		}}
		sessions := &fakeSessionSource{sessions: []SessionSnapshot{
			sessionSnapshot("sess-a", "ws-1", "active", false, nil, base),
			sessionSnapshot("sess-b", "ws-1", "active", false, nil, base.Add(time.Second)),
		}}
		waker := &fakeBatchWaker{}
		scheduler := newTestScheduler(t, source, sessions, waker, WithClock(clockwork.NewFakeClockAt(base)))

		result, err := scheduler.RunOnce(testutil.Context(t))
		if err != nil {
			t.Fatalf("RunOnce() error = %v", err)
		}
		if result.WakeAttempts != 2 || result.WakeSucceeded != 2 {
			t.Fatalf("wake result = %#v, want two successful batch wakes", result)
		}
		if got, want := waker.batchCallCount(), 1; got != want {
			t.Fatalf("batch wake calls = %d, want %d", got, want)
		}
		targets := waker.targetsSnapshot()
		if got, want := len(targets), 2; got != want {
			t.Fatalf("batch wake targets = %d, want %d", got, want)
		}
		for _, target := range targets {
			if got, want := target.Reason, defaultWakeReason; got != want {
				t.Fatalf("batch wake reason = %q, want %q", got, want)
			}
		}
	})
}

func TestShutdownCancelsLoopAndPreventsRestart(t *testing.T) {
	base := time.Date(2026, 4, 26, 13, 0, 0, 0, time.UTC)
	clock := clockwork.NewFakeClockAt(base)
	tickSeen := make(chan struct{}, 1)
	source := &fakeTaskSource{recoverCh: tickSeen}
	scheduler := newTestScheduler(
		t,
		source,
		&fakeSessionSource{},
		&fakeWaker{},
		WithClock(clock),
		WithInterval(time.Minute),
	)

	if err := scheduler.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitForClockTimers(t, clock, 1)
	clock.Advance(time.Minute)
	select {
	case <-tickSeen:
	case <-testutil.Context(t).Done():
		t.Fatal("timed out waiting for scheduler tick")
	}

	if err := scheduler.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := scheduler.Start(testutil.Context(t)); !errors.Is(err, ErrStopped) {
		t.Fatalf("Start() after Shutdown error = %v, want ErrStopped", err)
	}
}

func TestNewValidatesRequiredDependencies(t *testing.T) {
	validTasks := &fakeTaskSource{}
	validSessions := &fakeSessionSource{}
	validWaker := &fakeWaker{}

	if _, err := New(nil, validSessions, validWaker); err == nil {
		t.Fatal("New(nil tasks) error = nil, want validation error")
	}
	if _, err := New(validTasks, nil, validWaker); err == nil {
		t.Fatal("New(nil sessions) error = nil, want validation error")
	}
	if _, err := New(validTasks, validSessions, nil); err == nil {
		t.Fatal("New(nil waker) error = nil, want validation error")
	}
	if _, err := New(
		validTasks,
		validSessions,
		validWaker,
		WithActor(taskpkg.ActorContext{}),
	); err == nil {
		t.Fatal("New(invalid actor) error = nil, want validation error")
	}
}

func newTestScheduler(t *testing.T, tasks TaskSource, sessions SessionSource, waker Waker, opts ...Option) *Scheduler {
	t.Helper()

	options := append([]Option{
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	}, opts...)
	scheduler, err := New(tasks, sessions, waker, options...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := scheduler.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})
	return scheduler
}

func waitForClockTimers(t *testing.T, clock *clockwork.FakeClock, count int) {
	t.Helper()

	ctx, cancel := context.WithTimeout(testutil.Context(t), 2*time.Second)
	defer cancel()
	if err := clock.BlockUntilContext(ctx, count); err != nil {
		t.Fatalf("BlockUntilContext() error = %v", err)
	}
}

func workSnapshot(
	taskID string,
	runID string,
	scope taskpkg.Scope,
	workspaceID string,
	required []string,
	queuedAt time.Time,
) RunSnapshot {
	return RunSnapshot{
		Task: taskpkg.Task{
			ID:          taskID,
			Scope:       scope,
			WorkspaceID: workspaceID,
			Status:      taskpkg.TaskStatusReady,
			Priority:    taskpkg.PriorityMedium,
		},
		Run: taskpkg.Run{
			ID:                   runID,
			TaskID:               taskID,
			Status:               taskpkg.TaskRunStatusQueued,
			RequiredCapabilities: append([]string(nil), required...),
			QueuedAt:             queuedAt,
		},
	}
}

func sessionSnapshot(
	id string,
	workspaceID string,
	state string,
	prompting bool,
	capabilities []string,
	createdAt time.Time,
) SessionSnapshot {
	return SessionSnapshot{
		ID:           id,
		WorkspaceID:  workspaceID,
		State:        state,
		Prompting:    prompting,
		Capabilities: append([]string(nil), capabilities...),
		CreatedAt:    createdAt,
	}
}

type fakeTaskSource struct {
	mu            sync.Mutex
	pending       []RunSnapshot
	active        []taskpkg.Run
	recovered     []taskpkg.ExpiredLeaseRecoveryResult
	recoverErr    error
	recoverCh     chan<- struct{}
	recoveryCalls []taskpkg.ExpiredLeaseRecovery
	actors        []taskpkg.ActorContext
}

func (f *fakeTaskSource) PendingRuns(context.Context) ([]RunSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RunSnapshot(nil), f.pending...), nil
}

func (f *fakeTaskSource) ActiveRuns(context.Context) ([]taskpkg.Run, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]taskpkg.Run(nil), f.active...), nil
}

func (f *fakeTaskSource) RecoverExpiredRunLeases(
	_ context.Context,
	recovery taskpkg.ExpiredLeaseRecovery,
	actor taskpkg.ActorContext,
) ([]taskpkg.ExpiredLeaseRecoveryResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recoveryCalls = append(f.recoveryCalls, recovery)
	f.actors = append(f.actors, actor)
	if f.recoverCh != nil {
		select {
		case f.recoverCh <- struct{}{}:
		default:
		}
	}
	if f.recoverErr != nil {
		return nil, f.recoverErr
	}
	return append([]taskpkg.ExpiredLeaseRecoveryResult(nil), f.recovered...), nil
}

func (f *fakeTaskSource) recoveryCallsSnapshot() []taskpkg.ExpiredLeaseRecovery {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]taskpkg.ExpiredLeaseRecovery(nil), f.recoveryCalls...)
}

func (f *fakeTaskSource) actorsSnapshot() []taskpkg.ActorContext {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]taskpkg.ActorContext(nil), f.actors...)
}

type fakeSessionSource struct {
	mu       sync.Mutex
	sessions []SessionSnapshot
}

func (f *fakeSessionSource) Sessions(context.Context) ([]SessionSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]SessionSnapshot(nil), f.sessions...), nil
}

type fakeWaker struct {
	mu      sync.Mutex
	err     error
	targets []WakeTarget
}

func (f *fakeWaker) Wake(_ context.Context, target *WakeTarget) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	if target != nil {
		f.targets = append(f.targets, *target)
	}
	return nil
}

func (f *fakeWaker) targetsSnapshot() []WakeTarget {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]WakeTarget(nil), f.targets...)
}

type fakeBatchWaker struct {
	mu      sync.Mutex
	targets []WakeTarget
	calls   int
}

func (f *fakeBatchWaker) Wake(context.Context, *WakeTarget) error {
	return errors.New("single wake should not be called")
}

func (f *fakeBatchWaker) WakeMany(_ context.Context, targets []WakeTarget) []error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.targets = append(f.targets, targets...)
	return make([]error, len(targets))
}

func (f *fakeBatchWaker) batchCallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeBatchWaker) targetsSnapshot() []WakeTarget {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]WakeTarget(nil), f.targets...)
}
