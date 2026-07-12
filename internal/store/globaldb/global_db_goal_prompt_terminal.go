package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

// RecordGoalDriverAttached advances one exact claimed operation to sent without allocating a turn.
func (g *GlobalDB) RecordGoalDriverAttached(ctx context.Context, req goal.DriverAttachmentRequest) error {
	if err := g.checkReady(ctx, "record Goal driver attached"); err != nil {
		return err
	}
	if err := validateGoalDriverAttachment(req); err != nil {
		return err
	}
	return g.withTaskImmediateTransaction(ctx, "record Goal driver attached", func(exec taskSQLExecutor) error {
		if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
			return err
		}
		checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
		if err != nil {
			return err
		}
		row, err := loadGoalPromptRow(ctx, exec, req.Key.LoopRunID, req.PromptID)
		if err != nil {
			return err
		}
		if err := validateGoalLivePromptOwner(
			ctx,
			exec,
			checkpoint,
			&row,
			req.Key,
			req.ExpectedControlEpoch,
			req.ExpectedBindingEpoch,
			req.TaskRunID,
			req.QueueEntryID,
			req.PromptID,
			req.SessionID,
			req.BindingHandle,
			false,
		); err != nil {
			return err
		}
		if row.promptID.String != strings.TrimSpace(req.DriverTurnID) ||
			!row.dispatchTokenHash.Valid || row.dispatchTokenHash.String != hashGoalDispatchToken(req.DispatchToken) {
			return goalControlStaleError("Goal driver attachment identity changed")
		}
		if row.status == store.SessionInputQueueStatusSent {
			if row.terminalAt != nil {
				return goalControlStaleError("Goal driver attachment arrived after terminal evidence")
			}
			return nil
		}
		result, err := exec.ExecContext(
			ctx,
			`UPDATE session_input_queue
			 SET status = 'sent', sent_at = ?, updated_at = ?
			 WHERE id = ? AND loop_run_id = ? AND task_run_id = ? AND prompt_id = ?
			   AND owner_kind = ? AND owner_epoch = ? AND binding_epoch = ?
			   AND session_id = ? AND status = 'dispatching'
			   AND dispatch_token_hash = ? AND terminal_at IS NULL`,
			store.FormatTimestamp(req.AttachedAt),
			store.FormatTimestamp(req.AttachedAt),
			strings.TrimSpace(req.QueueEntryID),
			string(req.Key.LoopRunID),
			strings.TrimSpace(req.TaskRunID),
			strings.TrimSpace(req.PromptID),
			goalPromptOwnerKind,
			req.ExpectedControlEpoch,
			req.ExpectedBindingEpoch,
			strings.TrimSpace(req.SessionID),
			row.dispatchTokenHash.String,
		)
		if err != nil {
			return fmt.Errorf("store: record Goal driver attached: %w", err)
		}
		return requireGoalRowsAffected(result, "record Goal driver attached")
	})
}

// FinalizeGoalPrompt records one live token-authenticated terminal without settling its Goal turn.
func (g *GlobalDB) FinalizeGoalPrompt(ctx context.Context, req goal.FinalizePromptRequest) error {
	if err := g.checkReady(ctx, "finalize Goal prompt"); err != nil {
		return err
	}
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.ExpectedBindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" || strings.TrimSpace(req.SessionID) == "" ||
		strings.TrimSpace(req.BindingHandle) == "" || strings.TrimSpace(req.DispatchToken) == "" ||
		req.TerminalAt.IsZero() {
		return fmt.Errorf("%w: Goal prompt finalization identity is incomplete", looppkg.ErrValidation)
	}
	if err := validateGoalPromptResult(req.Result); err != nil {
		return err
	}
	if strings.TrimSpace(req.Result.PromptID) != strings.TrimSpace(req.PromptID) {
		return fmt.Errorf("%w: Goal prompt result correlation changed", looppkg.ErrValidation)
	}
	return g.withTaskImmediateTransaction(ctx, "finalize Goal prompt", func(exec taskSQLExecutor) error {
		return finalizeGoalPromptWithExecutor(ctx, exec, req)
	})
}

func finalizeGoalPromptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.FinalizePromptRequest,
) error {
	if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
		return err
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return err
	}
	row, err := loadGoalPromptRow(ctx, exec, req.Key.LoopRunID, req.PromptID)
	if err != nil {
		return err
	}
	if err := validateGoalLivePromptOwner(
		ctx,
		exec,
		checkpoint,
		&row,
		req.Key,
		req.ExpectedControlEpoch,
		req.ExpectedBindingEpoch,
		req.TaskRunID,
		req.QueueEntryID,
		req.PromptID,
		req.SessionID,
		req.BindingHandle,
		true,
	); err != nil {
		return err
	}
	if !row.dispatchTokenHash.Valid ||
		row.dispatchTokenHash.String != hashGoalDispatchToken(req.DispatchToken) {
		return goalControlStaleError("Goal prompt finalization identity changed")
	}
	if row.terminalAt != nil {
		if goalPromptTerminalMatches(&row, req.Result) {
			return nil
		}
		return goalControlStaleError("Goal prompt already has a different terminal")
	}
	status := goalPromptTerminalQueueStatus(req.Result.Outcome)
	if err := persistGoalQueueTerminal(ctx, exec, req, &row, req.Result.FenceDisposition, status); err != nil {
		return err
	}
	phase := goalTerminalCheckpointPhase(checkpoint.PromptKind, req.Result)
	return advanceGoalTerminalCheckpoint(ctx, exec, req, checkpoint, phase)
}

func goalPromptTerminalQueueStatus(outcome looppkg.ActionPromptOutcome) string {
	switch outcome {
	case looppkg.ActionPromptOutcomeInvalidResult,
		looppkg.ActionPromptOutcomeFailed,
		looppkg.ActionPromptOutcomeAmbiguous:
		return store.SessionInputQueueStatusFailed
	default:
		return store.SessionInputQueueStatusSent
	}
}

func persistGoalQueueTerminal(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.FinalizePromptRequest,
	row *goalPromptRow,
	disposition looppkg.ActionDisposition,
	status string,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE session_input_queue
		 SET status = ?, terminal_event_start_seq = ?, terminal_event_end_seq = ?,
		     terminal_kind = ?, terminal_stop_reason = ?, terminal_disposition = ?,
		     terminal_reason_code = ?, terminal_tokens_reported = ?, terminal_tokens_used = ?,
		     terminal_at = ?, failed_at = CASE WHEN ? = 'failed' THEN ? ELSE failed_at END,
		     updated_at = ?
		 WHERE id = ? AND loop_run_id = ? AND task_run_id = ? AND prompt_id = ?
		   AND owner_kind = ? AND owner_epoch = ? AND binding_epoch = ? AND session_id = ?
		   AND status IN ('dispatching','sent') AND terminal_at IS NULL
		   AND dispatch_token_hash = ?`,
		status,
		nullableGoalPositiveInt64(req.Result.EventStartSeq),
		nullableGoalPositiveInt64(req.Result.EventEndSeq),
		string(req.Result.Outcome),
		nullableGoalString(string(req.Result.StopReason)),
		nullableGoalString(string(disposition)),
		nullableGoalString(string(req.Result.ReasonCode)),
		boolToInt(req.Result.TokensReported),
		goalReportedTokens(req.Result.TokensUsed, req.Result.TokensReported),
		store.FormatTimestamp(req.TerminalAt),
		status,
		store.FormatTimestamp(req.TerminalAt),
		store.FormatTimestamp(req.TerminalAt),
		strings.TrimSpace(req.QueueEntryID),
		string(req.Key.LoopRunID),
		strings.TrimSpace(req.TaskRunID),
		strings.TrimSpace(req.PromptID),
		goalPromptOwnerKind,
		req.ExpectedControlEpoch,
		req.ExpectedBindingEpoch,
		strings.TrimSpace(req.SessionID),
		row.dispatchTokenHash.String,
	)
	if err != nil {
		return fmt.Errorf("store: finalize Goal queue terminal: %w", err)
	}
	return requireGoalRowsAffected(result, "finalize Goal queue terminal")
}

func goalTerminalCheckpointPhase(promptKind string, result looppkg.ActionPromptResult) string {
	if promptKind == goalPromptKindCompact {
		return goalCheckpointPhaseCompacting
	}
	judgeable := result.Outcome == looppkg.ActionPromptOutcomeCompleted &&
		(result.StopReason == looppkg.ActionStopEndTurn ||
			result.StopReason == looppkg.ActionStopMaxTurnRequests)
	if judgeable {
		return goalCheckpointPhaseJudging
	}
	return goalCheckpointPhasePersisting
}

func advanceGoalTerminalCheckpoint(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.FinalizePromptRequest,
	checkpoint goal.Checkpoint,
	phase string,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints SET phase = ?, updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND binding_epoch = ? AND queue_entry_id = ? AND prompt_id = ?
		   AND phase IN ('prompting','compacting')`,
		phase,
		store.FormatTimestamp(req.TerminalAt),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		checkpoint.ControlEpoch,
		checkpoint.BindingEpoch,
		strings.TrimSpace(req.QueueEntryID),
		strings.TrimSpace(req.PromptID),
	)
	if err != nil {
		return fmt.Errorf("store: advance Goal terminal checkpoint: %w", err)
	}
	return requireGoalRowsAffected(result, "advance Goal terminal checkpoint")
}

func validateGoalDriverAttachment(req goal.DriverAttachmentRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.ExpectedBindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" || strings.TrimSpace(req.SessionID) == "" ||
		strings.TrimSpace(req.BindingHandle) == "" || strings.TrimSpace(req.DispatchToken) == "" ||
		strings.TrimSpace(req.DriverTurnID) != strings.TrimSpace(req.PromptID) || req.AttachedAt.IsZero() {
		return fmt.Errorf("%w: Goal driver attachment identity is incomplete", looppkg.ErrValidation)
	}
	return nil
}

func goalPromptTerminalMatches(row *goalPromptRow, result looppkg.ActionPromptResult) bool {
	return row.terminalKind.Valid && row.terminalKind.String == string(result.Outcome) &&
		goalOptionalPositiveInt64Matches(row.terminalEventStart, result.EventStartSeq) &&
		goalOptionalPositiveInt64Matches(row.terminalEventEnd, result.EventEndSeq) &&
		row.terminalStopReason.String == string(result.StopReason) &&
		row.terminalDisposition.String == string(result.FenceDisposition) &&
		row.terminalReason.String == string(result.ReasonCode) &&
		(row.terminalTokensReported != 0) == result.TokensReported &&
		(!result.TokensReported || (row.terminalTokensUsed.Valid && row.terminalTokensUsed.Int64 == result.TokensUsed))
}

func goalOptionalPositiveInt64Matches(stored sql.NullInt64, want int64) bool {
	if want < 1 {
		return !stored.Valid
	}
	return stored.Valid && stored.Int64 == want
}
