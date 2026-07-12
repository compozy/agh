package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

// PrepareGoalPrompt inserts one idempotent non-dispatchable Goal queue ticket.
func (g *GlobalDB) PrepareGoalPrompt(
	ctx context.Context,
	req goal.PreparePromptRequest,
) (goal.PromptTicket, error) {
	if err := g.checkReady(ctx, "prepare Goal prompt"); err != nil {
		return goal.PromptTicket{}, err
	}
	if err := validatePrepareGoalPromptRequest(req); err != nil {
		return goal.PromptTicket{}, err
	}
	var ticket goal.PromptTicket
	err := g.withTaskImmediateTransaction(ctx, "prepare Goal prompt", func(exec taskSQLExecutor) error {
		var err error
		ticket, err = prepareGoalPromptWithExecutor(ctx, exec, req)
		return err
	})
	if err != nil {
		return goal.PromptTicket{}, err
	}
	return ticket, nil
}

func prepareGoalPromptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.PreparePromptRequest,
) (goal.PromptTicket, error) {
	if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
		return goal.PromptTicket{}, err
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return goal.PromptTicket{}, err
	}
	if checkpoint.ControlEpoch != req.ExpectedControlEpoch || checkpoint.Status != goalStatusActive ||
		(checkpoint.Phase != goalCheckpointPhaseIdle && checkpoint.Phase != goalCheckpointPhasePreparing &&
			checkpoint.Phase != goalCheckpointPhaseQueued) {
		return goal.PromptTicket{}, goalControlStaleError("Goal prompt prepare checkpoint changed")
	}
	if !goalUsageSequenceMatches(checkpoint.UsageSequence, req.ContextUsageSequence) &&
		strings.TrimSpace(req.PromptKind) == goalPromptKindCompact {
		return goal.PromptTicket{}, goalControlStaleError("Goal compaction context usage changed")
	}
	if strings.TrimSpace(req.PromptKind) == goalPromptKindCompact &&
		checkpoint.CompactionBaselineUsed != nil &&
		!goalUsageSequenceMatches(checkpoint.CompactionBaselineUsed, req.ContextUsageUsed) {
		return goal.PromptTicket{}, goalControlStaleError("Goal compaction context baseline changed")
	}
	if err := validateGoalPromptReferences(ctx, exec, req); err != nil {
		return goal.PromptTicket{}, err
	}
	if err := validateActiveGoalPromptBinding(
		ctx,
		exec,
		req.Key,
		req.BindingHandle,
		req.BindingEpoch,
		req.SessionID,
	); err != nil {
		return goal.PromptTicket{}, err
	}
	existing, found, err := findGoalPromptRow(ctx, exec, req.Key.LoopRunID, req.PromptID)
	if err != nil {
		return goal.PromptTicket{}, err
	}
	if found {
		return existingPreparedGoalPromptTicket(&existing, req)
	}
	if checkpoint.Phase == goalCheckpointPhaseQueued {
		return goal.PromptTicket{}, goalControlStaleError(
			"Goal checkpoint already owns a different prepared prompt",
		)
	}
	preparedAt := req.PreparedAt.UTC()
	if err := insertPreparedGoalPrompt(ctx, exec, req, preparedAt); err != nil {
		return goal.PromptTicket{}, err
	}
	if err := checkpointPreparedGoalPrompt(ctx, exec, req, preparedAt); err != nil {
		return goal.PromptTicket{}, err
	}
	return goal.PromptTicket{
		QueueEntryID: strings.TrimSpace(req.QueueEntryID),
		PromptID:     strings.TrimSpace(req.PromptID),
	}, nil
}

func existingPreparedGoalPromptTicket(
	row *goalPromptRow,
	req goal.PreparePromptRequest,
) (goal.PromptTicket, error) {
	matches := row.status == store.SessionInputQueueStatusQueued &&
		row.terminalAt == nil &&
		row.dispatchable == 0 &&
		row.id == strings.TrimSpace(req.QueueEntryID) &&
		row.sessionID == strings.TrimSpace(req.SessionID) &&
		row.taskRunID == strings.TrimSpace(req.TaskRunID) &&
		row.text == req.Message &&
		row.bindingEpoch.Valid && row.bindingEpoch.Int64 == req.BindingEpoch &&
		row.ownerEpoch.Valid && row.ownerEpoch.Int64 == req.ExpectedControlEpoch &&
		row.promptKind.Valid && row.promptKind.String == req.PromptKind &&
		preparedGoalUsageBaseMatches(row.operationUsageBase, req) &&
		row.promptAttempt == req.PromptAttempt
	if !matches {
		return goal.PromptTicket{}, goalControlStaleError(
			"Goal prompt ID already has different prepared identity",
		)
	}
	return goal.PromptTicket{QueueEntryID: row.id, PromptID: row.promptID.String}, nil
}

func insertPreparedGoalPrompt(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.PreparePromptRequest,
	preparedAt time.Time,
) error {
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO session_input_queue (
			id, session_id, status, mode, text, session_generation,
			task_run_id, run_generation, attempt_count, enqueued_at, updated_at,
			loop_run_id, owner_kind, owner_epoch, binding_epoch, prompt_id,
			prompt_kind, prompt_attempt, operation_usage_base_tokens, dispatchable
		) VALUES (
			?, ?, 'queued', ?, ?, (SELECT input_generation FROM sessions WHERE id = ?),
			?, ?, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0
		)`,
		strings.TrimSpace(req.QueueEntryID),
		strings.TrimSpace(req.SessionID),
		store.SessionInputQueueModeQueue,
		req.Message,
		strings.TrimSpace(req.SessionID),
		strings.TrimSpace(req.TaskRunID),
		req.Key.Generation,
		store.FormatTimestamp(preparedAt),
		store.FormatTimestamp(preparedAt),
		string(req.Key.LoopRunID),
		goalPromptOwnerKind,
		req.ExpectedControlEpoch,
		req.BindingEpoch,
		strings.TrimSpace(req.PromptID),
		strings.TrimSpace(req.PromptKind),
		req.PromptAttempt,
		goalReportedTokens(req.UsageBaseTokens, req.UsageBaseReported),
	)
	if err != nil {
		return fmt.Errorf("store: insert prepared Goal prompt: %w", err)
	}
	return nil
}

func checkpointPreparedGoalPrompt(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.PreparePromptRequest,
	preparedAt time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET phase = 'queued', task_run_id = ?, queue_entry_id = ?, prompt_id = ?,
		     prompt_kind = ?, prompt_attempt = ?, session_id = ?, binding_handle = ?,
		     binding_epoch = ?, compaction_baseline_used = ?, updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND goal_status = 'active'
		   AND phase IN ('idle','preparing')`,
		strings.TrimSpace(req.TaskRunID),
		strings.TrimSpace(req.QueueEntryID),
		strings.TrimSpace(req.PromptID),
		strings.TrimSpace(req.PromptKind),
		req.PromptAttempt,
		strings.TrimSpace(req.SessionID),
		strings.TrimSpace(req.BindingHandle),
		req.BindingEpoch,
		nullableGoalInt64Value(req.ContextUsageUsed),
		store.FormatTimestamp(preparedAt),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		req.ExpectedControlEpoch,
	)
	if err != nil {
		return fmt.Errorf("store: checkpoint prepared Goal prompt: %w", err)
	}
	return requireGoalRowsAffected(result, "checkpoint prepared Goal prompt")
}

func validatePrepareGoalPromptRequest(req goal.PreparePromptRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	kind := strings.TrimSpace(req.PromptKind)
	if req.ExpectedControlEpoch < 1 || strings.TrimSpace(req.TaskRunID) == "" ||
		strings.TrimSpace(req.QueueEntryID) == "" || strings.TrimSpace(req.PromptID) == "" ||
		(kind != goalPromptKindWork && kind != goalPromptKindContinuation && kind != goalPromptKindCompact) ||
		req.PromptAttempt < 0 || strings.TrimSpace(req.SessionID) == "" ||
		strings.TrimSpace(req.BindingHandle) == "" || req.BindingEpoch < 1 ||
		req.UsageBaseTokens < 0 || (!req.UsageBaseReported && req.UsageBaseTokens != 0) ||
		strings.TrimSpace(req.Message) == "" || req.PreparedAt.IsZero() {
		return fmt.Errorf("%w: prepared Goal prompt identity is incomplete", looppkg.ErrValidation)
	}
	return validatePreparedGoalContextBaseline(kind, req.ContextUsageSequence, req.ContextUsageUsed)
}

func validatePreparedGoalContextBaseline(kind string, sequence *int64, used *int64) error {
	if (sequence == nil) != (used == nil) ||
		(sequence != nil && (*sequence < 0 || *used < 0)) ||
		(kind != goalPromptKindCompact && sequence != nil) {
		return fmt.Errorf("%w: prepared Goal context baseline is incomplete", looppkg.ErrValidation)
	}
	return nil
}

func preparedGoalUsageBaseMatches(base sql.NullInt64, req goal.PreparePromptRequest) bool {
	if !req.UsageBaseReported {
		return !base.Valid
	}
	return base.Valid && base.Int64 == req.UsageBaseTokens
}

func validateGoalPromptReferences(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.PreparePromptRequest,
) error {
	var taskExists int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT 1 FROM task_runs WHERE id = ? AND loop_run_id = ?`,
		strings.TrimSpace(req.TaskRunID),
		string(req.Key.LoopRunID),
	).Scan(&taskExists); err != nil {
		return fmt.Errorf("store: validate Goal prompt task owner: %w", err)
	}
	var sessionExists int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT 1 FROM sessions WHERE id = ? AND workspace_id = ?`,
		strings.TrimSpace(req.SessionID),
		string(req.Key.WorkspaceID),
	).Scan(&sessionExists); err != nil {
		return fmt.Errorf("store: validate Goal prompt session owner: %w", err)
	}
	return nil
}
