//go:build integration

package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/kballard/go-shellquote"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

const daemonSessionStopHelperEnvKey = "AGH_TEST_DAEMON_SESSION_STOP_HELPER"

func (f *fakeSessionManager) promptCall(index int) struct {
	id  string
	msg string
} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if index < 0 || index >= len(f.promptCalls) {
		return struct {
			id  string
			msg string
		}{}
	}
	return f.promptCalls[index]
}

func (f *fakeSessionManager) promptCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.promptCalls)
}

func TestBootSequenceReady(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.sessions == nil || d.observer == nil || d.registry == nil {
		t.Fatalf("boot() did not wire runtime dependencies: sessions=%v observer=%v registry=%v", d.sessions, d.observer, d.registry)
	}
	if d.workspaceResolver == nil {
		t.Fatal("boot() did not wire the workspace resolver")
	}
	if _, err := os.Stat(homePaths.DatabaseFile); err != nil {
		t.Fatalf("stat global database error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); err != nil {
		t.Fatalf("stat daemon.json error = %v", err)
	}
	if _, err := AcquireLock(homePaths.DaemonLock, os.Getpid()); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("AcquireLock(second instance) error = %v, want ErrAlreadyRunning", err)
	}
}

func TestRunGracefulShutdownViaContextCancellation(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	runCtx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(runCtx)
	}()

	<-d.readyCh
	cancel()

	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("daemon.json after shutdown: stat error = %v, want os.ErrNotExist", err)
	}

	lock, err := AcquireLock(homePaths.DaemonLock, os.Getpid())
	if err != nil {
		t.Fatalf("AcquireLock(after shutdown) error = %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatalf("lock.Release() error = %v", err)
	}
}

func TestRunGracefulShutdownViaSignal(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	signalCh := make(chan os.Signal, 1)

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
		WithSignalChannel(signalCh),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(context.Background())
	}()

	<-d.readyCh
	signalCh <- syscall.SIGINT

	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if _, err := os.Stat(homePaths.DaemonInfo); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("daemon.json after signal shutdown: stat error = %v, want os.ErrNotExist", err)
	}
}

func TestShutdownPersistsShutdownStopReason(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	command := daemonSessionStopHelperCommand(t)
	cfg.Providers["claude"] = aghconfig.ProviderConfig{Command: command}
	writeDaemonIntegrationAgentDef(t, homePaths, "coder", command)

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspaceRoot, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", workspaceRoot, err)
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	shutdown := false
	t.Cleanup(func() {
		if shutdown {
			return
		}
		_ = d.Shutdown(testutil.Context(t))
	})

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}

	sess, err := d.sessions.Create(testutil.Context(t), session.CreateOpts{
		AgentName:     "coder",
		WorkspacePath: workspaceRoot,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := d.Shutdown(testutil.Context(t)); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	shutdown = true

	meta, err := store.ReadSessionMeta(sess.MetaPath())
	if err != nil {
		t.Fatalf("ReadSessionMeta(%q) error = %v", sess.MetaPath(), err)
	}
	if meta.StopReason == nil {
		t.Fatal("meta.StopReason = nil, want non-nil")
	}
	if *meta.StopReason != store.StopShutdown {
		t.Fatalf("meta.StopReason = %q, want %q", *meta.StopReason, store.StopShutdown)
	}
}

func TestBootInitializesMemoryStoreAndAssemblerIntegration(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.GlobalDir = filepath.Join(homePaths.HomeDir, "external-memory")

	var capturedDeps SessionManagerDeps

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.memoryStore == nil {
		t.Fatal("boot() did not initialize the memory store")
	}
	if capturedDeps.PromptAssembler == nil {
		t.Fatal("boot() did not inject the prompt assembler")
	}
	if capturedDeps.SkillRegistry == nil {
		t.Fatal("boot() did not inject the skills registry")
	}
	if capturedDeps.MCPResolver == nil {
		t.Fatal("boot() did not inject the MCP resolver")
	}
	if capturedDeps.WorkspaceResolver == nil {
		t.Fatal("boot() did not inject the workspace resolver")
	}
	if _, err := os.Stat(cfg.Memory.GlobalDir); err != nil {
		t.Fatalf("stat external memory directory error = %v", err)
	}
}

func TestBootLoadsBundledSkillsIntoPromptAssemblerInSkillsOnlyMode(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Enabled = false
	cfg.Skills.Enabled = true

	var capturedDeps SessionManagerDeps

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if capturedDeps.PromptAssembler == nil {
		t.Fatal("boot() did not inject the prompt assembler")
	}
	if capturedDeps.WorkspaceResolver == nil {
		t.Fatal("boot() did not inject the workspace resolver")
	}
	if d.skillsRegistry == nil {
		t.Fatal("boot() did not initialize the skills registry")
	}
	if _, ok := d.skillsRegistry.Get("agh-session-guide"); !ok {
		t.Fatal("skills registry does not contain bundled skill agh-session-guide")
	}

	prompt, err := capturedDeps.PromptAssembler.Assemble(context.Background(), testPromptAgent("Base prompt."), workspacepkg.ResolvedWorkspace{})
	if err != nil {
		t.Fatalf("PromptAssembler.Assemble() error = %v", err)
	}

	assertPromptContainsInOrder(t, prompt, "Base prompt.", "<available-skills>", "agh-session-guide")
	assertPromptExcludes(t, prompt, "# Persistent Memory")
}

func TestBootLeavesSkillDependenciesNilWhenSkillsDisabled(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Skills.Enabled = false

	var capturedDeps SessionManagerDeps

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if capturedDeps.SkillRegistry != nil {
		t.Fatalf("boot() SkillRegistry = %#v, want nil when skills are disabled", capturedDeps.SkillRegistry)
	}
	if capturedDeps.MCPResolver != nil {
		t.Fatalf("boot() MCPResolver = %#v, want nil when skills are disabled", capturedDeps.MCPResolver)
	}
}

func TestBootBuildsHooksFromWorkspaceConfigAgentAndSkills(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Enabled = false
	cfg.Skills.Enabled = true

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(filepath.Join(workspaceRoot, aghconfig.DirName), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Join(workspaceRoot, aghconfig.DirName), err)
	}

	scriptPath := writeDaemonHookScript(t, t.TempDir(), "capture.sh", "#!/bin/sh\ncat > \"$1\"\n")
	configOutput := filepath.Join(t.TempDir(), "config-create.json")
	agentOutput := filepath.Join(t.TempDir(), "agent-stop.json")
	skillOutput := filepath.Join(t.TempDir(), "skill-create.json")

	writeDaemonFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, "config.toml"), `
[[hooks.declarations]]
name = "config-create"
event = "session.post_create"
mode = "sync"
command = "`+scriptPath+`"
args = ["`+configOutput+`"]
`)
	writeDaemonFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, "agents", "coder", "AGENT.md"), `---
name: coder
provider: claude
hooks:
  - name: agent-stop
    event: session.post_stop
    mode: sync
    command: `+scriptPath+`
    args: ["`+agentOutput+`"]
---

Prompt.
`)
	writeDaemonFile(t, filepath.Join(workspaceRoot, aghconfig.DirName, "skills", "local-hook", "SKILL.md"), `---
name: local-hook
description: workspace lifecycle hook
metadata:
  agh:
    hooks:
      - event: session.post_create
        mode: sync
        command: `+scriptPath+`
        args:
          - `+skillOutput+`
---

body
`)

	resolvedWorkspace := seedDaemonWorkspace(t, homePaths, workspaceRoot)

	var capturedDeps SessionManagerDeps
	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if d.hooks == nil {
		t.Fatal("boot() did not initialize hooks runtime")
	}
	if capturedDeps.Notifier == nil {
		t.Fatal("boot() did not inject the hooks notifier")
	}
	if capturedDeps.Hooks == nil {
		t.Fatal("boot() did not inject the hooks dispatcher")
	}

	sess := &session.Session{
		ID:          "sess-1",
		Name:        "demo",
		AgentName:   "coder",
		WorkspaceID: resolvedWorkspace.ID,
		Workspace:   resolvedWorkspace.RootDir,
		Type:        session.SessionTypeUser,
		State:       session.StateStopped,
		CreatedAt:   time.Date(2026, 4, 9, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 9, 11, 0, 0, 0, time.UTC),
	}

	if _, err := capturedDeps.Hooks.DispatchSessionPostCreate(testutil.Context(t), hookspkg.SessionPostCreatePayload(hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostCreate, time.Now().UTC()))); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}
	if _, err := capturedDeps.Hooks.DispatchSessionPostStop(testutil.Context(t), hookspkg.SessionPostStopPayload(hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostStop, time.Now().UTC()))); err != nil {
		t.Fatalf("DispatchSessionPostStop() error = %v", err)
	}

	assertLifecycleHookPayload(t, configOutput, hookspkg.HookSessionPostCreate, resolvedWorkspace)
	assertLifecycleHookPayload(t, skillOutput, hookspkg.HookSessionPostCreate, resolvedWorkspace)
	assertLifecycleHookPayload(t, agentOutput, hookspkg.HookSessionPostStop, resolvedWorkspace)
}

func TestBootSkillsWatcherRebuildsHooksBeforeNextDispatch(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Enabled = false
	cfg.Skills.Enabled = true
	cfg.Skills.PollInterval = 10 * time.Millisecond

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	resolvedWorkspace := seedDaemonWorkspace(t, homePaths, workspaceRoot)
	outputPath := filepath.Join(t.TempDir(), "watched-create.json")
	scriptPath := writeDaemonHookScript(t, t.TempDir(), "capture.sh", "#!/bin/sh\ncat > \"$1\"\n")

	var capturedDeps SessionManagerDeps
	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(_ context.Context, deps SessionManagerDeps) (SessionManager, error) {
		capturedDeps = deps
		return &fakeSessionManager{}, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	if err := d.boot(testutil.Context(t)); err != nil {
		t.Fatalf("boot() error = %v", err)
	}
	t.Cleanup(func() {
		if err := d.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})
	if capturedDeps.Hooks == nil {
		t.Fatal("boot() did not inject the hooks dispatcher")
	}

	initialVersion := d.hooks.Version()
	writeDaemonFile(t, filepath.Join(homePaths.SkillsDir, "watched-hook", "SKILL.md"), `---
name: watched-hook
description: reloaded hook
metadata:
  agh:
    hooks:
      - event: session.post_create
        mode: sync
        command: `+scriptPath+`
        args:
          - `+outputPath+`
---

body
`)

	waitForCondition(t, "hooks rebuild after watcher refresh", func() bool {
		if _, ok := d.skillsRegistry.Get("watched-hook"); !ok {
			return false
		}
		return d.hooks.Version() > initialVersion
	})

	sess := &session.Session{
		ID:          "sess-watch",
		AgentName:   "general",
		WorkspaceID: resolvedWorkspace.ID,
		Workspace:   resolvedWorkspace.RootDir,
		Type:        session.SessionTypeUser,
		State:       session.StateActive,
		CreatedAt:   time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC),
	}

	if _, err := capturedDeps.Hooks.DispatchSessionPostCreate(testutil.Context(t), hookspkg.SessionPostCreatePayload(hookSessionLifecyclePayload(sess, hookspkg.HookSessionPostCreate, time.Now().UTC()))); err != nil {
		t.Fatalf("DispatchSessionPostCreate() error = %v", err)
	}
	assertLifecycleHookPayload(t, outputPath, hookspkg.HookSessionPostCreate, resolvedWorkspace)
}

func TestRunDreamTickerAndSpawnerIntegration(t *testing.T) {
	homePaths := integrationHomePaths(t)
	cfg := testConfig(t, homePaths)
	cfg.Memory.Dream.CheckInterval = 10 * time.Millisecond

	workspace := filepath.Join(t.TempDir(), "workspace")
	resolvedWorkspace := seedDaemonWorkspace(t, homePaths, workspace)
	dream := &fakeDreamService{
		shouldRun: true,
		runHook: func(ctx context.Context, spawn memory.SessionSpawner, workspace string) error {
			return spawn(ctx, "memory-consolidation", "integration prompt", workspace)
		},
	}
	sessions := &fakeSessionManager{
		infos: []*session.SessionInfo{
			{
				ID:          "sess-user",
				WorkspaceID: resolvedWorkspace.ID,
				Type:        session.SessionTypeUser,
				UpdatedAt:   time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	d, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithLogger(discardLogger()),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	d.newSessionManager = func(context.Context, SessionManagerDeps) (SessionManager, error) {
		return sessions, nil
	}
	d.newObserver = func(context.Context, RuntimeDeps) (Observer, error) {
		return &fakeObserver{}, nil
	}
	d.newDreamService = func(opts ...memory.Option) consolidation.Service {
		return dream
	}
	d.httpFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "http"}, nil
	}
	d.udsFactory = func(context.Context, RuntimeDeps) (Server, error) {
		return &fakeServer{name: "uds"}, nil
	}

	runCtx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- d.Run(runCtx)
	}()

	<-d.readyCh
	waitForCondition(t, "integration dream run", func() bool {
		return sessions.createCount() > 0
	})

	cancel()
	if err := <-errCh; err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := sessions.createCall(0).Type; got != session.SessionTypeDream {
		t.Fatalf("Create() session type = %q, want %q", got, session.SessionTypeDream)
	}
	if got := sessions.createCall(0).Workspace; got != resolvedWorkspace.ID {
		t.Fatalf("Create() workspace = %q, want %q", got, resolvedWorkspace.ID)
	}
	if got := sessions.createCall(0).WorkspacePath; got != "" {
		t.Fatalf("Create() workspace_path = %q, want empty", got)
	}
	if got := sessions.promptCount(); got == 0 || sessions.promptCall(0).msg != "integration prompt" {
		t.Fatalf("Prompt() calls = %d, want integration prompt", got)
	}
}

func integrationHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()

	homeDir := t.TempDir()
	t.Setenv("AGH_HOME", homeDir)
	t.Setenv("HOME", homeDir)

	homePaths, err := aghconfig.ResolveHomePathsFrom(homeDir)
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	homePaths.DaemonSocket = shortSocketPath(t)
	return homePaths
}

func TestDaemonSessionStopACPHelperProcess(t *testing.T) {
	if os.Getenv(daemonSessionStopHelperEnvKey) != "1" {
		return
	}

	conn := acpsdk.NewAgentSideConnection(daemonSessionStopACPAgent{}, os.Stdout, os.Stdin)
	<-conn.Done()
	os.Exit(0)
}

func seedDaemonWorkspace(t *testing.T, homePaths aghconfig.HomePaths, root string) workspacepkg.ResolvedWorkspace {
	t.Helper()

	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", root, err)
	}

	registry, err := globaldb.OpenGlobalDB(testutil.Context(t), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	defer func() {
		if err := registry.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	resolver, err := workspacepkg.NewResolver(
		registry,
		workspacepkg.WithHomePaths(homePaths),
		workspacepkg.WithLogger(discardLogger()),
		workspacepkg.WithConfigLoader(func(rootDir string) (aghconfig.Config, error) {
			return aghconfig.LoadForHome(homePaths, aghconfig.WithWorkspaceRoot(rootDir))
		}),
	)
	if err != nil {
		t.Fatalf("NewResolver() error = %v", err)
	}

	resolved, err := resolver.ResolveOrRegister(testutil.Context(t), root)
	if err != nil {
		t.Fatalf("ResolveOrRegister(%q) error = %v", root, err)
	}
	return resolved
}

func writeDaemonHookScript(t *testing.T, dir string, name string, contents string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
	return path
}

func daemonSessionStopHelperCommand(t *testing.T) string {
	t.Helper()

	bin, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}

	return shellquote.Join(
		"env",
		daemonSessionStopHelperEnvKey+"=1",
		bin,
		"-test.run=TestDaemonSessionStopACPHelperProcess",
	)
}

func writeDaemonIntegrationAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string, command string) {
	t.Helper()

	path := filepath.Join(homePaths.AgentsDir, name, "AGENT.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}

	content := strings.Join([]string{
		"---",
		"name: " + name,
		"provider: claude",
		"command: " + command,
		"---",
		"You are a coding assistant.",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", path, err)
	}
}

type daemonSessionStopACPAgent struct{}

func (daemonSessionStopACPAgent) Authenticate(context.Context, acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (daemonSessionStopACPAgent) Initialize(context.Context, acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersionNumber,
		AgentCapabilities: acpsdk.AgentCapabilities{
			LoadSession: true,
		},
		AuthMethods: []acpsdk.AuthMethod{},
	}, nil
}

func (daemonSessionStopACPAgent) Cancel(context.Context, acpsdk.CancelNotification) error {
	return nil
}

func (daemonSessionStopACPAgent) NewSession(context.Context, acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	return acpsdk.NewSessionResponse{SessionId: "daemon-stop-helper"}, nil
}

func (daemonSessionStopACPAgent) LoadSession(context.Context, acpsdk.LoadSessionRequest) (acpsdk.LoadSessionResponse, error) {
	return acpsdk.LoadSessionResponse{}, nil
}

func (daemonSessionStopACPAgent) Prompt(context.Context, acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
}

func (daemonSessionStopACPAgent) SetSessionMode(context.Context, acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func assertLifecycleHookPayload(t *testing.T, path string, wantEvent hookspkg.HookEvent, wantWorkspace workspacepkg.ResolvedWorkspace) {
	t.Helper()

	var (
		payloadBytes []byte
		payload      hookspkg.SessionLifecyclePayload
		readOK       bool
		unmarshalOK  bool
	)

	t.Run("read file", func(t *testing.T) {
		var err error
		payloadBytes, err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v", path, err)
		}
		readOK = true
	})

	t.Run("unmarshal", func(t *testing.T) {
		if !readOK {
			t.Skip("payload unavailable after read failure")
		}
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			t.Fatalf("json.Unmarshal(%q) error = %v", path, err)
		}
		unmarshalOK = true
	})

	t.Run("event", func(t *testing.T) {
		if !unmarshalOK {
			t.Skip("payload unavailable after unmarshal failure")
		}
		if payload.Event != wantEvent {
			t.Fatalf("payload.Event = %q, want %q", payload.Event, wantEvent)
		}
	})

	t.Run("workspace id", func(t *testing.T) {
		if !unmarshalOK {
			t.Skip("payload unavailable after unmarshal failure")
		}
		if payload.WorkspaceID != wantWorkspace.ID {
			t.Fatalf("payload.WorkspaceID = %q, want %q", payload.WorkspaceID, wantWorkspace.ID)
		}
	})

	t.Run("workspace path", func(t *testing.T) {
		if !unmarshalOK {
			t.Skip("payload unavailable after unmarshal failure")
		}
		if payload.Workspace != wantWorkspace.RootDir {
			t.Fatalf("payload.Workspace = %q, want %q", payload.Workspace, wantWorkspace.RootDir)
		}
	})
}
