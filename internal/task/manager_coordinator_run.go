package task

import (
	"context"
	"fmt"
	"strings"
)

func (m *Service) startCoordinatorRun(
	ctx context.Context,
	_ Task,
	run Run,
	req StartRun,
	actor ActorContext,
) (*Run, error) {
	if err := m.validateCoordinatorRuntime(run.ID, req); err != nil {
		return nil, err
	}
	if !VerifyClaimToken(req.ClaimToken, run.ClaimTokenHash) {
		return nil, fmt.Errorf("%w: coordinator run %q claim token mismatch", ErrInvalidClaimToken, run.ID)
	}

	lifecycleCtx := taskRunLifecycleContext(ctx)
	run.Status = TaskRunStatusStarting
	if err := m.store.UpdateTaskRun(lifecycleCtx, run); err != nil {
		return nil, err
	}
	startingTask, err := m.reconcileTaskCascade(lifecycleCtx, run.TaskID, actor)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(lifecycleCtx, run.TaskID, run.ID, taskEventRunStarting, actor, runTransitionPayload{
		Status:     run.Status,
		TaskStatus: startingTask.Status,
	}); err != nil {
		return nil, err
	}

	run.Status = TaskRunStatusRunning
	run.StartedAt = m.now().UTC()
	if err := m.store.UpdateTaskRun(lifecycleCtx, run); err != nil {
		return nil, err
	}
	runningTask, err := m.reconcileTaskCascade(lifecycleCtx, run.TaskID, actor)
	if err != nil {
		return nil, err
	}
	if err := m.recordTaskEvent(lifecycleCtx, run.TaskID, run.ID, taskEventRunStarted, actor, runTransitionPayload{
		Status:     run.Status,
		TaskStatus: runningTask.Status,
	}); err != nil {
		return nil, err
	}

	plan, err := m.coordinatorRunner.Run(lifecycleCtx, RunID(run.ID))
	if err != nil {
		failedRun, failErr := m.FailRunLease(lifecycleCtx, LeaseFailure{
			RunID:      run.ID,
			ClaimToken: req.ClaimToken,
			Failure: RunFailure{
				Error: fmt.Sprintf("coordinator runner: %v", err),
			},
			Now: m.now().UTC(),
		}, actor)
		if failErr != nil {
			return nil, errorsJoin(err, failErr)
		}
		return failedRun, fmt.Errorf("task: coordinator run %q failed: %w", run.ID, err)
	}
	plan = plan.Normalize()
	if err := m.validateCoordinatorPlan(plan, "coordinator_completion.plan"); err != nil {
		return nil, err
	}

	result, completionErr := m.store.CompleteCoordinatorAndEnqueueNext(lifecycleCtx, CoordinatorCompletion{
		RunID:      run.ID,
		ClaimToken: req.ClaimToken,
		Plan:       plan,
		Actor:      actor,
		Now:        m.now().UTC(),
	}, m.generationFinalizer)
	if strings.TrimSpace(result.Run.ID) == "" {
		return nil, completionErr
	}
	if eventErr := m.recordCoordinatorCompletionEvents(lifecycleCtx, &result, actor); eventErr != nil {
		return &result.Run, errorsJoin(completionErr, eventErr)
	}
	m.dispatchCoordinatorTerminal(lifecycleCtx, &result, actor)
	return &result.Run, completionErr
}

func (m *Service) validateCoordinatorRuntime(runID string, req StartRun) error {
	if m.coordinatorRunner == nil {
		return fmt.Errorf("%w: coordinator runner is required", ErrValidation)
	}
	if m.generationFinalizer == nil {
		return fmt.Errorf("%w: generation state finalizer is required", ErrValidation)
	}
	if strings.TrimSpace(req.ClaimToken) == "" {
		return fmt.Errorf(
			"%w: coordinator run %q requires claim_token",
			ErrInvalidClaimToken,
			runID,
		)
	}
	return nil
}

func (m *Service) recordCoordinatorCompletionEvents(
	ctx context.Context,
	result *CoordinatorCompletionResult,
	actor ActorContext,
) error {
	if result == nil {
		return fmt.Errorf("%w: coordinator completion result is required", ErrValidation)
	}
	completedTask, err := m.reconcileTaskCascade(ctx, result.Run.TaskID, actor)
	if err != nil {
		return err
	}
	if err := m.recordTaskEvent(
		ctx,
		result.Run.TaskID,
		result.Run.ID,
		taskEventRunCompleted,
		actor,
		completedRunPayload{
			Status:         result.Run.Status,
			TaskStatus:     completedTask.Status,
			Result:         cloneRawJSON(result.Run.Result),
			ClaimTokenHash: result.Run.ClaimTokenHash,
		},
	); err != nil {
		return err
	}
	m.dispatchTaskRunCompleted(ctx, result.Run, completedTask, actor)

	for _, enqueued := range result.EnqueuedRuns {
		enqueuedTask, err := m.reconcileTaskCascade(ctx, enqueued.TaskID, actor)
		if err != nil {
			return err
		}
		if err := m.recordTaskEvent(ctx, enqueued.TaskID, enqueued.ID, taskEventRunEnqueued, actor, runEnqueuedPayload{
			Attempt:               int(enqueued.Attempt),
			Status:                enqueued.Status,
			TaskStatus:            enqueuedTask.Status,
			NetworkChannel:        enqueued.NetworkChannel,
			CoordinationChannelID: enqueued.CoordinationChannelID,
			IdempotencyKey:        enqueued.IdempotencyKey,
		}); err != nil {
			return err
		}
		m.dispatchTaskRunEnqueued(ctx, enqueued, enqueuedTask, actor, enqueued.IdempotencyKey)
	}
	return nil
}
