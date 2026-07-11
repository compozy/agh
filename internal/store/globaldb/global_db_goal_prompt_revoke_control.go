package globaldb

import (
	"context"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

func advanceRevokedGoalCheckpoint(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	request goal.RevokePromptRequest,
) error {
	taskRunID := strings.TrimSpace(request.TaskRunID)
	queueEntryID := strings.TrimSpace(request.QueueEntryID)
	promptID := strings.TrimSpace(request.PromptID)
	if checkpoint.JudgeAttemptID != "" {
		if _, err := exec.ExecContext(
			ctx,
			`UPDATE loop_goal_judge_attempts
			 SET status = 'ambiguous', completed_at = ?
			 WHERE attempt_id = ? AND loop_run_id = ? AND status = 'running'`,
			store.FormatTimestamp(request.RevokedAt),
			checkpoint.JudgeAttemptID,
			string(request.Key.LoopRunID),
		); err != nil {
			return fmt.Errorf("store: revoke running Goal judge attempt: %w", err)
		}
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET control_epoch = control_epoch + 1, phase = 'terminal', goal_status = ?, control_cause = ?,
		     control_actor_kind = ?, control_actor_id = ?, control_requested_at = ?,
		     queue_entry_id = NULL, prompt_id = NULL, prompt_kind = NULL, prompt_attempt = 0,
		     judge_attempt_id = NULL,
		     report_prompt_id = NULL, report_status = NULL, report_evidence_ref = NULL,
		     report_binding_epoch = NULL, report_actor_kind = NULL, report_actor_id = NULL,
		     report_recorded_at = NULL, compaction_cancel_prompt_id = NULL,
		     compaction_cancel_cause = NULL, compaction_cancel_requested_at = NULL,
		     updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND binding_epoch = ? AND task_run_id = ?
		   AND queue_entry_id = ? AND prompt_id = ?
		   AND phase IN ('queued','prompting','compacting','judging','persisting')`,
		request.Status,
		string(request.Cause),
		request.ActorKind,
		request.ActorID,
		store.FormatTimestamp(request.RevokedAt),
		store.FormatTimestamp(request.RevokedAt),
		string(request.Key.LoopRunID),
		request.Key.Generation,
		string(request.Key.NodeID),
		request.Key.ItemIndex,
		request.ExpectedControlEpoch,
		request.ExpectedBindingEpoch,
		taskRunID,
		queueEntryID,
		promptID,
	)
	if err != nil {
		return fmt.Errorf("store: advance revoked Goal checkpoint: %w", err)
	}
	return requireGoalRowsAffected(result, "advance revoked Goal checkpoint")
}

func markGoalRunCleared(
	ctx context.Context,
	exec taskSQLExecutor,
	request goal.RevokePromptRequest,
) error {
	if request.ProjectionCause != goal.SessionOutboxCauseClear {
		return nil
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs SET goal_cleared_at = ? WHERE id = ? AND workspace_id = ?`,
		store.FormatTimestamp(request.RevokedAt),
		string(request.Key.LoopRunID),
		string(request.Key.WorkspaceID),
	)
	if err != nil {
		return fmt.Errorf("store: mark Goal run cleared: %w", err)
	}
	return requireGoalRowsAffected(result, "mark Goal run cleared")
}

func enqueueRevokedGoalProjection(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	request goal.RevokePromptRequest,
) error {
	var boundSessionID *string
	if request.ProjectionCause != goal.SessionOutboxCauseClear {
		value := checkpoint.SessionID
		boundSessionID = &value
	}
	return enqueueGoalProjectionOutboxIfSessionOrigin(
		ctx,
		exec,
		request.Key.WorkspaceID,
		request.Key.LoopRunID,
		boundSessionID,
		request.ProjectionCause,
		fmt.Sprintf("%d", request.ExpectedControlEpoch+1),
		request.RevokedAt,
	)
}
