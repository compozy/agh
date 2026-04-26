package task

import (
	"context"
	"errors"
	"testing"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

func TestNoopRunHookDispatcherPreservesRunLifecycle(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTest(t, store)
	actor := validActorContext()

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "No-op hook task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	claimed, err := manager.ClaimRun(context.Background(), run.ID, ClaimRun{}, actor)
	if err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	if got, want := claimed.Status, TaskRunStatusClaimed; got != want {
		t.Fatalf("claimed.Status = %q, want %q", got, want)
	}

	events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID, RunID: run.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if !containsEventType(events, taskEventRunEnqueued) || !containsEventType(events, taskEventRunClaimed) {
		t.Fatalf("event types = %#v, want enqueue and claim audit events", sortedEventTypes(events))
	}
}

func TestTaskRunPreClaimHookDenialPreservesQueuedRun(t *testing.T) {
	t.Parallel()

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTestWithOptions(t, store, WithTaskRunHooks(recordingTaskRunHooks{
		preClaim: func(
			_ context.Context,
			payload hookspkg.TaskRunPreClaimPayload,
		) (hookspkg.TaskRunPreClaimPayload, error) {
			payload.Denied = true
			payload.DenyReason = "agent lacks reviewer capability"
			return payload, nil
		},
	}))
	actor := validActorContext()

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Denied claim task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	_, err = manager.ClaimRun(context.Background(), run.ID, ClaimRun{}, actor)
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("ClaimRun() error = %v, want %v", err, ErrPermissionDenied)
	}

	storedRun, err := store.GetTaskRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetTaskRun() error = %v", err)
	}
	if got, want := storedRun.Status, TaskRunStatusQueued; got != want {
		t.Fatalf("storedRun.Status = %q, want %q", got, want)
	}
	events, err := store.ListTaskEvents(context.Background(), EventQuery{TaskID: taskRecord.ID, RunID: run.ID})
	if err != nil {
		t.Fatalf("ListTaskEvents() error = %v", err)
	}
	if containsEventType(events, taskEventRunClaimed) {
		t.Fatalf("event types = %#v, want no claimed audit event after denied pre-claim hook", sortedEventTypes(events))
	}
}

type recordingTaskRunHooks struct {
	enqueued  func(context.Context, hookspkg.TaskRunEnqueuedPayload) (hookspkg.TaskRunEnqueuedPayload, error)
	preClaim  func(context.Context, hookspkg.TaskRunPreClaimPayload) (hookspkg.TaskRunPreClaimPayload, error)
	postClaim func(context.Context, hookspkg.TaskRunPostClaimPayload) (hookspkg.TaskRunPostClaimPayload, error)
	recovered func(
		context.Context,
		hookspkg.TaskRunLeaseRecoveredPayload,
	) (hookspkg.TaskRunLeaseRecoveredPayload, error)
}

func (h recordingTaskRunHooks) DispatchTaskRunEnqueued(
	ctx context.Context,
	payload hookspkg.TaskRunEnqueuedPayload,
) (hookspkg.TaskRunEnqueuedPayload, error) {
	if h.enqueued != nil {
		return h.enqueued(ctx, payload)
	}
	return payload, nil
}

func (h recordingTaskRunHooks) DispatchTaskRunPreClaim(
	ctx context.Context,
	payload hookspkg.TaskRunPreClaimPayload,
) (hookspkg.TaskRunPreClaimPayload, error) {
	if h.preClaim != nil {
		return h.preClaim(ctx, payload)
	}
	return payload, nil
}

func (h recordingTaskRunHooks) DispatchTaskRunPostClaim(
	ctx context.Context,
	payload hookspkg.TaskRunPostClaimPayload,
) (hookspkg.TaskRunPostClaimPayload, error) {
	if h.postClaim != nil {
		return h.postClaim(ctx, payload)
	}
	return payload, nil
}

func (h recordingTaskRunHooks) DispatchTaskRunLeaseRecovered(
	ctx context.Context,
	payload hookspkg.TaskRunLeaseRecoveredPayload,
) (hookspkg.TaskRunLeaseRecoveredPayload, error) {
	if h.recovered != nil {
		return h.recovered(ctx, payload)
	}
	return payload, nil
}
