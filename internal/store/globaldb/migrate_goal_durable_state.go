package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

var goalDurableStateMigration = store.Migration{
	Version:  74,
	Name:     "add_goal_durable_state",
	Up:       migrateGoalDurableState,
	Checksum: "2026-07-10-add-goal-durable-state",
}

func migrateGoalDurableState(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range goalDurableTableStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: create goal durable table: %w", err)
		}
	}
	if err := addGoalDurableColumns(ctx, tx); err != nil {
		return err
	}
	for _, statement := range goalDurableIndexStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: create goal durable index: %w", err)
		}
	}
	return nil
}

func goalDurableTableStatements() []string {
	return []string{
		loopGoalTurnsTableStatement,
		loopGoalCheckpointsTableStatement,
		loopGoalJudgeAttemptsTableStatement,
		loopSessionBindingsTableStatement,
		loopGoalSessionOutboxTableStatement,
	}
}

func goalDurableIndexStatements() []string {
	return []string{
		`CREATE UNIQUE INDEX uq_loop_session_bindings_active
			ON loop_session_bindings(loop_run_id, handle) WHERE state='active'`,
		`CREATE UNIQUE INDEX uq_loop_runs_active_session_goal ON loop_runs(origin_session_id)
			WHERE origin_kind='session'
			  AND status IN ('queued','running','watching','needs-approval','paused')`,
		`CREATE INDEX idx_session_input_queue_goal_owner
			ON session_input_queue(loop_run_id, task_run_id, owner_epoch, status, dispatchable, fence_kind)`,
		`CREATE UNIQUE INDEX uq_session_input_queue_goal_prompt
			ON session_input_queue(loop_run_id, prompt_id)
			WHERE prompt_id IS NOT NULL`,
		`CREATE INDEX idx_loop_goal_session_outbox_pending
			ON loop_goal_session_outbox(id) WHERE delivered_at IS NULL`,
	}
}
