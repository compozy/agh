package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

var _ goal.JudgeStore = (*GlobalDB)(nil)

// BeginJudgeAttempt persists one aggregate judge attempt before evaluator effects.
func (g *GlobalDB) BeginJudgeAttempt(
	ctx context.Context,
	req goal.BeginJudgeAttemptRequest,
) (goal.JudgeAttempt, error) {
	if err := g.checkReady(ctx, "begin goal judge attempt"); err != nil {
		return goal.JudgeAttempt{}, err
	}
	if err := validateBeginJudgeAttemptRequest(req); err != nil {
		return goal.JudgeAttempt{}, err
	}
	now := g.now()
	var attempt goal.JudgeAttempt
	err := g.withTaskImmediateTransaction(ctx, "begin goal judge attempt", func(exec taskSQLExecutor) error {
		var err error
		attempt, err = beginJudgeAttemptWithExecutor(ctx, exec, req, now)
		return err
	})
	if err != nil {
		return goal.JudgeAttempt{}, err
	}
	return attempt, nil
}

func beginJudgeAttemptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.BeginJudgeAttemptRequest,
	now time.Time,
) (goal.JudgeAttempt, error) {
	if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
		return goal.JudgeAttempt{}, err
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return goal.JudgeAttempt{}, err
	}
	if strings.TrimSpace(checkpoint.ControlActorKind) != "" ||
		strings.TrimSpace(checkpoint.ControlActorID) != "" {
		return goal.JudgeAttempt{}, goalPromptFencedError("Goal pause intent won before judge start")
	}
	if err := validateJudgeCheckpointOwner(checkpoint, req.ExpectedControlEpoch,
		req.ExpectedBindingEpoch, req.TaskRunID, req.PromptID); err != nil {
		return goal.JudgeAttempt{}, err
	}
	if err := validateActiveGoalPromptBinding(
		ctx,
		exec,
		req.Key,
		checkpoint.BindingHandle,
		req.ExpectedBindingEpoch,
		checkpoint.SessionID,
	); err != nil {
		return goal.JudgeAttempt{}, err
	}
	if err := validateGoalBudgetDecision(ctx, exec, req.Key.LoopRunID, req.BudgetDecision); err != nil {
		return goal.JudgeAttempt{}, err
	}
	existing, found, digest, err := findGoalJudgeAttemptWithDigest(ctx, exec, req.Key, req.AttemptID)
	if err != nil {
		return goal.JudgeAttempt{}, err
	}
	if found {
		if digest != strings.TrimSpace(req.JudgeDigest) || existing.Turn != req.Turn ||
			existing.UsageBaseTokens != req.UsageBaseTokens {
			return goal.JudgeAttempt{}, goalControlStaleError(
				"judge attempt identity differs from persisted attempt",
			)
		}
		return existing, nil
	}
	if err := insertGoalJudgeAttempt(ctx, exec, req, now); err != nil {
		return goal.JudgeAttempt{}, err
	}
	if err := attachGoalJudgeAttempt(ctx, exec, req, now); err != nil {
		return goal.JudgeAttempt{}, err
	}
	return getGoalJudgeAttemptWithExecutor(ctx, exec, req.Key, req.AttemptID)
}

func insertGoalJudgeAttempt(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.BeginJudgeAttemptRequest,
	now time.Time,
) error {
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO loop_goal_judge_attempts (
			attempt_id, loop_run_id, generation, node_id, item_index, turn,
			judge_digest, status, blocking_json, usage_base_tokens, started_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'running', '[]', ?, ?)`,
		strings.TrimSpace(req.AttemptID),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		req.Turn,
		strings.TrimSpace(req.JudgeDigest),
		req.UsageBaseTokens,
		store.FormatTimestamp(now),
	)
	if err != nil {
		return fmt.Errorf("store: insert goal judge attempt: %w", err)
	}
	return nil
}

func attachGoalJudgeAttempt(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.BeginJudgeAttemptRequest,
	now time.Time,
) error {
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_checkpoints
		 SET judge_attempt_id = ?, updated_at = ?
		 WHERE loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?
		   AND control_epoch = ? AND binding_epoch = ? AND phase = 'judging'
		   AND task_run_id = ? AND prompt_id = ?
		   AND (judge_attempt_id IS NULL OR judge_attempt_id = ?)`,
		strings.TrimSpace(req.AttemptID),
		store.FormatTimestamp(now),
		string(req.Key.LoopRunID),
		req.Key.Generation,
		string(req.Key.NodeID),
		req.Key.ItemIndex,
		req.ExpectedControlEpoch,
		req.ExpectedBindingEpoch,
		strings.TrimSpace(req.TaskRunID),
		strings.TrimSpace(req.PromptID),
		strings.TrimSpace(req.AttemptID),
	)
	if err != nil {
		return fmt.Errorf("store: attach goal judge attempt to checkpoint: %w", err)
	}
	return requireGoalRowsAffected(result, "attach goal judge attempt")
}

// CompleteJudgeAttempt records the evaluator terminal exactly once.
func (g *GlobalDB) CompleteJudgeAttempt(
	ctx context.Context,
	req goal.CompleteJudgeAttemptRequest,
) (goal.JudgeAttempt, error) {
	if err := g.checkReady(ctx, "complete goal judge attempt"); err != nil {
		return goal.JudgeAttempt{}, err
	}
	if err := validateCompleteJudgeAttemptRequest(req); err != nil {
		return goal.JudgeAttempt{}, err
	}
	now := g.now()
	var completed goal.JudgeAttempt
	err := g.withTaskImmediateTransaction(ctx, "complete goal judge attempt", func(exec taskSQLExecutor) error {
		var err error
		completed, err = completeJudgeAttemptWithExecutor(ctx, exec, req, now)
		return err
	})
	if err != nil {
		return goal.JudgeAttempt{}, err
	}
	return completed, nil
}

func completeJudgeAttemptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	req goal.CompleteJudgeAttemptRequest,
	now time.Time,
) (goal.JudgeAttempt, error) {
	if err := validateGoalRunWorkspace(ctx, exec, req.Key); err != nil {
		return goal.JudgeAttempt{}, err
	}
	checkpoint, err := loadGoalCheckpointWithExecutor(ctx, exec, req.Key)
	if err != nil {
		return goal.JudgeAttempt{}, err
	}
	if err := validateJudgeCheckpointOwner(
		checkpoint,
		req.ExpectedControlEpoch,
		req.ExpectedBindingEpoch,
		req.TaskRunID,
		req.PromptID,
	); err != nil {
		return goal.JudgeAttempt{}, err
	}
	if err := validateActiveGoalPromptBinding(
		ctx,
		exec,
		req.Key,
		checkpoint.BindingHandle,
		req.ExpectedBindingEpoch,
		checkpoint.SessionID,
	); err != nil {
		return goal.JudgeAttempt{}, err
	}
	attempt, err := getGoalJudgeAttemptWithExecutor(ctx, exec, req.Key, req.AttemptID)
	if err != nil {
		return goal.JudgeAttempt{}, err
	}
	blockingJSON, err := json.Marshal(req.Verdict.BlockingIssues)
	if err != nil {
		return goal.JudgeAttempt{}, fmt.Errorf("store: encode goal judge blocking issues: %w", err)
	}
	evidenceRef, err := persistGoalVerdictEvidence(ctx, exec, req.Verdict, now)
	if err != nil {
		return goal.JudgeAttempt{}, err
	}
	if attempt.Status == goalJudgeStatusCompleted {
		if judgeAttemptMatchesCompletion(attempt, req, evidenceRef) {
			return attempt, nil
		}
		return goal.JudgeAttempt{}, goalControlStaleError(
			"judge attempt already completed with different terminal",
		)
	}
	if attempt.Status != goalJudgeStatusRunning {
		return goal.JudgeAttempt{}, goalControlStaleError("judge attempt is not running")
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_goal_judge_attempts
		 SET status = 'completed', outcome = ?, blocking_json = ?, evidence_ref = ?,
		     tokens_used = ?, completed_at = ?
		 WHERE attempt_id = ? AND loop_run_id = ? AND status = 'running'`,
		string(req.Verdict.Outcome),
		string(blockingJSON),
		nullableGoalString(evidenceRef),
		goalReportedTokens(req.TokensUsed, req.TokensReported),
		store.FormatTimestamp(now),
		strings.TrimSpace(req.AttemptID),
		string(req.Key.LoopRunID),
	)
	if err != nil {
		return goal.JudgeAttempt{}, fmt.Errorf("store: complete goal judge attempt: %w", err)
	}
	if err := requireGoalRowsAffected(result, "complete goal judge attempt"); err != nil {
		return goal.JudgeAttempt{}, err
	}
	return getGoalJudgeAttemptWithExecutor(ctx, exec, req.Key, req.AttemptID)
}

// GetJudgeAttempt returns one workspace-owned judge attempt.
func (g *GlobalDB) GetJudgeAttempt(
	ctx context.Context,
	key goal.TurnKey,
	attemptID string,
) (goal.JudgeAttempt, error) {
	if err := g.checkReady(ctx, "get goal judge attempt"); err != nil {
		return goal.JudgeAttempt{}, err
	}
	if err := validateGoalRunWorkspace(ctx, g.db, key); err != nil {
		return goal.JudgeAttempt{}, err
	}
	return getGoalJudgeAttemptWithExecutor(ctx, g.db, key, attemptID)
}

func getGoalJudgeAttemptWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	key goal.TurnKey,
	attemptID string,
) (goal.JudgeAttempt, error) {
	attempt, err := scanGoalJudgeAttempt(exec.QueryRowContext(
		ctx,
		`SELECT `+goalJudgeAttemptSelectColumns+`
		 FROM loop_goal_judge_attempts
		 WHERE attempt_id = ? AND loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?`,
		strings.TrimSpace(attemptID),
		string(key.LoopRunID),
		key.Generation,
		string(key.NodeID),
		key.ItemIndex,
	), key.WorkspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return goal.JudgeAttempt{}, fmt.Errorf("%w: judge attempt %s", goal.ErrTurnNotFound, attemptID)
	}
	if err != nil {
		return goal.JudgeAttempt{}, fmt.Errorf("store: scan goal judge attempt: %w", err)
	}
	return attempt, nil
}

func findGoalJudgeAttemptWithDigest(
	ctx context.Context,
	exec taskSQLExecutor,
	key goal.TurnKey,
	attemptID string,
) (goal.JudgeAttempt, bool, string, error) {
	var digest string
	err := exec.QueryRowContext(
		ctx,
		`SELECT judge_digest
		 FROM loop_goal_judge_attempts
		 WHERE attempt_id = ? AND loop_run_id = ? AND generation = ? AND node_id = ? AND item_index = ?`,
		strings.TrimSpace(attemptID), string(key.LoopRunID), key.Generation, string(key.NodeID), key.ItemIndex,
	).Scan(&digest)
	if errors.Is(err, sql.ErrNoRows) {
		return goal.JudgeAttempt{}, false, "", nil
	}
	if err != nil {
		return goal.JudgeAttempt{}, false, "", err
	}
	attempt, err := getGoalJudgeAttemptWithExecutor(ctx, exec, key, attemptID)
	if err != nil {
		return goal.JudgeAttempt{}, false, "", err
	}
	return attempt, true, digest, nil
}

func goalReportedTokens(tokens int64, reported bool) any {
	if !reported {
		return nil
	}
	return tokens
}
