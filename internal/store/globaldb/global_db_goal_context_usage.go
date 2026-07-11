package globaldb

import (
	"context"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

type goalContextUsageProjection struct {
	contextState     string
	usageSequence    int64
	pendingAfter     any
	baselineUsed     any
	recoveryRequired bool
	recoveryStreak   int
}

// RecordContextUsage advances the durable freshness cursor for one exact active binding.
func (g *GlobalDB) RecordContextUsage(
	ctx context.Context,
	req goal.RecordContextUsageRequest,
) (goal.Checkpoint, error) {
	if err := g.checkReady(ctx, "record Goal context usage"); err != nil {
		return goal.Checkpoint{}, err
	}
	if err := validateGoalContextUsageRequest(req); err != nil {
		return goal.Checkpoint{}, err
	}
	var updated goal.Checkpoint
	err := g.withTaskImmediateTransaction(ctx, "record Goal context usage", func(exec taskSQLExecutor) error {
		var err error
		updated, err = g.recordContextUsageWithExecutor(ctx, exec, req)
		return err
	})
	if err != nil {
		return goal.Checkpoint{}, err
	}
	return updated, nil
}

func (g *GlobalDB) recordContextUsageWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.RecordContextUsageRequest,
) (goal.Checkpoint, error) {
	if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
		return goal.Checkpoint{}, err
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return goal.Checkpoint{}, err
	}
	if !goalContextUsageOwnerMatches(checkpoint, req) {
		return goal.Checkpoint{}, goalControlStaleError("Goal context usage owner changed")
	}
	if err := validateActiveGoalPromptBinding(
		ctx,
		exec,
		req.Key,
		checkpoint.BindingHandle,
		req.ExpectedBindingEpoch,
		checkpoint.SessionID,
	); err != nil {
		return goal.Checkpoint{}, err
	}
	if !goalContextUsageAdvances(checkpoint, req.Usage.Sequence) {
		return checkpoint, nil
	}
	projection := goalContextUsageCheckpointProjection(checkpoint, req.Usage)
	now := g.now().UTC()
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET context_state = ?, usage_sequence = ?, usage_pending_after_sequence = ?,
		     compaction_baseline_used = ?, compaction_recovery_required = ?, recovery_streak = ?, updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND binding_epoch = ? AND phase = ? AND goal_status = 'active'
		   AND session_id = ? AND binding_handle = ?`,
		projection.contextState,
		projection.usageSequence,
		projection.pendingAfter,
		projection.baselineUsed,
		boolToInt(projection.recoveryRequired),
		projection.recoveryStreak,
		store.FormatTimestamp(now),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		req.ExpectedControlEpoch,
		req.ExpectedBindingEpoch,
		strings.TrimSpace(req.ExpectedPhase),
		strings.TrimSpace(req.SessionID),
		strings.TrimSpace(req.BindingHandle),
	)
	if err != nil {
		return goal.Checkpoint{}, fmt.Errorf("store: record Goal context usage: %w", err)
	}
	if err := requireGoalRowsAffected(result, "record Goal context usage"); err != nil {
		return goal.Checkpoint{}, err
	}
	return loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
}

func goalContextUsageCheckpointProjection(
	checkpoint goal.Checkpoint,
	usage goal.ContextUsage,
) goalContextUsageProjection {
	projection := goalContextUsageProjection{
		contextState:     goalContextStateKnown,
		usageSequence:    usage.Sequence,
		recoveryRequired: checkpoint.CompactionRecoveryRequired,
		recoveryStreak:   checkpoint.RecoveryStreak,
	}
	if checkpoint.ContextState != goalContextStatePending {
		return projection
	}
	if checkpoint.CompactionBaselineUsed == nil || usage.Used < *checkpoint.CompactionBaselineUsed {
		projection.recoveryRequired = false
		projection.recoveryStreak = 0
		return projection
	}
	projection.recoveryRequired = true
	return projection
}

func goalContextUsageOwnerMatches(
	checkpoint goal.Checkpoint,
	req goal.RecordContextUsageRequest,
) bool {
	return checkpoint.ControlEpoch == req.ExpectedControlEpoch &&
		checkpoint.BindingEpoch == req.ExpectedBindingEpoch &&
		checkpoint.Phase == strings.TrimSpace(req.ExpectedPhase) &&
		checkpoint.Status == goalStatusActive &&
		checkpoint.SessionID == strings.TrimSpace(req.SessionID) &&
		checkpoint.BindingHandle == strings.TrimSpace(req.BindingHandle)
}

func goalContextUsageAdvances(checkpoint goal.Checkpoint, sequence int64) bool {
	switch checkpoint.ContextState {
	case goalContextStateKnown:
		return checkpoint.UsageSequence == nil || sequence > *checkpoint.UsageSequence
	case goalContextStateUnknown:
		return true
	case goalContextStatePending:
		return checkpoint.UsagePendingAfterSequence == nil || sequence > *checkpoint.UsagePendingAfterSequence
	default:
		return false
	}
}

func validateGoalContextUsageRequest(req goal.RecordContextUsageRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	usage := req.Usage
	if req.ExpectedControlEpoch < 1 || req.ExpectedBindingEpoch < 1 ||
		!goalCheckpointPhaseValid(strings.TrimSpace(req.ExpectedPhase)) ||
		strings.TrimSpace(req.SessionID) == "" || strings.TrimSpace(req.BindingHandle) == "" ||
		!usage.Known || usage.Used < 0 || usage.Size <= 0 || usage.Sequence < 0 || usage.ReportedAt.IsZero() {
		return fmt.Errorf("%w: Goal context usage observation is invalid", looppkg.ErrValidation)
	}
	return nil
}
