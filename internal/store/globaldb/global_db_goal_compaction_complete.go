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

type goalCompactionProjection struct {
	contextState     string
	usageSequence    any
	pendingAfter     any
	baselineUsed     any
	recoveryRequired bool
	recoveryStreak   int
	consumeGrant     bool
}

// CompleteCompaction settles one correlated compact operation without creating a work-turn row.
func (g *GlobalDB) CompleteCompaction(
	ctx context.Context,
	req goal.CompleteCompactionRequest,
) (goal.Checkpoint, error) {
	if err := g.checkReady(ctx, "complete Goal compaction"); err != nil {
		return goal.Checkpoint{}, err
	}
	if err := validateCompleteGoalCompactionRequest(req); err != nil {
		return goal.Checkpoint{}, err
	}
	var completed goal.Checkpoint
	err := g.withTaskImmediateTransaction(ctx, "complete Goal compaction", func(exec taskSQLExecutor) error {
		var err error
		completed, err = g.completeCompactionWithExecutor(ctx, exec, req)
		return err
	})
	if err != nil {
		return goal.Checkpoint{}, err
	}
	return completed, nil
}

func (g *GlobalDB) completeCompactionWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.CompleteCompactionRequest,
) (goal.Checkpoint, error) {
	if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
		return goal.Checkpoint{}, err
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return goal.Checkpoint{}, err
	}
	if err := validateGoalCompactionSettlement(ctx, exec, checkpoint, req); err != nil {
		return goal.Checkpoint{}, err
	}
	projection := goalCompactionCheckpointProjection(checkpoint, req.Result)
	now := g.now().UTC()
	if err := settleGoalCompactionCheckpoint(ctx, exec, req, projection, now); err != nil {
		return goal.Checkpoint{}, err
	}
	return loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
}

func validateGoalCompactionSettlement(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	req goal.CompleteCompactionRequest,
) error {
	if checkpoint.ControlEpoch != req.ExpectedControlEpoch ||
		checkpoint.BindingEpoch != req.ExpectedBindingEpoch ||
		checkpoint.Phase != goalCheckpointPhaseCompacting ||
		checkpoint.TaskRunID != strings.TrimSpace(req.TaskRunID) ||
		checkpoint.QueueEntryID != strings.TrimSpace(req.QueueEntryID) ||
		checkpoint.PromptID != strings.TrimSpace(req.PromptID) ||
		checkpoint.PromptKind != goalPromptKindCompact {
		return goalControlStaleError("Goal compaction settlement owner changed")
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
	if row.terminalAt == nil || !goalPromptTerminalMatches(&row, req.Result.PromptResult) {
		return goalControlStaleError(
			"Goal compact queue terminal differs from settlement",
		)
	}
	if req.Result.Outcome == goal.CompactionSucceeded {
		return validateSuccessfulCompactionBaseline(checkpoint, req.Result)
	}
	return nil
}

func validateSuccessfulCompactionBaseline(checkpoint goal.Checkpoint, result goal.CompactionResult) error {
	if (checkpoint.CompactionBaselineUsed == nil) != (checkpoint.UsageSequence == nil) {
		return goalControlStaleError("Goal compaction durable freshness baseline is incomplete")
	}
	if result.UsageSequence != nil &&
		(checkpoint.UsageSequence == nil || *result.UsageSequence != *checkpoint.UsageSequence) {
		return goalControlStaleError("Goal compaction usage sequence changed")
	}
	if result.UsageBaselineUsed != nil &&
		(checkpoint.CompactionBaselineUsed == nil ||
			*result.UsageBaselineUsed != *checkpoint.CompactionBaselineUsed) {
		return goalControlStaleError("Goal compaction usage baseline changed")
	}
	return nil
}

func goalCompactionCheckpointProjection(
	checkpoint goal.Checkpoint,
	result goal.CompactionResult,
) goalCompactionProjection {
	projection := goalCompactionProjection{
		contextState:     checkpoint.ContextState,
		usageSequence:    nullableGoalInt64Value(checkpoint.UsageSequence),
		pendingAfter:     nullableGoalInt64Value(checkpoint.UsagePendingAfterSequence),
		baselineUsed:     nullableGoalInt64Value(checkpoint.CompactionBaselineUsed),
		recoveryRequired: checkpoint.CompactionRecoveryRequired,
		recoveryStreak:   checkpoint.RecoveryStreak,
		consumeGrant: checkpoint.ControlGrant != nil && !checkpoint.ControlGrant.Consumed &&
			checkpoint.ControlGrant.Kind == goal.ControlGrantBudget &&
			checkpoint.ControlGrant.Turn == checkpoint.TurnsUsed,
	}
	if result.Outcome == goal.CompactionSucceeded {
		projection.contextState = goalContextStatePending
		projection.usageSequence = nullableGoalInt64Value(checkpoint.UsageSequence)
		projection.pendingAfter = nullableGoalInt64Value(checkpoint.UsageSequence)
		projection.baselineUsed = nullableGoalInt64Value(checkpoint.CompactionBaselineUsed)
		projection.recoveryRequired = false
	} else {
		projection.baselineUsed = nil
	}
	return projection
}

func settleGoalCompactionCheckpoint(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.CompleteCompactionRequest,
	projection goalCompactionProjection,
	now time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
			 SET phase = 'idle', goal_status = 'active', control_cause = NULL,
			     queue_entry_id = NULL, prompt_id = NULL, prompt_kind = NULL, prompt_attempt = 0,
			     context_state = ?, usage_sequence = ?, usage_pending_after_sequence = ?,
			     compaction_baseline_used = ?, compaction_recovery_required = ?,
			     recovery_streak = ?,
			     compaction_cancel_prompt_id = NULL, compaction_cancel_cause = NULL,
			     compaction_cancel_requested_at = NULL,
			     control_grant_consumed = CASE WHEN ? = 1 THEN 1 ELSE control_grant_consumed END,
			     updated_at = ?
			 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
			   AND control_epoch = ? AND binding_epoch = ? AND phase = 'compacting'
			   AND task_run_id = ? AND queue_entry_id = ? AND prompt_id = ?`,
		projection.contextState,
		projection.usageSequence,
		projection.pendingAfter,
		projection.baselineUsed,
		boolToInt(projection.recoveryRequired),
		projection.recoveryStreak,
		boolToInt(projection.consumeGrant),
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
		return fmt.Errorf("store: settle Goal compaction checkpoint: %w", err)
	}
	return requireGoalRowsAffected(result, "settle Goal compaction checkpoint")
}

func validateCompleteGoalCompactionRequest(req goal.CompleteCompactionRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.ExpectedBindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" ||
		strings.TrimSpace(req.Result.PromptResult.PromptID) != strings.TrimSpace(req.PromptID) {
		return fmt.Errorf("%w: Goal compaction settlement identity is incomplete", looppkg.ErrValidation)
	}
	if err := validateGoalPromptResult(req.Result.PromptResult); err != nil {
		return err
	}
	switch req.Result.Outcome {
	case goal.CompactionSucceeded, goal.CompactionIneffective, goal.CompactionRefused,
		goal.CompactionCancelled, goal.CompactionTimedOut, goal.CompactionFailed:
	default:
		return fmt.Errorf("%w: Goal compaction outcome %q is invalid", looppkg.ErrValidation, req.Result.Outcome)
	}
	if req.Result.UsageSequence != nil && *req.Result.UsageSequence < 0 {
		return fmt.Errorf("%w: Goal compaction usage sequence is invalid", looppkg.ErrValidation)
	}
	if req.Result.UsageBaselineUsed != nil && *req.Result.UsageBaselineUsed < 0 {
		return fmt.Errorf("%w: Goal compaction baseline usage is invalid", looppkg.ErrValidation)
	}
	if (req.Result.UsageSequence == nil) != (req.Result.UsageBaselineUsed == nil) {
		return fmt.Errorf("%w: Goal compaction freshness baseline is incomplete", looppkg.ErrValidation)
	}
	return nil
}
