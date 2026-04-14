package automation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestSchedulerCronStateUsesDeterministicNextRun(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 4, 10, 8, 30, 0, 0, time.UTC)
	fakeClock := clockwork.NewFakeClockAt(baseTime)
	scheduler := newTestScheduler(t, newStubScheduleDispatcher(), WithSchedulerClock(fakeClock))

	job := testJob(AutomationScopeGlobal, "cron-next-run", "")
	job.Schedule = &ScheduleSpec{
		Mode: ScheduleModeCron,
		Expr: "0 9 * * *",
	}

	state, err := scheduler.Register(job)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if !state.Registered {
		t.Fatal("Register().Registered = false, want true")
	}
	if state.NextRun == nil {
		t.Fatal("Register().NextRun = nil, want populated")
	}

	want := time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC)
	if got := *state.NextRun; !got.Equal(want) {
		t.Fatalf("Register().NextRun = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestSchedulerEveryStateUsesIntervalSemantics(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	fakeClock := clockwork.NewFakeClockAt(baseTime)
	scheduler := newTestScheduler(t, newStubScheduleDispatcher(), WithSchedulerClock(fakeClock))

	job := testJob(AutomationScopeGlobal, "every-next-run", "")
	job.Schedule = &ScheduleSpec{
		Mode:     ScheduleModeEvery,
		Interval: "30m",
	}

	state, err := scheduler.Register(job)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if state.NextRun == nil {
		t.Fatal("Register().NextRun = nil, want populated")
	}

	want := baseTime.Add(30 * time.Minute)
	if got := *state.NextRun; !got.Equal(want) {
		t.Fatalf("Register().NextRun = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestSchedulerAtJobUnregistersAfterFiringOnce(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	fakeClock := clockwork.NewFakeClockAt(baseTime)
	dispatcher := newStubScheduleDispatcher()
	scheduler := newTestScheduler(t, dispatcher, WithSchedulerClock(fakeClock))

	job := testJob(AutomationScopeGlobal, "at-once", "")
	job.Schedule = &ScheduleSpec{
		Mode: ScheduleModeAt,
		Time: baseTime.Add(1 * time.Minute).Format(time.RFC3339),
	}

	state, err := scheduler.Register(job)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if !state.Registered {
		t.Fatal("Register().Registered = false, want true")
	}
	if err := scheduler.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	waitForTimers(t, fakeClock, 1)
	fakeClock.Advance(1 * time.Minute)
	dispatcher.waitForDispatchCount(t, 1, 2*time.Second)
	dispatcher.waitForCompletionCount(t, 1, 2*time.Second)

	if _, err := scheduler.State(job.ID); !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("State() error = %v, want ErrScheduledJobNotFound", err)
	}
	if got := len(scheduler.States()); got != 0 {
		t.Fatalf("len(States()) = %d, want 0", got)
	}
}

func TestSchedulerSingletonPreventsOverlap(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	fakeClock := clockwork.NewFakeClockAt(baseTime)
	dispatcher := newStubScheduleDispatcher()
	dispatcher.blockNextDispatch()
	scheduler := newTestScheduler(t, dispatcher, WithSchedulerClock(fakeClock))

	job := testJob(AutomationScopeGlobal, "singleton", "")
	job.Schedule = &ScheduleSpec{
		Mode:     ScheduleModeEvery,
		Interval: "1s",
	}

	if _, err := scheduler.Register(job); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := scheduler.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	waitForTimers(t, fakeClock, 1)
	fakeClock.Advance(1 * time.Second)
	dispatcher.waitForDispatchCount(t, 1, 2*time.Second)

	waitForTimers(t, fakeClock, 1)
	fakeClock.Advance(1 * time.Second)
	dispatcher.assertDispatchCount(t, 1)

	dispatcher.releaseBlockedDispatch()
	dispatcher.waitForCompletionCount(t, 1, 2*time.Second)
	waitForTimers(t, fakeClock, 1)
	fakeClock.Advance(1 * time.Second)
	dispatcher.waitForDispatchCount(t, 2, 2*time.Second)
}

func TestSchedulerDisableAndUnregisterRemoveFutureFires(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	fakeClock := clockwork.NewFakeClockAt(baseTime)
	dispatcher := newStubScheduleDispatcher()
	scheduler := newTestScheduler(t, dispatcher, WithSchedulerClock(fakeClock))

	disabledJob := testJob(AutomationScopeGlobal, "disabled", "")
	disabledJob.Schedule = &ScheduleSpec{
		Mode:     ScheduleModeEvery,
		Interval: "1s",
	}
	if _, err := scheduler.Register(disabledJob); err != nil {
		t.Fatalf("Register(disabledJob) error = %v", err)
	}

	unregisteredJob := testJob(AutomationScopeGlobal, "unregistered", "")
	unregisteredJob.Schedule = &ScheduleSpec{
		Mode:     ScheduleModeEvery,
		Interval: "1s",
	}
	if _, err := scheduler.Register(unregisteredJob); err != nil {
		t.Fatalf("Register(unregisteredJob) error = %v", err)
	}

	if err := scheduler.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	disabledJob.Enabled = false
	state, err := scheduler.Update(disabledJob)
	if err != nil {
		t.Fatalf("Update(disabledJob) error = %v", err)
	}
	if state.Registered {
		t.Fatal("Update(disabledJob).Registered = true, want false")
	}
	if err := scheduler.Unregister(unregisteredJob.ID); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	fakeClock.Advance(3 * time.Second)
	dispatcher.assertDispatchCount(t, 0)
}

func TestSchedulerPastOneTimeJobIsSkippedWithoutRegistration(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	fakeClock := clockwork.NewFakeClockAt(baseTime)
	scheduler := newTestScheduler(t, newStubScheduleDispatcher(), WithSchedulerClock(fakeClock))

	job := testJob(AutomationScopeGlobal, "past-at", "")
	job.Schedule = &ScheduleSpec{
		Mode: ScheduleModeAt,
		Time: baseTime.Add(-1 * time.Minute).Format(time.RFC3339),
	}

	state, err := scheduler.Register(job)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if state.Registered {
		t.Fatal("Register().Registered = true, want false")
	}
	if got := len(scheduler.States()); got != 0 {
		t.Fatalf("len(States()) = %d, want 0", got)
	}
}

func TestNewSchedulerAppliesLocationAndStopTimeoutOptions(t *testing.T) {
	t.Parallel()

	if _, err := NewScheduler(nil); err == nil {
		t.Fatal("NewScheduler(nil dispatcher) error = nil, want non-nil")
	}

	dispatcher := newStubScheduleDispatcher()
	location := time.FixedZone("UTC-3", -3*60*60)
	scheduler, err := NewScheduler(
		dispatcher,
		WithSchedulerClock(clockwork.NewFakeClock()),
		WithSchedulerLocation(location),
		WithSchedulerStopTimeout(3*time.Second),
	)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	t.Cleanup(func() {
		if err := scheduler.Stop(context.Background()); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
	})

	if got, want := scheduler.location, location; got != want {
		t.Fatalf("scheduler.location = %v, want %v", got, want)
	}
	if got, want := scheduler.stopTimeout, 3*time.Second; got != want {
		t.Fatalf("scheduler.stopTimeout = %s, want %s", got, want)
	}
}

func TestSchedulerStartAndShutdownLifecycleGuards(t *testing.T) {
	t.Parallel()

	scheduler := newTestScheduler(t, newStubScheduleDispatcher())

	cancelledCtx, cancel := context.WithCancel(testutil.Context(t))
	cancel()
	if err := scheduler.Start(cancelledCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start(cancelled) error = %v, want context.Canceled", err)
	}
	if err := scheduler.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := scheduler.Start(testutil.Context(t)); err != nil {
		t.Fatalf("Start(second) error = %v", err)
	}
	if err := scheduler.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if err := scheduler.Start(testutil.Context(t)); !errors.Is(err, ErrSchedulerStopped) {
		t.Fatalf("Start(after shutdown) error = %v, want ErrSchedulerStopped", err)
	}
}

func TestSchedulerRegisterAndLookupErrorPaths(t *testing.T) {
	t.Parallel()

	scheduler := newTestScheduler(t, newStubScheduleDispatcher())
	job := testJob(AutomationScopeGlobal, "duplicate", "")

	if _, err := scheduler.Register(job); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, err := scheduler.Register(job); !errors.Is(err, ErrScheduledJobAlreadyRegistered) {
		t.Fatalf("Register(duplicate) error = %v, want ErrScheduledJobAlreadyRegistered", err)
	}
	if _, err := scheduler.State("missing"); !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("State(missing) error = %v, want ErrScheduledJobNotFound", err)
	}

	missingJob := testJob(AutomationScopeGlobal, "missing", "")
	missingJob.ID = "missing"
	if _, err := scheduler.Update(missingJob); !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("Update(missing) error = %v, want ErrScheduledJobNotFound", err)
	}
	if err := scheduler.Unregister("missing"); !errors.Is(err, ErrScheduledJobNotFound) {
		t.Fatalf("Unregister(missing) error = %v, want ErrScheduledJobNotFound", err)
	}
}

func TestSchedulerUpdateReplacesNextRunAndStatesAreSorted(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	fakeClock := clockwork.NewFakeClockAt(baseTime)
	scheduler := newTestScheduler(t, newStubScheduleDispatcher(), WithSchedulerClock(fakeClock))

	jobB := testJob(AutomationScopeGlobal, "b", "")
	jobB.ID = "job-b"
	jobA := testJob(AutomationScopeGlobal, "a", "")
	jobA.ID = "job-a"

	if _, err := scheduler.Register(jobB); err != nil {
		t.Fatalf("Register(jobB) error = %v", err)
	}
	if _, err := scheduler.Register(jobA); err != nil {
		t.Fatalf("Register(jobA) error = %v", err)
	}

	jobB.Schedule = &ScheduleSpec{
		Mode:     ScheduleModeEvery,
		Interval: "2h",
	}
	state, err := scheduler.Update(jobB)
	if err != nil {
		t.Fatalf("Update(jobB) error = %v", err)
	}
	if state.NextRun == nil {
		t.Fatal("Update(jobB).NextRun = nil, want populated")
	}
	if got, want := *state.NextRun, baseTime.Add(2*time.Hour); !got.Equal(want) {
		t.Fatalf("Update(jobB).NextRun = %s, want %s", got.Format(time.RFC3339), want.Format(time.RFC3339))
	}

	states := scheduler.States()
	if got, want := len(states), 2; got != want {
		t.Fatalf("len(States()) = %d, want %d", got, want)
	}
	if got, want := states[0].JobID, "job-a"; got != want {
		t.Fatalf("States()[0].JobID = %q, want %q", got, want)
	}
	if got, want := states[1].JobID, "job-b"; got != want {
		t.Fatalf("States()[1].JobID = %q, want %q", got, want)
	}
}

func TestNormalizeScheduledJobAndPredictNextRunValidation(t *testing.T) {
	t.Parallel()

	if _, err := normalizeScheduledJob(Job{}); err == nil {
		t.Fatal("normalizeScheduledJob(empty) error = nil, want non-nil")
	}

	location := time.UTC
	registeredAt := time.Date(2026, 4, 10, 12, 0, 0, 0, location)
	invalidJob := testJob(AutomationScopeGlobal, "invalid-next-run", "")
	invalidJob.Schedule = &ScheduleSpec{
		Mode:     ScheduleModeEvery,
		Interval: "bad",
	}
	if got := predictNextRun(invalidJob, registeredAt, location); !got.IsZero() {
		t.Fatalf("predictNextRun(invalid every) = %s, want zero", got.Format(time.RFC3339))
	}
}

type stubScheduleDispatcher struct {
	mu             sync.Mutex
	calls          []DispatchRequest
	blocked        bool
	releaseCh      chan struct{}
	dispatchedCh   chan struct{}
	completedCh    chan struct{}
	dispatchResult error
}

func newStubScheduleDispatcher() *stubScheduleDispatcher {
	return &stubScheduleDispatcher{
		dispatchedCh: make(chan struct{}, 32),
		completedCh:  make(chan struct{}, 32),
	}
}

func (d *stubScheduleDispatcher) Dispatch(ctx context.Context, req DispatchRequest) (*Run, error) {
	defer notify(d.completedCh)

	d.mu.Lock()
	d.calls = append(d.calls, req)
	releaseCh := d.releaseCh
	blocked := d.blocked
	dispatchResult := d.dispatchResult
	d.mu.Unlock()

	notify(d.dispatchedCh)
	if blocked && releaseCh != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-releaseCh:
		}
	}
	if dispatchResult != nil {
		return nil, dispatchResult
	}
	return &Run{
		ID:     fmt.Sprintf("run-%d", d.count()),
		JobID:  req.Job.ID,
		Status: RunCompleted,
	}, nil
}

func (d *stubScheduleDispatcher) blockNextDispatch() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.blocked = true
	d.releaseCh = make(chan struct{})
}

func (d *stubScheduleDispatcher) releaseBlockedDispatch() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.releaseCh != nil {
		close(d.releaseCh)
	}
	d.blocked = false
	d.releaseCh = nil
}

func (d *stubScheduleDispatcher) count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.calls)
}

func (d *stubScheduleDispatcher) assertDispatchCount(t *testing.T, want int) {
	t.Helper()
	if got := d.count(); got != want {
		t.Fatalf("dispatch count = %d, want %d", got, want)
	}
}

func (d *stubScheduleDispatcher) waitForDispatchCount(t *testing.T, want int, timeout time.Duration) {
	t.Helper()

	deadline := time.After(timeout)
	for {
		if got := d.count(); got >= want {
			return
		}

		select {
		case <-deadline:
			t.Fatalf("dispatch count did not reach %d within %s; got %d", want, timeout, d.count())
		case <-d.dispatchedCh:
		}
	}
}

func (d *stubScheduleDispatcher) waitForCompletionCount(t *testing.T, want int, timeout time.Duration) {
	t.Helper()

	completed := 0
	deadline := time.After(timeout)
	for completed < want {
		select {
		case <-deadline:
			t.Fatalf("dispatch completion count did not reach %d within %s; got %d", want, timeout, completed)
		case <-d.completedCh:
			completed++
		}
	}
}

func newTestScheduler(t *testing.T, dispatcher ScheduleDispatcher, opts ...SchedulerOption) *Scheduler {
	t.Helper()

	scheduler, err := NewScheduler(dispatcher, append([]SchedulerOption{
		WithSchedulerLogger(slog.Default()),
	}, opts...)...)
	if err != nil {
		t.Fatalf("NewScheduler() error = %v", err)
	}
	t.Cleanup(func() {
		if err := scheduler.Stop(context.Background()); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
	})
	return scheduler
}

func waitForTimers(t *testing.T, clock *clockwork.FakeClock, count int) {
	t.Helper()

	ctx, cancel := context.WithTimeout(testutil.Context(t), 2*time.Second)
	defer cancel()
	if err := clock.BlockUntilContext(ctx, count); err != nil {
		t.Fatalf("BlockUntilContext() error = %v", err)
	}
}
