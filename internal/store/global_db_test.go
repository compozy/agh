package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/pedronauck/agh/internal/testutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func TestOpenGlobalDBCreatesSchemaAndEnablesWAL(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)

	assertTablesPresent(t, globalDB.db, "workspaces", "sessions", "event_summaries", "token_stats", "permission_log")
	assertJournalModeWAL(t, globalDB.db)
	assertSynchronousNormal(t, globalDB.db)
}

func TestGlobalDBRegisterUpdateAndListSessions(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	createdAt := time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC)
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "sess-global-workspace", filepath.Join(t.TempDir(), "workspace-global"))
	session := SessionInfo{
		ID:          "sess-global",
		Name:        "Alpha",
		AgentName:   "coder",
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
	if sessions[0].ACPSessionID == nil || *sessions[0].ACPSessionID != "acp-123" {
		t.Fatalf("sessions[0].ACPSessionID = %#v, want acp-123", sessions[0].ACPSessionID)
	}
}

func TestGlobalDBRegisterSessionDefaultsTypeToUser(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "sess-default-type-workspace", filepath.Join(t.TempDir(), "workspace-default-type"))
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
	if _, err := globalDB.GetWorkspace(testutil.Context(t), updated.ID); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("GetWorkspace(deleted) error = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestGlobalDBDeleteWorkspaceReturnsHasSessionsWhenReferenced(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "workspace-delete-guard", filepath.Join(t.TempDir(), "workspace-delete-guard"))
	if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          "sess-delete-guard",
		AgentName:   "coder",
		WorkspaceID: workspaceID,
		State:       "active",
		CreatedAt:   time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("RegisterSession() error = %v", err)
	}

	if err := globalDB.DeleteWorkspace(testutil.Context(t), workspaceID); !errors.Is(err, aghworkspace.ErrWorkspaceHasSessions) {
		t.Fatalf("DeleteWorkspace() error = %v, want ErrWorkspaceHasSessions", err)
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

	if _, err := globalDB.GetWorkspace(testutil.Context(t), "ws-missing"); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("GetWorkspace(missing) error = %v, want ErrWorkspaceNotFound", err)
	}
	if _, err := globalDB.GetWorkspaceByPath(testutil.Context(t), filepath.Join(t.TempDir(), "missing")); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
		t.Fatalf("GetWorkspaceByPath(missing) error = %v, want ErrWorkspaceNotFound", err)
	}
	if _, err := globalDB.GetWorkspaceByName(testutil.Context(t), "missing"); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
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
	if err := globalDB.DeleteWorkspace(testutil.Context(t), "ws-missing"); !errors.Is(err, aghworkspace.ErrWorkspaceNotFound) {
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
	if !strings.HasPrefix(workspaces[0].ID, "ws-") {
		t.Fatalf("workspaces[0].ID = %q, want ws- prefix", workspaces[0].ID)
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
				return globalDB.UpdateWorkspace(testutil.Context(t), aghworkspace.Workspace{RootDir: rootDir, Name: "missing-id"})
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
	workspaceID := registerWorkspaceForGlobalTests(t, globalDB, "session-workspace", filepath.Join(t.TempDir(), "session-workspace"))

	session := SessionInfo{
		ID:          "sess-workspace-id",
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
	if got, want := sessions[0].WorkspaceID, workspaceID; got != want {
		t.Fatalf("sessions[0].WorkspaceID = %q, want %q", got, want)
	}

	assertTableColumns(t, globalDB.db, "sessions", []string{"id", "name", "agent_name", "workspace_id", "session_type", "state", "acp_session_id", "created_at", "updated_at"})
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
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"sess-legacy-a", "A", "coder", rootA, "user", "active", formatTimestamp(time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)), formatTimestamp(time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert legacy session A error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"sess-legacy-b", "B", "reviewer", rootB, "dream", "stopped", formatTimestamp(time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)), formatTimestamp(time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert legacy session B error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO event_summaries (id, session_id, type, agent_name, summary, timestamp) VALUES (?, ?, ?, ?, ?, ?)`,
		"sum-legacy", "sess-legacy-a", "agent_message", "coder", "legacy summary", formatTimestamp(time.Date(2026, 4, 3, 10, 1, 0, 0, time.UTC)),
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

	assertTableColumns(t, globalDB.db, "sessions", []string{"id", "name", "agent_name", "workspace_id", "session_type", "state", "acp_session_id", "created_at", "updated_at"})
	assertTableColumns(t, globalDB.db, "workspaces", []string{"id", "root_dir", "add_dirs", "name", "default_agent", "created_at", "updated_at"})

	workspaces, err := globalDB.ListWorkspaces(ctx)
	if err != nil {
		t.Fatalf("ListWorkspaces() error = %v", err)
	}
	if got, want := len(workspaces), 2; got != want {
		t.Fatalf("len(workspaces) = %d, want %d", got, want)
	}
	if got, want := []string{workspaces[0].Name, workspaces[1].Name}, []string{"project", "project-2"}; !testutil.EqualStringSlices(got, want) {
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
	}

	summaries, err := globalDB.ListEventSummaries(ctx, EventSummaryQuery{SessionID: "sess-legacy-a"})
	if err != nil {
		t.Fatalf("ListEventSummaries() error = %v", err)
	}
	if got, want := len(summaries), 1; got != want {
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
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"sess-meta-legacy", "Legacy", "coder", rootDir, "user", "stopped", createdAt, createdAt,
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
		SessionID: "sess-summary",
		Type:      "agent_message",
		AgentName: "coder",
		Summary:   "assistant replied",
		Timestamp: time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
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
			ID:          "sess-keep",
			AgentName:   "coder",
			WorkspaceID: registerWorkspaceForGlobalTests(t, globalDB, "sess-keep-reconciled-workspace", filepath.Join(t.TempDir(), "sess-keep")),
			State:       "stopped",
			CreatedAt:   time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
		},
		{
			ID:          "sess-new",
			AgentName:   "reviewer",
			WorkspaceID: registerWorkspaceForGlobalTests(t, globalDB, "sess-new-reconciled-workspace", filepath.Join(t.TempDir(), "sess-new")),
			State:       "stopped",
			CreatedAt:   time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2026, 4, 3, 16, 0, 0, 0, time.UTC),
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
	for _, session := range sessions {
		stateByID[session.ID] = session.State
	}
	if stateByID["sess-new"] != "stopped" {
		t.Fatalf("stateByID[sess-new] = %q, want stopped", stateByID["sess-new"])
	}
	if stateByID["sess-orphan"] != "orphaned" {
		t.Fatalf("stateByID[sess-orphan] = %q, want orphaned", stateByID["sess-orphan"])
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

	assertTablesPresent(t, globalDB.db, "workspaces", "sessions", "event_summaries", "token_stats", "permission_log")

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

	globalDB, err := OpenGlobalDB(testutil.Context(t), filepath.Join(t.TempDir(), GlobalDatabaseName))
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := globalDB.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})
	return globalDB
}

func registerSessionForGlobalTests(t *testing.T, globalDB *GlobalDB, sessionID string) {
	t.Helper()

	now := time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC)
	if err := globalDB.RegisterSession(testutil.Context(t), SessionInfo{
		ID:          sessionID,
		AgentName:   "coder",
		WorkspaceID: registerWorkspaceForGlobalTests(t, globalDB, sessionID+"-workspace", filepath.Join(t.TempDir(), sessionID)),
		State:       "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("RegisterSession(%q) error = %v", sessionID, err)
	}
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

func assertWorkspaceEqual(t *testing.T, got aghworkspace.Workspace, want aghworkspace.Workspace) {
	t.Helper()

	if got.ID != want.ID ||
		got.RootDir != want.RootDir ||
		got.Name != want.Name ||
		got.DefaultAgent != want.DefaultAgent ||
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
		_ = rows.Close()
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
