package globaldb

import (
	"context"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

// GrantAndReactivate records one monotonic typed grant and advances the control epoch exactly once.
func (g *GlobalDB) GrantAndReactivate(ctx context.Context, req goal.GrantRequest) error {
	if err := g.checkReady(ctx, "grant and reactivate goal checkpoint"); err != nil {
		return err
	}
	if err := validateGoalGrantRequest(req); err != nil {
		return err
	}
	now := g.now()
	return g.withTaskImmediateTransaction(
		ctx,
		"grant and reactivate goal checkpoint",
		func(exec taskSQLExecutor) error {
			return grantAndReactivateWithExecutor(ctx, exec, req, now)
		},
	)
}

func grantAndReactivateWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.GrantRequest,
	now time.Time,
) error {
	if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
		return err
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return err
	}
	if checkpoint.ControlEpoch != req.ExpectedControlEpoch ||
		checkpoint.Phase != goalCheckpointPhaseAwaitingControl {
		return goalControlStaleError("grant checkpoint epoch or phase changed")
	}
	grantID := int64(1)
	if checkpoint.ControlGrant != nil {
		grantID = checkpoint.ControlGrant.ID + 1
	}
	consumed := req.Kind == goal.ControlGrantTurnExtension || req.Kind == goal.ControlGrantPlainResume
	turnLimit := checkpoint.TurnLimit
	if req.Kind == goal.ControlGrantTurnExtension {
		turnLimit += req.TurnIncrement
	}
	if err := persistGoalGrantCheckpoint(ctx, exec, req, grantID, consumed, turnLimit, now); err != nil {
		return err
	}
	if err := incrementGoalBudgetVersion(ctx, exec, req.Key.LoopRunID); err != nil {
		return err
	}
	if err := projectGoalGrantOutput(ctx, exec, req, checkpoint, turnLimit); err != nil {
		return err
	}
	return appendGoalStatusChangedEvent(
		ctx,
		exec,
		req.Key,
		checkpoint.Status,
		goalStatusActive,
		req.Cause,
		req.ActorKind,
		req.ActorID,
		now,
	)
}

func persistGoalGrantCheckpoint(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.GrantRequest,
	grantID int64,
	consumed bool,
	turnLimit int,
	now time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
			 SET control_epoch = control_epoch + 1,
			     phase = 'idle', goal_status = 'active', control_cause = NULL, turn_limit = ?,
			     control_actor_kind = NULL, control_actor_id = NULL, control_requested_at = NULL,
			     control_grant_id = ?, control_grant_kind = ?, control_grant_cause = ?,
			     control_grant_turn = ?, control_grant_scope = ?, control_grant_consumed = ?,
			     updated_at = ?
			 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
			   AND control_epoch = ? AND phase = 'awaiting_control'`,
		turnLimit,
		grantID,
		string(req.Kind),
		string(req.Cause),
		req.Turn,
		string(req.Scope),
		boolToInt(consumed),
		store.FormatTimestamp(now),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		req.ExpectedControlEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: grant goal checkpoint control: %w", err)
	}
	return requireGoalRowsAffected(result, "grant goal checkpoint control")
}

func projectGoalGrantOutput(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.GrantRequest,
	checkpoint goal.Checkpoint,
	turnLimit int,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_generation_outputs
			 SET goal_status = 'active', goal_turns_used = ?, goal_turn_limit = ?
			 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?`,
		checkpoint.TurnsUsed,
		turnLimit,
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
	)
	if err != nil {
		return fmt.Errorf("store: project goal grant reactivation: %w", err)
	}
	return requireGoalRowsAffected(result, "project goal grant reactivation")
}

// CheckpointControl records one typed nonterminal or terminal control boundary by CAS.
func (g *GlobalDB) CheckpointControl(ctx context.Context, req goal.ControlCheckpointRequest) error {
	if err := g.checkReady(ctx, "checkpoint goal control"); err != nil {
		return err
	}
	if err := validateGoalControlCheckpointRequest(req); err != nil {
		return err
	}
	now := g.now()
	return g.withTaskImmediateTransaction(ctx, "checkpoint goal control", func(exec taskSQLExecutor) error {
		if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
			return err
		}
		checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
		if err != nil {
			return err
		}
		if !goalControlCheckpointOwnerMatches(checkpoint, req) {
			return goalControlStaleError("control checkpoint owner changed")
		}
		phase := goalControlCheckpointTargetPhase(req)
		result, err := exec.ExecContext(
			ctx,
			`UPDATE loop_goal_checkpoints
			 SET phase = ?, goal_status = ?, control_cause = ?, control_actor_kind = ?, control_actor_id = ?,
			     control_requested_at = ?, updated_at = ?
			 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
			   AND control_epoch = ? AND COALESCE(binding_epoch, 0) = ? AND phase = ?
			   AND COALESCE(task_run_id, '') = ? AND COALESCE(queue_entry_id, '') = ?
			   AND COALESCE(prompt_id, '') = ?`,
			phase,
			req.Status,
			string(req.Cause),
			strings.TrimSpace(req.ActorKind),
			strings.TrimSpace(req.ActorID),
			store.FormatTimestamp(now),
			store.FormatTimestamp(now),
			string(req.Key.LoopRunID),
			req.Key.Generation,
			string(req.Key.NodeID),
			req.Key.ItemIndex,
			req.ExpectedControlEpoch,
			req.ExpectedBindingEpoch,
			strings.TrimSpace(req.ExpectedPhase),
			strings.TrimSpace(req.TaskRunID),
			strings.TrimSpace(req.QueueEntryID),
			strings.TrimSpace(req.PromptID),
		)
		if err != nil {
			return fmt.Errorf("store: checkpoint goal control: %w", err)
		}
		if err := requireGoalRowsAffected(result, "checkpoint goal control"); err != nil {
			return err
		}
		if phase == goalCheckpointPhaseTerminal {
			if err := closeTerminalGoalCheckpointBinding(ctx, exec, checkpoint, now); err != nil {
				return err
			}
		}
		return appendGoalStatusChangedEvent(
			ctx,
			exec,
			req.Key,
			checkpoint.Status,
			req.Status,
			req.Cause,
			req.ActorKind,
			req.ActorID,
			now,
		)
	})
}

func validateGoalGrantRequest(req goal.GrantRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.Kind == "" || req.Cause == "" || req.Scope == "" ||
		req.Turn < 0 || strings.TrimSpace(req.ActorKind) == "" || strings.TrimSpace(req.ActorID) == "" {
		return fmt.Errorf("%w: goal grant identity is incomplete", looppkg.ErrValidation)
	}
	if req.Kind == goal.ControlGrantTurnExtension && req.TurnIncrement < 1 {
		return fmt.Errorf("%w: turn extension must increase the limit", looppkg.ErrValidation)
	}
	if req.Kind != goal.ControlGrantTurnExtension && req.TurnIncrement != 0 {
		return fmt.Errorf("%w: only turn-extension may change the turn limit", looppkg.ErrValidation)
	}
	if !goalGrantKindScopeValid(req.Kind, req.Scope) {
		return fmt.Errorf(
			"%w: goal grant kind %q cannot use scope %q",
			looppkg.ErrValidation,
			req.Kind,
			req.Scope,
		)
	}
	return nil
}

func goalGrantKindScopeValid(kind goal.ControlGrantKind, scope goal.ControlGrantScope) bool {
	switch kind {
	case goal.ControlGrantTurnExtension:
		return scope == goal.ControlGrantScopeTurnLimit
	case goal.ControlGrantBudget:
		return scope == goal.ControlGrantScopeSettleCurrent ||
			scope == goal.ControlGrantScopeWorkAndSettle
	case goal.ControlGrantReseed:
		return scope == goal.ControlGrantScopeRotateBinding
	case goal.ControlGrantPlainResume:
		return scope == goal.ControlGrantScopeReactivate
	default:
		return false
	}
}

func incrementGoalBudgetVersion(ctx context.Context, exec taskSQLExecutor, runID looppkg.RunID) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs SET budget_version = budget_version + 1 WHERE id = ?`,
		string(runID),
	)
	if err != nil {
		return fmt.Errorf("store: increment goal budget version: %w", err)
	}
	return requireGoalRowsAffected(result, "increment goal budget version")
}
