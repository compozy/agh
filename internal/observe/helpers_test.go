package observe

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestNewWithEmptyHomePathsReturnsError(t *testing.T) {
	t.Parallel()

	if _, err := New(testContext(t), WithHomePaths(aghconfig.HomePaths{})); err == nil {
		t.Fatal("New(empty home paths) error = nil, want non-nil")
	}
}

func TestNewOpensRegistryAndCloseSucceeds(t *testing.T) {
	home, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	observer, err := New(testContext(t),
		WithHomePaths(home),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if observer.registry == nil {
		t.Fatal("observer.registry = nil, want opened registry")
	}
	if observer.registry.Path() != home.DatabaseFile {
		t.Fatalf("observer.registry.Path() = %q, want %q", observer.registry.Path(), home.DatabaseFile)
	}
	if err := observer.Close(testContext(t)); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestDefaultPermissionModeResolverUsesConfigAndAgent(t *testing.T) {
	home, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(home); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	agentDir := filepath.Join(home.AgentsDir, "coder")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "AGENT.md"), []byte(`---
name: coder
provider: codex
---

You write reliable code.
`), 0o644); err != nil {
		t.Fatalf("WriteFile(agent) error = %v", err)
	}
	if err := os.WriteFile(home.ConfigFile, []byte(`
[providers.codex]
command = "codex"

[permissions]
mode = "deny-all"
`), 0o644); err != nil {
		t.Fatalf("WriteFile(global config) error = %v", err)
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("MkdirAll(workspace) error = %v", err)
	}
	workspaceConfigDir := filepath.Join(workspace, aghconfig.DirName)
	if err := os.MkdirAll(workspaceConfigDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(workspace config) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceConfigDir, aghconfig.ConfigName), []byte(`
[permissions]
mode = "approve-all"
`), 0o644); err != nil {
		t.Fatalf("WriteFile(workspace config) error = %v", err)
	}

	resolver := defaultPermissionModeResolver(home, fakeObserveWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-observe",
				RootDir: workspace,
			},
		},
	})
	got, err := resolver(testContext(t), "coder", "ws-observe")
	if err != nil {
		t.Fatalf("resolver() error = %v", err)
	}
	if got != "approve-all" {
		t.Fatalf("resolver() = %q, want approve-all", got)
	}
}

func TestDefaultPermissionModeResolverReturnsErrorForMissingAgent(t *testing.T) {
	home, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(home); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("MkdirAll(workspace) error = %v", err)
	}
	if err := os.WriteFile(home.ConfigFile, []byte(`
[providers.codex]
command = "codex"
`), 0o644); err != nil {
		t.Fatalf("WriteFile(global config) error = %v", err)
	}

	resolver := defaultPermissionModeResolver(home, fakeObserveWorkspaceResolver{
		resolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{
				ID:      "ws-observe",
				RootDir: workspace,
			},
		},
	})
	if _, err := resolver(testContext(t), "missing", "ws-observe"); err == nil {
		t.Fatal("resolver(missing agent) error = nil, want non-nil")
	}
}

func TestDefaultPermissionModeResolverRequiresResolverForWorkspaceID(t *testing.T) {
	t.Parallel()

	home, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(home); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	resolver := defaultPermissionModeResolver(home, nil)
	if _, err := resolver(testContext(t), "coder", "ws-missing"); err == nil {
		t.Fatal("resolver(nil workspace resolver) error = nil, want non-nil")
	}
}

func TestOnSessionCreatedResolverFailureStillRegistersSession(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.resolvePermissionMode = func(context.Context, string, string) (string, error) {
		return "", errors.New("boom")
	}

	sess := newSession("sess-resolver-failure", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testContext(t), sess)

	sessions, err := h.observer.registry.ListSessions(testContext(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
}

func TestHealthFallsBackToRegistryWithoutSessionSource(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.sessionSource = nil

	now := h.now
	for _, info := range []store.SessionInfo{
		{ID: "sess-active", AgentName: "coder", WorkspaceID: h.workspaceID, State: "active", CreatedAt: now, UpdatedAt: now},
		{ID: "sess-stopped", AgentName: "coder", WorkspaceID: h.workspaceID, State: "stopped", CreatedAt: now, UpdatedAt: now},
		{ID: "sess-orphaned", AgentName: "coder", WorkspaceID: h.workspaceID, State: "orphaned", CreatedAt: now, UpdatedAt: now},
	} {
		if err := h.observer.registry.RegisterSession(testContext(t), info); err != nil {
			t.Fatalf("RegisterSession(%q) error = %v", info.ID, err)
		}
	}

	health, err := h.observer.Health(testContext(t))
	if err != nil {
		t.Fatalf("Health(nil) error = %v", err)
	}
	if health.ActiveSessions != 1 || health.ActiveAgents != 1 {
		t.Fatalf("Health() = %#v, want 1 active session/agent", health)
	}
}

func TestSessionDBSizeHelpers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sessionDir := filepath.Join(dir, "sess-1")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	dbPath := store.SessionDBFile(sessionDir)
	if err := os.WriteFile(dbPath, []byte("db"), 0o644); err != nil {
		t.Fatalf("WriteFile(db) error = %v", err)
	}
	if err := os.WriteFile(dbPath+"-wal", []byte("wal"), 0o644); err != nil {
		t.Fatalf("WriteFile(wal) error = %v", err)
	}
	if err := os.WriteFile(dbPath+"-shm", []byte("shm"), 0o644); err != nil {
		t.Fatalf("WriteFile(shm) error = %v", err)
	}

	gotDB, err := databaseSize(dbPath)
	if err != nil {
		t.Fatalf("databaseSize() error = %v", err)
	}
	if gotDB != int64(len("db")+len("wal")+len("shm")) {
		t.Fatalf("databaseSize() = %d", gotDB)
	}

	gotTotal, err := totalSessionDBSize(dir)
	if err != nil {
		t.Fatalf("totalSessionDBSize() error = %v", err)
	}
	if gotTotal != gotDB {
		t.Fatalf("totalSessionDBSize() = %d, want %d", gotTotal, gotDB)
	}

	empty, err := databaseSize("")
	if err != nil {
		t.Fatalf("databaseSize(empty) error = %v", err)
	}
	if empty != 0 {
		t.Fatalf("databaseSize(empty) = %d, want 0", empty)
	}
}

func TestLoadSessionMetadataSkipsMissingMetaAndKeepsStoppedState(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	if err := os.MkdirAll(filepath.Join(h.home.SessionsDir, "sess-empty"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	sessionDir := filepath.Join(h.home.SessionsDir, "sess-stopped")
	if err := store.WriteSessionMeta(store.SessionMetaFile(sessionDir), store.SessionMeta{
		ID:          "sess-stopped",
		Name:        "Stopped",
		AgentName:   "coder",
		WorkspaceID: h.workspaceID,
		State:       "stopped",
		CreatedAt:   h.now,
		UpdatedAt:   h.now,
	}); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	sessions, err := h.observer.loadSessionMetadata()
	if err != nil {
		t.Fatalf("loadSessionMetadata() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].State != "stopped" {
		t.Fatalf("sessions[0].State = %q, want stopped", sessions[0].State)
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	if got := sessionInfoFromSession(nil); got != (store.SessionInfo{}) {
		t.Fatalf("sessionInfoFromSession(nil) = %#v, want zero value", got)
	}
	if got := stringPointer(""); got != nil {
		t.Fatalf("stringPointer(\"\") = %#v, want nil", got)
	}
	if got := summarizeEvent(acp.AgentEvent{Raw: []byte(`{"hello":"world"}`)}); got != `{"hello":"world"}` {
		t.Fatalf("summarizeEvent(raw) = %q", got)
	}
	long := truncateSummary(strings.Repeat("a", 300))
	if len([]rune(long)) != 240 {
		t.Fatalf("truncateSummary(len=300) rune count = %d, want 240", len([]rune(long)))
	}
}

func TestObserverVersionSourceUsedByHealth(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.startedAt = h.now
	h.observer.now = func() time.Time { return h.now.Add(time.Second) }

	health, err := h.observer.Health(testContext(t))
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if health.Version != "1.2.3" {
		t.Fatalf("Health().Version = %q, want 1.2.3", health.Version)
	}
}

func TestMissingPathHelpers(t *testing.T) {
	t.Parallel()

	missingDir := filepath.Join(t.TempDir(), "missing")
	size, err := totalSessionDBSize(missingDir)
	if err != nil {
		t.Fatalf("totalSessionDBSize(missing) error = %v", err)
	}
	if size != 0 {
		t.Fatalf("totalSessionDBSize(missing) = %d, want 0", size)
	}

	h := newHarness(t)
	h.observer.homePaths.SessionsDir = missingDir
	sessions, err := h.observer.loadSessionMetadata()
	if err != nil {
		t.Fatalf("loadSessionMetadata(missing) error = %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("len(loadSessionMetadata(missing)) = %d, want 0", len(sessions))
	}
}

type fakeObserveWorkspaceResolver struct {
	resolved workspacepkg.ResolvedWorkspace
	err      error
}

func (r fakeObserveWorkspaceResolver) Resolve(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
	if r.err != nil {
		return workspacepkg.ResolvedWorkspace{}, r.err
	}
	return r.resolved, nil
}

func (r fakeObserveWorkspaceResolver) ResolveOrRegister(context.Context, string) (workspacepkg.ResolvedWorkspace, error) {
	if r.err != nil {
		return workspacepkg.ResolvedWorkspace{}, r.err
	}
	return r.resolved, nil
}
