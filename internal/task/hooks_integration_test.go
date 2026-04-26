//go:build integration

package task_test

import (
	"context"
	"testing"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestTaskRunPostClaimHookDispatchesAfterAuditEventIntegration(t *testing.T) {
	db := openTaskManagerGlobalDB(t)
	postClaimSawCommittedAudit := false
	manager := newTaskManagerIntegration(t, db, taskpkg.WithTaskRunHooks(integrationTaskRunHooks{
		postClaim: func(
			ctx context.Context,
			payload hookspkg.TaskRunPostClaimPayload,
		) (hookspkg.TaskRunPostClaimPayload, error) {
			events, err := db.ListTaskEvents(ctx, taskpkg.EventQuery{
				TaskID: payload.TaskID,
				RunID:  payload.RunID,
			})
			if err != nil {
				t.Fatalf("ListTaskEvents() during post-claim hook error = %v", err)
			}
			if !containsEventType(events, "task.run_claimed") {
				t.Fatalf("event types during post-claim hook = %#v, want task.run_claimed", sortedEventTypes(events))
			}
			postClaimSawCommittedAudit = true
			return payload, nil
		},
	}))

	actor, err := taskpkg.DeriveHumanActorContext("user-1", taskpkg.OriginKindCLI, "agh task run")
	if err != nil {
		t.Fatalf("DeriveHumanActorContext() error = %v", err)
	}
	taskRecord, err := manager.CreateTask(testutil.Context(t), taskpkg.CreateTask{
		Scope: taskpkg.ScopeGlobal,
		Title: "Post-claim hook ordering",
	}, actor)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	run, err := manager.EnqueueRun(testutil.Context(t), taskpkg.EnqueueRun{TaskID: taskRecord.ID}, actor)
	if err != nil {
		t.Fatalf("EnqueueRun() error = %v", err)
	}
	if _, err := manager.ClaimRun(testutil.Context(t), run.ID, taskpkg.ClaimRun{}, actor); err != nil {
		t.Fatalf("ClaimRun() error = %v", err)
	}
	if !postClaimSawCommittedAudit {
		t.Fatal("post-claim hook did not observe committed claim audit event")
	}
}

type integrationTaskRunHooks struct {
	enqueued  func(context.Context, hookspkg.TaskRunEnqueuedPayload) (hookspkg.TaskRunEnqueuedPayload, error)
	preClaim  func(context.Context, hookspkg.TaskRunPreClaimPayload) (hookspkg.TaskRunPreClaimPayload, error)
	postClaim func(context.Context, hookspkg.TaskRunPostClaimPayload) (hookspkg.TaskRunPostClaimPayload, error)
	recovered func(
		context.Context,
		hookspkg.TaskRunLeaseRecoveredPayload,
	) (hookspkg.TaskRunLeaseRecoveredPayload, error)
}

func (h integrationTaskRunHooks) DispatchTaskRunEnqueued(
	ctx context.Context,
	payload hookspkg.TaskRunEnqueuedPayload,
) (hookspkg.TaskRunEnqueuedPayload, error) {
	if h.enqueued != nil {
		return h.enqueued(ctx, payload)
	}
	return payload, nil
}

func (h integrationTaskRunHooks) DispatchTaskRunPreClaim(
	ctx context.Context,
	payload hookspkg.TaskRunPreClaimPayload,
) (hookspkg.TaskRunPreClaimPayload, error) {
	if h.preClaim != nil {
		return h.preClaim(ctx, payload)
	}
	return payload, nil
}

func (h integrationTaskRunHooks) DispatchTaskRunPostClaim(
	ctx context.Context,
	payload hookspkg.TaskRunPostClaimPayload,
) (hookspkg.TaskRunPostClaimPayload, error) {
	if h.postClaim != nil {
		return h.postClaim(ctx, payload)
	}
	return payload, nil
}

func (h integrationTaskRunHooks) DispatchTaskRunLeaseExtended(
	_ context.Context,
	payload hookspkg.TaskRunLeaseExtendedPayload,
) (hookspkg.TaskRunLeaseExtendedPayload, error) {
	return payload, nil
}

func (h integrationTaskRunHooks) DispatchTaskRunLeaseExpired(
	_ context.Context,
	payload hookspkg.TaskRunLeaseExpiredPayload,
) (hookspkg.TaskRunLeaseExpiredPayload, error) {
	return payload, nil
}

func (h integrationTaskRunHooks) DispatchTaskRunLeaseRecovered(
	ctx context.Context,
	payload hookspkg.TaskRunLeaseRecoveredPayload,
) (hookspkg.TaskRunLeaseRecoveredPayload, error) {
	if h.recovered != nil {
		return h.recovered(ctx, payload)
	}
	return payload, nil
}

func (h integrationTaskRunHooks) DispatchTaskRunReleased(
	_ context.Context,
	payload hookspkg.TaskRunReleasedPayload,
) (hookspkg.TaskRunReleasedPayload, error) {
	return payload, nil
}

func (h integrationTaskRunHooks) DispatchTaskRunCompleted(
	_ context.Context,
	payload hookspkg.TaskRunCompletedPayload,
) (hookspkg.TaskRunCompletedPayload, error) {
	return payload, nil
}

func (h integrationTaskRunHooks) DispatchTaskRunFailed(
	_ context.Context,
	payload hookspkg.TaskRunFailedPayload,
) (hookspkg.TaskRunFailedPayload, error) {
	return payload, nil
}
