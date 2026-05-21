package globaldb

func notificationPresetSchemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS notification_presets (
			name                     TEXT PRIMARY KEY CHECK (trim(name) <> ''),
			events                   TEXT NOT NULL CHECK (json_valid(events)),
			targets                  TEXT NOT NULL CHECK (json_valid(targets)),
			filter                   TEXT NOT NULL DEFAULT '',
			enabled                  BOOLEAN NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
			built_in                 BOOLEAN NOT NULL DEFAULT 0 CHECK (built_in IN (0, 1)),
			default_version          TEXT NOT NULL DEFAULT '',
			default_hash             TEXT NOT NULL DEFAULT '',
			user_modified            BOOLEAN NOT NULL DEFAULT 0 CHECK (user_modified IN (0, 1)),
			default_update_available BOOLEAN NOT NULL DEFAULT 0 CHECK (default_update_available IN (0, 1)),
			created_at               TEXT NOT NULL,
			updated_at               TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_notification_presets_enabled
			ON notification_presets(enabled, name);`,
		`CREATE INDEX IF NOT EXISTS idx_notification_presets_builtin
			ON notification_presets(built_in, name);`,
	}
}
