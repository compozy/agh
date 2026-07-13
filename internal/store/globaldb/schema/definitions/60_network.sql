CREATE TABLE network_audit_log (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			workspace_id TEXT NOT NULL,
			direction  TEXT NOT NULL,
			kind       TEXT NOT NULL,
		channel    TEXT NOT NULL,
		surface    TEXT,
		thread_id  TEXT,
		direct_id  TEXT,
		work_id    TEXT,
		peer_from  TEXT NOT NULL,
		peer_to    TEXT,
		message_id TEXT NOT NULL,
		reason     TEXT,
		size       INTEGER NOT NULL,
		timestamp  TEXT NOT NULL
	);

CREATE TABLE network_channel_kind_counts (
		workspace_id TEXT NOT NULL,
		channel      TEXT NOT NULL,
		kind         TEXT NOT NULL,
		message_count INTEGER NOT NULL CHECK (message_count >= 0),
		PRIMARY KEY (workspace_id, channel, kind)
	);

CREATE TABLE network_channel_participants (
		workspace_id TEXT NOT NULL,
		channel      TEXT NOT NULL,
		peer_id      TEXT NOT NULL,
		PRIMARY KEY (workspace_id, channel, peer_id)
	);

CREATE TABLE network_channel_stats (
		workspace_id                 TEXT NOT NULL,
		channel                      TEXT NOT NULL,
		message_count                INTEGER NOT NULL DEFAULT 0 CHECK (message_count >= 0),
		presence_count               INTEGER NOT NULL DEFAULT 0 CHECK (presence_count >= 0),
		historical_participant_count INTEGER NOT NULL DEFAULT 0 CHECK (historical_participant_count >= 0),
		last_activity_at             TEXT,
		last_presence_at             TEXT,
		last_message_id              TEXT NOT NULL DEFAULT '',
		last_message_preview         TEXT NOT NULL DEFAULT '', last_activity_sequence INTEGER NOT NULL DEFAULT 0, last_presence_sequence INTEGER NOT NULL DEFAULT 0, last_message_sequence INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (workspace_id, channel)
	);

CREATE TABLE network_channels (
			workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			channel      TEXT NOT NULL,
			purpose      TEXT NOT NULL,
			created_by   TEXT NOT NULL DEFAULT '',
			created_at   TEXT NOT NULL,
			updated_at   TEXT NOT NULL, fanout_policy TEXT NOT NULL DEFAULT 'capability_match' CHECK (
					fanout_policy IN ('capability_match', 'coordinator', 'all_members')
				), coordinator_peer_id TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (workspace_id, channel)
		);

CREATE TABLE network_delivery_guidance_state (
			session_id                   TEXT PRIMARY KEY,
			reply_guidance_delivered     BOOLEAN NOT NULL DEFAULT 0 CHECK (reply_guidance_delivered IN (0, 1)),
			protocol_guidance_delivered  BOOLEAN NOT NULL DEFAULT 0 CHECK (protocol_guidance_delivered IN (0, 1)),
			created_at                   TEXT NOT NULL,
			updated_at                   TEXT NOT NULL
		);

CREATE TABLE network_direct_rooms (
		workspace_id         TEXT NOT NULL,
		channel              TEXT NOT NULL,
		direct_id            TEXT NOT NULL,
		peer_a               TEXT NOT NULL,
		peer_b               TEXT NOT NULL,
		opened_at            TEXT NOT NULL,
		last_activity_at     TEXT NOT NULL,
		message_count        INTEGER NOT NULL DEFAULT 0 CHECK (message_count >= 0),
		open_work_count      INTEGER NOT NULL DEFAULT 0 CHECK (open_work_count >= 0),
		last_message_preview TEXT NOT NULL DEFAULT '', opened_sequence INTEGER NOT NULL DEFAULT 0, last_activity_sequence INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (workspace_id, channel, direct_id),
		UNIQUE (workspace_id, channel, peer_a, peer_b),
		CHECK (peer_a < peer_b)
	);

CREATE TABLE network_subscriptions (
			workspace_id          TEXT NOT NULL,
			channel               TEXT NOT NULL,
			thread_id             TEXT NOT NULL DEFAULT '',
			peer_id               TEXT NOT NULL,
			mode                  TEXT NOT NULL CHECK (mode IN ('mute', 'digest', 'full')),
			keyword_filters_json  TEXT NOT NULL DEFAULT '[]',
			created_at            TEXT NOT NULL,
			updated_at            TEXT NOT NULL,
			PRIMARY KEY (workspace_id, channel, thread_id, peer_id),
			FOREIGN KEY (workspace_id, channel)
				REFERENCES network_channels(workspace_id, channel)
				ON DELETE CASCADE
		);

CREATE TABLE network_task_thread_origins (
			task_id                 TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
			workspace_id            TEXT NOT NULL,
			channel                 TEXT NOT NULL,
			thread_id               TEXT NOT NULL,
			origin_message_id       TEXT NOT NULL,
			digest                  TEXT NOT NULL,
			source_message_ids_json TEXT NOT NULL DEFAULT '[]',
			created_at              TEXT NOT NULL,
			updated_at              TEXT NOT NULL,
			FOREIGN KEY (workspace_id, channel, thread_id)
				REFERENCES network_threads(workspace_id, channel, thread_id)
				ON DELETE CASCADE
		);

CREATE TABLE network_thread_participants (
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
	);

CREATE TABLE network_thread_peer_token_stats (
			workspace_id             TEXT NOT NULL,
			channel                  TEXT NOT NULL,
			thread_id                TEXT NOT NULL,
			peer_id                  TEXT NOT NULL,
			delivered_count          INTEGER NOT NULL DEFAULT 0 CHECK (delivered_count >= 0),
			prompt_size_bytes        INTEGER NOT NULL DEFAULT 0 CHECK (prompt_size_bytes >= 0),
			estimated_prompt_tokens  INTEGER NOT NULL DEFAULT 0 CHECK (estimated_prompt_tokens >= 0),
			first_delivered_at       TEXT NOT NULL,
			last_delivered_at        TEXT NOT NULL,
			updated_at              TEXT NOT NULL,
			PRIMARY KEY (workspace_id, channel, thread_id, peer_id),
			FOREIGN KEY (workspace_id, channel, thread_id)
				REFERENCES network_threads(workspace_id, channel, thread_id)
				ON DELETE CASCADE
		);

CREATE TABLE network_threads (
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
		last_message_preview TEXT NOT NULL DEFAULT '', opened_sequence INTEGER NOT NULL DEFAULT 0, last_activity_sequence INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (workspace_id, channel, thread_id)
	);

CREATE TABLE network_timeline_log (
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
);

CREATE TABLE network_work (
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
	);

CREATE INDEX idx_net_audit_conversation
			ON network_audit_log(workspace_id, channel, surface, thread_id, direct_id, timestamp);

CREATE INDEX idx_net_audit_ts ON network_audit_log(timestamp);

CREATE INDEX idx_net_audit_work
			ON network_audit_log(workspace_id, work_id, timestamp)
			WHERE work_id IS NOT NULL;

CREATE INDEX idx_net_audit_workspace_session ON network_audit_log(workspace_id, session_id);

CREATE INDEX idx_net_timeline_direct_sequence
		ON network_timeline_log(workspace_id, channel, direct_id, sequence)
		WHERE surface = 'direct';

CREATE INDEX idx_net_timeline_kind_sequence
		ON network_timeline_log(workspace_id, kind, sequence);

CREATE INDEX idx_net_timeline_presence_sequence
		ON network_timeline_log(workspace_id, channel, sequence)
		WHERE surface IS NULL;

CREATE INDEX idx_net_timeline_thread_sequence
		ON network_timeline_log(workspace_id, channel, thread_id, sequence)
		WHERE surface = 'thread';

CREATE INDEX idx_net_timeline_work_sequence
		ON network_timeline_log(workspace_id, work_id, sequence)
		WHERE work_id IS NOT NULL;

CREATE INDEX idx_network_channel_participants_peer
		ON network_channel_participants(workspace_id, peer_id, channel);

CREATE INDEX idx_network_channel_stats_activity
		ON network_channel_stats(workspace_id, last_activity_sequence DESC, channel);

CREATE INDEX idx_network_channels_updated_at ON network_channels(updated_at);

CREATE INDEX idx_network_channels_workspace ON network_channels(workspace_id);

CREATE INDEX idx_network_channels_workspace_updated_at
			ON network_channels(workspace_id, updated_at DESC, channel ASC);

CREATE INDEX idx_network_direct_rooms_activity
		ON network_direct_rooms(workspace_id, channel, last_activity_sequence DESC, direct_id);

CREATE INDEX idx_network_direct_rooms_created
		ON network_direct_rooms(workspace_id, channel, opened_at, direct_id);

CREATE INDEX idx_network_direct_rooms_open_work
		ON network_direct_rooms(workspace_id, channel, open_work_count, last_activity_sequence DESC, direct_id);

CREATE INDEX idx_network_direct_rooms_peer_a
		ON network_direct_rooms(workspace_id, channel, peer_a, last_activity_sequence DESC);

CREATE INDEX idx_network_direct_rooms_peer_b
		ON network_direct_rooms(workspace_id, channel, peer_b, last_activity_sequence DESC);

CREATE INDEX idx_network_subscriptions_peer
			ON network_subscriptions(workspace_id, peer_id, channel, thread_id);

CREATE INDEX idx_network_task_thread_origins_thread
			ON network_task_thread_origins(workspace_id, channel, thread_id, created_at DESC);

CREATE INDEX idx_network_thread_participants_peer
		ON network_thread_participants(workspace_id, peer_id, last_seen_at DESC);

CREATE INDEX idx_network_thread_peer_token_stats_peer
			ON network_thread_peer_token_stats(workspace_id, channel, peer_id, last_delivered_at DESC);

CREATE INDEX idx_network_threads_activity
		ON network_threads(workspace_id, channel, last_activity_sequence DESC, thread_id);

CREATE INDEX idx_network_threads_created
		ON network_threads(workspace_id, channel, opened_sequence, thread_id);

CREATE INDEX idx_network_threads_open_work
		ON network_threads(workspace_id, channel, open_work_count, last_activity_sequence DESC, thread_id);

CREATE INDEX idx_network_threads_title
		ON network_threads(workspace_id, channel, title COLLATE NOCASE, thread_id);

CREATE INDEX idx_network_work_conversation
		ON network_work(workspace_id, channel, surface, thread_id, direct_id, last_activity_at DESC);

CREATE INDEX idx_network_work_state
		ON network_work(workspace_id, state, last_activity_at DESC);
