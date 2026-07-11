package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

var _ goal.CheckpointStore = (*GlobalDB)(nil)

// CreateCheckpoint inserts one initial Goal checkpoint without overwriting durable progress.
func (g *GlobalDB) CreateCheckpoint(
	ctx context.Context,
	req goal.CreateCheckpointRequest,
) (goal.Checkpoint, error) {
	if err := g.checkReady(ctx, "create goal checkpoint"); err != nil {
		return goal.Checkpoint{}, err
	}
	checkpoint := req.Checkpoint
	if checkpoint.ControlEpoch == 0 {
		checkpoint.ControlEpoch = 1
	}
	if checkpoint.UpdatedAt.IsZero() {
		checkpoint.UpdatedAt = g.now()
	}
	if err := validateGoalCheckpoint(checkpoint); err != nil {
		return goal.Checkpoint{}, err
	}
	var persisted goal.Checkpoint
	err := g.withTaskImmediateTransaction(ctx, "create goal checkpoint", func(exec taskSQLExecutor) error {
		if err := validateGoalRunWorkspace(ctx, exec, checkpoint.Key); err != nil {
			return err
		}
		statusBeforePause := checkpoint.Status
		if err := projectPendingPauseToNewGoalCheckpoint(ctx, exec, &checkpoint); err != nil {
			return err
		}
		result, err := exec.ExecContext(
			ctx,
			`INSERT INTO loop_goal_checkpoints (
				loop_run_id, generation, node_id, item_index, control_epoch,
				control_actor_kind, control_actor_id, control_requested_at, phase, goal_status, control_cause,
				turns_used, turn_limit, broken_streak, recovery_streak, task_run_id,
				queue_entry_id, prompt_id, prompt_kind, prompt_attempt, context_state,
				usage_sequence, usage_pending_after_sequence, compaction_baseline_used,
				compaction_recovery_required, session_id, binding_handle, binding_epoch,
				context_nudge_ratio, control_grant_id, control_grant_kind, control_grant_cause,
				control_grant_turn, control_grant_scope, control_grant_consumed, judge_attempt_id,
				compaction_cancel_prompt_id, compaction_cancel_cause, compaction_cancel_requested_at,
				report_prompt_id, report_status, report_evidence_ref, report_binding_epoch,
				report_actor_kind, report_actor_id, report_recorded_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
			) ON CONFLICT(loop_run_id, generation, node_id, item_index) DO NOTHING`,
			goalCheckpointInsertArgs(checkpoint)...,
		)
		if err != nil {
			return fmt.Errorf("store: insert goal checkpoint: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("store: inspect inserted goal checkpoint: %w", err)
		}
		persisted, err = loadGoalCheckpointWithExecutor(ctx, exec, checkpoint.Key)
		if err != nil {
			return err
		}
		if persisted.TurnLimit != checkpoint.TurnLimit ||
			persisted.ContextNudgeRatio != checkpoint.ContextNudgeRatio {
			return goalControlStaleError("checkpoint already exists with different pinned policy")
		}
		if affected == 1 && statusBeforePause != checkpoint.Status {
			return appendGoalStatusChangedEvent(
				ctx,
				exec,
				checkpoint.Key,
				statusBeforePause,
				checkpoint.Status,
				checkpoint.ControlCause,
				checkpoint.ControlActorKind,
				checkpoint.ControlActorID,
				checkpoint.UpdatedAt,
			)
		}
		return nil
	})
	if err != nil {
		return goal.Checkpoint{}, err
	}
	return persisted, nil
}

func projectPendingPauseToNewGoalCheckpoint(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint *goal.Checkpoint,
) error {
	if checkpoint == nil || checkpoint.Phase == goalCheckpointPhaseAwaitingControl ||
		checkpoint.Phase == goalCheckpointPhaseTerminal {
		return nil
	}
	pauseRequested, actorKind, actorID, requestedAt, err := pendingGoalPauseActor(
		ctx,
		exec,
		checkpoint.Key.LoopRunID,
	)
	if err != nil {
		return err
	}
	if !pauseRequested {
		return nil
	}
	checkpoint.ControlActorKind = actorKind
	checkpoint.ControlActorID = actorID
	checkpoint.ControlRequestedAt = &requestedAt
	checkpoint.Phase = goalCheckpointPhaseAwaitingControl
	checkpoint.Status = goalStatusPaused
	checkpoint.ControlCause = loop.ReasonCode(loop.TransitionCausePauseBoundary)
	return validateGoalCheckpointControl(*checkpoint)
}

// LoadCheckpoint returns one checkpoint only through its owning workspace.
func (g *GlobalDB) LoadCheckpoint(ctx context.Context, key goal.TurnKey) (goal.Checkpoint, error) {
	if err := g.checkReady(ctx, "load goal checkpoint"); err != nil {
		return goal.Checkpoint{}, err
	}
	if err := validateGoalRunWorkspace(ctx, g.db, key); err != nil {
		return goal.Checkpoint{}, err
	}
	return loadGoalCheckpointWithExecutor(ctx, g.db, key)
}

func loadGoalCheckpointWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	key goal.TurnKey,
) (goal.Checkpoint, error) {
	checkpoint, err := scanGoalCheckpoint(exec.QueryRowContext(
		ctx,
		`SELECT `+goalCheckpointSelectColumns+`
		 FROM loop_goal_checkpoints
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?`,
		string(key.LoopRunID),
		key.Generation,
		string(key.NodeID),
		key.ItemIndex,
	), key.WorkspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return goal.Checkpoint{}, fmt.Errorf("%w: %s/%d/%s/%d", goal.ErrCheckpointNotFound,
			key.LoopRunID, key.Generation, key.NodeID, key.ItemIndex)
	}
	if err != nil {
		return goal.Checkpoint{}, fmt.Errorf("store: scan goal checkpoint: %w", err)
	}
	return checkpoint, nil
}

func validateGoalCheckpoint(checkpoint goal.Checkpoint) error {
	if err := checkpoint.Key.Validate(); err != nil {
		return err
	}
	if checkpoint.ControlEpoch < 1 || checkpoint.TurnLimit < 1 || checkpoint.TurnsUsed < 0 ||
		checkpoint.BrokenStreak < 0 || checkpoint.RecoveryStreak < 0 {
		return fmt.Errorf("%w: goal checkpoint counters are invalid", loop.ErrValidation)
	}
	if !goalCheckpointPhaseValid(checkpoint.Phase) || !goalStatusValid(checkpoint.Status) {
		return fmt.Errorf("%w: goal checkpoint phase or status is invalid", loop.ErrValidation)
	}
	if checkpoint.ContextNudgeRatio < 0 || checkpoint.ContextNudgeRatio > 1 {
		return fmt.Errorf("%w: goal context_nudge_ratio must be in [0,1]", loop.ErrValidation)
	}
	if err := validateGoalContextFreshness(checkpoint); err != nil {
		return err
	}
	if err := validateGoalCheckpointPrompt(checkpoint); err != nil {
		return err
	}
	if err := validateGoalCheckpointControl(checkpoint); err != nil {
		return err
	}
	return nil
}

func goalCheckpointPhaseValid(phase string) bool {
	switch strings.TrimSpace(phase) {
	case goalCheckpointPhaseIdle,
		goalCheckpointPhasePreparing,
		goalCheckpointPhaseQueued,
		goalCheckpointPhasePrompting,
		goalCheckpointPhaseCompacting,
		goalCheckpointPhaseJudging,
		goalCheckpointPhasePersisting,
		goalCheckpointPhaseAwaitingControl,
		goalCheckpointPhaseTerminal:
		return true
	default:
		return false
	}
}

func goalStatusValid(status string) bool {
	switch strings.TrimSpace(status) {
	case goalStatusActive,
		goalStatusPaused,
		goalStatusBlocked,
		goalStatusUsageLimited,
		goalStatusBudgetLimited,
		goalStatusComplete:
		return true
	default:
		return false
	}
}

func validateGoalContextFreshness(checkpoint goal.Checkpoint) error {
	if err := validateGoalCompactionFreshness(checkpoint); err != nil {
		return err
	}
	switch checkpoint.ContextState {
	case goalContextStateKnown:
		if checkpoint.UsageSequence == nil || checkpoint.UsagePendingAfterSequence != nil {
			return fmt.Errorf("%w: known goal context requires usage sequence only", loop.ErrValidation)
		}
	case goalContextStateUnknown:
		if checkpoint.UsageSequence != nil || checkpoint.UsagePendingAfterSequence != nil {
			return fmt.Errorf("%w: unknown goal context cannot carry usage sequence", loop.ErrValidation)
		}
	case goalContextStatePending:
		if checkpoint.UsagePendingAfterSequence != nil && checkpoint.UsageSequence != nil &&
			*checkpoint.UsagePendingAfterSequence != *checkpoint.UsageSequence {
			return fmt.Errorf("%w: pending goal context freshness floor is invalid", loop.ErrValidation)
		}
	default:
		return fmt.Errorf("%w: goal context state is invalid", loop.ErrValidation)
	}
	return nil
}

func validateGoalCompactionFreshness(checkpoint goal.Checkpoint) error {
	if checkpoint.CompactionBaselineUsed != nil && *checkpoint.CompactionBaselineUsed < 0 {
		return fmt.Errorf("%w: Goal compaction baseline usage is invalid", loop.ErrValidation)
	}
	if checkpoint.CompactionRecoveryRequired && checkpoint.ContextState != goalContextStateKnown {
		return fmt.Errorf("%w: Goal compaction recovery requires known usage", loop.ErrValidation)
	}
	baselineAllowed := checkpoint.ContextState == goalContextStatePending ||
		(checkpoint.ContextState == goalContextStateKnown && checkpoint.PromptKind == goalPromptKindCompact &&
			(checkpoint.Phase == goalCheckpointPhaseQueued || checkpoint.Phase == goalCheckpointPhaseCompacting))
	if checkpoint.CompactionBaselineUsed != nil && !baselineAllowed {
		return fmt.Errorf("%w: Goal compaction baseline has no active compact operation", loop.ErrValidation)
	}
	return nil
}

func validateGoalCheckpointPrompt(checkpoint goal.Checkpoint) error {
	if checkpoint.PromptKind == "" && checkpoint.PromptID == "" && checkpoint.QueueEntryID == "" {
		return nil
	}
	if checkpoint.PromptKind != goalPromptKindWork && checkpoint.PromptKind != goalPromptKindContinuation &&
		checkpoint.PromptKind != goalPromptKindCompact {
		return fmt.Errorf("%w: goal checkpoint prompt kind is invalid", loop.ErrValidation)
	}
	if strings.TrimSpace(checkpoint.PromptID) == "" || strings.TrimSpace(checkpoint.QueueEntryID) == "" ||
		strings.TrimSpace(checkpoint.TaskRunID) == "" || strings.TrimSpace(checkpoint.SessionID) == "" ||
		strings.TrimSpace(checkpoint.BindingHandle) == "" || checkpoint.BindingEpoch < 1 {
		return fmt.Errorf("%w: goal checkpoint prompt identity is incomplete", loop.ErrValidation)
	}
	return nil
}

func validateGoalCheckpointControl(checkpoint goal.Checkpoint) error {
	hasActor := checkpoint.ControlActorKind != "" || checkpoint.ControlActorID != "" ||
		checkpoint.ControlRequestedAt != nil
	if hasActor && (strings.TrimSpace(checkpoint.ControlActorKind) == "" ||
		strings.TrimSpace(checkpoint.ControlActorID) == "" || checkpoint.ControlRequestedAt == nil) {
		return fmt.Errorf("%w: goal control actor identity is incomplete", loop.ErrValidation)
	}
	if checkpoint.ControlGrant != nil && (checkpoint.ControlGrant.ID < 1 ||
		checkpoint.ControlGrant.Kind == "" || checkpoint.ControlGrant.Cause == "" ||
		checkpoint.ControlGrant.Scope == "" || checkpoint.ControlGrant.Turn < 0) {
		return fmt.Errorf("%w: goal control grant identity is incomplete", loop.ErrValidation)
	}
	return nil
}

func goalCheckpointInsertArgs(checkpoint goal.Checkpoint) []any {
	grant := checkpoint.ControlGrant
	return []any{
		string(checkpoint.Key.LoopRunID), checkpoint.Key.Generation, string(checkpoint.Key.NodeID),
		checkpoint.Key.ItemIndex, checkpoint.ControlEpoch, nullableGoalString(checkpoint.ControlActorKind),
		nullableGoalString(checkpoint.ControlActorID), nullableGoalTime(checkpoint.ControlRequestedAt),
		checkpoint.Phase, checkpoint.Status, nullableGoalString(string(checkpoint.ControlCause)),
		checkpoint.TurnsUsed, checkpoint.TurnLimit,
		checkpoint.BrokenStreak, checkpoint.RecoveryStreak, nullableGoalString(checkpoint.TaskRunID),
		nullableGoalString(checkpoint.QueueEntryID), nullableGoalString(checkpoint.PromptID),
		nullableGoalString(checkpoint.PromptKind), checkpoint.PromptAttempt, checkpoint.ContextState,
		nullableGoalInt64(checkpoint.UsageSequence), nullableGoalInt64(checkpoint.UsagePendingAfterSequence),
		nullableGoalInt64(checkpoint.CompactionBaselineUsed), boolToInt(checkpoint.CompactionRecoveryRequired),
		nullableGoalString(checkpoint.SessionID), nullableGoalString(checkpoint.BindingHandle),
		nullableGoalPositiveInt64(checkpoint.BindingEpoch), checkpoint.ContextNudgeRatio,
		goalGrantID(grant), goalGrantKind(grant), goalGrantCause(grant), goalGrantTurn(grant),
		goalGrantScope(grant), goalGrantConsumed(grant), nullableGoalString(checkpoint.JudgeAttemptID),
		goalCancelPromptID(checkpoint.CompactionCancel), goalCancelCause(checkpoint.CompactionCancel),
		goalCancelRequestedAt(checkpoint.CompactionCancel), goalReportPromptID(checkpoint.ReportIntent),
		goalReportStatus(checkpoint.ReportIntent), goalReportEvidence(checkpoint.ReportIntent),
		goalReportBindingEpoch(checkpoint.ReportIntent), goalReportActorKind(checkpoint.ReportIntent),
		goalReportActorID(checkpoint.ReportIntent), goalReportRecordedAt(checkpoint.ReportIntent),
		store.FormatTimestamp(checkpoint.UpdatedAt),
	}
}

func nullableGoalString(value string) any {
	return store.NullableString(strings.TrimSpace(value))
}

func nullableGoalTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return store.FormatTimestamp(value.UTC())
}

func nullableGoalInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableGoalPositiveInt64(value int64) any {
	if value < 1 {
		return nil
	}
	return value
}
