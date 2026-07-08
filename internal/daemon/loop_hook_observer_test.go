package daemon

import (
	"context"
	"errors"
	"testing"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	taskpkg "github.com/compozy/agh/internal/task"
)

func TestLoopNativeHookObserverShouldProtectDurableNodeTerminalWake(t *testing.T) {
	t.Parallel()

	t.Run("Should enqueue node terminal wake before public hook dispatch can exhaust the context", func(t *testing.T) {
		t.Parallel()

		fixedNow := time.Date(2026, 7, 4, 12, 30, 0, 0, time.UTC)
		loopStore := &contextAwareLoopHookStore{}
		backstop := &recordingLoopBackstopRunner{}
		observer, err := newLoopNativeHookObserver(
			loopStore,
			blockingLoopNodeTerminalDispatcher{},
			backstop,
			func() time.Time { return fixedNow },
		)
		if err != nil {
			t.Fatalf("newLoopNativeHookObserver() error = %v", err)
		}

		ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
		defer cancel()
		workerKind := taskpkg.RunKindWorker.String()
		err = observer.OnTaskRunTerminal(ctx, hookspkg.TaskRunLeasePayload{
			PayloadBase: hookspkg.PayloadBase{Event: hookspkg.HookTaskRunCompleted, Timestamp: fixedNow},
			TaskRunContext: hookspkg.TaskRunContext{
				TaskID:      "task-node",
				RunID:       "run-node",
				RunKind:     &workerKind,
				LoopRunID:   "loop-run-1",
				WorkspaceID: "ws-1",
				RunStatus:   taskpkg.TaskRunStatusCompleted.String(),
				TaskStatus:  string(taskpkg.TaskStatusCompleted),
			},
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("OnTaskRunTerminal() error = %v, want context deadline exceeded", err)
		}

		if got, want := len(loopStore.progress), 1; got != want {
			t.Fatalf("progress calls = %d, want %d", got, want)
		}
		if got, want := len(loopStore.wakes), 1; got != want {
			t.Fatalf("wake calls = %d, want %d", got, want)
		}
		if got, want := loopStore.wakes[0].idempotencyKey, "loop.coordinator.node_terminal.loop-run-1.run-node"; got != want {
			t.Fatalf("wake key = %q, want %q", got, want)
		}
		if got, want := len(backstop.calls), 1; got != want {
			t.Fatalf("backstop calls = %d, want %d", got, want)
		}
	})
}

type contextAwareLoopHookStore struct {
	recordingLoopHookStore
}

type blockingLoopNodeTerminalDispatcher struct{}

func (blockingLoopNodeTerminalDispatcher) DispatchLoopNodeTerminal(
	ctx context.Context,
	payload hookspkg.LoopNodeTerminalPayload,
) (hookspkg.LoopNodeTerminalPayload, error) {
	<-ctx.Done()
	return payload, ctx.Err()
}

func (r *contextAwareLoopHookStore) AdvanceLoopRunProgress(
	ctx context.Context,
	loopRunID string,
	at time.Time,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return r.recordingLoopHookStore.AdvanceLoopRunProgress(ctx, loopRunID, at)
}

func (r *contextAwareLoopHookStore) EnqueueLoopCoordinatorWake(
	ctx context.Context,
	loopRunID string,
	idempotencyKey string,
	origin taskpkg.Origin,
	now time.Time,
) (taskpkg.Run, bool, error) {
	if err := ctx.Err(); err != nil {
		return taskpkg.Run{}, false, err
	}
	return r.recordingLoopHookStore.EnqueueLoopCoordinatorWake(ctx, loopRunID, idempotencyKey, origin, now)
}
