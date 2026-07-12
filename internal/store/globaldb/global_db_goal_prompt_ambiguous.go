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

// MarkAmbiguous finalizes one already-started Goal operation without replaying or allocating a successor.
func (g *GlobalDB) MarkAmbiguous(ctx context.Context, req goal.AmbiguousRequest) error {
	if err := g.checkReady(ctx, "mark Goal prompt ambiguous"); err != nil {
		return err
	}
	if err := validateGoalAmbiguousRequest(req); err != nil {
		return err
	}
	now := g.now().UTC()
	return g.withTaskImmediateTransaction(ctx, "mark Goal prompt ambiguous", func(exec taskSQLExecutor) error {
		return markGoalPromptAmbiguousWithExecutor(ctx, exec, req, now)
	})
}

func markGoalPromptAmbiguousWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.AmbiguousRequest,
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
		checkpoint.BindingEpoch != req.ExpectedBindingEpoch ||
		checkpoint.TaskRunID != strings.TrimSpace(req.TaskRunID) ||
		checkpoint.QueueEntryID != strings.TrimSpace(req.QueueEntryID) ||
		checkpoint.PromptID != strings.TrimSpace(req.PromptID) {
		return goalControlStaleError("ambiguous Goal operation owner changed")
	}
	if checkpoint.Phase == goalCheckpointPhaseAwaitingControl && checkpoint.ControlCause == req.Cause {
		return nil
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
	row, err := loadGoalPromptRow(ctx, exec, req.Key.LoopRunID, req.PromptID)
	if err != nil {
		return err
	}
	if err := requireGoalPromptOwner(
		&row, req.Key, req.TaskRunID, req.QueueEntryID, req.ExpectedBindingEpoch,
	); err != nil {
		return err
	}
	if err := ensureAmbiguousGoalQueueTerminal(ctx, exec, &row, req, now); err != nil {
		return err
	}
	if checkpoint.PromptKind != goalPromptKindCompact {
		if err := finalizeAmbiguousGoalTurn(ctx, exec, req, now); err != nil {
			return err
		}
		if err := appendGoalTurnCompletedEvent(ctx, exec, req.Key, req.PromptID, now); err != nil {
			return err
		}
	}
	if err := checkpointAmbiguousGoalPrompt(ctx, exec, req, now); err != nil {
		return err
	}
	if err := projectGoalCheckpointCounts(
		ctx, exec, req.Key, goalStatusPaused, checkpoint.TurnsUsed, checkpoint.TurnLimit,
	); err != nil {
		return err
	}
	return appendGoalStatusChangedEvent(
		ctx,
		exec,
		req.Key,
		checkpoint.Status,
		goalStatusPaused,
		req.Cause,
		"system",
		"goal-recovery",
		now,
	)
}

func ensureAmbiguousGoalQueueTerminal(
	ctx context.Context,
	exec taskSQLExecutor,
	row *goalPromptRow,
	req goal.AmbiguousRequest,
	now time.Time,
) error {
	if row.terminalAt != nil {
		if row.terminalKind.Valid && row.terminalKind.String == string(looppkg.ActionPromptOutcomeAmbiguous) &&
			row.terminalReason.Valid && row.terminalReason.String == string(req.Cause) {
			return nil
		}
		return goalControlStaleError("Goal operation already has a non-ambiguous terminal")
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE session_input_queue
		 SET status = 'failed', terminal_kind = 'ambiguous', terminal_reason_code = ?,
		     terminal_tokens_reported = 0, terminal_tokens_used = NULL,
		     terminal_at = ?, failed_at = ?, failure_summary = ?, updated_at = ?
		 WHERE id = ? AND loop_run_id = ? AND prompt_id = ?
		   AND status IN ('dispatching','sent') AND terminal_at IS NULL`,
		string(req.Cause),
		store.FormatTimestamp(now),
		store.FormatTimestamp(now),
		string(req.Cause),
		store.FormatTimestamp(now),
		strings.TrimSpace(req.QueueEntryID),
		string(req.Key.LoopRunID),
		strings.TrimSpace(req.PromptID),
	)
	if err != nil {
		return fmt.Errorf("store: mark Goal queue terminal ambiguous: %w", err)
	}
	return requireGoalRowsAffected(result, "mark Goal queue terminal ambiguous")
}

func finalizeAmbiguousGoalTurn(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.AmbiguousRequest,
	now time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_turns
		 SET result_status = 'ambiguous', reason_code = ?, ended_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND prompt_id = ? AND result_status IS NULL`,
		string(req.Cause),
		store.FormatTimestamp(now),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		strings.TrimSpace(req.PromptID),
	)
	if err != nil {
		return fmt.Errorf("store: finalize ambiguous Goal turn: %w", err)
	}
	return requireGoalRowsAffected(result, "finalize ambiguous Goal turn")
}

func checkpointAmbiguousGoalPrompt(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.AmbiguousRequest,
	now time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET phase = 'awaiting_control', goal_status = 'paused', control_cause = ?, updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND binding_epoch = ?
		   AND task_run_id = ? AND queue_entry_id = ? AND prompt_id = ?
		   AND phase IN ('prompting','compacting','persisting')`,
		string(req.Cause),
		store.FormatTimestamp(now),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		req.ExpectedControlEpoch,
		req.ExpectedBindingEpoch,
		strings.TrimSpace(req.TaskRunID),
		strings.TrimSpace(req.QueueEntryID),
		strings.TrimSpace(req.PromptID),
	)
	if err != nil {
		return fmt.Errorf("store: checkpoint ambiguous Goal operation: %w", err)
	}
	return requireGoalRowsAffected(result, "checkpoint ambiguous Goal operation")
}

func validateGoalAmbiguousRequest(req goal.AmbiguousRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.ExpectedBindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" ||
		(req.Cause != looppkg.ReasonCodeGoalRecoveryAmbiguous &&
			req.Cause != looppkg.ReasonCodeGoalControlRevokedInFlight) {
		return fmt.Errorf("%w: ambiguous Goal operation identity is invalid", looppkg.ErrValidation)
	}
	return nil
}
