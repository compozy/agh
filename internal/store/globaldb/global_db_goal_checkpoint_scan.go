package globaldb

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
)

const goalCheckpointSelectColumns = `
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
	report_actor_kind, report_actor_id, report_recorded_at, updated_at`

type goalCheckpointScanFields struct {
	workspaceID                loop.WorkspaceID
	controlActorKind           sql.NullString
	controlActorID             sql.NullString
	controlRequestedAt         any
	controlCause               sql.NullString
	taskRunID                  sql.NullString
	queueEntryID               sql.NullString
	promptID                   sql.NullString
	promptKind                 sql.NullString
	usageSequence              sql.NullInt64
	usagePendingAfterSequence  sql.NullInt64
	compactionBaselineUsed     sql.NullInt64
	compactionRecoveryRequired int
	sessionID                  sql.NullString
	bindingHandle              sql.NullString
	bindingEpoch               sql.NullInt64
	controlGrantKind           sql.NullString
	controlGrantCause          sql.NullString
	controlGrantTurn           sql.NullInt64
	controlGrantScope          sql.NullString
	controlGrantConsumed       int
	judgeAttemptID             sql.NullString
	cancelPromptID             sql.NullString
	cancelCause                sql.NullString
	cancelRequestedAt          any
	reportPromptID             sql.NullString
	reportStatus               sql.NullString
	reportEvidenceRef          sql.NullString
	reportBindingEpoch         sql.NullInt64
	reportActorKind            sql.NullString
	reportActorID              sql.NullString
	reportRecordedAt           any
	updatedAt                  any
}

func scanGoalCheckpoint(
	scanner rowScanner,
	workspaceID loop.WorkspaceID,
) (goal.Checkpoint, error) {
	var checkpoint goal.Checkpoint
	var controlGrantID int64
	fields := goalCheckpointScanFields{workspaceID: workspaceID}
	if err := scanner.Scan(
		&checkpoint.Key.LoopRunID,
		&checkpoint.Key.Generation,
		&checkpoint.Key.NodeID,
		&checkpoint.Key.ItemIndex,
		&checkpoint.ControlEpoch,
		&fields.controlActorKind,
		&fields.controlActorID,
		&fields.controlRequestedAt,
		&checkpoint.Phase,
		&checkpoint.Status,
		&fields.controlCause,
		&checkpoint.TurnsUsed,
		&checkpoint.TurnLimit,
		&checkpoint.BrokenStreak,
		&checkpoint.RecoveryStreak,
		&fields.taskRunID,
		&fields.queueEntryID,
		&fields.promptID,
		&fields.promptKind,
		&checkpoint.PromptAttempt,
		&checkpoint.ContextState,
		&fields.usageSequence,
		&fields.usagePendingAfterSequence,
		&fields.compactionBaselineUsed,
		&fields.compactionRecoveryRequired,
		&fields.sessionID,
		&fields.bindingHandle,
		&fields.bindingEpoch,
		&checkpoint.ContextNudgeRatio,
		&controlGrantID,
		&fields.controlGrantKind,
		&fields.controlGrantCause,
		&fields.controlGrantTurn,
		&fields.controlGrantScope,
		&fields.controlGrantConsumed,
		&fields.judgeAttemptID,
		&fields.cancelPromptID,
		&fields.cancelCause,
		&fields.cancelRequestedAt,
		&fields.reportPromptID,
		&fields.reportStatus,
		&fields.reportEvidenceRef,
		&fields.reportBindingEpoch,
		&fields.reportActorKind,
		&fields.reportActorID,
		&fields.reportRecordedAt,
		&fields.updatedAt,
	); err != nil {
		return goal.Checkpoint{}, err
	}
	if controlGrantID > 0 {
		checkpoint.ControlGrant = &goal.ControlGrant{ID: controlGrantID}
	}
	if err := fields.apply(&checkpoint); err != nil {
		return goal.Checkpoint{}, err
	}
	return checkpoint, nil
}

func (fields *goalCheckpointScanFields) apply(checkpoint *goal.Checkpoint) error {
	checkpoint.Key.WorkspaceID = fields.workspaceID
	checkpoint.ControlActorKind = strings.TrimSpace(fields.controlActorKind.String)
	checkpoint.ControlActorID = strings.TrimSpace(fields.controlActorID.String)
	checkpoint.ControlCause = loop.ReasonCode(strings.TrimSpace(fields.controlCause.String))
	checkpoint.TaskRunID = strings.TrimSpace(fields.taskRunID.String)
	checkpoint.QueueEntryID = strings.TrimSpace(fields.queueEntryID.String)
	checkpoint.PromptID = strings.TrimSpace(fields.promptID.String)
	checkpoint.PromptKind = strings.TrimSpace(fields.promptKind.String)
	checkpoint.SessionID = strings.TrimSpace(fields.sessionID.String)
	checkpoint.BindingHandle = strings.TrimSpace(fields.bindingHandle.String)
	checkpoint.JudgeAttemptID = strings.TrimSpace(fields.judgeAttemptID.String)
	if fields.bindingEpoch.Valid {
		checkpoint.BindingEpoch = fields.bindingEpoch.Int64
	}
	if fields.usageSequence.Valid {
		value := fields.usageSequence.Int64
		checkpoint.UsageSequence = &value
	}
	if fields.usagePendingAfterSequence.Valid {
		value := fields.usagePendingAfterSequence.Int64
		checkpoint.UsagePendingAfterSequence = &value
	}
	if fields.compactionBaselineUsed.Valid {
		value := fields.compactionBaselineUsed.Int64
		checkpoint.CompactionBaselineUsed = &value
	}
	checkpoint.CompactionRecoveryRequired = fields.compactionRecoveryRequired != 0
	var err error
	if checkpoint.ControlRequestedAt, err = parseOptionalGoalTimestampValue(
		fields.controlRequestedAt,
		"checkpoint control_requested_at",
	); err != nil {
		return err
	}
	if err := fields.applyGrant(checkpoint); err != nil {
		return err
	}
	if err := fields.applyCancel(checkpoint); err != nil {
		return err
	}
	if err := fields.applyReport(checkpoint); err != nil {
		return err
	}
	checkpoint.UpdatedAt, err = parseGoalTimestampValue(fields.updatedAt, "checkpoint updated_at")
	return err
}

func (fields *goalCheckpointScanFields) applyGrant(checkpoint *goal.Checkpoint) error {
	if checkpoint.ControlGrant == nil || checkpoint.ControlGrant.ID == 0 {
		checkpoint.ControlGrant = nil
		return nil
	}
	checkpoint.ControlGrant.Kind = goal.ControlGrantKind(strings.TrimSpace(fields.controlGrantKind.String))
	checkpoint.ControlGrant.Cause = loop.ReasonCode(strings.TrimSpace(fields.controlGrantCause.String))
	checkpoint.ControlGrant.Scope = goal.ControlGrantScope(strings.TrimSpace(fields.controlGrantScope.String))
	checkpoint.ControlGrant.Consumed = fields.controlGrantConsumed != 0
	if !fields.controlGrantTurn.Valid {
		return fmt.Errorf("store: goal checkpoint grant %d has no turn", checkpoint.ControlGrant.ID)
	}
	checkpoint.ControlGrant.Turn = int(fields.controlGrantTurn.Int64)
	return nil
}

func (fields *goalCheckpointScanFields) applyCancel(checkpoint *goal.Checkpoint) error {
	if !fields.cancelCause.Valid {
		return nil
	}
	requestedAt, err := parseGoalTimestampValue(fields.cancelRequestedAt, "checkpoint cancel requested_at")
	if err != nil {
		return err
	}
	checkpoint.CompactionCancel = &goal.CompactionCancelIntent{
		PromptID:    strings.TrimSpace(fields.cancelPromptID.String),
		Cause:       strings.TrimSpace(fields.cancelCause.String),
		RequestedAt: requestedAt,
	}
	return nil
}

func (fields *goalCheckpointScanFields) applyReport(checkpoint *goal.Checkpoint) error {
	if !fields.reportStatus.Valid {
		return nil
	}
	recordedAt, err := parseGoalTimestampValue(fields.reportRecordedAt, "checkpoint report recorded_at")
	if err != nil {
		return err
	}
	checkpoint.ReportIntent = &goal.ReportIntent{
		PromptID:     strings.TrimSpace(fields.reportPromptID.String),
		Status:       strings.TrimSpace(fields.reportStatus.String),
		EvidenceRef:  strings.TrimSpace(fields.reportEvidenceRef.String),
		BindingEpoch: fields.reportBindingEpoch.Int64,
		ActorKind:    strings.TrimSpace(fields.reportActorKind.String),
		ActorID:      strings.TrimSpace(fields.reportActorID.String),
		RecordedAt:   recordedAt,
	}
	return nil
}
