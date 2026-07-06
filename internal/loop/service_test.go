package loop_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/task"
)

func TestServiceTransitionShouldEnforceTruthyStatusFSM(t *testing.T) {
	t.Parallel()

	t.Run("Should reject false done and failed coercions from truthful statuses", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name string
			from loop.Status
			to   loop.Status
		}{
			{name: "Should reject exhausted to done", from: loop.StatusExhausted, to: loop.StatusDone},
			{name: "Should reject stalled to done", from: loop.StatusStalled, to: loop.StatusDone},
			{
				name: "Should reject needs approval to done",
				from: loop.StatusNeedsApproval,
				to:   loop.StatusDone,
			},
			{
				name: "Should reject needs approval to failed without operator stop",
				from: loop.StatusNeedsApproval,
				to:   loop.StatusFailed,
			},
			{name: "Should reject paused to done", from: loop.StatusPaused, to: loop.StatusDone},
			{
				name: "Should reject paused to failed without operator stop",
				from: loop.StatusPaused,
				to:   loop.StatusFailed,
			},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				store := newFakeLoopStore()
				run := seedFakeRun(store, tt.from)
				svc := newTestService(t, store, validDefinition())

				err := svc.Transition(context.Background(), run.ID, tt.to, loop.TransitionCauseContract)
				if !errors.Is(err, loop.ErrInvalidTransition) {
					t.Fatalf("Transition(%s -> %s) error = %v, want ErrInvalidTransition", tt.from, tt.to, err)
				}
				stored := store.mustRun(t, run.ID)
				if stored.Status != tt.from {
					t.Fatalf("stored status = %q, want original %q", stored.Status, tt.from)
				}
			})
		}
	})

	t.Run("Should allow running paused edges and reject illegal paused edges", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		run := seedFakeRun(store, loop.StatusRunning)
		svc := newTestService(t, store, validDefinition())

		err := svc.Transition(
			context.Background(),
			run.ID,
			loop.StatusPaused,
			loop.TransitionCausePauseBoundary,
		)
		if err != nil {
			t.Fatalf("Transition(running -> paused) error = %v", err)
		}
		err = svc.Transition(
			context.Background(),
			run.ID,
			loop.StatusRunning,
			loop.TransitionCauseOperatorResume,
		)
		if err != nil {
			t.Fatalf("Transition(paused -> running) error = %v", err)
		}

		done := seedFakeRun(store, loop.StatusDone)
		err = svc.Transition(
			context.Background(),
			done.ID,
			loop.StatusPaused,
			loop.TransitionCausePauseBoundary,
		)
		if !errors.Is(err, loop.ErrInvalidTransition) {
			t.Fatalf("Transition(done -> paused) error = %v, want ErrInvalidTransition", err)
		}
		watching := seedFakeRun(store, loop.StatusWatching)
		err = svc.Transition(
			context.Background(),
			watching.ID,
			loop.StatusPaused,
			loop.TransitionCausePauseBoundary,
		)
		if !errors.Is(err, loop.ErrInvalidTransition) {
			t.Fatalf("Transition(watching -> paused) error = %v, want ErrInvalidTransition", err)
		}
	})
}

func TestEffectiveConfigShouldMergeLayersAndClampCeilings(t *testing.T) {
	t.Parallel()

	t.Run("Should merge definition defaults loop config and per run overrides", func(t *testing.T) {
		t.Parallel()

		def := validDefinition()
		def.Contract.IterationCap = 3
		def.Contract.NoProgress.Window = 2
		def.Contract.Budget.Tokens = 100
		def.Contract.Budget.WallClockSec = 10
		requireNode(t, &def, "fan").MaxParallel = 2
		appendGate(&def, dsl.Node{
			ID:           "review_gate",
			Class:        dsl.NodeClassControl,
			Kind:         string(dsl.ControlGate),
			MaxRevisions: 2,
		})
		resolved := compileDefinition(t, def)
		defaults := loop.LoopDefaults{
			Delivery: loop.LoopConfig{
				IterationCap:     new(6),
				NoProgressWindow: new(99),
				FanOutWidth:      new(99),
				GateMaxRevisions: new(99),
			},
		}
		stored := loop.LoopConfig{
			IterationCap:     new(7),
			NoProgressWindow: new(8),
			FanOutWidth:      new(12),
		}
		escalate := dsl.BudgetExceededEscalate
		perRun := loop.LoopConfig{
			IterationCap:     new(8),
			BudgetOnExceeded: &escalate,
			FanOutWidth:      new(99),
			GateMaxRevisions: new(99),
		}

		effective, err := loop.ResolveEffectiveConfig(resolved, defaults, &stored, perRun)
		if err != nil {
			t.Fatalf("ResolveEffectiveConfig() error = %v", err)
		}
		if effective.IterationCap != 8 {
			t.Fatalf("IterationCap = %d, want per-run override 8", effective.IterationCap)
		}
		if effective.NoProgressWindow != 8 {
			t.Fatalf("NoProgressWindow = %d, want stored override 8", effective.NoProgressWindow)
		}
		if effective.FanOutWidth != loop.LoopMaxFanoutWidth {
			t.Fatalf(
				"FanOutWidth = %d, want clamped ceiling %d",
				effective.FanOutWidth,
				loop.LoopMaxFanoutWidth,
			)
		}
		if effective.GateMaxRevisions != loop.LoopMaxGateRevisions {
			t.Fatalf(
				"GateMaxRevisions = %d, want clamped ceiling %d",
				effective.GateMaxRevisions,
				loop.LoopMaxGateRevisions,
			)
		}
		if effective.BudgetOnExceeded != dsl.BudgetExceededEscalate {
			t.Fatalf("BudgetOnExceeded = %q, want escalate", effective.BudgetOnExceeded)
		}
	})

	t.Run("Should clamp an override above a ceiling instead of honoring it", func(t *testing.T) {
		t.Parallel()

		resolved := compileDefinition(t, validDefinition())
		perRun := loop.LoopConfig{
			NoProgressWindow: new(loop.LoopMaxNoProgressWindow + 10),
			FanOutWidth:      new(loop.LoopMaxFanoutWidth + 10),
			GateMaxRevisions: new(loop.LoopMaxGateRevisions + 10),
		}

		effective, err := loop.ResolveEffectiveConfig(
			resolved,
			loop.LoopDefaults{},
			nil,
			perRun,
		)
		if err != nil {
			t.Fatalf("ResolveEffectiveConfig() error = %v", err)
		}
		if effective.NoProgressWindow != loop.LoopMaxNoProgressWindow {
			t.Fatalf(
				"NoProgressWindow = %d, want %d",
				effective.NoProgressWindow,
				loop.LoopMaxNoProgressWindow,
			)
		}
		if effective.FanOutWidth != loop.LoopMaxFanoutWidth {
			t.Fatalf("FanOutWidth = %d, want %d", effective.FanOutWidth, loop.LoopMaxFanoutWidth)
		}
		if effective.GateMaxRevisions != loop.LoopMaxGateRevisions {
			t.Fatalf("GateMaxRevisions = %d, want %d", effective.GateMaxRevisions, loop.LoopMaxGateRevisions)
		}
	})
}

func TestServiceStartShouldUseDefaultsResolver(t *testing.T) {
	t.Parallel()

	t.Run("Should seed effective config from workspace defaults resolver", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		halt := dsl.BudgetExceededHalt
		var resolvedWorkspace loop.WorkspaceID
		svc := newTestServiceWithOptions(
			t,
			store,
			validDefinition(),
			loop.WithDefaultsResolver(func(
				_ context.Context,
				ws loop.WorkspaceID,
			) (loop.LoopDefaults, error) {
				resolvedWorkspace = ws
				return loop.LoopDefaults{
					Delivery: loop.LoopConfig{
						IterationCap:     new(12),
						NoProgressWindow: new(5),
						BudgetTokens:     new(0),
						BudgetWallSec:    new(0),
						BudgetOnExceeded: &halt,
						FanOutWidth:      new(1),
					},
				}, nil
			}),
		)

		run, err := svc.Start(context.Background(), "ws-config", "valid-loop", loop.Inputs{
			Values: map[string]any{"tasks": "task-ref"},
		}, humanActor(t))
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if resolvedWorkspace != "ws-config" {
			t.Fatalf("defaults resolver workspace = %q, want ws-config", resolvedWorkspace)
		}
		if run.IterationCap != 12 {
			t.Fatalf("IterationCap = %d, want resolver default 12", run.IterationCap)
		}
	})
}

func TestServiceStartShouldEnforceConcurrencyAndAncestry(t *testing.T) {
	t.Parallel()

	t.Run("Should reject forbid concurrency when an active run exists", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		seedFakeRun(store, loop.StatusRunning)
		svc := newTestService(t, store, validDefinition())

		_, err := svc.Start(context.Background(), "ws-1", "valid-loop", loop.Inputs{
			Values: map[string]any{"tasks": "task-ref"},
		}, humanActor(t))
		if !errors.Is(err, loop.ErrConcurrencyConflict) {
			t.Fatalf("Start() error = %v, want ErrConcurrencyConflict", err)
		}
		var reason *loop.ReasonError
		if !errors.As(err, &reason) || reason.Code != loop.ReasonCodeActiveRunExists {
			t.Fatalf("Start() reason = %#v, want active run reason code", reason)
		}
	})

	t.Run("Should create a queued run when queue concurrency sees an active run", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		seedFakeRun(store, loop.StatusRunning)
		def := validDefinition()
		def.Concurrency = dsl.ConcurrencyQueue
		svc := newTestService(t, store, def)

		run, err := svc.Start(context.Background(), "ws-1", "valid-loop", loop.Inputs{
			Values: map[string]any{"tasks": "task-ref"},
		}, humanActor(t))
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if run.Status != loop.StatusQueued {
			t.Fatalf("run.Status = %q, want queued", run.Status)
		}
	})

	t.Run("Should reject a run loop target already present in the parent chain", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		parent := seedFakeRun(store, loop.StatusRunning)
		svc := newTestService(t, store, validDefinition())

		_, err := svc.Start(context.Background(), "ws-1", "valid-loop", loop.Inputs{
			Values:          map[string]any{"tasks": "task-ref"},
			ParentLoopRunID: parent.ID,
		}, humanActor(t))
		if !errors.Is(err, loop.ErrAncestryRejected) {
			t.Fatalf("Start() error = %v, want ErrAncestryRejected", err)
		}
		var reason *loop.ReasonError
		if !errors.As(err, &reason) || reason.Code != loop.ReasonCodeAncestryCycle {
			t.Fatalf("Start() reason = %#v, want ancestry cycle reason code", reason)
		}
	})
}

func TestServiceControlMethodsShouldPreserveStatusContracts(t *testing.T) {
	t.Parallel()

	t.Run("Should pause by setting intent without changing status", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		run := seedFakeRun(store, loop.StatusRunning)
		svc := newTestService(t, store, validDefinition())

		if err := svc.Pause(context.Background(), "ws-1", run.ID); err != nil {
			t.Fatalf("Pause() error = %v", err)
		}
		stored := store.mustRun(t, run.ID)
		if stored.Status != loop.StatusRunning || !stored.PauseRequested {
			t.Fatalf("stored run = %#v, want running with pause_requested", stored)
		}
		if err := svc.Pause(context.Background(), "ws-1", run.ID); err != nil {
			t.Fatalf("Pause() second call error = %v, want idempotent nil", err)
		}
	})

	t.Run("Should resume by clearing intent or transitioning paused to running", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		running := seedFakeRun(store, loop.StatusRunning)
		running.PauseRequested = true
		store.seed(running)
		paused := seedFakeRun(store, loop.StatusPaused)
		paused.ID = "run-paused-resume"
		paused.PauseRequested = true
		store.seed(paused)
		svc := newTestService(t, store, validDefinition())

		if err := svc.Resume(context.Background(), "ws-1", running.ID); err != nil {
			t.Fatalf("Resume(running) error = %v", err)
		}
		if got := store.mustRun(t, running.ID); got.PauseRequested {
			t.Fatalf("running PauseRequested = true, want false")
		}
		if err := svc.Resume(context.Background(), "ws-1", paused.ID); err != nil {
			t.Fatalf("Resume(paused) error = %v", err)
		}
		if got := store.mustRun(t, paused.ID); got.Status != loop.StatusRunning || got.PauseRequested {
			t.Fatalf("paused after resume = %#v, want running and not requested", got)
		}
	})

	t.Run("Should approve or reject only needs approval runs", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		approvable := seedFakeRun(store, loop.StatusNeedsApproval)
		approvable.ID = "run-approve"
		store.seed(approvable)
		rejectable := seedFakeRun(store, loop.StatusNeedsApproval)
		rejectable.ID = "run-reject"
		store.seed(rejectable)
		running := seedFakeRun(store, loop.StatusRunning)
		running.ID = "run-running-approve"
		store.seed(running)
		svc := newTestService(t, store, validDefinition())

		err := svc.Approve(
			context.Background(),
			"ws-1",
			approvable.ID,
			"review_gate",
			loop.GateDecisionApprove,
		)
		if err != nil {
			t.Fatalf("Approve(approve) error = %v", err)
		}
		if got := store.mustRun(t, approvable.ID); got.Status != loop.StatusRunning {
			t.Fatalf("approved status = %q, want running", got.Status)
		}
		err = svc.Approve(
			context.Background(),
			"ws-1",
			rejectable.ID,
			"review_gate",
			loop.GateDecisionReject,
		)
		if err != nil {
			t.Fatalf("Approve(reject) error = %v", err)
		}
		if got := store.mustRun(t, rejectable.ID); got.Status != loop.StatusBlocked {
			t.Fatalf("rejected status = %q, want blocked", got.Status)
		}
		err = svc.Approve(
			context.Background(),
			"ws-1",
			running.ID,
			"review_gate",
			loop.GateDecisionApprove,
		)
		if !errors.Is(err, loop.ErrInvalidTransition) {
			t.Fatalf("Approve(running) error = %v, want ErrInvalidTransition", err)
		}
	})
}

func TestServiceConfigMethodsShouldReadWriteRawOverrides(t *testing.T) {
	t.Parallel()

	t.Run("Should configure clamp and read loop config", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		svc := newTestService(t, store, validDefinition())

		if err := svc.Configure(context.Background(), "ws-1", "valid-loop", loop.LoopConfig{
			EnabledChecks: []byte(`{"human":true}`),
			FanOutWidth:   new(loop.LoopMaxFanoutWidth + 1),
		}); err != nil {
			t.Fatalf("Configure() error = %v", err)
		}
		cfg, err := svc.GetConfig(context.Background(), "ws-1", "valid-loop")
		if err != nil {
			t.Fatalf("GetConfig() error = %v", err)
		}
		if cfg.FanOutWidth == nil || *cfg.FanOutWidth != loop.LoopMaxFanoutWidth {
			t.Fatalf("FanOutWidth = %#v, want clamped %d", cfg.FanOutWidth, loop.LoopMaxFanoutWidth)
		}
		if string(cfg.EnabledChecks) != `{"human":true}` {
			t.Fatalf("EnabledChecks = %s, want persisted JSON", cfg.EnabledChecks)
		}
	})

	t.Run("Should reject invalid config JSON", func(t *testing.T) {
		t.Parallel()

		svc := newTestService(t, newFakeLoopStore(), validDefinition())
		err := svc.Configure(context.Background(), "ws-1", "valid-loop", loop.LoopConfig{
			EnabledChecks: []byte(`{`),
		})
		if !errors.Is(err, loop.ErrValidation) {
			t.Fatalf("Configure(invalid JSON) error = %v, want ErrValidation", err)
		}
	})
}

func TestServiceGetAndDefaultsShouldExposeRunState(t *testing.T) {
	t.Parallel()

	t.Run("Should return a workspace scoped run", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		run := seedFakeRun(store, loop.StatusRunning)
		svc := newTestService(t, store, validDefinition())

		got, err := svc.Get(context.Background(), "ws-1", run.ID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if got.ID != run.ID {
			t.Fatalf("Get().ID = %q, want %q", got.ID, run.ID)
		}
	})

	t.Run("Should use injected defaults in start effective config", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		svc := newTestServiceWithOptions(t, store, validDefinition(), loop.WithDefaults(loop.LoopDefaults{
			Delivery: loop.LoopConfig{BudgetTokens: new(4444)},
		}))

		run, err := svc.Start(context.Background(), "ws-1", "valid-loop", loop.Inputs{
			Values: map[string]any{"tasks": "task-ref"},
		}, humanActor(t))
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
		if run.BudgetTokens != 4444 {
			t.Fatalf("BudgetTokens = %d, want injected default 4444", run.BudgetTokens)
		}
	})
}

func TestResolveInputsShouldApplyDefaultsAndValidateTypes(t *testing.T) {
	t.Parallel()

	t.Run("Should apply defaults and clone nested values", func(t *testing.T) {
		t.Parallel()

		def := validDefinition()
		def.Inputs = map[string]dsl.Input{
			"name": {Type: dsl.InputTypeString, Required: true},
			"flag": {Type: dsl.InputTypeBoolean, Default: true},
		}
		resolved, err := loop.ResolveInputs(def, loop.Inputs{Values: map[string]any{"name": "loop"}})
		if err != nil {
			t.Fatalf("ResolveInputs() error = %v", err)
		}
		if resolved["flag"] != true {
			t.Fatalf("resolved flag = %#v, want true", resolved["flag"])
		}
	})

	t.Run("Should reject invalid declared input types", func(t *testing.T) {
		t.Parallel()

		def := validDefinition()
		def.Inputs = map[string]dsl.Input{"count": {Type: dsl.InputTypeNumber, Required: true}}
		_, err := loop.ResolveInputs(def, loop.Inputs{Values: map[string]any{"count": "not-a-number"}})
		if !errors.Is(err, loop.ErrValidation) {
			t.Fatalf("ResolveInputs() error = %v, want ErrValidation", err)
		}
	})
}

func TestServiceDryRunShouldReturnPlanPreviewWithoutState(t *testing.T) {
	t.Parallel()

	store := newFakeLoopStore()
	svc := newTestService(t, store, validDefinition())

	preview, err := svc.DryRun(context.Background(), "ws-1", "valid-loop", loop.Inputs{
		Values: map[string]any{"tasks": "task-ref"},
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	if preview.Generation != 1 {
		t.Fatalf("Generation = %d, want 1", preview.Generation)
	}
	if got, want := preview.ResolvedInputs["tasks"], "task-ref"; got != want {
		t.Fatalf("ResolvedInputs[tasks] = %#v, want %q", got, want)
	}
	if len(preview.Nodes) == 0 {
		t.Fatal("Nodes length = 0, want materialized gen-1 nodes")
	}
	if store.createCount() != 0 {
		t.Fatalf("CreateLoopRun calls = %d, want 0", store.createCount())
	}
}

func TestCostShouldBeDisplayOnly(t *testing.T) {
	t.Parallel()

	t.Run("Should derive cost as tokens multiplied by price", func(t *testing.T) {
		t.Parallel()

		cost := loop.DeriveDisplayCost(1500, 0.000002)
		if math.Abs(cost.USD-0.003) > 0.0000001 {
			t.Fatalf("USD = %.9f, want 0.003", cost.USD)
		}
	})

	t.Run("Should not expose a budget USD enforcement field", func(t *testing.T) {
		t.Parallel()

		if _, ok := reflect.TypeFor[loop.EffectiveConfig]().FieldByName("BudgetUSD"); ok {
			t.Fatal("EffectiveConfig exposes BudgetUSD, want token/wall budget enforcement only")
		}
	})
}

func TestServiceConstructorAndReasonErrorsShouldBeStable(t *testing.T) {
	t.Parallel()

	t.Run("Should reject missing service dependencies", func(t *testing.T) {
		t.Parallel()

		resolver := loop.DefinitionResolverFunc(func(
			context.Context,
			loop.WorkspaceID,
			string,
		) (*loop.ResolvedDefinition, error) {
			return nil, nil
		})
		if _, err := loop.NewService(nil, resolver); !errors.Is(err, loop.ErrValidation) {
			t.Fatalf("NewService(nil store) error = %v, want ErrValidation", err)
		}
		if _, err := loop.NewService(newFakeLoopStore(), nil); !errors.Is(err, loop.ErrValidation) {
			t.Fatalf("NewService(nil resolver) error = %v, want ErrValidation", err)
		}
	})

	t.Run("Should render reason errors with deterministic metadata", func(t *testing.T) {
		t.Parallel()

		err := &loop.ReasonError{
			Code: loop.ReasonCodeActiveRunExists,
			Err:  loop.ErrConcurrencyConflict,
			Meta: map[string]string{
				"status":        "running",
				"active_run_id": "run-1",
			},
		}
		want := "active_loop_run_exists: loop: concurrency conflict " +
			"(active_run_id=run-1, status=running)"
		if got := err.Error(); got != want {
			t.Fatalf("ReasonError.Error() = %q, want %q", got, want)
		}
		if !errors.Is(err, loop.ErrConcurrencyConflict) {
			t.Fatalf("ReasonError does not unwrap ErrConcurrencyConflict")
		}
	})
}

func TestServiceStopShouldFailRunWithOperatorCause(t *testing.T) {
	t.Parallel()

	t.Run("Should transition a live run to failed with operator stop cause", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		run := seedFakeRun(store, loop.StatusRunning)
		svc := newTestService(t, store, validDefinition())

		if err := svc.Stop(context.Background(), "ws-1", run.ID, loop.StopReasonOperator); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
		stored := store.mustRun(t, run.ID)
		if stored.Status != loop.StatusFailed {
			t.Fatalf("stored status = %q, want failed", stored.Status)
		}
		transition := store.lastTransition(t)
		if transition.cause != loop.TransitionCauseOperatorStop {
			t.Fatalf("transition cause = %q, want operator_stop", transition.cause)
		}
	})

	t.Run("Should reject stop for an already terminal run", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		run := seedFakeRun(store, loop.StatusDone)
		svc := newTestService(t, store, validDefinition())

		err := svc.Stop(context.Background(), "ws-1", run.ID, loop.StopReasonOperator)
		if !errors.Is(err, loop.ErrInvalidTransition) {
			t.Fatalf("Stop() error = %v, want ErrInvalidTransition", err)
		}
	})

	t.Run("Should reject an unknown stop reason", func(t *testing.T) {
		t.Parallel()

		store := newFakeLoopStore()
		run := seedFakeRun(store, loop.StatusRunning)
		svc := newTestService(t, store, validDefinition())

		err := svc.Stop(context.Background(), "ws-1", run.ID, loop.StopReason("unknown"))
		if !errors.Is(err, loop.ErrValidation) {
			t.Fatalf("Stop(unknown reason) error = %v, want ErrValidation", err)
		}
		if got := store.mustRun(t, run.ID); got.Status != loop.StatusRunning {
			t.Fatalf("stored status = %q, want original running", got.Status)
		}
	})
}

func newTestService(t *testing.T, store *fakeLoopStore, def dsl.Definition) loop.Service {
	t.Helper()

	return newTestServiceWithOptions(t, store, def)
}

func newTestServiceWithOptions(
	t *testing.T,
	store *fakeLoopStore,
	def dsl.Definition,
	opts ...loop.Option,
) loop.Service {
	t.Helper()

	resolved := compileDefinition(t, def)
	options := []loop.Option{
		loop.WithClock(func() time.Time {
			return time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
		}),
		loop.WithRunIDFactory(func() loop.RunID {
			return loop.RunID("looprun-test")
		}),
	}
	options = append(options, opts...)
	svc, err := loop.NewService(
		store,
		loop.DefinitionResolverFunc(func(
			context.Context,
			loop.WorkspaceID,
			string,
		) (*loop.ResolvedDefinition, error) {
			return resolved, nil
		}),
		options...,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return svc
}

func compileDefinition(t *testing.T, def dsl.Definition) *loop.ResolvedDefinition {
	t.Helper()

	resolved, err := loop.NewCompiler(
		loop.WithCompilerToolSchemaSource(fakeToolSchemas{}),
	).Compile(def)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	return resolved
}

func seedFakeRun(store *fakeLoopStore, status loop.Status) loop.Run {
	run := loop.Run{
		ID:                loop.RunID("run-" + string(status)),
		WorkspaceID:       "ws-1",
		LoopName:          "valid-loop",
		Status:            status,
		ReattemptStrategy: loop.ReattemptFailedOnly,
		CreatedAt:         time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC),
		LastProgressAt:    time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC),
		BudgetOnExceeded:  dsl.BudgetExceededHalt,
		Inputs:            map[string]any{"tasks": "task-ref"},
	}
	store.seed(run)
	return run
}

type fakeTransition struct {
	runID loop.RunID
	from  loop.Status
	to    loop.Status
	cause loop.TransitionCause
}

type fakeLoopStore struct {
	mu          sync.Mutex
	runs        map[loop.RunID]loop.Run
	configs     map[string]loop.LoopConfig
	transitions []fakeTransition
	creates     int
}

func newFakeLoopStore() *fakeLoopStore {
	return &fakeLoopStore{
		runs:    map[loop.RunID]loop.Run{},
		configs: map[string]loop.LoopConfig{},
	}
}

func (s *fakeLoopStore) seed(run loop.Run) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.ID] = run
}

func (s *fakeLoopStore) mustRun(t *testing.T, id loop.RunID) loop.Run {
	t.Helper()
	run, err := s.GetLoopRunByID(context.Background(), id)
	if err != nil {
		t.Fatalf("GetLoopRunByID(%q) error = %v", id, err)
	}
	return run
}

func (s *fakeLoopStore) lastTransition(t *testing.T) fakeTransition {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.transitions) == 0 {
		t.Fatal("transition count = 0, want at least one")
	}
	return s.transitions[len(s.transitions)-1]
}

func (s *fakeLoopStore) createCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.creates
}

func (s *fakeLoopStore) CreateLoopRunForStart(
	_ context.Context,
	run loop.Run,
	policy dsl.ConcurrencyPolicy,
) (loop.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if policy == "" {
		policy = dsl.ConcurrencyForbid
	}
	active := s.activeLoopRunLocked(run.WorkspaceID, run.LoopName)
	switch policy {
	case dsl.ConcurrencyForbid:
		if active != nil {
			return loop.Run{}, &loop.ReasonError{
				Code: loop.ReasonCodeActiveRunExists,
				Err:  loop.ErrConcurrencyConflict,
				Meta: map[string]string{
					"active_run_id": string(active.ID),
					"loop_name":     active.LoopName,
					"status":        string(active.Status),
				},
			}
		}
		run.Status = loop.StatusRunning
	case dsl.ConcurrencyQueue:
		if active != nil {
			run.Status = loop.StatusQueued
		} else {
			run.Status = loop.StatusRunning
		}
	case dsl.ConcurrencyAllow:
		run.Status = loop.StatusRunning
	default:
		return loop.Run{}, loop.ErrValidation
	}
	s.creates++
	s.runs[run.ID] = run
	return run, nil
}

func (s *fakeLoopStore) GetLoopRun(
	_ context.Context,
	ws loop.WorkspaceID,
	runID loop.RunID,
) (loop.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.WorkspaceID != ws {
		return loop.Run{}, loop.ErrRunNotFound
	}
	return run, nil
}

func (s *fakeLoopStore) GetLoopRunByID(_ context.Context, runID loop.RunID) (loop.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok {
		return loop.Run{}, loop.ErrRunNotFound
	}
	return run, nil
}

func (s *fakeLoopStore) FindActiveLoopRun(
	_ context.Context,
	ws loop.WorkspaceID,
	loopName string,
) (*loop.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeLoopRunLocked(ws, loopName), nil
}

func (s *fakeLoopStore) activeLoopRunLocked(ws loop.WorkspaceID, loopName string) *loop.Run {
	var oldest *loop.Run
	for _, run := range s.runs {
		if run.WorkspaceID != ws || run.LoopName != loopName || !run.Status.Live() {
			continue
		}
		candidate := run
		if oldest == nil || candidate.CreatedAt.Before(oldest.CreatedAt) {
			oldest = &candidate
		}
	}
	return oldest
}

func (s *fakeLoopStore) CompareAndSwapLoopRunStatus(
	_ context.Context,
	runID loop.RunID,
	from loop.Status,
	to loop.Status,
	cause loop.TransitionCause,
	_ time.Time,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok {
		return loop.ErrRunNotFound
	}
	if run.Status != from {
		return loop.ErrTransitionConflict
	}
	run.Status = to
	if to == loop.StatusRunning || to == loop.StatusPaused || to.Terminal() {
		run.PauseRequested = false
	}
	s.runs[runID] = run
	s.transitions = append(
		s.transitions,
		fakeTransition{runID: runID, from: from, to: to, cause: cause},
	)
	return nil
}

func (s *fakeLoopStore) SetLoopRunPauseRequested(
	_ context.Context,
	ws loop.WorkspaceID,
	runID loop.RunID,
	requested bool,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[runID]
	if !ok || run.WorkspaceID != ws {
		return loop.ErrRunNotFound
	}
	run.PauseRequested = requested
	s.runs[runID] = run
	return nil
}

func (s *fakeLoopStore) UpsertLoopConfig(
	_ context.Context,
	ws loop.WorkspaceID,
	loopName string,
	cfg loop.LoopConfig,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configs[string(ws)+"/"+loopName] = cfg
	return nil
}

func (s *fakeLoopStore) GetLoopConfig(
	_ context.Context,
	ws loop.WorkspaceID,
	loopName string,
) (*loop.LoopConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg, ok := s.configs[string(ws)+"/"+loopName]
	if !ok {
		return nil, loop.ErrConfigNotFound
	}
	return &cfg, nil
}

func humanActor(t *testing.T) task.ActorContext {
	t.Helper()
	actor, err := task.DeriveHumanActorContext("operator", task.OriginKindCLI, "cli")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}
	return actor
}
