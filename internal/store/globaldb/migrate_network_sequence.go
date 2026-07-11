package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

const (
	networkOpenedSequenceColumn       = "opened_sequence"
	networkLastActivitySequenceColumn = "last_activity_sequence"
	networkThreadProjectionLabel      = "thread"
	networkChannelProjectionLabel     = "channel"
)

// Migration 71: replace implicit Network rowid ordering with a durable sequence.
// Why: timestamps and message IDs do not encode append causality.
// Affects: network timeline, thread/direct summaries, and channel projections.
// Idempotent: the transactional registry retries from v70; projection columns are guarded.
// Reversible: no; sequence becomes the durable event identity.
var networkTimelineSequenceMigration = store.Migration{
	Version:  71,
	Name:     "add_network_timeline_sequence",
	Up:       migrateNetworkTimelineSequence,
	Checksum: "2026-07-11-add-network-timeline-sequence",
}

const networkTimelineSequenceTableStatement = `CREATE TABLE network_timeline_log (
	sequence        INTEGER PRIMARY KEY AUTOINCREMENT,
	message_id      TEXT NOT NULL,
	session_id      TEXT,
	workspace_id    TEXT NOT NULL,
	channel         TEXT NOT NULL,
	surface         TEXT CHECK (surface IN ('thread', 'direct') OR surface IS NULL),
	thread_id       TEXT,
	direct_id       TEXT,
	direction       TEXT NOT NULL,
	peer_from       TEXT NOT NULL,
	peer_to         TEXT,
	kind            TEXT NOT NULL,
	work_id         TEXT,
	reply_to        TEXT,
	trace_id        TEXT,
	causation_id    TEXT,
	intent          TEXT,
	text            TEXT,
	preview_text    TEXT NOT NULL DEFAULT '',
	body_json       TEXT NOT NULL,
	timestamp       TEXT NOT NULL,
	ext_json        TEXT NOT NULL DEFAULT '{}',
	mentions_json   TEXT NOT NULL DEFAULT '[]',
	work_opened     INTEGER NOT NULL DEFAULT 0 CHECK (work_opened IN (0, 1)),
	work_transitioned INTEGER NOT NULL DEFAULT 0 CHECK (work_transitioned IN (0, 1)),
	work_state      TEXT NOT NULL DEFAULT '' CHECK (
		work_state IN ('', 'submitted', 'working', 'needs_input', 'completed', 'failed', 'canceled')
	),
	UNIQUE (workspace_id, message_id),
	CHECK (
		(surface IS NULL AND thread_id IS NULL AND direct_id IS NULL AND work_id IS NULL
			AND kind IN ('greet', 'whois'))
		OR (surface = 'thread' AND thread_id IS NOT NULL AND direct_id IS NULL)
		OR (surface = 'direct' AND direct_id IS NOT NULL AND thread_id IS NULL)
	),
	CHECK (kind IN ('greet', 'whois', 'say', 'capability', 'receipt', 'trace'))
)`

var networkTimelineSequenceIndexStatements = []string{
	`CREATE INDEX idx_net_timeline_thread_sequence
		ON network_timeline_log(workspace_id, channel, thread_id, sequence)
		WHERE surface = 'thread'`,
	`CREATE INDEX idx_net_timeline_direct_sequence
		ON network_timeline_log(workspace_id, channel, direct_id, sequence)
		WHERE surface = 'direct'`,
	`CREATE INDEX idx_net_timeline_work_sequence
		ON network_timeline_log(workspace_id, work_id, sequence)
		WHERE work_id IS NOT NULL`,
	`CREATE INDEX idx_net_timeline_presence_sequence
		ON network_timeline_log(workspace_id, channel, sequence)
		WHERE surface IS NULL`,
	`CREATE INDEX idx_net_timeline_kind_sequence
		ON network_timeline_log(workspace_id, kind, sequence)`,
}

var networkSequenceProjectionIndexStatements = []string{
	`DROP INDEX IF EXISTS idx_network_threads_activity`,
	`CREATE INDEX idx_network_threads_activity
		ON network_threads(workspace_id, channel, last_activity_sequence DESC, thread_id)`,
	`DROP INDEX IF EXISTS idx_network_threads_created`,
	`CREATE INDEX idx_network_threads_created
		ON network_threads(workspace_id, channel, opened_sequence, thread_id)`,
	`DROP INDEX IF EXISTS idx_network_threads_open_work`,
	`CREATE INDEX idx_network_threads_open_work
		ON network_threads(workspace_id, channel, open_work_count, last_activity_sequence DESC, thread_id)`,
	`DROP INDEX IF EXISTS idx_network_direct_rooms_activity`,
	`CREATE INDEX idx_network_direct_rooms_activity
		ON network_direct_rooms(workspace_id, channel, last_activity_sequence DESC, direct_id)`,
	`DROP INDEX IF EXISTS idx_network_direct_rooms_peer_a`,
	`CREATE INDEX idx_network_direct_rooms_peer_a
		ON network_direct_rooms(workspace_id, channel, peer_a, last_activity_sequence DESC)`,
	`DROP INDEX IF EXISTS idx_network_direct_rooms_peer_b`,
	`CREATE INDEX idx_network_direct_rooms_peer_b
		ON network_direct_rooms(workspace_id, channel, peer_b, last_activity_sequence DESC)`,
	`DROP INDEX IF EXISTS idx_network_direct_rooms_created`,
	`CREATE INDEX idx_network_direct_rooms_created
		ON network_direct_rooms(workspace_id, channel, opened_sequence, direct_id)`,
	`DROP INDEX IF EXISTS idx_network_direct_rooms_open_work`,
	`CREATE INDEX idx_network_direct_rooms_open_work
		ON network_direct_rooms(workspace_id, channel, open_work_count, last_activity_sequence DESC, direct_id)`,
	`DROP INDEX IF EXISTS idx_network_channel_stats_activity`,
	`CREATE INDEX idx_network_channel_stats_activity
		ON network_channel_stats(workspace_id, last_activity_sequence DESC, channel)`,
}

func migrateNetworkTimelineSequence(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range networkTimelineSequenceRebuildStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: rebuild network timeline with causal sequence: %w", err)
		}
	}
	if err := addNetworkSequenceProjectionColumns(ctx, tx); err != nil {
		return err
	}
	if err := backfillNetworkSequenceProjections(ctx, tx); err != nil {
		return err
	}
	for _, statement := range networkSequenceProjectionIndexStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: recreate network projection sequence index: %w", err)
		}
	}
	return nil
}

func networkTimelineSequenceRebuildStatements() []string {
	statements := []string{
		`ALTER TABLE network_timeline_log RENAME TO network_timeline_log_v70`,
		networkTimelineSequenceTableStatement,
		`INSERT INTO network_timeline_log (
			sequence, message_id, session_id, workspace_id, channel, surface, thread_id, direct_id,
			direction, peer_from, peer_to, kind, work_id, reply_to, trace_id, causation_id, intent,
			text, preview_text, body_json, timestamp, ext_json, mentions_json, work_opened,
			work_transitioned, work_state
		)
		SELECT
			rowid, message_id, session_id, workspace_id, channel, surface, thread_id, direct_id,
			direction, peer_from, peer_to, kind, work_id, reply_to, trace_id, causation_id, intent,
			text, preview_text, body_json, timestamp, ext_json, mentions_json, work_opened,
			work_transitioned, work_state
		FROM network_timeline_log_v70
		ORDER BY rowid`,
		`DROP TABLE network_timeline_log_v70`,
	}
	return append(statements, networkTimelineSequenceIndexStatements...)
}

func addNetworkSequenceProjectionColumns(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "network_threads", []migrationColumnSpec{
		{
			name: networkOpenedSequenceColumn,
			sql:  `ALTER TABLE network_threads ADD COLUMN opened_sequence INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: networkLastActivitySequenceColumn,
			sql:  `ALTER TABLE network_threads ADD COLUMN last_activity_sequence INTEGER NOT NULL DEFAULT 0`,
		},
	}); err != nil {
		return err
	}
	if err := addMissingMigrationColumns(ctx, tx, "network_direct_rooms", []migrationColumnSpec{
		{
			name: networkOpenedSequenceColumn,
			sql:  `ALTER TABLE network_direct_rooms ADD COLUMN opened_sequence INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: networkLastActivitySequenceColumn,
			sql:  `ALTER TABLE network_direct_rooms ADD COLUMN last_activity_sequence INTEGER NOT NULL DEFAULT 0`,
		},
	}); err != nil {
		return err
	}
	return addMissingMigrationColumns(ctx, tx, "network_channel_stats", []migrationColumnSpec{
		{
			name: networkLastActivitySequenceColumn,
			sql:  `ALTER TABLE network_channel_stats ADD COLUMN last_activity_sequence INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: "last_presence_sequence",
			sql:  `ALTER TABLE network_channel_stats ADD COLUMN last_presence_sequence INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: "last_message_sequence",
			sql:  `ALTER TABLE network_channel_stats ADD COLUMN last_message_sequence INTEGER NOT NULL DEFAULT 0`,
		},
	})
}

func backfillNetworkSequenceProjections(ctx context.Context, tx *sql.Tx) error {
	statements := []struct {
		name string
		sql  string
	}{
		{name: networkThreadProjectionLabel, sql: backfillNetworkThreadSequenceStatement},
		{name: "direct room", sql: backfillNetworkDirectRoomSequenceStatement},
		{name: networkChannelProjectionLabel, sql: backfillNetworkChannelSequenceStatement},
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement.sql); err != nil {
			return fmt.Errorf("store: backfill network %s sequence projection: %w", statement.name, err)
		}
	}
	return nil
}

const backfillNetworkThreadSequenceStatement = `UPDATE network_threads
SET opened_sequence = COALESCE((
		SELECT timeline.sequence
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_threads.workspace_id
			AND timeline.channel = network_threads.channel
			AND timeline.surface = 'thread'
			AND timeline.thread_id = network_threads.thread_id
			AND timeline.message_id = network_threads.root_message_id
		LIMIT 1
	), 0),
	last_activity_sequence = COALESCE((
		SELECT timeline.sequence
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_threads.workspace_id
			AND timeline.channel = network_threads.channel
			AND timeline.surface = 'thread'
			AND timeline.thread_id = network_threads.thread_id
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), 0),
	last_activity_at = COALESCE((
		SELECT timeline.timestamp
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_threads.workspace_id
			AND timeline.channel = network_threads.channel
			AND timeline.surface = 'thread'
			AND timeline.thread_id = network_threads.thread_id
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), last_activity_at),
	last_message_preview = COALESCE((
		SELECT COALESCE(NULLIF(TRIM(timeline.preview_text), ''), NULLIF(TRIM(timeline.text), ''), '')
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_threads.workspace_id
			AND timeline.channel = network_threads.channel
			AND timeline.surface = 'thread'
			AND timeline.thread_id = network_threads.thread_id
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), '')`

const backfillNetworkDirectRoomSequenceStatement = `UPDATE network_direct_rooms
SET opened_sequence = COALESCE((
		SELECT timeline.sequence
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_direct_rooms.workspace_id
			AND timeline.channel = network_direct_rooms.channel
			AND timeline.surface = 'direct'
			AND timeline.direct_id = network_direct_rooms.direct_id
		ORDER BY timeline.sequence
		LIMIT 1
	), 0),
	last_activity_sequence = COALESCE((
		SELECT timeline.sequence
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_direct_rooms.workspace_id
			AND timeline.channel = network_direct_rooms.channel
			AND timeline.surface = 'direct'
			AND timeline.direct_id = network_direct_rooms.direct_id
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), 0),
	last_activity_at = COALESCE((
		SELECT timeline.timestamp
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_direct_rooms.workspace_id
			AND timeline.channel = network_direct_rooms.channel
			AND timeline.surface = 'direct'
			AND timeline.direct_id = network_direct_rooms.direct_id
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), last_activity_at),
	last_message_preview = COALESCE((
		SELECT COALESCE(NULLIF(TRIM(timeline.preview_text), ''), NULLIF(TRIM(timeline.text), ''), '')
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_direct_rooms.workspace_id
			AND timeline.channel = network_direct_rooms.channel
			AND timeline.surface = 'direct'
			AND timeline.direct_id = network_direct_rooms.direct_id
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), '')`

const backfillNetworkChannelSequenceStatement = `UPDATE network_channel_stats
SET last_activity_sequence = COALESCE((
		SELECT timeline.sequence
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_channel_stats.workspace_id
			AND timeline.channel = network_channel_stats.channel
			AND timeline.kind <> 'greet'
			AND COALESCE(TRIM(timeline.peer_to), '') = ''
			AND COALESCE(TRIM(timeline.direct_id), '') = ''
			AND COALESCE(TRIM(timeline.surface), '') <> 'direct'
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), 0),
	last_presence_sequence = COALESCE((
		SELECT timeline.sequence
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_channel_stats.workspace_id
			AND timeline.channel = network_channel_stats.channel
			AND timeline.kind = 'greet'
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), 0),
	last_message_sequence = COALESCE((
		SELECT timeline.sequence
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_channel_stats.workspace_id
			AND timeline.channel = network_channel_stats.channel
			AND timeline.kind <> 'greet'
			AND COALESCE(TRIM(timeline.peer_to), '') = ''
			AND COALESCE(TRIM(timeline.direct_id), '') = ''
			AND COALESCE(TRIM(timeline.surface), '') <> 'direct'
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), 0),
	last_activity_at = (
		SELECT timeline.timestamp
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_channel_stats.workspace_id
			AND timeline.channel = network_channel_stats.channel
			AND timeline.kind <> 'greet'
			AND COALESCE(TRIM(timeline.peer_to), '') = ''
			AND COALESCE(TRIM(timeline.direct_id), '') = ''
			AND COALESCE(TRIM(timeline.surface), '') <> 'direct'
		ORDER BY timeline.sequence DESC
		LIMIT 1
	),
	last_presence_at = (
		SELECT timeline.timestamp
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_channel_stats.workspace_id
			AND timeline.channel = network_channel_stats.channel
			AND timeline.kind = 'greet'
		ORDER BY timeline.sequence DESC
		LIMIT 1
	),
	last_message_id = COALESCE((
		SELECT timeline.message_id
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_channel_stats.workspace_id
			AND timeline.channel = network_channel_stats.channel
			AND timeline.kind <> 'greet'
			AND COALESCE(TRIM(timeline.peer_to), '') = ''
			AND COALESCE(TRIM(timeline.direct_id), '') = ''
			AND COALESCE(TRIM(timeline.surface), '') <> 'direct'
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), ''),
	last_message_preview = COALESCE((
		SELECT COALESCE(NULLIF(TRIM(timeline.preview_text), ''), NULLIF(TRIM(timeline.text), ''), '')
		FROM network_timeline_log AS timeline
		WHERE timeline.workspace_id = network_channel_stats.workspace_id
			AND timeline.channel = network_channel_stats.channel
			AND timeline.kind <> 'greet'
			AND COALESCE(TRIM(timeline.peer_to), '') = ''
			AND COALESCE(TRIM(timeline.direct_id), '') = ''
			AND COALESCE(TRIM(timeline.surface), '') <> 'direct'
		ORDER BY timeline.sequence DESC
		LIMIT 1
	), '')`
