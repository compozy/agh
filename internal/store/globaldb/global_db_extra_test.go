package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func nilGlobalContext() context.Context {
	return nil
}

func TestGlobalDBPathAndCloseVariants(t *testing.T) {
	t.Parallel()

	var nilDB *GlobalDB
	if got := nilDB.Path(); got != "" {
		t.Fatalf("nil Path() = %q, want empty", got)
	}
	if err := nilDB.Close(testutil.Context(t)); err != nil {
		t.Fatalf("nil Close() error = %v", err)
	}

	globalDB := openTestGlobalDB(t)
	if got, want := globalDB.Path(), globalDB.path; got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
	if err := globalDB.Close(nilGlobalContext()); err == nil {
		t.Fatal("Close(nil ctx) error = nil, want non-nil")
	}
}

func TestGlobalDBTransactionCleanupHelpers(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db, err := store.OpenSQLiteDatabase(ctx, filepath.Join(t.TempDir(), "cleanup.db"), nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := rollbackTx(nil, "nil"); err != nil {
		t.Fatalf("rollbackTx(nil) error = %v", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}
	if err := rollbackTx(tx, "committed"); err != nil {
		t.Fatalf("rollbackTx(committed) error = %v", err)
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("Conn() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := rollbackImmediate(ctx, nil, "nil"); err != nil {
		t.Fatalf("rollbackImmediate(nil) error = %v", err)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("BEGIN IMMEDIATE error = %v", err)
	}
	if err := rollbackImmediate(ctx, conn, "cleanup"); err != nil {
		t.Fatalf("rollbackImmediate(active) error = %v", err)
	}
	if err := restoreForeignKeys(ctx, nil); err != nil {
		t.Fatalf("restoreForeignKeys(nil) error = %v", err)
	}
	if err := restoreForeignKeys(ctx, conn); err != nil {
		t.Fatalf("restoreForeignKeys(conn) error = %v", err)
	}

	primaryErr := errors.New("primary")
	cleanupErr := errors.New("cleanup")
	joinCleanupError(nil, cleanupErr)
	var target error
	joinCleanupError(&target, nil)
	if target != nil {
		t.Fatalf("joinCleanupError(nil cleanup) = %v, want nil", target)
	}
	joinCleanupError(&target, cleanupErr)
	if !errors.Is(target, cleanupErr) {
		t.Fatalf("joinCleanupError(cleanup only) = %v, want cleanup", target)
	}
	target = primaryErr
	joinCleanupError(&target, cleanupErr)
	if !errors.Is(target, primaryErr) || !errors.Is(target, cleanupErr) {
		t.Fatalf("joinCleanupError(joined) = %v, want primary and cleanup", target)
	}
}

func TestTableDefinitionSQLAndSessionsDirForDatabasePath(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	db, err := store.OpenSQLiteDatabase(ctx, filepath.Join(t.TempDir(), "table-definition.db"), nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.ExecContext(
		ctx,
		`CREATE TABLE sample_table (id TEXT PRIMARY KEY, summary TEXT NOT NULL)`,
	); err != nil {
		t.Fatalf("CREATE TABLE sample_table error = %v", err)
	}

	definition, err := tableDefinitionSQL(ctx, db, "sample_table")
	if err != nil {
		t.Fatalf("tableDefinitionSQL(sample_table) error = %v", err)
	}
	if !strings.Contains(definition, "CREATE TABLE sample_table") {
		t.Fatalf("tableDefinitionSQL(sample_table) = %q, want CREATE TABLE statement", definition)
	}

	if _, err := tableDefinitionSQL(ctx, db, "missing_table"); err == nil {
		t.Fatal("tableDefinitionSQL(missing_table) error = nil, want missing-table failure")
	}

	if got := sessionsDirForDatabasePath("   "); got != "" {
		t.Fatalf("sessionsDirForDatabasePath(blank) = %q, want empty", got)
	}
	dbPath := filepath.Join(t.TempDir(), "agh.db")
	if got, want := sessionsDirForDatabasePath(dbPath), filepath.Join(filepath.Dir(dbPath), "sessions"); got != want {
		t.Fatalf("sessionsDirForDatabasePath(%q) = %q, want %q", dbPath, got, want)
	}
}

func TestGlobalDBGuardClauses(t *testing.T) {
	t.Parallel()

	var nilDB *GlobalDB
	if err := nilDB.RegisterSession(testutil.Context(t), SessionInfo{}); err == nil {
		t.Fatal("RegisterSession(nil receiver) error = nil, want non-nil")
	}
	if _, err := nilDB.ListSessions(testutil.Context(t), SessionListQuery{}); err == nil {
		t.Fatal("ListSessions(nil receiver) error = nil, want non-nil")
	}
	if err := nilDB.WriteEventSummary(testutil.Context(t), EventSummary{}); err == nil {
		t.Fatal("WriteEventSummary(nil receiver) error = nil, want non-nil")
	}
	if _, err := nilDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{}); err == nil {
		t.Fatal("ListEventSummaries(nil receiver) error = nil, want non-nil")
	}
	if err := nilDB.UpdateTokenStats(testutil.Context(t), TokenStatsUpdate{}); err == nil {
		t.Fatal("UpdateTokenStats(nil receiver) error = nil, want non-nil")
	}
	if _, err := nilDB.ListTokenStats(testutil.Context(t), TokenStatsQuery{}); err == nil {
		t.Fatal("ListTokenStats(nil receiver) error = nil, want non-nil")
	}
	if err := nilDB.WritePermissionLog(testutil.Context(t), PermissionLogEntry{}); err == nil {
		t.Fatal("WritePermissionLog(nil receiver) error = nil, want non-nil")
	}
	if _, err := nilDB.ListPermissionLog(testutil.Context(t), PermissionLogQuery{}); err == nil {
		t.Fatal("ListPermissionLog(nil receiver) error = nil, want non-nil")
	}

	globalDB := openTestGlobalDB(t)
	if err := globalDB.RegisterSession(nilGlobalContext(), SessionInfo{}); err == nil {
		t.Fatal("RegisterSession(nil ctx) error = nil, want non-nil")
	}
	if _, err := globalDB.ListSessions(nilGlobalContext(), SessionListQuery{}); err == nil {
		t.Fatal("ListSessions(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.WriteEventSummary(nilGlobalContext(), EventSummary{}); err == nil {
		t.Fatal("WriteEventSummary(nil ctx) error = nil, want non-nil")
	}
	if _, err := globalDB.ListEventSummaries(nilGlobalContext(), EventSummaryQuery{}); err == nil {
		t.Fatal("ListEventSummaries(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.UpdateTokenStats(nilGlobalContext(), TokenStatsUpdate{}); err == nil {
		t.Fatal("UpdateTokenStats(nil ctx) error = nil, want non-nil")
	}
	if _, err := globalDB.ListTokenStats(nilGlobalContext(), TokenStatsQuery{}); err == nil {
		t.Fatal("ListTokenStats(nil ctx) error = nil, want non-nil")
	}
	if err := globalDB.WritePermissionLog(nilGlobalContext(), PermissionLogEntry{}); err == nil {
		t.Fatal("WritePermissionLog(nil ctx) error = nil, want non-nil")
	}
	if _, err := globalDB.ListPermissionLog(nilGlobalContext(), PermissionLogQuery{}); err == nil {
		t.Fatal("ListPermissionLog(nil ctx) error = nil, want non-nil")
	}
}

func TestGlobalDBWorkspaceAndAutomationGuardClauses(t *testing.T) {
	t.Parallel()

	ctx := testutil.Context(t)
	assertErr := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s error = nil, want non-nil", name)
		}
	}

	var nilDB *GlobalDB
	assertErr("InsertWorkspace(nil receiver)", nilDB.InsertWorkspace(ctx, aghworkspace.Workspace{}))
	assertErr("UpdateWorkspace(nil receiver)", nilDB.UpdateWorkspace(ctx, aghworkspace.Workspace{}))
	assertErr("DeleteWorkspace(nil receiver)", nilDB.DeleteWorkspace(ctx, "ws-1"))
	_, err := nilDB.GetWorkspace(ctx, "ws-1")
	assertErr("GetWorkspace(nil receiver)", err)
	_, err = nilDB.GetWorkspaceByPath(ctx, "/tmp/ws-1")
	assertErr("GetWorkspaceByPath(nil receiver)", err)
	_, err = nilDB.GetWorkspaceByName(ctx, "ws-1")
	assertErr("GetWorkspaceByName(nil receiver)", err)
	_, err = nilDB.ListWorkspaces(ctx)
	assertErr("ListWorkspaces(nil receiver)", err)

	_, err = nilDB.CreateJob(ctx, Job{})
	assertErr("CreateJob(nil receiver)", err)
	_, err = nilDB.UpdateJob(ctx, Job{})
	assertErr("UpdateJob(nil receiver)", err)
	assertErr("DeleteJob(nil receiver)", nilDB.DeleteJob(ctx, "job-1"))
	_, err = nilDB.GetJob(ctx, "job-1")
	assertErr("GetJob(nil receiver)", err)
	_, err = nilDB.ListJobs(ctx, JobListQuery{})
	assertErr("ListJobs(nil receiver)", err)
	_, err = nilDB.CreateTrigger(ctx, Trigger{})
	assertErr("CreateTrigger(nil receiver)", err)
	_, err = nilDB.UpdateTrigger(ctx, Trigger{})
	assertErr("UpdateTrigger(nil receiver)", err)
	assertErr("DeleteTrigger(nil receiver)", nilDB.DeleteTrigger(ctx, "trigger-1"))
	_, err = nilDB.GetTrigger(ctx, "trigger-1")
	assertErr("GetTrigger(nil receiver)", err)
	_, err = nilDB.GetTriggerByWebhookID(ctx, "webhook-1")
	assertErr("GetTriggerByWebhookID(nil receiver)", err)
	_, err = nilDB.ListTriggers(ctx, TriggerListQuery{})
	assertErr("ListTriggers(nil receiver)", err)
	_, err = nilDB.CreateRun(ctx, Run{})
	assertErr("CreateRun(nil receiver)", err)
	_, err = nilDB.UpdateRun(ctx, Run{})
	assertErr("UpdateRun(nil receiver)", err)
	assertErr("DeleteRun(nil receiver)", nilDB.DeleteRun(ctx, "run-1"))
	_, err = nilDB.GetRun(ctx, "run-1")
	assertErr("GetRun(nil receiver)", err)
	_, err = nilDB.ListRuns(ctx, RunQuery{})
	assertErr("ListRuns(nil receiver)", err)
	_, err = nilDB.CountRuns(ctx, RunQuery{})
	assertErr("CountRuns(nil receiver)", err)
	_, err = nilDB.SetJobEnabledOverlay(ctx, JobEnabledOverlay{})
	assertErr("SetJobEnabledOverlay(nil receiver)", err)
	_, err = nilDB.GetJobEnabledOverlay(ctx, "job-1")
	assertErr("GetJobEnabledOverlay(nil receiver)", err)
	_, err = nilDB.ListJobEnabledOverlays(ctx)
	assertErr("ListJobEnabledOverlays(nil receiver)", err)
	assertErr("DeleteJobEnabledOverlay(nil receiver)", nilDB.DeleteJobEnabledOverlay(ctx, "job-1"))
	_, err = nilDB.SetTriggerEnabledOverlay(ctx, TriggerEnabledOverlay{})
	assertErr("SetTriggerEnabledOverlay(nil receiver)", err)
	_, err = nilDB.GetTriggerEnabledOverlay(ctx, "trigger-1")
	assertErr("GetTriggerEnabledOverlay(nil receiver)", err)
	_, err = nilDB.ListTriggerEnabledOverlays(ctx)
	assertErr("ListTriggerEnabledOverlays(nil receiver)", err)
	assertErr("DeleteTriggerEnabledOverlay(nil receiver)", nilDB.DeleteTriggerEnabledOverlay(ctx, "trigger-1"))
	assertErr("SetTriggerWebhookSecret(nil receiver)", nilDB.SetTriggerWebhookSecret(ctx, "trigger-1", "secret"))
	_, err = nilDB.GetTriggerWebhookSecret(ctx, "trigger-1")
	assertErr("GetTriggerWebhookSecret(nil receiver)", err)
	assertErr("DeleteTriggerWebhookSecret(nil receiver)", nilDB.DeleteTriggerWebhookSecret(ctx, "trigger-1"))

	globalDB := openTestGlobalDB(t)
	assertErr("InsertWorkspace(nil ctx)", globalDB.InsertWorkspace(nilGlobalContext(), aghworkspace.Workspace{}))
	assertErr("UpdateWorkspace(nil ctx)", globalDB.UpdateWorkspace(nilGlobalContext(), aghworkspace.Workspace{}))
	assertErr("DeleteWorkspace(nil ctx)", globalDB.DeleteWorkspace(nilGlobalContext(), "ws-1"))
	_, err = globalDB.GetWorkspace(nilGlobalContext(), "ws-1")
	assertErr("GetWorkspace(nil ctx)", err)
	_, err = globalDB.ListWorkspaces(nilGlobalContext())
	assertErr("ListWorkspaces(nil ctx)", err)
	_, err = globalDB.CreateJob(nilGlobalContext(), Job{})
	assertErr("CreateJob(nil ctx)", err)
	_, err = globalDB.CreateTrigger(nilGlobalContext(), Trigger{})
	assertErr("CreateTrigger(nil ctx)", err)
	_, err = globalDB.CreateRun(nilGlobalContext(), Run{})
	assertErr("CreateRun(nil ctx)", err)
	_, err = globalDB.CountRuns(nilGlobalContext(), RunQuery{})
	assertErr("CountRuns(nil ctx)", err)
}

func TestGlobalDBDefaultsAndFilteredListings(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	base := time.Date(2026, 4, 4, 13, 0, 0, 0, time.UTC)
	callCount := 0
	globalDB.now = func() time.Time {
		callCount++
		return base.Add(time.Duration(callCount) * time.Minute)
	}

	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"filtered-workspace",
		filepath.Join(t.TempDir(), "filtered-workspace"),
	)
	if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          "sess-defaults",
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		State:       "active",
	}); err != nil {
		t.Fatalf("RegisterSession(defaults) error = %v", err)
	}
	if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          "sess-reviewer",
		AgentName:   "reviewer",
		WorkspaceID: workspaceID,
		State:       "active",
		CreatedAt:   base.Add(-time.Hour),
		UpdatedAt:   base.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("RegisterSession(reviewer) error = %v", err)
	}

	sessions, err := globalDB.ListSessions(testutil.Context(t), SessionListQuery{AgentName: "coder", Limit: 1})
	if err != nil {
		t.Fatalf("ListSessions(filtered) error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if got, want := sessions[0].SessionType, defaultSessionType; got != want {
		t.Fatalf("sessions[0].SessionType = %q, want %q", got, want)
	}

	if err := globalDB.WriteEventSummary(testutil.Context(t), EventSummary{
		SessionID: "sess-defaults",
		Type:      "agent_message",
		AgentName: "coder",
	}); err != nil {
		t.Fatalf("WriteEventSummary(default timestamp) error = %v", err)
	}
	if err := globalDB.WriteEventSummary(testutil.Context(t), EventSummary{
		SessionID: "sess-reviewer",
		Type:      "tool_call",
		AgentName: "reviewer",
		Timestamp: base.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("WriteEventSummary(explicit timestamp) error = %v", err)
	}

	summaries, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{
		AgentName: "coder",
		Type:      "agent_message",
		Since:     base,
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("ListEventSummaries(filtered) error = %v", err)
	}
	if got, want := len(summaries), 1; got != want {
		t.Fatalf("len(summaries) = %d, want %d", got, want)
	}
	if got, want := summaries[0].AgentName, "coder"; got != want {
		t.Fatalf("summaries[0].AgentName = %q, want %q", got, want)
	}

	if err := globalDB.UpdateTokenStats(testutil.Context(t), TokenStatsUpdate{
		SessionID: "sess-defaults",
		AgentName: "coder",
	}); err != nil {
		t.Fatalf("UpdateTokenStats(default turns) error = %v", err)
	}
	stats, err := globalDB.ListTokenStats(testutil.Context(t), TokenStatsQuery{
		SessionID: "sess-defaults",
		AgentName: "coder",
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("ListTokenStats(filtered) error = %v", err)
	}
	if got, want := len(stats), 1; got != want {
		t.Fatalf("len(stats) = %d, want %d", got, want)
	}
	if got, want := stats[0].TurnCount, int64(1); got != want {
		t.Fatalf("stats[0].TurnCount = %d, want %d", got, want)
	}

	if err := globalDB.WritePermissionLog(testutil.Context(t), PermissionLogEntry{
		SessionID:  "sess-defaults",
		AgentName:  "coder",
		Action:     "bash",
		Resource:   "/tmp/a",
		Decision:   "allow",
		PolicyUsed: "approve-reads",
	}); err != nil {
		t.Fatalf("WritePermissionLog(default timestamp) error = %v", err)
	}
	if err := globalDB.WritePermissionLog(testutil.Context(t), PermissionLogEntry{
		SessionID:  "sess-reviewer",
		AgentName:  "reviewer",
		Action:     "bash",
		Resource:   "/tmp/b",
		Decision:   "deny",
		PolicyUsed: "sandbox",
		Timestamp:  base.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("WritePermissionLog(explicit timestamp) error = %v", err)
	}

	entries, err := globalDB.ListPermissionLog(testutil.Context(t), PermissionLogQuery{
		AgentName: "coder",
		Decision:  "allow",
		Since:     base,
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("ListPermissionLog(filtered) error = %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}
	if got, want := entries[0].Decision, "allow"; got != want {
		t.Fatalf("entries[0].Decision = %q, want %q", got, want)
	}
}

func TestGlobalDBMigrationHelpers(t *testing.T) {
	t.Parallel()

	db, err := store.OpenSQLiteDatabase(
		testutil.Context(t),
		filepath.Join(t.TempDir(), GlobalDatabaseName),
		func(ctx context.Context, db *sql.DB) error {
			return store.EnsureSchema(ctx, db, []string{
				`CREATE TABLE IF NOT EXISTS workspaces (
				id TEXT PRIMARY KEY,
				root_dir TEXT NOT NULL,
				name TEXT NOT NULL
			);`,
				`INSERT INTO workspaces (id, root_dir, name) VALUES ('ws-1', '/tmp/ws-1', 'alpha');`,
			})
		},
	)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if exists, err := tableExists(testutil.Context(t), db, "workspaces"); err != nil || !exists {
		t.Fatalf("tableExists(workspaces) = (%v, %v), want (true, nil)", exists, err)
	}
	if exists, err := tableExists(testutil.Context(t), db, "missing_table"); err != nil || exists {
		t.Fatalf("tableExists(missing_table) = (%v, %v), want (false, nil)", exists, err)
	}

	columns, err := tableColumns(testutil.Context(t), db, "workspaces")
	if err != nil {
		t.Fatalf("tableColumns() error = %v", err)
	}
	for _, column := range []string{"id", "root_dir", "name"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("tableColumns() missing %q in %#v", column, columns)
		}
	}

	rootToID, err := loadWorkspaceIDsByRootDir(testutil.Context(t), db)
	if err != nil {
		t.Fatalf("loadWorkspaceIDsByRootDir() error = %v", err)
	}
	if got, want := rootToID["/tmp/ws-1"], "ws-1"; got != want {
		t.Fatalf("loadWorkspaceIDsByRootDir()[/tmp/ws-1] = %q, want %q", got, want)
	}

	names, err := loadWorkspaceNames(testutil.Context(t), db)
	if err != nil {
		t.Fatalf("loadWorkspaceNames() error = %v", err)
	}
	if _, ok := names["alpha"]; !ok {
		t.Fatalf("loadWorkspaceNames() missing alpha in %#v", names)
	}

	if got := coalesceTimestamp(" 2026-04-04T12:00:00.000000000Z "); got != "2026-04-04T12:00:00.000000000Z" {
		t.Fatalf("coalesceTimestamp(non-empty) = %q", got)
	}
	if got := coalesceTimestamp("   "); got == "" {
		t.Fatal("coalesceTimestamp(blank) = empty, want generated timestamp")
	}
	if got := nullStringValue(sql.NullString{}); got != nil {
		t.Fatalf("nullStringValue(invalid) = %#v, want nil", got)
	}
	if got := nullStringValue(sql.NullString{String: "  value  ", Valid: true}); got != "value" {
		t.Fatalf("nullStringValue(valid) = %#v, want value", got)
	}
	if got, want := sessionsDirForDatabasePath("/tmp/state/global.db"), "/tmp/state/sessions"; got != want {
		t.Fatalf("sessionsDirForDatabasePath() = %q, want %q", got, want)
	}

	migrationDB, err := store.OpenSQLiteDatabase(
		testutil.Context(t),
		filepath.Join(t.TempDir(), "migration.db"),
		func(ctx context.Context, db *sql.DB) error {
			return store.EnsureSchema(ctx, db, []string{
				`CREATE TABLE IF NOT EXISTS sessions (
				id TEXT PRIMARY KEY,
				name TEXT,
				agent_name TEXT NOT NULL,
				workspace TEXT NOT NULL,
				session_type TEXT NOT NULL,
				state TEXT NOT NULL,
				acp_session_id TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			);`,
				`INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, acp_session_id, created_at, updated_at)
			 VALUES ('sess-1', 'alpha', 'coder', '/tmp/ws-legacy', 'user', 'active', NULL, '2026-04-04T12:00:00.000000000Z', '2026-04-04T12:05:00.000000000Z');`,
				`INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, acp_session_id, created_at, updated_at)
			 VALUES ('sess-2', 'beta', 'coder', '/tmp/ws-legacy', 'user', 'active', NULL, '2026-04-04T11:00:00.000000000Z', '2026-04-04T12:10:00.000000000Z');`,
			})
		},
	)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase(migration) error = %v", err)
	}
	t.Cleanup(func() { _ = migrationDB.Close() })

	legacySessions, seeds, err := loadLegacySessions(testutil.Context(t), migrationDB)
	if err != nil {
		t.Fatalf("loadLegacySessions() error = %v", err)
	}
	if got, want := len(legacySessions), 2; got != want {
		t.Fatalf("len(legacySessions) = %d, want %d", got, want)
	}
	seed, ok := seeds["/tmp/ws-legacy"]
	if !ok {
		t.Fatalf("loadLegacySessions() missing workspace seed: %#v", seeds)
	}
	if got, want := seed.createdAt, "2026-04-04T11:00:00.000000000Z"; got != want {
		t.Fatalf("seed.createdAt = %q, want %q", got, want)
	}
	if got, want := seed.updatedAt, "2026-04-04T12:10:00.000000000Z"; got != want {
		t.Fatalf("seed.updatedAt = %q, want %q", got, want)
	}

	tx, err := migrationDB.BeginTx(testutil.Context(t), nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	if err := createMigratedGlobalTables(testutil.Context(t), tx); err != nil {
		_ = tx.Rollback()
		t.Fatalf("createMigratedGlobalTables() error = %v", err)
	}
	checkForeignKey := func(table string) {
		rows, queryErr := tx.QueryContext(testutil.Context(t), `PRAGMA foreign_key_list(`+table+`)`)
		if queryErr != nil {
			t.Fatalf("PRAGMA foreign_key_list(%s) error = %v", table, queryErr)
		}
		defer func() { _ = rows.Close() }()

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
		if !rows.Next() {
			t.Fatalf("foreign_key_list(%s) returned no rows", table)
		}
		if err := rows.Scan(&id, &seq, &refTable, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatalf("Scan(foreign_key_list %s) error = %v", table, err)
		}
		if refTable != "sessions_new" {
			t.Fatalf("foreign key table for %s = %q, want sessions_new", table, refTable)
		}
	}
	checkForeignKey("event_summaries_new")
	checkForeignKey("token_stats_new")
	checkForeignKey("permission_log_new")
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
}

func TestListEventSummariesPreservesHarnessFiltersAndRecentOrdering(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	base := time.Date(2026, 4, 18, 13, 0, 0, 0, time.UTC)

	workspaceID := registerWorkspaceForGlobalTests(
		t,
		globalDB,
		"harness-observe-workspace",
		filepath.Join(t.TempDir(), "harness-observe-workspace"),
	)
	if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          "sess-harness",
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		State:       "active",
		CreatedAt:   base,
		UpdatedAt:   base,
	}); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	summaries := []EventSummary{
		{
			SessionID: "sess-harness",
			Type:      "harness.context_resolved",
			AgentName: "coder",
			Summary:   "surface=startup sections=memory|skills|network",
			Timestamp: base,
		},
		{
			SessionID: "sess-harness",
			Type:      "harness.section_selected",
			AgentName: "coder",
			Summary:   "selected=memory|skills|network count=3",
			Timestamp: base.Add(time.Second),
		},
		{
			SessionID: "sess-harness",
			Type:      "harness.augmenter_applied",
			AgentName: "coder",
			Summary:   "augmenter=durable_memory outcome=blank",
			Timestamp: base.Add(2 * time.Second),
		},
	}
	for _, summary := range summaries {
		if err := globalDB.WriteEventSummary(testutil.Context(t), summary); err != nil {
			t.Fatalf("WriteEventSummary(%q) error = %v", summary.Type, err)
		}
	}

	filtered, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{
		SessionID: "sess-harness",
		Type:      "harness.context_resolved",
	})
	if err != nil {
		t.Fatalf("ListEventSummaries(filtered) error = %v", err)
	}
	if got, want := len(filtered), 1; got != want {
		t.Fatalf("len(filtered) = %d, want %d", got, want)
	}
	if got, want := filtered[0].Summary, summaries[0].Summary; got != want {
		t.Fatalf("filtered[0].Summary = %q, want %q", got, want)
	}

	recent, err := globalDB.ListEventSummaries(testutil.Context(t), EventSummaryQuery{
		SessionID: "sess-harness",
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("ListEventSummaries(recent) error = %v", err)
	}
	if got, want := len(recent), 2; got != want {
		t.Fatalf("len(recent) = %d, want %d", got, want)
	}
	if got, want := recent[0].Type, "harness.section_selected"; got != want {
		t.Fatalf("recent[0].Type = %q, want %q", got, want)
	}
	if got, want := recent[1].Type, "harness.augmenter_applied"; got != want {
		t.Fatalf("recent[1].Type = %q, want %q", got, want)
	}
	if !recent[0].Timestamp.Before(recent[1].Timestamp) {
		t.Fatalf("recent timestamps = [%v %v], want ascending recent order", recent[0].Timestamp, recent[1].Timestamp)
	}
}

func TestMigrateBridgeInstanceColumnsNoopAndIdempotent(t *testing.T) {
	t.Parallel()

	db, err := store.OpenSQLiteDatabase(testutil.Context(t), filepath.Join(t.TempDir(), "bridge-columns.db"), nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := migrateBridgeInstanceColumns(testutil.Context(t), db); err != nil {
		t.Fatalf("migrateBridgeInstanceColumns(no table) error = %v", err)
	}

	if _, err := db.ExecContext(testutil.Context(t), `CREATE TABLE bridge_instances (
		id TEXT PRIMARY KEY,
		scope TEXT NOT NULL,
		workspace_id TEXT,
		platform TEXT NOT NULL,
		extension_name TEXT NOT NULL,
		display_name TEXT NOT NULL,
		enabled BOOLEAN NOT NULL DEFAULT 1,
		status TEXT NOT NULL,
		routing_policy TEXT NOT NULL,
		delivery_defaults TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		dm_policy TEXT NOT NULL DEFAULT 'open',
		provider_config TEXT,
		degradation_reason TEXT,
		degradation_message TEXT
	)`); err != nil {
		t.Fatalf("create bridge_instances error = %v", err)
	}

	if err := migrateBridgeInstanceColumns(testutil.Context(t), db); err != nil {
		t.Fatalf("migrateBridgeInstanceColumns(existing columns) error = %v", err)
	}

	columns, err := tableColumns(testutil.Context(t), db, "bridge_instances")
	if err != nil {
		t.Fatalf("tableColumns(bridge_instances) error = %v", err)
	}
	for _, column := range []string{"dm_policy", "provider_config", "degradation_reason", "degradation_message"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("tableColumns(bridge_instances) missing %q in %#v", column, columns)
		}
	}
}

func TestMigrateWorkspaceColumnsAddsEnvironmentRef(t *testing.T) {
	t.Parallel()

	db, err := store.OpenSQLiteDatabase(testutil.Context(t), filepath.Join(t.TempDir(), "workspace-columns.db"), nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := migrateWorkspaceColumns(testutil.Context(t), db); err != nil {
		t.Fatalf("migrateWorkspaceColumns(no table) error = %v", err)
	}

	if _, err := db.ExecContext(testutil.Context(t), `CREATE TABLE workspaces (
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

	if err := migrateWorkspaceColumns(testutil.Context(t), db); err != nil {
		t.Fatalf("migrateWorkspaceColumns(add column) error = %v", err)
	}
	if err := migrateWorkspaceColumns(testutil.Context(t), db); err != nil {
		t.Fatalf("migrateWorkspaceColumns(idempotent) error = %v", err)
	}

	columns, err := tableColumns(testutil.Context(t), db, "workspaces")
	if err != nil {
		t.Fatalf("tableColumns(workspaces) error = %v", err)
	}
	if _, ok := columns["environment_ref"]; !ok {
		t.Fatalf("tableColumns(workspaces) missing environment_ref in %#v", columns)
	}
}

func TestMigrateGlobalSchemaUpgradesLegacyBridgeAndExtensionTables(t *testing.T) {
	t.Parallel()

	db, err := store.OpenSQLiteDatabase(testutil.Context(t), filepath.Join(t.TempDir(), "legacy-global.db"), nil)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, statement := range []string{
		`CREATE TABLE sessions (
			id TEXT PRIMARY KEY,
			name TEXT,
			agent_name TEXT NOT NULL,
			workspace_id TEXT NOT NULL,
			session_type TEXT NOT NULL DEFAULT 'user',
			state TEXT NOT NULL,
			acp_session_id TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE workspaces (
			id TEXT PRIMARY KEY,
			root_dir TEXT NOT NULL UNIQUE,
			add_dirs TEXT NOT NULL DEFAULT '[]',
			name TEXT NOT NULL UNIQUE,
			default_agent TEXT DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE extensions (
			name TEXT PRIMARY KEY,
			version TEXT NOT NULL,
			source TEXT NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT 1,
			manifest_path TEXT NOT NULL,
			installed_at TEXT NOT NULL,
			capabilities TEXT NOT NULL DEFAULT '{}',
			actions TEXT NOT NULL DEFAULT '{}',
			checksum TEXT NOT NULL
		)`,
		`CREATE TABLE bridge_instances (
			id TEXT PRIMARY KEY,
			scope TEXT NOT NULL,
			workspace_id TEXT,
			platform TEXT NOT NULL,
			extension_name TEXT NOT NULL,
			display_name TEXT NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT 1,
			status TEXT NOT NULL,
			routing_policy TEXT NOT NULL,
			delivery_defaults TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE network_audit_log (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			direction TEXT NOT NULL,
			kind TEXT NOT NULL,
			channel TEXT NOT NULL,
			peer_from TEXT NOT NULL,
			peer_to TEXT,
			message_id TEXT NOT NULL,
			reason TEXT,
			size INTEGER NOT NULL,
			timestamp TEXT NOT NULL
		)`,
	} {
		if _, err := db.ExecContext(testutil.Context(t), statement); err != nil {
			t.Fatalf("ExecContext(%q) error = %v", statement, err)
		}
	}

	if err := migrateGlobalSchema(testutil.Context(t), db); err != nil {
		t.Fatalf("migrateGlobalSchema() error = %v", err)
	}

	for table, expected := range map[string][]string{
		"sessions":         {"stop_reason", "stop_detail", "channel"},
		"workspaces":       {"environment_ref"},
		"extensions":       {"registry_slug", "registry_name", "remote_version"},
		"bridge_instances": {"source", "dm_policy", "provider_config", "degradation_reason", "degradation_message"},
	} {
		columns, err := tableColumns(testutil.Context(t), db, table)
		if err != nil {
			t.Fatalf("tableColumns(%s) error = %v", table, err)
		}
		for _, column := range expected {
			if _, ok := columns[column]; !ok {
				t.Fatalf("tableColumns(%s) missing %q in %#v", table, column, columns)
			}
		}
	}
}

func TestGlobalDBLegacySessionMetaHelpers(t *testing.T) {
	t.Parallel()

	metaPath := filepath.Join(t.TempDir(), store.SessionMetaName)
	raw := map[string]any{
		"id":           "sess-legacy",
		"name":         "legacy",
		"agent_name":   "coder",
		"workspace":    "/tmp/ws-legacy",
		"session_type": "user",
		"state":        "active",
		"created_at":   time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC),
		"updated_at":   time.Date(2026, 4, 4, 12, 1, 0, 0, time.UTC),
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if err := os.WriteFile(metaPath, data, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	needsRewrite, meta, err := loadReconciledLegacySessionMeta(metaPath, map[string]string{"/tmp/ws-legacy": "ws-123"})
	if err != nil {
		t.Fatalf("loadReconciledLegacySessionMeta() error = %v", err)
	}
	if !needsRewrite {
		t.Fatal("loadReconciledLegacySessionMeta() needsRewrite = false, want true")
	}
	if got, want := meta.WorkspaceID, "ws-123"; got != want {
		t.Fatalf("meta.WorkspaceID = %q, want %q", got, want)
	}

	db, err := store.OpenSQLiteDatabase(
		testutil.Context(t),
		filepath.Join(t.TempDir(), GlobalDatabaseName),
		func(ctx context.Context, db *sql.DB) error {
			return store.EnsureSchema(ctx, db, []string{
				`CREATE TABLE IF NOT EXISTS workspaces (id TEXT PRIMARY KEY, root_dir TEXT NOT NULL, name TEXT NOT NULL);`,
				`INSERT INTO workspaces (id, root_dir, name) VALUES ('ws-123', '/tmp/ws-legacy', 'legacy');`,
			})
		},
	)
	if err != nil {
		t.Fatalf("OpenSQLiteDatabase(reconcile) error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := reconcileLegacySessionMetaWorkspaceIDs(testutil.Context(t), db, ""); err != nil {
		t.Fatalf("reconcileLegacySessionMetaWorkspaceIDs(empty dir) error = %v", err)
	}

	if err := os.WriteFile(metaPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile(invalid json) error = %v", err)
	}
	needsRewrite, _, err = loadReconciledLegacySessionMeta(metaPath, map[string]string{"/tmp/ws-legacy": "ws-123"})
	if err != nil {
		t.Fatalf("loadReconciledLegacySessionMeta(invalid json) error = %v", err)
	}
	if needsRewrite {
		t.Fatal("loadReconciledLegacySessionMeta(invalid json) needsRewrite = true, want false")
	}
}

func TestGlobalDBWorkspaceHelperUtilities(t *testing.T) {
	t.Parallel()

	dirs, err := decodeWorkspaceDirs(`[" /tmp/a ","","/tmp/b"]`)
	if err != nil {
		t.Fatalf("decodeWorkspaceDirs(valid) error = %v", err)
	}
	if !testutil.EqualStringSlices(dirs, []string{"/tmp/a", "/tmp/b"}) {
		t.Fatalf("decodeWorkspaceDirs(valid) = %#v", dirs)
	}
	if _, err := decodeWorkspaceDirs(`{`); err == nil {
		t.Fatal("decodeWorkspaceDirs(invalid) error = nil, want non-nil")
	}

	if err := mapWorkspaceConstraintError(nil); err != nil {
		t.Fatalf("mapWorkspaceConstraintError(nil) = %v, want nil", err)
	}
	if err := mapWorkspaceConstraintError(
		errors.New("UNIQUE constraint failed: workspaces.root_dir"),
	); !errors.Is(
		err,
		aghworkspace.ErrWorkspacePathTaken,
	) {
		t.Fatalf("mapWorkspaceConstraintError(root_dir) = %v, want ErrWorkspacePathTaken", err)
	}
	if err := mapWorkspaceConstraintError(
		errors.New("UNIQUE constraint failed: workspaces.name"),
	); !errors.Is(
		err,
		aghworkspace.ErrWorkspaceNameTaken,
	) {
		t.Fatalf("mapWorkspaceConstraintError(name) = %v, want ErrWorkspaceNameTaken", err)
	}
	if err := mapWorkspaceConstraintError(
		errors.New("FOREIGN KEY constraint failed"),
	); !errors.Is(
		err,
		aghworkspace.ErrWorkspaceHasSessions,
	) {
		t.Fatalf("mapWorkspaceConstraintError(fk) = %v, want ErrWorkspaceHasSessions", err)
	}
}
