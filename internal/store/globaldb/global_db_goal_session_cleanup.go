package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	looppkg "github.com/compozy/agh/internal/loop"
	"github.com/compozy/agh/internal/loop/goal"
	"github.com/compozy/agh/internal/store"
)

var _ goal.SessionCleanupStore = (*GlobalDB)(nil)

const goalSessionCleanupSelectColumns = `
	id, cleanup_id, workspace_id, loop_run_id, handle, binding_epoch,
	session_id, cause, created_at, completed_at`

const goalSessionCleanupClaimColumns = `
	cleanup.id, cleanup.cleanup_id, cleanup.workspace_id, cleanup.loop_run_id,
	cleanup.handle, cleanup.binding_epoch, cleanup.session_id, cleanup.cause,
	cleanup.created_at, cleanup.completed_at`

// ClaimGoalSessionCleanup lists the oldest unacknowledged run-owned session cleanup effects.
func (g *GlobalDB) ClaimGoalSessionCleanup(
	ctx context.Context,
	limit int,
) (obligations []goal.SessionCleanupObligation, err error) {
	if err := g.checkReady(ctx, "claim Goal session cleanup"); err != nil {
		return nil, err
	}
	if limit < 0 || limit > 200 {
		return nil, fmt.Errorf("%w: Goal cleanup claim limit must be between 0 and 200", looppkg.ErrValidation)
	}
	if limit == 0 {
		limit = 50
	}
	rows, err := g.db.QueryContext(
		ctx,
		`SELECT `+goalSessionCleanupClaimColumns+`
		 FROM loop_goal_session_cleanup AS cleanup
		 JOIN loop_session_bindings AS binding
		   ON binding.loop_run_id = cleanup.loop_run_id AND binding.handle = cleanup.handle
		  AND binding.binding_epoch = cleanup.binding_epoch
		 WHERE cleanup.completed_at IS NULL
		   AND NOT (binding.state = 'failed' AND binding.failure_code = ?)
		 ORDER BY cleanup.id ASC LIMIT ?`,
		goalBindingFailureStopCreationUnsettled,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: claim Goal session cleanup: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("store: close Goal session cleanup rows: %w", closeErr))
		}
	}()
	obligations = make([]goal.SessionCleanupObligation, 0, limit)
	for rows.Next() {
		obligation, scanErr := scanGoalSessionCleanup(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("store: scan Goal session cleanup: %w", scanErr)
		}
		obligations = append(obligations, obligation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate Goal session cleanup: %w", err)
	}
	return obligations, nil
}

// ReconcileGoalSessionCleanup releases Stop/create races left in-flight by a previous daemon process.
func (g *GlobalDB) ReconcileGoalSessionCleanup(ctx context.Context) error {
	if err := g.checkReady(ctx, "reconcile Goal session cleanup"); err != nil {
		return err
	}
	_, err := g.db.ExecContext(
		ctx,
		`UPDATE loop_session_bindings
		 SET failure_code = ?
		 WHERE state = 'failed' AND failure_code = ?
		   AND EXISTS (
		     SELECT 1 FROM loop_goal_session_cleanup AS cleanup
		     WHERE cleanup.loop_run_id = loop_session_bindings.loop_run_id
		       AND cleanup.handle = loop_session_bindings.handle
		       AND cleanup.binding_epoch = loop_session_bindings.binding_epoch
		       AND cleanup.completed_at IS NULL
		   )`,
		goalBindingFailureStopCreationSettled,
		goalBindingFailureStopCreationUnsettled,
	)
	if err != nil {
		return fmt.Errorf("store: reconcile stopped Goal binding creation: %w", err)
	}
	return nil
}

// AcknowledgeGoalSessionCleanup records the first successful idempotent session Stop.
func (g *GlobalDB) AcknowledgeGoalSessionCleanup(
	ctx context.Context,
	cleanupID string,
	completedAt time.Time,
) error {
	if err := g.checkReady(ctx, "acknowledge Goal session cleanup"); err != nil {
		return err
	}
	cleanupID = strings.TrimSpace(cleanupID)
	if cleanupID == "" || completedAt.IsZero() {
		return fmt.Errorf("%w: Goal cleanup acknowledgement is incomplete", looppkg.ErrValidation)
	}
	completedAt = completedAt.UTC()
	return g.withImmediateTransaction(ctx, "acknowledge Goal session cleanup", func(exec globalSQLExecutor) error {
		obligation, err := loadGoalSessionCleanup(ctx, exec, cleanupID)
		if err != nil {
			return err
		}
		if obligation.CompletedAt != nil {
			return nil
		}
		if completedAt.Before(obligation.CreatedAt) {
			return fmt.Errorf("%w: Goal cleanup completion precedes creation", looppkg.ErrValidation)
		}
		result, err := exec.ExecContext(
			ctx,
			`UPDATE loop_goal_session_cleanup SET completed_at = ?
			 WHERE cleanup_id = ? AND completed_at IS NULL`,
			store.FormatTimestamp(completedAt),
			cleanupID,
		)
		if err != nil {
			return fmt.Errorf("store: acknowledge Goal session cleanup %q: %w", cleanupID, err)
		}
		return requireGoalRowsAffected(result, "acknowledge Goal session cleanup")
	})
}

func enqueueGoalSessionCleanupWithExecutor(
	ctx context.Context,
	exec taskSQLExecutor,
	binding goal.SessionBinding,
	cause goal.SessionCleanupCause,
	createdAt time.Time,
) error {
	if binding.Ownership != goal.BindingOwnershipRunOwned {
		return nil
	}
	obligation := goal.SessionCleanupObligation{
		CleanupID:    goalSessionCleanupID(binding),
		WorkspaceID:  binding.Key.WorkspaceID,
		LoopRunID:    binding.Key.LoopRunID,
		Handle:       binding.Key.Handle,
		BindingEpoch: binding.BindingEpoch,
		SessionID:    binding.SessionID,
		Cause:        cause,
		CreatedAt:    createdAt.UTC(),
	}
	if err := obligation.Validate(); err != nil {
		return err
	}
	if _, err := exec.ExecContext(
		ctx,
		`INSERT INTO loop_goal_session_cleanup (
			cleanup_id, workspace_id, loop_run_id, handle, binding_epoch, session_id, cause, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(cleanup_id) DO NOTHING`,
		obligation.CleanupID,
		string(obligation.WorkspaceID),
		string(obligation.LoopRunID),
		obligation.Handle,
		obligation.BindingEpoch,
		obligation.SessionID,
		string(obligation.Cause),
		store.FormatTimestamp(obligation.CreatedAt),
	); err != nil {
		return fmt.Errorf("store: enqueue Goal session cleanup %q: %w", obligation.CleanupID, err)
	}
	persisted, err := loadGoalSessionCleanup(ctx, exec, obligation.CleanupID)
	if err != nil {
		return err
	}
	if !goalSessionCleanupMatches(persisted, obligation) {
		return fmt.Errorf(
			"%w: Goal session cleanup %q payload changed",
			looppkg.ErrTransitionConflict,
			obligation.CleanupID,
		)
	}
	return nil
}

func goalSessionCleanupID(binding goal.SessionBinding) string {
	return fmt.Sprintf(
		"goal-session-cleanup:%s:%s:%d",
		binding.Key.LoopRunID,
		binding.Key.Handle,
		binding.BindingEpoch,
	)
}

func loadGoalSessionCleanup(
	ctx context.Context,
	exec globalSQLExecutor,
	cleanupID string,
) (goal.SessionCleanupObligation, error) {
	obligation, err := scanGoalSessionCleanup(exec.QueryRowContext(
		ctx,
		`SELECT `+goalSessionCleanupSelectColumns+`
		 FROM loop_goal_session_cleanup WHERE cleanup_id = ?`,
		strings.TrimSpace(cleanupID),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return goal.SessionCleanupObligation{}, fmt.Errorf(
			"%w: Goal session cleanup %q",
			looppkg.ErrTransitionConflict,
			cleanupID,
		)
	}
	if err != nil {
		return goal.SessionCleanupObligation{}, fmt.Errorf("store: load Goal session cleanup %q: %w", cleanupID, err)
	}
	return obligation, nil
}

func scanGoalSessionCleanup(scanner rowScanner) (goal.SessionCleanupObligation, error) {
	var obligation goal.SessionCleanupObligation
	var createdAtRaw, completedAtRaw any
	if err := scanner.Scan(
		&obligation.ID,
		&obligation.CleanupID,
		&obligation.WorkspaceID,
		&obligation.LoopRunID,
		&obligation.Handle,
		&obligation.BindingEpoch,
		&obligation.SessionID,
		&obligation.Cause,
		&createdAtRaw,
		&completedAtRaw,
	); err != nil {
		return goal.SessionCleanupObligation{}, err
	}
	var err error
	obligation.CreatedAt, err = parseGoalTimestampValue(createdAtRaw, "session cleanup created_at")
	if err != nil {
		return goal.SessionCleanupObligation{}, err
	}
	obligation.CompletedAt, err = parseOptionalGoalTimestampValue(completedAtRaw, "session cleanup completed_at")
	if err != nil {
		return goal.SessionCleanupObligation{}, err
	}
	if err := obligation.Validate(); err != nil {
		return goal.SessionCleanupObligation{}, err
	}
	return obligation, nil
}

func goalSessionCleanupMatches(
	persisted goal.SessionCleanupObligation,
	want goal.SessionCleanupObligation,
) bool {
	return persisted.CleanupID == want.CleanupID && persisted.WorkspaceID == want.WorkspaceID &&
		persisted.LoopRunID == want.LoopRunID && persisted.Handle == want.Handle &&
		persisted.BindingEpoch == want.BindingEpoch && persisted.SessionID == want.SessionID &&
		persisted.Cause == want.Cause && persisted.CreatedAt.Equal(want.CreatedAt)
}
