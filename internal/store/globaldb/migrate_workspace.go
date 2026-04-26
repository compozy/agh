package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/store"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

type sqlQueryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type legacySessionRow struct {
	ID           string
	Name         sql.NullString
	AgentName    string
	Provider     string
	Workspace    string
	SessionType  string
	State        string
	ACPSessionID sql.NullString
	CreatedAt    string
	UpdatedAt    string
}

type legacyWorkspaceSeed struct {
	rootDir   string
	createdAt string
	updatedAt string
}

type legacySessionMetaCompat struct {
	ID           string    `json:"id"`
	Name         string    `json:"name,omitempty"`
	AgentName    string    `json:"agent_name"`
	Workspace    string    `json:"workspace,omitempty"`
	WorkspaceID  string    `json:"workspace_id,omitempty"`
	SessionType  string    `json:"session_type,omitempty"`
	State        string    `json:"state"`
	ACPSessionID *string   `json:"acp_session_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func migrateGlobalSchema(ctx context.Context, db *sql.DB) error {
	if err := migrateExtensionColumns(ctx, db); err != nil {
		return err
	}
	if err := migrateBridgeColumns(ctx, db); err != nil {
		return err
	}
	if err := migrateBridgeInstanceColumns(ctx, db); err != nil {
		return err
	}
	if err := migrateWorkspaceColumns(ctx, db); err != nil {
		return err
	}
	if err := migrateTaskTables(ctx, db); err != nil {
		return err
	}
	if err := migrateNetworkAuditTable(ctx, db); err != nil {
		return err
	}
	if err := migrateNetworkChannelsTable(ctx, db); err != nil {
		return err
	}
	hasSessions, err := tableExists(ctx, db, "sessions")
	if err != nil {
		return err
	}
	if hasSessions {
		columns, err := tableColumns(ctx, db, "sessions")
		if err != nil {
			return err
		}

		_, hasWorkspaceID := columns["workspace_id"]
		_, hasLegacyWorkspace := columns["workspace"]
		if !hasWorkspaceID && hasLegacyWorkspace {
			if err := migrateLegacyGlobalSessions(ctx, db); err != nil {
				return err
			}
		}

		if err := migrateSessionColumns(ctx, db); err != nil {
			return err
		}
	}

	return store.RunMigrations(ctx, db, globalSchemaMigrations)
}

func migrateWorkspaceColumns(ctx context.Context, db *sql.DB) error {
	exists, err := tableExists(ctx, db, "workspaces")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	columns, err := tableColumns(ctx, db, "workspaces")
	if err != nil {
		return err
	}
	if _, ok := columns["environment_ref"]; ok {
		return nil
	}

	if _, err := db.ExecContext(
		ctx,
		`ALTER TABLE workspaces ADD COLUMN environment_ref TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("store: add workspaces.environment_ref column: %w", err)
	}
	return nil
}

func migrateTaskTables(ctx context.Context, db *sql.DB) error {
	spec, needsRebuild, err := taskTableMigrationSpecForExistingSchema(ctx, db)
	if err != nil {
		return err
	}
	if needsRebuild {
		if err := rebuildTaskTable(ctx, db, spec); err != nil {
			return err
		}
	}
	if err := migrateTaskRunMetadataColumn(ctx, db); err != nil {
		return err
	}
	return migrateTaskEventSequenceColumn(ctx, db)
}

func migrateTaskRunMetadataColumn(ctx context.Context, db *sql.DB) error {
	exists, err := tableExists(ctx, db, "task_runs")
	if err != nil || !exists {
		return err
	}

	columns, err := tableColumns(ctx, db, "task_runs")
	if err != nil {
		return err
	}
	if _, ok := columns["metadata_json"]; ok {
		return nil
	}

	if _, err := db.ExecContext(ctx, `ALTER TABLE task_runs ADD COLUMN metadata_json TEXT`); err != nil {
		return fmt.Errorf("store: add task_runs.metadata_json column: %w", err)
	}
	return nil
}

type taskTableMigrationSpec struct {
	priorityExpr       string
	maxAttemptsExpr    string
	approvalPolicyExpr string
	approvalStateExpr  string
}

func taskTableMigrationSpecForExistingSchema(
	ctx context.Context,
	db *sql.DB,
) (taskTableMigrationSpec, bool, error) {
	exists, err := tableExists(ctx, db, "tasks")
	if err != nil || !exists {
		return taskTableMigrationSpec{}, false, err
	}

	columns, err := tableColumns(ctx, db, "tasks")
	if err != nil {
		return taskTableMigrationSpec{}, false, err
	}
	tableSQL, err := tableDefinitionSQL(ctx, db, "tasks")
	if err != nil {
		return taskTableMigrationSpec{}, false, err
	}
	if !taskTableNeedsRebuild(columns, tableSQL) {
		return taskTableMigrationSpec{}, false, nil
	}

	return taskTableMigrationSpec{
		priorityExpr:       existingTaskColumnExpr(columns, "priority", `'medium'`),
		maxAttemptsExpr:    existingTaskColumnExpr(columns, "max_attempts", "3"),
		approvalPolicyExpr: existingTaskColumnExpr(columns, "approval_policy", `'none'`),
		approvalStateExpr:  existingTaskColumnExpr(columns, "approval_state", `'not_required'`),
	}, true, nil
}

func taskTableNeedsRebuild(columns map[string]struct{}, tableSQL string) bool {
	if !strings.Contains(tableSQL, "'draft'") {
		return true
	}
	for _, column := range []string{"priority", "max_attempts", "approval_policy", "approval_state"} {
		if _, ok := columns[column]; !ok {
			return true
		}
	}
	return false
}

func existingTaskColumnExpr(columns map[string]struct{}, column string, fallback string) string {
	if _, ok := columns[column]; ok {
		return column
	}
	return fallback
}

func rebuildTaskTable(ctx context.Context, db *sql.DB, spec taskTableMigrationSpec) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open task schema migration connection: %w", err)
	}
	cleanupCtx := context.WithoutCancel(ctx)
	foreignKeysDisabled := false
	defer func() {
		if foreignKeysDisabled {
			joinCleanupError(&err, restoreForeignKeys(cleanupCtx, conn))
		}
		_ = conn.Close()
	}()

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("store: disable foreign keys for task schema migration: %w", err)
	}
	foreignKeysDisabled = true

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin task schema migration transaction: %w", err)
	}
	defer func() {
		joinCleanupError(&err, rollbackTx(tx, "task schema migration"))
	}()

	for _, stmt := range taskTableMigrationStatements(spec) {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate tasks table: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit task schema migration: %w", err)
	}
	return nil
}

func taskTableMigrationStatements(spec taskTableMigrationSpec) []string {
	statements := []string{
		taskTableCreateStatement(),
		taskTableCopyStatement(spec),
		`DROP TABLE tasks`,
		`ALTER TABLE tasks_new RENAME TO tasks`,
	}
	statements = append(statements, taskTableIndexStatements...)
	return statements
}

func migrateTaskEventSequenceColumn(ctx context.Context, db *sql.DB) error {
	exists, err := tableExists(ctx, db, "task_events")
	if err != nil || !exists {
		return err
	}

	columns, err := tableColumns(ctx, db, "task_events")
	if err != nil {
		return err
	}
	if _, ok := columns["event_seq"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE task_events ADD COLUMN event_seq INTEGER`); err != nil {
			return fmt.Errorf("store: add task_events.event_seq column: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE task_events SET event_seq = rowid WHERE event_seq IS NULL`); err != nil {
		return fmt.Errorf("store: backfill task_events.event_seq: %w", err)
	}
	for _, stmt := range taskEventIndexStatements[3:] {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: ensure task event sequence index: %w", err)
		}
	}
	return nil
}

func taskTableCreateStatement() string {
	return `CREATE TABLE tasks_new (
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
	);`
}

func taskTableCopyStatement(spec taskTableMigrationSpec) string {
	return fmt.Sprintf(
		`INSERT INTO tasks_new (
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			priority, max_attempts, status, approval_policy, approval_state,
			owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
			created_at, updated_at, closed_at, metadata_json
		) SELECT
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			%s, %s, status, %s, %s,
			owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
			created_at, updated_at, closed_at, metadata_json
		FROM tasks`,
		spec.priorityExpr,
		spec.maxAttemptsExpr,
		spec.approvalPolicyExpr,
		spec.approvalStateExpr,
	)
}

func migrateLegacyGlobalSessions(ctx context.Context, db *sql.DB) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open global schema migration connection: %w", err)
	}
	cleanupCtx := context.WithoutCancel(ctx)
	foreignKeysDisabled := false
	defer func() {
		if foreignKeysDisabled {
			joinCleanupError(&err, restoreForeignKeys(cleanupCtx, conn))
		}
		_ = conn.Close()
	}()

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("store: disable foreign keys for global schema migration: %w", err)
	}
	foreignKeysDisabled = true

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin global schema migration transaction: %w", err)
	}
	defer func() {
		joinCleanupError(&err, rollbackTx(tx, "global schema migration"))
	}()

	if err := migrateLegacyGlobalTables(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit global schema migration: %w", err)
	}
	return nil
}

func migrateLegacyGlobalTables(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, globalSchemaStatements[0]); err != nil {
		return fmt.Errorf("store: create workspaces table during migration: %w", err)
	}

	sessionRows, workspaceSeeds, err := loadLegacySessions(ctx, tx)
	if err != nil {
		return err
	}
	workspaceIDs, err := ensureMigratedWorkspaces(ctx, tx, workspaceSeeds)
	if err != nil {
		return err
	}
	if err := createMigratedGlobalTables(ctx, tx); err != nil {
		return err
	}
	if err := copyMigratedSessions(ctx, tx, sessionRows, workspaceIDs); err != nil {
		return err
	}
	if err := copyLegacyGlobalSupportTables(ctx, tx); err != nil {
		return err
	}
	return swapMigratedGlobalTables(ctx, tx)
}

func copyLegacyGlobalSupportTables(ctx context.Context, tx *sql.Tx) error {
	for _, spec := range []struct {
		source string
		target string
		query  string
	}{
		{
			source: "event_summaries",
			target: "event_summaries_new",
			query: `INSERT INTO event_summaries_new (id, session_id, type, agent_name, summary, timestamp)
				SELECT id, session_id, type, agent_name, summary, timestamp FROM event_summaries`,
		},
		{
			source: "token_stats",
			target: "token_stats_new",
			query: `INSERT INTO token_stats_new (
					id, session_id, agent_name, input_tokens, output_tokens, total_tokens,
					total_cost, cost_currency, turn_count, updated_at
				) SELECT
					id, session_id, agent_name, input_tokens, output_tokens, total_tokens,
					total_cost, cost_currency, turn_count, updated_at
				FROM token_stats`,
		},
		{
			source: "permission_log",
			target: "permission_log_new",
			query: `INSERT INTO permission_log_new (
					id, session_id, agent_name, action, resource, decision, policy_used, timestamp
				) SELECT
					id, session_id, agent_name, action, resource, decision, policy_used, timestamp
				FROM permission_log`,
		},
	} {
		if err := copyGlobalTableIfExists(ctx, tx, spec.source, spec.target, spec.query); err != nil {
			return err
		}
	}
	return nil
}

func migrateBridgeInstanceColumns(ctx context.Context, db *sql.DB) error {
	exists, err := tableExists(ctx, db, "bridge_instances")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	columns, err := tableColumns(ctx, db, "bridge_instances")
	if err != nil {
		return err
	}

	if _, ok := columns["dm_policy"]; !ok {
		if _, err := db.ExecContext(
			ctx,
			`ALTER TABLE bridge_instances ADD COLUMN dm_policy TEXT NOT NULL DEFAULT 'open'`,
		); err != nil {
			return fmt.Errorf("store: add bridge_instances.dm_policy column: %w", err)
		}
	}
	if _, ok := columns["provider_config"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE bridge_instances ADD COLUMN provider_config TEXT`); err != nil {
			return fmt.Errorf("store: add bridge_instances.provider_config column: %w", err)
		}
	}
	if _, ok := columns["degradation_reason"]; !ok {
		if _, err := db.ExecContext(
			ctx,
			`ALTER TABLE bridge_instances ADD COLUMN degradation_reason TEXT`,
		); err != nil {
			return fmt.Errorf("store: add bridge_instances.degradation_reason column: %w", err)
		}
	}
	if _, ok := columns["degradation_message"]; !ok {
		if _, err := db.ExecContext(
			ctx,
			`ALTER TABLE bridge_instances ADD COLUMN degradation_message TEXT`,
		); err != nil {
			return fmt.Errorf("store: add bridge_instances.degradation_message column: %w", err)
		}
	}

	return nil
}

func migrateExtensionColumns(ctx context.Context, db *sql.DB) error {
	exists, err := tableExists(ctx, db, "extensions")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	columns, err := tableColumns(ctx, db, "extensions")
	if err != nil {
		return err
	}

	if _, ok := columns["registry_slug"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE extensions ADD COLUMN registry_slug TEXT`); err != nil {
			return fmt.Errorf("store: add extensions.registry_slug column: %w", err)
		}
	}
	if _, ok := columns["registry_name"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE extensions ADD COLUMN registry_name TEXT`); err != nil {
			return fmt.Errorf("store: add extensions.registry_name column: %w", err)
		}
	}
	if _, ok := columns["remote_version"]; !ok {
		if _, err := db.ExecContext(ctx, `ALTER TABLE extensions ADD COLUMN remote_version TEXT`); err != nil {
			return fmt.Errorf("store: add extensions.remote_version column: %w", err)
		}
	}

	return nil
}

func migrateSessionColumns(ctx context.Context, db *sql.DB) error {
	columns, err := tableColumns(ctx, db, "sessions")
	if err != nil {
		return err
	}

	for _, column := range sessionColumnSpecs() {
		if _, ok := columns[column.name]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, column.sql); err != nil {
			return fmt.Errorf("store: add sessions.%s column: %w", column.name, err)
		}
	}
	if _, err := db.ExecContext(
		ctx,
		`UPDATE sessions SET root_session_id = id WHERE root_session_id IS NULL OR trim(root_session_id) = ''`,
	); err != nil {
		return fmt.Errorf("store: backfill sessions root lineage: %w", err)
	}

	return nil
}

type migrationColumnSpec struct {
	name string
	sql  string
}

func sessionColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{name: "provider", sql: `ALTER TABLE sessions ADD COLUMN provider TEXT NOT NULL DEFAULT ''`},
		{name: "stop_reason", sql: `ALTER TABLE sessions ADD COLUMN stop_reason TEXT`},
		{name: "stop_detail", sql: `ALTER TABLE sessions ADD COLUMN stop_detail TEXT`},
		{name: "failure_kind", sql: `ALTER TABLE sessions ADD COLUMN failure_kind TEXT`},
		{name: "failure_summary", sql: `ALTER TABLE sessions ADD COLUMN failure_summary TEXT NOT NULL DEFAULT ''`},
		{name: "crash_bundle_path", sql: `ALTER TABLE sessions ADD COLUMN crash_bundle_path TEXT NOT NULL DEFAULT ''`},
		{name: "channel", sql: `ALTER TABLE sessions ADD COLUMN channel TEXT NOT NULL DEFAULT ''`},
		{name: "subprocess_pid", sql: `ALTER TABLE sessions ADD COLUMN subprocess_pid INTEGER NOT NULL DEFAULT 0`},
		{name: "subprocess_started_at", sql: `ALTER TABLE sessions ADD COLUMN subprocess_started_at TEXT`},
		{name: "last_update_at", sql: `ALTER TABLE sessions ADD COLUMN last_update_at TEXT`},
		{name: "stall_state", sql: `ALTER TABLE sessions ADD COLUMN stall_state TEXT NOT NULL DEFAULT ''`},
		{name: "stall_reason", sql: `ALTER TABLE sessions ADD COLUMN stall_reason TEXT NOT NULL DEFAULT ''`},
		{name: "activity_json", sql: `ALTER TABLE sessions ADD COLUMN activity_json TEXT NOT NULL DEFAULT ''`},
		{name: "environment_id", sql: `ALTER TABLE sessions ADD COLUMN environment_id TEXT NOT NULL DEFAULT ''`},
		{
			name: "environment_backend",
			sql:  `ALTER TABLE sessions ADD COLUMN environment_backend TEXT NOT NULL DEFAULT 'local'`,
		},
		{
			name: "environment_profile",
			sql:  `ALTER TABLE sessions ADD COLUMN environment_profile TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: "environment_instance_id",
			sql:  `ALTER TABLE sessions ADD COLUMN environment_instance_id TEXT NOT NULL DEFAULT ''`,
		},
		{name: "environment_state", sql: `ALTER TABLE sessions ADD COLUMN environment_state TEXT NOT NULL DEFAULT ''`},
		{
			name: "environment_provider_state_json",
			sql:  `ALTER TABLE sessions ADD COLUMN environment_provider_state_json TEXT NOT NULL DEFAULT ''`,
		},
		{name: "environment_last_sync_at", sql: `ALTER TABLE sessions ADD COLUMN environment_last_sync_at TEXT`},
		{
			name: "environment_last_sync_error",
			sql:  `ALTER TABLE sessions ADD COLUMN environment_last_sync_error TEXT NOT NULL DEFAULT ''`,
		},
		{name: "parent_session_id", sql: `ALTER TABLE sessions ADD COLUMN parent_session_id TEXT`},
		{name: "root_session_id", sql: `ALTER TABLE sessions ADD COLUMN root_session_id TEXT`},
		{name: "spawn_depth", sql: `ALTER TABLE sessions ADD COLUMN spawn_depth INTEGER NOT NULL DEFAULT 0`},
		{name: "spawn_role", sql: `ALTER TABLE sessions ADD COLUMN spawn_role TEXT`},
		{name: "ttl_expires_at", sql: `ALTER TABLE sessions ADD COLUMN ttl_expires_at TEXT`},
		{
			name: "auto_stop_on_parent",
			sql:  `ALTER TABLE sessions ADD COLUMN auto_stop_on_parent BOOLEAN NOT NULL DEFAULT 0`,
		},
		{
			name: "spawn_budget_json",
			sql:  `ALTER TABLE sessions ADD COLUMN spawn_budget_json TEXT NOT NULL DEFAULT '{}'`,
		},
		{
			name: "permission_policy_json",
			sql:  `ALTER TABLE sessions ADD COLUMN permission_policy_json TEXT NOT NULL DEFAULT '{}'`,
		},
	}
}

func migrateSessionFailureColumns(ctx context.Context, tx *sql.Tx) error {
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
	for _, column := range sessionFailureColumnSpecs() {
		if _, ok := columns[column.name]; ok {
			continue
		}
		if _, err := tx.ExecContext(ctx, column.sql); err != nil {
			return fmt.Errorf("store: add sessions.%s column: %w", column.name, err)
		}
	}
	return nil
}

func sessionFailureColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{name: "failure_kind", sql: `ALTER TABLE sessions ADD COLUMN failure_kind TEXT`},
		{name: "failure_summary", sql: `ALTER TABLE sessions ADD COLUMN failure_summary TEXT NOT NULL DEFAULT ''`},
		{name: "crash_bundle_path", sql: `ALTER TABLE sessions ADD COLUMN crash_bundle_path TEXT NOT NULL DEFAULT ''`},
	}
}

func migrateBridgeColumns(ctx context.Context, db *sql.DB) error {
	exists, err := tableExists(ctx, db, "bridge_instances")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	columns, err := tableColumns(ctx, db, "bridge_instances")
	if err != nil {
		return err
	}
	if _, ok := columns["source"]; ok {
		return nil
	}

	if _, err := db.ExecContext(
		ctx,
		`ALTER TABLE bridge_instances ADD COLUMN source TEXT NOT NULL DEFAULT 'dynamic'`,
	); err != nil {
		return fmt.Errorf("store: add bridge_instances.source column: %w", err)
	}
	return nil
}

func migrateNetworkAuditTable(ctx context.Context, db *sql.DB) (err error) {
	exists, err := tableExists(ctx, db, "network_audit_log")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	hasSessionFK, err := tableHasForeignKey(ctx, db, "network_audit_log", "sessions")
	if err != nil {
		return err
	}
	if !hasSessionFK {
		return nil
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open network audit migration connection: %w", err)
	}
	cleanupCtx := context.WithoutCancel(ctx)
	foreignKeysDisabled := false
	defer func() {
		if foreignKeysDisabled {
			joinCleanupError(&err, restoreForeignKeys(cleanupCtx, conn))
		}
		_ = conn.Close()
	}()

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("store: disable foreign keys for network audit migration: %w", err)
	}
	foreignKeysDisabled = true

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin network audit migration transaction: %w", err)
	}
	defer func() {
		joinCleanupError(&err, rollbackTx(tx, "network audit migration"))
	}()

	statements := []string{
		`CREATE TABLE network_audit_log_new (
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
		`INSERT INTO network_audit_log_new (
			id, session_id, direction, kind, channel, peer_from, peer_to, message_id, reason, size, timestamp
		) SELECT
			id, session_id, direction, kind, channel, peer_from, peer_to, message_id, reason, size, timestamp
		FROM network_audit_log`,
		`DROP TABLE network_audit_log`,
		`ALTER TABLE network_audit_log_new RENAME TO network_audit_log`,
		`CREATE INDEX idx_net_audit_ts ON network_audit_log(timestamp)`,
		`CREATE INDEX idx_net_audit_session ON network_audit_log(session_id)`,
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate network_audit_log table: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit network audit migration: %w", err)
	}
	return nil
}

func migrateNetworkChannelsTable(ctx context.Context, db *sql.DB) (err error) {
	exists, err := tableExists(ctx, db, "network_channels")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	hasWorkspaceFK, err := tableHasForeignKey(ctx, db, "network_channels", "workspaces")
	if err != nil {
		return err
	}
	if hasWorkspaceFK {
		return nil
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: open network channels migration connection: %w", err)
	}
	cleanupCtx := context.WithoutCancel(ctx)
	foreignKeysDisabled := false
	defer func() {
		if foreignKeysDisabled {
			joinCleanupError(&err, restoreForeignKeys(cleanupCtx, conn))
		}
		_ = conn.Close()
	}()

	if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("store: disable foreign keys for network channels migration: %w", err)
	}
	foreignKeysDisabled = true

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin network channels migration transaction: %w", err)
	}
	defer func() {
		joinCleanupError(&err, rollbackTx(tx, "network channels migration"))
	}()

	statements := []string{
		`CREATE TABLE network_channels_new (
			channel      TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			purpose      TEXT NOT NULL,
			created_by   TEXT NOT NULL DEFAULT '',
			created_at   TEXT NOT NULL,
			updated_at   TEXT NOT NULL
		);`,
		`INSERT INTO network_channels_new (
			channel, workspace_id, purpose, created_by, created_at, updated_at
		) SELECT
			channel, TRIM(workspace_id), purpose, created_by, created_at, updated_at
		FROM network_channels
		WHERE TRIM(workspace_id) IN (SELECT id FROM workspaces)`,
		`DROP TABLE network_channels`,
		`ALTER TABLE network_channels_new RENAME TO network_channels`,
	}
	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: migrate network_channels table: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit network channels migration: %w", err)
	}
	return nil
}

func loadLegacySessions(
	ctx context.Context,
	exec sqlQueryExecutor,
) ([]legacySessionRow, map[string]legacyWorkspaceSeed, error) {
	columns, err := tableColumns(ctx, exec, "sessions")
	if err != nil {
		return nil, nil, err
	}
	providerExpr := `''`
	if _, ok := columns["provider"]; ok {
		providerExpr = `COALESCE(provider, '')`
	}

	rows, err := exec.QueryContext(
		ctx,
		fmt.Sprintf(
			`SELECT id, name, agent_name, %s AS provider, workspace, session_type, state, acp_session_id, created_at, updated_at
			FROM sessions ORDER BY created_at ASC, id ASC`,
			providerExpr,
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("store: query legacy sessions for migration: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	sessions := make([]legacySessionRow, 0)
	seeds := make(map[string]legacyWorkspaceSeed)
	for rows.Next() {
		var row legacySessionRow
		if err := rows.Scan(
			&row.ID,
			&row.Name,
			&row.AgentName,
			&row.Provider,
			&row.Workspace,
			&row.SessionType,
			&row.State,
			&row.ACPSessionID,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("store: scan legacy session for migration: %w", err)
		}

		rootDir := strings.TrimSpace(row.Workspace)
		if rootDir == "" {
			return nil, nil, fmt.Errorf("store: migrate legacy session %q: workspace path is required", row.ID)
		}

		seed, ok := seeds[rootDir]
		if !ok {
			seeds[rootDir] = legacyWorkspaceSeed{rootDir: rootDir, createdAt: row.CreatedAt, updatedAt: row.UpdatedAt}
		} else {
			if strings.TrimSpace(row.CreatedAt) != "" && (seed.createdAt == "" || row.CreatedAt < seed.createdAt) {
				seed.createdAt = row.CreatedAt
			}
			if strings.TrimSpace(row.UpdatedAt) != "" && row.UpdatedAt > seed.updatedAt {
				seed.updatedAt = row.UpdatedAt
			}
			seeds[rootDir] = seed
		}

		sessions = append(sessions, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("store: iterate legacy sessions for migration: %w", err)
	}

	return sessions, seeds, nil
}

func ensureMigratedWorkspaces(
	ctx context.Context,
	tx *sql.Tx,
	seeds map[string]legacyWorkspaceSeed,
) (map[string]string, error) {
	rootToID, err := loadWorkspaceIDsByRootDir(ctx, tx)
	if err != nil {
		return nil, err
	}
	takenNames, err := loadWorkspaceNames(ctx, tx)
	if err != nil {
		return nil, err
	}

	if len(seeds) == 0 {
		return rootToID, nil
	}

	orderedRoots := make([]string, 0, len(seeds))
	for rootDir := range seeds {
		orderedRoots = append(orderedRoots, rootDir)
	}
	sort.Strings(orderedRoots)

	for _, rootDir := range orderedRoots {
		if _, ok := rootToID[rootDir]; ok {
			continue
		}

		seed := seeds[rootDir]
		name := aghworkspace.UniqueWorkspaceName(rootDir, takenNames)
		workspaceID := store.NewID("ws")
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO workspaces (id, root_dir, add_dirs, name, default_agent, environment_ref, created_at, updated_at)
			 VALUES (?, ?, '[]', ?, '', '', ?, ?)`,
			workspaceID,
			rootDir,
			name,
			coalesceTimestamp(seed.createdAt),
			coalesceTimestamp(seed.updatedAt),
		); err != nil {
			return nil, fmt.Errorf("store: insert migrated workspace for %q: %w", rootDir, err)
		}

		rootToID[rootDir] = workspaceID
		takenNames[name] = struct{}{}
	}

	return rootToID, nil
}

func createMigratedGlobalTables(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`CREATE TABLE sessions_new (
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
			failure_kind   TEXT,
			failure_summary TEXT NOT NULL DEFAULT '',
			crash_bundle_path TEXT NOT NULL DEFAULT '',
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
		`CREATE TABLE event_summaries_new (
			id         TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES sessions_new(id),
			type       TEXT NOT NULL,
			agent_name TEXT NOT NULL,
			summary    TEXT,
			timestamp  TEXT NOT NULL
		);`,
		`CREATE TABLE token_stats_new (
			id            TEXT PRIMARY KEY,
			session_id    TEXT NOT NULL REFERENCES sessions_new(id),
			agent_name    TEXT NOT NULL,
			input_tokens  INTEGER,
			output_tokens INTEGER,
			total_tokens  INTEGER,
			total_cost    REAL,
			cost_currency TEXT,
			turn_count    INTEGER NOT NULL DEFAULT 0,
			updated_at    TEXT NOT NULL
		);`,
		`CREATE TABLE permission_log_new (
			id          TEXT PRIMARY KEY,
			session_id  TEXT NOT NULL REFERENCES sessions_new(id),
			agent_name  TEXT NOT NULL,
			action      TEXT NOT NULL,
			resource    TEXT NOT NULL,
			decision    TEXT NOT NULL,
			policy_used TEXT NOT NULL,
			timestamp   TEXT NOT NULL
		);`,
	}

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: create migrated global table: %w", err)
		}
	}

	return nil
}

func copyMigratedSessions(
	ctx context.Context,
	tx *sql.Tx,
	sessions []legacySessionRow,
	workspaceIDs map[string]string,
) error {
	for _, row := range sessions {
		workspaceID, ok := workspaceIDs[strings.TrimSpace(row.Workspace)]
		if !ok {
			return fmt.Errorf("store: missing migrated workspace id for legacy root %q", row.Workspace)
		}

		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO sessions_new (
				id, name, agent_name, provider, workspace_id, session_type, channel, state, acp_session_id,
				environment_backend, environment_profile, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			row.ID,
			nullStringValue(row.Name),
			row.AgentName,
			strings.TrimSpace(row.Provider),
			workspaceID,
			store.NormalizeSessionType(row.SessionType),
			"",
			row.State,
			nullStringValue(row.ACPSessionID),
			"local",
			"local",
			row.CreatedAt,
			row.UpdatedAt,
		); err != nil {
			return fmt.Errorf("store: copy migrated session %q: %w", row.ID, err)
		}
	}

	return nil
}

func copyGlobalTableIfExists(ctx context.Context, tx *sql.Tx, source string, target string, insertSQL string) error {
	exists, err := tableExists(ctx, tx, source)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if _, err := tx.ExecContext(ctx, insertSQL); err != nil {
		return fmt.Errorf("store: copy %s into %s: %w", source, target, err)
	}
	return nil
}

func swapMigratedGlobalTables(ctx context.Context, tx *sql.Tx) error {
	statements := []string{
		`DROP TABLE IF EXISTS event_summaries`,
		`DROP TABLE IF EXISTS token_stats`,
		`DROP TABLE IF EXISTS permission_log`,
		`DROP TABLE IF EXISTS sessions`,
		`ALTER TABLE sessions_new RENAME TO sessions`,
		`ALTER TABLE event_summaries_new RENAME TO event_summaries`,
		`ALTER TABLE token_stats_new RENAME TO token_stats`,
		`ALTER TABLE permission_log_new RENAME TO permission_log`,
	}

	for _, stmt := range statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: swap migrated global tables: %w", err)
		}
	}

	return nil
}

func loadWorkspaceIDsByRootDir(ctx context.Context, exec sqlQueryExecutor) (map[string]string, error) {
	exists, err := tableExists(ctx, exec, "workspaces")
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]string{}, nil
	}

	rows, err := exec.QueryContext(ctx, `SELECT id, root_dir FROM workspaces`)
	if err != nil {
		return nil, fmt.Errorf("store: query workspace ids by root_dir: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	rootToID := make(map[string]string)
	for rows.Next() {
		var id string
		var rootDir string
		if err := rows.Scan(&id, &rootDir); err != nil {
			return nil, fmt.Errorf("store: scan workspace id by root_dir: %w", err)
		}
		rootToID[strings.TrimSpace(rootDir)] = strings.TrimSpace(id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate workspace ids by root_dir: %w", err)
	}

	return rootToID, nil
}

func loadWorkspaceNames(ctx context.Context, exec sqlQueryExecutor) (map[string]struct{}, error) {
	exists, err := tableExists(ctx, exec, "workspaces")
	if err != nil {
		return nil, err
	}
	if !exists {
		return map[string]struct{}{}, nil
	}

	rows, err := exec.QueryContext(ctx, `SELECT name FROM workspaces`)
	if err != nil {
		return nil, fmt.Errorf("store: query workspace names: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	names := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("store: scan workspace name: %w", err)
		}
		names[strings.TrimSpace(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate workspace names: %w", err)
	}

	return names, nil
}

func tableExists(ctx context.Context, exec sqlQueryExecutor, table string) (bool, error) {
	var name string
	err := exec.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, strings.TrimSpace(table)).
		Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: check table %q existence: %w", table, err)
	}
	return true, nil
}

func tableDefinitionSQL(ctx context.Context, exec sqlQueryExecutor, table string) (string, error) {
	var sqlText sql.NullString
	if err := exec.QueryRowContext(
		ctx,
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`,
		strings.TrimSpace(table),
	).Scan(&sqlText); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("store: table %q does not exist", table)
		}
		return "", fmt.Errorf("store: query table definition for %q: %w", table, err)
	}
	if !sqlText.Valid {
		return "", fmt.Errorf("store: table definition for %q is empty", table)
	}
	return strings.TrimSpace(sqlText.String), nil
}

func tableColumns(ctx context.Context, exec sqlQueryExecutor, table string) (map[string]struct{}, error) {
	name, err := store.NormalizeSQLiteIdentifier(table)
	if err != nil {
		return nil, err
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%s)", name))
	if err != nil {
		return nil, fmt.Errorf("store: query table info for %q: %w", table, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	columns := make(map[string]struct{})
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return nil, fmt.Errorf("store: scan table info for %q: %w", table, err)
		}
		columns[strings.TrimSpace(name)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iterate table info for %q: %w", table, err)
	}

	return columns, nil
}

func tableHasForeignKey(
	ctx context.Context,
	exec sqlQueryExecutor,
	table string,
	referencedTable string,
) (bool, error) {
	name, err := store.NormalizeSQLiteIdentifier(table)
	if err != nil {
		return false, err
	}

	rows, err := exec.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%s)", name))
	if err != nil {
		return false, fmt.Errorf("store: query foreign key info for %q: %w", table, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	target := strings.TrimSpace(referencedTable)
	for rows.Next() {
		var (
			id       int
			seq      int
			refTable string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		)
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return false, fmt.Errorf("store: scan foreign key info for %q: %w", table, err)
		}
		if strings.EqualFold(strings.TrimSpace(refTable), target) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("store: iterate foreign key info for %q: %w", table, err)
	}

	return false, nil
}

func coalesceTimestamp(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed != "" {
		return trimmed
	}
	return store.FormatTimestamp(time.Now().UTC())
}

func nullStringValue(value sql.NullString) any {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func sessionsDirForDatabasePath(path string) string {
	cleanPath := strings.TrimSpace(path)
	if cleanPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(cleanPath), "sessions")
}

func reconcileLegacySessionMetaWorkspaceIDs(ctx context.Context, exec sqlQueryExecutor, sessionsDir string) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("store: reconcile session metadata workspace ids canceled: %w", err)
	}

	rootToID, err := loadWorkspaceIDsByRootDir(ctx, exec)
	if err != nil {
		return err
	}
	if len(rootToID) == 0 {
		return nil
	}

	cleanDir := strings.TrimSpace(sessionsDir)
	if cleanDir == "" {
		return nil
	}

	entries, err := os.ReadDir(cleanDir)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return nil
	default:
		return fmt.Errorf("store: read sessions directory %q for workspace id reconciliation: %w", cleanDir, err)
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("store: reconcile session metadata workspace ids canceled: %w", err)
		}
		if !entry.IsDir() {
			continue
		}

		metaPath := store.SessionMetaFile(filepath.Join(cleanDir, entry.Name()))
		needsRewrite, meta, err := loadReconciledLegacySessionMeta(metaPath, rootToID)
		if err != nil {
			return err
		}
		if !needsRewrite {
			continue
		}
		if err := store.WriteSessionMeta(metaPath, meta); err != nil {
			return fmt.Errorf("store: rewrite legacy session meta %q: %w", metaPath, err)
		}
	}

	return nil
}

func loadReconciledLegacySessionMeta(path string, rootToID map[string]string) (bool, store.SessionMeta, error) {
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		return false, store.SessionMeta{}, nil
	default:
		return false, store.SessionMeta{}, fmt.Errorf(
			"store: read session meta %q for workspace id reconciliation: %w",
			path,
			err,
		)
	}

	var raw legacySessionMetaCompat
	if err := json.Unmarshal(data, &raw); err != nil {
		return false, store.SessionMeta{}, nil
	}

	if strings.TrimSpace(raw.WorkspaceID) != "" {
		return false, store.SessionMeta{}, nil
	}

	workspaceRoot := strings.TrimSpace(raw.Workspace)
	if workspaceRoot == "" {
		return false, store.SessionMeta{}, nil
	}

	workspaceID, ok := rootToID[workspaceRoot]
	if !ok {
		return false, store.SessionMeta{}, nil
	}

	meta := store.SessionMeta{
		ID:           raw.ID,
		Name:         raw.Name,
		AgentName:    raw.AgentName,
		WorkspaceID:  workspaceID,
		SessionType:  raw.SessionType,
		State:        raw.State,
		ACPSessionID: raw.ACPSessionID,
		CreatedAt:    raw.CreatedAt,
		UpdatedAt:    raw.UpdatedAt,
	}
	if err := meta.Validate(); err != nil {
		return false, store.SessionMeta{}, nil
	}

	return true, meta, nil
}
