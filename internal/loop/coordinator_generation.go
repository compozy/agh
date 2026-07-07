package loop

import (
	"context"

	"github.com/compozy/agh/internal/loop/dsl"
	"github.com/compozy/agh/internal/loop/gate"
	"github.com/compozy/agh/internal/task"
)

func (r *CoordinatorRunner) buildGenerationFinisherPlan(
	ctx context.Context,
	taskRun task.Run,
	run Run,
	generation int,
	resolved *ResolvedDefinition,
	effective EffectiveConfig,
	fanOutWidth int,
	outputs []GenerationOutput,
) (task.CoordinatorCompletionPlan, error) {
	def := resolved.Definition
	graph := def.Graph
	topology := newControlTopology(graph)
	normalized, failed, live, loopStops, err := r.refreshGenerationOutputs(
		ctx,
		run,
		generation,
		graph,
		topology,
		outputs,
	)
	if err != nil {
		return task.CoordinatorCompletionPlan{}, err
	}
	plan := coordinatorFinisherPlan(run, generation, normalized, loopStops)
	plan.GenerationInFlight = live
	if failed != nil {
		return r.buildFailedGenerationPlan(
			ctx,
			taskRun,
			run,
			generation,
			def,
			effective,
			plan,
			normalized,
			*failed,
			live,
			loopStops,
		)
	}
	return r.buildLiveGenerationPlan(
		ctx,
		taskRun,
		run,
		generation,
		resolved,
		effective,
		topology,
		r.gateEvaluator,
		fanOutWidth,
		r.watchRuntime(),
		plan,
		normalized,
		live,
	)
}

func (r *CoordinatorRunner) buildLiveGenerationPlan(
	ctx context.Context,
	taskRun task.Run,
	run Run,
	generation int,
	resolved *ResolvedDefinition,
	effective EffectiveConfig,
	topology controlTopology,
	gateEvaluator gate.GateEvaluator,
	fanOutWidth int,
	watchRuntime coordinatorWatchRuntime,
	plan task.CoordinatorCompletionPlan,
	normalized []GenerationOutput,
	live bool,
) (task.CoordinatorCompletionPlan, error) {
	graph := resolved.Definition.Graph
	advancedOutputs := cloneGenerationOutputs(normalized)
	terminal, err := advanceControlNodes(
		ctx,
		&plan,
		run,
		generation,
		resolved,
		topology,
		effective,
		gateEvaluator,
		r.store,
		fanOutWidth,
		watchRuntime,
		&advancedOutputs,
	)
	if err != nil {
		return task.CoordinatorCompletionPlan{}, err
	}
	plan.Snapshot.Payload = GenerationSnapshotPayload{Outputs: sortedGenerationOutputs(advancedOutputs)}
	if terminal != nil {
		plan.Terminal = terminal
		return plan, nil
	}
	if plan.Yield {
		return plan, nil
	}
	hasReadyRuns, err := appendReadyNodeRunsToPlan(
		&plan,
		run,
		generation,
		resolved,
		topology,
		gateEvaluator,
		advancedOutputs,
	)
	if err != nil {
		return task.CoordinatorCompletionPlan{}, err
	}
	if hasReadyRuns {
		return plan, nil
	}
	if live {
		plan.Yield = true
		return plan, nil
	}
	if allGenerationOutputsSucceededControlAware(graph, topology, advancedOutputs) {
		return r.finishSucceededGenerationPlan(
			ctx,
			taskRun,
			run,
			generation,
			resolved,
			effective,
			topology,
			gateEvaluator,
			plan,
			advancedOutputs,
		)
	}
	plan.Terminal = noReadyNodesTerminal()
	return plan, nil
}

func (r *CoordinatorRunner) finishSucceededGenerationPlan(
	ctx context.Context,
	taskRun task.Run,
	run Run,
	generation int,
	resolved *ResolvedDefinition,
	effective EffectiveConfig,
	topology controlTopology,
	gateEvaluator gate.GateEvaluator,
	plan task.CoordinatorCompletionPlan,
	advancedOutputs []GenerationOutput,
) (task.CoordinatorCompletionPlan, error) {
	stopWhen, hasStopWhen, err := evaluateContractStopWhen(
		ctx,
		run,
		generation,
		resolved,
		topology,
		advancedOutputs,
	)
	if err != nil {
		return task.CoordinatorCompletionPlan{}, err
	}
	if hasStopWhen && !stopWhen {
		return r.buildStopWhenNextGenerationPlan(
			ctx,
			taskRun,
			run,
			generation,
			resolved.Definition.Graph,
			gateEvaluator != nil,
			plan,
			advancedOutputs,
		)
	}
	terminal, err := definitionOfDoneTerminal(
		ctx,
		run,
		generation,
		resolved,
		effective,
		topology,
		gateEvaluator,
		r.store,
		advancedOutputs,
	)
	if err != nil {
		return task.CoordinatorCompletionPlan{}, err
	}
	plan.Terminal = terminal
	return plan, nil
}

func (r *CoordinatorRunner) buildStopWhenNextGenerationPlan(
	ctx context.Context,
	taskRun task.Run,
	run Run,
	generation int,
	graph dsl.Graph,
	gatesEnabled bool,
	plan task.CoordinatorCompletionPlan,
	advancedOutputs []GenerationOutput,
) (task.CoordinatorCompletionPlan, error) {
	nextGeneration := generation + 1
	if terminal := iterationCapTerminal(run, nextGeneration); terminal != nil {
		plan.Terminal = terminal
		return plan, nil
	}
	if denied, deniedPlan := r.dispatchGenerationPre(ctx, taskRun, run, nextGeneration); denied {
		deniedPlan.Snapshot = plan.Snapshot
		return deniedPlan, nil
	}
	nextPlan, err := buildFreshGenerationCoordinatorPlan(
		taskRun,
		run,
		generation,
		nextGeneration,
		graph,
		gatesEnabled,
		advancedOutputs,
		plan.RunStops,
	)
	if err != nil {
		return task.CoordinatorCompletionPlan{}, err
	}
	r.dispatchGenerationPost(ctx, taskRun, run, nextPlan)
	return nextPlan, nil
}

func appendReadyNodeRunsToPlan(
	plan *task.CoordinatorCompletionPlan,
	run Run,
	generation int,
	resolved *ResolvedDefinition,
	topology controlTopology,
	gateEvaluator gate.GateEvaluator,
	advancedOutputs []GenerationOutput,
) (bool, error) {
	postReserveOutputs := cloneGenerationOutputs(sortedGenerationOutputs(advancedOutputs))
	if err := appendReadyNodeRunsControlAware(
		plan,
		run,
		generation,
		resolved,
		topology,
		gateEvaluator != nil,
		postReserveOutputs,
	); err != nil {
		return false, err
	}
	if len(plan.NodeRuns) == 0 {
		return false, nil
	}
	plan.PostReserveSnapshot = &task.GenerationSnapshot{
		LoopRunID:  string(run.ID),
		Generation: generation,
		Payload:    GenerationSnapshotPayload{Outputs: postReserveOutputs},
	}
	return true, nil
}

func (r *CoordinatorRunner) buildFailedGenerationPlan(
	ctx context.Context,
	taskRun task.Run,
	run Run,
	generation int,
	def dsl.Definition,
	effective EffectiveConfig,
	plan task.CoordinatorCompletionPlan,
	normalized []GenerationOutput,
	failed GenerationOutput,
	live bool,
	loopStops []task.CoordinatorStopSpec,
) (task.CoordinatorCompletionPlan, error) {
	terminal, terminalErr := r.terminalForFailedGeneration(
		ctx,
		run,
		generation,
		effective.NoProgressWindow,
		normalized,
		failed,
	)
	if terminalErr != nil {
		return task.CoordinatorCompletionPlan{}, terminalErr
	}
	if live || terminal.Status != string(StatusFailed) {
		plan.Terminal = terminal
		return plan, nil
	}
	nextGeneration := generation + 1
	if terminal := iterationCapTerminal(run, nextGeneration); terminal != nil {
		plan.Terminal = terminal
		return plan, nil
	}
	if denied, deniedPlan := r.dispatchGenerationPre(ctx, taskRun, run, nextGeneration); denied {
		deniedPlan.Snapshot = plan.Snapshot
		return deniedPlan, nil
	}
	reattemptPlan, err := buildReattemptCoordinatorPlan(
		taskRun,
		run,
		generation,
		nextGeneration,
		def.Graph,
		normalized,
		loopStops,
	)
	if err != nil {
		return task.CoordinatorCompletionPlan{}, err
	}
	r.dispatchGenerationPost(ctx, taskRun, run, reattemptPlan)
	return reattemptPlan, nil
}

func iterationCapTerminal(run Run, generation int) *task.CoordinatorTerminal {
	if run.IterationCap <= 0 || generation <= run.IterationCap {
		return nil
	}
	return &task.CoordinatorTerminal{
		Status:     string(StatusExhausted),
		Cause:      string(TransitionCauseIterationCap),
		ReasonCode: "iteration_cap_exceeded",
	}
}

func cloneGenerationOutputs(outputs []GenerationOutput) []GenerationOutput {
	cloned := make([]GenerationOutput, len(outputs))
	copy(cloned, outputs)
	return cloned
}

func coordinatorFinisherPlan(
	run Run,
	generation int,
	outputs []GenerationOutput,
	loopStops []task.CoordinatorStopSpec,
) task.CoordinatorCompletionPlan {
	return task.CoordinatorCompletionPlan{
		RunStops: loopStops,
		Snapshot: task.GenerationSnapshot{
			LoopRunID:  string(run.ID),
			Generation: generation,
			Payload:    GenerationSnapshotPayload{Outputs: outputs},
		},
	}
}
