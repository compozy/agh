package globaldb

func bridgeTaskSubscriptionSchemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS bridge_task_subscriptions (
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
		);`,
		`CREATE INDEX IF NOT EXISTS idx_bridge_task_subscriptions_task
			ON bridge_task_subscriptions(task_id, updated_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_bridge_task_subscriptions_bridge
			ON bridge_task_subscriptions(bridge_instance_id, updated_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_bridge_task_subscriptions_scope
			ON bridge_task_subscriptions(scope, workspace_id, updated_at DESC);`,
	}
}
