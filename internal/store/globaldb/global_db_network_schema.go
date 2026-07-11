package globaldb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/compozy/agh/internal/store"
)

var networkConversationSchemaStatementsBeforeNetworkFeatureTail = []string{
	networkAuditLogTableStatement,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_ts ON network_audit_log(timestamp);`,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_workspace_session ON network_audit_log(workspace_id, session_id);`,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_conversation
			ON network_audit_log(workspace_id, channel, surface, thread_id, direct_id, timestamp);`,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_work
			ON network_audit_log(workspace_id, work_id, timestamp)
			WHERE work_id IS NOT NULL;`,
	networkTimelineLogPrePolicyMentionsStatement,
	`CREATE INDEX IF NOT EXISTS idx_net_timeline_thread_ts
			ON network_timeline_log(workspace_id, channel, thread_id, timestamp, message_id)
			WHERE surface = 'thread';`,
	`CREATE INDEX IF NOT EXISTS idx_net_timeline_direct_ts
			ON network_timeline_log(workspace_id, channel, direct_id, timestamp, message_id)
			WHERE surface = 'direct';`,
	`CREATE INDEX IF NOT EXISTS idx_net_timeline_work_ts
			ON network_timeline_log(workspace_id, work_id, timestamp, message_id)
			WHERE work_id IS NOT NULL;`,
	`CREATE INDEX IF NOT EXISTS idx_net_timeline_presence_ts
			ON network_timeline_log(workspace_id, channel, timestamp, message_id)
			WHERE surface IS NULL;`,
	`CREATE INDEX IF NOT EXISTS idx_net_timeline_kind_ts
			ON network_timeline_log(workspace_id, kind, timestamp, message_id);`,
	`CREATE TABLE IF NOT EXISTS network_threads (
		workspace_id         TEXT NOT NULL,
		channel              TEXT NOT NULL,
		thread_id            TEXT NOT NULL,
		root_message_id      TEXT NOT NULL,
		title                TEXT NOT NULL DEFAULT '',
		opened_by_peer_id    TEXT NOT NULL DEFAULT '',
		opened_session_id    TEXT NOT NULL DEFAULT '',
		opened_at            TEXT NOT NULL,
		last_activity_at     TEXT NOT NULL,
		message_count        INTEGER NOT NULL DEFAULT 0 CHECK (message_count >= 0),
		participant_count    INTEGER NOT NULL DEFAULT 0 CHECK (participant_count >= 0),
		open_work_count      INTEGER NOT NULL DEFAULT 0 CHECK (open_work_count >= 0),
		last_message_preview TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (workspace_id, channel, thread_id)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_network_threads_activity
		ON network_threads(workspace_id, channel, last_activity_at DESC, thread_id);`,
	`CREATE TABLE IF NOT EXISTS network_thread_participants (
		workspace_id     TEXT NOT NULL,
		channel          TEXT NOT NULL,
		thread_id        TEXT NOT NULL,
		peer_id          TEXT NOT NULL,
		first_message_id TEXT NOT NULL,
		first_seen_at    TEXT NOT NULL,
		last_seen_at     TEXT NOT NULL,
		PRIMARY KEY (workspace_id, channel, thread_id, peer_id),
		FOREIGN KEY (workspace_id, channel, thread_id)
			REFERENCES network_threads(workspace_id, channel, thread_id)
			ON DELETE CASCADE
	);`,
	`CREATE INDEX IF NOT EXISTS idx_network_thread_participants_peer
		ON network_thread_participants(workspace_id, peer_id, last_seen_at DESC);`,
	`CREATE TABLE IF NOT EXISTS network_direct_rooms (
		workspace_id         TEXT NOT NULL,
		channel              TEXT NOT NULL,
		direct_id            TEXT NOT NULL,
		peer_a               TEXT NOT NULL,
		peer_b               TEXT NOT NULL,
		opened_at            TEXT NOT NULL,
		last_activity_at     TEXT NOT NULL,
		message_count        INTEGER NOT NULL DEFAULT 0 CHECK (message_count >= 0),
		open_work_count      INTEGER NOT NULL DEFAULT 0 CHECK (open_work_count >= 0),
		last_message_preview TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (workspace_id, channel, direct_id),
		UNIQUE (workspace_id, channel, peer_a, peer_b),
		CHECK (peer_a < peer_b)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_network_direct_rooms_activity
		ON network_direct_rooms(workspace_id, channel, last_activity_at DESC, direct_id);`,
	`CREATE INDEX IF NOT EXISTS idx_network_direct_rooms_peer_a
		ON network_direct_rooms(workspace_id, channel, peer_a, last_activity_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_network_direct_rooms_peer_b
		ON network_direct_rooms(workspace_id, channel, peer_b, last_activity_at DESC);`,
	`CREATE TABLE IF NOT EXISTS network_work (
		work_id           TEXT NOT NULL,
		workspace_id      TEXT NOT NULL,
		channel           TEXT NOT NULL,
		surface           TEXT NOT NULL CHECK (surface IN ('thread', 'direct')),
		thread_id         TEXT,
		direct_id         TEXT,
		opened_by_peer_id TEXT NOT NULL,
		opened_session_id TEXT NOT NULL DEFAULT '',
		target_peer_id    TEXT NOT NULL DEFAULT '',
		state             TEXT NOT NULL CHECK (
			state IN ('submitted', 'working', 'needs_input', 'completed', 'failed', 'canceled')
		),
		opened_at         TEXT NOT NULL,
		last_activity_at  TEXT NOT NULL,
		terminal_at       TEXT,
		CHECK (
			(surface = 'thread' AND thread_id IS NOT NULL AND direct_id IS NULL)
			OR (surface = 'direct' AND direct_id IS NOT NULL AND thread_id IS NULL)
		),
		PRIMARY KEY (workspace_id, work_id),
		FOREIGN KEY (workspace_id, channel, thread_id)
			REFERENCES network_threads(workspace_id, channel, thread_id)
			ON DELETE RESTRICT,
		FOREIGN KEY (workspace_id, channel, direct_id)
			REFERENCES network_direct_rooms(workspace_id, channel, direct_id)
			ON DELETE RESTRICT
	);`,
	`CREATE INDEX IF NOT EXISTS idx_network_work_conversation
		ON network_work(workspace_id, channel, surface, thread_id, direct_id, last_activity_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_network_work_state
		ON network_work(workspace_id, state, last_activity_at DESC);`,
}

var networkChannelProjectionsMigration = store.Migration{
	Version:  67,
	Name:     "add_network_channel_projections",
	Up:       migrateNetworkChannelProjections,
	Checksum: "2026-07-10-add-network-channel-projections",
}

var networkChannelProjectionSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS network_channel_stats (
		workspace_id                 TEXT NOT NULL,
		channel                      TEXT NOT NULL,
		message_count                INTEGER NOT NULL DEFAULT 0 CHECK (message_count >= 0),
		presence_count               INTEGER NOT NULL DEFAULT 0 CHECK (presence_count >= 0),
		historical_participant_count INTEGER NOT NULL DEFAULT 0 CHECK (historical_participant_count >= 0),
		last_activity_at             TEXT,
		last_presence_at             TEXT,
		last_message_id              TEXT NOT NULL DEFAULT '',
		last_message_preview         TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (workspace_id, channel)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_network_channel_stats_activity
		ON network_channel_stats(workspace_id, last_activity_at DESC, channel);`,
	`CREATE TABLE IF NOT EXISTS network_channel_participants (
		workspace_id TEXT NOT NULL,
		channel      TEXT NOT NULL,
		peer_id      TEXT NOT NULL,
		PRIMARY KEY (workspace_id, channel, peer_id)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_network_channel_participants_peer
		ON network_channel_participants(workspace_id, peer_id, channel);`,
	`CREATE TABLE IF NOT EXISTS network_channel_kind_counts (
		workspace_id TEXT NOT NULL,
		channel      TEXT NOT NULL,
		kind         TEXT NOT NULL,
		message_count INTEGER NOT NULL CHECK (message_count >= 0),
		PRIMARY KEY (workspace_id, channel, kind)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_network_threads_created
		ON network_threads(workspace_id, channel, opened_at, thread_id);`,
	`CREATE INDEX IF NOT EXISTS idx_network_threads_title
		ON network_threads(workspace_id, channel, title COLLATE NOCASE, thread_id);`,
	`CREATE INDEX IF NOT EXISTS idx_network_threads_open_work
		ON network_threads(workspace_id, channel, open_work_count, last_activity_at DESC, thread_id);`,
	`CREATE INDEX IF NOT EXISTS idx_network_direct_rooms_created
		ON network_direct_rooms(workspace_id, channel, opened_at, direct_id);`,
	`CREATE INDEX IF NOT EXISTS idx_network_direct_rooms_open_work
		ON network_direct_rooms(workspace_id, channel, open_work_count, last_activity_at DESC, direct_id);`,
}

func migrateNetworkChannelProjections(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range networkChannelProjectionSchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: create network channel projections: %w", err)
		}
	}
	for _, statement := range []string{
		`DELETE FROM network_channel_kind_counts`,
		`DELETE FROM network_channel_participants`,
		`DELETE FROM network_channel_stats`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: clear network channel projection: %w", err)
		}
	}
	if err := backfillNetworkChannelStats(ctx, tx); err != nil {
		return err
	}
	if err := backfillNetworkChannelParticipants(ctx, tx); err != nil {
		return err
	}
	return backfillNetworkChannelKindCounts(ctx, tx)
}

func backfillNetworkChannelStats(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO network_channel_stats (
		workspace_id, channel, message_count, presence_count, historical_participant_count,
		last_activity_at, last_presence_at, last_message_id, last_message_preview
	)
	SELECT
		t.workspace_id,
		t.channel,
		SUM(CASE WHEN t.kind <> 'greet'
			AND COALESCE(TRIM(t.peer_to), '') = ''
			AND COALESCE(TRIM(t.direct_id), '') = ''
			AND COALESCE(TRIM(t.surface), '') <> 'direct'
			THEN 1 ELSE 0 END),
		SUM(CASE WHEN t.kind = 'greet' THEN 1 ELSE 0 END),
		0,
		MAX(CASE WHEN t.kind <> 'greet'
			AND COALESCE(TRIM(t.peer_to), '') = ''
			AND COALESCE(TRIM(t.direct_id), '') = ''
			AND COALESCE(TRIM(t.surface), '') <> 'direct'
			THEN t.timestamp END),
		MAX(CASE WHEN t.kind = 'greet' THEN t.timestamp END),
		COALESCE((
			SELECT latest.message_id
			FROM network_timeline_log AS latest
			WHERE latest.workspace_id = t.workspace_id
				AND latest.channel = t.channel
				AND latest.kind <> 'greet'
				AND COALESCE(TRIM(latest.peer_to), '') = ''
				AND COALESCE(TRIM(latest.direct_id), '') = ''
				AND COALESCE(TRIM(latest.surface), '') <> 'direct'
			ORDER BY latest.timestamp DESC, latest.message_id DESC
			LIMIT 1
		), ''),
		COALESCE((
			SELECT COALESCE(NULLIF(TRIM(latest.preview_text), ''), NULLIF(TRIM(latest.text), ''), '')
			FROM network_timeline_log AS latest
			WHERE latest.workspace_id = t.workspace_id
				AND latest.channel = t.channel
				AND latest.kind <> 'greet'
				AND COALESCE(TRIM(latest.peer_to), '') = ''
				AND COALESCE(TRIM(latest.direct_id), '') = ''
				AND COALESCE(TRIM(latest.surface), '') <> 'direct'
			ORDER BY latest.timestamp DESC, latest.message_id DESC
			LIMIT 1
		), '')
	FROM network_timeline_log AS t
	GROUP BY t.workspace_id, t.channel`)
	if err != nil {
		return fmt.Errorf("store: backfill network channel stats: %w", err)
	}
	return nil
}

func backfillNetworkChannelParticipants(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO network_channel_participants (workspace_id, channel, peer_id)
		SELECT workspace_id, channel, peer_id
		FROM (
			SELECT workspace_id, channel, TRIM(peer_from) AS peer_id FROM network_timeline_log
			UNION
			SELECT workspace_id, channel, TRIM(peer_to) AS peer_id FROM network_timeline_log
			UNION
			SELECT timeline.workspace_id, timeline.channel, TRIM(CAST(mention.value AS TEXT)) AS peer_id
			FROM network_timeline_log AS timeline, json_each(timeline.mentions_json) AS mention
		)
		WHERE peer_id <> ''`); err != nil {
		return fmt.Errorf("store: backfill network channel participants: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE network_channel_stats
		SET historical_participant_count = (
			SELECT COUNT(*)
			FROM network_channel_participants AS participant
			WHERE participant.workspace_id = network_channel_stats.workspace_id
				AND participant.channel = network_channel_stats.channel
		)`); err != nil {
		return fmt.Errorf("store: backfill network channel participant totals: %w", err)
	}
	return nil
}

func backfillNetworkChannelKindCounts(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO network_channel_kind_counts (
		workspace_id, channel, kind, message_count
	)
	SELECT workspace_id, channel, kind, COUNT(*)
	FROM network_timeline_log AS timeline
	WHERE timeline.kind <> 'greet'
		AND COALESCE(TRIM(timeline.peer_to), '') = ''
		AND COALESCE(TRIM(timeline.direct_id), '') = ''
		AND COALESCE(TRIM(timeline.surface), '') <> 'direct'
	GROUP BY workspace_id, channel, kind`)
	if err != nil {
		return fmt.Errorf("store: backfill network channel kind counts: %w", err)
	}
	return nil
}
