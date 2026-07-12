package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
)

var _ goal.SessionProjectionReader = (*GlobalDB)(nil)

// GetSessionGoalProjection selects the newest session-origin Run before applying its clear tombstone.
func (g *GlobalDB) GetSessionGoalProjection(
	ctx context.Context,
	workspaceID looppkg.WorkspaceID,
	sessionID string,
) (projection goal.SessionProjection, err error) {
	if err := g.checkReady(ctx, "get session Goal projection"); err != nil {
		return goal.SessionProjection{}, err
	}
	workspaceID = looppkg.WorkspaceID(strings.TrimSpace(string(workspaceID)))
	sessionID = strings.TrimSpace(sessionID)
	if workspaceID == "" || sessionID == "" {
		return goal.SessionProjection{}, fmt.Errorf(
			"%w: Goal workspace_id and session_id are required",
			looppkg.ErrValidation,
		)
	}
	tx, err := g.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return goal.SessionProjection{}, fmt.Errorf("store: begin session Goal projection read: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			joinCleanupError(&err, rollbackTx(tx, "session Goal projection read"))
		}
	}()

	projection, err = readSessionGoalProjection(ctx, tx, workspaceID, sessionID)
	if err != nil {
		return goal.SessionProjection{}, err
	}
	if err = tx.Commit(); err != nil {
		return goal.SessionProjection{}, fmt.Errorf("store: commit session Goal projection read: %w", err)
	}
	committed = true
	return projection, nil
}

func readSessionGoalProjection(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID looppkg.WorkspaceID,
	sessionID string,
) (goal.SessionProjection, error) {
	var projection goal.SessionProjection
	var runID string
	var runStatus string
	var clearedAt sql.NullString
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, status, definition_digest, origin_session_id, goal_context_nudge_ratio, goal_cleared_at
		 FROM loop_runs
		 WHERE workspace_id = ? AND origin_kind = 'session' AND origin_session_id = ?
		 ORDER BY created_at DESC, rowid DESC
		 LIMIT 1`,
		string(workspaceID),
		sessionID,
	).Scan(
		&runID,
		&runStatus,
		&projection.DefinitionDigest,
		&projection.OriginSessionID,
		&projection.ContextNudgeRatio,
		&clearedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return goal.SessionProjection{}, nil
	}
	if err != nil {
		return goal.SessionProjection{}, fmt.Errorf("store: select newest session Goal: %w", err)
	}
	projection.Found = true
	projection.RunID = looppkg.RunID(strings.TrimSpace(runID))
	projection.RunStatus = looppkg.Status(strings.TrimSpace(runStatus))
	projection.DefinitionDigest = strings.TrimSpace(projection.DefinitionDigest)
	projection.OriginSessionID = strings.TrimSpace(projection.OriginSessionID)
	projection.Cleared = clearedAt.Valid
	if !projection.RunStatus.Valid() {
		return goal.SessionProjection{}, fmt.Errorf(
			"%w: session Goal run status is invalid: %q",
			looppkg.ErrValidation,
			runStatus,
		)
	}
	if projection.Cleared {
		return projection, nil
	}

	projection.Checkpoint, err = readLatestSessionGoalCheckpoint(ctx, tx, workspaceID, projection.RunID)
	if err != nil {
		return goal.SessionProjection{}, err
	}
	projection.LastVerdict, err = readLatestSessionGoalVerdict(ctx, tx, workspaceID, projection.RunID)
	if err != nil {
		return goal.SessionProjection{}, err
	}
	return projection, nil
}

func readLatestSessionGoalCheckpoint(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID looppkg.WorkspaceID,
	runID looppkg.RunID,
) (*goal.Checkpoint, error) {
	checkpoint, err := scanGoalCheckpoint(tx.QueryRowContext(
		ctx,
		`SELECT `+goalCheckpointSelectColumns+`
		 FROM loop_goal_checkpoints
		 WHERE loop_run_id = ?
		 ORDER BY generation DESC, updated_at DESC, node_id ASC, item_index ASC
		 LIMIT 1`,
		string(runID),
	), workspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: select session Goal checkpoint: %w", err)
	}
	return &checkpoint, nil
}

func readLatestSessionGoalVerdict(
	ctx context.Context,
	tx *sql.Tx,
	workspaceID looppkg.WorkspaceID,
	runID looppkg.RunID,
) (*goal.Turn, error) {
	verdict, err := scanGoalTurn(tx.QueryRowContext(
		ctx,
		`SELECT `+goalTurnSelectColumns+`
		 FROM loop_goal_turns
		 WHERE loop_run_id = ? AND verdict_outcome IS NOT NULL
		 ORDER BY seq DESC
		 LIMIT 1`,
		string(runID),
	), workspaceID, runID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: select session Goal verdict: %w", err)
	}
	return &verdict, nil
}
