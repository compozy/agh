package resources

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/testutil"
)

type recordingReconcileEventSink struct {
	mu     sync.Mutex
	events []ReconcileEvent
}

func (s *recordingReconcileEventSink) ObserveReconcileEvent(_ context.Context, event ReconcileEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *recordingReconcileEventSink) count(eventType ReconcileEventType) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, event := range s.events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

type recordingReconcileHealthSink struct {
	mu      sync.Mutex
	updates []ReconcileHealth
}

func (s *recordingReconcileHealthSink) ReportReconcileHealth(_ context.Context, health ReconcileHealth) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, health)
}

func (s *recordingReconcileHealthSink) latest() ReconcileHealth {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.updates) == 0 {
		return ReconcileHealth{}
	}
	return s.updates[len(s.updates)-1]
}

func newTestProjectorRegistration(
	kind ResourceKind,
	dependsOn []ResourceKind,
	build func(context.Context, projectionInput) (ProjectionPlan, error),
	apply func(context.Context, ProjectionPlan) error,
) ProjectorRegistration {
	return &projectorRegistration{
		kind:      kind.Normalize(),
		dependsOn: normalizeKinds(dependsOn),
		build:     build,
		apply:     apply,
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not satisfied before timeout")
}

func TestReconcileDriverSingleFlightCoalescesSameKind(t *testing.T) {
	t.Parallel()

	kernel, _ := openTestKernel(t)
	ctx := testutil.Context(t)
	eventSink := &recordingReconcileEventSink{}

	var mu sync.Mutex
	buildCalls := 0
	firstBuildStarted := make(chan struct{})
	releaseFirstBuild := make(chan struct{})

	driver, err := NewReconcileDriver(
		kernel,
		testDaemonActor(),
		[]ProjectorRegistration{
			newTestProjectorRegistration(testResourceKind, nil,
				func(_ context.Context, _ projectionInput) (ProjectionPlan, error) {
					mu.Lock()
					buildCalls++
					call := buildCalls
					mu.Unlock()

					if call == 1 {
						close(firstBuildStarted)
						<-releaseFirstBuild
					}
					return testPlan{kind: testResourceKind, revision: int64(call), operations: 1}, nil
				},
				func(context.Context, ProjectionPlan) error { return nil },
			),
		},
		WithReconcileEventSink(eventSink),
	)
	if err != nil {
		t.Fatalf("NewReconcileDriver() error = %v", err)
	}
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if closeErr := driver.Close(closeCtx); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	if err := driver.Trigger(ctx, testResourceKind, ReconcileReasonWrite); err != nil {
		t.Fatalf("Trigger(first) error = %v", err)
	}
	<-firstBuildStarted

	for i := range 4 {
		if err := driver.Trigger(ctx, testResourceKind, ReconcileReasonWrite); err != nil {
			t.Fatalf("Trigger(coalesced %d) error = %v", i, err)
		}
	}

	close(releaseFirstBuild)

	waitForCondition(t, time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return buildCalls == 2
	})
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	gotBuildCalls := buildCalls
	mu.Unlock()
	if gotBuildCalls != 2 {
		t.Fatalf("buildCalls = %d, want 2", gotBuildCalls)
	}
	if got, wantMin := eventSink.count(ReconcileEventCoalesced), 1; got < wantMin {
		t.Fatalf("coalesced events = %d, want at least %d", got, wantMin)
	}
}

func TestReconcileDriverPropagatesTimeoutToProjectorContexts(t *testing.T) {
	t.Parallel()

	kernel, _ := openTestKernel(t)
	timeout := 30 * time.Millisecond

	var mu sync.Mutex
	var capturedDeadline time.Time
	var buildErr error

	driver, err := NewReconcileDriver(
		kernel,
		testDaemonActor(),
		[]ProjectorRegistration{
			newTestProjectorRegistration(testResourceKind, nil,
				func(ctx context.Context, _ projectionInput) (ProjectionPlan, error) {
					deadline, ok := ctx.Deadline()
					if !ok {
						t.Fatal("Build() context missing deadline")
					}

					mu.Lock()
					capturedDeadline = deadline
					mu.Unlock()

					<-ctx.Done()
					buildErr = ctx.Err()
					return nil, ctx.Err()
				},
				func(context.Context, ProjectionPlan) error { return nil },
			),
		},
		WithReconcileTimeout(timeout),
	)
	if err != nil {
		t.Fatalf("NewReconcileDriver() error = %v", err)
	}
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if closeErr := driver.Close(closeCtx); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	startedAt := time.Now()
	err = driver.RunBoot(testutil.Context(t))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("RunBoot() error = %v, want context.DeadlineExceeded", err)
	}
	if !errors.Is(buildErr, context.DeadlineExceeded) {
		t.Fatalf("Build() error = %v, want context.DeadlineExceeded", buildErr)
	}

	mu.Lock()
	deadline := capturedDeadline
	mu.Unlock()
	if deadline.IsZero() {
		t.Fatal("captured deadline was not set")
	}

	remaining := deadline.Sub(startedAt)
	if remaining <= 0 || remaining > timeout+50*time.Millisecond {
		t.Fatalf("deadline offset = %s, want within (0,%s]", remaining, timeout+50*time.Millisecond)
	}
}

func TestReconcileDriverOpensDegradedCircuitAndWaitsForFreshWrite(t *testing.T) {
	t.Parallel()

	kernel, _ := openTestKernel(t)
	ctx := testutil.Context(t)
	eventSink := &recordingReconcileEventSink{}
	healthSink := &recordingReconcileHealthSink{}

	var mu sync.Mutex
	buildCalls := 0
	firstBuildStarted := make(chan struct{})
	releaseFirstBuild := make(chan struct{})

	driver, err := NewReconcileDriver(
		kernel,
		testDaemonActor(),
		[]ProjectorRegistration{
			newTestProjectorRegistration(testResourceKind, nil,
				func(_ context.Context, _ projectionInput) (ProjectionPlan, error) {
					mu.Lock()
					buildCalls++
					call := buildCalls
					mu.Unlock()

					if call == 1 {
						close(firstBuildStarted)
						<-releaseFirstBuild
					}
					return nil, errors.New("boom")
				},
				func(context.Context, ProjectionPlan) error { return nil },
			),
		},
		WithReconcileCoalesceWindow(20*time.Millisecond),
		WithReconcileFailureThreshold(1),
		WithReconcileDegradedBackoff(time.Hour),
		WithReconcileEventSink(eventSink),
		WithReconcileHealthSink(healthSink),
	)
	if err != nil {
		t.Fatalf("NewReconcileDriver() error = %v", err)
	}
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if closeErr := driver.Close(closeCtx); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	if err := driver.Trigger(ctx, testResourceKind, ReconcileReasonWrite); err != nil {
		t.Fatalf("Trigger(first) error = %v", err)
	}
	<-firstBuildStarted
	if err := driver.Trigger(ctx, testResourceKind, ReconcileReasonWrite); err != nil {
		t.Fatalf("Trigger(coalesced) error = %v", err)
	}

	close(releaseFirstBuild)
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	gotBuildCalls := buildCalls
	mu.Unlock()
	if gotBuildCalls != 1 {
		t.Fatalf("buildCalls after degraded failure = %d, want 1", gotBuildCalls)
	}
	if got := eventSink.count(ReconcileEventDegraded); got != 1 {
		t.Fatalf("degraded events = %d, want 1", got)
	}
	if got := healthSink.latest().Status; got != ReconcileHealthStatusDegraded {
		t.Fatalf("latest health status = %q, want %q", got, ReconcileHealthStatusDegraded)
	}

	if err := driver.Trigger(ctx, testResourceKind, ReconcileReasonWrite); err != nil {
		t.Fatalf("Trigger(fresh write) error = %v", err)
	}

	waitForCondition(t, time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return buildCalls == 2
	})
}

func TestReconcileDriverSchedulesReverseDependenciesAfterWritesOnly(t *testing.T) {
	t.Parallel()

	t.Run("root write fans out to dependents in order", func(t *testing.T) {
		t.Parallel()

		kernel, _ := openTestKernel(t)
		ctx := testutil.Context(t)

		var mu sync.Mutex
		var order []ResourceKind
		recordOrder := func(kind ResourceKind) {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, kind)
		}

		driver, err := NewReconcileDriver(
			kernel,
			testDaemonActor(),
			[]ProjectorRegistration{
				newTestProjectorRegistration(bundleKind, nil,
					func(context.Context, projectionInput) (ProjectionPlan, error) {
						recordOrder(bundleKind)
						return testPlan{kind: bundleKind, revision: 1, operations: 1}, nil
					},
					func(context.Context, ProjectionPlan) error { return nil },
				),
				newTestProjectorRegistration(bundleActivationKind, []ResourceKind{bundleKind},
					func(context.Context, projectionInput) (ProjectionPlan, error) {
						recordOrder(bundleActivationKind)
						return testPlan{kind: bundleActivationKind, revision: 1, operations: 1}, nil
					},
					func(context.Context, ProjectionPlan) error { return nil },
				),
				newTestProjectorRegistration(ResourceKind("automation.job"), nil,
					func(context.Context, projectionInput) (ProjectionPlan, error) {
						recordOrder(ResourceKind("automation.job"))
						return testPlan{kind: ResourceKind("automation.job"), revision: 1, operations: 1}, nil
					},
					func(context.Context, ProjectionPlan) error { return nil },
				),
			},
		)
		if err != nil {
			t.Fatalf("NewReconcileDriver() error = %v", err)
		}
		t.Cleanup(func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if closeErr := driver.Close(closeCtx); closeErr != nil {
				t.Fatalf("Close() error = %v", closeErr)
			}
		})

		if err := driver.Trigger(ctx, bundleKind, ReconcileReasonWrite); err != nil {
			t.Fatalf("Trigger(bundle) error = %v", err)
		}

		waitForCondition(t, time.Second, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(order) == 2
		})

		mu.Lock()
		gotOrder := append([]ResourceKind(nil), order...)
		mu.Unlock()
		wantOrder := []ResourceKind{bundleKind, bundleActivationKind}
		if len(gotOrder) != len(wantOrder) {
			t.Fatalf("order = %#v, want %#v", gotOrder, wantOrder)
		}
		for idx := range wantOrder {
			if gotOrder[idx] != wantOrder[idx] {
				t.Fatalf("order[%d] = %q, want %q (full=%#v)", idx, gotOrder[idx], wantOrder[idx], gotOrder)
			}
		}
	})

	t.Run("dependent write does not reverse-trigger dependencies", func(t *testing.T) {
		t.Parallel()

		kernel, _ := openTestKernel(t)
		ctx := testutil.Context(t)

		var mu sync.Mutex
		var order []ResourceKind
		recordOrder := func(kind ResourceKind) {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, kind)
		}

		driver, err := NewReconcileDriver(
			kernel,
			testDaemonActor(),
			[]ProjectorRegistration{
				newTestProjectorRegistration(bundleKind, nil,
					func(context.Context, projectionInput) (ProjectionPlan, error) {
						recordOrder(bundleKind)
						return testPlan{kind: bundleKind, revision: 1, operations: 1}, nil
					},
					func(context.Context, ProjectionPlan) error { return nil },
				),
				newTestProjectorRegistration(bundleActivationKind, []ResourceKind{bundleKind},
					func(context.Context, projectionInput) (ProjectionPlan, error) {
						recordOrder(bundleActivationKind)
						return testPlan{kind: bundleActivationKind, revision: 1, operations: 1}, nil
					},
					func(context.Context, ProjectionPlan) error { return nil },
				),
			},
		)
		if err != nil {
			t.Fatalf("NewReconcileDriver() error = %v", err)
		}
		t.Cleanup(func() {
			closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if closeErr := driver.Close(closeCtx); closeErr != nil {
				t.Fatalf("Close() error = %v", closeErr)
			}
		})

		if err := driver.Trigger(ctx, bundleActivationKind, ReconcileReasonWrite); err != nil {
			t.Fatalf("Trigger(bundle.activation) error = %v", err)
		}

		waitForCondition(t, time.Second, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(order) == 1
		})

		mu.Lock()
		gotOrder := append([]ResourceKind(nil), order...)
		mu.Unlock()
		if len(gotOrder) != 1 || gotOrder[0] != bundleActivationKind {
			t.Fatalf("order = %#v, want only %q", gotOrder, bundleActivationKind)
		}
	})
}

func TestReconcileDriverValidationAndLifecycleErrors(t *testing.T) {
	t.Parallel()

	t.Run("registered projectors require raw store", func(t *testing.T) {
		t.Parallel()

		_, err := NewReconcileDriver(
			nil,
			testDaemonActor(),
			[]ProjectorRegistration{
				newTestProjectorRegistration(testResourceKind, nil,
					func(context.Context, projectionInput) (ProjectionPlan, error) {
						return testPlan{kind: testResourceKind, revision: 1, operations: 1}, nil
					},
					func(context.Context, ProjectionPlan) error { return nil },
				),
			},
		)
		if err == nil {
			t.Fatal("NewReconcileDriver() error = nil, want raw-store validation failure")
		}
	})

	t.Run("closed and unknown kinds are rejected", func(t *testing.T) {
		t.Parallel()

		driver, err := NewReconcileDriver(nil, MutationActor{}, nil)
		if err != nil {
			t.Fatalf("NewReconcileDriver() error = %v", err)
		}

		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := driver.Close(closeCtx); err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		if err := driver.Trigger(testutil.Context(t), testResourceKind, ReconcileReasonWrite); err == nil {
			t.Fatal("Trigger(closed) error = nil, want closed-driver failure")
		}
	})

	t.Run("reason validation rejects unsupported values", func(t *testing.T) {
		t.Parallel()

		if err := ReconcileReason("invalid").Validate("reason"); !errors.Is(err, ErrValidation) {
			t.Fatalf("ReconcileReason.Validate() error = %v, want ErrValidation", err)
		}
	})
}
