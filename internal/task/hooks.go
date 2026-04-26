package task

import (
	"context"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

// RunHookDispatcher is the narrow hook bridge consumed by task-run transitions.
type RunHookDispatcher interface {
	DispatchTaskRunEnqueued(
		context.Context,
		hookspkg.TaskRunEnqueuedPayload,
	) (hookspkg.TaskRunEnqueuedPayload, error)
	DispatchTaskRunPreClaim(
		context.Context,
		hookspkg.TaskRunPreClaimPayload,
	) (hookspkg.TaskRunPreClaimPayload, error)
	DispatchTaskRunPostClaim(
		context.Context,
		hookspkg.TaskRunPostClaimPayload,
	) (hookspkg.TaskRunPostClaimPayload, error)
	DispatchTaskRunLeaseRecovered(
		context.Context,
		hookspkg.TaskRunLeaseRecoveredPayload,
	) (hookspkg.TaskRunLeaseRecoveredPayload, error)
}

var _ RunHookDispatcher = noopTaskRunHooks{}

type noopTaskRunHooks struct{}

func (noopTaskRunHooks) DispatchTaskRunEnqueued(
	_ context.Context,
	payload hookspkg.TaskRunEnqueuedPayload,
) (hookspkg.TaskRunEnqueuedPayload, error) {
	return payload, nil
}

func (noopTaskRunHooks) DispatchTaskRunPreClaim(
	_ context.Context,
	payload hookspkg.TaskRunPreClaimPayload,
) (hookspkg.TaskRunPreClaimPayload, error) {
	return payload, nil
}

func (noopTaskRunHooks) DispatchTaskRunPostClaim(
	_ context.Context,
	payload hookspkg.TaskRunPostClaimPayload,
) (hookspkg.TaskRunPostClaimPayload, error) {
	return payload, nil
}

func (noopTaskRunHooks) DispatchTaskRunLeaseRecovered(
	_ context.Context,
	payload hookspkg.TaskRunLeaseRecoveredPayload,
) (hookspkg.TaskRunLeaseRecoveredPayload, error) {
	return payload, nil
}

func defaultTaskRunHooks(hooks RunHookDispatcher) RunHookDispatcher {
	if hooks != nil {
		return hooks
	}
	return noopTaskRunHooks{}
}
