package task

import (
	"context"
	"fmt"
	"strings"
)

// EnqueueRun persists one new queue-first task run under manager authority.
func (m *Service) EnqueueRun(
	ctx context.Context,
	spec EnqueueRun,
	actor ActorContext,
) (*Run, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	normalizedSpec, err := normalizeEnqueueRunSpec(spec)
	if err != nil {
		return nil, err
	}
	if err := requireLifecycleIdempotency(actor, normalizedSpec.IdempotencyKey, "enqueue_run"); err != nil {
		return nil, err
	}
	if existing, ok, err := m.existingQueuedRun(
		ctx,
		normalizedSpec.TaskID,
		normalizedSpec.IdempotencyKey,
		actor.Origin,
	); err != nil {
		return nil, err
	} else if ok {
		return existing, nil
	}
	taskRecord, err := m.store.GetTask(ctx, normalizedSpec.TaskID)
	if err != nil {
		return nil, err
	}
	runID := m.newID("run")
	networkSpec, err := m.resolveQueuedRunParticipation(
		ctx,
		taskRecord,
		runID,
		normalizedSpec.RunKind,
		normalizedSpec.LoopRunID,
		normalizedSpec.NetworkParticipation,
		normalizedSpec.NetworkParticipationSource,
	)
	if err != nil {
		return nil, err
	}

	_, run, existing, err := m.store.ReserveQueuedRun(
		ctx,
		QueueRunReservation{
			TaskID:             normalizedSpec.TaskID,
			RunID:              runID,
			RunKind:            normalizedSpec.RunKind,
			LoopRunID:          normalizedSpec.LoopRunID,
			IdempotencyKey:     normalizedSpec.IdempotencyKey,
			Origin:             actor.Origin,
			NetworkSpec:        networkSpec,
			DesignationGroupID: normalizedSpec.DesignationGroupID,
			Metadata:           normalizedSpec.Metadata,
			QueuedAt:           m.now().UTC(),
		},
	)
	if err != nil {
		return nil, err
	}
	if existing {
		return &run, nil
	}

	if err := m.publishEnqueuedRun(ctx, run, actor, normalizedSpec.IdempotencyKey); err != nil {
		return nil, err
	}

	return &run, nil
}

func (m *Service) publishEnqueuedRun(
	ctx context.Context,
	run Run,
	actor ActorContext,
	idempotencyKey string,
) error {
	reconciledTask, err := m.reconcileTaskCascade(ctx, run.TaskID, actor)
	if err != nil {
		return err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunEnqueued, actor, runEnqueuedPayload{
		Attempt:        int(run.Attempt),
		Status:         run.Status,
		TaskStatus:     reconciledTask.Status,
		IdempotencyKey: run.IdempotencyKey,
	}); err != nil {
		return err
	}
	m.dispatchTaskRunEnqueued(ctx, run, reconciledTask, actor, idempotencyKey)
	return nil
}

func validateTaskForEnqueue(taskRecord Task) error {
	switch taskRecord.Status.Normalize() {
	case TaskStatusDraft:
		return fmt.Errorf("%w: task %q is draft", ErrInvalidStatusTransition, taskRecord.ID)
	case TaskStatusCanceled:
		return fmt.Errorf("%w: task %q is canceled", ErrInvalidStatusTransition, taskRecord.ID)
	default:
		return nil
	}
}

// StartRun transitions one claimed or starting run into active execution.
func (m *Service) StartRun(
	ctx context.Context,
	runID string,
	req StartRun,
	actor ActorContext,
) (*Run, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}

	normalizedReq, err := normalizeStartRun(req)
	if err != nil {
		return nil, err
	}
	if err := requireLifecycleIdempotency(actor, normalizedReq.IdempotencyKey, "start_run"); err != nil {
		return nil, err
	}

	run, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := m.ensureTaskExecutable(ctx, taskRecord); err != nil {
		return nil, err
	}
	switch run.Status.Normalize() {
	case TaskRunStatusClaimed:
		if run.RunKind.Normalize() == RunKindCoordinator {
			return m.startCoordinatorRun(ctx, taskRecord, run, normalizedReq, actor)
		}
		var failedRun *Run
		run, failedRun, err = m.transitionClaimedRunToStarting(ctx, taskRecord, run, actor)
		if err != nil {
			if failedRun != nil {
				return failedRun, err
			}
			return nil, err
		}
	case TaskRunStatusStarting:
		if err := validateRunningSessionBinding(run); err != nil {
			return nil, err
		}
	default:
		return nil, requireRunTransition(run, TaskRunStatusRunning)
	}

	lifecycleCtx := taskRunLifecycleContext(ctx)
	run.Status = TaskRunStatusRunning
	run.StartedAt = m.now().UTC()
	if err := m.store.UpdateTaskRun(lifecycleCtx, run); err != nil {
		return nil, err
	}

	reconciledTask, err := m.reconcileTaskCascade(lifecycleCtx, run.TaskID, actor)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(lifecycleCtx, run.TaskID, run.ID, taskEventRunStarted, actor, runTransitionPayload{
		Status:     run.Status,
		TaskStatus: reconciledTask.Status,
		SessionID:  run.SessionID,
	}); err != nil {
		return nil, err
	}

	return &run, nil
}

// AttachRunSession binds one existing session to a claimed or starting run.
func (m *Service) AttachRunSession(
	ctx context.Context,
	runID string,
	sessionID string,
	actor ActorContext,
) (*Run, error) {
	if err := requireWriteAuthority(actor); err != nil {
		return nil, err
	}
	if err := m.requireSessionExecutor("attach run session"); err != nil {
		return nil, err
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return nil, fmt.Errorf("%w: session id is required", ErrValidation)
	}

	run, taskRecord, err := m.loadRunWithTask(ctx, runID)
	if err != nil {
		return nil, err
	}
	if err := m.ensureTaskExecutable(ctx, taskRecord); err != nil {
		return nil, err
	}
	if strings.TrimSpace(run.SessionID) != "" {
		return nil, ErrSessionAlreadyBound
	}

	switch run.Status.Normalize() {
	case TaskRunStatusClaimed, TaskRunStatusStarting:
	default:
		return nil, ErrSessionAttachNotAllowed
	}

	activeBindings, err := m.store.CountActiveSessionBindings(ctx, trimmedSessionID)
	if err != nil {
		return nil, err
	}
	if activeBindings > 0 {
		return nil, ErrSessionAlreadyBound
	}

	sessionRef, err := m.sessions.AttachTaskSession(ctx, run.ID, trimmedSessionID)
	if err != nil {
		return nil, err
	}
	if sessionRef == nil {
		return nil, fmt.Errorf(
			"%w: attach_task_session returned nil session reference",
			ErrValidation,
		)
	}
	if err := sessionRef.Validate(); err != nil {
		return nil, err
	}

	run.SessionID = strings.TrimSpace(sessionRef.SessionID)
	if run.Status.Normalize() == TaskRunStatusClaimed {
		run.Status = TaskRunStatusStarting
	}
	if err := m.store.UpdateTaskRun(ctx, run); err != nil {
		return nil, err
	}

	reconciledTask, err := m.reconcileTaskCascade(ctx, run.TaskID, actor)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(ctx, run.TaskID, run.ID, taskEventRunSessionBound, actor, runTransitionPayload{
		Status:     run.Status,
		TaskStatus: reconciledTask.Status,
		SessionID:  run.SessionID,
	}); err != nil {
		return nil, err
	}

	return &run, nil
}
