package daemon

import (
	"testing"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	loopdsl "github.com/compozy/agh/internal/loop/dsl"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
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
		if _, err := db.CreateLoopRunForStart(ctx, looppkg.Run{
			ID:                "looprun-scheduler-hidden",
			WorkspaceID:       "ws-scheduler",
			LoopName:          "software-delivery",
			Status:            looppkg.StatusRunning,
			ReattemptStrategy: looppkg.ReattemptFailedOnly,
			IterationCap:      50,
			BudgetOnExceeded:  loopdsl.BudgetExceededHalt,
			CreatedAt:         now,
			LastProgressAt:    now,
		}, loopdsl.ConcurrencyAllow); err != nil {
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
