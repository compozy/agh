package task

import (
	"context"
	"strings"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

func (m *Service) dispatchTaskRunLeaseExtended(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
) error {
	payload := hookspkg.TaskRunLeaseExtendedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunLeaseExtended,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext: m.taskRunHookContext(run, taskRecord, actor),
	}
	_, err := m.taskHooks.DispatchTaskRunLeaseExtended(ctx, payload)
	return err
}

func (m *Service) dispatchTaskRunLeaseExpired(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
	recovery *ExpiredLeaseRecoveryResult,
) error {
	if recovery == nil {
		return nil
	}
	contextPayload := m.taskRunHookContext(run, taskRecord, actor)
	contextPayload.ReleaseReason = strings.TrimSpace(recovery.Reason)
	contextPayload.LeaseUntil = recovery.PreviousLeaseUntil
	payload := hookspkg.TaskRunLeaseExpiredPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunLeaseExpired,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext:    contextPayload,
		PreviousRunStatus: string(recovery.PreviousRunStatus.Normalize()),
		PreviousSessionID: strings.TrimSpace(recovery.PreviousSessionID),
		RecoveryReason:    strings.TrimSpace(recovery.Reason),
	}
	_, err := m.taskHooks.DispatchTaskRunLeaseExpired(ctx, payload)
	return err
}

func (m *Service) dispatchTaskRunLeaseRecoveredFromExpiration(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
	recovery *ExpiredLeaseRecoveryResult,
) error {
	if recovery == nil {
		return nil
	}
	payload := hookspkg.TaskRunLeaseRecoveredPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunLeaseRecovered,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext:    m.taskRunHookContext(run, taskRecord, actor),
		PreviousRunStatus: string(recovery.PreviousRunStatus.Normalize()),
		PreviousSessionID: strings.TrimSpace(recovery.PreviousSessionID),
		RecoveryAction:    string(RunBootRecoveryRequeue),
		RecoveryReason:    strings.TrimSpace(recovery.Reason),
	}
	_, err := m.taskHooks.DispatchTaskRunLeaseRecovered(ctx, payload)
	return err
}

func (m *Service) dispatchTaskRunReleased(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
	previous Run,
	reason string,
) error {
	contextPayload := m.taskRunHookContext(run, taskRecord, actor)
	contextPayload.ReleaseReason = strings.TrimSpace(reason)
	payload := hookspkg.TaskRunReleasedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunReleased,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext:    contextPayload,
		PreviousRunStatus: string(previous.Status.Normalize()),
		PreviousSessionID: strings.TrimSpace(previous.SessionID),
		RecoveryReason:    strings.TrimSpace(reason),
	}
	_, err := m.taskHooks.DispatchTaskRunReleased(ctx, payload)
	return err
}

func (m *Service) dispatchTaskRunCompleted(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
) error {
	payload := hookspkg.TaskRunCompletedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunCompleted,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext: m.taskRunHookContext(run, taskRecord, actor),
	}
	_, err := m.taskHooks.DispatchTaskRunCompleted(ctx, payload)
	return err
}

func (m *Service) dispatchTaskRunFailed(
	ctx context.Context,
	run Run,
	taskRecord Task,
	actor ActorContext,
) error {
	payload := hookspkg.TaskRunFailedPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookTaskRunFailed,
			Timestamp: m.now().UTC(),
		},
		TaskRunContext: m.taskRunHookContext(run, taskRecord, actor),
	}
	_, err := m.taskHooks.DispatchTaskRunFailed(ctx, payload)
	return err
}
