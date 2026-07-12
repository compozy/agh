package globaldb

import (
	"context"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

func revokeGoalPromptEvidence(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	row *goalPromptRow,
	request goal.RevokePromptRequest,
) (bool, error) {
	if checkpoint.Phase == goalCheckpointPhaseQueued {
		return false, terminalizePreparedGoalRevocation(ctx, exec, row, request)
	}
	if row.terminalAt == nil {
		if err := terminalizeClaimedGoalRevocation(ctx, exec, row, request); err != nil {
			return false, err
		}
		var err error
		*row, err = loadGoalPromptRow(ctx, exec, request.Key.LoopRunID, request.PromptID)
		if err != nil {
			return false, err
		}
	}
	if checkpoint.PromptKind == goalPromptKindCompact {
		return false, nil
	}
	if err := terminalizeRevokedGoalTurn(ctx, exec, row, request); err != nil {
		return false, err
	}
	return true, nil
}

func terminalizePreparedGoalRevocation(
	ctx context.Context,
	exec taskSQLExecutor,
	row *goalPromptRow,
	request goal.RevokePromptRequest,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE session_input_queue
		 SET status = 'canceled', terminal_kind = 'control-fenced', terminal_disposition = ?,
		     terminal_reason_code = ?, terminal_tokens_reported = 0, terminal_tokens_used = NULL,
		     terminal_at = ?, canceled_at = ?, updated_at = ?
		 WHERE id = ? AND loop_run_id = ? AND task_run_id = ? AND prompt_id = ?
		   AND owner_kind = ? AND owner_epoch = ? AND binding_epoch = ?
		   AND status = 'queued' AND dispatchable = 0 AND terminal_at IS NULL`,
		string(request.Disposition),
		string(looppkg.ReasonCodeGoalControlRevokedInFlight),
		store.FormatTimestamp(request.RevokedAt),
		store.FormatTimestamp(request.RevokedAt),
		store.FormatTimestamp(request.RevokedAt),
		row.id,
		string(request.Key.LoopRunID),
		strings.TrimSpace(request.TaskRunID),
		strings.TrimSpace(request.PromptID),
		goalPromptOwnerKind,
		request.ExpectedControlEpoch,
		request.ExpectedBindingEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: revoke prepared Goal prompt: %w", err)
	}
	return requireGoalRowsAffected(result, "revoke prepared Goal prompt")
}

func terminalizeClaimedGoalRevocation(
	ctx context.Context,
	exec taskSQLExecutor,
	row *goalPromptRow,
	request goal.RevokePromptRequest,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE session_input_queue
		 SET status = 'failed', terminal_kind = 'ambiguous', terminal_reason_code = ?,
		     terminal_tokens_reported = 0, terminal_tokens_used = NULL,
		     terminal_at = ?, failed_at = ?, failure_summary = ?, updated_at = ?
		 WHERE id = ? AND loop_run_id = ? AND task_run_id = ? AND prompt_id = ?
		   AND owner_kind = ? AND owner_epoch = ? AND binding_epoch = ?
		   AND status IN ('dispatching','sent') AND terminal_at IS NULL`,
		string(looppkg.ReasonCodeGoalControlRevokedInFlight),
		store.FormatTimestamp(request.RevokedAt),
		store.FormatTimestamp(request.RevokedAt),
		string(looppkg.ReasonCodeGoalControlRevokedInFlight),
		store.FormatTimestamp(request.RevokedAt),
		row.id,
		string(request.Key.LoopRunID),
		strings.TrimSpace(request.TaskRunID),
		strings.TrimSpace(request.PromptID),
		goalPromptOwnerKind,
		request.ExpectedControlEpoch,
		request.ExpectedBindingEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: revoke claimed Goal prompt: %w", err)
	}
	return requireGoalRowsAffected(result, "revoke claimed Goal prompt")
}

func terminalizeRevokedGoalTurn(
	ctx context.Context,
	exec taskSQLExecutor,
	row *goalPromptRow,
	request goal.RevokePromptRequest,
) error {
	if !row.terminalKind.Valid || row.terminalAt == nil {
		return fmt.Errorf("%w: revoked Goal prompt has no terminal evidence", looppkg.ErrTransitionConflict)
	}
	if !revokedGoalTurnOutcomeValid(row.terminalKind.String) {
		return fmt.Errorf(
			"%w: revoked Goal prompt terminal %q cannot settle a turn",
			looppkg.ErrTransitionConflict,
			row.terminalKind.String,
		)
	}
	endedAt, err := parseGoalTimestampValue(row.terminalAt, "revoked prompt terminal_at")
	if err != nil {
		return err
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_turns
		 SET result_status = ?, stop_reason = ?, reason_code = ?, verdict_outcome = NULL,
		     blocking_json = '[]', evidence_ref = NULL, tokens_used = ?, ended_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND prompt_id = ? AND binding_epoch = ? AND result_status IS NULL`,
		row.terminalKind.String,
		nullableGoalString(row.terminalStopReason.String),
		nullableGoalString(row.terminalReason.String),
		goalReportedTokens(row.terminalTokensUsed.Int64, row.terminalTokensReported != 0),
		store.FormatTimestamp(endedAt),
		string(request.Key.LoopRunID),
		request.Key.Generation,
		string(request.Key.NodeID),
		request.Key.ItemIndex,
		strings.TrimSpace(request.PromptID),
		request.ExpectedBindingEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: terminalize revoked Goal turn: %w", err)
	}
	return requireGoalRowsAffected(result, "terminalize revoked Goal turn")
}

func revokedGoalTurnOutcomeValid(outcome string) bool {
	switch looppkg.ActionPromptOutcome(outcome) {
	case looppkg.ActionPromptOutcomeCompleted,
		looppkg.ActionPromptOutcomeInvalidResult,
		looppkg.ActionPromptOutcomeFailed,
		looppkg.ActionPromptOutcomeAmbiguous:
		return true
	default:
		return false
	}
}
