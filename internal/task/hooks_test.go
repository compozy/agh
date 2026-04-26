package task

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestTaskRunEnqueuedHookIncludesActorAndOrigin(t *testing.T) {
	t.Parallel()

	var got hookspkg.TaskRunEnqueuedPayload
	store := newInMemoryManagerStore()
	manager := newTaskManagerForTestWithOptions(t, store, WithTaskRunHooks(recordingTaskRunHooks{
		enqueued: func(
			_ context.Context,
			payload hookspkg.TaskRunEnqueuedPayload,
		) (hookspkg.TaskRunEnqueuedPayload, error) {
			got = payload
			return payload, nil
		},
	}))
	actor, err := DeriveHumanActorContext("operator-1", OriginKindCLI, "agh task start")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}
	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Hook context task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	execution, err := manager.StartTask(context.Background(), taskRecord.ID, ExecutionRequest{}, actor)
	if err != nil {
		t.Fatalf("StartTask() error = %v", err)
	}
	if got.TaskID != taskRecord.ID || got.RunID != execution.Run.ID {
		t.Fatalf("hook context ids = %#v, want task/run ids", got.TaskRunContext)
	}
	if got.ActorKind != string(ActorKindHuman) || got.ActorRef != "operator-1" {
		t.Fatalf("hook actor context = %#v, want operator actor", got.TaskRunContext)
	}
	if got.OriginKind != string(OriginKindCLI) || got.OriginRef != "agh task start" {
		t.Fatalf("hook origin context = %#v, want cli origin", got.TaskRunContext)
	}
}

func TestTaskRunObservationHooksDetachFromCallerCancellation(t *testing.T) {
	t.Parallel()

	var enqueuedCtx context.Context
	var postClaimCtx context.Context
	store := newInMemoryManagerStore()
	manager := newTaskManagerForTestWithOptions(t, store, WithTaskRunHooks(recordingTaskRunHooks{
		enqueued: func(
			ctx context.Context,
			payload hookspkg.TaskRunEnqueuedPayload,
		) (hookspkg.TaskRunEnqueuedPayload, error) {
			enqueuedCtx = ctx
			return payload, nil
		},
		postClaim: func(
			ctx context.Context,
			payload hookspkg.TaskRunPostClaimPayload,
		) (hookspkg.TaskRunPostClaimPayload, error) {
			postClaimCtx = ctx
			return payload, nil
		},
	}))
	actor := validActorContext()
	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Observation hook context task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	enqueueCtx, cancelEnqueue := context.WithCancel(context.Background())
	run, err := manager.EnqueueRun(enqueueCtx, EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	cancelEnqueue()
	t.Run("Should keep enqueued hook context active", func(t *testing.T) {
		t.Parallel()
		assertContextStillActive(enqueuedCtx, t, "enqueued")
	})

	claimCtx, cancelClaim := context.WithCancel(context.Background())
	if _, err := manager.ClaimRun(claimCtx, run.ID, ClaimRun{}, actor); err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	cancelClaim()
	t.Run("Should keep post-claim hook context active", func(t *testing.T) {
		t.Parallel()
		assertContextStillActive(postClaimCtx, t, "post-claim")
	})
}

func TestTaskRunPreClaimHookUsesCallerCancellation(t *testing.T) {
	t.Parallel()

	var preClaimCtx context.Context
	store := newInMemoryManagerStore()
	manager := newTaskManagerForTestWithOptions(t, store, WithTaskRunHooks(recordingTaskRunHooks{
		preClaim: func(
			ctx context.Context,
			payload hookspkg.TaskRunPreClaimPayload,
		) (hookspkg.TaskRunPreClaimPayload, error) {
			preClaimCtx = ctx
			return payload, nil
		},
	}))
	actor := validActorContext()
	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Pre-claim hook context task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}

	claimCtx, cancelClaim := context.WithCancel(context.Background())
	if _, err := manager.ClaimRun(claimCtx, run.ID, ClaimRun{}, actor); err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	cancelClaim()

	if preClaimCtx == nil {
		t.Fatal("pre-claim hook context was not captured")
	}
	select {
	case <-preClaimCtx.Done():
	default:
		t.Fatal("pre-claim hook context was not canceled with caller context")
	}
}

func TestTokenFencedLeaseTransitionsDispatchTaskRunHooks(t *testing.T) {
	t.Parallel()

	var events []hookspkg.HookEvent
	record := func(event hookspkg.HookEvent) {
		events = append(events, event)
	}

	store := newInMemoryManagerStore()
	manager := newTaskManagerForTestWithOptions(t, store, WithTaskRunHooks(recordingTaskRunHooks{
		postClaim: func(
			_ context.Context,
			payload hookspkg.TaskRunPostClaimPayload,
		) (hookspkg.TaskRunPostClaimPayload, error) {
			record(payload.Event)
			return payload, nil
		},
		leaseExtended: func(
			_ context.Context,
			payload hookspkg.TaskRunLeaseExtendedPayload,
		) (hookspkg.TaskRunLeaseExtendedPayload, error) {
			record(payload.Event)
			return payload, nil
		},
		leaseExpired: func(
			_ context.Context,
			payload hookspkg.TaskRunLeaseExpiredPayload,
		) (hookspkg.TaskRunLeaseExpiredPayload, error) {
			record(payload.Event)
			return payload, nil
		},
		recovered: func(
			_ context.Context,
			payload hookspkg.TaskRunLeaseRecoveredPayload,
		) (hookspkg.TaskRunLeaseRecoveredPayload, error) {
			record(payload.Event)
			return payload, nil
		},
		released: func(
			_ context.Context,
			payload hookspkg.TaskRunReleasedPayload,
		) (hookspkg.TaskRunReleasedPayload, error) {
			record(payload.Event)
			return payload, nil
		},
		completed: func(
			_ context.Context,
			payload hookspkg.TaskRunCompletedPayload,
		) (hookspkg.TaskRunCompletedPayload, error) {
			record(payload.Event)
			return payload, nil
		},
		failed: func(
			_ context.Context,
			payload hookspkg.TaskRunFailedPayload,
		) (hookspkg.TaskRunFailedPayload, error) {
			record(payload.Event)
			return payload, nil
		},
	}))
	actor := validActorContext()
	agent := validActorContext()
	agent.Actor = ActorIdentity{Kind: ActorKindAgentSession, Ref: "sess-hooks"}
	agent.Origin = Origin{Kind: OriginKindAgentSession, Ref: "codex"}
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)

	taskRecord, err := manager.CreateTask(context.Background(), CreateTask{
		Scope: ScopeGlobal,
		Title: "Hooked lease task",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	claimAndRun := func(label string, at time.Time) *ClaimResult {
		t.Helper()
		if _, err := manager.EnqueueRun(context.Background(), EnqueueRun{TaskID: taskRecord.ID}, actor); err != nil {
			t.Fatalf("EnqueueRun(%s) error = %v", label, err)
		}
		claim, err := manager.ClaimNextRun(context.Background(), ClaimCriteria{
			Scope:            ScopeGlobal,
			ClaimerSessionID: "sess-hooks",
			LeaseDuration:    2 * time.Minute,
			Now:              at,
		}, agent)
		if err != nil {
			t.Fatalf("ClaimNextRun(%s) error = %v", label, err)
		}
		return claim
	}

	claim := claimAndRun("complete", now)
	if _, err := manager.HeartbeatRunLease(context.Background(), LeaseHeartbeat{
		RunID:         claim.Run.ID,
		ClaimToken:    claim.ClaimToken,
		LeaseDuration: 2 * time.Minute,
		Now:           now.Add(10 * time.Second),
	}, agent); err != nil {
		t.Fatalf("HeartbeatRunLease() error = %v", err)
	}
	if _, err := manager.CompleteRunLease(context.Background(), LeaseCompletion{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Result:     RunResult{},
		Now:        now.Add(20 * time.Second),
	}, agent); err != nil {
		t.Fatalf("CompleteRunLease() error = %v", err)
	}

	claim = claimAndRun("release", now.Add(time.Minute))
	if _, err := manager.ReleaseRunLease(context.Background(), LeaseRelease{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Reason:     "handoff",
		Now:        now.Add(70 * time.Second),
	}, agent); err != nil {
		t.Fatalf("ReleaseRunLease() error = %v", err)
	}

	claim, err = manager.ClaimNextRun(context.Background(), ClaimCriteria{
		Scope:            ScopeGlobal,
		ClaimerSessionID: "sess-hooks",
		LeaseDuration:    2 * time.Minute,
		Now:              now.Add(2 * time.Minute),
	}, agent)
	if err != nil {
		t.Fatalf("ClaimNextRun(fail) error = %v", err)
	}
	if _, err := manager.FailRunLease(context.Background(), LeaseFailure{
		RunID:      claim.Run.ID,
		ClaimToken: claim.ClaimToken,
		Failure:    RunFailure{Error: "boom"},
		Now:        now.Add(130 * time.Second),
	}, agent); err != nil {
		t.Fatalf("FailRunLease() error = %v", err)
	}

	expiring := claimAndRun("expire", now.Add(3*time.Minute))
	expiredRun := store.runs[expiring.Run.ID]
	expiredRun.LeaseUntil = now.Add(3*time.Minute - time.Second)
	store.runs[expiring.Run.ID] = expiredRun
	if recovered, err := manager.RecoverExpiredRunLeases(context.Background(), ExpiredLeaseRecovery{
		Now:    now.Add(3 * time.Minute),
		Reason: "orphaned_on_boot",
	}, agent); err != nil {
		t.Fatalf("RecoverExpiredRunLeases() error = %v", err)
	} else if got, want := len(recovered), 1; got != want {
		t.Fatalf("len(RecoverExpiredRunLeases()) = %d, want %d", got, want)
	}

	want := []hookspkg.HookEvent{
		hookspkg.HookTaskRunPostClaim,
		hookspkg.HookTaskRunLeaseExtended,
		hookspkg.HookTaskRunCompleted,
		hookspkg.HookTaskRunPostClaim,
		hookspkg.HookTaskRunReleased,
		hookspkg.HookTaskRunPostClaim,
		hookspkg.HookTaskRunFailed,
		hookspkg.HookTaskRunPostClaim,
		hookspkg.HookTaskRunLeaseExpired,
		hookspkg.HookTaskRunLeaseRecovered,
	}
	if len(events) != len(want) {
		t.Fatalf("events = %#v, want %#v", events, want)
	}
	for idx := range want {
		if events[idx] != want[idx] {
			t.Fatalf("events[%d] = %q, want %q (events=%#v)", idx, events[idx], want[idx], events)
		}
	}
}

func assertContextStillActive(ctx context.Context, t *testing.T, label string) {
	t.Helper()
	if ctx == nil {
		t.Fatalf("%s hook context was not captured", label)
	}
	select {
	case <-ctx.Done():
		t.Fatalf("%s hook context canceled after caller returned: %v", label, ctx.Err())
	default:
	}
}

type recordingTaskRunHooks struct {
	enqueued      func(context.Context, hookspkg.TaskRunEnqueuedPayload) (hookspkg.TaskRunEnqueuedPayload, error)
	preClaim      func(context.Context, hookspkg.TaskRunPreClaimPayload) (hookspkg.TaskRunPreClaimPayload, error)
	postClaim     func(context.Context, hookspkg.TaskRunPostClaimPayload) (hookspkg.TaskRunPostClaimPayload, error)
	leaseExtended func(
		context.Context,
		hookspkg.TaskRunLeaseExtendedPayload,
	) (hookspkg.TaskRunLeaseExtendedPayload, error)
	leaseExpired func(
		context.Context,
		hookspkg.TaskRunLeaseExpiredPayload,
	) (hookspkg.TaskRunLeaseExpiredPayload, error)
	recovered func(
		context.Context,
		hookspkg.TaskRunLeaseRecoveredPayload,
	) (hookspkg.TaskRunLeaseRecoveredPayload, error)
	released  func(context.Context, hookspkg.TaskRunReleasedPayload) (hookspkg.TaskRunReleasedPayload, error)
	completed func(context.Context, hookspkg.TaskRunCompletedPayload) (hookspkg.TaskRunCompletedPayload, error)
	failed    func(context.Context, hookspkg.TaskRunFailedPayload) (hookspkg.TaskRunFailedPayload, error)
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

func (h recordingTaskRunHooks) DispatchTaskRunLeaseExtended(
	ctx context.Context,
	payload hookspkg.TaskRunLeaseExtendedPayload,
) (hookspkg.TaskRunLeaseExtendedPayload, error) {
	if h.leaseExtended != nil {
		return h.leaseExtended(ctx, payload)
	}
	return payload, nil
}

func (h recordingTaskRunHooks) DispatchTaskRunLeaseExpired(
	ctx context.Context,
	payload hookspkg.TaskRunLeaseExpiredPayload,
) (hookspkg.TaskRunLeaseExpiredPayload, error) {
	if h.leaseExpired != nil {
		return h.leaseExpired(ctx, payload)
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

func (h recordingTaskRunHooks) DispatchTaskRunReleased(
	ctx context.Context,
	payload hookspkg.TaskRunReleasedPayload,
) (hookspkg.TaskRunReleasedPayload, error) {
	if h.released != nil {
		return h.released(ctx, payload)
	}
	return payload, nil
}

func (h recordingTaskRunHooks) DispatchTaskRunCompleted(
	ctx context.Context,
	payload hookspkg.TaskRunCompletedPayload,
) (hookspkg.TaskRunCompletedPayload, error) {
	if h.completed != nil {
		return h.completed(ctx, payload)
	}
	return payload, nil
}

func (h recordingTaskRunHooks) DispatchTaskRunFailed(
	ctx context.Context,
	payload hookspkg.TaskRunFailedPayload,
) (hookspkg.TaskRunFailedPayload, error) {
	if h.failed != nil {
		return h.failed(ctx, payload)
	}
	return payload, nil
}
