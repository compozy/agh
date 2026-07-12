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

var _ goal.PreparedPromptRejectionStore = (*GlobalDB)(nil)

// RejectPreparedGoalPrompt records a proven no-effect attempt and advances its prompt identity.
func (g *GlobalDB) RejectPreparedGoalPrompt(
	ctx context.Context,
	req goal.RejectPreparedPromptRequest,
) error {
	if err := g.checkReady(ctx, "reject prepared Goal prompt"); err != nil {
		return err
	}
	if err := validateRejectedGoalPromptRequest(req); err != nil {
		return err
	}
	now := g.now().UTC()
	return g.withTaskImmediateTransaction(ctx, "reject prepared Goal prompt", func(exec taskSQLExecutor) error {
		checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
		if err != nil {
			return err
		}
		row, err := loadGoalPromptRow(ctx, exec, req.Key.LoopRunID, req.PromptID)
		if err != nil {
			return err
		}
		if rejectedGoalPromptAlreadyApplied(checkpoint, &row, req) {
			return nil
		}
		if checkpoint.ControlEpoch != req.ExpectedControlEpoch ||
			checkpoint.BindingEpoch != req.ExpectedBindingEpoch || checkpoint.Phase != goalCheckpointPhaseQueued ||
			checkpoint.TaskRunID != strings.TrimSpace(req.TaskRunID) ||
			checkpoint.QueueEntryID != strings.TrimSpace(req.QueueEntryID) ||
			checkpoint.PromptID != strings.TrimSpace(req.PromptID) {
			return goalControlStaleError("rejected Goal prompt checkpoint owner changed")
		}
		if err := requireGoalPromptOwner(
			&row,
			req.Key,
			req.TaskRunID,
			req.QueueEntryID,
			req.ExpectedBindingEpoch,
		); err != nil {
			return err
		}
		if row.status != store.SessionInputQueueStatusQueued || row.terminalAt != nil ||
			row.dispatchTokenHash.Valid {
			return goalControlStaleError("rejected Goal prompt already crossed submission")
		}
		if err := persistRejectedGoalQueueRow(ctx, exec, req, now); err != nil {
			return err
		}
		return advanceRejectedGoalCheckpoint(ctx, exec, req, checkpoint.PromptAttempt+1, now)
	})
}

func persistRejectedGoalQueueRow(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.RejectPreparedPromptRequest,
	now time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE session_input_queue
		 SET status = 'failed', dispatchable = 0, terminal_kind = 'rejected-before-submit',
		     terminal_reason_code = ?, terminal_tokens_reported = 0, terminal_tokens_used = NULL,
		     terminal_at = ?, failed_at = ?, failure_summary = ?, updated_at = ?
		 WHERE id = ? AND loop_run_id = ? AND prompt_id = ? AND task_run_id = ?
		   AND owner_kind = 'goal' AND owner_epoch = ? AND binding_epoch = ?
		   AND status = 'queued' AND terminal_at IS NULL AND dispatch_token_hash IS NULL`,
		string(req.ReasonCode),
		store.FormatTimestamp(now),
		store.FormatTimestamp(now),
		string(req.ReasonCode),
		store.FormatTimestamp(now),
		strings.TrimSpace(req.QueueEntryID),
		string(req.Key.LoopRunID),
		strings.TrimSpace(req.PromptID),
		strings.TrimSpace(req.TaskRunID),
		req.ExpectedControlEpoch,
		req.ExpectedBindingEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: reject prepared Goal queue row: %w", err)
	}
	return requireGoalRowsAffected(result, "reject prepared Goal queue row")
}

func advanceRejectedGoalCheckpoint(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.RejectPreparedPromptRequest,
	nextAttempt int,
	now time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET phase = 'preparing', prompt_attempt = ?, queue_entry_id = NULL, prompt_id = NULL,
		     prompt_kind = NULL, updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND binding_epoch = ? AND phase = 'queued'
		   AND task_run_id = ? AND queue_entry_id = ? AND prompt_id = ?`,
		nextAttempt,
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
		return fmt.Errorf("store: advance rejected Goal checkpoint: %w", err)
	}
	return requireGoalRowsAffected(result, "advance rejected Goal checkpoint")
}

func rejectedGoalPromptAlreadyApplied(
	checkpoint goal.Checkpoint,
	row *goalPromptRow,
	req goal.RejectPreparedPromptRequest,
) bool {
	return checkpoint.ControlEpoch == req.ExpectedControlEpoch &&
		checkpoint.BindingEpoch == req.ExpectedBindingEpoch && checkpoint.Phase == goalCheckpointPhasePreparing &&
		checkpoint.PromptAttempt == row.promptAttempt+1 && checkpoint.QueueEntryID == "" && checkpoint.PromptID == "" &&
		row.terminalAt != nil && row.terminalKind.Valid &&
		row.terminalKind.String == string(looppkg.ActionPromptOutcomeRejectedBeforeSubmit) &&
		row.terminalReason.Valid && row.terminalReason.String == string(req.ReasonCode)
}

func validateRejectedGoalPromptRequest(req goal.RejectPreparedPromptRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.ExpectedBindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" || strings.TrimSpace(string(req.ReasonCode)) == "" {
		return fmt.Errorf("%w: rejected Goal prompt identity is incomplete", looppkg.ErrValidation)
	}
	return nil
}
