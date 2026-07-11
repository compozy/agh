package globaldb

import (
	"database/sql"
	"fmt"
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

	t.Run("Should use the covering recent catalog index for workspace and state filters", func(t *testing.T) {
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
		if !strings.Contains(plan, sessionCatalogRecentIndex) {
			t.Fatalf("EXPLAIN QUERY PLAN detail = %q, want %s", plan, sessionCatalogRecentIndex)
		}
		assertIndexAbsent(t, globalDB.db, "idx_sessions_workspace_state")

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

func TestPageSessionsVisibilityExclusion(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve normal sessions with null spawn roles", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"workspace-visible-sessions",
			filepath.Join(t.TempDir(), "workspace-visible-sessions"),
		)
		baseAt := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
		normal := sessionInfoForWorkspaceStateIndexTest(
			"sess-normal",
			workspaceID,
			globalDBSessionStateActive,
			baseAt,
		)
		internal := sessionInfoForWorkspaceStateIndexTest(
			"sess-memory",
			workspaceID,
			globalDBSessionStateActive,
			baseAt,
		)
		internal.Lineage = &store.SessionLineage{SpawnRole: "memory-extractor"}
		dream := sessionInfoForWorkspaceStateIndexTest(
			"sess-dream",
			workspaceID,
			globalDBSessionStateActive,
			baseAt,
		)
		dream.SessionType = "dream"
		overlaid := sessionInfoForWorkspaceStateIndexTest(
			"sess-active-overlay",
			workspaceID,
			globalDBSessionStateActive,
			baseAt,
		)
		for _, sessionInfo := range []store.SessionInfo{normal, internal, dream, overlaid} {
			if err := globalDB.RegisterSession(ctx, sessionInfo); err != nil {
				t.Fatalf("RegisterSession(%q) error = %v", sessionInfo.ID, err)
			}
		}

		page, err := globalDB.PageSessions(ctx, store.SessionCatalogPageQuery{
			WorkspaceID:         workspaceID,
			Sort:                "recent",
			Limit:               10,
			ExcludeIDs:          []string{overlaid.ID},
			ExcludeSessionTypes: []string{"dream"},
			ExcludeSpawnRoles:   []string{"memory-extractor"},
		})
		if err != nil {
			t.Fatalf("PageSessions() error = %v", err)
		}
		got := sessionIDsForWorkspaceStateIndexTest(page.Sessions)
		want := []string{"sess-normal"}
		if !slices.Equal(got, want) {
			t.Fatalf("PageSessions() ids = %#v, want %#v", got, want)
		}
		if page.Total != 1 {
			t.Fatalf("PageSessions().Total = %d, want 1", page.Total)
		}
	})

	t.Run("Should count only resumable sessions before the page cut", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		now := time.Date(2026, 7, 10, 12, 30, 0, 0, time.UTC)
		globalDB.now = func() time.Time { return now }
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"workspace-resumable-page",
			filepath.Join(t.TempDir(), "workspace-resumable-page"),
		)
		available := sessionInfoForWorkspaceStateIndexTest(
			"sess-available",
			workspaceID,
			globalDBSessionStateActive,
			now,
		)
		locked := sessionInfoForWorkspaceStateIndexTest(
			"sess-locked",
			workspaceID,
			globalDBSessionStateActive,
			now,
		)
		for _, sessionInfo := range []store.SessionInfo{available, locked} {
			if err := globalDB.RegisterSession(ctx, sessionInfo); err != nil {
				t.Fatalf("RegisterSession(%q) error = %v", sessionInfo.ID, err)
			}
		}
		if _, err := globalDB.AttachSession(ctx, store.SessionAttachRequest{
			SessionID:  locked.ID,
			AttachedTo: "uds:test",
			Now:        now,
			TTL:        time.Hour,
		}); err != nil {
			t.Fatalf("AttachSession(locked) error = %v", err)
		}

		page, err := globalDB.PageSessions(ctx, store.SessionCatalogPageQuery{
			WorkspaceID: workspaceID,
			Resumable:   true,
			Sort:        "last_activity",
			Limit:       10,
		})
		if err != nil {
			t.Fatalf("PageSessions(resumable) error = %v", err)
		}
		got := sessionIDsForWorkspaceStateIndexTest(page.Sessions)
		want := []string{available.ID}
		if !slices.Equal(got, want) {
			t.Fatalf("PageSessions(resumable) ids = %#v, want %#v", got, want)
		}
		if page.Total != 1 {
			t.Fatalf("PageSessions(resumable).Total = %d, want 1", page.Total)
		}
	})
}

func TestPageSessionsStableKeyset(t *testing.T) {
	t.Parallel()

	t.Run("Should walk matches after an anchor mutation without gaps or duplicates", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		globalDB := openTestGlobalDB(t)
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"workspace-session-page",
			filepath.Join(t.TempDir(), "workspace-session-page"),
		)
		foreignWorkspaceID := registerWorkspaceForGlobalTests(
			t,
			globalDB,
			"workspace-session-page-foreign",
			filepath.Join(t.TempDir(), "workspace-session-page-foreign"),
		)
		baseAt := time.Date(2026, 7, 10, 14, 0, 0, 0, time.UTC)
		matching := make([]store.SessionInfo, 0, 5)
		for index := range 5 {
			info := sessionInfoForWorkspaceStateIndexTest(
				fmt.Sprintf("sess-match-%d", index),
				workspaceID,
				globalDBSessionStateActive,
				baseAt,
			)
			info.Name = fmt.Sprintf("Match %d", index)
			info.UpdatedAt = baseAt.Add(time.Duration(index) * time.Minute)
			matching = append(matching, info)
		}
		nonMatching := []store.SessionInfo{
			sessionInfoForWorkspaceStateIndexTest(
				"sess-foreign",
				foreignWorkspaceID,
				globalDBSessionStateActive,
				baseAt,
			),
			sessionInfoForWorkspaceStateIndexTest(
				"sess-stopped",
				workspaceID,
				globalDBSessionStateStopped,
				baseAt,
			),
			sessionInfoForWorkspaceStateIndexTest(
				"sess-other-agent",
				workspaceID,
				globalDBSessionStateActive,
				baseAt,
			),
		}
		nonMatching[0].Name = "Match foreign"
		nonMatching[1].Name = "Match stopped"
		nonMatching[2].Name = "Match other agent"
		nonMatching[2].AgentName = "reviewer"
		for _, info := range append(matching, nonMatching...) {
			if err := globalDB.RegisterSession(ctx, info); err != nil {
				t.Fatalf("RegisterSession(%q) error = %v", info.ID, err)
			}
		}

		query := store.SessionCatalogPageQuery{
			WorkspaceID: workspaceID,
			State:       globalDBSessionStateActive,
			AgentName:   "coder",
			Search:      "MATCH",
			Sort:        "recent",
			Limit:       2,
		}
		first, err := globalDB.PageSessions(ctx, query)
		if err != nil {
			t.Fatalf("PageSessions(first) error = %v", err)
		}
		if first.Total != len(matching) || len(first.Sessions) != 2 {
			t.Fatalf(
				"PageSessions(first) = total %d rows %d, want %d/2",
				first.Total,
				len(first.Sessions),
				len(matching),
			)
		}
		anchor := first.Sessions[len(first.Sessions)-1]
		query.After = &store.SessionCatalogPosition{
			PrimaryAt:   anchor.UpdatedAt,
			SecondaryAt: anchor.CreatedAt,
			CreatedAt:   anchor.CreatedAt,
			ID:          anchor.ID,
		}
		if err := globalDB.UpdateSessionState(ctx, store.SessionStateUpdate{
			ID:        anchor.ID,
			State:     anchor.State,
			UpdatedAt: baseAt.Add(24 * time.Hour),
		}); err != nil {
			t.Fatalf("UpdateSessionState(anchor) error = %v", err)
		}

		seen := sessionIDsForWorkspaceStateIndexTest(first.Sessions)
		for {
			page, pageErr := globalDB.PageSessions(ctx, query)
			if pageErr != nil {
				t.Fatalf("PageSessions(next) error = %v", pageErr)
			}
			if page.Total != len(matching) {
				t.Fatalf("PageSessions(next).Total = %d, want %d", page.Total, len(matching))
			}
			if len(page.Sessions) == 0 {
				break
			}
			seen = append(seen, sessionIDsForWorkspaceStateIndexTest(page.Sessions)...)
			last := page.Sessions[len(page.Sessions)-1]
			query.After = &store.SessionCatalogPosition{
				PrimaryAt:   last.UpdatedAt,
				SecondaryAt: last.CreatedAt,
				CreatedAt:   last.CreatedAt,
				ID:          last.ID,
			}
		}
		if len(seen) != len(matching) {
			t.Fatalf("walked session ids = %#v, want %d rows", seen, len(matching))
		}
		seenSet := make(map[string]struct{}, len(seen))
		for _, id := range seen {
			if _, duplicate := seenSet[id]; duplicate {
				t.Fatalf("walked session ids contain duplicate %q: %#v", id, seen)
			}
			seenSet[id] = struct{}{}
		}
	})
}

func TestSessionsWorkspaceStateIndexMigration(t *testing.T) {
	t.Parallel()

	t.Run("Should replace the legacy workspace state index on fresh and upgraded DB", func(t *testing.T) {
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
		legacyIndexMigrationIndex := migrationIndexByName(t, "add_sessions_workspace_state_index")
		catalogIndexMigrationIndex := migrationIndexByName(t, "add_session_catalog_paging_indexes")
		cleanupMigrationIndex := migrationIndexByName(t, "drop_redundant_sessions_workspace_state_index")
		indexMigrationPath := []store.Migration{
			globalSchemaMigrations[0],
			globalSchemaMigrations[epochMigrationIndex],
			globalSchemaMigrations[legacyIndexMigrationIndex],
			globalSchemaMigrations[catalogIndexMigrationIndex],
			globalSchemaMigrations[cleanupMigrationIndex],
		}
		if err := store.RunMigrations(ctx, upgradeSeed, indexMigrationPath[:2]); err != nil {
			t.Fatalf("RunMigrations(pre-legacy-index prefix) error = %v", err)
		}
		assertIndexAbsent(t, upgradeSeed, "idx_sessions_workspace_state")
		if err := store.RunMigrations(ctx, upgradeSeed, indexMigrationPath[:3]); err != nil {
			t.Fatalf("RunMigrations(legacy index migration) error = %v", err)
		}
		assertIndexesPresent(t, upgradeSeed, "sessions", "idx_sessions_workspace_state")
		legacySQL := schemaObjectSQL(t, upgradeSeed, "index", "idx_sessions_workspace_state")
		if !strings.Contains(legacySQL, "ON sessions(workspace_id, state)") {
			t.Fatalf("legacy index SQL = %q, want sessions(workspace_id, state)", legacySQL)
		}
		if err := store.RunMigrations(ctx, upgradeSeed, indexMigrationPath[:4]); err != nil {
			t.Fatalf("RunMigrations(catalog index migration) error = %v", err)
		}
		assertIndexesPresent(t, upgradeSeed, "sessions", "idx_sessions_workspace_state", sessionCatalogRecentIndex)
		if err := store.RunMigrations(ctx, upgradeSeed, indexMigrationPath); err != nil {
			t.Fatalf("RunMigrations(index cleanup migration) error = %v", err)
		}

		assertIndexAbsent(t, freshDB.db, "idx_sessions_workspace_state")
		assertIndexAbsent(t, upgradeSeed, "idx_sessions_workspace_state")
		assertIndexesPresent(t, freshDB.db, "sessions", sessionCatalogRecentIndex)
		assertIndexesPresent(t, upgradeSeed, "sessions", sessionCatalogRecentIndex)
	})
}

func TestSessionCatalogPagingIndexesFreshDB(t *testing.T) {
	t.Parallel()

	t.Run("Should create both paging indexes on a fresh database", func(t *testing.T) {
		t.Parallel()

		globalDB := openTestGlobalDB(t)
		assertIndexesPresent(
			t,
			globalDB.db,
			"sessions",
			sessionCatalogRecentIndex,
			sessionCatalogActivityIndex,
		)
	})
}

func TestSessionCatalogPagingIndexesReopenAfterRestart(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve sessions while upgrading an existing database", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		path := filepath.Join(t.TempDir(), GlobalDatabaseName)
		db, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase() error = %v", err)
		}
		migrationIndex := migrationIndexByName(t, sessionCatalogPagingMigration.Name)
		if err := store.RunMigrations(ctx, db, globalSchemaMigrations[:migrationIndex]); err != nil {
			t.Fatalf("RunMigrations(previous) error = %v", err)
		}
		legacy := &GlobalDB{db: db, path: path, now: func() time.Time { return time.Now().UTC() }}
		workspaceID := registerWorkspaceForGlobalTests(
			t,
			legacy,
			"workspace-preserved",
			filepath.Join(t.TempDir(), "workspace-preserved"),
		)
		preserved := sessionInfoForWorkspaceStateIndexTest(
			"sess-preserved",
			workspaceID,
			globalDBSessionStateStopped,
			time.Date(2026, 7, 10, 13, 0, 0, 0, time.UTC),
		)
		if err := legacy.RegisterSession(ctx, preserved); err != nil {
			t.Fatalf("RegisterSession() error = %v", err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("Close(previous database) error = %v", err)
		}

		reopened, err := store.OpenSQLiteDatabase(ctx, path, nil)
		if err != nil {
			t.Fatalf("OpenSQLiteDatabase(reopen) error = %v", err)
		}
		t.Cleanup(func() {
			if closeErr := reopened.Close(); closeErr != nil {
				t.Errorf("Close(reopened database) error = %v", closeErr)
			}
		})
		if err := store.RunMigrations(ctx, reopened, globalSchemaMigrations); err != nil {
			t.Fatalf("RunMigrations(upgrade) error = %v", err)
		}
		assertIndexesPresent(
			t,
			reopened,
			"sessions",
			sessionCatalogRecentIndex,
			sessionCatalogActivityIndex,
		)
		var found int
		if err := reopened.QueryRowContext(
			ctx,
			"SELECT COUNT(1) FROM sessions WHERE id = ?",
			preserved.ID,
		).Scan(&found); err != nil {
			t.Fatalf("query preserved session error = %v", err)
		}
		if found != 1 {
			t.Fatalf("preserved session count = %d, want 1", found)
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
