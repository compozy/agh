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

// FencePreparedPrompt persists one pre-submit budget/control boundary without allocating a turn.
func (g *GlobalDB) FencePreparedPrompt(ctx context.Context, req goal.FencePreparedPromptRequest) error {
	if err := g.checkReady(ctx, "fence prepared Goal prompt"); err != nil {
		return err
	}
	if err := validateFencePreparedPromptRequest(req); err != nil {
		return err
	}
	now := g.now().UTC()
	return g.withTaskImmediateTransaction(ctx, "fence prepared Goal prompt", func(exec taskSQLExecutor) error {
		return fencePreparedGoalPromptWithExecutor(ctx, exec, req, now)
	})
}

func fencePreparedGoalPromptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.FencePreparedPromptRequest,
	now time.Time,
) error {
	if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
		return err
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return err
	}
	if err := validatePreparedGoalFenceCheckpoint(checkpoint, req); err != nil {
		return err
	}
	if err := validateActiveGoalPromptBinding(
		ctx,
		exec,
		req.Key,
		checkpoint.BindingHandle,
		req.ExpectedBindingEpoch,
		checkpoint.SessionID,
	); err != nil {
		return err
	}
	if err := validatePreparedGoalBudgetFence(ctx, exec, req); err != nil {
		return err
	}
	row, err := loadGoalPromptRow(ctx, exec, req.Key.LoopRunID, req.PromptID)
	if err != nil {
		return err
	}
	if err := requireGoalPromptOwner(
		&row, req.Key, req.TaskRunID, req.QueueEntryID, req.ExpectedBindingEpoch,
	); err != nil {
		return err
	}
	if existing, err := existingPreparedGoalFence(&row, req); existing || err != nil {
		return err
	}
	grantable := req.Disposition == looppkg.ActionDispositionNeedsApproval ||
		req.Disposition == looppkg.ActionDispositionPaused
	if err := persistPreparedGoalQueueFence(ctx, exec, req, grantable, now); err != nil {
		return err
	}
	goalStatus, phase := preparedGoalFenceProjection(req)
	if err := persistPreparedGoalCheckpointFence(ctx, exec, req, goalStatus, phase, now); err != nil {
		return err
	}
	if err := projectGoalCheckpointCounts(
		ctx, exec, req.Key, goalStatus, checkpoint.TurnsUsed, checkpoint.TurnLimit,
	); err != nil {
		return err
	}
	return appendGoalStatusChangedEvent(
		ctx, exec, req.Key, checkpoint.Status, goalStatus, req.Cause, "system", "goal-budget", now,
	)
}

func validatePreparedGoalFenceCheckpoint(
	checkpoint goal.Checkpoint,
	req goal.FencePreparedPromptRequest,
) error {
	matches := checkpoint.ControlEpoch == req.ExpectedControlEpoch &&
		checkpoint.BindingEpoch == req.ExpectedBindingEpoch &&
		checkpoint.Phase == goalCheckpointPhaseQueued &&
		checkpoint.TaskRunID == strings.TrimSpace(req.TaskRunID) &&
		checkpoint.QueueEntryID == strings.TrimSpace(req.QueueEntryID) &&
		checkpoint.PromptID == strings.TrimSpace(req.PromptID)
	if !matches {
		return goalControlStaleError("prepared Goal fence owner changed")
	}
	return nil
}

func validatePreparedGoalBudgetFence(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.FencePreparedPromptRequest,
) error {
	if req.Outcome != looppkg.ActionPromptOutcomeBudgetFenced {
		return nil
	}
	if req.BudgetDecision.Cause != req.Cause ||
		req.BudgetDecision.Disposition != req.Disposition {
		return goalPromptFencedError("denied Goal budget decision differs from prepared fence")
	}
	return validateDeniedGoalBudgetDecision(ctx, exec, req.Key.LoopRunID, req.BudgetDecision)
}

func existingPreparedGoalFence(
	row *goalPromptRow,
	req goal.FencePreparedPromptRequest,
) (bool, error) {
	if !row.fenceKind.Valid {
		return false, nil
	}
	matches := row.fenceKind.String == string(req.Outcome) &&
		row.fenceDisposition.String == string(req.Disposition) &&
		row.fenceReason.String == string(req.Cause)
	if !matches {
		return true, goalControlStaleError("prepared Goal prompt already has a different fence")
	}
	return true, nil
}

func persistPreparedGoalQueueFence(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.FencePreparedPromptRequest,
	grantable bool,
	now time.Time,
) error {
	status := store.SessionInputQueueStatusQueued
	if !grantable {
		status = store.SessionInputQueueStatusCanceled
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE session_input_queue
		 SET status = ?, dispatchable = 0, fence_kind = ?, fence_disposition = ?,
		     fence_reason_code = ?, fenced_at = ?,
		     terminal_kind = CASE WHEN ? = 1 THEN NULL ELSE ? END,
		     terminal_disposition = CASE WHEN ? = 1 THEN NULL ELSE ? END,
		     terminal_reason_code = CASE WHEN ? = 1 THEN NULL ELSE ? END,
		     terminal_at = CASE WHEN ? = 1 THEN NULL ELSE ? END,
		     canceled_at = CASE WHEN ? = 1 THEN canceled_at ELSE ? END,
		     updated_at = ?
		 WHERE id = ? AND loop_run_id = ? AND prompt_id = ?
		   AND status = 'queued' AND dispatchable = 0 AND terminal_at IS NULL
		   AND owner_epoch = ?`,
		status,
		string(req.Outcome),
		string(req.Disposition),
		string(req.Cause),
		store.FormatTimestamp(now),
		boolToInt(grantable),
		string(req.Outcome),
		boolToInt(grantable),
		string(req.Disposition),
		boolToInt(grantable),
		string(req.Cause),
		boolToInt(grantable),
		store.FormatTimestamp(now),
		boolToInt(grantable),
		store.FormatTimestamp(now),
		store.FormatTimestamp(now),
		strings.TrimSpace(req.QueueEntryID),
		string(req.Key.LoopRunID),
		strings.TrimSpace(req.PromptID),
		req.ExpectedControlEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: fence prepared Goal queue row: %w", err)
	}
	return requireGoalRowsAffected(result, "fence prepared Goal queue row")
}

func preparedGoalFenceProjection(req goal.FencePreparedPromptRequest) (string, string) {
	status := goalStatusPaused
	if req.Outcome == looppkg.ActionPromptOutcomeBudgetFenced {
		status = goalStatusBudgetLimited
	}
	phase := goalCheckpointPhaseAwaitingControl
	if req.Disposition == looppkg.ActionDispositionExhausted {
		phase = goalCheckpointPhaseTerminal
	}
	return status, phase
}

func persistPreparedGoalCheckpointFence(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.FencePreparedPromptRequest,
	status string,
	phase string,
	now time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET phase = ?, goal_status = ?, control_cause = ?, updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND binding_epoch = ? AND phase = 'queued'
		   AND queue_entry_id = ? AND prompt_id = ?`,
		phase,
		status,
		string(req.Cause),
		store.FormatTimestamp(now),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		req.ExpectedControlEpoch,
		req.ExpectedBindingEpoch,
		strings.TrimSpace(req.QueueEntryID),
		strings.TrimSpace(req.PromptID),
	)
	if err != nil {
		return fmt.Errorf("store: checkpoint prepared Goal fence: %w", err)
	}
	return requireGoalRowsAffected(result, "checkpoint prepared Goal fence")
}

func validateFencePreparedPromptRequest(req goal.FencePreparedPromptRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	validDisposition := req.Disposition == looppkg.ActionDispositionPaused ||
		req.Disposition == looppkg.ActionDispositionNeedsApproval ||
		req.Disposition == looppkg.ActionDispositionExhausted
	if req.ExpectedControlEpoch < 1 || req.ExpectedBindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" ||
		(req.Outcome != looppkg.ActionPromptOutcomeBudgetFenced &&
			req.Outcome != looppkg.ActionPromptOutcomeControlFenced) ||
		!validDisposition || req.Cause == "" {
		return fmt.Errorf("%w: prepared Goal fence identity is invalid", looppkg.ErrValidation)
	}
	return nil
}

func validateDeniedGoalBudgetDecision(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
	decision goal.BudgetDecision,
) error {
	if decision.Allowed || decision.BudgetVersion < 1 || decision.ValidUntil.IsZero() {
		return goalPromptFencedError("denied Goal budget decision is invalid")
	}
	var currentVersion int64
	if err := exec.QueryRowContext(
		ctx,
		`SELECT budget_version FROM loop_runs WHERE id = ?`,
		string(runID),
	).Scan(&currentVersion); err != nil {
		return fmt.Errorf("store: load Goal budget version for fence: %w", err)
	}
	if currentVersion != decision.BudgetVersion {
		return goalPromptFencedError("denied Goal budget decision is stale")
	}
	return nil
}
