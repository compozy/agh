package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func TestValidationHelpersAndPathUtilities(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 3, 20, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		validate  func() error
		wantError bool
	}{
		{
			name: "session event valid",
			validate: func() error {
				return (SessionEvent{TurnID: "turn-1", Type: "agent_message", AgentName: "coder"}).Validate()
			},
		},
		{
			name: "session event invalid",
			validate: func() error {
				return (SessionEvent{}).Validate()
			},
			wantError: true,
		},
		{
			name: "session event missing type",
			validate: func() error {
				return (SessionEvent{TurnID: "turn-1"}).Validate()
			},
			wantError: true,
		},
		{
			name: "session event missing agent",
			validate: func() error {
				return (SessionEvent{TurnID: "turn-1", Type: "agent_message"}).Validate()
			},
			wantError: true,
		},
		{
			name: "event query valid",
			validate: func() error {
				return (EventQuery{Limit: 1, AfterSequence: 1}).Validate()
			},
		},
		{
			name: "event query invalid",
			validate: func() error {
				return (EventQuery{Limit: -1}).Validate()
			},
			wantError: true,
		},
		{
			name: "token usage valid",
			validate: func() error {
				return (TokenUsage{TurnID: "turn-1"}).Validate()
			},
		},
		{
			name: "token usage invalid",
			validate: func() error {
				return (TokenUsage{}).Validate()
			},
			wantError: true,
		},
		{
			name: "session info valid",
			validate: func() error {
				return (SessionInfo{ID: "sess-1", AgentName: "coder", WorkspaceID: "ws-1", State: "active"}).Validate()
			},
		},
		{
			name: "session info invalid",
			validate: func() error {
				return (SessionInfo{}).Validate()
			},
			wantError: true,
		},
		{
			name: "session info missing agent",
			validate: func() error {
				return (SessionInfo{ID: "sess-1"}).Validate()
			},
			wantError: true,
		},
		{
			name: "session info missing workspace",
			validate: func() error {
				return (SessionInfo{ID: "sess-1", AgentName: "coder"}).Validate()
			},
			wantError: true,
		},
		{
			name: "session info missing state",
			validate: func() error {
				return (SessionInfo{ID: "sess-1", AgentName: "coder", WorkspaceID: "ws-1"}).Validate()
			},
			wantError: true,
		},
		{
			name: "session list query invalid",
			validate: func() error {
				return (SessionListQuery{Limit: -1}).Validate()
			},
			wantError: true,
		},
		{
			name: "session state update valid",
			validate: func() error {
				return (SessionStateUpdate{ID: "sess-1", State: "stopped"}).Validate()
			},
		},
		{
			name: "session state update invalid",
			validate: func() error {
				return (SessionStateUpdate{}).Validate()
			},
			wantError: true,
		},
		{
			name: "session state update missing state",
			validate: func() error {
				return (SessionStateUpdate{ID: "sess-1"}).Validate()
			},
			wantError: true,
		},
		{
			name: "event summary valid",
			validate: func() error {
				return (EventSummary{SessionID: "sess-1", Type: "agent_message", AgentName: "coder"}).Validate()
			},
		},
		{
			name: "event summary invalid",
			validate: func() error {
				return (EventSummary{}).Validate()
			},
			wantError: true,
		},
		{
			name: "event summary missing type",
			validate: func() error {
				return (EventSummary{SessionID: "sess-1"}).Validate()
			},
			wantError: true,
		},
		{
			name: "event summary missing agent",
			validate: func() error {
				return (EventSummary{SessionID: "sess-1", Type: "agent_message"}).Validate()
			},
			wantError: true,
		},
		{
			name: "event summary query invalid",
			validate: func() error {
				return (EventSummaryQuery{Limit: -1}).Validate()
			},
			wantError: true,
		},
		{
			name: "token stats update valid",
			validate: func() error {
				return (TokenStatsUpdate{SessionID: "sess-1", AgentName: "coder"}).Validate()
			},
		},
		{
			name: "token stats update invalid",
			validate: func() error {
				return (TokenStatsUpdate{}).Validate()
			},
			wantError: true,
		},
		{
			name: "token stats update missing agent",
			validate: func() error {
				return (TokenStatsUpdate{SessionID: "sess-1"}).Validate()
			},
			wantError: true,
		},
		{
			name: "token stats query invalid",
			validate: func() error {
				return (TokenStatsQuery{Limit: -1}).Validate()
			},
			wantError: true,
		},
		{
			name: "permission log entry valid",
			validate: func() error {
				return (PermissionLogEntry{
					SessionID:  "sess-1",
					AgentName:  "coder",
					Action:     "bash",
					Resource:   "/tmp",
					Decision:   "allow",
					PolicyUsed: "approve-reads",
				}).Validate()
			},
		},
		{
			name: "permission log entry invalid",
			validate: func() error {
				return (PermissionLogEntry{}).Validate()
			},
			wantError: true,
		},
		{
			name: "permission log entry missing agent",
			validate: func() error {
				return (PermissionLogEntry{SessionID: "sess-1"}).Validate()
			},
			wantError: true,
		},
		{
			name: "permission log entry missing action",
			validate: func() error {
				return (PermissionLogEntry{SessionID: "sess-1", AgentName: "coder"}).Validate()
			},
			wantError: true,
		},
		{
			name: "permission log entry missing resource",
			validate: func() error {
				return (PermissionLogEntry{SessionID: "sess-1", AgentName: "coder", Action: "bash"}).Validate()
			},
			wantError: true,
		},
		{
			name: "permission log entry missing decision",
			validate: func() error {
				return (PermissionLogEntry{SessionID: "sess-1", AgentName: "coder", Action: "bash", Resource: "/tmp"}).Validate()
			},
			wantError: true,
		},
		{
			name: "permission log entry missing policy",
			validate: func() error {
				return (PermissionLogEntry{SessionID: "sess-1", AgentName: "coder", Action: "bash", Resource: "/tmp", Decision: "allow"}).Validate()
			},
			wantError: true,
		},
		{
			name: "permission log query invalid",
			validate: func() error {
				return (PermissionLogQuery{Limit: -1}).Validate()
			},
			wantError: true,
		},
		{
			name: "session meta valid",
			validate: func() error {
				return (SessionMeta{
					ID:          "sess-meta",
					AgentName:   "coder",
					WorkspaceID: "ws-meta",
					State:       "active",
					CreatedAt:   now,
					UpdatedAt:   now,
				}).Validate()
			},
		},
		{
			name: "session meta invalid",
			validate: func() error {
				return (SessionMeta{}).Validate()
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validate()
			if tt.wantError && err == nil {
				t.Fatal("validate() error = nil, want non-nil")
			}
			if !tt.wantError && err != nil {
				t.Fatalf("validate() error = %v", err)
			}
		})
	}

	if got, want := SessionDBFile("/tmp/session-a"), filepath.Join("/tmp/session-a", SessionDatabaseName); got != want {
		t.Fatalf("SessionDBFile() = %q, want %q", got, want)
	}
	if got, want := SessionMetaFile("/tmp/session-a"), filepath.Join("/tmp/session-a", SessionMetaName); got != want {
		t.Fatalf("SessionMetaFile() = %q, want %q", got, want)
	}
}

func TestStoreHelpersAndErrorPaths(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 3, 20, 15, 0, 0, time.UTC)
	if got := normalizeTime(now.In(time.FixedZone("test", -3*60*60))); got.Location() != time.UTC {
		t.Fatalf("normalizeTime() location = %v, want UTC", got.Location())
	}
	if got := normalizeTime(time.Time{}); !got.IsZero() {
		t.Fatalf("normalizeTime(zero) = %v, want zero", got)
	}

	formatted := formatTimestamp(now)
	parsed, err := parseTimestamp(formatted)
	if err != nil {
		t.Fatalf("parseTimestamp() error = %v", err)
	}
	if !parsed.Equal(now.UTC()) {
		t.Fatalf("parseTimestamp() = %v, want %v", parsed, now.UTC())
	}
	if _, err := parseTimestamp("bad-timestamp"); err == nil {
		t.Fatal("parseTimestamp() error = nil, want non-nil")
	}

	if got := nullableString(""); got != nil {
		t.Fatalf("nullableString(\"\") = %#v, want nil", got)
	}
	if got := nullableString("value"); got != "value" {
		t.Fatalf("nullableString(value) = %#v, want value", got)
	}

	var nilString *string
	if got := nullableStringPointer(nilString); got != nil {
		t.Fatalf("nullableStringPointer(nil) = %#v, want nil", got)
	}
	value := "abc"
	if got := nullableStringPointer(&value); got != "abc" {
		t.Fatalf("nullableStringPointer(&value) = %#v, want abc", got)
	}
	if got := nullString(sql.NullString{}); got != nil {
		t.Fatalf("nullString(invalid) = %#v, want nil", got)
	}
	if got := nullString(sql.NullString{String: "value", Valid: true}); got == nil || *got != "value" {
		t.Fatalf("nullString(valid) = %#v, want value", got)
	}
	if got := nullInt64(sql.NullInt64{}); got != nil {
		t.Fatalf("nullInt64(invalid) = %#v, want nil", got)
	}
	if got := nullInt64(sql.NullInt64{Int64: 7, Valid: true}); got == nil || *got != 7 {
		t.Fatalf("nullInt64(valid) = %#v, want 7", got)
	}
	if got := nullFloat64(sql.NullFloat64{}); got != nil {
		t.Fatalf("nullFloat64(invalid) = %#v, want nil", got)
	}
	if got := nullFloat64(sql.NullFloat64{Float64: 1.25, Valid: true}); got == nil || *got != 1.25 {
		t.Fatalf("nullFloat64(valid) = %#v, want 1.25", got)
	}

	if got := newID("prefix"); got == "" || filepath.Base(got) != got {
		t.Fatalf("newID(prefix) = %q, want non-empty plain value", got)
	}
	if got := newID(""); got == "" {
		t.Fatal("newID(\"\") = empty, want non-empty")
	}

	if !shouldRecoverSQLite(errors.New("file is not a database")) {
		t.Fatal("shouldRecoverSQLite(not a database) = false, want true")
	}
	if shouldRecoverSQLite(errors.New("permission denied")) {
		t.Fatal("shouldRecoverSQLite(permission denied) = true, want false")
	}

	if err := checkpoint(testContext(t), nil); err != nil {
		t.Fatalf("checkpoint(nil) error = %v", err)
	}
	if _, err := openSQLiteDatabase(testContext(t), "", func(ctx context.Context, db *sql.DB) error {
		return ensureSchema(ctx, db, sessionSchemaStatements)
	}); err == nil {
		t.Fatal("openSQLiteDatabase(\"\") error = nil, want non-nil")
	}
}

func TestMetaReadWriteErrors(t *testing.T) {
	t.Parallel()

	if _, err := ReadSessionMeta(""); err == nil {
		t.Fatal("ReadSessionMeta(\"\") error = nil, want non-nil")
	}
	invalidPath := filepath.Join(t.TempDir(), SessionMetaName)
	if err := os.WriteFile(invalidPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := ReadSessionMeta(invalidPath); err == nil {
		t.Fatal("ReadSessionMeta(invalid JSON) error = nil, want non-nil")
	}

	if err := WriteSessionMeta("", SessionMeta{}); err == nil {
		t.Fatal("WriteSessionMeta(\"\") error = nil, want non-nil")
	}
	if err := WriteSessionMeta(filepath.Join(t.TempDir(), SessionMetaName), SessionMeta{}); err == nil {
		t.Fatal("WriteSessionMeta(invalid meta) error = nil, want non-nil")
	}
}

func TestWorkspaceHelperFunctions(t *testing.T) {
	t.Parallel()

	normalized, addDirsJSON, err := normalizeWorkspaceRecord(aghworkspace.Workspace{
		ID:             " ws-helper ",
		RootDir:        " /tmp/workspace ",
		AdditionalDirs: []string{" /tmp/a ", "", " /tmp/b "},
		Name:           " alpha ",
		DefaultAgent:   " coder ",
	})
	if err != nil {
		t.Fatalf("normalizeWorkspaceRecord() error = %v", err)
	}
	if normalized.ID != "ws-helper" || normalized.RootDir != "/tmp/workspace" || normalized.Name != "alpha" || normalized.DefaultAgent != "coder" {
		t.Fatalf("normalizeWorkspaceRecord() = %#v", normalized)
	}
	if got, want := normalized.AdditionalDirs, []string{"/tmp/a", "/tmp/b"}; !equalStringSlices(got, want) {
		t.Fatalf("normalizeWorkspaceRecord().AdditionalDirs = %#v, want %#v", got, want)
	}
	if addDirsJSON != `["/tmp/a","/tmp/b"]` {
		t.Fatalf("normalizeWorkspaceRecord() addDirsJSON = %q, want %q", addDirsJSON, `["/tmp/a","/tmp/b"]`)
	}

	if _, _, err := normalizeWorkspaceRecord(aghworkspace.Workspace{Name: "alpha"}); err == nil {
		t.Fatal("normalizeWorkspaceRecord(missing root) error = nil, want non-nil")
	}
	if _, _, err := normalizeWorkspaceRecord(aghworkspace.Workspace{RootDir: "/tmp/workspace"}); err == nil {
		t.Fatal("normalizeWorkspaceRecord(missing name) error = nil, want non-nil")
	}

	if got, err := encodeWorkspaceDirs(nil); err != nil {
		t.Fatalf("encodeWorkspaceDirs(nil) error = %v", err)
	} else if got != "[]" {
		t.Fatalf("encodeWorkspaceDirs(nil) = %q, want []", got)
	}
	if got, err := decodeWorkspaceDirs(`[" /tmp/a ", "", "/tmp/b"]`); err != nil {
		t.Fatalf("decodeWorkspaceDirs() error = %v", err)
	} else if want := []string{"/tmp/a", "/tmp/b"}; !equalStringSlices(got, want) {
		t.Fatalf("decodeWorkspaceDirs() = %#v, want %#v", got, want)
	}
	if _, err := decodeWorkspaceDirs(`{`); err == nil {
		t.Fatal("decodeWorkspaceDirs(invalid JSON) error = nil, want non-nil")
	}

	if got := mapWorkspaceConstraintError(errors.New("UNIQUE constraint failed: workspaces.root_dir")); !errors.Is(got, aghworkspace.ErrWorkspacePathTaken) {
		t.Fatalf("mapWorkspaceConstraintError(root_dir) = %v, want ErrWorkspacePathTaken", got)
	}
	if got := mapWorkspaceConstraintError(errors.New("UNIQUE constraint failed: workspaces.name")); !errors.Is(got, aghworkspace.ErrWorkspaceNameTaken) {
		t.Fatalf("mapWorkspaceConstraintError(name) = %v, want ErrWorkspaceNameTaken", got)
	}
	rawErr := errors.New("boom")
	if got := mapWorkspaceConstraintError(rawErr); !errors.Is(got, rawErr) {
		t.Fatalf("mapWorkspaceConstraintError(raw) = %v, want raw error", got)
	}

	if got := coalesceTimestamp(""); got == "" {
		t.Fatal("coalesceTimestamp(\"\") = empty, want timestamp")
	}
	if got := coalesceTimestamp(" 2026-04-03T10:00:00.000000000Z "); got != "2026-04-03T10:00:00.000000000Z" {
		t.Fatalf("coalesceTimestamp(spaced) = %q", got)
	}

	if got := nullStringValue(sql.NullString{}); got != nil {
		t.Fatalf("nullStringValue(invalid) = %#v, want nil", got)
	}
	if got := nullStringValue(sql.NullString{String: " value ", Valid: true}); got != "value" {
		t.Fatalf("nullStringValue(valid) = %#v, want value", got)
	}
}

func TestWorkspaceSchemaHelpers(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	ctx := testContext(t)

	if exists, err := tableExists(ctx, globalDB.db, "workspaces"); err != nil {
		t.Fatalf("tableExists(workspaces) error = %v", err)
	} else if !exists {
		t.Fatal("tableExists(workspaces) = false, want true")
	}
	if exists, err := tableExists(ctx, globalDB.db, "missing_table"); err != nil {
		t.Fatalf("tableExists(missing_table) error = %v", err)
	} else if exists {
		t.Fatal("tableExists(missing_table) = true, want false")
	}

	columns, err := tableColumns(ctx, globalDB.db, "workspaces")
	if err != nil {
		t.Fatalf("tableColumns(workspaces) error = %v", err)
	}
	for _, column := range []string{"id", "root_dir", "add_dirs", "name", "default_agent", "created_at", "updated_at"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("tableColumns(workspaces) missing %q: %#v", column, columns)
		}
	}

	rootDir := filepath.Join(t.TempDir(), "workspace-helper")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if _, err := globalDB.db.ExecContext(
		ctx,
		`INSERT INTO workspaces (id, root_dir, add_dirs, name, default_agent, created_at, updated_at) VALUES (?, ?, '[]', ?, '', ?, ?)`,
		"ws-helper",
		rootDir,
		"workspace-helper",
		formatTimestamp(time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)),
		formatTimestamp(time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert workspace helper row error = %v", err)
	}

	rootToID, err := loadWorkspaceIDsByRootDir(ctx, globalDB.db)
	if err != nil {
		t.Fatalf("loadWorkspaceIDsByRootDir() error = %v", err)
	}
	if got := rootToID[rootDir]; got != "ws-helper" {
		t.Fatalf("rootToID[%q] = %q, want ws-helper", rootDir, got)
	}

	names, err := loadWorkspaceNames(ctx, globalDB.db)
	if err != nil {
		t.Fatalf("loadWorkspaceNames() error = %v", err)
	}
	if _, ok := names["workspace-helper"]; !ok {
		t.Fatalf("loadWorkspaceNames() missing workspace-helper: %#v", names)
	}

	if got := uniqueWorkspaceName(rootDir, map[string]struct{}{"workspace-helper": {}}); got != "workspace-helper-2" {
		t.Fatalf("uniqueWorkspaceName() = %q, want workspace-helper-2", got)
	}
}

func TestLegacyMigrationHelperFlow(t *testing.T) {
	t.Parallel()

	ctx := testContext(t)
	db, err := openSQLiteDatabase(ctx, filepath.Join(t.TempDir(), "legacy.db"), nil)
	if err != nil {
		t.Fatalf("openSQLiteDatabase() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

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
	rootA := filepath.Join(t.TempDir(), "apps", "project")
	rootB := filepath.Join(t.TempDir(), "services", "project")
	for _, rootDir := range []string{rootA, rootB} {
		if err := os.MkdirAll(rootDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", rootDir, err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"sess-a", "A", "coder", rootA, "user", "active", formatTimestamp(time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)), formatTimestamp(time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert legacy sess-a error = %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO sessions (id, name, agent_name, workspace, session_type, state, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"sess-b", "B", "reviewer", rootB, "dream", "stopped", formatTimestamp(time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)), formatTimestamp(time.Date(2026, 4, 3, 11, 30, 0, 0, time.UTC)),
	); err != nil {
		t.Fatalf("insert legacy sess-b error = %v", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, globalSchemaStatements[0]); err != nil {
		t.Fatalf("create workspaces table in tx error = %v", err)
	}

	sessionRows, seeds, err := loadLegacySessions(ctx, tx)
	if err != nil {
		t.Fatalf("loadLegacySessions() error = %v", err)
	}
	if got, want := len(sessionRows), 2; got != want {
		t.Fatalf("len(sessionRows) = %d, want %d", got, want)
	}
	if got, want := len(seeds), 2; got != want {
		t.Fatalf("len(seeds) = %d, want %d", got, want)
	}

	workspaceIDs, err := ensureMigratedWorkspaces(ctx, tx, seeds)
	if err != nil {
		t.Fatalf("ensureMigratedWorkspaces() error = %v", err)
	}
	if got, want := len(workspaceIDs), 2; got != want {
		t.Fatalf("len(workspaceIDs) = %d, want %d", got, want)
	}

	names, err := loadWorkspaceNames(ctx, tx)
	if err != nil {
		t.Fatalf("loadWorkspaceNames() error = %v", err)
	}
	for _, name := range []string{"project", "project-2"} {
		if _, ok := names[name]; !ok {
			t.Fatalf("loadWorkspaceNames() missing %q: %#v", name, names)
		}
	}

	if err := createMigratedGlobalTables(ctx, tx); err != nil {
		t.Fatalf("createMigratedGlobalTables() error = %v", err)
	}
	if err := copyGlobalTableIfExists(ctx, tx, "event_summaries", "event_summaries_new", `INSERT INTO event_summaries_new (id, session_id, type, agent_name, summary, timestamp) SELECT id, session_id, type, agent_name, summary, timestamp FROM event_summaries`); err != nil {
		t.Fatalf("copyGlobalTableIfExists(missing source) error = %v", err)
	}
	if err := copyMigratedSessions(ctx, tx, sessionRows, workspaceIDs); err != nil {
		t.Fatalf("copyMigratedSessions() error = %v", err)
	}
	if err := swapMigratedGlobalTables(ctx, tx); err != nil {
		t.Fatalf("swapMigratedGlobalTables() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	assertTableColumns(t, db, "sessions", []string{"id", "name", "agent_name", "workspace_id", "session_type", "state", "acp_session_id", "created_at", "updated_at"})
}

func TestSessionDBMethodsAfterCloseAndErrors(t *testing.T) {
	t.Parallel()

	sessionDB := openTestSessionDB(t, "sess-errors")
	if got, want := sessionDB.Path(), sessionDB.path; got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
	if got, want := sessionDB.SessionID(), "sess-errors"; got != want {
		t.Fatalf("SessionID() = %q, want %q", got, want)
	}
	if got := ((*SessionDB)(nil)).Path(); got != "" {
		t.Fatalf("nil SessionDB Path() = %q, want empty", got)
	}
	if got := ((*SessionDB)(nil)).SessionID(); got != "" {
		t.Fatalf("nil SessionDB SessionID() = %q, want empty", got)
	}
	if err := sessionDB.Record(testContext(t), SessionEvent{SessionID: "other", TurnID: "turn-1", Type: "agent_message", AgentName: "coder"}); err == nil {
		t.Fatal("Record(mismatched session id) error = nil, want non-nil")
	}
	if _, err := sessionDB.Query(testContext(t), EventQuery{Limit: -1}); err == nil {
		t.Fatal("Query(invalid) error = nil, want non-nil")
	}
	if err := sessionDB.Close(testContext(t)); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := sessionDB.Record(testContext(t), SessionEvent{TurnID: "turn-1", Type: "agent_message", AgentName: "coder"}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Record(after close) error = %v, want ErrClosed", err)
	}

	var nilSessionDB *SessionDB
	if err := nilSessionDB.Record(testContext(t), SessionEvent{}); err == nil {
		t.Fatal("nil SessionDB Record() error = nil, want non-nil")
	}
	if err := nilSessionDB.RecordTokenUsage(testContext(t), TokenUsage{}); err == nil {
		t.Fatal("nil SessionDB RecordTokenUsage() error = nil, want non-nil")
	}
	if _, err := nilSessionDB.Query(testContext(t), EventQuery{}); err == nil {
		t.Fatal("nil SessionDB Query() error = nil, want non-nil")
	}
	if err := nilSessionDB.Close(testContext(t)); err != nil {
		t.Fatalf("nil SessionDB Close() error = %v, want nil", err)
	}
}

func TestGlobalDBMethodsAndErrors(t *testing.T) {
	t.Parallel()

	globalDB := openTestGlobalDB(t)
	if got, want := globalDB.Path(), globalDB.path; got != want {
		t.Fatalf("Path() = %q, want %q", got, want)
	}
	if got := ((*GlobalDB)(nil)).Path(); got != "" {
		t.Fatalf("nil GlobalDB Path() = %q, want empty", got)
	}

	var nilGlobalDB *GlobalDB
	if err := nilGlobalDB.RegisterSession(testContext(t), SessionInfo{}); err == nil {
		t.Fatal("nil GlobalDB RegisterSession() error = nil, want non-nil")
	}
	if err := nilGlobalDB.UpdateSessionState(testContext(t), SessionStateUpdate{}); err == nil {
		t.Fatal("nil GlobalDB UpdateSessionState() error = nil, want non-nil")
	}
	if _, err := nilGlobalDB.ListSessions(testContext(t), SessionListQuery{}); err == nil {
		t.Fatal("nil GlobalDB ListSessions() error = nil, want non-nil")
	}
	if _, err := nilGlobalDB.ReconcileSessions(testContext(t), nil); err == nil {
		t.Fatal("nil GlobalDB ReconcileSessions() error = nil, want non-nil")
	}
	if err := nilGlobalDB.WriteEventSummary(testContext(t), EventSummary{}); err == nil {
		t.Fatal("nil GlobalDB WriteEventSummary() error = nil, want non-nil")
	}
	if _, err := nilGlobalDB.ListEventSummaries(testContext(t), EventSummaryQuery{}); err == nil {
		t.Fatal("nil GlobalDB ListEventSummaries() error = nil, want non-nil")
	}
	if err := nilGlobalDB.UpdateTokenStats(testContext(t), TokenStatsUpdate{}); err == nil {
		t.Fatal("nil GlobalDB UpdateTokenStats() error = nil, want non-nil")
	}
	if _, err := nilGlobalDB.ListTokenStats(testContext(t), TokenStatsQuery{}); err == nil {
		t.Fatal("nil GlobalDB ListTokenStats() error = nil, want non-nil")
	}
	if err := nilGlobalDB.WritePermissionLog(testContext(t), PermissionLogEntry{}); err == nil {
		t.Fatal("nil GlobalDB WritePermissionLog() error = nil, want non-nil")
	}
	if _, err := nilGlobalDB.ListPermissionLog(testContext(t), PermissionLogQuery{}); err == nil {
		t.Fatal("nil GlobalDB ListPermissionLog() error = nil, want non-nil")
	}
	if err := nilGlobalDB.Close(testContext(t)); err != nil {
		t.Fatalf("nil GlobalDB Close() error = %v, want nil", err)
	}
}

func TestSessionWriterHelpers(t *testing.T) {
	t.Parallel()

	t.Run("waitForShutdownResult returns request result", func(t *testing.T) {
		t.Parallel()

		resultCh := make(chan error, 1)
		resultCh <- errors.New("boom")
		if err := waitForShutdownResult(testContext(t), resultCh); err == nil {
			t.Fatal("waitForShutdownResult() error = nil, want non-nil")
		}
	})

	t.Run("waitForShutdownResult times out", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := waitForShutdownResult(ctx, make(chan error)); !errors.Is(err, ErrDrainTimeout) {
			t.Fatalf("waitForShutdownResult() error = %v, want ErrDrainTimeout", err)
		}
	})

	t.Run("waitForWriterExit returns on done", func(t *testing.T) {
		t.Parallel()

		done := make(chan struct{})
		close(done)
		if err := waitForWriterExit(testContext(t), done); err != nil {
			t.Fatalf("waitForWriterExit() error = %v", err)
		}
	})

	t.Run("waitForWriterExit times out", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := waitForWriterExit(ctx, make(chan struct{})); !errors.Is(err, ErrDrainTimeout) {
			t.Fatalf("waitForWriterExit() error = %v, want ErrDrainTimeout", err)
		}
	})

	t.Run("drainWrites executes queued requests and returns aggregate error", func(t *testing.T) {
		t.Parallel()

		sessionDB := &SessionDB{
			writeCh: make(chan sessionWriteRequest, 1),
		}
		req := sessionWriteRequest{
			ctx:    context.Background(),
			kind:   255,
			result: make(chan error, 1),
		}
		sessionDB.writeCh <- req

		if err := sessionDB.drainWrites(context.Background()); err == nil {
			t.Fatal("drainWrites() error = nil, want non-nil")
		}
		if err := <-req.result; err == nil {
			t.Fatal("queued write result error = nil, want non-nil")
		}
	})

	t.Run("drainWrites honors cancellation", func(t *testing.T) {
		t.Parallel()

		sessionDB := &SessionDB{
			writeCh: make(chan sessionWriteRequest),
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := sessionDB.drainWrites(ctx); !errors.Is(err, ErrDrainTimeout) {
			t.Fatalf("drainWrites() error = %v, want ErrDrainTimeout", err)
		}
	})
}
