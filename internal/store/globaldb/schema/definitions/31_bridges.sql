CREATE TABLE bridge_ingest_dedup (
		idempotency_key    TEXT PRIMARY KEY,
		bridge_instance_id TEXT NOT NULL REFERENCES bridge_instances(id) ON DELETE CASCADE,
		received_at        TEXT NOT NULL,
		expires_at         TEXT NOT NULL
	);

CREATE TABLE bridge_instances (
		id                TEXT PRIMARY KEY,
		scope             TEXT NOT NULL,
		workspace_id      TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
		platform          TEXT NOT NULL,
		extension_name    TEXT NOT NULL,
		display_name      TEXT NOT NULL,
		source            TEXT NOT NULL DEFAULT 'dynamic',
		enabled           BOOLEAN NOT NULL DEFAULT 1,
		status            TEXT NOT NULL,
		dm_policy         TEXT NOT NULL DEFAULT 'open',
		routing_policy    TEXT NOT NULL,
		provider_config   TEXT,
		delivery_defaults TEXT,
		notification_suppress BOOLEAN NOT NULL DEFAULT 0,
		degradation_reason TEXT,
		degradation_message TEXT,
		created_at        TEXT NOT NULL,
		updated_at        TEXT NOT NULL
	);

CREATE TABLE bridge_routes (
		routing_key_hash    TEXT PRIMARY KEY,
		scope               TEXT NOT NULL,
		workspace_id        TEXT,
		bridge_instance_id TEXT NOT NULL REFERENCES bridge_instances(id) ON DELETE CASCADE,
		peer_id             TEXT,
		thread_id           TEXT,
		group_id            TEXT,
		session_id          TEXT NOT NULL,
		agent_name          TEXT NOT NULL,
		last_activity_at    TEXT NOT NULL,
		created_at          TEXT NOT NULL,
		updated_at          TEXT NOT NULL
	);

CREATE TABLE bridge_secret_bindings (
		bridge_instance_id TEXT NOT NULL REFERENCES bridge_instances(id) ON DELETE CASCADE,
		binding_name        TEXT NOT NULL,
		secret_ref           TEXT NOT NULL,
		kind                TEXT NOT NULL,
		created_at          TEXT NOT NULL,
		updated_at          TEXT NOT NULL,
		PRIMARY KEY (bridge_instance_id, binding_name)
	);

CREATE TABLE bridge_target_directory (
			bridge_id       TEXT NOT NULL REFERENCES bridge_instances(id) ON DELETE CASCADE,
			canonical_route TEXT NOT NULL,
			display_name    TEXT NOT NULL,
			normalized      TEXT NOT NULL,
			target_type     TEXT NOT NULL CHECK (target_type IN ('channel','user','room','thread','group')),
			qualifier       TEXT NOT NULL DEFAULT '',
			capabilities    TEXT NOT NULL DEFAULT '',
			updated_at      TEXT NOT NULL,
			last_seen_at    TEXT,
			PRIMARY KEY (bridge_id, canonical_route)
		);

CREATE TABLE bridge_target_directory_refresh (
			bridge_id                  TEXT PRIMARY KEY REFERENCES bridge_instances(id) ON DELETE CASCADE,
			last_successful_refresh_at TEXT NOT NULL
		);

CREATE TABLE bridge_task_subscriptions (
			subscription_id    TEXT PRIMARY KEY,
			task_id            TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			bridge_instance_id TEXT NOT NULL REFERENCES bridge_instances(id) ON DELETE CASCADE,
			scope              TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
			workspace_id       TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
			peer_id            TEXT,
			thread_id          TEXT,
			group_id           TEXT,
			delivery_mode      TEXT NOT NULL CHECK (delivery_mode IN ('direct-send', 'reply')),
			created_by_kind    TEXT NOT NULL CHECK (
				created_by_kind IN (
					'human', 'agent_session', 'automation', 'extension', 'network_peer', 'daemon'
				)
			),
			created_by_ref     TEXT NOT NULL,
			created_at         TEXT NOT NULL,
			updated_at         TEXT NOT NULL,
			CHECK (
				(scope = 'global' AND workspace_id IS NULL) OR
				(scope = 'workspace' AND workspace_id IS NOT NULL)
			),
			CHECK (peer_id IS NOT NULL OR group_id IS NOT NULL)
		);

CREATE INDEX idx_bridge_ingest_dedup_expires ON bridge_ingest_dedup(expires_at);

CREATE INDEX idx_bridge_instances_scope ON bridge_instances(scope, workspace_id, id);

CREATE INDEX idx_bridge_routes_instance ON bridge_routes(bridge_instance_id, updated_at DESC);

CREATE INDEX idx_bridge_routes_session ON bridge_routes(session_id);

CREATE INDEX idx_bridge_secret_bindings_instance ON bridge_secret_bindings(bridge_instance_id);

CREATE INDEX idx_bridge_task_subscriptions_bridge
			ON bridge_task_subscriptions(bridge_instance_id, updated_at DESC);

CREATE INDEX idx_bridge_task_subscriptions_scope
			ON bridge_task_subscriptions(scope, workspace_id, updated_at DESC);

CREATE INDEX idx_bridge_task_subscriptions_task
			ON bridge_task_subscriptions(task_id, updated_at DESC);

CREATE INDEX idx_btd_bridge_norm
			ON bridge_target_directory(bridge_id, normalized);

CREATE INDEX idx_btd_bridge_qualifier
			ON bridge_target_directory(bridge_id, qualifier);
