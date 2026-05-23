package globaldb

import (
	"database/sql"
	"path/filepath"
	"slices"
	"testing"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
)

func TestGlobalDBTaskOrchestrationProfileSchemaMigration(t *testing.T) {
	t.Parallel()

	t.Run("Should create task orchestration profile schema on fresh DB", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)

		assertTaskOrchestrationProfileSchema(t, globalDB.db)
	})

	t.Run("Should migrate previous task schema and preserve task rows", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)
		legacyDB := openPreviousTaskOrchestrationSchemaDB(t, dbPath)
		insertPreviousTaskOrchestrationMigrationRecords(t, legacyDB)
		insertPreviousTaskOrchestrationRows(t, legacyDB)
		if err := legacyDB.Close(); err != nil {
			t.Fatalf("legacyDB.Close() error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, dbPath)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if err := globalDB.Close(ctx); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
		})

		assertTaskOrchestrationProfileSchema(t, globalDB.db)
		assertPreviousTaskOrchestrationRowsPreserved(t, globalDB.db)
	})
}

func TestTaskOrchestrationProfileSchemaStatements(t *testing.T) {
	t.Parallel()

	t.Run("Should use shared profile DDL in fresh global schema", func(t *testing.T) {
		t.Parallel()

		for _, statement := range taskOrchestrationProfileSchemaStatements() {
			if !schemaStatementsContain(globalSchemaStatements, statement) {
				t.Fatalf("globalSchemaStatements missing shared profile statement %q", statement)
			}
		}
	})

	t.Run("Should expose current-run index through named schema primitive", func(t *testing.T) {
		t.Parallel()

		if !schemaStatementsContain(taskTableIndexStatements, taskCurrentRunIndexStatement) {
			t.Fatalf("taskTableIndexStatements missing %q", taskCurrentRunIndexStatement)
		}
		if taskOrchestrationProfileSchemaStatements()[0] != taskCurrentRunIndexStatement {
			t.Fatalf("profile schema first statement must create the current-run index")
		}
		if got := countSchemaStatements(globalSchemaStatements, taskCurrentRunIndexStatement); got != 1 {
			t.Fatalf("globalSchemaStatements contains current-run index %d times, want 1", got)
		}
	})
}

func schemaStatementsContain(statements []string, want string) bool {
	return slices.Contains(statements, want)
}

func countSchemaStatements(statements []string, want string) int {
	count := 0
	for _, statement := range statements {
		if statement == want {
			count++
		}
	}
	return count
}

func assertTaskOrchestrationProfileSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	assertTablesPresent(
		t,
		db,
		"task_execution_profiles",
		"task_profile_agents",
		"task_profile_channels",
		"task_profile_peers",
		"task_profile_capabilities",
	)
	assertTableColumns(t, db, "task_execution_profiles", []string{
		"task_id",
		"coordinator_mode",
		"coordinator_agent_name",
		"coordinator_provider",
		"coordinator_model",
		"coordinator_guidance",
		"worker_mode",
		"worker_agent_name",
		"worker_provider",
		"worker_model",
		"review_agent_name",
		"review_provider",
		"review_model",
		"sandbox_mode",
		"sandbox_ref",
		"created_at",
		"updated_at",
	})
	assertTableColumns(t, db, "task_profile_agents", []string{
		"task_id",
		"role",
		"preference",
		"agent_name",
	})
	assertTableColumns(t, db, "task_profile_channels", []string{
		"task_id",
		"role",
		"preference",
		"channel_id",
	})
	assertTableColumns(t, db, "task_profile_peers", []string{
		"task_id",
		"role",
		"preference",
		"peer_id",
	})
	assertTableColumns(t, db, "task_profile_capabilities", []string{
		"task_id",
		"role",
		"preference",
		"capability_id",
	})
	assertIndexesPresent(t, db, "task_execution_profiles", "task_execution_profiles_task_id_idx")
	assertIndexesPresent(t, db, "task_profile_agents", "task_profile_agents_lookup_idx")
	assertIndexesPresent(t, db, "task_profile_channels", "task_profile_channels_lookup_idx")
	assertIndexesPresent(t, db, "task_profile_peers", "task_profile_peers_lookup_idx")
	assertIndexesPresent(t, db, "task_profile_capabilities", "task_profile_capabilities_lookup_idx")
}

func openPreviousTaskOrchestrationSchemaDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()

	ctx := testutil.Context(t)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	for _, statement := range []string{
		`CREATE TABLE tasks (
			id              TEXT PRIMARY KEY,
			identifier      TEXT,
			scope           TEXT NOT NULL CHECK (scope IN ('global', 'workspace')),
			workspace_id    TEXT,
			parent_task_id  TEXT,
			network_channel TEXT,
			title           TEXT NOT NULL,
			description     TEXT,
			priority        TEXT NOT NULL DEFAULT 'medium',
			max_attempts    INTEGER NOT NULL DEFAULT 3,
			status          TEXT NOT NULL,
			approval_policy TEXT NOT NULL DEFAULT 'none',
			approval_state  TEXT NOT NULL DEFAULT 'not_required',
			owner_kind      TEXT,
			owner_ref       TEXT,
			created_by_kind TEXT NOT NULL,
			created_by_ref  TEXT NOT NULL,
			origin_kind     TEXT NOT NULL,
			origin_ref      TEXT NOT NULL,
			created_at      TEXT NOT NULL,
			updated_at      TEXT NOT NULL,
			closed_at       TEXT,
			metadata_json   TEXT
		);`,
		`CREATE TABLE task_runs (
			id              TEXT PRIMARY KEY,
			task_id         TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
			status          TEXT NOT NULL,
			attempt         INTEGER NOT NULL,
			claimed_by_kind TEXT,
			claimed_by_ref  TEXT,
			session_id      TEXT,
			origin_kind     TEXT NOT NULL,
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
			claim_token     TEXT,
			claim_token_hash TEXT,
			lease_until     TEXT,
			heartbeat_at    TEXT,
			coordination_channel_id TEXT
		);`,
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("ExecContext(previous schema) error = %v", err)
		}
	}
	if err := store.RunMigrations(ctx, db, nil); err != nil {
		t.Fatalf("RunMigrations(empty) error = %v", err)
	}
	return db
}

func insertPreviousTaskOrchestrationMigrationRecords(t *testing.T, legacyDB *sql.DB) {
	t.Helper()

	insertMigrationRecordsThroughVersion(t, legacyDB, 16)
}

func insertMigrationRecordsThroughVersion(t *testing.T, legacyDB *sql.DB, maxVersion int) {
	t.Helper()

	ctx := testutil.Context(t)
	seedPath := filepath.Join(t.TempDir(), "seed.db")
	seedDB, err := sql.Open("sqlite", seedPath)
	if err != nil {
		t.Fatalf("sql.Open(seed) error = %v", err)
	}
	t.Cleanup(func() {
		if err := seedDB.Close(); err != nil {
			t.Fatalf("seedDB.Close() error = %v", err)
		}
	})
	if maxVersion < 1 || maxVersion > len(globalSchemaMigrations) {
		t.Fatalf("maxVersion = %d, want 1..%d", maxVersion, len(globalSchemaMigrations))
	}
	previousMigrations := globalSchemaMigrations[:maxVersion]
	if err := store.RunMigrations(ctx, seedDB, previousMigrations); err != nil {
		t.Fatalf("RunMigrations(seed) error = %v", err)
	}
	records, err := store.AppliedMigrations(ctx, seedDB)
	if err != nil {
		t.Fatalf("AppliedMigrations(seed) error = %v", err)
	}
	for _, record := range records {
		if _, err := legacyDB.ExecContext(
			ctx,
			`INSERT INTO schema_migrations (version, name, checksum, applied_at) VALUES (?, ?, ?, ?)`,
			record.Version,
			record.Name,
			record.Checksum,
			store.FormatTimestamp(record.AppliedAt),
		); err != nil {
			t.Fatalf("insert schema_migrations(%d) error = %v", record.Version, err)
		}
	}
}

func insertPreviousTaskOrchestrationRows(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := testutil.Context(t)
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO tasks (
			id, identifier, scope, workspace_id, parent_task_id, network_channel, title, description,
			priority, max_attempts, status, approval_policy, approval_state, owner_kind, owner_ref,
			created_by_kind, created_by_ref, origin_kind, origin_ref, created_at, updated_at,
			closed_at, metadata_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"task-profile-migration",
		"profile-migration",
		string(taskpkg.ScopeGlobal),
		nil,
		nil,
		nil,
		"Profile migration task",
		"Preserve this task",
		string(taskpkg.PriorityMedium),
		3,
		string(taskpkg.TaskStatusReady),
		string(taskpkg.ApprovalPolicyNone),
		string(taskpkg.ApprovalStateNotRequired),
		nil,
		nil,
		string(taskpkg.ActorKindHuman),
		"user:alice",
		string(taskpkg.OriginKindCLI),
		"cli",
		"2026-05-05T12:00:00.000000000Z",
		"2026-05-05T12:00:00.000000000Z",
		nil,
		`{"preserved":true}`,
	); err != nil {
		t.Fatalf("insert previous task error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO task_runs (
			id, task_id, status, attempt, claimed_by_kind, claimed_by_ref, session_id,
			origin_kind, origin_ref, idempotency_key, network_channel, queued_at,
			claimed_at, started_at, ended_at, error, metadata_json, result_json,
			claim_token, claim_token_hash, lease_until, heartbeat_at, coordination_channel_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"run-profile-migration",
		"task-profile-migration",
		string(taskpkg.TaskRunStatusQueued),
		1,
		nil,
		nil,
		nil,
		string(taskpkg.OriginKindCLI),
		"cli",
		nil,
		nil,
		"2026-05-05T12:00:00.000000000Z",
		nil,
		nil,
		nil,
		nil,
		`{"run":true}`,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	); err != nil {
		t.Fatalf("insert previous task run error = %v", err)
	}
}

func assertPreviousTaskOrchestrationRowsPreserved(t *testing.T, db *sql.DB) {
	t.Helper()

	ctx := testutil.Context(t)
	var (
		currentRunID      sql.NullString
		maxRuntimeSeconds int
		spawnFailureCount int
		lastSpawnError    string
	)
	if err := db.QueryRowContext(
		ctx,
		`SELECT current_run_id, max_runtime_seconds, spawn_failure_count, last_spawn_error
		 FROM tasks
		 WHERE id = ?`,
		"task-profile-migration",
	).Scan(&currentRunID, &maxRuntimeSeconds, &spawnFailureCount, &lastSpawnError); err != nil {
		t.Fatalf("query migrated task defaults error = %v", err)
	}
	if currentRunID.Valid {
		t.Fatalf("current_run_id = %q, want NULL", currentRunID.String)
	}
	if maxRuntimeSeconds != 0 || spawnFailureCount != 0 || lastSpawnError != "" {
		t.Fatalf(
			"task orchestration defaults = max_runtime_seconds:%d spawn_failure_count:%d last_spawn_error:%q",
			maxRuntimeSeconds,
			spawnFailureCount,
			lastSpawnError,
		)
	}

	var (
		summary                 string
		claimedAgentName        string
		claimedPeerID           string
		terminalizedBySessionID string
		terminalizedByAgentName string
		terminalizedByPeerID    string
		terminalizedByActorKind string
		terminalizedByActorRef  string
	)
	if err := db.QueryRowContext(
		ctx,
		`SELECT summary, claimed_agent_name, claimed_peer_id, terminalized_by_session_id,
		        terminalized_by_agent_name, terminalized_by_peer_id, terminalized_by_actor_kind,
		        terminalized_by_actor_ref
		 FROM task_runs
		 WHERE id = ?`,
		"run-profile-migration",
	).Scan(
		&summary,
		&claimedAgentName,
		&claimedPeerID,
		&terminalizedBySessionID,
		&terminalizedByAgentName,
		&terminalizedByPeerID,
		&terminalizedByActorKind,
		&terminalizedByActorRef,
	); err != nil {
		t.Fatalf("query migrated task run defaults error = %v", err)
	}
	if summary != "" ||
		claimedAgentName != "" ||
		claimedPeerID != "" ||
		terminalizedBySessionID != "" ||
		terminalizedByAgentName != "" ||
		terminalizedByPeerID != "" ||
		terminalizedByActorKind != "" ||
		terminalizedByActorRef != "" {
		t.Fatalf(
			"task run orchestration defaults = %q/%q/%q/%q/%q/%q/%q/%q, want empty strings",
			summary,
			claimedAgentName,
			claimedPeerID,
			terminalizedBySessionID,
			terminalizedByAgentName,
			terminalizedByPeerID,
			terminalizedByActorKind,
			terminalizedByActorRef,
		)
	}
}
