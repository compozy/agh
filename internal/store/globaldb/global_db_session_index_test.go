package globaldb

import (
	"database/sql"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

func TestListSessionsWorkspaceStateIndex(t *testing.T) {
	t.Parallel()

	t.Run("Should use workspace state index for list filters", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		alphaWorkspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"workspace-alpha",
			filepath.Join(t.TempDir(), "workspace-alpha"),
		)
		betaWorkspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"workspace-beta",
			filepath.Join(t.TempDir(), "workspace-beta"),
		)
		baseAt := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
		for _, sessionInfo := range []store.SessionInfo{
			sessionInfoForWorkspaceStateIndexTest("sess-alpha-active", alphaWorkspaceID, globalDBSessionStateActive, baseAt),
			sessionInfoForWorkspaceStateIndexTest("sess-alpha-stopped", alphaWorkspaceID, globalDBSessionStateStopped, baseAt),
			sessionInfoForWorkspaceStateIndexTest("sess-beta-active", betaWorkspaceID, globalDBSessionStateActive, baseAt),
		} {
			if err := globalDB.RegisterSession(ctx, sessionInfo); err != nil {
				t.Fatalf("RegisterSession(%q) error = %v", sessionInfo.ID, err)
			}
		}

		plan := explainQueryPlan(
			t,
			globalDB.db,
			`SELECT id FROM sessions WHERE state = ? AND workspace_id = ? ORDER BY updated_at DESC, created_at DESC, id DESC`,
			globalDBSessionStateActive,
			alphaWorkspaceID,
		)
		if !strings.Contains(plan, "idx_sessions_workspace_state") {
			t.Fatalf("EXPLAIN QUERY PLAN detail = %q, want idx_sessions_workspace_state", plan)
		}

		sessions, err := globalDB.ListSessions(ctx, store.SessionListQuery{
			State:       globalDBSessionStateActive,
			WorkspaceID: alphaWorkspaceID,
		})
		if err != nil {
			t.Fatalf("ListSessions(workspace/state) error = %v", err)
		}
		got := sessionIDsForWorkspaceStateIndexTest(sessions)
		want := []string{"sess-alpha-active"}
		if !slices.Equal(got, want) {
			t.Fatalf("ListSessions(workspace/state) ids = %#v, want %#v", got, want)
		}
	})
}

func TestSessionsWorkspaceStateIndexMigration(t *testing.T) {
	t.Parallel()

	t.Run("Should converge sessions workspace state index on fresh and upgraded DB", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		freshDB := openTestGlobalDB(t)

		upgradePath := filepath.Join(t.TempDir(), "upgrade-"+GlobalDatabaseName)
		upgradeSeed, err := store.OpenSQLiteDatabase(ctx, upgradePath, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(upgrade seed) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := upgradeSeed.Close(); closeErr != nil {
				t.Errorf("Close(upgrade seed cleanup) error = %v", closeErr)
			}
		})
		epochMigrationIndex := migrationIndexByName(t, "add_session_transcript_epoch")
		indexMigrationIndex := migrationIndexByName(t, "add_sessions_workspace_state_index")
		indexMigrationPath := []store.Migration{
			globalSchemaMigrations[0],
			globalSchemaMigrations[epochMigrationIndex],
			globalSchemaMigrations[indexMigrationIndex],
		}
		if err := store.RunMigrations(ctx, upgradeSeed, indexMigrationPath[:2]); err != nil {
			t.Fatalf("RunMigrations(pre-index prefix) error = %v", err)
		}
		assertIndexAbsent(t, upgradeSeed, "idx_sessions_workspace_state")
		if err := store.RunMigrations(ctx, upgradeSeed, indexMigrationPath); err != nil {
			t.Fatalf("RunMigrations(index migration) error = %v", err)
		}

		assertIndexesPresent(t, freshDB.db, "sessions", "idx_sessions_workspace_state")
		assertIndexesPresent(t, upgradeSeed, "sessions", "idx_sessions_workspace_state")
		freshSQL := schemaObjectSQL(t, freshDB.db, "index", "idx_sessions_workspace_state")
		upgradedSQL := schemaObjectSQL(t, upgradeSeed, "index", "idx_sessions_workspace_state")
		if upgradedSQL != freshSQL {
			t.Fatalf("upgraded index SQL = %q, want fresh %q", upgradedSQL, freshSQL)
		}
		if !strings.Contains(freshSQL, "ON sessions(workspace_id, state)") {
			t.Fatalf("fresh index SQL = %q, want sessions(workspace_id, state)", freshSQL)
		}
	})
}

func sessionInfoForWorkspaceStateIndexTest(
	id string,
	workspaceID string,
	state string,
	baseAt time.Time,
) store.SessionInfo {
	return store.SessionInfo{
		ID:          id,
		Name:        id,
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		SessionType: "root",
		State:       state,
		CreatedAt:   baseAt,
		UpdatedAt:   baseAt.Add(time.Duration(len(id)) * time.Second),
	}
}

func explainQueryPlan(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()

	rows, err := db.QueryContext(testutil.Context(t), "EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN error = %v", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Fatalf("rows.Close() error = %v", closeErr)
		}
	}()

	var details []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatalf("rows.Scan() error = %v", err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err() error = %v", err)
	}
	return strings.Join(details, "\n")
}

func assertIndexAbsent(t *testing.T, db *sql.DB, index string) {
	t.Helper()

	var found int
	if err := db.QueryRowContext(
		testutil.Context(t),
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`,
		index,
	).Scan(&found); err != nil {
		t.Fatalf("query sqlite_master index %s error = %v", index, err)
	}
	if found != 0 {
		t.Fatalf("index %s exists before migration", index)
	}
}

func sessionIDsForWorkspaceStateIndexTest(sessions []store.SessionInfo) []string {
	ids := make([]string, 0, len(sessions))
	for _, sessionInfo := range sessions {
		ids = append(ids, sessionInfo.ID)
	}
	return ids
}
