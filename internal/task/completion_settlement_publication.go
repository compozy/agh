package task

import "context"

func (m *Service) publishCompletedLeaseSettlement(
	ctx context.Context,
	settlement *CompletedRunSettlement,
	actor ActorContext,
) (*Run, error) {
	run := settlement.Run
	reconciledTask, err := m.publishCompletedRunSettlement(ctx, settlement, actor)
	if err != nil {
		return nil, err
	}
	m.dispatchTerminalWake(ctx, reconciledTask, run, actor)
	advisoryCtx, advisoryCancel := context.WithTimeout(context.WithoutCancel(ctx), autoEnqueueDispatchTimeout)
	defer advisoryCancel()
	m.recordCompletionHallucinationSuspected(advisoryCtx, run, actor)
	m.dispatchTaskRunCompleted(ctx, run, reconciledTask, actor)
	if !run.IsLoopWorker() {
		autoCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), autoEnqueueDispatchTimeout)
		defer cancel()
		for _, transition := range settlement.StatusTransitions {
			m.autoEnqueueReadyDependents(autoCtx, transition.Task.ID, autoEnqueueTrigger{
				Kind: autoEnqueueTriggerDependencyCompletion,
				Ref:  run.ID,
			}, actor)
		}
	}
	return &run, nil
}

func (m *Service) publishCompletedRunSettlement(
	ctx context.Context,
	settlement *CompletedRunSettlement,
	actor ActorContext,
) (Task, error) {
	for _, transition := range settlement.StatusTransitions {
		m.dispatchTaskStatusChanged(
			ctx,
			transition.Task,
			transition.PreviousStatus,
			transition.Task.Status,
			actor,
		)
		if err := m.reconcileDependentTasks(
			ctx,
			transition.Task.ID,
			map[string]struct{}{transition.Task.ID: {}},
			actor,
		); err != nil {
			return Task{}, err
		}
	}

	rolledUpTasks := make(map[string]Task, len(settlement.StatusTransitions))
	for _, transition := range settlement.StatusTransitions {
		rolledUpTasks[transition.Task.ID] = transition.Task
	}
	for _, run := range settlement.RolledUpRuns {
		rolledUpTask, ok := rolledUpTasks[run.TaskID]
		if !ok {
			continue
		}
		m.dispatchTerminalWake(ctx, rolledUpTask, run, actor)
		m.dispatchTaskRunCompleted(ctx, run, rolledUpTask, actor)
	}

	return settlement.Task, nil
}
