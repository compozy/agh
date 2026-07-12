package globaldb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/gate"
	"github.com/compozy/agh/internal/loop/goal"
)

const goalJudgeAttemptSelectColumns = `
	attempt_id, loop_run_id, generation, node_id, item_index, turn,
	status, outcome, blocking_json, evidence_ref, tokens_used, usage_base_tokens,
	started_at, completed_at`

func scanGoalJudgeAttempt(scanner rowScanner, workspaceID loop.WorkspaceID) (goal.JudgeAttempt, error) {
	var (
		attempt        goal.JudgeAttempt
		outcome        sql.NullString
		blockingJSON   string
		evidenceRef    sql.NullString
		tokensUsed     sql.NullInt64
		startedAtRaw   any
		completedAtRaw any
	)
	if err := scanner.Scan(
		&attempt.AttemptID,
		&attempt.Key.LoopRunID,
		&attempt.Key.Generation,
		&attempt.Key.NodeID,
		&attempt.Key.ItemIndex,
		&attempt.Turn,
		&attempt.Status,
		&outcome,
		&blockingJSON,
		&evidenceRef,
		&tokensUsed,
		&attempt.UsageBaseTokens,
		&startedAtRaw,
		&completedAtRaw,
	); err != nil {
		return goal.JudgeAttempt{}, err
	}
	attempt.Key.WorkspaceID = workspaceID
	attempt.Outcome = strings.TrimSpace(outcome.String)
	attempt.EvidenceRef = strings.TrimSpace(evidenceRef.String)
	if err := json.Unmarshal([]byte(blockingJSON), &attempt.BlockingIssues); err != nil {
		return goal.JudgeAttempt{}, fmt.Errorf("store: decode goal judge blocking issues: %w", err)
	}
	if attempt.BlockingIssues == nil {
		attempt.BlockingIssues = []gate.BlockingIssue{}
	}
	if tokensUsed.Valid {
		attempt.TokensReported = true
		attempt.TokensUsed = tokensUsed.Int64
	}
	var err error
	attempt.StartedAt, err = parseGoalTimestampValue(startedAtRaw, "judge started_at")
	if err != nil {
		return goal.JudgeAttempt{}, err
	}
	attempt.CompletedAt, err = parseOptionalGoalTimestampValue(completedAtRaw, "judge completed_at")
	if err != nil {
		return goal.JudgeAttempt{}, err
	}
	return attempt, nil
}
