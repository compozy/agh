package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/gate"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

func goalSettlementEvidenceRef(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	req goal.CompleteTurnRequest,
) (string, error) {
	if checkpoint.ReportIntent != nil && checkpoint.ReportIntent.PromptID == req.PromptID &&
		checkpoint.ReportIntent.BindingEpoch == req.ExpectedBindingEpoch {
		return strings.TrimSpace(checkpoint.ReportIntent.EvidenceRef), nil
	}
	if checkpoint.JudgeAttemptID == "" {
		return "", nil
	}
	var evidence sql.NullString
	if err := exec.QueryRowContext(
		ctx,
		`SELECT evidence_ref FROM loop_goal_judge_attempts
		 WHERE attempt_id = ? AND loop_run_id = ? AND status = 'completed'`,
		checkpoint.JudgeAttemptID,
		string(req.Key.LoopRunID),
	).Scan(&evidence); err != nil {
		return "", fmt.Errorf("store: load Goal judge evidence for settlement: %w", err)
	}
	if evidence.Valid {
		return strings.TrimSpace(evidence.String), nil
	}
	return "", nil
}

func resolveGoalTurnSettlement(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	req goal.CompleteTurnRequest,
) (goalTurnSettlement, error) {
	settlement := goalTurnSettlement{
		phase:          goalCheckpointPhaseIdle,
		status:         goalStatusActive,
		actorKind:      "system",
		actorID:        "goal-executor",
		brokenStreak:   checkpoint.BrokenStreak,
		recoveryStreak: checkpoint.RecoveryStreak,
	}
	if req.Result.StopReason == looppkg.ActionStopMaxTokens {
		settlement.recoveryStreak++
	}
	if req.Verdict != nil {
		switch req.Verdict.Outcome {
		case gate.VerdictOutcomeError, gate.VerdictOutcomeTimeout, gate.VerdictOutcomeInvalidOutput:
			settlement.brokenStreak++
		default:
			settlement.brokenStreak = 0
		}
	}
	switch {
	case checkpoint.ReportIntent != nil && checkpoint.ReportIntent.PromptID == req.PromptID &&
		checkpoint.ReportIntent.BindingEpoch == req.ExpectedBindingEpoch &&
		checkpoint.ReportIntent.Status == goalReportStatusBlocked:
		settlement.phase = goalCheckpointPhaseTerminal
		settlement.status = goalStatusBlocked
		settlement.cause = looppkg.ReasonCodeGoalReportedBlocked
		settlement.actorKind = checkpoint.ReportIntent.ActorKind
		settlement.actorID = checkpoint.ReportIntent.ActorID
	case req.Verdict != nil && req.Verdict.Outcome == gate.VerdictOutcomeApproved:
		settlement.phase = goalCheckpointPhaseTerminal
		settlement.status = goalStatusComplete
	case req.Verdict != nil && req.Verdict.Outcome == gate.VerdictOutcomeBlocked:
		settlement.phase = goalCheckpointPhaseTerminal
		settlement.status = goalStatusBlocked
		settlement.cause = looppkg.ReasonCodeGoalReportedBlocked
	}
	if settlement.phase != goalCheckpointPhaseTerminal {
		pauseRequested, actorKind, actorID, requestedAt, err := pendingGoalPauseActor(
			ctx,
			exec,
			req.Key.LoopRunID,
		)
		if err != nil {
			return goalTurnSettlement{}, err
		}
		if pauseRequested {
			settlement.phase = goalCheckpointPhaseAwaitingControl
			settlement.status = goalStatusPaused
			settlement.cause = looppkg.ReasonCode(looppkg.TransitionCausePauseBoundary)
			settlement.actorKind = actorKind
			settlement.actorID = actorID
			settlement.pauseRequested = true
			settlement.pauseRequestedAt = requestedAt
		}
	}
	if checkpoint.ControlGrant != nil && !checkpoint.ControlGrant.Consumed &&
		checkpoint.ControlGrant.Kind == goal.ControlGrantBudget &&
		checkpoint.ControlGrant.Turn == checkpoint.TurnsUsed {
		settlement.consumeGrant = true
	}
	return settlement, nil
}

func pendingGoalPauseActor(
	ctx context.Context,
	exec taskSQLExecutor,
	runID looppkg.RunID,
) (bool, string, string, time.Time, error) {
	var pauseRequested int
	var actorKind, actorID sql.NullString
	var requestedAt any
	if err := exec.QueryRowContext(
		ctx,
		`SELECT pause_requested, control_actor_kind, control_actor_id, control_requested_at
		 FROM loop_runs WHERE id = ?`,
		string(runID),
	).Scan(&pauseRequested, &actorKind, &actorID, &requestedAt); err != nil {
		return false, "", "", time.Time{}, fmt.Errorf("store: load Goal pause actor: %w", err)
	}
	if pauseRequested == 0 {
		return false, "", "", time.Time{}, nil
	}
	if !actorKind.Valid || !actorID.Valid || strings.TrimSpace(actorKind.String) == "" ||
		strings.TrimSpace(actorID.String) == "" {
		return false, "", "", time.Time{}, fmt.Errorf(
			"%w: pending Goal pause actor is incomplete",
			looppkg.ErrValidation,
		)
	}
	parsedRequestedAt, err := parseGoalTimestampValue(requestedAt, "loop pause control_requested_at")
	if err != nil {
		return false, "", "", time.Time{}, err
	}
	return true, strings.TrimSpace(actorKind.String), strings.TrimSpace(actorID.String), parsedRequestedAt, nil
}

func updateGoalCheckpointAfterTurn(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	settlement goalTurnSettlement,
	now time.Time,
) error {
	controlActorKind := any(nil)
	controlActorID := any(nil)
	controlRequestedAt := any(nil)
	if settlement.pauseRequested {
		controlActorKind = settlement.actorKind
		controlActorID = settlement.actorID
		controlRequestedAt = store.FormatTimestamp(settlement.pauseRequestedAt)
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET phase = ?, goal_status = ?, control_cause = ?, broken_streak = ?, recovery_streak = ?,
		     queue_entry_id = NULL, prompt_id = NULL, prompt_kind = NULL, prompt_attempt = 0,
		     judge_attempt_id = NULL,
		     report_prompt_id = NULL, report_status = NULL, report_evidence_ref = NULL,
		     report_binding_epoch = NULL, report_actor_kind = NULL, report_actor_id = NULL,
		     report_recorded_at = NULL,
		     control_actor_kind = ?, control_actor_id = ?, control_requested_at = ?,
		     control_grant_consumed = CASE WHEN ? = 1 THEN 1 ELSE control_grant_consumed END,
		     updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND binding_epoch = ?
		   AND task_run_id = ? AND queue_entry_id = ? AND prompt_id = ?
		   AND phase IN ('prompting','judging','persisting')`,
		settlement.phase,
		settlement.status,
		nullableGoalString(string(settlement.cause)),
		settlement.brokenStreak,
		settlement.recoveryStreak,
		controlActorKind,
		controlActorID,
		controlRequestedAt,
		boolToInt(settlement.consumeGrant),
		store.FormatTimestamp(now),
		string(checkpoint.Key.LoopRunID),
		checkpoint.Key.Generation,
		string(checkpoint.Key.NodeID),
		checkpoint.Key.ItemIndex,
		checkpoint.ControlEpoch,
		checkpoint.BindingEpoch,
		checkpoint.TaskRunID,
		checkpoint.QueueEntryID,
		checkpoint.PromptID,
	)
	if err != nil {
		return fmt.Errorf("store: advance Goal checkpoint after turn: %w", err)
	}
	return requireGoalRowsAffected(result, "advance Goal checkpoint after turn")
}

func appendGoalTurnCompletedEvent(
	ctx context.Context,
	exec taskSQLExecutor,
	key goal.TurnKey,
	promptID string,
	at time.Time,
) error {
	turn, err := scanGoalTurn(exec.QueryRowContext(
		ctx,
		`SELECT `+goalTurnSelectColumns+`
		 FROM loop_goal_turns
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ? AND prompt_id = ?`,
		string(key.LoopRunID),
		key.Generation,
		string(key.NodeID),
		key.ItemIndex,
		strings.TrimSpace(promptID),
	), key.WorkspaceID, key.LoopRunID)
	if err != nil {
		return fmt.Errorf("store: load completed Goal turn event payload: %w", err)
	}
	if turn.ResultStatus == nil || turn.EndedAt == nil {
		return fmt.Errorf("%w: Goal turn event requires a terminal row", looppkg.ErrTransitionConflict)
	}
	return appendLoopRunEventWithExecutor(
		ctx,
		exec,
		key.LoopRunID,
		key.WorkspaceID,
		loopRunEventGoalTurnCompleted,
		map[string]any{
			loopRunEventPayloadKeyGeneration:     turn.Key.Generation,
			loopRunEventPayloadKeyNodeID:         string(turn.Key.NodeID),
			loopRunEventPayloadKeyItemIndex:      turn.Key.ItemIndex,
			"turn":                               turn.Turn,
			columnPromptAttempt:                  turn.PromptAttempt,
			loopRunEventPayloadKeyPromptID:       turn.PromptID,
			"seq":                                turn.Seq,
			columnSessionID:                      turn.SessionID,
			"binding_handle":                     turn.BindingHandle,
			columnBindingEpoch:                   turn.BindingEpoch,
			"result_status":                      *turn.ResultStatus,
			loopRunEventPayloadKeyStopReason:     goalTurnEventOptionalStopReason(turn.StopReason),
			"reason_code":                        goalTurnEventOptionalReasonCode(turn.ReasonCode),
			"verdict_outcome":                    goalTurnEventOptionalVerdict(turn.VerdictOutcome),
			loopRunEventPayloadKeyBlockingIssues: turn.BlockingIssues,
			"evidence_ref":                       goalTurnEventOptionalString(turn.EvidenceRef),
			"tokens_used":                        goalTurnEventOptionalInt64(turn.TokensUsed),
			loopRunEventPayloadKeyActorKind:      turn.ActorKind,
			loopRunEventPayloadKeyActorID:        turn.ActorID,
		},
		at,
	)
}

func goalTurnEventOptionalStopReason(value *looppkg.ActionStopReason) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func goalTurnEventOptionalReasonCode(value *looppkg.ReasonCode) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func goalTurnEventOptionalVerdict(value *gate.VerdictOutcome) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func goalTurnEventOptionalString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func goalTurnEventOptionalInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}
