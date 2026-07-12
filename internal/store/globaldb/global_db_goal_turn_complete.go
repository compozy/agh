package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/gate"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

type goalTurnSettlement struct {
	phase            string
	status           string
	cause            looppkg.ReasonCode
	actorKind        string
	actorID          string
	brokenStreak     int
	recoveryStreak   int
	consumeGrant     bool
	pauseRequested   bool
	pauseRequestedAt time.Time
}

// CompleteTurn atomically terminalizes one existing turn, consumes its exact intent, and advances its checkpoint.
func (g *GlobalDB) CompleteTurn(
	ctx context.Context,
	req goal.CompleteTurnRequest,
) (goal.Checkpoint, error) {
	if err := g.checkReady(ctx, "complete Goal turn"); err != nil {
		return goal.Checkpoint{}, err
	}
	if err := validateCompleteGoalTurnRequest(req); err != nil {
		return goal.Checkpoint{}, err
	}
	var completed goal.Checkpoint
	err := g.withTaskImmediateTransaction(ctx, "complete Goal turn", func(exec taskSQLExecutor) error {
		var err error
		completed, err = g.completeGoalTurnWithExecutor(ctx, exec, req)
		return err
	})
	if err != nil {
		return goal.Checkpoint{}, err
	}
	return completed, nil
}

func (g *GlobalDB) completeGoalTurnWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.CompleteTurnRequest,
) (goal.Checkpoint, error) {
	if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
		return goal.Checkpoint{}, err
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return goal.Checkpoint{}, err
	}
	alreadyDone, err := goalTurnAlreadyCompleted(ctx, exec, req)
	if err != nil {
		return goal.Checkpoint{}, err
	}
	if alreadyDone {
		return checkpoint, nil
	}
	if err := validateCompleteGoalTurnOwner(checkpoint, req); err != nil {
		return goal.Checkpoint{}, err
	}
	if err := validateGoalTurnSettlementEffect(ctx, exec, checkpoint, req); err != nil {
		return goal.Checkpoint{}, err
	}
	blockingJSON, err := goalSettlementBlockingJSON(req.Verdict)
	if err != nil {
		return goal.Checkpoint{}, err
	}
	evidenceRef, err := goalSettlementEvidenceRef(ctx, exec, checkpoint, req)
	if err != nil {
		return goal.Checkpoint{}, err
	}
	now := g.now().UTC()
	if err := persistGoalTurnSettlementRow(ctx, exec, req, blockingJSON, evidenceRef, now); err != nil {
		return goal.Checkpoint{}, err
	}
	settlement, err := resolveGoalTurnSettlement(ctx, exec, checkpoint, req)
	if err != nil {
		return goal.Checkpoint{}, err
	}
	if err := updateGoalCheckpointAfterTurn(ctx, exec, checkpoint, settlement, now); err != nil {
		return goal.Checkpoint{}, err
	}
	if err := finalizeGoalTurnSettlement(ctx, exec, checkpoint, settlement, req, now); err != nil {
		return goal.Checkpoint{}, err
	}
	return loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
}

func validateGoalTurnSettlementEffect(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	req goal.CompleteTurnRequest,
) error {
	if err := validateActiveGoalPromptBinding(
		ctx, exec, req.Key, checkpoint.BindingHandle, req.ExpectedBindingEpoch, checkpoint.SessionID,
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
	if row.terminalAt == nil || !goalPromptTerminalMatches(&row, req.Result) {
		return goalControlStaleError("Goal turn queue terminal differs from settlement")
	}
	return nil
}

func finalizeGoalTurnSettlement(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	settlement goalTurnSettlement,
	req goal.CompleteTurnRequest,
	now time.Time,
) error {
	if settlement.phase == goalCheckpointPhaseTerminal {
		if err := closeTerminalGoalCheckpointBinding(ctx, exec, checkpoint, now); err != nil {
			return err
		}
	}
	if settlement.pauseRequested || settlement.phase == goalCheckpointPhaseTerminal {
		if err := clearSettledGoalPause(ctx, exec, req.Key.LoopRunID); err != nil {
			return err
		}
	}
	return finalizeGoalTurnProjection(ctx, exec, checkpoint, settlement, req, now)
}

func persistGoalTurnSettlementRow(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.CompleteTurnRequest,
	blockingJSON []byte,
	evidenceRef string,
	now time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_turns
		 SET result_status = ?, stop_reason = ?, reason_code = ?, verdict_outcome = ?,
		     blocking_json = ?, evidence_ref = ?, tokens_used = ?, ended_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND prompt_id = ? AND result_status IS NULL`,
		string(req.Result.Outcome),
		nullableGoalString(string(req.Result.StopReason)),
		nullableGoalString(string(req.Result.ReasonCode)),
		nullableGoalVerdict(req.Verdict),
		string(blockingJSON),
		nullableGoalString(evidenceRef),
		goalReportedTokens(req.Result.TokensUsed, req.Result.TokensReported),
		store.FormatTimestamp(now),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		strings.TrimSpace(req.PromptID),
	)
	if err != nil {
		return fmt.Errorf("store: complete Goal turn row: %w", err)
	}
	return requireGoalRowsAffected(result, "complete Goal turn row")
}

func clearSettledGoalPause(ctx context.Context, exec taskSQLExecutor, runID looppkg.RunID) error {
	_, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET pause_requested = 0, control_actor_kind = NULL, control_actor_id = NULL,
		     control_requested_at = NULL
		 WHERE id = ? AND pause_requested = 1`,
		string(runID),
	)
	if err != nil {
		return fmt.Errorf("store: clear settled Goal pause request: %w", err)
	}
	return nil
}

func finalizeGoalTurnProjection(
	ctx context.Context,
	exec taskSQLExecutor,
	checkpoint goal.Checkpoint,
	settlement goalTurnSettlement,
	req goal.CompleteTurnRequest,
	now time.Time,
) error {
	if err := projectGoalCheckpointCounts(
		ctx, exec, req.Key, settlement.status, checkpoint.TurnsUsed, checkpoint.TurnLimit,
	); err != nil {
		return err
	}
	if err := appendGoalTurnCompletedEvent(ctx, exec, req.Key, req.PromptID, now); err != nil {
		return err
	}
	if err := appendGoalStatusChangedEvent(
		ctx,
		exec,
		req.Key,
		checkpoint.Status,
		settlement.status,
		settlement.cause,
		settlement.actorKind,
		settlement.actorID,
		now,
	); err != nil {
		return err
	}
	return nil
}

func validateCompleteGoalTurnRequest(req goal.CompleteTurnRequest) error {
	if err := req.Key.Validate(); err != nil {
		return err
	}
	if req.ExpectedControlEpoch < 1 || req.ExpectedBindingEpoch < 1 ||
		strings.TrimSpace(req.TaskRunID) == "" || strings.TrimSpace(req.QueueEntryID) == "" ||
		strings.TrimSpace(req.PromptID) == "" ||
		strings.TrimSpace(req.Result.PromptID) != strings.TrimSpace(req.PromptID) ||
		strings.TrimSpace(req.DispatchActorKind) == "" || strings.TrimSpace(req.DispatchActorID) == "" {
		return fmt.Errorf("%w: Goal turn settlement identity is incomplete", looppkg.ErrValidation)
	}
	if err := validateGoalPromptResult(req.Result); err != nil {
		return err
	}
	switch req.Result.Outcome {
	case looppkg.ActionPromptOutcomeCompleted, looppkg.ActionPromptOutcomeInvalidResult,
		looppkg.ActionPromptOutcomeFailed, looppkg.ActionPromptOutcomeAmbiguous:
	default:
		return fmt.Errorf("%w: Goal turn cannot settle outcome %q", looppkg.ErrValidation, req.Result.Outcome)
	}
	if req.Verdict != nil && !goalVerdictOutcomeValid(req.Verdict.Outcome) {
		return fmt.Errorf("%w: Goal turn verdict outcome %q is invalid", looppkg.ErrValidation, req.Verdict.Outcome)
	}
	return nil
}

func validateCompleteGoalTurnOwner(checkpoint goal.Checkpoint, req goal.CompleteTurnRequest) error {
	if checkpoint.ControlEpoch != req.ExpectedControlEpoch ||
		checkpoint.BindingEpoch != req.ExpectedBindingEpoch ||
		checkpoint.TaskRunID != strings.TrimSpace(req.TaskRunID) ||
		checkpoint.QueueEntryID != strings.TrimSpace(req.QueueEntryID) ||
		checkpoint.PromptID != strings.TrimSpace(req.PromptID) ||
		(checkpoint.Phase != goalCheckpointPhasePrompting &&
			checkpoint.Phase != goalCheckpointPhaseJudging &&
			checkpoint.Phase != goalCheckpointPhasePersisting) {
		return goalControlStaleError("Goal turn settlement owner changed")
	}
	return nil
}

func goalTurnAlreadyCompleted(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.CompleteTurnRequest,
) (bool, error) {
	var resultStatus, stopReason, reasonCode, verdictOutcome sql.NullString
	var blockingJSON, actorKind, actorID string
	var tokensUsed sql.NullInt64
	var bindingEpoch int64
	err := exec.QueryRowContext(
		ctx,
		`SELECT result_status, stop_reason, reason_code, verdict_outcome, blocking_json,
		        tokens_used, actor_kind, actor_id, binding_epoch
		 FROM loop_goal_turns
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ? AND prompt_id = ?`,
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		strings.TrimSpace(req.PromptID),
	).Scan(
		&resultStatus,
		&stopReason,
		&reasonCode,
		&verdictOutcome,
		&blockingJSON,
		&tokensUsed,
		&actorKind,
		&actorID,
		&bindingEpoch,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return false, goal.ErrTurnNotFound
	}
	if err != nil {
		return false, fmt.Errorf("store: inspect Goal turn settlement: %w", err)
	}
	if !resultStatus.Valid {
		return false, nil
	}
	wantBlocking, err := goalSettlementBlockingJSON(req.Verdict)
	if err != nil {
		return false, err
	}
	wantVerdict := ""
	if req.Verdict != nil {
		wantVerdict = string(req.Verdict.Outcome)
	}
	matches := resultStatus.String == string(req.Result.Outcome) &&
		goalOptionalStringMatches(stopReason, string(req.Result.StopReason)) &&
		goalOptionalStringMatches(reasonCode, string(req.Result.ReasonCode)) &&
		goalOptionalStringMatches(verdictOutcome, wantVerdict) &&
		blockingJSON == string(wantBlocking) &&
		goalOptionalTokensMatch(tokensUsed, req.Result.TokensUsed, req.Result.TokensReported) &&
		actorKind == strings.TrimSpace(req.DispatchActorKind) &&
		actorID == strings.TrimSpace(req.DispatchActorID) &&
		bindingEpoch == req.ExpectedBindingEpoch
	if !matches {
		return false, goalControlStaleError("Goal turn already has a different settlement")
	}
	prompt, err := loadGoalPromptRow(ctx, exec, req.Key.LoopRunID, req.PromptID)
	if err != nil {
		return false, err
	}
	if err := requireGoalPromptOwner(
		&prompt,
		req.Key,
		req.TaskRunID,
		req.QueueEntryID,
		req.ExpectedBindingEpoch,
	); err != nil {
		return false, err
	}
	if !prompt.ownerEpoch.Valid || prompt.ownerEpoch.Int64 != req.ExpectedControlEpoch {
		return false, goalControlStaleError("Goal turn settlement control epoch changed")
	}
	return true, nil
}

func goalOptionalStringMatches(stored sql.NullString, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return !stored.Valid || strings.TrimSpace(stored.String) == ""
	}
	return stored.Valid && stored.String == want
}

func goalOptionalTokensMatch(stored sql.NullInt64, want int64, reported bool) bool {
	if !reported {
		return !stored.Valid
	}
	return stored.Valid && stored.Int64 == want
}

func goalSettlementBlockingJSON(verdict *gate.Verdict) ([]byte, error) {
	issues := []gate.BlockingIssue{}
	if verdict != nil {
		issues = append(issues, verdict.BlockingIssues...)
	}
	data, err := json.Marshal(issues)
	if err != nil {
		return nil, fmt.Errorf("store: encode Goal turn blockers: %w", err)
	}
	return data, nil
}

func nullableGoalVerdict(verdict *gate.Verdict) any {
	if verdict == nil {
		return nil
	}
	return string(verdict.Outcome)
}

func goalVerdictOutcomeValid(outcome gate.VerdictOutcome) bool {
	switch outcome {
	case gate.VerdictOutcomeApproved, gate.VerdictOutcomeRejected, gate.VerdictOutcomeBlocked,
		gate.VerdictOutcomeError, gate.VerdictOutcomeTimeout, gate.VerdictOutcomeInvalidOutput:
		return true
	default:
		return false
	}
}
