package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	migrateTaskReviewGateContinuationReasonKey    = "continuation_reason"
	migrateTaskReviewGateLastReviewIDKey          = "last_review_id"
	migrateTaskReviewGateLastReviewOutcomeKey     = "last_review_outcome"
	migrateTaskReviewGateMissingWorkJSONKey       = "missing_work_json"
	migrateTaskReviewGateNextRoundGuidanceKey     = "next_round_guidance"
	migrateTaskReviewGateParentRunIDKey           = "parent_run_id"
	migrateTaskReviewGateReviewCircuitOpenedAtKey = "review_circuit_opened_at"
	migrateTaskReviewGateReviewCircuitReasonKey   = "review_circuit_reason"
	migrateTaskReviewGateReviewIDKey              = "review_id"
	migrateTaskReviewGateReviewMaxRoundsKey       = "review_max_rounds"
	migrateTaskReviewGateReviewPolicyKey          = "review_policy"
	migrateTaskReviewGateReviewPolicySnapshotKey  = "review_policy_snapshot"
	migrateTaskReviewGateReviewRequestIDKey       = "review_request_id"
	migrateTaskReviewGateReviewRequestRoundKey    = "review_request_round"
	migrateTaskReviewGateReviewRequiredKey        = "review_required"
	migrateTaskReviewGateReviewRoundKey           = "review_round"
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
			name: migrateTaskReviewGateReviewPolicyKey,
			sql: `ALTER TABLE tasks ADD COLUMN review_policy TEXT NOT NULL DEFAULT 'none' ` +
				`CHECK (review_policy IN ('none', 'on_success', 'on_failure', 'always'))`,
		},
		{
			name: migrateTaskReviewGateReviewMaxRoundsKey,
			sql: `ALTER TABLE tasks ADD COLUMN review_max_rounds INTEGER NOT NULL DEFAULT 3 ` +
				`CHECK (review_max_rounds >= 0)`,
		},
		{
			name: migrateTaskReviewGateReviewRoundKey,
			sql: `ALTER TABLE tasks ADD COLUMN review_round INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (review_round >= 0)`,
		},
		{name: migrateTaskReviewGateLastReviewIDKey, sql: `ALTER TABLE tasks ADD COLUMN last_review_id TEXT`},
		{
			name: migrateTaskReviewGateLastReviewOutcomeKey,
			sql: `ALTER TABLE tasks ADD COLUMN last_review_outcome TEXT CHECK (
				last_review_outcome IS NULL OR last_review_outcome IN (
					'approved', 'rejected', 'blocked', 'error', 'timeout', 'invalid_output'
				)
			)`,
		},
		{
			name: migrateTaskReviewGateReviewCircuitOpenedAtKey,
			sql:  `ALTER TABLE tasks ADD COLUMN review_circuit_opened_at TEXT`,
		},
		{
			name: migrateTaskReviewGateReviewCircuitReasonKey,
			sql:  `ALTER TABLE tasks ADD COLUMN review_circuit_reason TEXT`,
		},
	}
}

func taskReviewGateRunColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{
			name: migrateTaskReviewGateReviewRequiredKey,
			sql: `ALTER TABLE task_runs ADD COLUMN review_required BOOLEAN NOT NULL DEFAULT 0 ` +
				`CHECK (review_required IN (0, 1))`,
		},
		{
			name: migrateTaskReviewGateReviewRequestRoundKey,
			sql: `ALTER TABLE task_runs ADD COLUMN review_request_round INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (review_request_round >= 0)`,
		},
		{
			name: migrateTaskReviewGateReviewPolicySnapshotKey,
			sql: `ALTER TABLE task_runs ADD COLUMN review_policy_snapshot TEXT NOT NULL DEFAULT '' CHECK (
				review_policy_snapshot = '' OR
				review_policy_snapshot IN ('none', 'on_success', 'on_failure', 'always')
			)`,
		},
		{
			name: migrateTaskReviewGateReviewRequestIDKey,
			sql:  `ALTER TABLE task_runs ADD COLUMN review_request_id TEXT REFERENCES task_run_reviews(review_id)`,
		},
		{
			name: migrateTaskReviewGateParentRunIDKey,
			sql:  `ALTER TABLE task_runs ADD COLUMN parent_run_id TEXT REFERENCES task_runs(id)`,
		},
		{
			name: migrateTaskReviewGateReviewIDKey,
			sql:  `ALTER TABLE task_runs ADD COLUMN review_id TEXT REFERENCES task_run_reviews(review_id)`,
		},
		{
			name: migrateTaskReviewGateReviewRoundKey,
			sql: `ALTER TABLE task_runs ADD COLUMN review_round INTEGER NOT NULL DEFAULT 0 ` +
				`CHECK (review_round >= 0)`,
		},
		{
			name: migrateTaskReviewGateContinuationReasonKey,
			sql:  `ALTER TABLE task_runs ADD COLUMN continuation_reason TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: migrateTaskReviewGateMissingWorkJSONKey,
			sql:  `ALTER TABLE task_runs ADD COLUMN missing_work_json TEXT NOT NULL DEFAULT '[]'`,
		},
		{
			name: migrateTaskReviewGateNextRoundGuidanceKey,
			sql:  `ALTER TABLE task_runs ADD COLUMN next_round_guidance TEXT NOT NULL DEFAULT ''`,
		},
	}
}
