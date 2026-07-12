package globaldb

import (
	"context"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

var _ goal.PromptTerminalRecoveryStore = (*GlobalDB)(nil)

// RecoverGoalPromptTerminal records exact session-event proof without exposing the raw dispatch token.
func (g *GlobalDB) RecoverGoalPromptTerminal(
	ctx context.Context,
	req goal.RecoverPromptTerminalRequest,
) error {
	if err := g.checkReady(ctx, "recover Goal prompt terminal"); err != nil {
		return err
	}
	if err := validateRecoveredGoalPromptRequest(req); err != nil {
		return err
	}
	return g.withTaskImmediateTransaction(ctx, "recover Goal prompt terminal", func(exec taskSQLExecutor) error {
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
		if row.terminalAt != nil {
			if goalPromptTerminalMatches(&row, req.Result) {
				return nil
			}
			return goalControlStaleError("recovered Goal prompt already has a different terminal")
		}
		if !row.dispatchTokenHash.Valid {
			return goalControlStaleError("recovered Goal prompt has no committed dispatch token digest")
		}
		finalize := goal.FinalizePromptRequest{
			Key: req.Key, ExpectedControlEpoch: req.ExpectedControlEpoch,
			ExpectedBindingEpoch: req.ExpectedBindingEpoch, TaskRunID: req.TaskRunID,
			QueueEntryID: req.QueueEntryID, PromptID: req.PromptID, SessionID: req.SessionID,
			BindingHandle: req.BindingHandle, Result: req.Result, TerminalAt: req.TerminalAt,
		}
		status := goalPromptTerminalQueueStatus(req.Result.Outcome)
		if err := persistRecoveredGoalQueueTerminal(
			ctx,
			exec,
			finalize,
			&row,
			req.Result.FenceDisposition,
			status,
		); err != nil {
			return err
		}
		phase := goalTerminalCheckpointPhase(checkpoint.PromptKind, req.Result)
		return advanceGoalTerminalCheckpoint(ctx, exec, finalize, checkpoint, phase)
	})
}

func persistRecoveredGoalQueueTerminal(
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
		return fmt.Errorf("store: recover Goal queue terminal: %w", err)
	}
	return requireGoalRowsAffected(result, "recover Goal queue terminal")
}

func validateRecoveredGoalPromptRequest(req goal.RecoverPromptTerminalRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.ExpectedBindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" || strings.TrimSpace(req.SessionID) == "" ||
		strings.TrimSpace(req.BindingHandle) == "" || req.TerminalAt.IsZero() {
		return fmt.Errorf("%w: recovered Goal prompt identity is incomplete", looppkg.ErrValidation)
	}
	if err := validateGoalPromptResult(req.Result); err != nil {
		return err
	}
	if strings.TrimSpace(req.Result.PromptID) != strings.TrimSpace(req.PromptID) {
		return fmt.Errorf("%w: recovered Goal prompt correlation changed", looppkg.ErrValidation)
	}
	return nil
}
