package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/store"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

var taskTableIndexStatements = []string{
	`CREATE INDEX IF NOT EXISTS idx_tasks_scope ON tasks(scope);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_workspace ON tasks(workspace_id);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_priority ON tasks(priority);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_approval_state ON tasks(approval_state);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_owner ON tasks(owner_kind, owner_ref);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_channel ON tasks(network_channel);`,
}

var taskEventIndexStatements = []string{
	`CREATE INDEX IF NOT EXISTS idx_task_events_task ON task_events(task_id, timestamp DESC, id DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_task_events_run ON task_events(run_id, timestamp DESC, id DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_task_events_type ON task_events(event_type, timestamp DESC, id DESC);`,
	`CREATE UNIQUE INDEX IF NOT EXISTS uq_task_events_event_seq ON task_events(event_seq);`,
	`CREATE INDEX IF NOT EXISTS idx_task_events_task_seq ON task_events(task_id, event_seq ASC);`,
}

var globalSchemaStatements = append([]string{
	`CREATE TABLE IF NOT EXISTS workspaces (
		id            TEXT PRIMARY KEY,
		root_dir      TEXT NOT NULL UNIQUE,
		add_dirs      TEXT NOT NULL DEFAULT '[]',
		name          TEXT NOT NULL UNIQUE,
		default_agent TEXT DEFAULT '',
		environment_ref TEXT NOT NULL DEFAULT '',
		created_at    TEXT NOT NULL,
		updated_at    TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_workspaces_name ON workspaces(name);`,
	`CREATE TABLE IF NOT EXISTS sessions (
		id             TEXT PRIMARY KEY,
		name           TEXT,
		agent_name     TEXT NOT NULL,
		workspace_id   TEXT NOT NULL REFERENCES workspaces(id),
		session_type   TEXT NOT NULL DEFAULT 'user',
		channel          TEXT NOT NULL DEFAULT '',
		state          TEXT NOT NULL,
		acp_session_id TEXT,
		stop_reason    TEXT,
		stop_detail    TEXT,
		environment_id TEXT NOT NULL DEFAULT '',
		environment_backend TEXT NOT NULL DEFAULT 'local',
		environment_profile TEXT NOT NULL DEFAULT '',
		environment_instance_id TEXT NOT NULL DEFAULT '',
		environment_state TEXT NOT NULL DEFAULT '',
		environment_provider_state_json TEXT NOT NULL DEFAULT '',
		environment_last_sync_at TEXT,
		environment_last_sync_error TEXT NOT NULL DEFAULT '',
		created_at     TEXT NOT NULL,
		updated_at     TEXT NOT NULL
	);`,
	`CREATE TABLE IF NOT EXISTS event_summaries (
		id         TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		type       TEXT NOT NULL,
		agent_name TEXT NOT NULL,
		summary    TEXT,
		timestamp  TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_summaries_session ON event_summaries(session_id);`,
	`CREATE INDEX IF NOT EXISTS idx_summaries_type ON event_summaries(type);`,
	`CREATE INDEX IF NOT EXISTS idx_summaries_timestamp ON event_summaries(timestamp);`,
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
	`CREATE TABLE IF NOT EXISTS network_audit_log (
		id         TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		direction  TEXT NOT NULL,
		kind       TEXT NOT NULL,
		channel      TEXT NOT NULL,
		peer_from  TEXT NOT NULL,
		peer_to    TEXT,
		message_id TEXT NOT NULL,
		reason     TEXT,
		size       INTEGER NOT NULL,
		timestamp  TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_ts ON network_audit_log(timestamp);`,
	`CREATE INDEX IF NOT EXISTS idx_net_audit_session ON network_audit_log(session_id);`,
	`CREATE TABLE IF NOT EXISTS network_message_log (
		message_id TEXT PRIMARY KEY,
		session_id TEXT,
		channel    TEXT NOT NULL,
		peer_from  TEXT NOT NULL,
		kind       TEXT NOT NULL,
		intent     TEXT,
		text       TEXT NOT NULL,
		timestamp  TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_net_msg_channel_ts ON network_message_log(channel, timestamp);`,
	`CREATE INDEX IF NOT EXISTS idx_net_msg_peer_ts ON network_message_log(peer_from, timestamp);`,
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
		remote_version TEXT
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
	`CREATE TABLE IF NOT EXISTS automation_trigger_webhook_secrets (
		trigger_id  TEXT PRIMARY KEY,
		secret      TEXT NOT NULL,
		updated_at  TEXT NOT NULL
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
	`CREATE TABLE IF NOT EXISTS task_runs (
		id              TEXT PRIMARY KEY,
		task_id         TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		status          TEXT NOT NULL CHECK (
			status IN (
				'queued', 'claimed', 'starting', 'running', 'completed', 'failed', 'canceled'
			)
		),
		attempt         INTEGER NOT NULL CHECK (attempt > 0),
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
		result_json     TEXT,
		CHECK (
			(claimed_by_kind IS NULL AND claimed_by_ref IS NULL) OR
			(claimed_by_kind IS NOT NULL AND claimed_by_ref IS NOT NULL)
		),
		CHECK (status <> 'queued' OR session_id IS NULL)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_task ON task_runs(task_id, queued_at DESC, id DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_task_status ON task_runs(task_id, status, queued_at DESC, id DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_task_runs_status ON task_runs(status);`,
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
		actor_ref   TEXT NOT NULL,
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
	`CREATE TABLE IF NOT EXISTS task_triage_state (
		task_id               TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		actor_kind            TEXT NOT NULL CHECK (
			actor_kind IN (
				'human', 'agent_session', 'automation', 'extension', 'network_peer', 'daemon'
			)
		),
		actor_ref             TEXT NOT NULL,
		is_read               BOOLEAN NOT NULL DEFAULT 0,
		archived              BOOLEAN NOT NULL DEFAULT 0,
		dismissed             BOOLEAN NOT NULL DEFAULT 0,
		last_seen_activity_at TEXT,
		updated_at            TEXT NOT NULL,
		PRIMARY KEY (task_id, actor_kind, actor_ref)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_task_triage_task ON task_triage_state(task_id, updated_at DESC);`,
	`CREATE INDEX IF NOT EXISTS idx_task_triage_actor ON task_triage_state(actor_kind, actor_ref, updated_at DESC, task_id);`,
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
		degradation_reason TEXT,
		degradation_message TEXT,
		created_at        TEXT NOT NULL,
		updated_at        TEXT NOT NULL
	);`,
	`CREATE INDEX IF NOT EXISTS idx_bridge_instances_scope ON bridge_instances(scope, workspace_id, id);`,
	`CREATE TABLE IF NOT EXISTS bridge_secret_bindings (
		bridge_instance_id TEXT NOT NULL REFERENCES bridge_instances(id) ON DELETE CASCADE,
		binding_name        TEXT NOT NULL,
		vault_ref           TEXT NOT NULL,
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
}, resources.SchemaStatements()...)

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

	return &GlobalDB{
		db:   db,
		path: strings.TrimSpace(path),
		now: func() time.Time {
			return time.Now().UTC()
		},
	}, nil
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
		if err := store.EnsureSchema(ctx, db, globalSchemaStatements); err != nil {
			return err
		}
		return reconcileLegacySessionMetaWorkspaceIDs(ctx, db, sessionsDirForDatabasePath(path))
	})
}
