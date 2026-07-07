package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateAutomationCatchUpContract(ctx context.Context, conn *sql.Conn) (err error) {
	cleanupCtx := context.WithoutCancel(ctx)
	foreignKeysDisabled := false
	defer func() {
		if foreignKeysDisabled {
			joinCleanupError(&err, restoreForeignKeys(cleanupCtx, conn))
		}
	}()
	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("store: disable foreign keys for automation catch-up migration: %w", err)
	}
	foreignKeysDisabled = true

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin automation catch-up migration: %w", err)
	}
	defer func() {
		joinCleanupError(&err, rollbackTx(tx, "automation catch-up migration"))
	}()

	if err := addAutomationRunMetadataColumn(ctx, tx); err != nil {
		return err
	}
	if err := rebuildAutomationSchedulerStateCatchUpContract(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit automation catch-up migration: %w", err)
	}
	return nil
}

func rebuildAutomationSchedulerStateCatchUpContract(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "automation_scheduler_state")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	for _, statement := range automationSchedulerStateCatchUpStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: migrate automation catch-up contract: %w", err)
		}
	}
	return nil
}

func automationSchedulerStateCatchUpStatements() []string {
	return []string{
		`CREATE TABLE automation_scheduler_state_new (
			job_id                       TEXT PRIMARY KEY,
			next_run_at                  TEXT,
			last_run_at                  TEXT,
			last_scheduled_at            TEXT,
			last_fire_id                 TEXT NOT NULL DEFAULT '',
			schedule_hash                TEXT NOT NULL DEFAULT '',
			catch_up_policy              TEXT NOT NULL DEFAULT 'skip'
				CHECK (catch_up_policy IN ('skip', 'coalesce', 'replay')),
			misfire_grace_seconds        INTEGER NOT NULL DEFAULT 0
				CHECK (misfire_grace_seconds >= 0),
			consecutive_resume_failures  INTEGER NOT NULL DEFAULT 0
				CHECK (consecutive_resume_failures >= 0),
			last_misfire_at              TEXT,
			misfire_count                INTEGER NOT NULL DEFAULT 0
				CHECK (misfire_count >= 0),
			updated_at                   TEXT NOT NULL
		);`,
		`INSERT INTO automation_scheduler_state_new (
			job_id, next_run_at, last_run_at, last_scheduled_at, last_fire_id,
			schedule_hash, catch_up_policy, misfire_grace_seconds,
			consecutive_resume_failures, last_misfire_at, misfire_count, updated_at
		)
		SELECT
			job_id, next_run_at, last_run_at, last_scheduled_at, last_fire_id,
			schedule_hash,
			CASE catch_up_policy
				WHEN 'skip_missed' THEN 'skip'
				WHEN 'skip' THEN 'skip'
				WHEN 'coalesce' THEN 'coalesce'
				WHEN 'replay' THEN 'replay'
				ELSE 'skip'
			END,
			misfire_grace_seconds, consecutive_resume_failures,
			last_misfire_at, misfire_count, updated_at
		FROM automation_scheduler_state;`,
		`DROP TABLE automation_scheduler_state;`,
		`ALTER TABLE automation_scheduler_state_new RENAME TO automation_scheduler_state;`,
		`CREATE INDEX IF NOT EXISTS idx_automation_scheduler_next_run
			ON automation_scheduler_state(next_run_at);`,
		`CREATE INDEX IF NOT EXISTS idx_automation_scheduler_misfire
			ON automation_scheduler_state(last_misfire_at);`,
	}
}

func addAutomationRunMetadataColumn(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "automation_runs")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	columns, err := tableColumns(ctx, tx, "automation_runs")
	if err != nil {
		return err
	}
	if _, ok := columns["metadata_json"]; ok {
		return nil
	}
	if _, err := tx.ExecContext(
		ctx,
		`ALTER TABLE automation_runs ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '{}'`,
	); err != nil {
		return fmt.Errorf("store: add automation_runs.metadata_json: %w", err)
	}
	return nil
}
