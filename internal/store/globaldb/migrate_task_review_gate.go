package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func migrateTaskReviewGateSchema(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "tasks", taskReviewGateTaskColumnSpecs()); err != nil {
		return err
	}
	for _, statement := range taskRunReviewTableSchemaStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply task run review table schema: %w", err)
		}
	}
	if err := addMissingMigrationColumns(ctx, tx, "task_runs", taskReviewGateRunColumnSpecs()); err != nil {
		return err
	}
	for _, statement := range taskReviewGateIndexStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply task review gate indexes: %w", err)
		}
	}
	return nil
}

func taskReviewGateTaskColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{
			name: "review_policy",
			sql: `ALTER TABLE tasks ADD COLUMN review_policy TEXT NOT NULL DEFAULT 'none' ` +
				`CHECK (review_policy IN ('none', 'on_success', 'on_failure', 'always'))`,
		},
		{
			name: "review_max_rounds",
			sql: `ALTER TABLE tasks ADD COLUMN review_max_rounds INTEGER NOT NULL DEFAULT 3 ` +
				`CHECK (review_max_rounds >= 0)`,
		},
		{
			name: "review_round",
			sql: `ALTER TABLE tasks ADD COLUMN review_round INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (review_round >= 0)`,
		},
		{name: "last_review_id", sql: `ALTER TABLE tasks ADD COLUMN last_review_id TEXT`},
		{
			name: "last_review_outcome",
			sql: `ALTER TABLE tasks ADD COLUMN last_review_outcome TEXT CHECK (
				last_review_outcome IS NULL OR last_review_outcome IN (
					'approved', 'rejected', 'blocked', 'error', 'timeout', 'invalid_output'
				)
			)`,
		},
		{name: "review_circuit_opened_at", sql: `ALTER TABLE tasks ADD COLUMN review_circuit_opened_at TEXT`},
		{name: "review_circuit_reason", sql: `ALTER TABLE tasks ADD COLUMN review_circuit_reason TEXT`},
	}
}

func taskReviewGateRunColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{
			name: "review_required",
			sql: `ALTER TABLE task_runs ADD COLUMN review_required BOOLEAN NOT NULL DEFAULT 0 ` +
				`CHECK (review_required IN (0, 1))`,
		},
		{
			name: "review_request_round",
			sql: `ALTER TABLE task_runs ADD COLUMN review_request_round INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (review_request_round >= 0)`,
		},
		{
			name: "review_policy_snapshot",
			sql: `ALTER TABLE task_runs ADD COLUMN review_policy_snapshot TEXT NOT NULL DEFAULT '' CHECK (
				review_policy_snapshot = '' OR
				review_policy_snapshot IN ('none', 'on_success', 'on_failure', 'always')
			)`,
		},
		{
			name: "review_request_id",
			sql:  `ALTER TABLE task_runs ADD COLUMN review_request_id TEXT REFERENCES task_run_reviews(review_id)`,
		},
		{
			name: "parent_run_id",
			sql:  `ALTER TABLE task_runs ADD COLUMN parent_run_id TEXT REFERENCES task_runs(id)`,
		},
		{
			name: "review_id",
			sql:  `ALTER TABLE task_runs ADD COLUMN review_id TEXT REFERENCES task_run_reviews(review_id)`,
		},
		{
			name: "review_round",
			sql: `ALTER TABLE task_runs ADD COLUMN review_round INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (review_round >= 0)`,
		},
		{
			name: "continuation_reason",
			sql:  `ALTER TABLE task_runs ADD COLUMN continuation_reason TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "missing_work_json",
			sql:  `ALTER TABLE task_runs ADD COLUMN missing_work_json TEXT NOT NULL DEFAULT '[]'`,
		},
		{
			name: "next_round_guidance",
			sql:  `ALTER TABLE task_runs ADD COLUMN next_round_guidance TEXT NOT NULL DEFAULT ''`,
		},
	}
}
