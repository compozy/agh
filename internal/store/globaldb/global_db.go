package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/compozy/agh/internal/resources"
	"github.com/compozy/agh/internal/store"
	aghworkspace "github.com/compozy/agh/internal/workspace"
)

const (
	modelCatalogSourceConstraintChecksum = "2026-05-07-rebuild-model-catalog-source-constraints"
	idxSummaryActorSQL                   = "CREATE INDEX IF NOT EXISTS idx_summaries_actor " +
		"ON event_summaries(actor_kind, actor_id);"
	idxSummaryHookEventSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_hook_event " +
		"ON event_summaries(hook_event);"
	idxSummaryOutcomeTimestampSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_outcome_timestamp " +
		"ON event_summaries(outcome, timestamp DESC);"
	idxSummaryParentSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_parent " +
		"ON event_summaries(parent_session_id);"
	idxSummaryProviderTimestampSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_provider_timestamp " +
		"ON event_summaries(provider, timestamp DESC);"
	idxSummaryRootSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_root " +
		"ON event_summaries(root_session_id);"
	idxSummaryRunSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_run " +
		"ON event_summaries(run_id);"
	idxSummarySessionSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_session " +
		"ON event_summaries(session_id);"
	idxSummaryTaskSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_task " +
		"ON event_summaries(task_id);"
	idxSummaryTimeSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_timestamp " +
		"ON event_summaries(timestamp);"
	idxSummaryTypeSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_type " +
		"ON event_summaries(type);"
	idxSummaryWorkflowSQL = "CREATE INDEX IF NOT EXISTS idx_summaries_workflow " +
		"ON event_summaries(workflow_id);"
	globalDBActorIDKey                              = "actor_id"
	globalDBActorRefKey                             = "actor_ref"
	globalDBAddAgentHeartbeatStorageKey             = "add_agent_heartbeat_storage"
	globalDBAddAgentSoulSnapshotsKey                = "add_agent_soul_snapshots"
	globalDBAttachExpiresAtColumn                   = "attach_expires_at"
	globalDBAutoStopOnParentKey                     = "auto_stop_on_parent"
	globalDBClaimTokenKey                           = "claim_token"
	globalDBClaimTokenHashKey                       = "claim_token_hash"
	globalDBLeaseUntilKey                           = "lease_until"
	globalDBOutcomeKey                              = "outcome"
	globalDBParentSessionIDKey                      = "parent_session_id"
	globalDBPermissionPolicyJSONKey                 = "permission_policy_json"
	globalDBExtensionProvenanceJSONKey              = "provenance_json"
	globalDBBridgeNotificationSuppressColumn        = "notification_suppress"
	globalDBRebuildModelCatalogSourceConstraintsKey = "rebuild_model_catalog_source_constraints"
	globalDBRootSessionIDKey                        = "root_session_id"
	globalDBScopeKey                                = "scope"
	globalDBSpawnBudgetJSONKey                      = "spawn_budget_json"
	globalDBSpawnDepthKey                           = "spawn_depth"
	globalDBSpawnRoleKey                            = "spawn_role"
	globalDBSessionAttachedToColumn                 = "attached_to"
	globalDBSessionStateActive                      = "active"
	globalDBSessionStateStopped                     = "stopped"
	globalDBSummaryKey                              = "summary"
	globalDBTaskEventsKey                           = "task_events"
	globalDBTTLExpiresAtKey                         = "ttl_expires_at"
	globalDBWorkspaceKey                            = "workspace"
)

const globalMemoryEventWriteCommitted = "memory.write.committed"

var taskTableIndexStatements = []string{
	`CREATE INDEX IF NOT EXISTS idx_tasks_scope ON tasks(scope);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_workspace ON tasks(workspace_id);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_approval_state ON tasks(approval_state);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_owner ON tasks(owner_kind, owner_ref);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_channel ON tasks(network_channel);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_paused ON tasks(paused, updated_at DESC);`,
	taskCurrentRunIndexStatement,
}

var taskEventIndexStatements = []string{
	`CREATE INDEX IF NOT EXISTS idx_task_events_task ON task_events(task_id, timestamp DESC, id DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_task_events_run ON task_events(run_id, timestamp DESC, id DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_task_events_type ON task_events(event_type, timestamp DESC, id DESC);`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_task_events_event_seq ON task_events(event_seq);`,
	`CREATE INDEX IF NOT EXISTS idx_task_events_task_seq ON task_events(task_id, event_seq ASC);`,
}

const networkAuditLogTableStatement = `CREATE TABLE IF NOT EXISTS network_audit_log (
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
	);`

const networkTimelineLogTableStatement = `CREATE TABLE IF NOT EXISTS network_timeline_log (
			message_id    TEXT NOT NULL,
			session_id    TEXT,
			workspace_id  TEXT NOT NULL,
			channel       TEXT NOT NULL,
		surface       TEXT CHECK (surface IN ('thread', 'direct') OR surface IS NULL),
		thread_id     TEXT,
		direct_id     TEXT,
		direction     TEXT NOT NULL,
		peer_from     TEXT NOT NULL,
		peer_to       TEXT,
		kind          TEXT NOT NULL,
		work_id       TEXT,
		reply_to      TEXT,
		trace_id      TEXT,
		causation_id  TEXT,
		intent        TEXT,
		text          TEXT,
		preview_text  TEXT NOT NULL DEFAULT '',
		body_json     TEXT NOT NULL,
		timestamp     TEXT NOT NULL,
		CHECK (
			(surface IS NULL AND thread_id IS NULL AND direct_id IS NULL AND work_id IS NULL AND kind IN ('greet', 'whois'))
			OR (surface = 'thread' AND thread_id IS NOT NULL AND direct_id IS NULL)
			OR (surface = 'direct' AND direct_id IS NOT NULL AND thread_id IS NULL)
		),
			CHECK (kind IN ('greet', 'whois', 'say', 'capability', 'receipt', 'trace')),
			PRIMARY KEY (workspace_id, message_id)
		);`

var networkConversationSchemaStatements = []string{
	networkAuditLogTableStatement,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_ts ON network_audit_log(timestamp);`,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_workspace_session ON network_audit_log(workspace_id, session_id);`,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_conversation
			ON network_audit_log(workspace_id, channel, surface, thread_id, direct_id, timestamp);`,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_work
			ON network_audit_log(workspace_id, work_id, timestamp)
			WHERE work_id IS NOT NULL;`,
	networkTimelineLogTableStatement,
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

var globalSchemaStatements = appendSchemaStatements(
	[]string{
		`CREATE TABLE IF NOT EXISTS workspaces (
		id            TEXT PRIMARY KEY,
		root_dir      TEXT NOT NULL UNIQUE,
		add_dirs      TEXT NOT NULL DEFAULT '[]',
		name          TEXT NOT NULL UNIQUE,
		default_agent TEXT DEFAULT '',
		sandbox_ref TEXT NOT NULL DEFAULT '',
		created_at    TEXT NOT NULL,
		updated_at    TEXT NOT NULL
	);`,
		`CREATE INDEX IF NOT EXISTS idx_workspaces_name ON workspaces(name);`,
		`CREATE TABLE IF NOT EXISTS sessions (
		id             TEXT PRIMARY KEY,
		name           TEXT,
		agent_name     TEXT NOT NULL,
		provider       TEXT NOT NULL DEFAULT '',
		workspace_id   TEXT NOT NULL REFERENCES workspaces(id),
		session_type   TEXT NOT NULL DEFAULT 'user',
		channel          TEXT NOT NULL DEFAULT '',
		state          TEXT NOT NULL,
		acp_session_id TEXT,
		stop_reason    TEXT,
		stop_detail    TEXT,
		subprocess_pid INTEGER NOT NULL DEFAULT 0,
		subprocess_started_at TEXT,
		last_update_at TEXT,
		stall_state    TEXT NOT NULL DEFAULT '',
		stall_reason   TEXT NOT NULL DEFAULT '',
		activity_json  TEXT NOT NULL DEFAULT '',
		attached_to    TEXT NOT NULL DEFAULT '',
		attach_expires_at TEXT,
		sandbox_id TEXT NOT NULL DEFAULT '',
		sandbox_backend TEXT NOT NULL DEFAULT 'local',
		sandbox_profile TEXT NOT NULL DEFAULT '',
		sandbox_instance_id TEXT NOT NULL DEFAULT '',
		sandbox_state TEXT NOT NULL DEFAULT '',
		sandbox_provider_state_json TEXT NOT NULL DEFAULT '',
		sandbox_last_sync_at TEXT,
		sandbox_last_sync_error TEXT NOT NULL DEFAULT '',
		created_at     TEXT NOT NULL,
		updated_at     TEXT NOT NULL
	);`,
		`CREATE TABLE IF NOT EXISTS event_summaries (
		id                     TEXT PRIMARY KEY,
		session_id             TEXT NOT NULL DEFAULT '',
		workspace_id           TEXT NOT NULL DEFAULT '',
		type                   TEXT NOT NULL,
		agent_name             TEXT NOT NULL DEFAULT '',
		content_json           TEXT NOT NULL DEFAULT '',
		task_id                TEXT NOT NULL DEFAULT '',
		run_id                 TEXT NOT NULL DEFAULT '',
		workflow_id            TEXT NOT NULL DEFAULT '',
		claim_token_hash       TEXT NOT NULL DEFAULT '',
		lease_until            TEXT NOT NULL DEFAULT '',
		coordinator_session_id TEXT NOT NULL DEFAULT '',
		scheduler_reason       TEXT NOT NULL DEFAULT '',
		hook_event             TEXT NOT NULL DEFAULT '',
		hook_name              TEXT NOT NULL DEFAULT '',
		actor_kind             TEXT NOT NULL DEFAULT '',
		actor_id               TEXT NOT NULL DEFAULT '',
		release_reason         TEXT NOT NULL DEFAULT '',
		parent_session_id      TEXT NOT NULL DEFAULT '',
		root_session_id        TEXT NOT NULL DEFAULT '',
		spawn_depth            INTEGER NOT NULL DEFAULT 0,
		summary                TEXT,
		timestamp              TEXT NOT NULL
	);`,
		idxSummarySessionSQL,
		idxSummaryTypeSQL,
		idxSummaryTimeSQL,
		idxSummaryTaskSQL,
		idxSummaryRunSQL,
		idxSummaryWorkflowSQL,
		idxSummaryHookEventSQL,
		idxSummaryActorSQL,
		`CREATE TABLE IF NOT EXISTS memory_operation_log (
		id         TEXT PRIMARY KEY,
		type       TEXT NOT NULL,
		agent_name TEXT NOT NULL DEFAULT 'daemon',
		summary    TEXT NOT NULL DEFAULT '',
		timestamp  TEXT NOT NULL
	);`,
		`CREATE INDEX IF NOT EXISTS idx_memory_operation_log_type ON memory_operation_log(type);`,
		`CREATE INDEX IF NOT EXISTS idx_memory_operation_log_timestamp ON memory_operation_log(timestamp);`,
		`CREATE TABLE IF NOT EXISTS token_stats (
		id            TEXT PRIMARY KEY,
		session_id    TEXT NOT NULL REFERENCES sessions(id),
		agent_name    TEXT NOT NULL,
		input_tokens  INTEGER,
		output_tokens INTEGER,
		total_tokens  INTEGER,
		total_cost    REAL,
		cost_currency TEXT,
		turn_count    INTEGER NOT NULL DEFAULT 0,
		updated_at    TEXT NOT NULL
	);`,
		`CREATE INDEX IF NOT EXISTS idx_token_stats_session ON token_stats(session_id);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_token_stats_session_agent ON token_stats(session_id, agent_name);`,
		`CREATE TABLE IF NOT EXISTS permission_log (
		id          TEXT PRIMARY KEY,
		session_id  TEXT NOT NULL REFERENCES sessions(id),
		agent_name  TEXT NOT NULL,
		action      TEXT NOT NULL,
		resource    TEXT NOT NULL,
		decision    TEXT NOT NULL,
		policy_used TEXT NOT NULL,
		timestamp   TEXT NOT NULL
	);`,
		`CREATE INDEX IF NOT EXISTS idx_perm_session ON permission_log(session_id);`,
		`CREATE TABLE IF NOT EXISTS network_channels (
			workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			channel      TEXT NOT NULL,
			purpose      TEXT NOT NULL,
			created_by   TEXT NOT NULL DEFAULT '',
			created_at   TEXT NOT NULL,
			updated_at   TEXT NOT NULL,
			PRIMARY KEY (workspace_id, channel)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_network_channels_workspace ON network_channels(workspace_id);`,
		`CREATE INDEX IF NOT EXISTS idx_network_channels_updated_at ON network_channels(updated_at);`,
		`CREATE INDEX IF NOT EXISTS idx_network_channels_workspace_updated_at ON network_channels(workspace_id, updated_at DESC, channel ASC);`,
		`CREATE TABLE IF NOT EXISTS extensions (
		name          TEXT PRIMARY KEY,
		version       TEXT NOT NULL,
		source        TEXT NOT NULL,
		enabled       BOOLEAN NOT NULL DEFAULT 1,
		manifest_path TEXT NOT NULL,
		installed_at  TEXT NOT NULL,
		capabilities  TEXT NOT NULL DEFAULT '{}',
		actions       TEXT NOT NULL DEFAULT '{}',
		checksum      TEXT NOT NULL,
		registry_slug TEXT,
		registry_name TEXT,
		remote_version TEXT,
		` + globalDBExtensionProvenanceJSONKey + ` TEXT NOT NULL DEFAULT '{}'
	);`,
		`CREATE TABLE IF NOT EXISTS automation_jobs (
		id           TEXT PRIMARY KEY,
		scope        TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
		name         TEXT NOT NULL,
		agent_name   TEXT NOT NULL,
		workspace_id TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
		prompt       TEXT NOT NULL,
		schedule     TEXT,
		task         TEXT,
		enabled      BOOLEAN NOT NULL DEFAULT 1,
		retry        TEXT NOT NULL,
		fire_limit   TEXT NOT NULL,
		source       TEXT NOT NULL DEFAULT 'dynamic',
		created_at   TEXT NOT NULL,
		updated_at   TEXT NOT NULL,
		CHECK (
			(scope = 'global' AND workspace_id IS NULL) OR
			(scope = 'workspace' AND workspace_id IS NOT NULL)
		)
	);`,
		`CREATE TABLE IF NOT EXISTS automation_triggers (
		id            TEXT PRIMARY KEY,
		scope         TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
		name          TEXT NOT NULL,
		agent_name    TEXT NOT NULL,
		workspace_id  TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
		prompt        TEXT NOT NULL,
		event         TEXT NOT NULL,
		filter        TEXT,
		enabled       BOOLEAN NOT NULL DEFAULT 1,
		retry         TEXT NOT NULL,
		fire_limit    TEXT NOT NULL,
		source        TEXT NOT NULL DEFAULT 'dynamic',
		webhook_id    TEXT,
		endpoint_slug TEXT,
		webhook_secret_ref TEXT,
		created_at    TEXT NOT NULL,
		updated_at    TEXT NOT NULL,
		CHECK (
			(scope = 'global' AND workspace_id IS NULL) OR
			(scope = 'workspace' AND workspace_id IS NOT NULL)
		)
	);`,
		`CREATE TABLE IF NOT EXISTS automation_runs (
		id         TEXT PRIMARY KEY,
		job_id     TEXT,
		trigger_id TEXT,
		session_id TEXT,
		task_id    TEXT,
		task_run_id TEXT,
		status     TEXT NOT NULL,
		attempt    INTEGER NOT NULL DEFAULT 1,
		started_at TEXT,
		ended_at   TEXT,
		error      TEXT
	);`,
		`CREATE TABLE IF NOT EXISTS automation_job_overlays (
		job_id            TEXT PRIMARY KEY,
		enabled_override  BOOLEAN NOT NULL,
		updated_at        TEXT NOT NULL
	);`,
		`CREATE TABLE IF NOT EXISTS automation_trigger_overlays (
		trigger_id        TEXT PRIMARY KEY,
		enabled_override  BOOLEAN NOT NULL,
		updated_at        TEXT NOT NULL
	);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_automation_jobs_global_name ON automation_jobs(name) WHERE scope = 'global';`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_automation_jobs_workspace_name ON automation_jobs(workspace_id, name) WHERE scope = 'workspace';`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_automation_triggers_global_name ON automation_triggers(name) WHERE scope = 'global';`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_automation_triggers_workspace_name ON automation_triggers(workspace_id, name) WHERE scope = 'workspace';`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_automation_triggers_webhook_id ON automation_triggers(webhook_id) WHERE webhook_id IS NOT NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_automation_jobs_enabled ON automation_jobs(enabled);`,
		`CREATE INDEX IF NOT EXISTS idx_automation_triggers_enabled ON automation_triggers(enabled);`,
		`CREATE INDEX IF NOT EXISTS idx_automation_triggers_event ON automation_triggers(event);`,
		`CREATE INDEX IF NOT EXISTS idx_automation_runs_job ON automation_runs(job_id);`,
		`CREATE INDEX IF NOT EXISTS idx_automation_runs_trigger ON automation_runs(trigger_id);`,
		`CREATE INDEX IF NOT EXISTS idx_automation_runs_status ON automation_runs(status);`,
		`CREATE INDEX IF NOT EXISTS idx_automation_runs_started ON automation_runs(started_at);`,
		`CREATE TABLE IF NOT EXISTS tasks (
		id              TEXT PRIMARY KEY,
		identifier      TEXT,
		scope           TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
		workspace_id    TEXT REFERENCES workspaces(id) ON DELETE CASCADE,
		parent_task_id  TEXT REFERENCES tasks(id),
		network_channel TEXT,
		title           TEXT NOT NULL,
		description     TEXT,
		priority        TEXT NOT NULL DEFAULT 'medium' CHECK (
			priority IN ('low', 'medium', 'high', 'urgent')
		),
		max_attempts    INTEGER NOT NULL DEFAULT 3 CHECK (max_attempts > 0 AND max_attempts <= 10),
		status          TEXT NOT NULL CHECK (
			status IN (
				'draft', 'pending', 'blocked', 'ready', 'in_progress', 'completed', 'failed', 'canceled'
			)
		),
		approval_policy TEXT NOT NULL DEFAULT 'none' CHECK (
			approval_policy IN ('none', 'manual')
		),
		approval_state  TEXT NOT NULL DEFAULT 'not_required' CHECK (
			approval_state IN ('not_required', 'pending', 'approved', 'rejected')
		),
		owner_kind      TEXT CHECK (
			owner_kind IS NULL OR owner_kind IN (
				'human', 'agent_session', 'automation', 'extension', 'network_peer', 'pool'
			)
		),
		owner_ref       TEXT,
		created_by_kind TEXT NOT NULL CHECK (
			created_by_kind IN (
				'human', 'agent_session', 'automation', 'extension', 'network_peer', 'daemon'
			)
		),
		created_by_ref  TEXT NOT NULL,
		origin_kind     TEXT NOT NULL CHECK (
			origin_kind IN (
				'cli', 'web', 'uds', 'http', 'automation', 'extension', 'network', 'agent_session', 'daemon'
			)
		),
		origin_ref      TEXT NOT NULL,
		created_at      TEXT NOT NULL,
		updated_at      TEXT NOT NULL,
		closed_at       TEXT,
		metadata_json   TEXT,
		current_run_id  TEXT REFERENCES task_runs(id) ON DELETE SET NULL,
		paused          INTEGER NOT NULL DEFAULT 0 CHECK (paused IN (0, 1)),
		paused_by       TEXT NOT NULL DEFAULT '',
		paused_at       TEXT,
		paused_reason   TEXT NOT NULL DEFAULT '',
		max_runtime_seconds INTEGER NOT NULL DEFAULT 0 CHECK (max_runtime_seconds >= 0),
		spawn_failure_count INTEGER NOT NULL DEFAULT 0 CHECK (spawn_failure_count >= 0),
		last_spawn_error TEXT NOT NULL DEFAULT '',
		review_policy TEXT NOT NULL DEFAULT 'none' CHECK (
			review_policy IN ('none', 'on_success', 'on_failure', 'always')
		),
		review_max_rounds INTEGER NOT NULL DEFAULT 3 CHECK (review_max_rounds >= 0),
		review_round INTEGER NOT NULL DEFAULT 0 CHECK (review_round >= 0),
		last_review_id TEXT,
		last_review_outcome TEXT CHECK (
			last_review_outcome IS NULL OR last_review_outcome IN (
				'approved', 'rejected', 'blocked', 'error', 'timeout', 'invalid_output'
			)
		),
		review_circuit_opened_at TEXT,
		review_circuit_reason TEXT,
		CHECK (
			(scope = 'global' AND workspace_id IS NULL) OR
			(scope = 'workspace' AND workspace_id IS NOT NULL)
		),
		CHECK (
			(owner_kind IS NULL AND owner_ref IS NULL) OR
			(owner_kind IS NOT NULL AND owner_ref IS NOT NULL)
		),
		CHECK (parent_task_id IS NULL OR parent_task_id <> id),
		CHECK (
			(approval_policy = 'none' AND approval_state = 'not_required') OR
			(approval_policy = 'manual' AND approval_state IN ('pending', 'approved', 'rejected'))
		)
	);`,
		taskTableIndexStatements[0],
		taskTableIndexStatements[1],
		taskTableIndexStatements[2],
		taskTableIndexStatements[3],
		taskTableIndexStatements[4],
		taskTableIndexStatements[5],
		taskTableIndexStatements[6],
		taskTableIndexStatements[7],
		taskTableIndexStatements[8],
		`CREATE TABLE IF NOT EXISTS task_runs (
		id              TEXT PRIMARY KEY,
		task_id         TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		status          TEXT NOT NULL CHECK (
			status IN (
				'queued', 'claimed', 'starting', 'running', 'completed', 'failed', 'canceled'
			)
		),
		attempt         INTEGER NOT NULL CHECK (attempt > 0),
		previous_run_id TEXT,
		failure_kind    TEXT NOT NULL DEFAULT '' CHECK (
			failure_kind = '' OR failure_kind IN ('operator_forced')
		),
		claimed_by_kind TEXT CHECK (
			claimed_by_kind IS NULL OR claimed_by_kind IN (
				'human', 'agent_session', 'automation', 'extension', 'network_peer', 'daemon'
			)
		),
		claimed_by_ref  TEXT,
		session_id      TEXT,
		origin_kind     TEXT NOT NULL CHECK (
			origin_kind IN (
				'cli', 'web', 'uds', 'http', 'automation', 'extension', 'network', 'agent_session', 'daemon'
			)
		),
		origin_ref      TEXT NOT NULL,
		idempotency_key TEXT,
		network_channel TEXT,
		queued_at       TEXT NOT NULL,
		claimed_at      TEXT,
		started_at      TEXT,
		ended_at        TEXT,
		error           TEXT,
		metadata_json   TEXT,
		result_json     TEXT,
		summary         TEXT NOT NULL DEFAULT '',
		claimed_agent_name TEXT NOT NULL DEFAULT '',
		claimed_peer_id TEXT NOT NULL DEFAULT '',
		terminalized_by_session_id TEXT NOT NULL DEFAULT '',
		terminalized_by_agent_name TEXT NOT NULL DEFAULT '',
		terminalized_by_peer_id TEXT NOT NULL DEFAULT '',
		terminalized_by_actor_kind TEXT NOT NULL DEFAULT '',
		terminalized_by_actor_ref TEXT NOT NULL DEFAULT '',
		review_required BOOLEAN NOT NULL DEFAULT 0 CHECK (review_required IN (0, 1)),
		review_request_round INTEGER NOT NULL DEFAULT 0 CHECK (review_request_round >= 0),
		review_policy_snapshot TEXT NOT NULL DEFAULT '' CHECK (
			review_policy_snapshot = '' OR
			review_policy_snapshot IN ('none', 'on_success', 'on_failure', 'always')
		),
		review_request_id TEXT REFERENCES task_run_reviews(review_id),
		parent_run_id TEXT REFERENCES task_runs(id),
		review_id TEXT REFERENCES task_run_reviews(review_id),
		review_round INTEGER NOT NULL DEFAULT 0 CHECK (review_round >= 0),
		continuation_reason TEXT NOT NULL DEFAULT '',
		missing_work_json TEXT NOT NULL DEFAULT '[]',
		next_round_guidance TEXT NOT NULL DEFAULT '',
		claim_token TEXT,
		claim_token_hash TEXT,
		lease_until TEXT,
		heartbeat_at TEXT,
		coordination_channel_id TEXT,
		CHECK (
			(claimed_by_kind IS NULL AND claimed_by_ref IS NULL) OR
			(claimed_by_kind IS NOT NULL AND claimed_by_ref IS NOT NULL)
		),
		CHECK (status <> 'queued' OR session_id IS NULL)
	);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_task ON task_runs(task_id, queued_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_task_status ON task_runs(task_id, status, queued_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_status ON task_runs(status);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_previous ON task_runs(previous_run_id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_session ON task_runs(session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_channel ON task_runs(network_channel);`,
		`CREATE TABLE IF NOT EXISTS task_dependencies (
		task_id             TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		depends_on_task_id  TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		kind                TEXT NOT NULL CHECK (kind IN ('blocks')),
		created_at          TEXT NOT NULL,
		PRIMARY KEY (task_id, depends_on_task_id, kind),
		CHECK (task_id <> depends_on_task_id)
	);`,
		`CREATE INDEX IF NOT EXISTS idx_task_dependencies_task ON task_dependencies(task_id, created_at ASC, depends_on_task_id ASC);`,
		`CREATE INDEX IF NOT EXISTS idx_task_dependencies_depends_on ON task_dependencies(depends_on_task_id, task_id ASC);`,
		`CREATE TABLE IF NOT EXISTS task_events (
		id          TEXT PRIMARY KEY,
		event_seq   INTEGER NOT NULL,
		task_id     TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		run_id      TEXT REFERENCES task_runs(id) ON DELETE SET NULL,
		event_type  TEXT NOT NULL,
		actor_kind  TEXT NOT NULL CHECK (
			actor_kind IN (
				'human', 'agent_session', 'automation', 'extension', 'network_peer', 'daemon'
			)
		),
		actor_id    TEXT NOT NULL,
		origin_kind TEXT NOT NULL CHECK (
			origin_kind IN (
				'cli', 'web', 'uds', 'http', 'automation', 'extension', 'network', 'agent_session', 'daemon'
			)
		),
		origin_ref  TEXT NOT NULL,
		payload_json TEXT,
		timestamp   TEXT NOT NULL
	);`,
		taskEventIndexStatements[0],
		taskEventIndexStatements[1],
		taskEventIndexStatements[2],
		taskEventIndexStatements[3],
		taskEventIndexStatements[4],
		`CREATE TABLE IF NOT EXISTS task_run_idempotency (
		idempotency_key TEXT NOT NULL,
		origin_kind     TEXT NOT NULL CHECK (
			origin_kind IN (
				'cli', 'web', 'uds', 'http', 'automation', 'extension', 'network', 'agent_session', 'daemon'
			)
		),
		origin_ref      TEXT NOT NULL,
		run_id          TEXT NOT NULL REFERENCES task_runs(id) ON DELETE CASCADE,
		created_at      TEXT NOT NULL,
		PRIMARY KEY (idempotency_key, origin_kind, origin_ref)
	);`,
		`CREATE INDEX IF NOT EXISTS idx_task_run_idempotency_run ON task_run_idempotency(run_id);`,
	},
	taskRunClaimLeaseAuxiliarySchemaStatements(),
	taskOrchestrationProfileSchemaStatements(),
	taskRunReviewTableSchemaStatements(),
	taskReviewGateIndexStatements(),
	notificationCursorSchemaStatements(),
	notificationPresetSchemaStatements(),
	[]string{
		`CREATE TABLE IF NOT EXISTS task_triage_state (
		task_id               TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		actor_kind            TEXT NOT NULL CHECK (
			actor_kind IN (
				'human', 'agent_session', 'automation', 'extension', 'network_peer', 'daemon'
			)
		),
		actor_id              TEXT NOT NULL,
		is_read               BOOLEAN NOT NULL DEFAULT 0,
		archived              BOOLEAN NOT NULL DEFAULT 0,
		dismissed             BOOLEAN NOT NULL DEFAULT 0,
		last_seen_activity_at TEXT,
		updated_at            TEXT NOT NULL,
		PRIMARY KEY (task_id, actor_kind, actor_id)
	);`,
		`CREATE INDEX IF NOT EXISTS idx_task_triage_task ON task_triage_state(task_id, updated_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_task_triage_actor ON task_triage_state(actor_kind, actor_id, updated_at DESC, task_id);`,
		`CREATE TABLE IF NOT EXISTS bridge_instances (
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
	);`,
		`CREATE INDEX IF NOT EXISTS idx_bridge_instances_scope ON bridge_instances(scope, workspace_id, id);`,
		`CREATE TABLE IF NOT EXISTS bridge_secret_bindings (
		bridge_instance_id TEXT NOT NULL REFERENCES bridge_instances(id) ON DELETE CASCADE,
		binding_name        TEXT NOT NULL,
		secret_ref           TEXT NOT NULL,
		kind                TEXT NOT NULL,
		created_at          TEXT NOT NULL,
		updated_at          TEXT NOT NULL,
		PRIMARY KEY (bridge_instance_id, binding_name)
	);`,
		`CREATE INDEX IF NOT EXISTS idx_bridge_secret_bindings_instance ON bridge_secret_bindings(bridge_instance_id);`,
		`CREATE TABLE IF NOT EXISTS bridge_routes (
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
	);`,
		`CREATE INDEX IF NOT EXISTS idx_bridge_routes_instance ON bridge_routes(bridge_instance_id, updated_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_bridge_routes_session ON bridge_routes(session_id);`,
		`CREATE TABLE IF NOT EXISTS bridge_ingest_dedup (
		idempotency_key    TEXT PRIMARY KEY,
		bridge_instance_id TEXT NOT NULL REFERENCES bridge_instances(id) ON DELETE CASCADE,
		received_at        TEXT NOT NULL,
		expires_at         TEXT NOT NULL
	);`,
		`CREATE INDEX IF NOT EXISTS idx_bridge_ingest_dedup_expires ON bridge_ingest_dedup(expires_at);`,
	},
	bridgeTaskSubscriptionSchemaStatements(),
	resources.SchemaStatements(),
)

func appendSchemaStatements(groups ...[]string) []string {
	var statements []string
	for _, group := range groups {
		statements = append(statements, group...)
	}
	return statements
}

func migrateTaskRunClaimLeaseSchema(ctx context.Context, tx *sql.Tx) error {
	if err := addMissingMigrationColumns(ctx, tx, "task_runs", []migrationColumnSpec{
		{name: globalDBClaimTokenKey, sql: `ALTER TABLE task_runs ADD COLUMN claim_token TEXT`},
		{name: globalDBClaimTokenHashKey, sql: `ALTER TABLE task_runs ADD COLUMN claim_token_hash TEXT`},
		{name: globalDBLeaseUntilKey, sql: `ALTER TABLE task_runs ADD COLUMN lease_until TEXT`},
		{name: "heartbeat_at", sql: `ALTER TABLE task_runs ADD COLUMN heartbeat_at TEXT`},
		{
			name: "coordination_channel_id",
			sql:  `ALTER TABLE task_runs ADD COLUMN coordination_channel_id TEXT`,
		},
	}); err != nil {
		return err
	}
	for _, statement := range taskRunClaimLeaseAuxiliarySchemaStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply task run claim lease schema: %w", err)
		}
	}
	return nil
}

func taskRunClaimLeaseAuxiliarySchemaStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS task_run_required_capabilities (
			run_id        TEXT NOT NULL REFERENCES task_runs(id) ON DELETE CASCADE,
			capability_id TEXT NOT NULL,
			PRIMARY KEY (run_id, capability_id)
		);`,
		`CREATE TABLE IF NOT EXISTS task_run_preferred_capabilities (
			run_id        TEXT NOT NULL REFERENCES task_runs(id) ON DELETE CASCADE,
			capability_id TEXT NOT NULL,
			PRIMARY KEY (run_id, capability_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_pending_claim
			ON task_runs(status, lease_until, queued_at, id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_active_lease_recovery
			ON task_runs(status, lease_until, heartbeat_at, id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_coordination_channel
			ON task_runs(coordination_channel_id, queued_at DESC, id DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_session_status
			ON task_runs(session_id, status, lease_until);`,
		`CREATE INDEX IF NOT EXISTS idx_task_run_required_capabilities_capability
			ON task_run_required_capabilities(capability_id, run_id);`,
		`CREATE INDEX IF NOT EXISTS idx_task_run_preferred_capabilities_capability
			ON task_run_preferred_capabilities(capability_id, run_id);`,
	}
}

var globalSchemaMigrations = []store.Migration{
	{
		Version:    1,
		Name:       "create_global_schema",
		Statements: globalSchemaStatements,
		Checksum:   "70e2c16c9d32e692891ab71d075ca823782626e7c9f6ffbbc88c5d662704e089",
	},
	{
		Version:  2,
		Name:     "add_session_failure_diagnostics",
		Up:       migrateSessionFailureColumns,
		Checksum: "2026-04-24-add-session-failure-diagnostics",
	},
	{
		Version:  3,
		Name:     "add_automation_scheduler_state",
		Up:       migrateAutomationSchedulerState,
		Checksum: "2026-04-24-add-automation-scheduler-state",
	},
	{
		Version:  4,
		Name:     "add_mcp_auth_tokens",
		Up:       migrateMCPAuthTokens,
		Checksum: "2026-04-25-add-mcp-auth-tokens",
	},
	{
		Version:  5,
		Name:     "add_tool_process_records",
		Up:       migrateToolProcessRecords,
		Checksum: "2026-04-24-add-tool-process-records",
	},
	{
		Version:  6,
		Name:     "add_memory_operation_scope",
		Up:       migrateMemoryOperationScopeColumns,
		Checksum: "2026-04-25-add-memory-operation-scope",
	},
	{
		Version:  7,
		Name:     "add_task_run_claim_lease_schema",
		Up:       migrateTaskRunClaimLeaseSchema,
		Checksum: "2026-04-26-add-task-run-claim-lease-schema",
	},
	{
		Version:  8,
		Name:     "add_session_lineage_metadata",
		Up:       migrateSessionLineageColumns,
		Checksum: "2026-04-26-add-session-lineage-metadata",
	},
	{
		Version:  9,
		Name:     "rename_environment_columns_to_sandbox",
		Up:       migrateSandboxColumnNames,
		Checksum: "2026-04-28-rename-environment-columns-to-sandbox",
	},
	{
		Version:  10,
		Name:     "add_vault_secrets",
		Up:       migrateVaultSecrets,
		Checksum: "2026-05-01-add-vault-secrets",
	},
	{
		Version:  11,
		Name:     "unify_secret_refs",
		Up:       migrateUnifiedSecretRefs,
		Checksum: "2026-05-01-unify-secret-refs",
	},
	{
		Version:  12,
		Name:     globalDBAddAgentSoulSnapshotsKey,
		Up:       migrateAgentSoulSnapshots,
		Checksum: "2026-05-02-add-agent-soul-snapshots",
	},
	{
		Version:  13,
		Name:     globalDBAddAgentHeartbeatStorageKey,
		Up:       migrateAgentHeartbeatStorage,
		Checksum: "2026-05-02-add-agent-heartbeat-storage",
	},
	{
		Version:  14,
		Name:     "add_event_summary_lineage",
		Up:       migrateEventSummaryLineageColumns,
		Checksum: "2026-05-04-add-event-summary-lineage",
	},
	{
		Version:  15,
		Name:     "rebuild_event_summaries_for_global_payloads",
		Up:       migrateEventSummaryGlobalPayloads,
		Checksum: "2026-05-04-rebuild-event-summaries-for-global-payloads",
	},
	{
		Version:  16,
		Name:     "rename_actor_ref_columns_to_actor_id",
		Up:       migrateActorIDColumns,
		Checksum: "2026-05-04-rename-actor-ref-columns-to-actor-id",
	},
	{
		Version:  17,
		Name:     "add_task_orchestration_profile_schema",
		Up:       migrateTaskOrchestrationProfileSchema,
		Checksum: "2026-05-05-add-task-orchestration-profile-schema",
	},
	{
		Version:  18,
		Name:     "add_task_review_gate_schema",
		Up:       migrateTaskReviewGateSchema,
		Checksum: "2026-05-05-add-task-review-gate-schema",
	},
	{
		Version:  19,
		Name:     "add_notification_cursors",
		Up:       migrateNotificationCursors,
		Checksum: "2026-05-05-add-notification-cursors",
	},
	{
		Version:  20,
		Name:     "add_bridge_task_subscriptions",
		Up:       migrateBridgeTaskSubscriptions,
		Checksum: "2026-05-05-add-bridge-task-subscriptions",
	},
	{
		Version:  21,
		Name:     "rebuild_network_conversation_containers",
		Up:       migrateNetworkConversationContainers,
		Checksum: "2026-05-05-rebuild-network-conversation-containers",
	},
	{
		Version:  22,
		Name:     "memv2_memory_events",
		Up:       migrateMemoryV2Events,
		Checksum: "2026-05-05-memv2-memory-events",
	},
	{
		Version:  23,
		Name:     "add_model_catalog_persistence",
		Up:       migrateModelCatalogPersistence,
		Checksum: "2026-05-07-add-model-catalog-persistence",
	},
	{
		Version:  24,
		Name:     globalDBRebuildModelCatalogSourceConstraintsKey,
		Up:       migrateModelCatalogSourceConstraints,
		Checksum: modelCatalogSourceConstraintChecksum,
	},
	{
		Version:  25,
		Name:     "workspace_qualified_network_identity",
		Up:       migrateWorkspaceQualifiedNetworkIdentity,
		Checksum: "2026-05-12-workspace-qualified-network-identity",
	},
	{
		Version:  26,
		Name:     "add_network_timeline_extensions",
		Up:       migrateNetworkTimelineExtensions,
		Checksum: "2026-05-16-add-network-timeline-extensions",
	},
	{
		Version:  27,
		Name:     "add_config_apply_records",
		Up:       migrateConfigApplyRecords,
		Checksum: "2026-05-20-add-config-apply-records",
	},
	{
		Version:  28,
		Name:     "add_event_summary_projections",
		Up:       migrateEventSummaryProjections,
		Checksum: "2026-05-20-add-event-summary-projections",
	},
	{
		Version:  29,
		Name:     "add_scheduler_pause_state",
		Up:       migrateSchedulerPauseState,
		Checksum: "2026-05-20-add-scheduler-pause-state",
	},
	{
		Version:  30,
		Name:     "add_session_attach_lock",
		Up:       migrateSessionAttachLock,
		Checksum: "2026-05-20-add-session-attach-lock",
	},
	{
		Version:  31,
		Name:     "add_session_input_queue",
		Up:       migrateSessionInputQueue,
		Checksum: "2026-05-21-add-session-input-queue",
	},
	{
		Version:  32,
		Name:     "add_task_run_force_ops",
		Up:       migrateTaskRunForceOps,
		Checksum: "2026-05-21-add-task-run-force-ops",
	},
	{
		Version:  33,
		Name:     "add_task_pause_state",
		Up:       migratePauseState,
		Checksum: "2026-05-21-add-task-pause-state",
	},
	{
		Version:  34,
		Name:     "add_extension_provenance",
		Up:       migrateExtensionProvenance,
		Checksum: "2026-05-21-add-extension-provenance",
	},
	{
		Version:  35,
		Name:     "add_bridge_target_directory",
		Up:       migrateBridgeTargetDirectory,
		Checksum: "2026-05-21-add-bridge-target-directory",
	},
	{
		Version:  36,
		Name:     "add_notification_presets",
		Up:       migrateNotificationPresets,
		Checksum: "2026-05-21-add-notification-presets",
	},
	{
		Version:  37,
		Name:     "add_app_metadata",
		Up:       migrateAppMetadata,
		Checksum: "2026-05-25-add-app-metadata",
	},
}

func migrateSessionInputQueue(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "sessions")
	if err != nil {
		return err
	}
	if exists {
		if err := addMissingMigrationColumns(ctx, tx, "sessions", []migrationColumnSpec{
			{
				name: sessionInputGenerationColumn,
				sql:  `ALTER TABLE sessions ADD COLUMN input_generation INTEGER NOT NULL DEFAULT 0`,
			},
		}); err != nil {
			return err
		}
	}
	statements := []string{
		`CREATE TABLE IF NOT EXISTS session_input_queue (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
			status TEXT NOT NULL CHECK (status IN ('queued', 'dispatching', 'sent', 'failed', 'canceled')),
			mode TEXT NOT NULL CHECK (mode IN ('queue', 'steer')),
			text TEXT NOT NULL,
			session_generation INTEGER NOT NULL DEFAULT 0,
			task_run_id TEXT NOT NULL DEFAULT '',
			run_generation INTEGER,
			attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
			enqueued_at TEXT NOT NULL,
			dispatch_started_at TEXT,
			sent_at TEXT,
			failed_at TEXT,
			failure_summary TEXT NOT NULL DEFAULT '',
			canceled_at TEXT,
			updated_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_session_input_queue_pending
			ON session_input_queue(session_id, status, enqueued_at, id);`,
		`CREATE INDEX IF NOT EXISTS idx_session_input_queue_generation
			ON session_input_queue(session_id, session_generation, status);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_session_input_queue_active_steer
			ON session_input_queue(session_id)
			WHERE mode = 'steer' AND status IN ('queued', 'dispatching');`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: create session input queue schema: %w", err)
		}
	}
	return nil
}

func migrateSessionAttachLock(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "sessions")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := addMissingMigrationColumns(ctx, tx, "sessions", []migrationColumnSpec{
		{
			name: globalDBSessionAttachedToColumn,
			sql:  `ALTER TABLE sessions ADD COLUMN attached_to TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: globalDBAttachExpiresAtColumn,
			sql:  `ALTER TABLE sessions ADD COLUMN attach_expires_at TEXT`,
		},
	}); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_sessions_attach_lock
			ON sessions(attached_to, attach_expires_at);
		`); err != nil {
		return fmt.Errorf("store: create session attach lock index: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_sessions_resumable
			ON sessions(state, failure_kind, last_update_at, updated_at);
		`); err != nil {
		return fmt.Errorf("store: create session resumable index: %w", err)
	}
	return nil
}

func migrateNetworkTimelineExtensions(ctx context.Context, tx *sql.Tx) error {
	return addMissingMigrationColumns(ctx, tx, "network_timeline_log", []migrationColumnSpec{
		{
			name: "ext_json",
			sql:  `ALTER TABLE network_timeline_log ADD COLUMN ext_json TEXT NOT NULL DEFAULT '{}'`,
		},
	})
}

func migrateWorkspaceQualifiedNetworkIdentity(ctx context.Context, tx *sql.Tx) error {
	if err := snapshotNetworkChannelsForWorkspaceIdentity(ctx, tx); err != nil {
		return err
	}
	for _, statement := range workspaceQualifiedNetworkIdentityStatements() {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: apply workspace-qualified network identity migration: %w", err)
		}
	}
	return nil
}

func snapshotNetworkChannelsForWorkspaceIdentity(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS temp.network_channels_v25_keep;`); err != nil {
		return fmt.Errorf("store: prepare workspace-qualified network channel snapshot: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TEMP TABLE network_channels_v25_keep (
		workspace_id TEXT NOT NULL,
		channel      TEXT NOT NULL,
		purpose      TEXT NOT NULL,
		created_by   TEXT NOT NULL,
		created_at   TEXT NOT NULL,
		updated_at   TEXT NOT NULL
	);`); err != nil {
		return fmt.Errorf("store: create workspace-qualified network channel snapshot: %w", err)
	}
	hasNetworkChannels, err := tableExists(ctx, tx, "network_channels")
	if err != nil {
		return fmt.Errorf("store: inspect network_channels before workspace-qualified rebuild: %w", err)
	}
	if hasNetworkChannels {
		_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO network_channels_v25_keep (
			workspace_id, channel, purpose, created_by, created_at, updated_at
		)
		SELECT
			TRIM(nc.workspace_id),
			TRIM(nc.channel),
			nc.purpose,
			nc.created_by,
			nc.created_at,
			nc.updated_at
		FROM network_channels nc
		INNER JOIN workspaces w ON w.id = TRIM(nc.workspace_id)
		WHERE TRIM(nc.workspace_id) <> '' AND TRIM(nc.channel) <> '';`)
		if err != nil {
			return fmt.Errorf("store: snapshot workspace-qualified network channels: %w", err)
		}
	}
	return nil
}

func workspaceQualifiedNetworkIdentityStatements() []string {
	statements := []string{
		`DROP TABLE IF EXISTS event_summaries;`,
		`CREATE TABLE IF NOT EXISTS event_summaries (
			id                     TEXT PRIMARY KEY,
			session_id             TEXT NOT NULL DEFAULT '',
			workspace_id           TEXT NOT NULL DEFAULT '',
			type                   TEXT NOT NULL,
			agent_name             TEXT NOT NULL DEFAULT '',
			content_json           TEXT NOT NULL DEFAULT '',
			task_id                TEXT NOT NULL DEFAULT '',
			run_id                 TEXT NOT NULL DEFAULT '',
			workflow_id            TEXT NOT NULL DEFAULT '',
			claim_token_hash       TEXT NOT NULL DEFAULT '',
			lease_until            TEXT NOT NULL DEFAULT '',
			coordinator_session_id TEXT NOT NULL DEFAULT '',
			scheduler_reason       TEXT NOT NULL DEFAULT '',
			hook_event             TEXT NOT NULL DEFAULT '',
			hook_name              TEXT NOT NULL DEFAULT '',
			actor_kind             TEXT NOT NULL DEFAULT '',
			actor_id               TEXT NOT NULL DEFAULT '',
			release_reason         TEXT NOT NULL DEFAULT '',
			parent_session_id      TEXT NOT NULL DEFAULT '',
			root_session_id        TEXT NOT NULL DEFAULT '',
			spawn_depth            INTEGER NOT NULL DEFAULT 0,
			summary                TEXT,
			timestamp              TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_summaries_workspace ON event_summaries(workspace_id);`,
		idxSummarySessionSQL,
		idxSummaryTypeSQL,
		idxSummaryTimeSQL,
		idxSummaryTaskSQL,
		idxSummaryRunSQL,
		idxSummaryWorkflowSQL,
		idxSummaryHookEventSQL,
		idxSummaryActorSQL,
		idxSummaryParentSQL,
		idxSummaryRootSQL,
		`DROP TABLE IF EXISTS network_thread_participants;`,
		`DROP TABLE IF EXISTS network_work;`,
		`DROP TABLE IF EXISTS network_direct_rooms;`,
		`DROP TABLE IF EXISTS network_threads;`,
		`DROP TABLE IF EXISTS network_timeline_log;`,
		`DROP TABLE IF EXISTS network_audit_log;`,
		`DROP TABLE IF EXISTS network_channels;`,
		`CREATE TABLE IF NOT EXISTS workspaces (
			id            TEXT PRIMARY KEY,
			root_dir      TEXT NOT NULL UNIQUE,
			add_dirs      TEXT NOT NULL DEFAULT '[]',
			name          TEXT NOT NULL UNIQUE,
			default_agent TEXT DEFAULT '',
			sandbox_ref   TEXT NOT NULL DEFAULT '',
			created_at    TEXT NOT NULL,
			updated_at    TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS network_channels (
			workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			channel      TEXT NOT NULL,
			purpose      TEXT NOT NULL,
			created_by   TEXT NOT NULL DEFAULT '',
			created_at   TEXT NOT NULL,
			updated_at   TEXT NOT NULL,
			PRIMARY KEY (workspace_id, channel)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_network_channels_workspace ON network_channels(workspace_id);`,
		`CREATE INDEX IF NOT EXISTS idx_network_channels_updated_at ON network_channels(updated_at);`,
		`CREATE INDEX IF NOT EXISTS idx_network_channels_workspace_updated_at
			ON network_channels(workspace_id, updated_at DESC, channel ASC);`,
		`INSERT OR IGNORE INTO network_channels (
			workspace_id, channel, purpose, created_by, created_at, updated_at
		)
		SELECT workspace_id, channel, purpose, created_by, created_at, updated_at
		FROM network_channels_v25_keep;`,
		`DROP TABLE IF EXISTS temp.network_channels_v25_keep;`,
	}
	return append(statements, networkConversationSchemaStatements...)
}

func migrateModelCatalogSourceConstraints(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`DROP TABLE IF EXISTS model_catalog_reasoning_efforts;`,
		`DROP TABLE IF EXISTS model_catalog_rows;`,
		`DROP TABLE IF EXISTS model_catalog_sources;`,
		modelCatalogSourcesSchemaStatement(),
		modelCatalogRowsSchemaStatementWithSourceForeignKey(),
		modelCatalogReasoningEffortsSchemaStatement(),
	}
	statements = append(statements, modelCatalogIndexStatements()...)
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: rebuild model catalog source constraints: %w", err)
		}
	}
	return nil
}

func migrateNetworkConversationContainers(ctx context.Context, tx *sql.Tx) error {
	if err := migrateNetworkTimelineLogConversationColumns(ctx, tx); err != nil {
		return err
	}
	if err := migrateNetworkAuditLogConversationColumns(ctx, tx); err != nil {
		return err
	}
	if err := ensureNetworkConversationSchema(ctx, tx); err != nil {
		return err
	}
	return nil
}

func migrateNetworkTimelineLogConversationColumns(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "network_timeline_log")
	if err != nil {
		return err
	}
	if !exists {
		if _, err := tx.ExecContext(ctx, networkTimelineLogTableStatement); err != nil {
			return fmt.Errorf("store: create network_timeline_log: %w", err)
		}
		return nil
	}

	columns, err := tableColumns(ctx, tx, "network_timeline_log")
	if err != nil {
		return err
	}
	if _, hasInteractionID := columns["interaction_id"]; !hasInteractionID {
		if _, hasSurface := columns["surface"]; hasSurface {
			return nil
		}
		return errors.New("store: network_timeline_log schema is stale; recreate the AGH database")
	}

	statements := []string{
		`DROP TABLE IF EXISTS network_timeline_log_new`,
		strings.Replace(networkTimelineLogTableStatement, "network_timeline_log", "network_timeline_log_new", 1),
		`INSERT INTO network_timeline_log_new (
				message_id,
				session_id,
				workspace_id,
				channel,
			surface,
			thread_id,
			direct_id,
			direction,
			peer_from,
			peer_to,
			kind,
			work_id,
			reply_to,
			trace_id,
			causation_id,
			intent,
			text,
			preview_text,
			body_json,
			timestamp
		)
			SELECT
				message_id,
				session_id,
				'legacy_workspace',
				channel,
			NULL,
			NULL,
			NULL,
			direction,
			peer_from,
			peer_to,
			kind,
			NULL,
			reply_to,
			trace_id,
			causation_id,
			intent,
			text,
			preview_text,
			body_json,
			timestamp
		FROM network_timeline_log
		WHERE kind IN ('greet', 'whois')`,
		`DROP TABLE network_timeline_log`,
		`ALTER TABLE network_timeline_log_new RENAME TO network_timeline_log`,
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: rebuild network_timeline_log for conversation containers: %w", err)
		}
	}
	return nil
}

func migrateNetworkAuditLogConversationColumns(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "network_audit_log")
	if err != nil {
		return err
	}
	if !exists {
		if _, err := tx.ExecContext(ctx, networkAuditLogTableStatement); err != nil {
			return fmt.Errorf("store: create network_audit_log: %w", err)
		}
		return nil
	}

	columns, err := tableColumns(ctx, tx, "network_audit_log")
	if err != nil {
		return err
	}
	if _, hasSurface := columns["surface"]; hasSurface {
		if _, hasWorkID := columns["work_id"]; hasWorkID {
			return nil
		}
	}

	statements := []string{
		`DROP TABLE IF EXISTS network_audit_log_new`,
		strings.Replace(networkAuditLogTableStatement, "network_audit_log", "network_audit_log_new", 1),
		`INSERT INTO network_audit_log_new (
				id,
				session_id,
				workspace_id,
				direction,
			kind,
			channel,
			surface,
			thread_id,
			direct_id,
			work_id,
			peer_from,
			peer_to,
			message_id,
			reason,
			size,
			timestamp
		)
			SELECT
				id,
				session_id,
				'legacy_workspace',
				direction,
			kind,
			channel,
			NULL,
			NULL,
			NULL,
			NULL,
			peer_from,
			peer_to,
			message_id,
			reason,
			size,
			timestamp
		FROM network_audit_log`,
		`DROP TABLE network_audit_log`,
		`ALTER TABLE network_audit_log_new RENAME TO network_audit_log`,
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: rebuild network_audit_log for conversation containers: %w", err)
		}
	}
	return nil
}

func ensureNetworkConversationSchema(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range networkConversationSchemaStatements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: ensure network conversation schema: %w", err)
		}
	}
	return nil
}

func migrateActorIDColumns(ctx context.Context, tx *sql.Tx) error {
	specs := []struct {
		table string
		from  string
		to    string
		sql   string
	}{
		{
			table: globalDBTaskEventsKey,
			from:  globalDBActorRefKey,
			to:    globalDBActorIDKey,
			sql:   `ALTER TABLE task_events RENAME COLUMN actor_ref TO actor_id`,
		},
		{
			table: "task_triage_state",
			from:  globalDBActorRefKey,
			to:    globalDBActorIDKey,
			sql:   `ALTER TABLE task_triage_state RENAME COLUMN actor_ref TO actor_id`,
		},
		{
			table: "agent_soul_revisions",
			from:  globalDBActorRefKey,
			to:    globalDBActorIDKey,
			sql:   `ALTER TABLE agent_soul_revisions RENAME COLUMN actor_ref TO actor_id`,
		},
		{
			table: "agent_heartbeat_revisions",
			from:  globalDBActorRefKey,
			to:    globalDBActorIDKey,
			sql:   `ALTER TABLE agent_heartbeat_revisions RENAME COLUMN actor_ref TO actor_id`,
		},
	}
	for _, spec := range specs {
		exists, err := tableExists(ctx, tx, spec.table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}

		columns, err := tableColumns(ctx, tx, spec.table)
		if err != nil {
			return err
		}
		if _, ok := columns[spec.to]; ok {
			continue
		}
		if _, ok := columns[spec.from]; !ok {
			return fmt.Errorf("store: %s schema is stale; recreate the AGH database", spec.table)
		}
		if _, err := tx.ExecContext(ctx, spec.sql); err != nil {
			return fmt.Errorf("store: rename %s.%s to %s: %w", spec.table, spec.from, spec.to, err)
		}
	}
	return nil
}

func migrateUnifiedSecretRefs(ctx context.Context, tx *sql.Tx) error {
	automationColumns, err := tableColumns(ctx, tx, "automation_triggers")
	if err != nil {
		return err
	}
	if _, ok := automationColumns["webhook_secret_ref"]; !ok {
		if _, err := tx.ExecContext(
			ctx,
			`ALTER TABLE automation_triggers ADD COLUMN webhook_secret_ref TEXT`,
		); err != nil {
			return fmt.Errorf("store: add automation_triggers.webhook_secret_ref column: %w", err)
		}
	}

	bridgeColumns, err := tableColumns(ctx, tx, "bridge_secret_bindings")
	if err != nil {
		return err
	}
	if _, hasSecretRef := bridgeColumns["secret_ref"]; !hasSecretRef {
		if _, hasLegacyVaultRef := bridgeColumns["vault_ref"]; hasLegacyVaultRef {
			if _, err := tx.ExecContext(
				ctx,
				`ALTER TABLE bridge_secret_bindings RENAME COLUMN vault_ref TO secret_ref`,
			); err != nil {
				return fmt.Errorf(
					"store: rename bridge_secret_bindings.vault_ref to secret_ref: %w",
					err,
				)
			}
		} else {
			return errors.New("store: bridge_secret_bindings schema is stale; recreate the AGH database")
		}
	}

	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS automation_trigger_webhook_secrets`); err != nil {
		return fmt.Errorf("store: drop automation trigger webhook plaintext secrets: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS mcp_auth_tokens`); err != nil {
		return fmt.Errorf("store: drop legacy MCP auth token store: %w", err)
	}
	if err := migrateMCPAuthTokens(ctx, tx); err != nil {
		return err
	}
	return nil
}

func migrateAppMetadata(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS app_metadata (
			key        TEXT PRIMARY KEY,
			value      TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: migrate app metadata: %w", err)
		}
	}
	return nil
}

func migrateVaultSecrets(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS vault_secrets (
			ref             TEXT PRIMARY KEY,
			kind            TEXT NOT NULL DEFAULT '',
			encrypted_value TEXT NOT NULL,
			created_at      TEXT NOT NULL,
			updated_at      TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_vault_secrets_kind
			ON vault_secrets(kind);`,
		`CREATE INDEX IF NOT EXISTS idx_vault_secrets_updated_at
			ON vault_secrets(updated_at);`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: migrate vault secrets: %w", err)
		}
	}
	return nil
}

func migrateSessionLineageColumns(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "sessions")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	columns, err := tableColumns(ctx, tx, "sessions")
	if err != nil {
		return err
	}
	specs := []struct {
		name string
		sql  string
	}{
		{name: globalDBParentSessionIDKey, sql: `ALTER TABLE sessions ADD COLUMN parent_session_id TEXT`},
		{name: globalDBRootSessionIDKey, sql: `ALTER TABLE sessions ADD COLUMN root_session_id TEXT`},
		{name: globalDBSpawnDepthKey, sql: `ALTER TABLE sessions ADD COLUMN spawn_depth INTEGER NOT NULL DEFAULT 0`},
		{name: globalDBSpawnRoleKey, sql: `ALTER TABLE sessions ADD COLUMN spawn_role TEXT`},
		{name: globalDBTTLExpiresAtKey, sql: `ALTER TABLE sessions ADD COLUMN ttl_expires_at TEXT`},
		{
			name: globalDBAutoStopOnParentKey,
			sql:  `ALTER TABLE sessions ADD COLUMN auto_stop_on_parent BOOLEAN NOT NULL DEFAULT 0`,
		},
		{
			name: globalDBSpawnBudgetJSONKey,
			sql:  `ALTER TABLE sessions ADD COLUMN spawn_budget_json TEXT NOT NULL DEFAULT '{}'`,
		},
		{
			name: globalDBPermissionPolicyJSONKey,
			sql:  `ALTER TABLE sessions ADD COLUMN permission_policy_json TEXT NOT NULL DEFAULT '{}'`,
		},
	}
	for _, spec := range specs {
		if _, ok := columns[spec.name]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, spec.sql); err != nil {
			return fmt.Errorf("store: add sessions.%s column: %w", spec.name, err)
		}
	}
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_sessions_parent ON sessions(parent_session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_root ON sessions(root_session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_type_depth ON sessions(session_type, spawn_depth);`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_spawn_role ON sessions(spawn_role);`,
	}
	for _, stmt := range indexes {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate session lineage indexes: %w", err)
		}
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE sessions SET root_session_id = id WHERE root_session_id IS NULL OR trim(root_session_id) = ''`,
	); err != nil {
		return fmt.Errorf("store: backfill root session lineage: %w", err)
	}
	return nil
}

func migrateEventSummaryLineageColumns(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "event_summaries")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	columns, err := tableColumns(ctx, tx, "event_summaries")
	if err != nil {
		return err
	}
	specs := []struct {
		name string
		sql  string
	}{
		{
			name: globalDBParentSessionIDKey,
			sql:  `ALTER TABLE event_summaries ADD COLUMN parent_session_id TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: globalDBRootSessionIDKey,
			sql:  `ALTER TABLE event_summaries ADD COLUMN root_session_id TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: globalDBSpawnDepthKey,
			sql:  `ALTER TABLE event_summaries ADD COLUMN spawn_depth INTEGER NOT NULL DEFAULT 0`,
		},
	}
	for _, spec := range specs {
		if _, ok := columns[spec.name]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, spec.sql); err != nil {
			return fmt.Errorf("store: add event_summaries.%s column: %w", spec.name, err)
		}
	}

	indexes := []string{
		idxSummaryParentSQL,
		idxSummaryRootSQL,
	}
	for _, stmt := range indexes {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate event summary lineage indexes: %w", err)
		}
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE event_summaries
		 SET parent_session_id = COALESCE(
		 	(SELECT parent_session_id FROM sessions WHERE sessions.id = event_summaries.session_id),
		 	''
		 ),
		     root_session_id = COALESCE(
		     	NULLIF((SELECT root_session_id FROM sessions WHERE sessions.id = event_summaries.session_id), ''),
		     	session_id
		     ),
		     spawn_depth = COALESCE(
		     	(SELECT spawn_depth FROM sessions WHERE sessions.id = event_summaries.session_id),
		     	0
		     )
		 WHERE trim(session_id) <> ''`,
	); err != nil {
		return fmt.Errorf("store: backfill event summary lineage: %w", err)
	}
	return nil
}

func migrateEventSummaryGlobalPayloads(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "event_summaries")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	columns, err := tableColumns(ctx, tx, "event_summaries")
	if err != nil {
		return err
	}

	if err := createRebuiltEventSummariesTable(ctx, tx); err != nil {
		return err
	}
	if err := copyRebuiltEventSummaries(ctx, tx, columns); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DROP TABLE event_summaries`); err != nil {
		return fmt.Errorf("store: drop legacy event_summaries: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE event_summaries_new RENAME TO event_summaries`); err != nil {
		return fmt.Errorf("store: rename rebuilt event_summaries: %w", err)
	}

	return rebuildEventSummaryIndexes(ctx, tx)
}

func createRebuiltEventSummariesTable(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `DROP TABLE IF EXISTS event_summaries_new`); err != nil {
		return fmt.Errorf("store: drop stale event_summaries_new: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE TABLE event_summaries_new (
			id                     TEXT PRIMARY KEY,
			session_id             TEXT NOT NULL DEFAULT '',
			type                   TEXT NOT NULL,
			agent_name             TEXT NOT NULL DEFAULT '',
			content_json           TEXT NOT NULL DEFAULT '',
			task_id                TEXT NOT NULL DEFAULT '',
			run_id                 TEXT NOT NULL DEFAULT '',
			workflow_id            TEXT NOT NULL DEFAULT '',
			claim_token_hash       TEXT NOT NULL DEFAULT '',
			lease_until            TEXT NOT NULL DEFAULT '',
			coordinator_session_id TEXT NOT NULL DEFAULT '',
			scheduler_reason       TEXT NOT NULL DEFAULT '',
			hook_event             TEXT NOT NULL DEFAULT '',
			hook_name              TEXT NOT NULL DEFAULT '',
			actor_kind             TEXT NOT NULL DEFAULT '',
			actor_id               TEXT NOT NULL DEFAULT '',
			release_reason         TEXT NOT NULL DEFAULT '',
			parent_session_id      TEXT NOT NULL DEFAULT '',
			root_session_id        TEXT NOT NULL DEFAULT '',
			spawn_depth            INTEGER NOT NULL DEFAULT 0,
			summary                TEXT,
			timestamp              TEXT NOT NULL
		);`,
	); err != nil {
		return fmt.Errorf("store: create rebuilt event_summaries table: %w", err)
	}
	return nil
}

func copyRebuiltEventSummaries(
	ctx context.Context,
	tx *sql.Tx,
	columns map[string]struct{},
) error {
	selectList := []string{
		eventSummaryColumnExpr(columns, "id", `''`),
		eventSummaryColumnExpr(columns, "session_id", `''`),
		eventSummaryColumnExpr(columns, "type", `''`),
		eventSummaryColumnExpr(columns, "agent_name", `''`),
		eventSummaryColumnExpr(columns, "content_json", `''`),
		eventSummaryColumnExpr(columns, "task_id", `''`),
		eventSummaryColumnExpr(columns, "run_id", `''`),
		eventSummaryColumnExpr(columns, "workflow_id", `''`),
		eventSummaryColumnExpr(columns, globalDBClaimTokenHashKey, `''`),
		eventSummaryColumnExpr(columns, globalDBLeaseUntilKey, `''`),
		eventSummaryColumnExpr(columns, "coordinator_session_id", `''`),
		eventSummaryColumnExpr(columns, "scheduler_reason", `''`),
		eventSummaryColumnExpr(columns, "hook_event", `''`),
		eventSummaryColumnExpr(columns, "hook_name", `''`),
		eventSummaryColumnExpr(columns, "actor_kind", `''`),
		eventSummaryColumnExpr(columns, globalDBActorIDKey, `''`),
		eventSummaryColumnExpr(columns, "release_reason", `''`),
		eventSummaryColumnExpr(columns, globalDBParentSessionIDKey, `''`),
		eventSummaryColumnExpr(columns, globalDBRootSessionIDKey, `''`),
		eventSummaryColumnExpr(columns, globalDBSpawnDepthKey, `0`),
		eventSummaryColumnExpr(columns, globalDBSummaryKey, `NULL`),
		eventSummaryColumnExpr(columns, "timestamp", `''`),
	}
	if _, err := tx.ExecContext(ctx, buildEventSummaryCopyQuery(selectList)); err != nil {
		return fmt.Errorf("store: copy rebuilt event summaries: %w", err)
	}
	return nil
}

func buildEventSummaryCopyQuery(selectList []string) string {
	var builder strings.Builder
	builder.WriteString(`INSERT INTO event_summaries_new (
		id, session_id, type, agent_name, content_json, task_id, run_id, workflow_id,
		claim_token_hash, lease_until, coordinator_session_id, scheduler_reason, hook_event,
		hook_name, actor_kind, actor_id, release_reason, parent_session_id, root_session_id,
		spawn_depth, summary, timestamp
	) SELECT `)
	builder.WriteString(strings.Join(selectList, ", "))
	builder.WriteString(` FROM event_summaries`)
	return builder.String()
}

func rebuildEventSummaryIndexes(ctx context.Context, tx *sql.Tx) error {
	indexes := []string{
		idxSummarySessionSQL,
		idxSummaryTypeSQL,
		idxSummaryTimeSQL,
		idxSummaryTaskSQL,
		idxSummaryRunSQL,
		idxSummaryWorkflowSQL,
		idxSummaryHookEventSQL,
		idxSummaryActorSQL,
		idxSummaryParentSQL,
		idxSummaryRootSQL,
	}
	for _, stmt := range indexes {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: rebuild event_summaries indexes: %w", err)
		}
	}
	return nil
}

func eventSummaryColumnExpr(columns map[string]struct{}, name string, fallback string) string {
	if _, ok := columns[name]; ok {
		return name
	}
	return fallback + ` AS ` + name
}

func migrateMemoryOperationScopeColumns(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "memory_operation_log")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	columns, err := tableColumns(ctx, tx, "memory_operation_log")
	if err != nil {
		return err
	}
	specs := []struct {
		name string
		sql  string
	}{
		{name: globalDBScopeKey, sql: `ALTER TABLE memory_operation_log ADD COLUMN scope TEXT NOT NULL DEFAULT ''`},
		{
			name: "workspace_root",
			sql:  `ALTER TABLE memory_operation_log ADD COLUMN workspace_root TEXT NOT NULL DEFAULT ''`,
		},
		{name: "filename", sql: `ALTER TABLE memory_operation_log ADD COLUMN filename TEXT NOT NULL DEFAULT ''`},
	}
	for _, spec := range specs {
		if _, ok := columns[spec.name]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, spec.sql); err != nil {
			return fmt.Errorf("store: add memory_operation_log.%s column: %w", spec.name, err)
		}
	}
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_memory_operation_log_scope ON memory_operation_log(scope);`,
		`CREATE INDEX IF NOT EXISTS idx_memory_operation_log_workspace_root ON memory_operation_log(workspace_root);`,
	}
	for _, stmt := range indexes {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate memory operation scope indexes: %w", err)
		}
	}
	return nil
}

var globalMemoryV2EventStatements = []string{
	`CREATE TABLE IF NOT EXISTS memory_events (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		op           TEXT NOT NULL CHECK (op IN (
			'` + globalMemoryEventWriteCommitted + `',
			'memory.write.rejected',
			'memory.write.shadowed',
			'memory.write.reindex',
			'memory.write.reverted',
			'memory.recall.executed',
			'memory.recall.skipped',
			'memory.recall.signal_dropped',
			'memory.recall.signal_update_failed',
			'memory.decisions.audit_summarized',
			'memory.decisions.pruned',
			'memory.dream.run.started',
			'memory.dream.run.promoted',
			'memory.dream.run.failed',
			'memory.extractor.started',
			'memory.extractor.completed',
			'memory.extractor.failed',
			'memory.extractor.coalesced',
			'memory.extractor.dropped',
			'memory.daily.rotated',
			'memory.daily.archived',
			'memory.daily.restored',
			'memory.daily.purged',
			'memory.daily.archive_purged',
			'memory.provider.enabled',
			'memory.provider.disabled',
			'memory.provider.collision',
			'memory.workspace.relocated',
			'memory.workspace.recovered',
			'memory.agent.purged',
			'memory.migration.applied'
		)),
		scope        TEXT CHECK (scope IN ('global', 'workspace', 'agent')),
		agent_name   TEXT,
		agent_tier   TEXT CHECK (agent_tier IS NULL OR agent_tier IN ('workspace', 'global')),
		workspace_id TEXT,
		session_id   TEXT,
		actor_kind   TEXT NOT NULL,
		decision_id  TEXT,
		target_id    TEXT,
		metadata     TEXT NOT NULL DEFAULT '{}',
		ts_ms        INTEGER NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_events_workspace ON memory_events(workspace_id, ts_ms);`,
	`CREATE INDEX IF NOT EXISTS idx_events_op ON memory_events(op, ts_ms);`,
	`CREATE INDEX IF NOT EXISTS idx_events_session ON memory_events(session_id, ts_ms);`,
}

func migrateMemoryV2Events(ctx context.Context, tx *sql.Tx) error {
	if err := ensureMemoryV2EventsSchema(ctx, tx); err != nil {
		return err
	}
	return migrateLegacyMemoryOperationLog(ctx, tx)
}

func ensureMemoryV2EventsSchema(ctx context.Context, tx *sql.Tx) error {
	for _, stmt := range globalMemoryV2EventStatements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate memory events schema: %w", err)
		}
	}
	return nil
}

func migrateLegacyMemoryOperationLog(ctx context.Context, tx *sql.Tx) error {
	exists, err := tableExists(ctx, tx, "memory_operation_log")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	columns, err := tableColumns(ctx, tx, "memory_operation_log")
	if err != nil {
		return err
	}
	rows, err := tx.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT id, type, %s, %s, %s, agent_name, summary, timestamp
			 FROM memory_operation_log ORDER BY timestamp ASC, id ASC`,
			eventSummaryColumnExpr(columns, globalDBScopeKey, "''"),
			eventSummaryColumnExpr(columns, "workspace_root", "''"),
			eventSummaryColumnExpr(columns, "filename", "''"),
		),
	)
	if err != nil {
		return fmt.Errorf("store: read legacy memory operation log: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()
	for rows.Next() {
		if err := migrateMemoryOperationRow(ctx, tx, rows); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("store: iterate legacy memory operation log: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE memory_operation_log`); err != nil {
		return fmt.Errorf("store: drop legacy memory_operation_log: %w", err)
	}
	return nil
}

func migrateMemoryOperationRow(ctx context.Context, tx *sql.Tx, rows *sql.Rows) error {
	var (
		id           string
		op           string
		scope        string
		workspace    string
		filename     string
		agentName    string
		summary      string
		timestampRaw string
	)
	if err := rows.Scan(&id, &op, &scope, &workspace, &filename, &agentName, &summary, &timestampRaw); err != nil {
		return fmt.Errorf("store: scan legacy memory operation row: %w", err)
	}
	timestamp, err := store.ParseTimestamp(timestampRaw)
	if err != nil {
		return err
	}
	workspaceID := strings.TrimSpace(workspace)
	if strings.TrimSpace(scope) == globalDBWorkspaceKey &&
		workspaceID != "" &&
		!aghworkspace.IsWorkspaceID(workspaceID) {
		identity, err := aghworkspace.EnsureIdentity(ctx, workspaceID)
		if err != nil {
			return fmt.Errorf("store: resolve legacy memory operation workspace identity %q: %w", workspaceID, err)
		}
		workspaceID = identity.WorkspaceID
	}
	metadata, err := json.Marshal(map[string]string{
		"legacy_id":        id,
		"action":           strings.TrimSpace(op),
		"filename":         strings.TrimSpace(filename),
		globalDBSummaryKey: strings.TrimSpace(summary),
	})
	if err != nil {
		return fmt.Errorf("store: encode legacy memory event metadata: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO memory_events (
			op, scope, agent_name, agent_tier, workspace_id, session_id, actor_kind,
			decision_id, target_id, metadata, ts_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		canonicalMemoryEventOp(op),
		nullableString(scope),
		nullableString(agentName),
		nil,
		nullableString(workspaceID),
		nil,
		"system",
		nil,
		nullableString(filename),
		string(metadata),
		timestamp.UTC().UnixNano()/int64(time.Millisecond),
	); err != nil {
		return fmt.Errorf("store: migrate legacy memory event: %w", err)
	}
	return nil
}

func canonicalMemoryEventOp(op string) string {
	switch strings.TrimSpace(op) {
	case "memory.search":
		return "memory.recall.executed"
	case "memory.reindex":
		return "memory.write.reindex"
	default:
		return globalMemoryEventWriteCommitted
	}
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func migrateMCPAuthTokens(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS mcp_auth_tokens (
			server_name    TEXT PRIMARY KEY,
			issuer         TEXT NOT NULL DEFAULT '',
			client_id      TEXT NOT NULL,
			scopes_json    TEXT NOT NULL DEFAULT '[]',
			access_token_ref   TEXT NOT NULL,
			refresh_token_ref  TEXT NOT NULL DEFAULT '',
			token_type     TEXT NOT NULL DEFAULT 'Bearer',
			expires_at     TEXT,
			obtained_at    TEXT NOT NULL,
			updated_at     TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_mcp_auth_tokens_updated_at
			ON mcp_auth_tokens(updated_at);`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: migrate MCP auth tokens: %w", err)
		}
	}
	return nil
}

func migrateAutomationSchedulerState(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS automation_scheduler_state (
			job_id                       TEXT PRIMARY KEY,
			next_run_at                  TEXT,
			last_run_at                  TEXT,
			last_scheduled_at            TEXT,
			last_fire_id                 TEXT NOT NULL DEFAULT '',
			schedule_hash                TEXT NOT NULL DEFAULT '',
			catch_up_policy              TEXT NOT NULL DEFAULT 'skip_missed'
				CHECK (catch_up_policy IN ('skip_missed')),
			misfire_grace_seconds        INTEGER NOT NULL DEFAULT 0
				CHECK (misfire_grace_seconds >= 0),
			consecutive_resume_failures  INTEGER NOT NULL DEFAULT 0
				CHECK (consecutive_resume_failures >= 0),
			last_misfire_at              TEXT,
			misfire_count                INTEGER NOT NULL DEFAULT 0
				CHECK (misfire_count >= 0),
			updated_at                   TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_automation_scheduler_next_run
			ON automation_scheduler_state(next_run_at);`,
		`CREATE INDEX IF NOT EXISTS idx_automation_scheduler_misfire
			ON automation_scheduler_state(last_misfire_at);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_automation_runs_fire_id
			ON automation_runs(fire_id) WHERE fire_id IS NOT NULL;`,
	}

	runColumns, err := tableColumns(ctx, tx, "automation_runs")
	if err != nil {
		return err
	}
	runColumnStatements := make([]string, 0, 4)
	if _, ok := runColumns["fire_id"]; !ok {
		runColumnStatements = append(runColumnStatements, `ALTER TABLE automation_runs ADD COLUMN fire_id TEXT;`)
	}
	if _, ok := runColumns["scheduled_at"]; !ok {
		runColumnStatements = append(runColumnStatements, `ALTER TABLE automation_runs ADD COLUMN scheduled_at TEXT;`)
	}
	if _, ok := runColumns["delivery_error"]; !ok {
		runColumnStatements = append(runColumnStatements, `ALTER TABLE automation_runs ADD COLUMN delivery_error TEXT;`)
	}
	if _, ok := runColumns["delivery_error_at"]; !ok {
		runColumnStatements = append(
			runColumnStatements,
			`ALTER TABLE automation_runs ADD COLUMN delivery_error_at TEXT;`,
		)
	}
	statements = append(runColumnStatements, statements...)

	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("store: migrate automation scheduler state: %w", err)
		}
	}
	return nil
}

// GlobalDB owns the global session index and observability database.
type GlobalDB struct {
	db     *sql.DB
	path   string
	now    func() time.Time
	closed atomic.Int32
}

var _ store.SessionRegistry = (*GlobalDB)(nil)
var _ aghworkspace.Store = (*GlobalDB)(nil)

// OpenGlobalDB opens or creates the global AGH index database.
func OpenGlobalDB(ctx context.Context, path string) (*GlobalDB, error) {
	if ctx == nil {
		return nil, errors.New("store: open global database context is required")
	}

	db, err := openGlobalSQLite(ctx, path)
	if err != nil {
		return nil, err
	}
	if err := enforcePrivateGlobalDBFiles(path); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &GlobalDB{
		db:   db,
		path: strings.TrimSpace(path),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
}

func enforcePrivateGlobalDBFiles(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil
	}
	for _, candidate := range []string{trimmed, trimmed + "-wal", trimmed + "-shm"} {
		info, err := os.Stat(candidate)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("store: stat global database file %q: %w", candidate, err)
		}
		if info.IsDir() {
			continue
		}
		mode := info.Mode().Perm()
		if mode&0o077 == 0 {
			continue
		}
		if err := os.Chmod(candidate, mode&0o700); err != nil {
			return fmt.Errorf("store: restrict global database file permissions %q: %w", candidate, err)
		}
	}
	return nil
}

func (g *GlobalDB) checkReady(ctx context.Context, action string) error {
	if g == nil {
		return errors.New("store: global database is required")
	}
	if g.closed.Load() != 0 {
		return store.ErrClosed
	}
	if ctx == nil {
		return fmt.Errorf("store: %s context is required", action)
	}
	return nil
}

// Path reports the on-disk path for the global database file.
func (g *GlobalDB) Path() string {
	if g == nil {
		return ""
	}
	return g.path
}

// DB exposes the underlying SQL connection for composition-root adapters such
// as the extension registry.
func (g *GlobalDB) DB() *sql.DB {
	if g == nil {
		return nil
	}
	return g.db
}

// Close checkpoints the WAL and closes the database.
func (g *GlobalDB) Close(ctx context.Context) error {
	if g == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("store: close global database context is required")
	}
	if !g.closed.CompareAndSwap(0, 1) {
		return nil
	}

	checkpointErr := store.Checkpoint(ctx, g.db)
	closeErr := g.db.Close()
	return errors.Join(checkpointErr, closeErr)
}

func openGlobalSQLite(ctx context.Context, path string) (*sql.DB, error) {
	return store.OpenSQLiteDatabase(ctx, path, func(ctx context.Context, db *sql.DB) error {
		if err := migrateGlobalSchema(ctx, db); err != nil {
			return err
		}
		return reconcileLegacySessionMetaWorkspaceIDs(ctx, db, sessionsDirForDatabasePath(path))
	})
}
