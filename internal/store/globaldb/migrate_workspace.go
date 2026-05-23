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

	"github.com/compozy/agh/internal/store"
	aghworkspace "github.com/compozy/agh/internal/workspace"
)

const (
	migrateWorkspaceActivityJSONKey             = "activity_json"
	migrateWorkspaceAutoStopOnParentKey         = "auto_stop_on_parent"
	migrateWorkspaceChannelKey                  = "channel"
	migrateWorkspaceCrashBundlePathKey          = "crash_bundle_path"
	migrateWorkspaceFailureKindKey              = "failure_kind"
	migrateWorkspaceFailureSummaryKey           = "failure_summary"
	migrateWorkspaceLastUpdateAtKey             = "last_update_at"
	migrateWorkspaceParentSessionIDKey          = "parent_session_id"
	migrateWorkspacePermissionPolicyJSONKey     = "permission_policy_json"
	migrateWorkspacePriorityKey                 = "priority"
	migrateWorkspaceProviderKey                 = "provider"
	migrateWorkspaceRootSessionIDKey            = "root_session_id"
	migrateWorkspaceSandboxBackendKey           = "sandbox_backend"
	migrateWorkspaceSandboxIDKey                = "sandbox_id"
	migrateWorkspaceSandboxInstanceIDKey        = "sandbox_instance_id"
	migrateWorkspaceSandboxLastSyncAtKey        = "sandbox_last_sync_at"
	migrateWorkspaceSandboxLastSyncErrorKey     = "sandbox_last_sync_error"
	migrateWorkspaceSandboxProfileKey           = "sandbox_profile"
	migrateWorkspaceSandboxProviderStateJSONKey = "sandbox_provider_state_json"
	migrateWorkspaceSandboxRefKey               = "sandbox_ref"
	migrateWorkspaceSandboxStateKey             = "sandbox_state"
	migrateWorkspaceSpawnBudgetJSONKey          = "spawn_budget_json"
	migrateWorkspaceSpawnDepthKey               = "spawn_depth"
	migrateWorkspaceSpawnRoleKey                = "spawn_role"
	migrateWorkspaceStallReasonKey              = "stall_reason"
	migrateWorkspaceStallStateKey               = "stall_state"
	migrateWorkspaceStopDetailKey               = "stop_detail"
	migrateWorkspaceStopReasonKey               = "stop_reason"
	migrateWorkspaceSubprocessPidKey            = "subprocess_pid"
	migrateWorkspaceSubprocessStartedAtKey      = "subprocess_started_at"
	migrateWorkspaceTTLExpiresAtKey             = "ttl_expires_at"
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
	fresh, err := globalDatabaseIsEmpty(ctx, db)
	if err != nil {
		return err
	}
	if fresh {
		return bootstrapFreshGlobalSchema(ctx, db)
	}

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

func globalDatabaseIsEmpty(ctx context.Context, db *sql.DB) (bool, error) {
	var name string
	err := db.QueryRowContext(
		ctx,
		`SELECT name
		 FROM sqlite_master
		 WHERE type IN ('table', 'index', 'view', 'trigger')
		   AND name NOT LIKE 'sqlite_%'
		 LIMIT 1`,
	).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("store: inspect global schema objects: %w", err)
	}
	return false, nil
}

func bootstrapFreshGlobalSchema(ctx context.Context, db *sql.DB) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin fresh global schema bootstrap: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, fmt.Errorf("store: rollback fresh global schema bootstrap: %w", rollbackErr))
		}
	}()

	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		name       TEXT NOT NULL,
		checksum   TEXT NOT NULL,
		applied_at TEXT NOT NULL
	);`); err != nil {
		return fmt.Errorf("store: bootstrap schema_migrations table: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_schema_migrations_name ON schema_migrations(name);`,
	); err != nil {
		return fmt.Errorf("store: bootstrap schema_migrations name index: %w", err)
	}

	appliedAt := store.FormatTimestamp(time.Now().UTC())
	for _, migration := range globalSchemaMigrations {
		name := strings.TrimSpace(migration.Name)
		if migration.Up != nil {
			if err := migration.Up(ctx, tx); err != nil {
				return fmt.Errorf("store: bootstrap fresh migration %d %q: %w", migration.Version, name, err)
			}
		} else {
			for _, statement := range migration.Statements {
				if _, err := tx.ExecContext(ctx, statement); err != nil {
					return fmt.Errorf("store: bootstrap fresh migration %d %q: %w", migration.Version, name, err)
				}
			}
		}

		checksum, err := store.MigrationChecksum(migration)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (version, name, checksum, applied_at) VALUES (?, ?, ?, ?)`,
			migration.Version,
			name,
			checksum,
			appliedAt,
		); err != nil {
			return fmt.Errorf("store: record fresh global schema migration %d: %w", migration.Version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit fresh global schema bootstrap: %w", err)
	}
	return nil
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
	if _, ok := columns[migrateWorkspaceSandboxRefKey]; ok {
		return nil
	}

	if _, err := db.ExecContext(
		ctx,
		`ALTER TABLE workspaces ADD COLUMN sandbox_ref TEXT NOT NULL DEFAULT ''`,
	); err != nil {
		return fmt.Errorf("store: add workspaces.sandbox_ref column: %w", err)
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
	priorityExpr              string
	maxAttemptsExpr           string
	approvalPolicyExpr        string
	approvalStateExpr         string
	currentRunIDExpr          string
	pausedExpr                string
	pausedByExpr              string
	pausedAtExpr              string
	pausedReasonExpr          string
	maxRuntimeSecondsExpr     string
	spawnFailureCountExpr     string
	lastSpawnErrorExpr        string
	reviewPolicyExpr          string
	reviewMaxRoundsExpr       string
	reviewRoundExpr           string
	lastReviewIDExpr          string
	lastReviewOutcomeExpr     string
	reviewCircuitOpenedAtExpr string
	reviewCircuitReasonExpr   string
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
		priorityExpr:              existingTaskColumnExpr(columns, migrateWorkspacePriorityKey, `'medium'`),
		maxAttemptsExpr:           existingTaskColumnExpr(columns, "max_attempts", "3"),
		approvalPolicyExpr:        existingTaskColumnExpr(columns, "approval_policy", `'none'`),
		approvalStateExpr:         existingTaskColumnExpr(columns, "approval_state", `'not_required'`),
		currentRunIDExpr:          existingTaskColumnExpr(columns, "current_run_id", "NULL"),
		pausedExpr:                existingTaskColumnExpr(columns, "paused", "0"),
		pausedByExpr:              existingTaskColumnExpr(columns, "paused_by", `''`),
		pausedAtExpr:              existingTaskColumnExpr(columns, "paused_at", "NULL"),
		pausedReasonExpr:          existingTaskColumnExpr(columns, "paused_reason", `''`),
		maxRuntimeSecondsExpr:     existingTaskColumnExpr(columns, "max_runtime_seconds", "0"),
		spawnFailureCountExpr:     existingTaskColumnExpr(columns, "spawn_failure_count", "0"),
		lastSpawnErrorExpr:        existingTaskColumnExpr(columns, "last_spawn_error", `''`),
		reviewPolicyExpr:          existingTaskColumnExpr(columns, "review_policy", `'none'`),
		reviewMaxRoundsExpr:       existingTaskColumnExpr(columns, "review_max_rounds", "3"),
		reviewRoundExpr:           existingTaskColumnExpr(columns, "review_round", "0"),
		lastReviewIDExpr:          existingTaskColumnExpr(columns, "last_review_id", "NULL"),
		lastReviewOutcomeExpr:     existingTaskColumnExpr(columns, "last_review_outcome", "NULL"),
		reviewCircuitOpenedAtExpr: existingTaskColumnExpr(columns, "review_circuit_opened_at", "NULL"),
		reviewCircuitReasonExpr:   existingTaskColumnExpr(columns, "review_circuit_reason", "NULL"),
	}, true, nil
}

func taskTableNeedsRebuild(columns map[string]struct{}, tableSQL string) bool {
	if !strings.Contains(tableSQL, "'draft'") {
		return true
	}
	for _, column := range []string{migrateWorkspacePriorityKey, "max_attempts", "approval_policy", "approval_state"} {
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

const taskTableCreateSQL = `CREATE TABLE tasks_new (
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
		);`

func taskTableCreateStatement() string {
	return taskTableCreateSQL
}

func taskTableCopyStatement(spec taskTableMigrationSpec) string {
	return fmt.Sprintf(
		`INSERT INTO tasks_new (
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			priority, max_attempts, status, approval_policy, approval_state,
			owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
			created_at, updated_at, closed_at, metadata_json, current_run_id,
			paused, paused_by, paused_at, paused_reason, max_runtime_seconds,
			spawn_failure_count, last_spawn_error, review_policy, review_max_rounds, review_round,
			last_review_id, last_review_outcome, review_circuit_opened_at, review_circuit_reason
		) SELECT
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			%s, %s, status, %s, %s,
			owner_kind, owner_ref, created_by_kind, created_by_ref, origin_kind, origin_ref,
			created_at, updated_at, closed_at, metadata_json, %s,
			%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
		FROM tasks`,
		spec.priorityExpr,
		spec.maxAttemptsExpr,
		spec.approvalPolicyExpr,
		spec.approvalStateExpr,
		spec.currentRunIDExpr,
		spec.pausedExpr,
		spec.pausedByExpr,
		spec.pausedAtExpr,
		spec.pausedReasonExpr,
		spec.maxRuntimeSecondsExpr,
		spec.spawnFailureCountExpr,
		spec.lastSpawnErrorExpr,
		spec.reviewPolicyExpr,
		spec.reviewMaxRoundsExpr,
		spec.reviewRoundExpr,
		spec.lastReviewIDExpr,
		spec.lastReviewOutcomeExpr,
		spec.reviewCircuitOpenedAtExpr,
		spec.reviewCircuitReasonExpr,
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
			query: `INSERT INTO event_summaries_new (
					id, session_id, type, agent_name, content_json, task_id, run_id, workflow_id,
					claim_token_hash, lease_until, coordinator_session_id, scheduler_reason, hook_event,
					hook_name, actor_kind, actor_id, release_reason, parent_session_id, root_session_id,
					spawn_depth, summary, timestamp
				) SELECT
					id, session_id, type, agent_name, '' AS content_json, '' AS task_id, '' AS run_id,
					'' AS workflow_id, '' AS claim_token_hash, '' AS lease_until,
					'' AS coordinator_session_id, '' AS scheduler_reason, '' AS hook_event,
					'' AS hook_name, '' AS actor_kind, '' AS actor_id, '' AS release_reason,
					'' AS parent_session_id, '' AS root_session_id, 0 AS spawn_depth, summary, timestamp
				FROM event_summaries`,
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
	if _, ok := columns[globalDBBridgeNotificationSuppressColumn]; !ok {
		if _, err := db.ExecContext(
			ctx,
			`ALTER TABLE bridge_instances ADD COLUMN notification_suppress BOOLEAN NOT NULL DEFAULT 0`,
		); err != nil {
			return fmt.Errorf("store: add bridge_instances.notification_suppress column: %w", err)
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
	specs := sessionCoreColumnSpecs()
	return append(specs, sessionSandboxColumnSpecs()...)
}

func sessionCoreColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{name: migrateWorkspaceProviderKey, sql: `ALTER TABLE sessions ADD COLUMN provider TEXT NOT NULL DEFAULT ''`},
		{name: migrateWorkspaceStopReasonKey, sql: `ALTER TABLE sessions ADD COLUMN stop_reason TEXT`},
		{name: migrateWorkspaceStopDetailKey, sql: `ALTER TABLE sessions ADD COLUMN stop_detail TEXT`},
		{name: migrateWorkspaceFailureKindKey, sql: `ALTER TABLE sessions ADD COLUMN failure_kind TEXT`},
		{
			name: migrateWorkspaceFailureSummaryKey,
			sql:  `ALTER TABLE sessions ADD COLUMN failure_summary TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: migrateWorkspaceCrashBundlePathKey,
			sql:  `ALTER TABLE sessions ADD COLUMN crash_bundle_path TEXT NOT NULL DEFAULT ''`,
		},
		{name: migrateWorkspaceChannelKey, sql: `ALTER TABLE sessions ADD COLUMN channel TEXT NOT NULL DEFAULT ''`},
		{
			name: migrateWorkspaceSubprocessPidKey,
			sql:  `ALTER TABLE sessions ADD COLUMN subprocess_pid INTEGER NOT NULL DEFAULT 0`,
		},
		{
			name: migrateWorkspaceSubprocessStartedAtKey,
			sql:  `ALTER TABLE sessions ADD COLUMN subprocess_started_at TEXT`,
		},
		{name: migrateWorkspaceLastUpdateAtKey, sql: `ALTER TABLE sessions ADD COLUMN last_update_at TEXT`},
		{
			name: migrateWorkspaceStallStateKey,
			sql:  `ALTER TABLE sessions ADD COLUMN stall_state TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: migrateWorkspaceStallReasonKey,
			sql:  `ALTER TABLE sessions ADD COLUMN stall_reason TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: migrateWorkspaceActivityJSONKey,
			sql:  `ALTER TABLE sessions ADD COLUMN activity_json TEXT NOT NULL DEFAULT ''`,
		},
	}
}

func sessionSandboxColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{
			name: migrateWorkspaceSandboxIDKey,
			sql:  `ALTER TABLE sessions ADD COLUMN sandbox_id TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: migrateWorkspaceSandboxBackendKey,
			sql:  `ALTER TABLE sessions ADD COLUMN sandbox_backend TEXT NOT NULL DEFAULT 'local'`,
		},
		{
			name: migrateWorkspaceSandboxProfileKey,
			sql:  `ALTER TABLE sessions ADD COLUMN sandbox_profile TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: migrateWorkspaceSandboxInstanceIDKey,
			sql:  `ALTER TABLE sessions ADD COLUMN sandbox_instance_id TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: migrateWorkspaceSandboxStateKey,
			sql:  `ALTER TABLE sessions ADD COLUMN sandbox_state TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: migrateWorkspaceSandboxProviderStateJSONKey,
			sql:  `ALTER TABLE sessions ADD COLUMN sandbox_provider_state_json TEXT NOT NULL DEFAULT ''`,
		},
		{name: migrateWorkspaceSandboxLastSyncAtKey, sql: `ALTER TABLE sessions ADD COLUMN sandbox_last_sync_at TEXT`},
		{
			name: migrateWorkspaceSandboxLastSyncErrorKey,
			sql:  `ALTER TABLE sessions ADD COLUMN sandbox_last_sync_error TEXT NOT NULL DEFAULT ''`,
		},
		{name: migrateWorkspaceParentSessionIDKey, sql: `ALTER TABLE sessions ADD COLUMN parent_session_id TEXT`},
		{name: migrateWorkspaceRootSessionIDKey, sql: `ALTER TABLE sessions ADD COLUMN root_session_id TEXT`},
		{
			name: migrateWorkspaceSpawnDepthKey,
			sql:  `ALTER TABLE sessions ADD COLUMN spawn_depth INTEGER NOT NULL DEFAULT 0`,
		},
		{name: migrateWorkspaceSpawnRoleKey, sql: `ALTER TABLE sessions ADD COLUMN spawn_role TEXT`},
		{name: migrateWorkspaceTTLExpiresAtKey, sql: `ALTER TABLE sessions ADD COLUMN ttl_expires_at TEXT`},
		{
			name: migrateWorkspaceAutoStopOnParentKey,
			sql:  `ALTER TABLE sessions ADD COLUMN auto_stop_on_parent BOOLEAN NOT NULL DEFAULT 0`,
		},
		{
			name: migrateWorkspaceSpawnBudgetJSONKey,
			sql:  `ALTER TABLE sessions ADD COLUMN spawn_budget_json TEXT NOT NULL DEFAULT '{}'`,
		},
		{
			name: migrateWorkspacePermissionPolicyJSONKey,
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

func migrateSandboxColumnNames(ctx context.Context, tx *sql.Tx) error {
	renames := map[string][]struct {
		oldName string
		newName string
	}{
		"workspaces": {
			{oldName: "environment_ref", newName: migrateWorkspaceSandboxRefKey},
		},
		"sessions": {
			{oldName: "environment_id", newName: migrateWorkspaceSandboxIDKey},
			{oldName: "environment_backend", newName: migrateWorkspaceSandboxBackendKey},
			{oldName: "environment_profile", newName: migrateWorkspaceSandboxProfileKey},
			{oldName: "environment_instance_id", newName: migrateWorkspaceSandboxInstanceIDKey},
			{oldName: "environment_state", newName: migrateWorkspaceSandboxStateKey},
			{oldName: "environment_provider_state_json", newName: migrateWorkspaceSandboxProviderStateJSONKey},
			{oldName: "environment_last_sync_at", newName: migrateWorkspaceSandboxLastSyncAtKey},
			{oldName: "environment_last_sync_error", newName: migrateWorkspaceSandboxLastSyncErrorKey},
		},
		"tool_processes": {
			{oldName: "environment_id", newName: migrateWorkspaceSandboxIDKey},
		},
	}
	for table, specs := range renames {
		exists, err := tableExists(ctx, tx, table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		columns, err := tableColumns(ctx, tx, table)
		if err != nil {
			return err
		}
		for _, spec := range specs {
			if _, ok := columns[spec.newName]; ok {
				continue
			}
			if _, ok := columns[spec.oldName]; !ok {
				continue
			}
			stmt := fmt.Sprintf(
				`ALTER TABLE %s RENAME COLUMN %s TO %s`,
				table,
				spec.oldName,
				spec.newName,
			)
			if _, err := tx.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("store: rename %s.%s to %s: %w", table, spec.oldName, spec.newName, err)
			}
		}
	}
	return nil
}

func sessionFailureColumnSpecs() []migrationColumnSpec {
	return []migrationColumnSpec{
		{name: migrateWorkspaceFailureKindKey, sql: `ALTER TABLE sessions ADD COLUMN failure_kind TEXT`},
		{
			name: migrateWorkspaceFailureSummaryKey,
			sql:  `ALTER TABLE sessions ADD COLUMN failure_summary TEXT NOT NULL DEFAULT ''`,
		},
		{
			name: migrateWorkspaceCrashBundlePathKey,
			sql:  `ALTER TABLE sessions ADD COLUMN crash_bundle_path TEXT NOT NULL DEFAULT ''`,
		},
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
	if _, ok := columns[migrateWorkspaceProviderKey]; ok {
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
			`INSERT INTO workspaces (id, root_dir, add_dirs, name, default_agent, sandbox_ref, created_at, updated_at)
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
	for _, stmt := range migratedGlobalTableStatements() {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("store: create migrated global table: %w", err)
		}
	}

	return nil
}

func migratedGlobalTableStatements() []string {
	return []string{
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
				sandbox_backend, sandbox_profile, created_at, updated_at
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
	err := exec.QueryRowContext(
		ctx,
		`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
		strings.TrimSpace(table),
	).Scan(&name)
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
