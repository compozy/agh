package globaldb

import (
	"context"
	"database/sql"
	"fmt"
)

func addGoalDurableColumns(ctx context.Context, tx *sql.Tx) error {
	groups := []struct {
		table string
		specs []migrationColumnSpec
	}{
		{table: columnSessions, specs: goalSessionColumnSpecs()},
		{table: "loop_generation_outputs", specs: goalGenerationOutputColumnSpecs()},
		{table: "loop_runs", specs: goalLoopRunColumnSpecs()},
		{table: "session_input_queue", specs: goalSessionInputQueueColumnSpecs()},
	}
	for _, group := range groups {
		if err := addMissingMigrationColumns(ctx, tx, group.table, group.specs); err != nil {
			return fmt.Errorf("store: add goal durable columns to %s: %w", group.table, err)
		}
	}
	return nil
}

func goalSessionColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{
			name: columnCreationDigest,
			sql: `ALTER TABLE sessions ADD COLUMN creation_digest TEXT
				CHECK (creation_digest IS NULL OR length(trim(creation_digest)) > 0)`,
		},
		{
			name: columnPolicySpecDigest,
			sql: `ALTER TABLE sessions ADD COLUMN policy_spec_digest TEXT
				CHECK (policy_spec_digest IS NULL OR length(trim(policy_spec_digest)) > 0)`,
		},
		{
			name: columnCreationProfile,
			sql: `ALTER TABLE sessions ADD COLUMN creation_profile_ref TEXT
				CHECK (creation_profile_ref IS NULL OR length(trim(creation_profile_ref)) > 0)`,
		},
	}
}

func goalGenerationOutputColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{name: "goal_status", sql: `ALTER TABLE loop_generation_outputs ADD COLUMN goal_status TEXT`},
		{name: "goal_turns_used", sql: `ALTER TABLE loop_generation_outputs ADD COLUMN goal_turns_used INTEGER`},
		{name: "goal_turn_limit", sql: `ALTER TABLE loop_generation_outputs ADD COLUMN goal_turn_limit INTEGER`},
	}
}

func goalLoopRunColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{name: columnOriginKind, sql: `ALTER TABLE loop_runs ADD COLUMN origin_kind TEXT NOT NULL DEFAULT 'catalog'`},
		{name: "origin_session_id", sql: `ALTER TABLE loop_runs ADD COLUMN origin_session_id TEXT`},
		{name: "goal_cleared_at", sql: `ALTER TABLE loop_runs ADD COLUMN goal_cleared_at TIMESTAMP`},
		{
			name: "budget_version",
			sql:  `ALTER TABLE loop_runs ADD COLUMN budget_version INTEGER NOT NULL DEFAULT 0 CHECK (budget_version >= 0)`,
		},
		{
			name: "goal_context_nudge_ratio",
			sql: `ALTER TABLE loop_runs ADD COLUMN goal_context_nudge_ratio REAL NOT NULL DEFAULT 0.8
				CHECK (goal_context_nudge_ratio >= 0.0 AND goal_context_nudge_ratio <= 1.0)`,
		},
	}
}

func goalSessionInputQueueColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{name: columnLoopRunID, sql: `ALTER TABLE session_input_queue ADD COLUMN loop_run_id TEXT`},
		{name: columnOwnerKind, sql: `ALTER TABLE session_input_queue ADD COLUMN owner_kind TEXT`},
		{name: "owner_epoch", sql: `ALTER TABLE session_input_queue ADD COLUMN owner_epoch INTEGER`},
		{name: columnBindingEpoch, sql: `ALTER TABLE session_input_queue ADD COLUMN binding_epoch INTEGER`},
		{name: columnPromptID, sql: `ALTER TABLE session_input_queue ADD COLUMN prompt_id TEXT`},
		{name: "prompt_kind", sql: `ALTER TABLE session_input_queue ADD COLUMN prompt_kind TEXT`},
		{
			name: "operation_usage_base_tokens",
			sql:  `ALTER TABLE session_input_queue ADD COLUMN operation_usage_base_tokens INTEGER`,
		},
		{
			name: columnPromptAttempt,
			sql:  `ALTER TABLE session_input_queue ADD COLUMN prompt_attempt INTEGER NOT NULL DEFAULT 0 CHECK (prompt_attempt >= 0)`,
		},
		{
			name: "dispatchable",
			sql:  `ALTER TABLE session_input_queue ADD COLUMN dispatchable INTEGER NOT NULL DEFAULT 1 CHECK (dispatchable IN (0,1))`,
		},
		{name: "activated_at", sql: `ALTER TABLE session_input_queue ADD COLUMN activated_at TIMESTAMP`},
		{name: "dispatch_token_hash", sql: `ALTER TABLE session_input_queue ADD COLUMN dispatch_token_hash TEXT`},
		{name: "fence_kind", sql: `ALTER TABLE session_input_queue ADD COLUMN fence_kind TEXT`},
		{name: "fence_disposition", sql: `ALTER TABLE session_input_queue ADD COLUMN fence_disposition TEXT`},
		{name: "fence_reason_code", sql: `ALTER TABLE session_input_queue ADD COLUMN fence_reason_code TEXT`},
		{name: "fenced_at", sql: `ALTER TABLE session_input_queue ADD COLUMN fenced_at TIMESTAMP`},
		{
			name: "terminal_event_start_seq",
			sql:  `ALTER TABLE session_input_queue ADD COLUMN terminal_event_start_seq INTEGER`,
		},
		{
			name: "terminal_event_end_seq",
			sql:  `ALTER TABLE session_input_queue ADD COLUMN terminal_event_end_seq INTEGER`,
		},
		{name: "terminal_kind", sql: `ALTER TABLE session_input_queue ADD COLUMN terminal_kind TEXT`},
		{name: "terminal_stop_reason", sql: `ALTER TABLE session_input_queue ADD COLUMN terminal_stop_reason TEXT`},
		{name: "terminal_disposition", sql: `ALTER TABLE session_input_queue ADD COLUMN terminal_disposition TEXT`},
		{name: "terminal_reason_code", sql: `ALTER TABLE session_input_queue ADD COLUMN terminal_reason_code TEXT`},
		{
			name: "terminal_tokens_reported",
			sql: `ALTER TABLE session_input_queue ADD COLUMN terminal_tokens_reported INTEGER NOT NULL DEFAULT 0
				CHECK (terminal_tokens_reported IN (0,1))`,
		},
		{
			name: "terminal_tokens_used",
			sql: `ALTER TABLE session_input_queue ADD COLUMN terminal_tokens_used INTEGER
				CHECK (terminal_tokens_used IS NULL OR terminal_tokens_used >= 0)`,
		},
		{name: "terminal_at", sql: `ALTER TABLE session_input_queue ADD COLUMN terminal_at TIMESTAMP`},
	}
}
