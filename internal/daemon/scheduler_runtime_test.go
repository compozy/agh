package daemon

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	looppkg "github.com/compozy/agh/internal/loop"
	loopdsl "github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/store/globaldb"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

func TestSchedulerTaskSourcePendingRunsShouldHideLoopActionRuns(t *testing.T) {
	t.Parallel()

	t.Run("Should expose only generic queued worker runs", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 5, 19, 0, 0, 0, time.UTC)
		db := openDaemonTestGlobalDB(t)
		if err := db.InsertWorkspace(ctx, workspacepkg.Workspace{
			ID:        "ws-scheduler",
			RootDir:   t.TempDir(),
			Name:      "Scheduler",
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			t.Fatalf("InsertWorkspace(ws-scheduler) error = %v", err)
		}
		normalTask := daemonTaskRecordForTest("task-normal-worker", now)
		if err := db.CreateTask(ctx, normalTask); err != nil {
			t.Fatalf("CreateTask(normal) error = %v", err)
		}
		loopTask := daemonTaskRecordForTest("task-loop-worker", now)
		if err := db.CreateTask(ctx, loopTask); err != nil {
			t.Fatalf("CreateTask(loop) error = %v", err)
		}
		seedRun := looppkg.Run{
			ID:                "looprun-scheduler-hidden",
			WorkspaceID:       "ws-scheduler",
			LoopName:          "software-delivery",
			Status:            looppkg.StatusRunning,
			ReattemptStrategy: looppkg.ReattemptFailedOnly,
			IterationCap:      50,
			BudgetOnExceeded:  loopdsl.BudgetExceededHalt,
			CreatedAt:         now,
			LastProgressAt:    now,
		}
		applyLoopRunPinningForTest(&seedRun, now)
		if _, err := db.CreateLoopRunForStart(ctx, seedRun, loopdsl.ConcurrencyAllow); err != nil {
			t.Fatalf("CreateLoopRunForStart() error = %v", err)
		}
		if _, _, _, err := db.ReserveQueuedRun(ctx, taskpkg.QueueRunReservation{
			TaskID:         normalTask.ID,
			RunID:          "run-normal-worker",
			RunKind:        taskpkg.RunKindWorker,
			IdempotencyKey: "scheduler.normal",
			Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "scheduler-test"},
			QueuedAt:       now,
		}); err != nil {
			t.Fatalf("ReserveQueuedRun(normal) error = %v", err)
		}
		if _, _, _, err := db.ReserveQueuedRun(ctx, taskpkg.QueueRunReservation{
			TaskID:         loopTask.ID,
			RunID:          "run-loop-action",
			RunKind:        taskpkg.RunKindWorker,
			LoopRunID:      "looprun-scheduler-hidden",
			IdempotencyKey: "scheduler.loop-action",
			Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "loop"},
			QueuedAt:       now,
		}); err != nil {
			t.Fatalf("ReserveQueuedRun(loop) error = %v", err)
		}

		pending, err := (schedulerTaskSource{store: db}).PendingRuns(ctx)
		if err != nil {
			t.Fatalf("PendingRuns() error = %v", err)
		}
		if got, want := len(pending), 1; got != want {
			t.Fatalf("len(PendingRuns()) = %d, want %d", got, want)
		}
		if got, want := pending[0].Run.ID, "run-normal-worker"; got != want {
			t.Fatalf("PendingRuns()[0].Run.ID = %q, want %q", got, want)
		}
	})
}

func TestSchedulerTaskSourceLoopCoordinatorBackstopShouldLimitPerScope(t *testing.T) {
	// This test starts 32 coordinator runs against SQLite; keep it serial so the
	// package's race-heavy parallel tests do not exhaust the default test context.
	t.Run("Should limit coordinator starts per workspace scope", func(t *testing.T) {
		ctx := testutil.Context(t)
		now := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
		db := openDaemonTestGlobalDB(t)
		workspaceIDs := []string{"ws-backstop-alpha", "ws-backstop-beta"}
		for _, workspaceID := range workspaceIDs {
			if err := db.InsertWorkspace(ctx, workspacepkg.Workspace{
				ID:        workspaceID,
				RootDir:   t.TempDir(),
				Name:      workspaceID,
				CreatedAt: now,
				UpdatedAt: now,
			}); err != nil {
				t.Fatalf("InsertWorkspace(%s) error = %v", workspaceID, err)
			}
		}
		manager, err := taskpkg.NewManager(
			taskpkg.WithStore(db),
			taskpkg.WithCoordinatorRunner(schedulerBackstopCoordinatorRunner{}),
			taskpkg.WithGenerationStateFinalizer(schedulerBackstopGenerationFinalizer{}),
			taskpkg.WithCoordinatorTerminalStatusValidator(func(status string) bool {
				return looppkg.Status(strings.TrimSpace(status)).Valid()
			}),
			taskpkg.WithCoordinatorTerminalHookStatusValidator(func(status string) bool {
				return looppkg.Status(strings.TrimSpace(status)).Terminal()
			}),
			taskpkg.WithManagerNow(func() time.Time { return now }),
		)
		if err != nil {
			t.Fatalf("task.NewManager() error = %v", err)
		}
		for index := range defaultLoopCoordinatorBackstopLimit + 1 {
			seedCoordinatorBackstopRun(t, db, now, "alpha", index, workspaceIDs[0])
			seedCoordinatorBackstopRun(t, db, now, "beta", index, workspaceIDs[1])
		}
		actor, err := taskpkg.DeriveDaemonActorContext("scheduler", "daemon.scheduler")
		if err != nil {
			t.Fatalf("DeriveDaemonActorContext() error = %v", err)
		}

		started, err := (schedulerTaskSource{manager: manager, store: db}).RunLoopCoordinatorBackstop(ctx, now, actor)
		if err != nil {
			t.Fatalf("RunLoopCoordinatorBackstop() error = %v", err)
		}
		if got, want := started, defaultLoopCoordinatorBackstopLimit*len(workspaceIDs); got != want {
			t.Fatalf("started = %d, want %d", got, want)
		}
		queued, err := db.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusQueued})
		if err != nil {
			t.Fatalf("ListTaskRunsByStatus(queued) error = %v", err)
		}
		if got, want := len(queued), len(workspaceIDs); got != want {
			t.Fatalf("queued runs = %d, want one remaining per scope", got)
		}
	})
}

func TestSchedulerTaskSourceLoopCoordinatorBackstopShouldRecoverWatchEventsGap(t *testing.T) {
	t.Parallel()

	t.Run(
		"Should enqueue and start a coordinator wake when only the durable watch-events ledger has advanced",
		func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t)
			now := time.Date(2026, 7, 8, 19, 30, 0, 0, time.UTC)
			db := openDaemonTestGlobalDB(t)
			workspaceID := "ws-backstop-watch-events"
			if err := db.InsertWorkspace(ctx, workspacepkg.Workspace{
				ID:        workspaceID,
				RootDir:   t.TempDir(),
				Name:      "Watch Events Backstop",
				CreatedAt: now,
				UpdatedAt: now,
			}); err != nil {
				t.Fatalf("InsertWorkspace() error = %v", err)
			}
			created, targetTask := parkSchedulerWatchEventsLoopForTest(ctx, t, db, now, workspaceID)
			appendSchedulerWatchEventForTest(ctx, t, db, targetTask.ID, now.Add(time.Second))
			manager, err := taskpkg.NewManager(
				taskpkg.WithStore(db),
				taskpkg.WithCoordinatorRunner(schedulerBackstopCoordinatorRunner{}),
				taskpkg.WithGenerationStateFinalizer(schedulerBackstopGenerationFinalizer{}),
				taskpkg.WithCoordinatorTerminalStatusValidator(func(status string) bool {
					return looppkg.Status(strings.TrimSpace(status)).Valid()
				}),
				taskpkg.WithCoordinatorTerminalHookStatusValidator(func(status string) bool {
					return looppkg.Status(strings.TrimSpace(status)).Terminal()
				}),
				taskpkg.WithManagerNow(func() time.Time { return now.Add(2 * time.Second) }),
			)
			if err != nil {
				t.Fatalf("task.NewManager() error = %v", err)
			}
			actor, err := taskpkg.DeriveDaemonActorContext("scheduler", "daemon.scheduler")
			if err != nil {
				t.Fatalf("DeriveDaemonActorContext() error = %v", err)
			}

			started, err := (schedulerTaskSource{
				manager:            manager,
				store:              db,
				watchEventsGapScan: newLoopWatchEventsGapScanState(),
			}).RunLoopCoordinatorBackstop(
				ctx,
				now.Add(2*time.Second),
				actor,
			)
			if err != nil {
				t.Fatalf("RunLoopCoordinatorBackstop() error = %v", err)
			}
			if got, want := started, 1; got != want {
				t.Fatalf("started = %d, want %d", got, want)
			}
			startedRuns, err := db.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusCompleted})
			if err != nil {
				t.Fatalf("ListTaskRunsByStatus(completed) error = %v", err)
			}
			if !schedulerCompletedRunForLoop(startedRuns, string(created.ID)) {
				t.Fatalf("completed coordinator runs = %#v, want loop_run_id %q", startedRuns, created.ID)
			}
		},
	)
}

func seedCoordinatorBackstopRun(
	t *testing.T,
	db *globaldb.GlobalDB,
	now time.Time,
	prefix string,
	index int,
	workspaceID string,
) {
	t.Helper()

	at := now.Add(time.Duration(index) * time.Millisecond)
	loopRun := looppkg.Run{
		ID:                looppkg.RunID("looprun-backstop-" + prefix + "-" + strconv.Itoa(index)),
		WorkspaceID:       looppkg.WorkspaceID(workspaceID),
		LoopName:          "backstop-" + prefix + "-" + strconv.Itoa(index),
		Status:            looppkg.StatusRunning,
		ReattemptStrategy: looppkg.ReattemptFailedOnly,
		IterationCap:      50,
		BudgetOnExceeded:  loopdsl.BudgetExceededHalt,
		CreatedAt:         at,
		LastProgressAt:    at,
	}
	applyLoopRunPinningForTest(&loopRun, now)
	if _, err := db.CreateLoopRunForStart(testutil.Context(t), loopRun, loopdsl.ConcurrencyAllow); err != nil {
		t.Fatalf("CreateLoopRunForStart(%s/%d) error = %v", prefix, index, err)
	}
}

func parkSchedulerWatchEventsLoopForTest(
	ctx context.Context,
	t *testing.T,
	db *globaldb.GlobalDB,
	now time.Time,
	workspaceID string,
) (looppkg.Run, taskpkg.Task) {
	t.Helper()
	targetTask := daemonTaskRecordForTest("task-watch-events-backstop-target", now)
	targetTask.Scope = taskpkg.ScopeWorkspace
	targetTask.WorkspaceID = workspaceID
	if err := db.CreateTask(ctx, targetTask); err != nil {
		t.Fatalf("CreateTask(target) error = %v", err)
	}
	loopRun := looppkg.Run{
		ID:                "looprun-watch-events-backstop",
		WorkspaceID:       looppkg.WorkspaceID(workspaceID),
		LoopName:          "watch-events-backstop",
		Status:            looppkg.StatusRunning,
		ReattemptStrategy: looppkg.ReattemptFailedOnly,
		IterationCap:      50,
		BudgetOnExceeded:  loopdsl.BudgetExceededHalt,
		CreatedAt:         now,
		LastProgressAt:    now,
		Inputs:            map[string]any{"target_task_id": targetTask.ID},
	}
	applyLoopRunPinningForTest(&loopRun, now)
	created, err := db.CreateLoopRunForStart(ctx, loopRun, loopdsl.ConcurrencyAllow)
	if err != nil {
		t.Fatalf("CreateLoopRunForStart() error = %v", err)
	}
	runner := newSchedulerWatchEventsCoordinatorForTest(t, db, targetTask.ID)
	actor := schedulerCoordinatorActorContextForTest(t)
	claim, err := db.ClaimNextRun(ctx, taskpkg.ClaimCriteria{
		Scope:            taskpkg.ScopeWorkspace,
		WorkspaceID:      workspaceID,
		RunKind:          taskpkg.RunKindCoordinator,
		ClaimerSessionID: "daemon-loop-coordinator",
		ClaimedBy:        &taskpkg.ActorIdentity{Kind: taskpkg.ActorKindDaemon, Ref: "loop-coordinator"},
		LeaseDuration:    time.Minute,
		Now:              now,
	})
	if err != nil {
		t.Fatalf("ClaimNextRun(initial coordinator) error = %v", err)
	}
	plan, err := runner.Run(ctx, taskpkg.RunID(claim.Run.ID))
	if err != nil {
		t.Fatalf("Run(initial coordinator) error = %v", err)
	}
	if _, err := db.CompleteCoordinatorAndEnqueueNext(ctx, taskpkg.CoordinatorCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Actor:      actor,
		Plan:       plan,
		Now:        now.Add(time.Second),
	}, looppkg.NewStoreFinalizer()); err != nil {
		t.Fatalf("CompleteCoordinatorAndEnqueueNext(initial) error = %v", err)
	}
	storedLoop, err := db.GetLoopRunByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetLoopRunByID(initial) error = %v", err)
	}
	if got, want := storedLoop.Status, looppkg.StatusWatching; got != want {
		t.Fatalf("initial terminal = %q, want %q", got, want)
	}
	return created, targetTask
}

func newSchedulerWatchEventsCoordinatorForTest(
	t *testing.T,
	db *globaldb.GlobalDB,
	targetTaskID string,
) *looppkg.CoordinatorRunner {
	t.Helper()
	resolved, err := looppkg.NewCompiler().Compile(loopdsl.Definition{
		APIVersion: loopdsl.APIVersion,
		Kind:       loopdsl.KindLoop,
		Inputs: map[string]loopdsl.Input{
			"target_task_id": {Type: loopdsl.InputTypeString},
		},
		Graph: loopdsl.Graph{
			Nodes: []loopdsl.Node{{
				ID:    "watch_tasks",
				Class: loopdsl.NodeClassSource,
				Kind:  string(loopdsl.SourceWatchEvents),
				Events: []loopdsl.EventSubscription{{
					Kind:   string(hookspkg.HookTaskBlocked),
					Filter: `event.task_id == inputs.target_task_id`,
				}},
			}, {
				ID:    "summarize",
				Class: loopdsl.NodeClassAction,
				Kind:  string(loopdsl.ActionTransform),
				Params: loopdsl.NodeParams{
					"map": map[string]any{"ok": map[string]any{"value": true}},
				},
			}},
			Edges: []loopdsl.Edge{{From: "watch_tasks", To: "summarize"}},
		},
	})
	if err != nil {
		t.Fatalf("Compile(watch-events backstop loop) error = %v", err)
	}
	runner, err := looppkg.NewCoordinatorRunner(
		db,
		db,
		db,
		looppkg.DefinitionResolverFunc(
			func(context.Context, looppkg.WorkspaceID, string) (*looppkg.ResolvedDefinition, error) {
				return resolved, nil
			},
		),
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		looppkg.WithCoordinatorWatchEventsLedger(db),
	)
	if err != nil {
		t.Fatalf("NewCoordinatorRunner(%s) error = %v", targetTaskID, err)
	}
	return runner
}

func appendSchedulerWatchEventForTest(
	ctx context.Context,
	t *testing.T,
	db *globaldb.GlobalDB,
	taskID string,
	at time.Time,
) {
	t.Helper()
	actor := schedulerCoordinatorActorContextForTest(t)
	manager, err := taskpkg.NewManager(
		taskpkg.WithStore(db),
		taskpkg.WithManagerNow(func() time.Time { return at }),
	)
	if err != nil {
		t.Fatalf("task.NewManager(block event) error = %v", err)
	}
	if _, err := manager.BlockTask(ctx, taskpkg.BlockRequest{
		TaskID: taskID,
		Kind:   taskpkg.BlockKindTransient,
		Reason: "scheduler backstop watch-events test",
	}, actor); err != nil {
		t.Fatalf("BlockTask() error = %v", err)
	}
}

func schedulerCoordinatorActorContextForTest(t *testing.T) taskpkg.ActorContext {
	t.Helper()
	actor, err := taskpkg.DeriveDaemonActorContext("loop", "daemon.loop")
	if err != nil {
		t.Fatalf("DeriveDaemonActorContext(loop) error = %v", err)
	}
	return actor
}

func schedulerCompletedRunForLoop(runs []taskpkg.Run, loopRunID string) bool {
	for _, run := range runs {
		if strings.TrimSpace(run.LoopRunID) == loopRunID && run.RunKind.Normalize() == taskpkg.RunKindCoordinator {
			return true
		}
	}
	return false
}

type schedulerBackstopCoordinatorRunner struct{}

func (schedulerBackstopCoordinatorRunner) Run(
	context.Context,
	taskpkg.RunID,
) (taskpkg.CoordinatorCompletionPlan, error) {
	return taskpkg.CoordinatorCompletionPlan{
		Snapshot: taskpkg.GenerationSnapshot{Generation: 1},
		Terminal: &taskpkg.CoordinatorTerminal{
			Status: "no-op",
			Cause:  "backstop-test",
		},
	}, nil
}

type schedulerBackstopGenerationFinalizer struct{}

func (schedulerBackstopGenerationFinalizer) WriteGenerationSnapshot(
	context.Context,
	taskpkg.Tx,
	taskpkg.GenerationSnapshot,
) error {
	return nil
}
