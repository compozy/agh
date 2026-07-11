package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

// Migration 78 pins the immutable session creation identity used by inline Goal Runs.
//
// Why: ADR-008 requires inline execution and reseed to avoid resolving mutable session defaults.
// Affects: agh.db loop_runs.
// Idempotent: yes; each nullable column is guarded through pragma_table_info.
// Reversible: no; the fields are part of the durable Run execution contract.
var loopRunOriginIdentityMigration = store.Migration{
	Version:  78,
	Name:     "add_loop_run_origin_identity",
	Up:       migrateLoopRunOriginIdentity,
	Checksum: "2026-07-10-add-loop-run-origin-identity",
}

func migrateLoopRunOriginIdentity(ctx context.Context, tx *sql.Tx) error {
	specs := []migrationColumnSpec{
		{
			name: "origin_creation_profile_ref",
			sql: `ALTER TABLE loop_runs ADD COLUMN origin_creation_profile_ref TEXT
				CHECK (origin_creation_profile_ref IS NULL OR length(trim(origin_creation_profile_ref)) > 0)`,
		},
		{
			name: "origin_policy_spec_digest",
			sql: `ALTER TABLE loop_runs ADD COLUMN origin_policy_spec_digest TEXT
				CHECK (origin_policy_spec_digest IS NULL OR length(trim(origin_policy_spec_digest)) > 0)`,
		},
		{
			name: "origin_creation_digest",
			sql: `ALTER TABLE loop_runs ADD COLUMN origin_creation_digest TEXT
				CHECK (origin_creation_digest IS NULL OR length(trim(origin_creation_digest)) > 0)`,
		},
	}
	if err := addMissingMigrationColumns(ctx, tx, "loop_runs", specs); err != nil {
		return fmt.Errorf("store: add Loop Run origin identity columns: %w", err)
	}
	return nil
}
