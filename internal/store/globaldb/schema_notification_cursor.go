package globaldb

func notificationCursorSchemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS notification_cursors (
			consumer_id       TEXT NOT NULL CHECK (trim(consumer_id) <> ''),
			stream_name       TEXT NOT NULL CHECK (trim(stream_name) <> ''),
			subject_id        TEXT NOT NULL DEFAULT '',
			last_sequence     INTEGER NOT NULL DEFAULT 0 CHECK (last_sequence >= 0),
			last_delivery_id  TEXT NOT NULL DEFAULT '',
			last_delivered_at TEXT,
			last_error        TEXT NOT NULL DEFAULT '',
			updated_at        TEXT NOT NULL,
			PRIMARY KEY (consumer_id, stream_name, subject_id)
		);`,
		`CREATE INDEX IF NOT EXISTS notification_cursors_stream_sequence_idx
			ON notification_cursors(stream_name, last_sequence DESC)
			WHERE last_sequence > 0;`,
	}
}
