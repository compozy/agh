package daemon

import (
	"context"
	"strconv"
	"testing"
	"time"

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
	t.Parallel()

	ctx := testutil.Context(t)
	now := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	db := openDaemonTestGlobalDB(t)
	workspaceID := "ws-backstop"
	if err := db.InsertWorkspace(ctx, workspacepkg.Workspace{
		ID:        workspaceID,
		RootDir:   t.TempDir(),
		Name:      "Backstop",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}
	manager, err := taskpkg.NewManager(
		taskpkg.WithStore(db),
		taskpkg.WithCoordinatorRunner(schedulerBackstopCoordinatorRunner{}),
		taskpkg.WithGenerationStateFinalizer(schedulerBackstopGenerationFinalizer{}),
		taskpkg.WithManagerNow(func() time.Time { return now }),
	)
	if err != nil {
		t.Fatalf("task.NewManager() error = %v", err)
	}
	for index := range defaultLoopCoordinatorBackstopLimit + 1 {
		seedCoordinatorBackstopRun(t, db, now, "global", index, taskpkg.ScopeGlobal, "")
		seedCoordinatorBackstopRun(t, db, now, "workspace", index, taskpkg.ScopeWorkspace, workspaceID)
	}
	actor, err := taskpkg.DeriveDaemonActorContext("scheduler", "daemon.scheduler")
	if err != nil {
		t.Fatalf("DeriveDaemonActorContext() error = %v", err)
	}

	started, err := (schedulerTaskSource{manager: manager, store: db}).RunLoopCoordinatorBackstop(ctx, now, actor)
	if err != nil {
		t.Fatalf("RunLoopCoordinatorBackstop() error = %v", err)
	}
	if got, want := started, defaultLoopCoordinatorBackstopLimit*2; got != want {
		t.Fatalf("started = %d, want %d", got, want)
	}
	queued, err := db.ListTaskRunsByStatus(ctx, []taskpkg.RunStatus{taskpkg.TaskRunStatusQueued})
	if err != nil {
		t.Fatalf("ListTaskRunsByStatus(queued) error = %v", err)
	}
	if got, want := len(queued), 2; got != want {
		t.Fatalf("queued runs = %d, want one remaining per scope", got)
	}
}

func seedCoordinatorBackstopRun(
	t *testing.T,
	db *globaldb.GlobalDB,
	now time.Time,
	prefix string,
	index int,
	scope taskpkg.Scope,
	workspaceID string,
) {
	t.Helper()

	taskRecord := daemonTaskRecordForTest("task-backstop-"+prefix+"-"+strconv.Itoa(index), now)
	taskRecord.Scope = scope
	taskRecord.WorkspaceID = workspaceID
	if err := db.CreateTask(testutil.Context(t), taskRecord); err != nil {
		t.Fatalf("CreateTask(%s/%d) error = %v", prefix, index, err)
	}
	loopRun := looppkg.Run{
		ID:                looppkg.RunID("looprun-backstop-" + prefix + "-" + strconv.Itoa(index)),
		WorkspaceID:       looppkg.WorkspaceID("ws-backstop-" + prefix),
		LoopName:          "backstop-" + prefix + "-" + strconv.Itoa(index),
		Status:            looppkg.StatusRunning,
		ReattemptStrategy: looppkg.ReattemptFailedOnly,
		IterationCap:      50,
		BudgetOnExceeded:  loopdsl.BudgetExceededHalt,
		CreatedAt:         now,
		LastProgressAt:    now,
	}
	if workspaceID != "" {
		loopRun.WorkspaceID = looppkg.WorkspaceID(workspaceID)
	}
	applyLoopRunPinningForTest(&loopRun, now)
	createdLoopRun, err := db.CreateLoopRunForStart(testutil.Context(t), loopRun, loopdsl.ConcurrencyAllow)
	if err != nil {
		t.Fatalf("CreateLoopRunForStart(%s/%d) error = %v", prefix, index, err)
	}
	if _, _, _, err := db.ReserveQueuedRun(testutil.Context(t), taskpkg.QueueRunReservation{
		TaskID:         taskRecord.ID,
		RunID:          "run-backstop-" + prefix + "-" + strconv.Itoa(index),
		RunKind:        taskpkg.RunKindCoordinator,
		LoopRunID:      string(createdLoopRun.ID),
		IdempotencyKey: "backstop." + prefix + "." + strconv.Itoa(index),
		Origin:         taskpkg.Origin{Kind: taskpkg.OriginKindDaemon, Ref: "backstop-test"},
		QueuedAt:       now.Add(time.Duration(index) * time.Millisecond),
	}); err != nil {
		t.Fatalf("ReserveQueuedRun(%s/%d) error = %v", prefix, index, err)
	}
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
