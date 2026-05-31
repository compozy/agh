package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	taskpkg "github.com/compozy/agh/internal/task"
	"github.com/compozy/agh/internal/testutil"
	aghworkspace "github.com/compozy/agh/internal/workspace"
)

type SessionInfo = store.SessionInfo
type SessionStateUpdate = store.SessionStateUpdate
type SessionListQuery = store.SessionListQuery
type EventSummary = store.EventSummary
type EventSummaryQuery = store.EventSummaryQuery
type TokenStats = store.TokenStats
type TokenStatsUpdate = store.TokenStatsUpdate
type TokenStatsQuery = store.TokenStatsQuery
type PermissionLogEntry = store.PermissionLogEntry
type PermissionLogQuery = store.PermissionLogQuery

const GlobalDatabaseName = store.GlobalDatabaseName
const defaultSessionType = "user"
const sqliteDriverName = "sqlite"

var testGlobalDBCurrentSchemaSeedPath string

func formatTimestamp(value time.Time) string {
	return store.FormatTimestamp(value)
}

func sqliteDSN(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func openSQLiteDatabase(
	ctx context.Context,
	path string,
	initialize func(context.Context, *sql.DB) error,
) (*sql.DB, error) {
	return store.OpenSQLiteDatabase(ctx, path, initialize)
}

func SessionMetaFile(sessionDir string) string {
	return store.SessionMetaFile(sessionDir)
}

func ReadSessionMeta(path string) (store.SessionMeta, error) {
	return store.ReadSessionMeta(path)
}

func TestMain(m *testing.M) {
	code := runGlobalDBTests(m)
	os.Exit(code)
}

func runGlobalDBTests(m *testing.M) (code int) {
	dir, err := os.MkdirTemp("", "agh-globaldb-current-schema-*")
	if err != nil {
		reportTestMainError("MkdirTemp(globaldb seed) error = %v", err)
		return 1
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			reportTestMainError("RemoveAll(globaldb seed) error = %v", err)
			if code == 0 {
				code = 1
			}
		}
	}()

	ctx := context.Background()
	path := filepath.Join(dir, GlobalDatabaseName)
	globalDB, err := OpenGlobalDB(ctx, path)
	if err != nil {
		reportTestMainError("OpenGlobalDB(globaldb seed) error = %v", err)
		return 1
	}
	if err := globalDB.Close(ctx); err != nil {
		reportTestMainError("Close(globaldb seed) error = %v", err)
		return 1
	}

	testGlobalDBCurrentSchemaSeedPath = path
	return m.Run()
}

func reportTestMainError(format string, args ...any) {
	if _, err := fmt.Fprintf(os.Stderr, format+"\n", args...); err != nil {
		panic(err)
	}
}

func TestOpenGlobalDBCreatesSchemaAndEnablesWAL(t *testing.T) {
	t.Parallel()

	globalDB := openFreshTestGlobalDB(t)

	assertTablesPresent(
		t,
		globalDB.db,
		"schema_migrations",
		"workspaces",
		"sessions",
		"event_summaries",
		"memory_events",
		"token_stats",
		"permission_log",
		"extensions",
		"config_apply_records",
		"scheduler_pause",
	)
	assertTableHasColumns(t, globalDB.db, "event_summaries", []string{
		"provider",
		"outcome",
	})
	assertTableColumns(t, globalDB.db, "memory_events", []string{
		"id",
		"op",
		"scope",
		"agent_name",
		"agent_tier",
		"workspace_id",
		"session_id",
		"actor_kind",
		"decision_id",
		"target_id",
		"metadata",
		"ts_ms",
	})
	assertTableColumns(t, globalDB.db, "config_apply_records", []string{
		"id",
		"desired_config_hash",
		"active_config_hash",
		"generation",
		"actor",
		"diff_class",
		"status",
		"diagnostic_json",
		"created_at",
		"applied_at",
		"updated_at",
	})
	assertIndexesPresent(
		t,
		globalDB.db,
		"config_apply_records",
		"idx_config_apply_records_desired",
		"idx_config_apply_records_active",
		"idx_config_apply_records_generation",
		"idx_config_apply_records_actor",
		"idx_config_apply_records_status",
	)
	assertJournalModeWAL(t, globalDB.db)
	assertSynchronousNormal(t, globalDB.db)
}

func TestSweepObservabilityDeletesOnlyRowsOlderThanCutoff(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-retention")

	cutoff := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	old := cutoff.Add(-time.Nanosecond)
	boundary := cutoff
	fresh := cutoff.Add(time.Nanosecond)

	for _, event := range []EventSummary{
		{ID: "sum-old", SessionID: "sess-retention", Type: "agent_message", AgentName: "coder", Timestamp: old},
		{ID: "sum-boundary", SessionID: "sess-retention", Type: "agent_message", AgentName: "coder", Timestamp: boundary},
		{ID: "sum-fresh", SessionID: "sess-retention", Type: "agent_message", AgentName: "coder", Timestamp: fresh},
	} {
		if err := globalDB.WriteEventSummary(ctx, event); err != nil {
			t.Fatalf("WriteEventSummary(%q) error = %v", event.ID, err)
		}
	}

	for _, update := range []TokenStatsUpdate{
		{SessionID: "sess-retention", AgentName: "coder-old", Turns: 1, UpdatedAt: old},
		{SessionID: "sess-retention", AgentName: "coder-boundary", Turns: 1, UpdatedAt: boundary},
		{SessionID: "sess-retention", AgentName: "coder-fresh", Turns: 1, UpdatedAt: fresh},
	} {
		if err := globalDB.UpdateTokenStats(ctx, update); err != nil {
			t.Fatalf("UpdateTokenStats(%q) error = %v", update.AgentName, err)
		}
	}

	for _, entry := range []PermissionLogEntry{
		{
			ID: "perm-old", SessionID: "sess-retention", AgentName: "coder", Action: "fs/read",
			Resource: "old.md", Decision: "allow", PolicyUsed: "approve-all", Timestamp: old,
		},
		{
			ID: "perm-boundary", SessionID: "sess-retention", AgentName: "coder", Action: "fs/read",
			Resource: "boundary.md", Decision: "allow", PolicyUsed: "approve-all", Timestamp: boundary,
		},
		{
			ID: "perm-fresh", SessionID: "sess-retention", AgentName: "coder", Action: "fs/read",
			Resource: "fresh.md", Decision: "allow", PolicyUsed: "approve-all", Timestamp: fresh,
		},
	} {
		if err := globalDB.WritePermissionLog(ctx, entry); err != nil {
			t.Fatalf("WritePermissionLog(%q) error = %v", entry.ID, err)
		}
	}

	result, err := globalDB.SweepObservability(ctx, cutoff)
	if err != nil {
		t.Fatalf("SweepObservability() error = %v", err)
	}
	if result.DeletedEventSummaries != 1 || result.DeletedTokenStats != 1 || result.DeletedPermissionLogs != 1 {
		t.Fatalf("SweepObservability() = %#v, want one deleted row per observe table", result)
	}
	if !result.CutoffAt.Equal(cutoff) {
		t.Fatalf("SweepObservability().CutoffAt = %s, want %s", result.CutoffAt, cutoff)
	}

	assertEventSummaryIDs(t, globalDB, []string{"sum-boundary", "sum-fresh"})
	assertTokenStatAgents(t, globalDB, []string{"coder-boundary", "coder-fresh"})
	assertPermissionLogIDs(t, globalDB, []string{"perm-boundary", "perm-fresh"})
}

func TestOpenGlobalDBRecordsSchemaMigrationAndRepeatedBootIsIdempotent(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	path := filepath.Join(t.TempDir(), GlobalDatabaseName)
	first, err := OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB(first) error = %v", err)
	}
	firstRecords, err := store.AppliedMigrations(ctx, first.db)
	if err != nil {
		t.Fatalf("AppliedMigrations(first) error = %v", err)
	}
	if got, want := len(firstRecords), len(globalSchemaMigrations); got != want {
		t.Fatalf("len(firstRecords) = %d, want %d", got, want)
	}
	assertAppliedGlobalMigrationOrder(t, firstRecords)
	if err := first.Close(ctx); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	second, err := OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB(second) error = %v", err)
	}
	t.Cleanup(func() {
		if err := second.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	})
	secondRecords, err := store.AppliedMigrations(ctx, second.db)
	if err != nil {
		t.Fatalf("AppliedMigrations(second) error = %v", err)
	}
	if got, want := len(secondRecords), len(globalSchemaMigrations); got != want {
		t.Fatalf("len(secondRecords) = %d, want %d", got, want)
	}
	for i := range firstRecords {
		if !secondRecords[i].AppliedAt.Equal(firstRecords[i].AppliedAt) {
			t.Fatalf(
				"second record %d applied_at = %s, want unchanged %s",
				i,
				secondRecords[i].AppliedAt,
				firstRecords[i].AppliedAt,
			)
		}
	}
}

func TestOpenGlobalDBFailsOnSchemaMigrationIntegrityMismatch(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	path := filepath.Join(t.TempDir(), GlobalDatabaseName)
	globalDB, err := OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB(initial) error = %v", err)
	}
	if err := globalDB.Close(ctx); err != nil {
		t.Fatalf("Close(initial) error = %v", err)
	}

	db, err := store.OpenSQLiteDatabase(ctx, path, nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`UPDATE schema_migrations SET checksum = 'tampered' WHERE version = 1`,
	); err != nil {
		t.Fatalf("tamper schema_migrations error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close() error = %v", err)
	}

	_, err = OpenGlobalDB(ctx, path)
	if err == nil || !strings.Contains(err.Error(), "migration 1 integrity mismatch") {
		t.Fatalf("OpenGlobalDB(tampered) error = %v, want integrity mismatch", err)
	}
}

func TestGlobalSchemaMigrationsAreAppendOnlyContract(t *testing.T) {
	t.Parallel()

	t.Run("Should keep shipped migration prefix identities stable", func(t *testing.T) {
		t.Parallel()

		assertGlobalSchemaMigrationDefinitions(t, globalSchemaMigrations)
	})

	t.Run("Should apply shipped migration prefix on fresh DB", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		globalDB, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB() error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close() error = %v", closeErr)
			}
		})

		records, err := store.AppliedMigrations(ctx, globalDB.db)
		if err != nil {
			t.Fatalf("AppliedMigrations() error = %v", err)
		}
		if got, want := len(records), len(globalSchemaMigrations); got != want {
			t.Fatalf("len(records) = %d, want %d", got, want)
		}
		assertAppliedGlobalMigrationOrder(t, records)
	})

	t.Run("Should preserve shipped migration prefix across reopen", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		first, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(first) error = %v", err)
		}
		firstRecords, err := store.AppliedMigrations(ctx, first.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(first) error = %v", err)
		}
		assertAppliedGlobalMigrationOrder(t, firstRecords)
		if err := first.Close(ctx); err != nil {
			t.Fatalf("Close(first) error = %v", err)
		}

		second, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(second) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := second.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close(second) error = %v", closeErr)
			}
		})
		secondRecords, err := store.AppliedMigrations(ctx, second.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(second) error = %v", err)
		}
		if got, want := len(secondRecords), len(firstRecords); got != want {
			t.Fatalf("len(secondRecords) = %d, want %d", got, want)
		}
		assertAppliedGlobalMigrationOrder(t, secondRecords)
		for index := range expectedGlobalMigrationPrefix() {
			if !secondRecords[index].AppliedAt.Equal(firstRecords[index].AppliedAt) {
				t.Fatalf(
					"migration %d applied_at = %s, want unchanged %s",
					firstRecords[index].Version,
					secondRecords[index].AppliedAt,
					firstRecords[index].AppliedAt,
				)
			}
		}
	})

	t.Run("Should upgrade recorded shipped prefix by appending later migrations", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(prefix) error = %v", err)
		}
		prefix := expectedGlobalMigrationPrefix()
		if err := store.RunMigrations(ctx, db, globalSchemaMigrations[:len(prefix)]); err != nil {
			t.Fatalf("RunMigrations(prefix) error = %v", err)
		}
		prefixRecords, err := store.AppliedMigrations(ctx, db)
		if err != nil {
			t.Fatalf("AppliedMigrations(prefix) error = %v", err)
		}
		assertAppliedGlobalMigrationPrefix(t, prefixRecords, len(prefix))
		if err := db.Close(); err != nil {
			t.Fatalf("prefix db.Close() error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(upgrade) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close(upgrade) error = %v", closeErr)
			}
		})
		records, err := store.AppliedMigrations(ctx, globalDB.db)
		if err != nil {
			t.Fatalf("AppliedMigrations(upgrade) error = %v", err)
		}
		if got, want := len(records), len(globalSchemaMigrations); got != want {
			t.Fatalf("len(records) = %d, want %d", got, want)
		}
		assertAppliedGlobalMigrationOrder(t, records)
		for index, prefixRecord := range prefixRecords {
			if !records[index].AppliedAt.Equal(prefixRecord.AppliedAt) {
				t.Fatalf(
					"migration %d applied_at = %s, want unchanged %s",
					prefixRecord.Version,
					records[index].AppliedAt,
					prefixRecord.AppliedAt,
				)
			}
		}
	})
}

type expectedGlobalMigrationIdentity struct {
	version  int
	name     string
	checksum string
}

func expectedGlobalMigrationPrefix() []expectedGlobalMigrationIdentity {
	return []expectedGlobalMigrationIdentity{
		{
			version:  1,
			name:     "create_global_schema",
			checksum: "70e2c16c9d32e692891ab71d075ca823782626e7c9f6ffbbc88c5d662704e089",
		},
		{version: 2, name: "add_session_failure_diagnostics", checksum: "2026-04-24-add-session-failure-diagnostics"},
		{version: 3, name: "add_automation_scheduler_state", checksum: "2026-04-24-add-automation-scheduler-state"},
		{version: 4, name: "add_mcp_auth_tokens", checksum: "2026-04-25-add-mcp-auth-tokens"},
		{version: 5, name: "add_tool_process_records", checksum: "2026-04-24-add-tool-process-records"},
		{version: 6, name: "add_memory_operation_scope", checksum: "2026-04-25-add-memory-operation-scope"},
		{version: 7, name: "add_task_run_claim_lease_schema", checksum: "2026-04-26-add-task-run-claim-lease-schema"},
		{version: 8, name: "add_session_lineage_metadata", checksum: "2026-04-26-add-session-lineage-metadata"},
		{
			version:  9,
			name:     "rename_environment_columns_to_sandbox",
			checksum: "2026-04-28-rename-environment-columns-to-sandbox",
		},
		{version: 10, name: "add_vault_secrets", checksum: "2026-05-01-add-vault-secrets"},
		{version: 11, name: "unify_secret_refs", checksum: "2026-05-01-unify-secret-refs"},
		{version: 12, name: "add_agent_soul_snapshots", checksum: "2026-05-02-add-agent-soul-snapshots"},
		{version: 13, name: "add_agent_heartbeat_storage", checksum: "2026-05-02-add-agent-heartbeat-storage"},
		{version: 14, name: "add_event_summary_lineage", checksum: "2026-05-04-add-event-summary-lineage"},
		{
			version:  15,
			name:     "rebuild_event_summaries_for_global_payloads",
			checksum: "2026-05-04-rebuild-event-summaries-for-global-payloads",
		},
		{
			version:  16,
			name:     "rename_actor_ref_columns_to_actor_id",
			checksum: "2026-05-04-rename-actor-ref-columns-to-actor-id",
		},
		{
			version:  17,
			name:     "add_task_orchestration_profile_schema",
			checksum: "2026-05-05-add-task-orchestration-profile-schema",
		},
		{version: 18, name: "add_task_review_gate_schema", checksum: "2026-05-05-add-task-review-gate-schema"},
		{version: 19, name: "add_notification_cursors", checksum: "2026-05-05-add-notification-cursors"},
		{version: 20, name: "add_bridge_task_subscriptions", checksum: "2026-05-05-add-bridge-task-subscriptions"},
		{
			version:  21,
			name:     "rebuild_network_conversation_containers",
			checksum: "2026-05-05-rebuild-network-conversation-containers",
		},
		{version: 22, name: "memv2_memory_events", checksum: "2026-05-05-memv2-memory-events"},
		{version: 23, name: "add_model_catalog_persistence", checksum: "2026-05-07-add-model-catalog-persistence"},
		{
			version:  24,
			name:     "rebuild_model_catalog_source_constraints",
			checksum: "2026-05-07-rebuild-model-catalog-source-constraints",
		},
		{
			version:  25,
			name:     "workspace_qualified_network_identity",
			checksum: "2026-05-12-workspace-qualified-network-identity",
		},
		{
			version:  26,
			name:     "add_network_timeline_extensions",
			checksum: "2026-05-16-add-network-timeline-extensions",
		},
		{
			version:  27,
			name:     "add_config_apply_records",
			checksum: "2026-05-20-add-config-apply-records",
		},
		{
			version:  28,
			name:     "add_event_summary_projections",
			checksum: "2026-05-20-add-event-summary-projections",
		},
		{
			version:  29,
			name:     "add_scheduler_pause_state",
			checksum: "2026-05-20-add-scheduler-pause-state",
		},
		{
			version:  30,
			name:     "add_session_attach_lock",
			checksum: "2026-05-20-add-session-attach-lock",
		},
		{
			version:  31,
			name:     "add_session_input_queue",
			checksum: "2026-05-21-add-session-input-queue",
		},
		{
			version:  32,
			name:     "add_task_run_force_ops",
			checksum: "2026-05-21-add-task-run-force-ops",
		},
		{
			version:  33,
			name:     "add_task_pause_state",
			checksum: "2026-05-21-add-task-pause-state",
		},
		{
			version:  34,
			name:     "add_extension_provenance",
			checksum: "2026-05-21-add-extension-provenance",
		},
		{
			version:  35,
			name:     "add_bridge_target_directory",
			checksum: "2026-05-21-add-bridge-target-directory",
		},
		{
			version:  36,
			name:     "add_notification_presets",
			checksum: "2026-05-21-add-notification-presets",
		},
		{
			version:  37,
			name:     "add_app_metadata",
			checksum: "2026-05-25-add-app-metadata",
		},
		{
			version:  38,
			name:     "heal_scheduler_pause_updated_at",
			checksum: "2026-05-28-heal-scheduler-pause-updated-at",
		},
		{
			version:  39,
			name:     "drop_task_run_status_check",
			checksum: "2026-05-28-drop-task-run-status-check",
		},
		{
			version:  40,
			name:     "add_task_run_starvation_tracking",
			checksum: "2026-05-28-add-task-run-starvation-tracking",
		},
		{
			version:  41,
			name:     "add_task_auto_enqueue",
			checksum: "2026-05-29-add-task-auto-enqueue",
		},
	}
}

func assertGlobalSchemaMigrationDefinitions(t *testing.T, migrations []store.Migration) {
	t.Helper()

	want := expectedGlobalMigrationPrefix()
	if got, wantLen := len(migrations), len(want); got != wantLen {
		t.Fatalf("globalSchemaMigrations length = %d, want exact shipped length %d", got, wantLen)
	}
	for index, expected := range want {
		got := migrations[index]
		if got.Version != expected.version || got.Name != expected.name || got.Checksum != expected.checksum {
			t.Fatalf(
				"globalSchemaMigrations[%d] = version %d name %q checksum %q, want version %d name %q checksum %q",
				index,
				got.Version,
				got.Name,
				got.Checksum,
				expected.version,
				expected.name,
				expected.checksum,
			)
		}
	}
}

func assertAppliedGlobalMigrationOrder(t *testing.T, records []store.MigrationRecord) {
	t.Helper()

	want := expectedGlobalMigrationPrefix()
	if got, wantLen := len(records), len(want); got != wantLen {
		t.Fatalf("schema_migrations length = %d, want exact shipped length %d", got, wantLen)
	}
	for index, expected := range want {
		got := records[index]
		if got.Version != expected.version || got.Name != expected.name || got.Checksum != expected.checksum {
			t.Fatalf(
				"schema_migrations[%d] = version %d name %q checksum %q, want version %d name %q checksum %q",
				index,
				got.Version,
				got.Name,
				got.Checksum,
				expected.version,
				expected.name,
				expected.checksum,
			)
		}
	}
}

func assertAppliedGlobalMigrationPrefix(t *testing.T, records []store.MigrationRecord, length int) {
	t.Helper()

	want := expectedGlobalMigrationPrefix()
	if length < 0 || length > len(want) {
		t.Fatalf("migration prefix length = %d, want 0..%d", length, len(want))
	}
	if got := len(records); got < length {
		t.Fatalf("schema_migrations length = %d, want at least prefix length %d", got, length)
	}
	for index := range records[:length] {
		expected := want[index]
		got := records[index]
		if got.Version != expected.version || got.Name != expected.name || got.Checksum != expected.checksum {
			t.Fatalf(
				"schema_migrations[%d] = version %d name %q checksum %q, want version %d name %q checksum %q",
				index,
				got.Version,
				got.Name,
				got.Checksum,
				expected.version,
				expected.name,
				expected.checksum,
			)
		}
	}
}

func TestOpenGlobalDBHealsSchedulerPauseUpdatedAt(t *testing.T) {
	t.Parallel()

	t.Run("Should heal the non-canonical updated_at left by the v29 seed", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase() error = %v", err)
		}
		healIdx := -1
		for index := range globalSchemaMigrations {
			if globalSchemaMigrations[index].Name == migrationNameHealSchedulerPause {
				healIdx = index
				break
			}
		}
		if healIdx < 0 {
			t.Fatal("heal_scheduler_pause_updated_at migration not found in registry")
		}
		preHeal := globalSchemaMigrations[:healIdx]
		if err := store.RunMigrations(ctx, db, preHeal); err != nil {
			t.Fatalf("RunMigrations(pre-heal) error = %v", err)
		}
		var seeded string
		if err := db.QueryRowContext(
			ctx,
			`SELECT updated_at FROM scheduler_pause WHERE id = 1`,
		).Scan(&seeded); err != nil {
			t.Fatalf("select seeded updated_at error = %v", err)
		}
		if _, parseErr := store.ParseTimestamp(seeded); parseErr == nil {
			t.Fatalf("expected non-canonical seed updated_at, got parseable %q", seeded)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("pre-heal db.Close() error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(upgrade) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close() error = %v", closeErr)
			}
		})

		var healed string
		if err := globalDB.db.QueryRowContext(
			ctx,
			`SELECT updated_at FROM scheduler_pause WHERE id = 1`,
		).Scan(&healed); err != nil {
			t.Fatalf("select healed updated_at error = %v", err)
		}
		if _, parseErr := store.ParseTimestamp(healed); parseErr != nil {
			t.Fatalf("healed updated_at %q is not canonical: %v", healed, parseErr)
		}
		if _, err := globalDB.GetSchedulerPause(ctx); err != nil {
			t.Fatalf("GetSchedulerPause() after heal error = %v", err)
		}
	})

	t.Run("Should heal idempotently and match FormatTimestamp", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE scheduler_pause SET updated_at = '2026-05-28 15:50:25' WHERE id = 1`,
		); err != nil {
			t.Fatalf("seed non-canonical updated_at error = %v", err)
		}
		if _, err := globalDB.db.ExecContext(ctx, healSchedulerPauseUpdatedAtSQL); err != nil {
			t.Fatalf("first heal error = %v", err)
		}
		first := schedulerPauseUpdatedAtFromDB(ctx, t, globalDB)
		if _, err := globalDB.db.ExecContext(ctx, healSchedulerPauseUpdatedAtSQL); err != nil {
			t.Fatalf("second heal error = %v", err)
		}
		second := schedulerPauseUpdatedAtFromDB(ctx, t, globalDB)
		if first != second {
			t.Fatalf("heal is not idempotent: first %q, second %q", first, second)
		}
		if want := "2026-05-28T15:50:25.000000000Z"; second != want {
			t.Fatalf("healed updated_at = %q, want %q", second, want)
		}
		if _, parseErr := store.ParseTimestamp(second); parseErr != nil {
			t.Fatalf("healed updated_at %q is not canonical: %v", second, parseErr)
		}
	})
}

func schedulerPauseUpdatedAtFromDB(ctx context.Context, t *testing.T, globalDB *GlobalDB) string {
	t.Helper()
	var value string
	if err := globalDB.db.QueryRowContext(
		ctx,
		`SELECT updated_at FROM scheduler_pause WHERE id = 1`,
	).Scan(&value); err != nil {
		t.Fatalf("select scheduler_pause updated_at error = %v", err)
	}
	return value
}

func TestOpenGlobalDBDropsTaskRunStatusCheck(t *testing.T) {
	t.Parallel()

	t.Run("Should accept needs_attention status on a freshly bootstrapped DB", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		seedQueuedRunForStatusTest(ctx, t, globalDB, "task-na-fresh", "run-na-fresh")
		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE task_runs SET status = 'needs_attention' WHERE id = ?`,
			"run-na-fresh",
		); err != nil {
			t.Fatalf("UPDATE to needs_attention on fresh DB error = %v, want nil (CHECK must be dropped)", err)
		}
		assertIndexesPresent(
			t, globalDB.db, "task_runs",
			"idx_task_runs_status", "idx_task_runs_pending_claim",
			"idx_task_runs_coordination_channel", "idx_task_runs_session_status",
		)
	})

	t.Run("Should accept needs_attention status on a DB migrated from v38", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase() error = %v", err)
		}
		preV39 := globalSchemaMigrations[:len(globalSchemaMigrations)-2]
		if err := store.RunMigrations(ctx, db, preV39); err != nil {
			t.Fatalf("RunMigrations(pre-v39) error = %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("pre-v39 db.Close() error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(upgrade) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close() error = %v", closeErr)
			}
		})
		seedQueuedRunForStatusTest(ctx, t, globalDB, "task-na-mig", "run-na-mig")
		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE task_runs SET status = 'needs_attention' WHERE id = ?`,
			"run-na-mig",
		); err != nil {
			t.Fatalf("UPDATE to needs_attention on migrated DB error = %v, want nil (CHECK must be dropped)", err)
		}
	})

	t.Run("Should preserve non-status constraints and foreign keys after the rebuild", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		seedQueuedRunForStatusTest(ctx, t, globalDB, "task-na-keep", "run-na-keep")
		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE task_runs SET attempt = 0 WHERE id = ?`,
			"run-na-keep",
		); err == nil {
			t.Fatal("UPDATE attempt=0 succeeded, want CHECK(attempt > 0) violation (rebuild must keep it)")
		}
		if _, err := globalDB.db.ExecContext(
			ctx,
			`UPDATE task_runs SET session_id = 'sess-x' WHERE id = ?`,
			"run-na-keep",
		); err == nil {
			t.Fatal("UPDATE session_id on a queued run succeeded, want compound CHECK violation (rebuild must keep it)")
		}
		var fkCount int
		if err := globalDB.db.QueryRowContext(
			ctx,
			`SELECT count(*) FROM pragma_foreign_key_list('task_runs')`,
		).Scan(&fkCount); err != nil {
			t.Fatalf("pragma_foreign_key_list error = %v", err)
		}
		if fkCount != 4 {
			t.Fatalf("task_runs foreign key count = %d, want 4 (rebuild must preserve all FKs)", fkCount)
		}
	})

	t.Run("Should keep the rebuilt task_runs columns aligned with the live schema", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		assertTableColumns(t, globalDB.db, "task_runs", taskRunColumnNamesForStatusMigration())
	})
}

func taskRunColumnNamesForStatusMigration() []string {
	parts := strings.Split(taskRunColumns, ",")
	columns := make([]string, 0, len(parts))
	for _, part := range parts {
		columns = append(columns, strings.TrimSpace(part))
	}
	return columns
}

func seedQueuedRunForStatusTest(ctx context.Context, t *testing.T, globalDB *GlobalDB, taskID, runID string) {
	t.Helper()
	task := taskRecordForTest(taskID)
	task.Status = taskpkg.TaskStatusReady
	if err := globalDB.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask(%q) error = %v", taskID, err)
	}
	run := taskRunForTest(runID, taskID)
	if err := globalDB.CreateTaskRun(ctx, run); err != nil {
		t.Fatalf("CreateTaskRun(%q) error = %v", runID, err)
	}
}

func TestOpenGlobalDBCreatesExtensionsTableWithExpectedColumns(t *testing.T) {
	t.Parallel()

	globalDB := openFreshTestGlobalDB(t)

	assertTableColumns(t, globalDB.db, "extensions", []string{
		"name",
		"version",
		"source",
		"enabled",
		"manifest_path",
		"installed_at",
		"capabilities",
		"actions",
		"checksum",
		"registry_slug",
		"registry_name",
		"remote_version",
		globalDBExtensionProvenanceJSONKey,
	})
}

func TestOpenGlobalDBExtensionsSchemaIsIdempotent(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), GlobalDatabaseName)
	first, err := OpenGlobalDB(testutil.Context(t), dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(first) error = %v", err)
	}
	if err := first.Close(testutil.Context(t)); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	second, err := OpenGlobalDB(testutil.Context(t), dbPath)
	if err != nil {
		t.Fatalf("OpenGlobalDB(second) error = %v", err)
	}
	t.Cleanup(func() {
		if err := second.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close(second) error = %v", err)
		}
	})

	assertTableColumns(t, second.db, "extensions", []string{
		"name",
		"version",
		"source",
		"enabled",
		"manifest_path",
		"installed_at",
		"capabilities",
		"actions",
		"checksum",
		"registry_slug",
		"registry_name",
		"remote_version",
		globalDBExtensionProvenanceJSONKey,
	})
}

func TestOpenGlobalDBMigratesLegacyExtensionsTableColumns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, GlobalDatabaseName)

	db, err := sql.Open(sqliteDriverName, sqliteDSN(path))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	ctx := testutil.Context(t)
	if _, err := db.ExecContext(ctx, `CREATE TABLE extensions (
		name          TEXT PRIMARY KEY,
		version       TEXT NOT NULL,
		source        TEXT NOT NULL,
		enabled       BOOLEAN NOT NULL DEFAULT 1,
		manifest_path TEXT NOT NULL,
		installed_at  TEXT NOT NULL,
		capabilities  TEXT NOT NULL DEFAULT '{}',
		actions       TEXT NOT NULL DEFAULT '{}',
		checksum      TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy extensions error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO extensions (name, version, source, enabled, manifest_path, installed_at, capabilities, actions, checksum) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"legacy-extension",
		"0.1.0",
		"user",
		true,
		"/tmp/legacy-extension/extension.toml",
		formatTimestamp(time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)),
		"{}",
		"{}",
		"abc123",
	); err != nil {
		t.Fatalf("insert legacy extension error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close(legacy db) error = %v", err)
	}

	globalDB, err := OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	assertTableColumns(t, globalDB.db, "extensions", []string{
		"name",
		"version",
		"source",
		"enabled",
		"manifest_path",
		"installed_at",
		"capabilities",
		"actions",
		"checksum",
		"registry_slug",
		"registry_name",
		"remote_version",
		globalDBExtensionProvenanceJSONKey,
	})

	var (
		version       string
		source        string
		enabled       bool
		registrySlug  sql.NullString
		registryName  sql.NullString
		remoteVersion sql.NullString
	)
	if err := globalDB.db.QueryRowContext(ctx, `
		SELECT version, source, enabled, registry_slug, registry_name, remote_version
		FROM extensions
		WHERE name = ?
	`, "legacy-extension").Scan(&version, &source, &enabled, &registrySlug, &registryName, &remoteVersion); err != nil {
		t.Fatalf("QueryRowContext(legacy extension) error = %v", err)
	}
	if version != "0.1.0" || source != "user" || !enabled {
		t.Fatalf("legacy extension row = version:%q source:%q enabled:%v", version, source, enabled)
	}
	if registrySlug.Valid || registryName.Valid || remoteVersion.Valid {
		t.Fatalf(
			"legacy extension provenance = (%v, %v, %v), want all NULL",
			registrySlug,
			registryName,
			remoteVersion,
		)
	}
}

func TestGlobalDBCheckReady(t *testing.T) {
	t.Parallel()

	var nilDB *GlobalDB
	if err := nilDB.checkReady(context.Background(), "list sessions"); err == nil {
		t.Fatal("checkReady(nil receiver) error = nil, want non-nil")
	}

	globalDB := openTestGlobalDB(t)
	nilContext := func() context.Context { return nil }
	if err := globalDB.checkReady(nilContext(), "list sessions"); err == nil {
		t.Fatal("checkReady(nil context) error = nil, want non-nil")
	}
	if err := globalDB.checkReady(testutil.Context(t), "list sessions"); err != nil {
		t.Fatalf("checkReady(valid) error = %v", err)
	}
	if err := globalDB.Close(testutil.Context(t)); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := globalDB.checkReady(testutil.Context(t), "list sessions"); !errors.Is(err, store.ErrClosed) {
		t.Fatalf("checkReady(after close) error = %v, want ErrClosed", err)
	}
}

func TestGlobalDBRegisterUpdateAndListSessions(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	createdAt := time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"sess-global-workspace",
		filepath.Join(t.TempDir(), "workspace-global"),
	)
	session := SessionInfo{
		ID:          "sess-global",
		Name:        "Alpha",
		AgentName:   "coder",
		Provider:    "claude",
		WorkspaceID: workspaceID,
		SessionType: "dream",
		State:       "active",
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}

	if err := globalDB.RegisterSession(testutil.Context(t), session); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	acpSessionID := "acp-123"
	if err := globalDB.UpdateSessionState(testutil.Context(t), SessionStateUpdate{
		ID:           session.ID,
		State:        "stopped",
		ACPSessionID: &acpSessionID,
		UpdatedAt:    createdAt.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("UpdateSessionState() error = %v", err)
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{State: "stopped"})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].State != "stopped" {
		t.Fatalf("sessions[0].State = %q, want stopped", sessions[0].State)
	}
	if sessions[0].SessionType != "dream" {
		t.Fatalf("sessions[0].SessionType = %q, want dream", sessions[0].SessionType)
	}
	if sessions[0].WorkspaceID != workspaceID {
		t.Fatalf("sessions[0].WorkspaceID = %q, want %q", sessions[0].WorkspaceID, workspaceID)
	}
	if sessions[0].Provider != "claude" {
		t.Fatalf("sessions[0].Provider = %q, want claude", sessions[0].Provider)
	}
	if sessions[0].ACPSessionID == nil || *sessions[0].ACPSessionID != "acp-123" {
		t.Fatalf("sessions[0].ACPSessionID = %#v, want acp-123", sessions[0].ACPSessionID)
	}
}

func TestGlobalDBRegisterSessionUpsertsProvider(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"provider-upsert-workspace",
		filepath.Join(t.TempDir(), "provider-upsert"),
	)
	createdAt := time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC)

	session := SessionInfo{
		ID:          "sess-provider-upsert",
		AgentName:   "coder",
		Provider:    "claude",
		WorkspaceID: workspaceID,
		State:       "active",
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
	if err := globalDB.RegisterSession(testutil.Context(t), session); err != nil {
		t.Fatalf("RegisterSession(initial) error = %v", err)
	}

	session.Provider = "codex"
	session.UpdatedAt = createdAt.Add(time.Minute)
	if err := globalDB.RegisterSession(testutil.Context(t), session); err != nil {
		t.Fatalf("RegisterSession(update) error = %v", err)
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if got, want := sessions[0].Provider, "codex"; got != want {
		t.Fatalf("sessions[0].Provider = %q, want %q", got, want)
	}

	var provider string
	if err := globalDB.db.QueryRowContext(
		testutil.Context(t),
		`SELECT provider FROM sessions WHERE id = ?`,
		session.ID,
	).Scan(&provider); err != nil {
		t.Fatalf("QueryRowContext(provider) error = %v", err)
	}
	if got, want := provider, "codex"; got != want {
		t.Fatalf("stored provider = %q, want %q", got, want)
	}
}

func TestGlobalDBRegisterSessionPersistsStopFields(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name           string
		stopReason     store.StopReason
		stopDetail     string
		wantStopReason *string
		wantStopDetail *string
	}{
		{
			name: "empty stop reason stores nulls",
		},
		{
			name:           "valid stop reason stores values",
			stopReason:     store.StopTimeout,
			stopDetail:     "deadline exceeded",
			wantStopReason: stringPointerForTest(string(store.StopTimeout)),
			wantStopDetail: stringPointerForTest("deadline exceeded"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			globalDB := openTestGlobalDB(t)
			workspaceID := registerWorkspaceForGlobalTests(
				t,
				globalDB,
				"persist-stop-workspace-"+strings.ReplaceAll(tc.name, " ", "-"),
				filepath.Join(t.TempDir(), "workspace"),
			)
			session := SessionInfo{
				ID:          "sess-" + strings.ReplaceAll(tc.name, " ", "-"),
				AgentName:   "coder",
				WorkspaceID: workspaceID,
				State:       "stopped",
				StopReason:  tc.stopReason,
				StopDetail:  tc.stopDetail,
				CreatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
			}

			if err := globalDB.RegisterSession(testutil.Context(t), session); err != nil {
				t.Fatalf("RegisterSession() error = %v", err)
			}

			gotStopReason, gotStopDetail := queryStoredSessionStopFields(t, globalDB.db, session.ID)
			assertOptionalStringEqual(t, gotStopReason, tc.wantStopReason, "stop_reason")
			assertOptionalStringEqual(t, gotStopDetail, tc.wantStopDetail, "stop_detail")

			sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{})
			if err != nil {
				t.Fatalf("ListSessions() error = %v", err)
			}
			if got, want := len(sessions), 1; got != want {
				t.Fatalf("len(sessions) = %d, want %d", got, want)
			}
			if got, want := sessions[0].StopReason, tc.stopReason; got != want {
				t.Fatalf("sessions[0].StopReason = %q, want %q", got, want)
			}
			if got, want := sessions[0].StopDetail, tc.stopDetail; got != want {
				t.Fatalf("sessions[0].StopDetail = %q, want %q", got, want)
			}
		})
	}
}

func TestGlobalDBRegisterSessionDefaultsTypeToUser(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"sess-default-type-workspace",
		filepath.Join(t.TempDir(), "workspace-default-type"),
	)
	session := SessionInfo{
		ID:          "sess-default-type",
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		State:       "active",
		CreatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
	}

	if err := globalDB.RegisterSession(testutil.Context(t), session); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if got, want := sessions[0].SessionType, defaultSessionType; got != want {
		t.Fatalf("sessions[0].SessionType = %q, want %q", got, want)
	}
}

func TestGlobalDBTaskEventSequenceReads(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	ctx := testutil.Context(t)
	createdAt := time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)
	actor := taskpkg.ActorIdentity{Kind: taskpkg.ActorKindHuman, Ref: "user-1"}
	origin := taskpkg.Origin{Kind: taskpkg.OriginKindCLI, Ref: "agh task test"}

	if err := globalDB.CreateTask(ctx, taskpkg.Task{
		ID:             "task-seq",
		Scope:          taskpkg.ScopeGlobal,
		Title:          "Sequence task",
		Priority:       taskpkg.DefaultPriority,
		MaxAttempts:    taskpkg.DefaultTaskMaxAttempts,
		Status:         taskpkg.TaskStatusReady,
		ApprovalPolicy: taskpkg.ApprovalPolicyNone,
		ApprovalState:  taskpkg.ApprovalStateNotRequired,
		CreatedBy:      actor,
		Origin:         origin,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}); err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}
	if err := globalDB.CreateTaskRun(ctx, taskpkg.Run{
		ID:        "run-seq",
		TaskID:    "task-seq",
		Status:    taskpkg.TaskRunStatusRunning,
		Attempt:   1,
		Origin:    origin,
		QueuedAt:  createdAt,
		StartedAt: createdAt.Add(time.Minute),
	}); err != nil {
		t.Fatalf("CreateTaskRun() error = %v", err)
	}
	emptyTask, err := globalDB.GetTask(ctx, "task-seq")
	if err != nil {
		t.Fatalf("GetTask(empty) error = %v", err)
	}
	if emptyTask.LatestEventSeq != 0 {
		t.Fatalf("LatestEventSeq before events = %d, want 0", emptyTask.LatestEventSeq)
	}

	for _, event := range []taskpkg.Event{
		{
			ID:        "evt-1",
			TaskID:    "task-seq",
			EventType: "task.created",
			Actor:     actor,
			Origin:    origin,
			Timestamp: createdAt,
		},
		{
			ID:        "evt-2",
			TaskID:    "task-seq",
			RunID:     "run-seq",
			EventType: "task.run_started",
			Actor:     actor,
			Origin:    origin,
			Timestamp: createdAt,
		},
		{
			ID:        "evt-3",
			TaskID:    "task-seq",
			EventType: "task.updated",
			Actor:     actor,
			Origin:    origin,
			Timestamp: createdAt,
		},
	} {
		if err := globalDB.CreateTaskEvent(ctx, event); err != nil {
			t.Fatalf("CreateTaskEvent(%q) error = %v", event.ID, err)
		}
	}

	record, err := globalDB.GetTaskEventRecord(ctx, "evt-2")
	if err != nil {
		t.Fatalf("GetTaskEventRecord() error = %v", err)
	}
	if got, want := record.Sequence, int64(2); got != want {
		t.Fatalf("record.Sequence = %d, want %d", got, want)
	}
	if got, want := record.Event.RunID, "run-seq"; got != want {
		t.Fatalf("record.Event.RunID = %q, want %q", got, want)
	}

	records, err := globalDB.ListTaskEventRecords(ctx, taskpkg.EventRecordQuery{
		TaskID:        "task-seq",
		AfterSequence: 1,
		Limit:         2,
	})
	if err != nil {
		t.Fatalf("ListTaskEventRecords() error = %v", err)
	}
	if got, want := len(records), 2; got != want {
		t.Fatalf("len(records) = %d, want %d", got, want)
	}
	if got, want := []string{
		records[0].Event.ID,
		records[1].Event.ID,
	}, []string{
		"evt-2",
		"evt-3",
	}; !testutil.EqualStringSlices(
		got,
		want,
	) {
		t.Fatalf("record ids = %#v, want %#v", got, want)
	}
	if got, want := []int64{
		records[0].Sequence,
		records[1].Sequence,
	}, []int64{
		2,
		3,
	}; got[0] != want[0] ||
		got[1] != want[1] {
		t.Fatalf("record sequences = %#v, want %#v", got, want)
	}

	taskRecord, err := globalDB.GetTask(ctx, "task-seq")
	if err != nil {
		t.Fatalf("GetTask(after events) error = %v", err)
	}
	if taskRecord.LatestEventSeq != 3 {
		t.Fatalf("LatestEventSeq after events = %d, want 3", taskRecord.LatestEventSeq)
	}
	summaries, err := globalDB.ListTasks(ctx, taskpkg.Query{})
	if err != nil {
		t.Fatalf("ListTasks() error = %v", err)
	}
	if len(summaries) != 1 || summaries[0].LatestEventSeq != 3 {
		t.Fatalf("ListTasks() = %#v, want latest_event_seq=3", summaries)
	}
	descRecords, err := globalDB.ListTaskEventRecords(ctx, taskpkg.EventRecordQuery{
		TaskID:     "task-seq",
		Limit:      2,
		Descending: true,
	})
	if err != nil {
		t.Fatalf("ListTaskEventRecords(descending) error = %v", err)
	}
	if len(descRecords) != 2 {
		t.Fatalf("len(descRecords) = %d, want 2", len(descRecords))
	}
	if got, want := []string{
		descRecords[0].Event.ID,
		descRecords[1].Event.ID,
	}, []string{
		"evt-3",
		"evt-2",
	}; !testutil.EqualStringSlices(
		got,
		want,
	) {
		t.Fatalf("descending record ids = %#v, want %#v", got, want)
	}
}

func TestGlobalDBWorkspaceCRUDAndLookups(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	rootParent := t.TempDir()
	rootDir := filepath.Join(rootParent, "workspace-root")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(rootDir) error = %v", err)
	}
	symlinkPath := filepath.Join(t.TempDir(), "workspace-link")
	if err := os.Symlink(rootDir, symlinkPath); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	canonicalRoot, err := filepath.EvalSymlinks(symlinkPath)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}

	createdAt := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	ws := aghworkspace.Workspace{
		ID:             "ws-primary",
		RootDir:        canonicalRoot,
		AdditionalDirs: []string{filepath.Join(rootDir, "a"), "", filepath.Join(rootDir, "b")},
		Name:           "alpha",
		DefaultAgent:   "coder",
		SandboxRef:     "daytona-dev",
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
	if err := globalDB.InsertWorkspace(testutil.Context(t), ws); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}

	byID, err := globalDB.GetWorkspace(testutil.Context(t), ws.ID)
	if err != nil {
		t.Fatalf("GetWorkspace() error = %v", err)
	}
	assertWorkspaceEqual(t, byID, aghworkspace.Workspace{
		ID:             ws.ID,
		RootDir:        canonicalRoot,
		AdditionalDirs: []string{filepath.Join(rootDir, "a"), filepath.Join(rootDir, "b")},
		Name:           "alpha",
		DefaultAgent:   "coder",
		SandboxRef:     "daytona-dev",
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	})

	byPath, err := globalDB.GetWorkspaceByPath(testutil.Context(t), canonicalRoot)
	if err != nil {
		t.Fatalf("GetWorkspaceByPath() error = %v", err)
	}
	assertWorkspaceEqual(t, byPath, byID)

	byName, err := globalDB.GetWorkspaceByName(testutil.Context(t), "alpha")
	if err != nil {
		t.Fatalf("GetWorkspaceByName() error = %v", err)
	}
	assertWorkspaceEqual(t, byName, byID)

	updated := byID
	updated.Name = "beta"
	updated.DefaultAgent = "reviewer"
	updated.SandboxRef = "local-dev"
	updated.AdditionalDirs = []string{filepath.Join(rootDir, "tools")}
	updated.UpdatedAt = createdAt.Add(5 * time.Minute)
	if err := globalDB.UpdateWorkspace(testutil.Context(t), updated); err != nil {
		t.Fatalf("UpdateWorkspace() error = %v", err)
	}

	gotUpdated, err := globalDB.GetWorkspace(testutil.Context(t), updated.ID)
	if err != nil {
		t.Fatalf("GetWorkspace(updated) error = %v", err)
	}
	assertWorkspaceEqual(t, gotUpdated, updated)

	if err := globalDB.DeleteWorkspace(testutil.Context(t), updated.ID); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v", err)
	}
	if _, err := globalDB.GetWorkspace(
		testutil.Context(t),
		updated.ID,
	); !errors.Is(
		err,
		aghworkspace.ErrWorkspaceNotFound,
	) {
		t.Fatalf("GetWorkspace(deleted) error = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestGlobalDBDeleteWorkspaceCascadeDeletesStoppedSessions(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"workspace-delete-guard",
		filepath.Join(t.TempDir(), "workspace-delete-guard"),
	)
	if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          "sess-delete-guard",
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		State:       "stopped",
		CreatedAt:   time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	if err := globalDB.DeleteWorkspace(testutil.Context(t), workspaceID); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v, want nil", err)
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("ListSessions() = %d sessions, want 0 (cascade delete)", len(sessions))
	}
}

func TestGlobalDBDeleteWorkspaceWithoutSessions(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"ws-no-sessions",
		filepath.Join(t.TempDir(), "ws-no-sessions"),
	)

	if err := globalDB.DeleteWorkspace(testutil.Context(t), workspaceID); err != nil {
		t.Fatalf("DeleteWorkspace() error = %v, want nil", err)
	}
}

func TestGlobalDBDeleteWorkspaceRejectsActiveSessions(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"ws-active-sessions",
		filepath.Join(t.TempDir(), "ws-active-sessions"),
	)
	if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          "sess-active-guard",
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		State:       "active",
		CreatedAt:   time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	err := globalDB.DeleteWorkspace(testutil.Context(t), workspaceID)
	if !errors.Is(err, aghworkspace.ErrWorkspaceHasActiveSessions) {
		t.Fatalf("DeleteWorkspace() error = %v, want ErrWorkspaceHasActiveSessions", err)
	}
}

func TestGlobalDBWorkspaceConstraintViolations(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	rootA := filepath.Join(t.TempDir(), "root-a")
	rootB := filepath.Join(t.TempDir(), "root-b")
	if err := os.MkdirAll(rootA, 0o755); err != nil {
		t.Fatalf("MkdirAll(rootA) error = %v", err)
	}
	if err := os.MkdirAll(rootB, 0o755); err != nil {
		t.Fatalf("MkdirAll(rootB) error = %v", err)
	}

	base := aghworkspace.Workspace{
		ID:        "ws-base",
		RootDir:   rootA,
		Name:      "alpha",
		CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
	}
	if err := globalDB.InsertWorkspace(testutil.Context(t), base); err != nil {
		t.Fatalf("InsertWorkspace(base) error = %v", err)
	}

	tests := []struct {
		name string
		ws   aghworkspace.Workspace
		want error
	}{
		{
			name: "duplicate root dir",
			ws: aghworkspace.Workspace{
				ID:        "ws-duplicate-root",
				RootDir:   rootA,
				Name:      "beta",
				CreatedAt: base.CreatedAt,
				UpdatedAt: base.UpdatedAt,
			},
			want: aghworkspace.ErrWorkspacePathTaken,
		},
		{
			name: "duplicate name",
			ws: aghworkspace.Workspace{
				ID:        "ws-duplicate-name",
				RootDir:   rootB,
				Name:      "alpha",
				CreatedAt: base.CreatedAt,
				UpdatedAt: base.UpdatedAt,
			},
			want: aghworkspace.ErrWorkspaceNameTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := globalDB.InsertWorkspace(testutil.Context(t), tt.ws)
			if !errors.Is(err, tt.want) {
				t.Fatalf("InsertWorkspace() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestGlobalDBWorkspaceNotFoundErrors(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	if _, err := globalDB.GetWorkspace(
		testutil.Context(t),
		"ws-missing",
	); !errors.Is(
		err,
		aghworkspace.ErrWorkspaceNotFound,
	) {
		t.Fatalf("GetWorkspace(missing) error = %v, want ErrWorkspaceNotFound", err)
	}
	if _, err := globalDB.GetWorkspaceByPath(
		testutil.Context(t),
		filepath.Join(t.TempDir(), "missing"),
	); !errors.Is(
		err,
		aghworkspace.ErrWorkspaceNotFound,
	) {
		t.Fatalf("GetWorkspaceByPath(missing) error = %v, want ErrWorkspaceNotFound", err)
	}
	if _, err := globalDB.GetWorkspaceByName(
		testutil.Context(t),
		"missing",
	); !errors.Is(
		err,
		aghworkspace.ErrWorkspaceNotFound,
	) {
		t.Fatalf("GetWorkspaceByName(missing) error = %v, want ErrWorkspaceNotFound", err)
	}
	if err := globalDB.UpdateWorkspace(testutil.Context(t), aghworkspace.Workspace{
		ID:        "ws-missing",
		RootDir:   filepath.Join(t.TempDir(), "missing"),
		Name:      "missing",
		UpdatedAt: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
	}); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("UpdateWorkspace(missing) error = %v, want ErrWorkspaceNotFound", err)
	}
	if err := globalDB.DeleteWorkspace(
		testutil.Context(t),
		"ws-missing",
	); !errors.Is(
		err,
		aghworkspace.ErrWorkspaceNotFound,
	) {
		t.Fatalf("DeleteWorkspace(missing) error = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestGlobalDBWorkspaceValidationAndDefaulting(t *testing.T) {
	t.Parallel()

	var nilCtx context.Context
	if _, err := OpenGlobalDB(nilCtx, filepath.Join(t.TempDir(), GlobalDatabaseName)); err == nil {
		t.Fatal("OpenGlobalDB(nil) error = nil, want non-nil")
	}

	var nilGlobalDB *GlobalDB
	if got := nilGlobalDB.Path(); got != "" {
		t.Fatalf("(*GlobalDB)(nil).Path() = %q, want empty", got)
	}

	globalDB := openTestGlobalDB(t)
	rootDir := filepath.Join(t.TempDir(), "workspace-defaulted")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if err := globalDB.InsertWorkspace(testutil.Context(t), aghworkspace.Workspace{
		RootDir: rootDir,
		Name:    "defaulted",
	}); err != nil {
		t.Fatalf("InsertWorkspace(defaulted) error = %v", err)
	}

	workspaces, err := globalDB.ListWorkspaces(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListWorkspaces() error = %v", err)
	}
	if got, want := len(workspaces), 1; got != want {
		t.Fatalf("len(workspaces) = %d, want %d", got, want)
	}
	if !aghworkspace.IsWorkspaceID(workspaces[0].ID) {
		t.Fatalf("workspaces[0].ID = %q, want workspace_id ULID", workspaces[0].ID)
	}
	if workspaces[0].CreatedAt.IsZero() || workspaces[0].UpdatedAt.IsZero() {
		t.Fatalf("workspace timestamps = %#v, want non-zero", workspaces[0])
	}

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "insert missing root",
			run: func() error {
				return globalDB.InsertWorkspace(testutil.Context(t), aghworkspace.Workspace{Name: "missing-root"})
			},
		},
		{
			name: "insert missing name",
			run: func() error {
				return globalDB.InsertWorkspace(testutil.Context(t), aghworkspace.Workspace{RootDir: rootDir})
			},
		},
		{
			name: "update missing id",
			run: func() error {
				return globalDB.UpdateWorkspace(
					testutil.Context(t),
					aghworkspace.Workspace{RootDir: rootDir, Name: "missing-id"},
				)
			},
		},
		{
			name: "delete missing id",
			run: func() error {
				return globalDB.DeleteWorkspace(testutil.Context(t), "")
			},
		},
		{
			name: "get missing id",
			run: func() error {
				_, err := globalDB.GetWorkspace(testutil.Context(t), "")
				return err
			},
		},
		{
			name: "get by missing path",
			run: func() error {
				_, err := globalDB.GetWorkspaceByPath(testutil.Context(t), "")
				return err
			},
		},
		{
			name: "get by missing name",
			run: func() error {
				_, err := globalDB.GetWorkspaceByName(testutil.Context(t), "")
				return err
			},
		},
		{
			name: "list nil context",
			run: func() error {
				var nilCtx context.Context
				_, err := globalDB.ListWorkspaces(nilCtx)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil {
				t.Fatal("error = nil, want non-nil")
			}
		})
	}
}

func TestGlobalDBNilReceiverWorkspaceMethods(t *testing.T) {
	t.Parallel()

	var nilGlobalDB *GlobalDB
	ctx := testutil.Context(t)

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "insert workspace",
			run:  func() error { return nilGlobalDB.InsertWorkspace(ctx, aghworkspace.Workspace{}) },
		},
		{
			name: "update workspace",
			run:  func() error { return nilGlobalDB.UpdateWorkspace(ctx, aghworkspace.Workspace{}) },
		},
		{
			name: "delete workspace",
			run:  func() error { return nilGlobalDB.DeleteWorkspace(ctx, "ws-1") },
		},
		{
			name: "get workspace",
			run: func() error {
				_, err := nilGlobalDB.GetWorkspace(ctx, "ws-1")
				return err
			},
		},
		{
			name: "get workspace by path",
			run: func() error {
				_, err := nilGlobalDB.GetWorkspaceByPath(ctx, "/tmp/workspace")
				return err
			},
		},
		{
			name: "get workspace by name",
			run: func() error {
				_, err := nilGlobalDB.GetWorkspaceByName(ctx, "workspace")
				return err
			},
		},
		{
			name: "list workspaces",
			run: func() error {
				_, err := nilGlobalDB.ListWorkspaces(ctx)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil {
				t.Fatal("error = nil, want non-nil")
			}
		})
	}
}

func TestGlobalDBListWorkspacesStableOrder(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	first := insertWorkspaceForGlobalTests(t, globalDB, aghworkspace.Workspace{
		ID:        "ws-zeta",
		RootDir:   filepath.Join(t.TempDir(), "workspace-zeta"),
		Name:      "zeta",
		CreatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
	})
	second := insertWorkspaceForGlobalTests(t, globalDB, aghworkspace.Workspace{
		ID:        "ws-alpha",
		RootDir:   filepath.Join(t.TempDir(), "workspace-alpha"),
		Name:      "alpha",
		CreatedAt: time.Date(2026, 4, 3, 10, 1, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 10, 1, 0, 0, time.UTC),
	})

	workspaces, err := globalDB.ListWorkspaces(testutil.Context(t))
	if err != nil {
		t.Fatalf("ListWorkspaces() error = %v", err)
	}

	if got, want := len(workspaces), 2; got != want {
		t.Fatalf("len(workspaces) = %d, want %d", got, want)
	}
	assertWorkspaceEqual(t, workspaces[0], second)
	assertWorkspaceEqual(t, workspaces[1], first)
}

func TestGlobalDBRegisterAndListSessionsUseWorkspaceID(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"session-workspace",
		filepath.Join(t.TempDir(), "session-workspace"),
	)

	session := SessionInfo{
		ID:          "sess-workspace-id",
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		Channel:     "builders",
		State:       "active",
		Liveness: &store.SessionLivenessMeta{
			SubprocessPID: 77,
			LastUpdateAt:  ptrTime(time.Date(2026, 4, 3, 13, 1, 0, 0, time.UTC)),
			StallState:    store.SessionStallStateDetected,
			StallReason:   store.SessionStallReasonActivityTimeout,
		},
		Sandbox: &store.SessionSandboxMeta{
			SandboxID:     "env-workspace-id",
			Backend:       "local",
			Profile:       "local",
			State:         "prepared",
			InstanceID:    "instance-workspace-id",
			ProviderState: []byte(`{"provider":true}`),
			LastSyncError: "last sync failed",
		},
		CreatedAt: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
	}
	if err := globalDB.RegisterSession(testutil.Context(t), session); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if got, want := sessions[0].WorkspaceID, workspaceID; got != want {
		t.Fatalf("sessions[0].WorkspaceID = %q, want %q", got, want)
	}
	if got, want := sessions[0].Channel, "builders"; got != want {
		t.Fatalf("sessions[0].Channel = %q, want %q", got, want)
	}
	if sessions[0].Sandbox == nil {
		t.Fatal("sessions[0].Sandbox = nil, want sandbox metadata")
	}
	if got, want := sessions[0].Sandbox.SandboxID, "env-workspace-id"; got != want {
		t.Fatalf("sessions[0].Sandbox.SandboxID = %q, want %q", got, want)
	}
	if got, want := sessions[0].Sandbox.InstanceID, "instance-workspace-id"; got != want {
		t.Fatalf("sessions[0].Sandbox.InstanceID = %q, want %q", got, want)
	}
	if got, want := sessions[0].Sandbox.LastSyncError, "last sync failed"; got != want {
		t.Fatalf("sessions[0].Sandbox.LastSyncError = %q, want %q", got, want)
	}
	if sessions[0].Liveness == nil {
		t.Fatal("sessions[0].Liveness = nil, want liveness metadata")
	}
	if got, want := sessions[0].Liveness.SubprocessPID, 77; got != want {
		t.Fatalf("sessions[0].Liveness.SubprocessPID = %d, want %d", got, want)
	}
	if sessions[0].Liveness.LastUpdateAt == nil ||
		!sessions[0].Liveness.LastUpdateAt.Equal(*session.Liveness.LastUpdateAt) {
		t.Fatalf(
			"sessions[0].Liveness.LastUpdateAt = %#v, want %s",
			sessions[0].Liveness.LastUpdateAt,
			session.Liveness.LastUpdateAt,
		)
	}
	if got, want := sessions[0].Liveness.StallState, store.SessionStallStateDetected; got != want {
		t.Fatalf("sessions[0].Liveness.StallState = %q, want %q", got, want)
	}
	if got, want := sessions[0].Liveness.StallReason, store.SessionStallReasonActivityTimeout; got != want {
		t.Fatalf("sessions[0].Liveness.StallReason = %q, want %q", got, want)
	}

	assertTableColumns(
		t,
		globalDB.db,
		"sessions",
		[]string{
			"id",
			"name",
			"agent_name",
			"provider",
			"workspace_id",
			"session_type",
			"channel",
			"state",
			"acp_session_id",
			"stop_reason",
			"stop_detail",
			"subprocess_pid",
			"subprocess_started_at",
			"last_update_at",
			"stall_state",
			"stall_reason",
			"activity_json",
			"attached_to",
			"attach_expires_at",
			"sandbox_id",
			"sandbox_backend",
			"sandbox_profile",
			"sandbox_instance_id",
			"sandbox_state",
			"sandbox_provider_state_json",
			"sandbox_last_sync_at",
			"sandbox_last_sync_error",
			"created_at",
			"updated_at",
			"failure_kind",
			"failure_summary",
			"crash_bundle_path",
			"parent_session_id",
			"root_session_id",
			"spawn_depth",
			"spawn_role",
			"ttl_expires_at",
			"auto_stop_on_parent",
			"spawn_budget_json",
			"permission_policy_json",
			"soul_snapshot_id",
			"soul_digest",
			"parent_soul_digest",
			sessionInputGenerationColumn,
		},
	)
}

func TestGlobalDBRegisterSessionRejectsStallStateWithoutReason(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"invalid-stall-session",
		filepath.Join(t.TempDir(), "invalid-stall-session"),
	)

	err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          "sess-invalid-stall",
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		State:       "active",
		Liveness: &store.SessionLivenessMeta{
			SubprocessPID: 77,
			StallState:    store.SessionStallStateDetected,
		},
		CreatedAt: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("RegisterSession() error = nil, want invalid stall reason failure")
	}
	if got, want := err.Error(), "store: session stall reason required when stall state is set"; !strings.Contains(
		got,
		want,
	) {
		t.Fatalf("RegisterSession() error = %v, want substring %q", err, want)
	}
}

func TestGlobalDBRegisterSessionRejectsUnmarshalableActivity(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"invalid-activity-session",
		filepath.Join(t.TempDir(), "invalid-activity-session"),
	)
	unmarshalableTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          "sess-invalid-activity",
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		State:       "active",
		Liveness: &store.SessionLivenessMeta{
			Activity: &store.SessionActivityMeta{
				TurnStartedAt: &unmarshalableTime,
			},
		},
		CreatedAt: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("RegisterSession(unmarshalable activity) error = nil, want marshal failure")
	}
	if !strings.Contains(err.Error(), "store: session liveness activity marshal") {
		t.Fatalf("RegisterSession(unmarshalable activity) error = %v, want activity marshal context", err)
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("len(sessions) = %d, want failed register to skip write", len(sessions))
	}
}

func TestOpenGlobalDBMigratesLegacyWorkspaceColumn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, GlobalDatabaseName)

	db, err := sql.Open(sqliteDriverName, sqliteDSN(path))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	ctx := testutil.Context(t)
	if _, err := db.ExecContext(ctx, `CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		name TEXT,
		agent_name TEXT NOT NULL,
		workspace TEXT NOT NULL,
		session_type TEXT NOT NULL DEFAULT 'user',
		state TEXT NOT NULL,
		acp_session_id TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy sessions error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE event_summaries (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		type TEXT NOT NULL,
		agent_name TEXT NOT NULL,
		summary TEXT,
		timestamp TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy event_summaries error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE token_stats (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		agent_name TEXT NOT NULL,
		input_tokens INTEGER,
		output_tokens INTEGER,
		total_tokens INTEGER,
		total_cost REAL,
		cost_currency TEXT,
		turn_count INTEGER NOT NULL DEFAULT 0,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy token_stats error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE permission_log (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id),
		agent_name TEXT NOT NULL,
		action TEXT NOT NULL,
		resource TEXT NOT NULL,
		decision TEXT NOT NULL,
		policy_used TEXT NOT NULL,
		timestamp TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy permission_log error = %v", err)
	}

	rootA := filepath.Join(dir, "apps", "project")
	rootB := filepath.Join(dir, "services", "project")
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"sess-legacy-a",
		"A",
		"coder",
		rootA,
		"user",
		"active",
		formatTimestamp(time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)),
		formatTimestamp(time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert legacy session A error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"sess-legacy-b",
		"B",
		"reviewer",
		rootB,
		"dream",
		"stopped",
		formatTimestamp(time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)),
		formatTimestamp(time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert legacy session B error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO event_summaries (id, session_id, type, agent_name, summary, timestamp) VALUES (?, ?, ?, ?, ?, ?)`,
		"sum-legacy",
		"sess-legacy-a",
		"agent_message",
		"coder",
		"legacy summary",
		formatTimestamp(time.Date(2026, 4, 3, 10, 1, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert legacy event summary error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close(legacy db) error = %v", err)
	}

	globalDB, err := OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	assertTableColumns(
		t,
		globalDB.db,
		"sessions",
		[]string{
			"id",
			"name",
			"agent_name",
			"provider",
			"workspace_id",
			"session_type",
			"channel",
			"state",
			"acp_session_id",
			"stop_reason",
			"stop_detail",
			"failure_kind",
			"failure_summary",
			"crash_bundle_path",
			"sandbox_id",
			"sandbox_backend",
			"sandbox_profile",
			"sandbox_instance_id",
			"sandbox_state",
			"sandbox_provider_state_json",
			"sandbox_last_sync_at",
			"sandbox_last_sync_error",
			"created_at",
			"updated_at",
			"subprocess_pid",
			"subprocess_started_at",
			"last_update_at",
			"stall_state",
			"stall_reason",
			"activity_json",
			"parent_session_id",
			"root_session_id",
			"spawn_depth",
			"spawn_role",
			"ttl_expires_at",
			"auto_stop_on_parent",
			"spawn_budget_json",
			"permission_policy_json",
			"soul_snapshot_id",
			"soul_digest",
			"parent_soul_digest",
			"attached_to",
			"attach_expires_at",
			sessionInputGenerationColumn,
		},
	)
	assertTableColumns(
		t,
		globalDB.db,
		"event_summaries",
		[]string{
			"id",
			"session_id",
			"workspace_id",
			"type",
			"agent_name",
			"content_json",
			"task_id",
			"run_id",
			"workflow_id",
			"claim_token_hash",
			"lease_until",
			"coordinator_session_id",
			"scheduler_reason",
			"hook_event",
			"hook_name",
			"actor_kind",
			"actor_id",
			"release_reason",
			"parent_session_id",
			"root_session_id",
			"spawn_depth",
			"summary",
			"timestamp",
			"provider",
			"outcome",
		},
	)
	assertTableColumns(
		t,
		globalDB.db,
		"workspaces",
		[]string{"id", "root_dir", "add_dirs", "name", "default_agent", "sandbox_ref", "created_at", "updated_at"},
	)

	workspaces, err := globalDB.ListWorkspaces(ctx)
	if err != nil {
		t.Fatalf("ListWorkspaces() error = %v", err)
	}
	if got, want := len(workspaces), 2; got != want {
		t.Fatalf("len(workspaces) = %d, want %d", got, want)
	}
	if got, want := []string{
		workspaces[0].Name,
		workspaces[1].Name,
	}, []string{
		"project",
		"project-2",
	}; !testutil.EqualStringSlices(
		got,
		want,
	) {
		t.Fatalf("workspace names = %#v, want %#v", got, want)
	}

	sessions, err := globalDB.ListSessions(ctx, SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 2; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	for _, session := range sessions {
		if strings.HasPrefix(session.WorkspaceID, "/") {
			t.Fatalf("session.WorkspaceID = %q, want migrated ws_ id", session.WorkspaceID)
		}
		if session.Channel != "" {
			t.Fatalf("session.Channel = %q, want empty for migrated legacy rows", session.Channel)
		}
	}

	summaries, err := globalDB.ListEventSummaries(ctx, EventSummaryQuery{SessionID: "sess-legacy-a"})
	if err != nil {
		t.Fatalf("ListEventSummaries() error = %v", err)
	}
	if got, want := len(summaries), 0; got != want {
		t.Fatalf("len(summaries) = %d, want %d", got, want)
	}
}

func TestOpenGlobalDBRewritesLegacySessionMetaWorkspaceID(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	homeDir := t.TempDir()
	path := filepath.Join(homeDir, GlobalDatabaseName)

	db, err := openSQLiteDatabase(ctx, path, nil)
	if err != nil {
		t.Fatalf("openSQLiteDatabase() error = %v", err)
	}

	rootDir := filepath.Join(homeDir, "workspace")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(rootDir) error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		name TEXT,
		agent_name TEXT NOT NULL,
		workspace TEXT NOT NULL,
		session_type TEXT NOT NULL DEFAULT 'user',
		state TEXT NOT NULL,
		acp_session_id TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy sessions error = %v", err)
	}
	createdAt := formatTimestamp(time.Date(2026, 4, 3, 15, 0, 0, 0, time.UTC))
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"sess-meta-legacy",
		"Legacy",
		"coder",
		rootDir,
		"user",
		"stopped",
		createdAt,
		createdAt,
	); err != nil {
		t.Fatalf("insert legacy session error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close(legacy db) error = %v", err)
	}

	sessionDir := filepath.Join(homeDir, "sessions", "sess-meta-legacy")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(sessionDir) error = %v", err)
	}
	metaPath := SessionMetaFile(sessionDir)
	legacyMeta := `{
  "id": "sess-meta-legacy",
  "name": "Legacy",
  "agent_name": "coder",
  "workspace": "` + rootDir + `",
  "session_type": "user",
  "state": "stopped",
  "created_at": "2026-04-03T15:00:00Z",
  "updated_at": "2026-04-03T15:00:00Z"
}
`
	if err := os.WriteFile(metaPath, []byte(legacyMeta), 0o644); err != nil {
		t.Fatalf("WriteFile(legacy meta) error = %v", err)
	}

	globalDB, err := OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	sessions, err := globalDB.ListSessions(ctx, SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}

	meta, err := ReadSessionMeta(metaPath)
	if err != nil {
		t.Fatalf("ReadSessionMeta() error = %v", err)
	}
	if got, want := meta.WorkspaceID, sessions[0].WorkspaceID; got != want {
		t.Fatalf("meta.WorkspaceID = %q, want %q", got, want)
	}

	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile(metaPath) error = %v", err)
	}
	if strings.Contains(string(data), `"workspace":`) {
		t.Fatalf("legacy workspace field still present in %s", metaPath)
	}
}

func TestGlobalDBWriteEventSummary(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-summary")

	if err := globalDB.WriteEventSummary(testutil.Context(t), EventSummary{
		SessionID:     "sess-summary",
		Type:          "agent_message",
		AgentName:     "coder",
		RootSessionID: "sess-summary",
		Summary:       "assistant replied",
		Timestamp:     time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("WriteEventSummary() error = %v", err)
	}

	summaries, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{SessionID: "sess-summary"})
	if err != nil {
		t.Fatalf("ListEventSummaries() error = %v", err)
	}
	if got, want := len(summaries), 1; got != want {
		t.Fatalf("len(summaries) = %d, want %d", got, want)
	}
	if summaries[0].Summary != "assistant replied" {
		t.Fatalf("summaries[0].Summary = %q, want %q", summaries[0].Summary, "assistant replied")
	}
	if got, want := summaries[0].RootSessionID, "sess-summary"; got != want {
		t.Fatalf("summaries[0].RootSessionID = %q, want %q", got, want)
	}
	if got, want := summaries[0].Provider, "claude"; got != want {
		t.Fatalf("summaries[0].Provider = %q, want %q", got, want)
	}
	if got, want := summaries[0].Outcome, "info"; got != want {
		t.Fatalf("summaries[0].Outcome = %q, want %q", got, want)
	}

	providerFiltered, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{
		Provider:  "claude",
		Component: "session",
	})
	if err != nil {
		t.Fatalf("ListEventSummaries(provider component) error = %v", err)
	}
	if got, want := len(providerFiltered), 1; got != want {
		t.Fatalf("len(providerFiltered) = %d, want %d", got, want)
	}

	if err := globalDB.WriteEventSummary(testutil.Context(t), EventSummary{
		SessionID: "sess-summary",
		Type:      "task.run_failed",
		AgentName: "coder",
		Summary:   "task run failed",
		Timestamp: time.Date(2026, 4, 3, 14, 1, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("WriteEventSummary(failed task run) error = %v", err)
	}
	errorOnly, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{
		Provider:  "claude",
		Component: "task",
		ErrorOnly: true,
	})
	if err != nil {
		t.Fatalf("ListEventSummaries(error only) error = %v", err)
	}
	if got, want := len(errorOnly), 1; got != want {
		t.Fatalf("len(errorOnly) = %d, want %d", got, want)
	}
	if got, want := errorOnly[0].Outcome, "failure"; got != want {
		t.Fatalf("errorOnly[0].Outcome = %q, want %q", got, want)
	}
}

func TestGlobalDBWriteEventSummaryAllowsGlobalEvents(t *testing.T) {
	t.Parallel()

	const skillShadowedBody = "" +
		`{"skill_name":"review","old_source":"workspace","new_source":"agent-local",` +
		`"layer_pair":"agent-local>workspace","shadow_kind":"logical_path",` +
		`"resolution_scope":"agent","agent_name":"reviewer"}`

	tests := []struct {
		name      string
		summary   EventSummary
		wantType  string
		wantAgent string
		wantBody  string
	}{
		{
			name: "Should persist settings changed events with content",
			summary: EventSummary{
				Type:      "settings.changed",
				Content:   []byte(`{"section":"skills","source":"http","operation":"patch"}`),
				Summary:   "skills settings changed",
				Timestamp: time.Date(2026, 5, 4, 14, 5, 0, 0, time.UTC),
			},
			wantType: "settings.changed",
			wantBody: `{"section":"skills","source":"http","operation":"patch"}`,
		},
		{
			name: "Should persist skill shadowed events without a session",
			summary: EventSummary{
				Type:      "skill.shadowed",
				AgentName: "reviewer",
				Content:   []byte(skillShadowedBody),
				Summary:   "skill review shadowed workspace with agent-local",
				Timestamp: time.Date(2026, 5, 4, 14, 6, 0, 0, time.UTC),
			},
			wantType:  "skill.shadowed",
			wantAgent: "reviewer",
			wantBody:  skillShadowedBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			globalDB := openTestGlobalDB(t)

			if err := globalDB.WriteEventSummary(testutil.Context(t), tt.summary); err != nil {
				t.Fatalf("WriteEventSummary(global event) error = %v", err)
			}

			summaries, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{})
			if err != nil {
				t.Fatalf("ListEventSummaries() error = %v", err)
			}
			if got, want := len(summaries), 1; got != want {
				t.Fatalf("len(summaries) = %d, want %d", got, want)
			}
			if got := summaries[0].SessionID; got != "" {
				t.Fatalf("summaries[0].SessionID = %q, want empty for global event", got)
			}
			if got, want := summaries[0].Type, tt.wantType; got != want {
				t.Fatalf("summaries[0].Type = %q, want %q", got, want)
			}
			if got, want := summaries[0].AgentName, tt.wantAgent; got != want {
				t.Fatalf("summaries[0].AgentName = %q, want %q", got, want)
			}
			if got, want := string(
				summaries[0].Content,
			), tt.wantBody; got != want {
				t.Fatalf("summaries[0].Content = %q, want %q", got, want)
			}
		})
	}
}

func TestProdReadyFoundationMigrationsAppendSharedState(t *testing.T) {
	t.Parallel()

	t.Run("Should upgrade v26 tail with shared event projections and operation state", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(prefix) error = %v", err)
		}
		if err := store.RunMigrations(ctx, db, globalSchemaMigrations[:26]); err != nil {
			t.Fatalf("RunMigrations(v26 prefix) error = %v", err)
		}

		now := formatTimestamp(time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC))
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO workspaces (id, root_dir, name, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?)`,
			"ws-prod-ready",
			filepath.Join(t.TempDir(), "workspace"),
			"prod-ready",
			now,
			now,
		); err != nil {
			t.Fatalf("insert workspace error = %v", err)
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO sessions (
				id, name, agent_name, provider, workspace_id, session_type, state, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			"sess-prod-ready",
			"Prod Ready",
			"coder",
			"codex",
			"ws-prod-ready",
			defaultSessionType,
			"stopped",
			now,
			now,
		); err != nil {
			t.Fatalf("insert session error = %v", err)
		}
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO event_summaries (id, session_id, workspace_id, type, agent_name, timestamp)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			"sum-prod-ready",
			"sess-prod-ready",
			"ws-prod-ready",
			"task.run_failed",
			"coder",
			now,
		); err != nil {
			t.Fatalf("insert event summary error = %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("prefix db.Close() error = %v", err)
		}

		globalDB, err := OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(upgrade) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
				t.Errorf("Close(upgrade) error = %v", closeErr)
			}
		})

		assertTablesPresent(t, globalDB.db, "config_apply_records", "scheduler_pause")
		assertTableHasColumns(t, globalDB.db, "event_summaries", []string{"provider", "outcome"})
		assertIndexesPresent(
			t,
			globalDB.db,
			"event_summaries",
			"idx_summaries_provider_timestamp",
			"idx_summaries_outcome_timestamp",
		)

		var provider string
		var outcome string
		if err := globalDB.db.QueryRowContext(
			ctx,
			`SELECT provider, outcome FROM event_summaries WHERE id = ?`,
			"sum-prod-ready",
		).Scan(&provider, &outcome); err != nil {
			t.Fatalf("select event summary projections error = %v", err)
		}
		if provider != "codex" || outcome != "failure" {
			t.Fatalf("event projections = provider %q outcome %q, want codex failure", provider, outcome)
		}

		var paused int
		if err := globalDB.db.QueryRowContext(
			ctx,
			`SELECT paused FROM scheduler_pause WHERE id = 1`,
		).Scan(&paused); err != nil {
			t.Fatalf("select scheduler_pause singleton error = %v", err)
		}
		if paused != 0 {
			t.Fatalf("scheduler_pause.paused = %d, want 0", paused)
		}
	})
}

func TestGlobalDBListEventSummariesIncludesMemoryOperations(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-summary")

	if err := globalDB.WriteEventSummary(testutil.Context(t), EventSummary{
		SessionID: "sess-summary",
		Type:      "agent_message",
		AgentName: "coder",
		Summary:   "assistant replied",
		Timestamp: time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("WriteEventSummary() error = %v", err)
	}
	if _, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`INSERT INTO memory_events (
			op, scope, agent_name, actor_kind, metadata, ts_ms
		) VALUES (?, ?, ?, ?, ?, ?)`,
		"memory.write.committed",
		"global",
		"daemon",
		"system",
		`{"summary":"scope=global filename=prefs.md"}`,
		time.Date(2026, 4, 3, 14, 1, 0, 0, time.UTC).UnixNano()/int64(time.Millisecond),
	); err != nil {
		t.Fatalf("insert memory event error = %v", err)
	}

	summaries, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{})
	if err != nil {
		t.Fatalf("ListEventSummaries() error = %v", err)
	}
	if got, want := len(summaries), 2; got != want {
		t.Fatalf("len(summaries) = %d, want %d", got, want)
	}
	if got, want := summaries[1].Type, "memory.write.committed"; got != want {
		t.Fatalf("summaries[1].Type = %q, want %q", got, want)
	}
	if got := summaries[1].SessionID; got != "" {
		t.Fatalf("summaries[1].SessionID = %q, want empty for memory operation", got)
	}

	sessionOnly, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{SessionID: "sess-summary"})
	if err != nil {
		t.Fatalf("ListEventSummaries(session filter) error = %v", err)
	}
	if got, want := len(sessionOnly), 1; got != want {
		t.Fatalf("len(sessionOnly) = %d, want %d", got, want)
	}
	if got, want := sessionOnly[0].Type, "agent_message"; got != want {
		t.Fatalf("sessionOnly[0].Type = %q, want %q", got, want)
	}

	if err := globalDB.WriteEventSummary(testutil.Context(t), EventSummary{
		SessionID: "sess-summary",
		Type:      "tool_call",
		AgentName: "coder",
		Summary:   "tool executed",
		Timestamp: time.Date(2026, 4, 3, 14, 2, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("WriteEventSummary(second event) error = %v", err)
	}

	limited, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{Limit: 2})
	if err != nil {
		t.Fatalf("ListEventSummaries(limit) error = %v", err)
	}
	if got, want := len(limited), 2; got != want {
		t.Fatalf("len(limited) = %d, want %d", got, want)
	}
	if got, want := limited[0].Type, "memory.write.committed"; got != want {
		t.Fatalf("limited[0].Type = %q, want %q", got, want)
	}
	if got, want := limited[1].Type, "tool_call"; got != want {
		t.Fatalf("limited[1].Type = %q, want %q", got, want)
	}

	if _, err := globalDB.db.ExecContext(
		testutil.Context(t),
		`INSERT INTO memory_events (
				op, scope, agent_name, actor_kind, metadata, ts_ms
			) VALUES (?, ?, ?, ?, ?, ?)`,
		"memory.write.rejected",
		"global",
		"daemon",
		"system",
		`{"summary":"subagent direct write denied"}`,
		time.Date(2026, 4, 3, 14, 3, 0, 0, time.UTC).UnixNano()/int64(time.Millisecond),
	); err != nil {
		t.Fatalf("insert rejected memory event error = %v", err)
	}

	errorMemory, err := globalDB.ListEventSummaries(
		testutil.Context(t),
		EventSummaryQuery{Component: "memory", ErrorOnly: true},
	)
	if err != nil {
		t.Fatalf("ListEventSummaries(memory error only) error = %v", err)
	}
	if got, want := len(errorMemory), 1; got != want {
		t.Fatalf("len(errorMemory) = %d, want %d; events=%#v", got, want, errorMemory)
	}
	if got, want := errorMemory[0].Type, "memory.write.rejected"; got != want {
		t.Fatalf("errorMemory[0].Type = %q, want %q", got, want)
	}
	if got, want := errorMemory[0].Outcome, "warning"; got != want {
		t.Fatalf("errorMemory[0].Outcome = %q, want %q", got, want)
	}

	taskOnly, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{Component: "task"})
	if err != nil {
		t.Fatalf("ListEventSummaries(task component) error = %v", err)
	}
	if len(taskOnly) != 0 {
		t.Fatalf("taskOnly events = %#v, want no memory rows for task component", taskOnly)
	}
}

func TestGlobalDBUpdateTokenStatsAggregation(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-stats")

	currency := "USD"
	inputA := int64(10)
	outputA := int64(20)
	totalA := int64(30)
	costA := 1.25
	if err := globalDB.UpdateTokenStats(testutil.Context(t), TokenStatsUpdate{
		SessionID:    "sess-stats",
		AgentName:    "coder",
		InputTokens:  &inputA,
		OutputTokens: &outputA,
		TotalTokens:  &totalA,
		CostAmount:   &costA,
		CostCurrency: &currency,
		Turns:        1,
	}); err != nil {
		t.Fatalf("UpdateTokenStats() error = %v", err)
	}

	outputB := int64(5)
	totalB := int64(5)
	costB := 0.75
	if err := globalDB.UpdateTokenStats(testutil.Context(t), TokenStatsUpdate{
		SessionID:    "sess-stats",
		AgentName:    "coder",
		OutputTokens: &outputB,
		TotalTokens:  &totalB,
		CostAmount:   &costB,
		CostCurrency: &currency,
		Turns:        1,
	}); err != nil {
		t.Fatalf("UpdateTokenStats() error = %v", err)
	}

	stats, err := globalDB.ListTokenStats(testutil.Context(t), TokenStatsQuery{SessionID: "sess-stats"})
	if err != nil {
		t.Fatalf("ListTokenStats() error = %v", err)
	}
	if got, want := len(stats), 1; got != want {
		t.Fatalf("len(stats) = %d, want %d", got, want)
	}
	if stats[0].InputTokens == nil || *stats[0].InputTokens != 10 {
		t.Fatalf("InputTokens = %#v, want 10", stats[0].InputTokens)
	}
	if stats[0].OutputTokens == nil || *stats[0].OutputTokens != 25 {
		t.Fatalf("OutputTokens = %#v, want 25", stats[0].OutputTokens)
	}
	if stats[0].TotalTokens == nil || *stats[0].TotalTokens != 35 {
		t.Fatalf("TotalTokens = %#v, want 35", stats[0].TotalTokens)
	}
	if stats[0].TotalCost == nil || *stats[0].TotalCost != 2.0 {
		t.Fatalf("TotalCost = %#v, want 2.0", stats[0].TotalCost)
	}
	if stats[0].CostCurrency == nil || *stats[0].CostCurrency != "USD" {
		t.Fatalf("CostCurrency = %#v, want USD", stats[0].CostCurrency)
	}
	if stats[0].TurnCount != 2 {
		t.Fatalf("TurnCount = %d, want 2", stats[0].TurnCount)
	}
}

func TestGlobalDBUpdateTokenStatsKeepsPerAgentRows(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-multi-agent")

	input := int64(10)
	if err := globalDB.UpdateTokenStats(testutil.Context(t), TokenStatsUpdate{
		SessionID:   "sess-multi-agent",
		AgentName:   "coder",
		InputTokens: &input,
	}); err != nil {
		t.Fatalf("UpdateTokenStats(coder) error = %v", err)
	}
	if err := globalDB.UpdateTokenStats(testutil.Context(t), TokenStatsUpdate{
		SessionID:   "sess-multi-agent",
		AgentName:   "reviewer",
		InputTokens: &input,
	}); err != nil {
		t.Fatalf("UpdateTokenStats(reviewer) error = %v", err)
	}

	stats, err := globalDB.ListTokenStats(testutil.Context(t), TokenStatsQuery{SessionID: "sess-multi-agent"})
	if err != nil {
		t.Fatalf("ListTokenStats() error = %v", err)
	}
	if got := len(stats); got != 2 {
		t.Fatalf("len(stats) = %d, want 2", got)
	}

	byAgent := make(map[string]TokenStats, len(stats))
	for _, stat := range stats {
		byAgent[stat.AgentName] = stat
	}
	if _, ok := byAgent["coder"]; !ok {
		t.Fatalf("missing coder stats: %#v", stats)
	}
	if _, ok := byAgent["reviewer"]; !ok {
		t.Fatalf("missing reviewer stats: %#v", stats)
	}
}

func TestGlobalDBUpdateSessionStateReturnsNotFoundForMissingSession(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	err := globalDB.UpdateSessionState(testutil.Context(t), SessionStateUpdate{
		ID:    "missing",
		State: "stopped",
	})
	if err == nil || !strings.Contains(err.Error(), `session "missing" not found`) {
		t.Fatalf("UpdateSessionState(missing) error = %v, want missing session error", err)
	}
}

func TestGlobalDBUpdateSessionStateRejectsUnmarshalableActivity(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-update-invalid-activity")
	unmarshalableTime := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	err := globalDB.UpdateSessionState(testutil.Context(t), SessionStateUpdate{
		ID:    "sess-update-invalid-activity",
		State: "active",
		Liveness: &store.SessionLivenessMeta{
			Activity: &store.SessionActivityMeta{
				TurnStartedAt: &unmarshalableTime,
			},
		},
	})
	if err == nil {
		t.Fatal("UpdateSessionState(unmarshalable activity) error = nil, want marshal failure")
	}
	if !strings.Contains(err.Error(), "store: build update session state") ||
		!strings.Contains(err.Error(), "store: session liveness activity marshal") {
		t.Fatalf("UpdateSessionState(unmarshalable activity) error = %v, want activity marshal context", err)
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].Liveness != nil && sessions[0].Liveness.Activity != nil {
		t.Fatalf(
			"sessions[0].Liveness.Activity = %#v, want failed update to skip activity write",
			sessions[0].Liveness.Activity,
		)
	}
}

func TestGlobalDBListSessionsWrapsInvalidActivityJSONValidation(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-invalid-activity-json")
	if _, err := globalDB.DB().ExecContext(
		testutil.Context(t),
		`UPDATE sessions SET activity_json = ? WHERE id = ?`,
		`{"idle_seconds":-1}`,
		"sess-invalid-activity-json",
	); err != nil {
		t.Fatalf("update invalid activity_json error = %v", err)
	}

	_, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{})
	if err == nil {
		t.Fatal("ListSessions(invalid activity_json) error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "store: validate session activity json") ||
		!strings.Contains(err.Error(), "store: session activity idle seconds must be zero or positive") {
		t.Fatalf("ListSessions(invalid activity_json) error = %v, want validation context", err)
	}
}

func TestGlobalDBUpdateSessionStateHandlesStopFields(t *testing.T) {
	t.Parallel()

	t.Run("Should update columns when stop reason is set", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"update-stop-reason",
			filepath.Join(t.TempDir(), "workspace"),
		)
		if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
			ID:          "sess-update-stop",
			AgentName:   "coder",
			WorkspaceID: workspaceID,
			State:       "active",
			CreatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("RegisterSession() error = %v", err)
		}

		stopReason := string(store.StopUserCanceled)
		if err := globalDB.UpdateSessionState(testutil.Context(t), SessionStateUpdate{
			ID:            "sess-update-stop",
			State:         "stopped",
			StopReasonSet: true,
			StopReason:    &stopReason,
			StopDetail:    "requested by user",
			UpdatedAt:     time.Date(2026, 4, 3, 13, 2, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("UpdateSessionState() error = %v", err)
		}

		gotStopReason, gotStopDetail := queryStoredSessionStopFields(t, globalDB.db, "sess-update-stop")
		assertOptionalStringEqual(t, gotStopReason, stringPointerForTest(string(store.StopUserCanceled)), "stop_reason")
		assertOptionalStringEqual(t, gotStopDetail, stringPointerForTest("requested by user"), "stop_detail")
	})

	t.Run("Should leave existing columns unchanged when stop reason is missing", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"preserve-stop-reason",
			filepath.Join(t.TempDir(), "workspace"),
		)
		if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
			ID:          "sess-preserve-stop",
			AgentName:   "coder",
			WorkspaceID: workspaceID,
			State:       "stopped",
			StopReason:  store.StopTimeout,
			StopDetail:  "deadline exceeded",
			CreatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("RegisterSession() error = %v", err)
		}

		if err := globalDB.UpdateSessionState(testutil.Context(t), SessionStateUpdate{
			ID:        "sess-preserve-stop",
			State:     "orphaned",
			UpdatedAt: time.Date(2026, 4, 3, 13, 5, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("UpdateSessionState() error = %v", err)
		}

		gotStopReason, gotStopDetail := queryStoredSessionStopFields(t, globalDB.db, "sess-preserve-stop")
		assertOptionalStringEqual(t, gotStopReason, stringPointerForTest(string(store.StopTimeout)), "stop_reason")
		assertOptionalStringEqual(t, gotStopDetail, stringPointerForTest("deadline exceeded"), "stop_detail")
	})

	t.Run("Should clear existing columns when stop reason is explicitly nil", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"clear-stop-reason",
			filepath.Join(t.TempDir(), "workspace"),
		)
		if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
			ID:          "sess-clear-stop",
			AgentName:   "coder",
			WorkspaceID: workspaceID,
			State:       "stopped",
			StopReason:  store.StopTimeout,
			StopDetail:  "deadline exceeded",
			CreatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("RegisterSession() error = %v", err)
		}

		if err := globalDB.UpdateSessionState(testutil.Context(t), SessionStateUpdate{
			ID:            "sess-clear-stop",
			State:         "active",
			StopReasonSet: true,
			UpdatedAt:     time.Date(2026, 4, 3, 13, 5, 0, 0, time.UTC),
		}); err != nil {
			t.Fatalf("UpdateSessionState() error = %v", err)
		}

		gotStopReason, gotStopDetail := queryStoredSessionStopFields(t, globalDB.db, "sess-clear-stop")
		assertOptionalStringEqual(t, gotStopReason, nil, "stop_reason")
		assertOptionalStringEqual(t, gotStopDetail, nil, "stop_detail")
	})
}

func TestGlobalDBWritePermissionLogEntry(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-perm")

	if err := globalDB.WritePermissionLog(testutil.Context(t), PermissionLogEntry{
		SessionID:  "sess-perm",
		AgentName:  "coder",
		Action:     "bash",
		Resource:   "/tmp/project",
		Decision:   "allow",
		PolicyUsed: "approve-reads",
		Timestamp:  time.Date(2026, 4, 3, 15, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("WritePermissionLog() error = %v", err)
	}

	entries, err := globalDB.ListPermissionLog(testutil.Context(t), PermissionLogQuery{SessionID: "sess-perm"})
	if err != nil {
		t.Fatalf("ListPermissionLog() error = %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}
	if entries[0].Decision != "allow" || entries[0].PolicyUsed != "approve-reads" {
		t.Fatalf("entry = %#v, want allow/approve-reads", entries[0])
	}
}

func TestGlobalDBReconcileSessions(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	registerSessionForGlobalTests(t, globalDB, "sess-keep")
	registerSessionForGlobalTests(t, globalDB, "sess-orphan")

	onDisk := []SessionInfo{
		{
			ID:        "sess-keep",
			AgentName: "coder",
			Provider:  "claude",
			WorkspaceID: registerWorkspaceForGlobalTests(
				t,
				globalDB,
				"sess-keep-reconciled-workspace",
				filepath.Join(t.TempDir(), "sess-keep"),
			),
			State:      "stopped",
			StopReason: store.StopCompleted,
			CreatedAt:  time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
			UpdatedAt:  time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
		},
		{
			ID:        "sess-new",
			AgentName: "reviewer",
			Provider:  "codex",
			WorkspaceID: registerWorkspaceForGlobalTests(
				t,
				globalDB,
				"sess-new-reconciled-workspace",
				filepath.Join(t.TempDir(), "sess-new"),
			),
			State:      "stopped",
			StopReason: store.StopUserCanceled,
			StopDetail: "requested by API",
			CreatedAt:  time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
			UpdatedAt:  time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
		},
	}

	result, err := globalDB.ReconcileSessions(testutil.Context(t), onDisk)
	if err != nil {
		t.Fatalf("ReconcileSessions() error = %v", err)
	}
	sort.Strings(result.Indexed)
	sort.Strings(result.Orphaned)
	if !testutil.EqualStringSlices(result.Indexed, []string{"sess-new"}) {
		t.Fatalf("Indexed = %#v, want %#v", result.Indexed, []string{"sess-new"})
	}
	if !testutil.EqualStringSlices(result.Orphaned, []string{"sess-orphan"}) {
		t.Fatalf("Orphaned = %#v, want %#v", result.Orphaned, []string{"sess-orphan"})
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	stateByID := make(map[string]string, len(sessions))
	stopReasonByID := make(map[string]store.StopReason, len(sessions))
	providerByID := make(map[string]string, len(sessions))
	for _, session := range sessions {
		stateByID[session.ID] = session.State
		stopReasonByID[session.ID] = session.StopReason
		providerByID[session.ID] = session.Provider
	}
	if stateByID["sess-new"] != "stopped" {
		t.Fatalf("stateByID[sess-new] = %q, want stopped", stateByID["sess-new"])
	}
	if stopReasonByID["sess-new"] != store.StopUserCanceled {
		t.Fatalf("stopReasonByID[sess-new] = %q, want %q", stopReasonByID["sess-new"], store.StopUserCanceled)
	}
	if providerByID["sess-new"] != "codex" {
		t.Fatalf("providerByID[sess-new] = %q, want codex", providerByID["sess-new"])
	}
	if stateByID["sess-orphan"] != "orphaned" {
		t.Fatalf("stateByID[sess-orphan] = %q, want orphaned", stateByID["sess-orphan"])
	}
}

func TestGlobalDBReconcileSessionsSkipsDuplicateIDsAndDefaultsTimestamps(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	reconciledAt := time.Date(2026, 4, 3, 16, 30, 0, 0, time.UTC)
	globalDB.now = func() time.Time {
		return reconciledAt
	}

	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"sess-duplicate-reconciled-workspace",
		filepath.Join(t.TempDir(), "sess-duplicate"),
	)
	onDisk := []SessionInfo{
		{
			ID:          "sess-duplicate",
			AgentName:   "coder",
			Provider:    "claude",
			WorkspaceID: workspaceID,
			State:       "stopped",
		},
		{
			ID:          "sess-duplicate",
			AgentName:   "coder",
			Provider:    "codex",
			WorkspaceID: workspaceID,
			State:       "orphaned",
		},
	}

	result, err := globalDB.ReconcileSessions(testutil.Context(t), onDisk)
	if err != nil {
		t.Fatalf("ReconcileSessions() error = %v", err)
	}
	if !testutil.EqualStringSlices(result.Indexed, []string{"sess-duplicate"}) {
		t.Fatalf("Indexed = %#v, want %#v", result.Indexed, []string{"sess-duplicate"})
	}
	if len(result.Orphaned) != 0 {
		t.Fatalf("Orphaned = %#v, want empty", result.Orphaned)
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}

	got := sessions[0]
	if got.Provider != "claude" {
		t.Fatalf("sessions[0].Provider = %q, want claude from first reconcile entry", got.Provider)
	}
	if got.State != "stopped" {
		t.Fatalf("sessions[0].State = %q, want stopped from first reconcile entry", got.State)
	}
	if !got.CreatedAt.Equal(reconciledAt) {
		t.Fatalf("sessions[0].CreatedAt = %v, want %v", got.CreatedAt, reconciledAt)
	}
	if !got.UpdatedAt.Equal(reconciledAt) {
		t.Fatalf("sessions[0].UpdatedAt = %v, want %v", got.UpdatedAt, reconciledAt)
	}
}

func TestOpenGlobalDBAddsStopColumnsToCurrentSessionSchema(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, GlobalDatabaseName)

	db, err := sql.Open(sqliteDriverName, sqliteDSN(path))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	ctx := testutil.Context(t)
	if _, err := db.ExecContext(ctx, `CREATE TABLE workspaces (
		id TEXT PRIMARY KEY,
		root_dir TEXT NOT NULL UNIQUE,
		add_dirs TEXT NOT NULL DEFAULT '[]',
		name TEXT NOT NULL UNIQUE,
		default_agent TEXT DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create workspaces error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		name TEXT,
		agent_name TEXT NOT NULL,
		workspace_id TEXT NOT NULL REFERENCES workspaces(id),
		session_type TEXT NOT NULL DEFAULT 'user',
		state TEXT NOT NULL,
		acp_session_id TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create current sessions error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO workspaces (id, root_dir, add_dirs, name, default_agent, created_at, updated_at) VALUES (?, ?, '[]', ?, '', ?, ?)`,
		"ws-current",
		filepath.Join(dir, "workspace"),
		"current",
		formatTimestamp(time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)),
		formatTimestamp(time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert workspace error = %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, name, agent_name, workspace_id, session_type, state, acp_session_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"sess-current",
		"Current",
		"coder",
		"ws-current",
		"user",
		"stopped",
		"acp-current",
		formatTimestamp(time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)),
		formatTimestamp(time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert session error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close(current schema db) error = %v", err)
	}

	globalDB, err := OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	assertTableColumns(
		t,
		globalDB.db,
		"sessions",
		[]string{
			"id",
			"name",
			"agent_name",
			"workspace_id",
			"session_type",
			"state",
			"acp_session_id",
			"created_at",
			"updated_at",
			"provider",
			"stop_reason",
			"stop_detail",
			"failure_kind",
			"failure_summary",
			"crash_bundle_path",
			"channel",
			"subprocess_pid",
			"subprocess_started_at",
			"last_update_at",
			"stall_state",
			"stall_reason",
			"activity_json",
			"sandbox_id",
			"sandbox_backend",
			"sandbox_profile",
			"sandbox_instance_id",
			"sandbox_state",
			"sandbox_provider_state_json",
			"sandbox_last_sync_at",
			"sandbox_last_sync_error",
			"parent_session_id",
			"root_session_id",
			"spawn_depth",
			"spawn_role",
			"ttl_expires_at",
			"auto_stop_on_parent",
			"spawn_budget_json",
			"permission_policy_json",
			"soul_snapshot_id",
			"soul_digest",
			"parent_soul_digest",
			"attached_to",
			"attach_expires_at",
			sessionInputGenerationColumn,
		},
	)
	assertTableColumns(
		t,
		globalDB.db,
		"workspaces",
		[]string{"id", "root_dir", "add_dirs", "name", "default_agent", "created_at", "updated_at", "sandbox_ref"},
	)

	sessions, err := globalDB.ListSessions(ctx, SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].StopReason != "" || sessions[0].StopDetail != "" {
		t.Fatalf("sessions[0] stop fields = %#v, want empty after migration", sessions[0])
	}
	if sessions[0].Channel != "" {
		t.Fatalf("sessions[0].Channel = %q, want empty after migration", sessions[0].Channel)
	}
}

func TestGlobalDBRecoversFromCorruption(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, GlobalDatabaseName)
	if err := os.WriteFile(path, []byte("bad sqlite"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	globalDB, err := OpenGlobalDB(testutil.Context(t), path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := globalDB.Close(testutil.Context(t)); closeErr != nil {
			t.Fatalf("Close() error = %v", closeErr)
		}
	})

	assertTablesPresent(
		t,
		globalDB.db,
		"schema_migrations",
		"workspaces",
		"sessions",
		"event_summaries",
		"memory_events",
		"token_stats",
		"permission_log",
	)

	matches, err := filepath.Glob(path + ".corrupt.*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if got, want := len(matches), 1; got != want {
		t.Fatalf("len(corrupt files) = %d, want %d (%v)", got, want, matches)
	}
}

func openTestGlobalDB(t *testing.T) *GlobalDB {
	t.Helper()

	path := filepath.Join(t.TempDir(), GlobalDatabaseName)
	copyCurrentSchemaGlobalDBSeed(t, path)
	return openGlobalDBForTest(t, path)
}

func openFreshTestGlobalDB(t *testing.T) *GlobalDB {
	t.Helper()

	return openGlobalDBForTest(t, filepath.Join(t.TempDir(), GlobalDatabaseName))
}

func openGlobalDBForTest(t *testing.T, path string) *GlobalDB {
	t.Helper()

	globalDB, err := OpenGlobalDB(testutil.Context(t), path)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(testutil.Context(t)); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return globalDB
}

func copyCurrentSchemaGlobalDBSeed(t *testing.T, targetPath string) {
	t.Helper()

	if testGlobalDBCurrentSchemaSeedPath == "" {
		t.Fatal("globaldb current schema seed path is empty")
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		sourcePath := testGlobalDBCurrentSchemaSeedPath + suffix
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			if suffix != "" && errors.Is(err, os.ErrNotExist) {
				continue
			}
			t.Fatalf("ReadFile(%q) error = %v", sourcePath, err)
		}
		if err := os.WriteFile(targetPath+suffix, data, 0o600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", targetPath+suffix, err)
		}
	}
}

func registerSessionForGlobalTests(t *testing.T, globalDB *GlobalDB, sessionID string) string {
	t.Helper()

	now := time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC)
	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		sessionID+"-workspace",
		filepath.Join(t.TempDir(), sessionID),
	)
	if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          sessionID,
		AgentName:   "coder",
		Provider:    "claude",
		WorkspaceID: workspaceID,
		State:       "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("RegisterSession(%q) error = %v", sessionID, err)
	}
	return workspaceID
}

func insertWorkspaceForGlobalTests(t *testing.T, globalDB *GlobalDB, ws aghworkspace.Workspace) aghworkspace.Workspace {
	t.Helper()

	if strings.TrimSpace(ws.RootDir) == "" {
		t.Fatal("insertWorkspaceForGlobalTests() requires RootDir")
	}
	if err := os.MkdirAll(ws.RootDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", ws.RootDir, err)
	}
	if ws.CreatedAt.IsZero() {
		ws.CreatedAt = time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)
	}
	if ws.UpdatedAt.IsZero() {
		ws.UpdatedAt = ws.CreatedAt
	}
	if err := globalDB.InsertWorkspace(testutil.Context(t), ws); err != nil {
		t.Fatalf("InsertWorkspace(%q) error = %v", ws.ID, err)
	}
	return ws
}

func registerWorkspaceForGlobalTests(t *testing.T, globalDB *GlobalDB, name string, rootDir string) string {
	t.Helper()

	workspace := insertWorkspaceForGlobalTests(t, globalDB, aghworkspace.Workspace{
		ID:        "ws-" + strings.ReplaceAll(name, " ", "-"),
		RootDir:   rootDir,
		Name:      name,
		CreatedAt: time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC),
	})
	return workspace.ID
}

func assertEventSummaryIDs(t *testing.T, globalDB *GlobalDB, want []string) {
	t.Helper()

	events, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{})
	if err != nil {
		t.Fatalf("ListEventSummaries() error = %v", err)
	}
	got := make([]string, 0, len(events))
	for _, event := range events {
		got = append(got, event.ID)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("event summary ids = %#v, want %#v", got, want)
	}
}

func assertTokenStatAgents(t *testing.T, globalDB *GlobalDB, want []string) {
	t.Helper()

	stats, err := globalDB.ListTokenStats(testutil.Context(t), TokenStatsQuery{})
	if err != nil {
		t.Fatalf("ListTokenStats() error = %v", err)
	}
	got := make([]string, 0, len(stats))
	for _, stat := range stats {
		got = append(got, stat.AgentName)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("token stat agents = %#v, want %#v", got, want)
	}
}

func assertPermissionLogIDs(t *testing.T, globalDB *GlobalDB, want []string) {
	t.Helper()

	entries, err := globalDB.ListPermissionLog(testutil.Context(t), PermissionLogQuery{})
	if err != nil {
		t.Fatalf("ListPermissionLog() error = %v", err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.ID)
	}
	sort.Strings(got)
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Fatalf("permission log ids = %#v, want %#v", got, want)
	}
}

func assertWorkspaceEqual(t *testing.T, got aghworkspace.Workspace, want aghworkspace.Workspace) {
	t.Helper()

	if got.ID != want.ID ||
		got.RootDir != want.RootDir ||
		got.Name != want.Name ||
		got.DefaultAgent != want.DefaultAgent ||
		got.SandboxRef != want.SandboxRef ||
		!got.CreatedAt.Equal(want.CreatedAt) ||
		!got.UpdatedAt.Equal(want.UpdatedAt) ||
		!testutil.EqualStringSlices(got.AdditionalDirs, want.AdditionalDirs) {
		t.Fatalf("workspace = %#v, want %#v", got, want)
	}
}

func assertTableColumns(t *testing.T, db *sql.DB, table string, want []string) {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), "PRAGMA table_info("+table+")")
	if err != nil {
		t.Fatalf("QueryContext(table_info %q) error = %v", table, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Fatalf("rows.Close(table_info %q) error = %v", table, closeErr)
		}
	}()

	got := make([]string, 0)
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
			t.Fatalf("Scan(table_info %q) error = %v", table, err)
		}
		got = append(got, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err(table_info %q) error = %v", table, err)
	}

	if !testutil.EqualStringSlices(got, want) {
		t.Fatalf("columns(%s) = %#v, want %#v", table, got, want)
	}
}

func queryStoredSessionStopFields(t *testing.T, db *sql.DB, sessionID string) (*string, *string) {
	t.Helper()

	var stopReason sql.NullString
	var stopDetail sql.NullString
	if err := db.QueryRowContext(testutil.Context(t), `SELECT stop_reason, stop_detail FROM sessions WHERE id = ?`, sessionID).
		Scan(&stopReason, &stopDetail); err != nil {
		t.Fatalf("QueryRowContext(stop fields %q) error = %v", sessionID, err)
	}
	return store.NullString(stopReason), store.NullString(stopDetail)
}

func assertOptionalStringEqual(t *testing.T, got *string, want *string, field string) {
	t.Helper()

	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	case *got != *want:
		t.Fatalf("%s = %q, want %q", field, *got, *want)
	}
}

func stringPointerForTest(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	copyValue := value
	return &copyValue
}

func ptrTime(value time.Time) *time.Time {
	copyValue := value.UTC()
	return &copyValue
}

func assertTablesPresent(t *testing.T, db *sql.DB, want ...string) {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), `SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		t.Fatalf("QueryContext(sqlite_master) error = %v", err)
	}
	defer func() { _ = rows.Close() }()

	got := make(map[string]struct{})
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			t.Fatalf("rows.Scan() error = %v", scanErr)
		}
		got[name] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err() = %v", err)
	}

	for _, table := range want {
		if _, ok := got[table]; !ok {
			t.Fatalf("table %q missing from sqlite_master: %#v", table, got)
		}
	}
}

func assertJournalModeWAL(t *testing.T, db *sql.DB) {
	t.Helper()

	var journalMode string
	if err := db.QueryRowContext(testutil.Context(t), `PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatalf("QueryRowContext(PRAGMA journal_mode) error = %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("PRAGMA journal_mode = %q, want wal", journalMode)
	}
}

func assertSynchronousNormal(t *testing.T, db *sql.DB) {
	t.Helper()

	var synchronous int
	if err := db.QueryRowContext(testutil.Context(t), `PRAGMA synchronous`).Scan(&synchronous); err != nil {
		t.Fatalf("QueryRowContext(PRAGMA synchronous) error = %v", err)
	}
	if synchronous != 1 {
		t.Fatalf("PRAGMA synchronous = %d, want 1 (NORMAL)", synchronous)
	}
}
