package task

import (
	"context"
	"strings"
)

// RunReadAuthorizer owns task-backed and taskless run-read policy.
type RunReadAuthorizer interface {
	AuthorizeRunRead(ctx context.Context, actor ActorContext, run Run, task *Task) error
}

type taskRunReadAuthorizer struct{}

var _ RunReadAuthorizer = taskRunReadAuthorizer{}

func (taskRunReadAuthorizer) AuthorizeRunRead(
	_ context.Context,
	actor ActorContext,
	run Run,
	taskRecord *Task,
) error {
	if taskRecord != nil {
		return authorizeTaskBackedRunRead(actor, *taskRecord)
	}
	return authorizeTasklessRunRead(actor, run)
}

func authorizeTaskBackedRunRead(actor ActorContext, taskRecord Task) error {
	if actor.Scope.Operator || actor.Actor.Kind.Normalize() == ActorKindDaemon {
		return nil
	}
	if taskRecord.Owner != nil &&
		string(taskRecord.Owner.Kind.Normalize()) == string(actor.Actor.Kind.Normalize()) &&
		strings.TrimSpace(taskRecord.Owner.Ref) == strings.TrimSpace(actor.Actor.Ref) {
		return nil
	}
	if taskRecord.Scope.Normalize() != ScopeWorkspace {
		return nil
	}
	if strings.TrimSpace(actor.Scope.WorkspaceID) != strings.TrimSpace(taskRecord.WorkspaceID) {
		return ErrPermissionDenied
	}
	return nil
}

func authorizeTasklessRunRead(actor ActorContext, run Run) error {
	if actor.Scope.Operator || actor.Actor.Kind.Normalize() == ActorKindDaemon {
		return nil
	}
	if run.RunKind.Normalize() != RunKindNetworkWake || actor.Actor.Kind.Normalize() != ActorKindAgentSession {
		return ErrPermissionDenied
	}

	_, targetSessionID, _ := run.NetworkWakeCorrelation()
	if strings.TrimSpace(actor.Scope.SessionID) != strings.TrimSpace(targetSessionID) ||
		strings.TrimSpace(actor.Scope.WorkspaceID) != strings.TrimSpace(run.WorkspaceID) {
		return ErrTaskRunNotFound
	}
	return nil
}
