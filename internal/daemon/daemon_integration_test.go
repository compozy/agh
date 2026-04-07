//go:build integration

package daemon

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/memory/consolidation"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func (f *fakeSessionManager) promptCall(index int) struct {
	id  string
	msg string
} {
	f.mu.Lock()
	defer f.mu.Unlock()
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
