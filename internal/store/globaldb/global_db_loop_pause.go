package globaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
)

// SetLoopRunPauseRequested updates the durable operator pause intent and its authenticated actor.
func (g *GlobalDB) SetLoopRunPauseRequested(
	ctx context.Context,
	ws looppkg.WorkspaceID,
	runID looppkg.RunID,
	requested bool,
	actor taskpkg.ActorContext,
) error {
	if err := g.checkReady(ctx, "set loop run pause requested"); err != nil {
		return err
	}
	if err := actor.Validate(); err != nil {
		return fmt.Errorf("%w: pause actor: %w", looppkg.ErrValidation, err)
	}
	workspaceID := strings.TrimSpace(string(ws))
	trimmedRunID := strings.TrimSpace(string(runID))
	if workspaceID == "" || trimmedRunID == "" {
		return fmt.Errorf("%w: workspace_id and run_id are required", looppkg.ErrValidation)
	}
	now := g.now()
	return g.withTaskImmediateTransaction(ctx, "set loop run pause requested", func(exec taskSQLExecutor) error {
		changed, err := setLoopRunPauseState(
			ctx,
			exec,
			workspaceID,
			trimmedRunID,
			requested,
			actor,
			now,
		)
		if err != nil || !changed {
			return err
		}
		if _, err := exec.ExecContext(
			ctx,
			`UPDATE loop_goal_checkpoints
			 SET control_actor_kind = CASE WHEN ? = 1 THEN ? ELSE NULL END,
			     control_actor_id = CASE WHEN ? = 1 THEN ? ELSE NULL END,
			     control_requested_at = CASE WHEN ? = 1 THEN ? ELSE NULL END,
			     updated_at = ?
			 WHERE loop_run_id = ? AND phase NOT IN ('awaiting_control','terminal')`,
			boolToInt(requested),
			string(actor.Actor.Kind.Normalize()),
			boolToInt(requested),
			strings.TrimSpace(actor.Actor.Ref),
			boolToInt(requested),
			store.FormatTimestamp(now),
			store.FormatTimestamp(now),
			trimmedRunID,
		); err != nil {
			return fmt.Errorf("store: project loop pause intent to Goal checkpoints: %w", err)
		}
		return nil
	})
}

func setLoopRunPauseState(
	ctx context.Context,
	exec taskSQLExecutor,
	workspaceID string,
	runID string,
	requested bool,
	actor taskpkg.ActorContext,
	now time.Time,
) (bool, error) {
	expectedPause := 1
	if requested {
		expectedPause = 0
	}
	result, err := exec.ExecContext(
		ctx,
		`UPDATE loop_runs
		 SET pause_requested = ?,
		     control_actor_kind = CASE WHEN ? = 1 THEN ? ELSE NULL END,
		     control_actor_id = CASE WHEN ? = 1 THEN ? ELSE NULL END,
		     control_requested_at = CASE WHEN ? = 1 THEN ? ELSE NULL END
		 WHERE workspace_id = ? AND id = ? AND status = 'running' AND pause_requested = ?`,
		boolToInt(requested),
		boolToInt(requested),
		string(actor.Actor.Kind.Normalize()),
		boolToInt(requested),
		strings.TrimSpace(actor.Actor.Ref),
		boolToInt(requested),
		store.FormatTimestamp(now),
		workspaceID,
		runID,
		expectedPause,
	)
	if err != nil {
		return false, fmt.Errorf("store: set loop run %q pause_requested: %w", runID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("store: inspect loop run %q pause update: %w", runID, err)
	}
	if rows == 1 {
		return true, nil
	}
	return equivalentLoopPauseState(ctx, exec, workspaceID, runID, requested, actor)
}

func equivalentLoopPauseState(
	ctx context.Context,
	exec taskSQLExecutor,
	workspaceID string,
	runID string,
	requested bool,
	actor taskpkg.ActorContext,
) (bool, error) {
	var status string
	var pauseRequested int
	var actorKind, actorID sql.NullString
	if err := exec.QueryRowContext(
		ctx,
		`SELECT status, pause_requested, control_actor_kind, control_actor_id
		 FROM loop_runs WHERE workspace_id = ? AND id = ?`,
		workspaceID,
		runID,
	).Scan(&status, &pauseRequested, &actorKind, &actorID); err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("%w: %s", looppkg.ErrRunNotFound, runID)
		}
		return false, fmt.Errorf("store: load loop run %q pause state: %w", runID, err)
	}
	if status != string(looppkg.StatusRunning) {
		return false, fmt.Errorf("%w: loop run %q status changed to %q", looppkg.ErrTransitionConflict, runID, status)
	}
	if !requested && pauseRequested == 0 {
		return false, nil
	}
	if requested && pauseRequested == 1 && actorKind.Valid && actorID.Valid &&
		strings.TrimSpace(actorKind.String) == string(actor.Actor.Kind.Normalize()) &&
		strings.TrimSpace(actorID.String) == strings.TrimSpace(actor.Actor.Ref) {
		return false, nil
	}
	return false, fmt.Errorf("%w: loop run %q pause writer changed", looppkg.ErrTransitionConflict, runID)
}
