package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	migrateTaskOrchestrationProfileSummaryKey = "summary"
	taskExecutionProfileRuntimeModeColumn     = "runtime_mode"
)

func migrateTaskOrchestrationProfileSchema(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "tasks", []migrationColumnSpec{
		{
			name: "current_run_id",
			sql:  `ALTER TABLE tasks ADD COLUMN current_run_id TEXT REFERENCES task_runs(id) ON DELETE SET NULL`,
		},
		{
			name: "max_runtime_seconds",
			sql: `ALTER TABLE tasks ADD COLUMN max_runtime_seconds INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (max_runtime_seconds >= 0)`,
		},
		{
			name: "spawn_failure_count",
			sql: `ALTER TABLE tasks ADD COLUMN spawn_failure_count INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (spawn_failure_count >= 0)`,
		},
		{name: "last_spawn_error", sql: `ALTER TABLE tasks ADD COLUMN last_spawn_error TEXT NOT NULL DEFAULT ''`},
	}); err != nil {
		return err
	}
	if err := addMissingMigrationColumns(ctx, tx, "task_runs", []migrationColumnSpec{
		{
			name: migrateTaskOrchestrationProfileSummaryKey,
			sql:  `ALTER TABLE task_runs ADD COLUMN summary TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "claimed_agent_name",
			sql:  `ALTER TABLE task_runs ADD COLUMN claimed_agent_name TEXT NOT NULL DEFAULT ''`,
		},
		{name: "claimed_peer_id", sql: `ALTER TABLE task_runs ADD COLUMN claimed_peer_id TEXT NOT NULL DEFAULT ''`},
		{
			name: "terminalized_by_session_id",
			sql:  `ALTER TABLE task_runs ADD COLUMN terminalized_by_session_id TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "terminalized_by_agent_name",
			sql:  `ALTER TABLE task_runs ADD COLUMN terminalized_by_agent_name TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "terminalized_by_peer_id",
			sql:  `ALTER TABLE task_runs ADD COLUMN terminalized_by_peer_id TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "terminalized_by_actor_kind",
			sql:  `ALTER TABLE task_runs ADD COLUMN terminalized_by_actor_kind TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "terminalized_by_actor_ref",
			sql:  `ALTER TABLE task_runs ADD COLUMN terminalized_by_actor_ref TEXT NOT NULL DEFAULT ''`,
		},
	}); err != nil {
		return err
	}
	for _, statement := range taskOrchestrationProfileMigrationSchemaStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply task orchestration profile schema: %w", err)
		}
	}
	return nil
}

func migrateTaskExecutionProfileRuntimeMode(ctx context.Context, tx *sql.Tx) error {
	return addTaskExecutionProfileRuntimeModeColumn(ctx, tx)
}

func addTaskExecutionProfileRuntimeModeColumn(ctx context.Context, tx *sql.Tx) error {
	return addMissingMigrationColumns(ctx, tx, "task_execution_profiles", []migrationColumnSpec{
		{
			name: taskExecutionProfileRuntimeModeColumn,
			sql: `ALTER TABLE task_execution_profiles ADD COLUMN runtime_mode TEXT NOT NULL DEFAULT 'default' ` +
				`CHECK (runtime_mode IN ('default', 'evidence'))`,
		},
	})
}

func addMissingMigrationColumns(
	ctx context.Context,
	tx *sql.Tx,
	table string,
	specs []migrationColumnSpec,
) error {
	columns, err := tableColumns(ctx, tx, table)
	if err != nil {
		return err
	}
	for _, spec := range specs {
		if _, ok := columns[spec.name]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, spec.sql); err != nil {
			return fmt.Errorf("store: add %s.%s column: %w", table, spec.name, err)
		}
	}
	return nil
}
