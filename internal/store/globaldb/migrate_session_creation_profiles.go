package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

var sessionCreationProfilesMigration = store.Migration{
	Version:  77,
	Name:     "add_session_creation_profiles",
	Up:       migrateSessionCreationProfiles,
	Checksum: "2026-07-10-add-session-creation-profiles",
}

func migrateSessionCreationProfiles(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `CREATE TABLE session_creation_profiles (
		profile_ref TEXT PRIMARY KEY CHECK (length(trim(profile_ref)) > 0),
		profile_json TEXT NOT NULL CHECK (json_valid(profile_json)),
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("store: create session creation profiles table: %w", err)
	}
	return nil
}
