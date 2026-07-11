package globaldb

import (
	"context"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

var _ looppkg.GoalControlStore = (*GlobalDB)(nil)

// LoadAwaitingGoalControl returns the one node checkpoint that currently owns Run re-entry.
func (g *GlobalDB) LoadAwaitingGoalControl(
	ctx context.Context,
	workspaceID looppkg.WorkspaceID,
	runID looppkg.RunID,
) (looppkg.GoalControlState, bool, error) {
	if err := g.checkReady(ctx, "load awaiting Goal control"); err != nil {
		return looppkg.GoalControlState{}, false, err
	}
	if strings.TrimSpace(string(workspaceID)) == "" || strings.TrimSpace(string(runID)) == "" {
		return looppkg.GoalControlState{}, false, fmt.Errorf(
			"%w: workspace_id and run_id are required",
			looppkg.ErrValidation,
		)
	}
	row, found, err := loadAwaitingGoalControlWithExecutor(ctx, g.db, workspaceID, runID)
	if err != nil || !found {
		return looppkg.GoalControlState{}, found, err
	}
	return row.state, true, nil
}

// ReactivateGoalRun atomically grants control, clears the Run gate, and queues one successor segment.
func (g *GlobalDB) ReactivateGoalRun(
	ctx context.Context,
	req looppkg.GoalReactivationRequest,
) (looppkg.GoalReactivationResult, error) {
	if err := g.checkReady(ctx, "reactivate Goal run"); err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	if err := validateGoalReactivationRequest(req); err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	decisions, err := normalizeGoalReentryDecisions(req.Decisions, g.now)
	if err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	var result looppkg.GoalReactivationResult
	err = g.withTaskImmediateTransaction(ctx, "reactivate Goal run", func(exec taskSQLExecutor) error {
		applied, applyErr := g.reactivateGoalRunWithExecutor(ctx, exec, req, decisions)
		if applyErr != nil {
			return applyErr
		}
		result = applied
		return nil
	})
	if err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	return result, nil
}

func (g *GlobalDB) reactivateGoalRunWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req looppkg.GoalReactivationRequest,
	decisions []looppkg.GateDecisionRecord,
) (looppkg.GoalReactivationResult, error) {
	current, err := loadGoalReentryRowWithExecutor(ctx, exec, req.State)
	if err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	if !goalReentryStillAwaiting(current, req.State) {
		return g.replayedGoalReactivation(ctx, exec, current, req)
	}
	if err := validateGoalReentryControl(&current, req.State); err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	oldRun, err := g.loadCompletedGoalSegment(ctx, exec, current)
	if err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	successor, nextEpoch, grantID, err := g.reserveGoalReentrySuccessor(ctx, exec, current, oldRun)
	if err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	return g.applyGoalReentry(ctx, exec, req, decisions, current, successor, nextEpoch, grantID)
}

func validateGoalReentryControl(
	current *goalReentryRow,
	expected looppkg.GoalControlState,
) error {
	current.state.Cause = goalControlCause(current.state)
	if current.state.Cause == "" {
		return goalReentryStaleError("Goal control cause is invalid")
	}
	if current.state.Cause != expected.Cause || current.state.GoalStatus != expected.GoalStatus {
		return goalReentryStaleError("Goal control cause changed")
	}
	return nil
}

func (g *GlobalDB) loadCompletedGoalSegment(
	ctx context.Context,
	exec taskSQLExecutor,
	current goalReentryRow,
) (taskpkg.Run, error) {
	oldRun, err := g.getTaskRunWithExecutor(ctx, exec, current.state.TaskRunID)
	if err != nil {
		return taskpkg.Run{}, err
	}
	if oldRun.Status.Normalize() != taskpkg.TaskRunStatusCompleted || !oldRun.IsLoopWorker() {
		return taskpkg.Run{}, goalReentryStaleError("previous Goal segment is not completed")
	}
	metadata, ok, err := loopNodeMetadataFromTaskRun(oldRun.Metadata)
	if err != nil {
		return taskpkg.Run{}, err
	}
	if !ok || metadata.Generation != current.state.Generation ||
		metadata.NodeID != string(current.state.NodeID) || metadata.ItemIndex != current.state.ItemIndex {
		return taskpkg.Run{}, goalReentryStaleError("Goal worker metadata changed")
	}
	return oldRun, nil
}

func (g *GlobalDB) reserveGoalReentrySuccessor(
	ctx context.Context,
	exec taskSQLExecutor,
	current goalReentryRow,
	oldRun taskpkg.Run,
) (taskpkg.Run, int64, int64, error) {
	nextEpoch := current.state.ControlEpoch + 1
	grantID := int64(1)
	if current.grantID.Valid {
		grantID = current.grantID.Int64 + 1
	}
	successorRunID := looppkg.GoalSegmentRunID(
		current.state.LoopRunID,
		current.state.Generation,
		current.state.NodeID,
		current.state.ItemIndex,
		nextEpoch,
	)
	successorMetadata, err := goalSuccessorMetadata(oldRun.Metadata, nextEpoch)
	if err != nil {
		return taskpkg.Run{}, 0, 0, err
	}
	idempotencyKey := looppkg.GoalSegmentIdempotencyKey(
		current.state.LoopRunID,
		current.state.Generation,
		current.state.NodeID,
		current.state.ItemIndex,
		nextEpoch,
	)
	_, successor, existing, err := g.reserveQueuedRunWithExecutor(ctx, exec, queuedRunReservationInput{
		taskID:             oldRun.TaskID,
		runID:              successorRunID,
		runKind:            taskpkg.RunKindWorker,
		loopRunID:          string(current.state.LoopRunID),
		idempotencyKey:     idempotencyKey,
		origin:             oldRun.Origin,
		requestedChannel:   oldRun.NetworkChannel,
		designationGroupID: oldRun.DesignationGroupID,
		metadata:           successorMetadata,
		queuedAt:           g.now(),
	})
	if err != nil {
		return taskpkg.Run{}, 0, 0, err
	}
	if existing {
		return taskpkg.Run{}, 0, 0, goalReentryStaleError(
			"successor segment already existed before grant",
		)
	}
	return successor, nextEpoch, grantID, nil
}

func (g *GlobalDB) applyGoalReentry(
	ctx context.Context,
	exec taskSQLExecutor,
	req looppkg.GoalReactivationRequest,
	decisions []looppkg.GateDecisionRecord,
	current goalReentryRow,
	successor taskpkg.Run,
	nextEpoch int64,
	grantID int64,
) (looppkg.GoalReactivationResult, error) {
	turnLimit := current.state.TurnLimit + req.TurnIncrement
	if err := updateGoalPromptOwnerForReentry(ctx, exec, req, current, successor.ID, nextEpoch, g.now()); err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	if err := updateGoalCheckpointForReentry(
		ctx,
		exec,
		req,
		current,
		successor.ID,
		nextEpoch,
		grantID,
		turnLimit,
		g.now(),
	); err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	if err := updateGoalOutputForReentry(ctx, exec, current, successor.ID, turnLimit); err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	for _, decision := range decisions {
		if err := insertLoopGateDecision(ctx, exec, decision); err != nil {
			return looppkg.GoalReactivationResult{}, err
		}
	}
	if err := g.transitionGoalRunForReentry(ctx, exec, req, current); err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	if err := incrementGoalBudgetVersion(ctx, exec, current.state.LoopRunID); err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	return looppkg.GoalReactivationResult{
		Run:          successor,
		ControlEpoch: nextEpoch,
		GrantID:      grantID,
	}, nil
}

func (g *GlobalDB) replayedGoalReactivation(
	ctx context.Context,
	exec taskSQLExecutor,
	current goalReentryRow,
	req looppkg.GoalReactivationRequest,
) (looppkg.GoalReactivationResult, error) {
	nextEpoch := req.State.ControlEpoch + 1
	expectedRunID := looppkg.GoalSegmentRunID(
		req.State.LoopRunID,
		req.State.Generation,
		req.State.NodeID,
		req.State.ItemIndex,
		nextEpoch,
	)
	if current.state.ControlEpoch != nextEpoch || current.outputStatus != goalGenerationOutputEnqueued ||
		current.outputTaskRun != expectedRunID || !current.grantID.Valid ||
		current.grantKind.String != string(req.Kind) || current.grantCause.String != string(req.State.Cause) ||
		!current.grantTurn.Valid || current.grantTurn.Int64 != int64(goalReentryGrantTurn(req)) ||
		current.grantScope.String != string(req.Scope) || !current.grantConsumed.Valid ||
		(current.grantConsumed.Int64 != 0) != goalReentryGrantConsumed(req.Kind) {
		return looppkg.GoalReactivationResult{}, goalReentryStaleError("Goal control epoch changed")
	}
	run, err := g.getTaskRunWithExecutor(ctx, exec, expectedRunID)
	if err != nil {
		return looppkg.GoalReactivationResult{}, err
	}
	return looppkg.GoalReactivationResult{
		Run:          run,
		ControlEpoch: nextEpoch,
		GrantID:      current.grantID.Int64,
		Existing:     true,
	}, nil
}

func updateGoalCheckpointForReentry(
	ctx context.Context,
	exec taskSQLExecutor,
	req looppkg.GoalReactivationRequest,
	current goalReentryRow,
	successorRunID string,
	nextEpoch int64,
	grantID int64,
	turnLimit int,
	now time.Time,
) error {
	clearOperation := req.State.Cause == looppkg.ReasonCodeGoalRecoveryAmbiguous
	phase, err := goalCheckpointPhaseForReentry(ctx, exec, current, clearOperation)
	if err != nil {
		return err
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET control_epoch = ?, phase = ?, goal_status = 'active', control_cause = NULL,
		     turn_limit = ?, task_run_id = ?,
		     queue_entry_id = CASE WHEN ? = 1 THEN NULL ELSE queue_entry_id END,
		     prompt_id = CASE WHEN ? = 1 THEN NULL ELSE prompt_id END,
		     prompt_kind = CASE WHEN ? = 1 THEN NULL ELSE prompt_kind END,
		     judge_attempt_id = CASE WHEN ? = 1 THEN NULL ELSE judge_attempt_id END,
		     control_actor_kind = NULL, control_actor_id = NULL, control_requested_at = NULL,
		     control_grant_id = ?, control_grant_kind = ?, control_grant_cause = ?,
		     control_grant_turn = ?, control_grant_scope = ?, control_grant_consumed = ?, updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND phase = 'awaiting_control' AND task_run_id = ?`,
		nextEpoch,
		phase,
		turnLimit,
		successorRunID,
		boolToInt(clearOperation),
		boolToInt(clearOperation),
		boolToInt(clearOperation),
		boolToInt(clearOperation),
		grantID,
		string(req.Kind),
		string(req.State.Cause),
		goalReentryGrantTurn(req),
		string(req.Scope),
		boolToInt(goalReentryGrantConsumed(req.Kind)),
		store.FormatTimestamp(now),
		string(current.state.LoopRunID),
		current.state.Generation,
		string(current.state.NodeID),
		current.state.ItemIndex,
		current.state.ControlEpoch,
		current.state.TaskRunID,
	)
	if err != nil {
		return fmt.Errorf("store: reactivate Goal checkpoint: %w", err)
	}
	return requireGoalRowsAffected(result, "reactivate Goal checkpoint")
}

func goalCheckpointPhaseForReentry(
	ctx context.Context,
	exec taskSQLExecutor,
	current goalReentryRow,
	clearOperation bool,
) (string, error) {
	if clearOperation || strings.TrimSpace(current.state.QueueEntryID) == "" ||
		strings.TrimSpace(current.state.PromptID) == "" {
		return goalCheckpointPhaseIdle, nil
	}
	if !current.state.OpenTurn {
		return goalCheckpointPhaseQueued, nil
	}
	row, err := loadGoalPromptRow(ctx, exec, current.state.LoopRunID, current.state.PromptID)
	if err != nil {
		return "", err
	}
	if row.terminalAt == nil {
		return "", goalReentryStaleError("open Goal turn has no terminal result to settle")
	}
	if row.terminalKind.Valid && row.terminalKind.String == string(looppkg.ActionPromptOutcomeCompleted) &&
		row.terminalStopReason.Valid &&
		(row.terminalStopReason.String == string(looppkg.ActionStopEndTurn) ||
			row.terminalStopReason.String == string(looppkg.ActionStopMaxTurnRequests)) {
		return goalCheckpointPhaseJudging, nil
	}
	return goalCheckpointPhasePersisting, nil
}

func updateGoalPromptOwnerForReentry(
	ctx context.Context,
	exec taskSQLExecutor,
	req looppkg.GoalReactivationRequest,
	current goalReentryRow,
	successorRunID string,
	nextEpoch int64,
	now time.Time,
) error {
	if strings.TrimSpace(current.state.QueueEntryID) == "" ||
		strings.TrimSpace(current.state.PromptID) == "" ||
		req.State.Cause == looppkg.ReasonCodeGoalRecoveryAmbiguous {
		return nil
	}
	clearControlFence := req.Kind == looppkg.GoalGrantPlainResume
	result, err := exec.ExecContext(
		ctx,
		`UPDATE session_input_queue
		 SET task_run_id = ?, owner_epoch = ?,
		     fence_kind = CASE
		       WHEN ? = 1 AND fence_kind = 'control-fenced' THEN NULL
		       ELSE fence_kind
		     END,
		     fence_disposition = CASE
		       WHEN ? = 1 AND fence_kind = 'control-fenced' THEN NULL
		       ELSE fence_disposition
		     END,
		     fence_reason_code = CASE
		       WHEN ? = 1 AND fence_kind = 'control-fenced' THEN NULL
		       ELSE fence_reason_code
		     END,
		     fenced_at = CASE
		       WHEN ? = 1 AND fence_kind = 'control-fenced' THEN NULL
		       ELSE fenced_at
		     END,
		     updated_at = ?
		 WHERE id = ? AND loop_run_id = ? AND prompt_id = ? AND owner_kind = 'goal'
		   AND owner_epoch = ? AND task_run_id = ?`,
		successorRunID,
		nextEpoch,
		boolToInt(clearControlFence),
		boolToInt(clearControlFence),
		boolToInt(clearControlFence),
		boolToInt(clearControlFence),
		store.FormatTimestamp(now),
		current.state.QueueEntryID,
		string(current.state.LoopRunID),
		current.state.PromptID,
		current.state.ControlEpoch,
		current.state.TaskRunID,
	)
	if err != nil {
		return fmt.Errorf("store: reassign Goal prompt to successor segment: %w", err)
	}
	return requireGoalRowsAffected(result, "reassign Goal prompt to successor segment")
}

func updateGoalOutputForReentry(
	ctx context.Context,
	exec taskSQLExecutor,
	current goalReentryRow,
	successorRunID string,
	turnLimit int,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_generation_outputs
		 SET status = 'enqueued', task_run_id = ?, goal_status = 'active',
		     goal_turns_used = ?, goal_turn_limit = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND status = ? AND task_run_id = ?`,
		successorRunID,
		current.state.TurnsUsed,
		turnLimit,
		string(current.state.LoopRunID),
		current.state.Generation,
		string(current.state.NodeID),
		current.state.ItemIndex,
		looppkg.GenerationOutputStatusAwaitingGoal,
		current.state.TaskRunID,
	)
	if err != nil {
		return fmt.Errorf("store: reactivate Goal generation output: %w", err)
	}
	return requireGoalRowsAffected(result, "reactivate Goal generation output")
}

func (g *GlobalDB) transitionGoalRunForReentry(
	ctx context.Context,
	exec taskSQLExecutor,
	req looppkg.GoalReactivationRequest,
	current goalReentryRow,
) error {
	loopRun, err := getLoopRunByIDWithExecutor(ctx, exec, current.state.LoopRunID)
	if err != nil {
		return err
	}
	if loopRun.WorkspaceID != current.state.WorkspaceID || loopRun.Status != current.state.RunStatus ||
		loopRun.ActiveGateID != current.state.GateID {
		return goalReentryStaleError("Loop Run control changed")
	}
	cause := looppkg.TransitionCauseApproval
	if current.state.RunStatus == looppkg.StatusPaused {
		cause = looppkg.TransitionCauseOperatorResume
	}
	if err := updateLoopBoundaryStatusWithExecutor(
		ctx,
		exec,
		loopRun,
		looppkg.StatusRunning,
		cause,
		g.now(),
		current.state.Generation,
	); err != nil {
		return err
	}
	if _, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET control_actor_kind = NULL, control_actor_id = NULL, control_requested_at = NULL
		 WHERE id = ?`,
		string(current.state.LoopRunID),
	); err != nil {
		return fmt.Errorf("store: clear Goal control actor: %w", err)
	}
	return appendLoopRunEventWithExecutor(
		ctx,
		exec,
		current.state.LoopRunID,
		current.state.WorkspaceID,
		loopRunEventGoalStatusChanged,
		map[string]any{
			loopRunEventPayloadKeyNodeID:     string(current.state.NodeID),
			loopRunEventPayloadKeyItemIndex:  current.state.ItemIndex,
			loopRunEventPayloadKeyGeneration: current.state.Generation,
			loopRunEventPayloadKeyFrom:       current.state.GoalStatus,
			loopRunEventPayloadKeyTo:         goalStatusActive,
			loopRunEventPayloadKeyCause:      string(current.state.Cause),
			loopRunEventPayloadKeyActorKind:  string(req.Actor.Actor.Kind.Normalize()),
			loopRunEventPayloadKeyActorID:    strings.TrimSpace(req.Actor.Actor.Ref),
		},
		g.now(),
	)
}
