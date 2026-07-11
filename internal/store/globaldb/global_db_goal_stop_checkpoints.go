package globaldb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

type stoppableGoalCheckpointKey struct {
	generation int
	nodeID     looppkg.NodeID
	itemIndex  int
}

func stopGoalCheckpointsWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	request looppkg.GoalRunStopRequest,
	enqueueProjection bool,
) ([]looppkg.GoalPromptLease, error) {
	keys, err := loadStoppableGoalCheckpointKeys(ctx, exec, request)
	if err != nil {
		return nil, err
	}
	leases := make([]looppkg.GoalPromptLease, 0, len(keys))
	for _, key := range keys {
		checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, goal.TurnKey{
			WorkspaceID: request.WorkspaceID,
			LoopRunID:   request.RunID,
			Generation:  key.generation,
			NodeID:      key.nodeID,
			ItemIndex:   key.itemIndex,
		})
		if err != nil {
			return nil, err
		}
		if revocableGoalCheckpointPhase(checkpoint.Phase) {
			revokeRequest := stoppedGoalRevokeRequest(request, checkpoint)
			if _, err := revokeGoalPromptWithExecutorOptions(
				ctx,
				exec,
				revokeRequest,
				enqueueProjection,
			); err != nil {
				return nil, err
			}
			lease := goalPromptLease(checkpoint)
			if !validGoalPromptLease(lease) {
				return nil, fmt.Errorf("%w: stopped Goal prompt lease is incomplete", looppkg.ErrTransitionConflict)
			}
			leases = append(leases, lease)
			continue
		}
		if err := terminalizeStoppedGoalCheckpoint(
			ctx,
			exec,
			request,
			checkpoint,
			enqueueProjection,
		); err != nil {
			return nil, err
		}
	}
	return leases, nil
}

func loadStoppableGoalCheckpointKeys(
	ctx context.Context,
	exec taskSQLExecutor,
	request looppkg.GoalRunStopRequest,
) ([]stoppableGoalCheckpointKey, error) {
	rows, err := exec.QueryContext(
		ctx,
		`SELECT generation, node_id, item_index
		 FROM loop_goal_checkpoints
		 WHERE loop_run_id = ? AND phase != 'terminal'
		 ORDER BY generation ASC, node_id ASC, item_index ASC`,
		string(request.RunID),
	)
	if err != nil {
		return nil, fmt.Errorf("store: list stoppable Goal checkpoints: %w", err)
	}
	keys := make([]stoppableGoalCheckpointKey, 0)
	for rows.Next() {
		var key stoppableGoalCheckpointKey
		if err := rows.Scan(&key.generation, &key.nodeID, &key.itemIndex); err != nil {
			return nil, closeGoalStopRows(rows, fmt.Errorf("store: scan stoppable Goal checkpoint: %w", err))
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, closeGoalStopRows(rows, fmt.Errorf("store: iterate stoppable Goal checkpoints: %w", err))
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("store: close stoppable Goal checkpoints: %w", err)
	}
	return keys, nil
}

func closeGoalStopRows(rows interface{ Close() error }, cause error) error {
	if err := rows.Close(); err != nil {
		return errors.Join(cause, fmt.Errorf("store: close stoppable Goal checkpoints: %w", err))
	}
	return cause
}

func terminalizeStoppedGoalCheckpoint(
	ctx context.Context,
	exec taskSQLExecutor,
	request looppkg.GoalRunStopRequest,
	checkpoint goal.Checkpoint,
	enqueueProjection bool,
) error {
	revoke := stoppedGoalRevokeRequest(request, checkpoint)
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET control_epoch = control_epoch + 1, phase = 'terminal', goal_status = 'paused', control_cause = ?,
		     control_actor_kind = ?, control_actor_id = ?, control_requested_at = ?,
		     queue_entry_id = NULL, prompt_id = NULL, prompt_kind = NULL, prompt_attempt = 0,
		     judge_attempt_id = NULL,
		     report_prompt_id = NULL, report_status = NULL, report_evidence_ref = NULL,
		     report_binding_epoch = NULL, report_actor_kind = NULL, report_actor_id = NULL,
		     report_recorded_at = NULL, compaction_cancel_prompt_id = NULL,
		     compaction_cancel_cause = NULL, compaction_cancel_requested_at = NULL,
		     updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND phase = ? AND phase != 'terminal'`,
		string(revoke.Cause),
		revoke.ActorKind,
		revoke.ActorID,
		store.FormatTimestamp(request.StoppedAt),
		store.FormatTimestamp(request.StoppedAt),
		string(request.RunID),
		checkpoint.Key.Generation,
		string(checkpoint.Key.NodeID),
		checkpoint.Key.ItemIndex,
		checkpoint.ControlEpoch,
		checkpoint.Phase,
	)
	if err != nil {
		return fmt.Errorf("store: terminalize stopped Goal checkpoint: %w", err)
	}
	if err := requireGoalRowsAffected(result, "terminalize stopped Goal checkpoint"); err != nil {
		return err
	}
	if err := projectGoalCheckpointCounts(
		ctx,
		exec,
		checkpoint.Key,
		goalStatusPaused,
		checkpoint.TurnsUsed,
		checkpoint.TurnLimit,
	); err != nil {
		return err
	}
	if _, _, err := appendGoalStatusChangedRunEvent(
		ctx,
		exec,
		checkpoint.Key,
		checkpoint.Status,
		goalStatusPaused,
		revoke.Cause,
		revoke.ActorKind,
		revoke.ActorID,
		request.StoppedAt,
	); err != nil {
		return err
	}
	if !enqueueProjection {
		return nil
	}
	return enqueueRevokedGoalProjection(ctx, exec, checkpoint, revoke)
}

func closeStoppedGoalBindings(
	ctx context.Context,
	exec taskSQLExecutor,
	request looppkg.GoalRunStopRequest,
) error {
	rows, err := exec.QueryContext(
		ctx,
		`SELECT handle, binding_epoch, session_id
		 FROM loop_session_bindings
		 WHERE loop_run_id = ? AND workspace_id = ? AND state = 'active'
		 ORDER BY handle ASC, binding_epoch ASC`,
		string(request.RunID),
		string(request.WorkspaceID),
	)
	if err != nil {
		return fmt.Errorf("store: list stopped Goal bindings: %w", err)
	}
	type stoppedBinding struct {
		handle    string
		epoch     int64
		sessionID string
	}
	bindings := make([]stoppedBinding, 0)
	for rows.Next() {
		var binding stoppedBinding
		if err := rows.Scan(&binding.handle, &binding.epoch, &binding.sessionID); err != nil {
			return closeGoalStopRows(rows, fmt.Errorf("store: scan stopped Goal binding: %w", err))
		}
		bindings = append(bindings, binding)
	}
	if err := rows.Err(); err != nil {
		return closeGoalStopRows(rows, fmt.Errorf("store: iterate stopped Goal bindings: %w", err))
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("store: close stopped Goal binding rows: %w", err)
	}
	for _, binding := range bindings {
		if err := closeGoalBindingWithCleanup(
			ctx,
			exec,
			goal.BindingKey{
				WorkspaceID: request.WorkspaceID,
				LoopRunID:   request.RunID,
				Handle:      binding.handle,
			},
			binding.epoch,
			binding.sessionID,
			goal.SessionCleanupCauseStop,
			request.StoppedAt,
		); err != nil {
			return err
		}
	}
	return failStoppedCreatingGoalBindings(ctx, exec, request)
}

func failStoppedCreatingGoalBindings(
	ctx context.Context,
	exec taskSQLExecutor,
	request looppkg.GoalRunStopRequest,
) error {
	rows, err := exec.QueryContext(
		ctx,
		`SELECT handle, binding_epoch FROM loop_session_bindings
		 WHERE loop_run_id = ? AND workspace_id = ? AND ownership = 'run-owned' AND state = 'creating'
		 ORDER BY handle ASC, binding_epoch ASC`,
		string(request.RunID),
		string(request.WorkspaceID),
	)
	if err != nil {
		return fmt.Errorf("store: list stopped creating Goal bindings: %w", err)
	}
	type creatingBinding struct {
		handle string
		epoch  int64
	}
	creating := make([]creatingBinding, 0)
	for rows.Next() {
		var binding creatingBinding
		if err := rows.Scan(&binding.handle, &binding.epoch); err != nil {
			return closeGoalStopRows(rows, fmt.Errorf("store: scan stopped creating Goal binding: %w", err))
		}
		creating = append(creating, binding)
	}
	if err := rows.Err(); err != nil {
		return closeGoalStopRows(rows, fmt.Errorf("store: iterate stopped creating Goal bindings: %w", err))
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("store: close stopped creating Goal binding rows: %w", err)
	}
	for _, candidate := range creating {
		key := goal.BindingKey{WorkspaceID: request.WorkspaceID, LoopRunID: request.RunID, Handle: candidate.handle}
		binding, err := getSessionBindingAttemptWithExecutor(ctx, exec, key, candidate.epoch)
		if err != nil {
			return err
		}
		result, err := exec.ExecContext(
			ctx,
			`UPDATE loop_session_bindings
			 SET state = 'failed', failure_code = ?, failed_at = ?
			 WHERE loop_run_id = ? AND handle = ? AND binding_epoch = ? AND state = 'creating'`,
			goalBindingFailureStopCreationUnsettled,
			store.FormatTimestamp(request.StoppedAt),
			string(request.RunID),
			candidate.handle,
			candidate.epoch,
		)
		if err != nil {
			return fmt.Errorf("store: fail stopped creating Goal binding: %w", err)
		}
		if err := requireGoalRowsAffected(result, "fail stopped creating Goal binding"); err != nil {
			return err
		}
		if err := enqueueGoalSessionCleanupWithExecutor(
			ctx, exec, binding, goal.SessionCleanupCauseStop, request.StoppedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

func validGoalPromptLease(lease looppkg.GoalPromptLease) bool {
	return strings.TrimSpace(lease.QueueEntryID) != "" && strings.TrimSpace(lease.SessionID) != "" &&
		lease.OwnerKind == "goal" && strings.TrimSpace(lease.LoopRunID) != "" &&
		strings.TrimSpace(lease.TaskRunID) != "" && lease.RunGeneration > 0 &&
		lease.PromptAttempt >= 0 && lease.ControlEpoch > 0 && lease.BindingEpoch > 0 &&
		strings.TrimSpace(lease.PromptID) != "" && strings.TrimSpace(lease.PromptKind) != ""
}
