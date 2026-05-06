package consolidation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/session"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestRuntimeTriggerReturnsAlreadyRunningWhenLockUnavailable(t *testing.T) {
	t.Parallel()

	service := &fakeDreamService{
		shouldRun: true,
		runErr:    memory.ErrLockUnavailable,
	}
	runtime := NewRuntime(true, service, func(context.Context, string, string, string) error {
		return nil
	}, time.Minute, discardLogger(), nil)

	triggered, reason, err := runtime.Trigger(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("Trigger() error = %v", err)
	}
	if triggered {
		t.Fatal("Trigger() triggered = true, want false")
	}
	if reason != "dream consolidation is already running" {
		t.Fatalf("Trigger() reason = %q, want already-running message", reason)
	}
}

func TestRuntimeTriggerReturnsGateMissWhenRunSignalGateMisses(t *testing.T) {
	t.Parallel()

	service := &fakeDreamService{
		shouldRun: true,
		runErr:    memory.ErrDreamGateNotSatisfied,
	}
	runtime := NewRuntime(true, service, func(context.Context, string, string, string) error {
		return nil
	}, time.Minute, discardLogger(), nil)

	triggered, reason, err := runtime.Trigger(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("Trigger() error = %v", err)
	}
	if triggered {
		t.Fatal("Trigger() triggered = true, want false")
	}
	if reason != "dream consolidation gates are not satisfied" {
		t.Fatalf("Trigger() reason = %q, want gates-not-satisfied message", reason)
	}
}

func TestRuntimeTriggerStates(t *testing.T) {
	t.Parallel()

	t.Run("disabled returns disabled message", func(t *testing.T) {
		runtime := NewRuntime(
			false,
			&fakeDreamService{shouldRun: true},
			func(context.Context, string, string, string) error {
				return nil
			},
			time.Minute,
			discardLogger(),
			nil,
		)

		triggered, reason, err := runtime.Trigger(context.Background(), "ws-1")
		if err != nil {
			t.Fatalf("Trigger() error = %v", err)
		}
		if triggered {
			t.Fatal("Trigger() triggered = true, want false")
		}
		if reason != "dream consolidation is disabled" {
			t.Fatalf("Trigger() reason = %q, want disabled message", reason)
		}
	})

	t.Run("gate miss returns not satisfied message", func(t *testing.T) {
		runtime := NewRuntime(
			true,
			&fakeDreamService{shouldRun: false},
			func(context.Context, string, string, string) error {
				return nil
			},
			time.Minute,
			discardLogger(),
			nil,
		)

		triggered, reason, err := runtime.Trigger(context.Background(), "ws-1")
		if err != nil {
			t.Fatalf("Trigger() error = %v", err)
		}
		if triggered {
			t.Fatal("Trigger() triggered = true, want false")
		}
		if reason != "dream consolidation gates are not satisfied" {
			t.Fatalf("Trigger() reason = %q, want gates-not-satisfied message", reason)
		}
	})

	t.Run("service error is returned", func(t *testing.T) {
		expectedErr := errors.New("gate failed")
		runtime := NewRuntime(
			true,
			&fakeDreamService{shouldRunErr: expectedErr},
			func(context.Context, string, string, string) error {
				return nil
			},
			time.Minute,
			discardLogger(),
			nil,
		)

		_, _, err := runtime.Trigger(context.Background(), "ws-1")
		if !errors.Is(err, expectedErr) {
			t.Fatalf("Trigger() error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("successful run trims workspace", func(t *testing.T) {
		service := &fakeDreamService{shouldRun: true}
		runtime := NewRuntime(true, service, func(context.Context, string, string, string) error {
			return nil
		}, time.Minute, discardLogger(), nil)

		triggered, reason, err := runtime.Trigger(context.Background(), "  ws-1  ")
		if err != nil {
			t.Fatalf("Trigger() error = %v", err)
		}
		if !triggered {
			t.Fatal("Trigger() triggered = false, want true")
		}
		if reason != "" {
			t.Fatalf("Trigger() reason = %q, want empty", reason)
		}
		if got := service.lastWorkspace(); got != "ws-1" {
			t.Fatalf("service workspace = %q, want ws-1", got)
		}
	})
}

func TestRuntimeLastConsolidatedAt(t *testing.T) {
	t.Parallel()

	t.Run("nil callback returns zero time", func(t *testing.T) {
		runtime := NewRuntime(true, nil, nil, time.Minute, discardLogger(), nil)
		got, err := runtime.LastConsolidatedAt()
		if err != nil {
			t.Fatalf("LastConsolidatedAt() error = %v", err)
		}
		if !got.IsZero() {
			t.Fatalf("LastConsolidatedAt() = %v, want zero time", got)
		}
	})

	t.Run("callback result is returned", func(t *testing.T) {
		expected := time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC)
		runtime := NewRuntime(true, nil, nil, time.Minute, discardLogger(), func() (time.Time, error) {
			return expected, nil
		})

		got, err := runtime.LastConsolidatedAt()
		if err != nil {
			t.Fatalf("LastConsolidatedAt() error = %v", err)
		}
		if !got.Equal(expected) {
			t.Fatalf("LastConsolidatedAt() = %v, want %v", got, expected)
		}
	})
}

func TestRuntimeTickerRunsAndStopsOnCancellation(t *testing.T) {
	t.Parallel()

	service := &fakeDreamService{shouldRun: true}
	runtime := NewRuntime(true, service, func(context.Context, string, string, string) error {
		return nil
	}, 10*time.Millisecond, discardLogger(), nil)

	ctx, cancel := context.WithCancel(context.Background())
	runtime.Start(ctx)
	t.Cleanup(runtime.Shutdown)

	waitForCondition(t, "dream ticker run", func() bool {
		return service.runCount() > 0
	})

	cancel()
	runtime.Shutdown()

	runCount := service.runCount()
	time.Sleep(30 * time.Millisecond)
	if got := service.runCount(); got != runCount {
		t.Fatalf("run count after shutdown = %d, want %d", got, runCount)
	}
}

func TestRuntimeEnqueueCheckRunsQueuedRequest(t *testing.T) {
	t.Parallel()

	service := &fakeDreamService{shouldRun: true}
	runtime := NewRuntime(true, service, func(context.Context, string, string, string) error {
		return nil
	}, time.Hour, discardLogger(), nil)

	ctx := t.Context()
	runtime.Start(ctx)
	t.Cleanup(runtime.Shutdown)

	runtime.EnqueueCheck("session_stop", "  ws-queued  ")
	waitForCondition(t, "queued dream check", func() bool {
		return service.runCount() == 1
	})

	if got := service.lastWorkspace(); got != "ws-queued" {
		t.Fatalf("queued workspace = %q, want trimmed queued workspace", got)
	}
}

func TestRuntimeStartDoesNothingWhenDisabled(t *testing.T) {
	t.Parallel()

	service := &fakeDreamService{shouldRun: true}
	runtime := NewRuntime(false, service, func(context.Context, string, string, string) error {
		return nil
	}, 10*time.Millisecond, discardLogger(), nil)

	ctx := t.Context()
	runtime.Start(ctx)
	runtime.EnqueueCheck("manual", "ws-disabled")

	if got := service.runCount(); got != 0 {
		t.Fatalf("run count = %d, want 0", got)
	}
}

func TestRuntimeRunCheckStopsOnErrors(t *testing.T) {
	t.Parallel()

	t.Run("lock unavailable is swallowed", func(t *testing.T) {
		service := &fakeDreamService{shouldRun: true, runErr: memory.ErrLockUnavailable}
		runtime := NewRuntime(true, service, func(context.Context, string, string, string) error {
			return nil
		}, time.Minute, discardLogger(), nil)

		runtime.runCheck(
			context.Background(),
			discardLogger(),
			service,
			func(context.Context, string, string, string) error {
				return nil
			},
			"manual",
			"ws-1",
		)
		if got := service.runCount(); got != 1 {
			t.Fatalf("run count = %d, want 1", got)
		}
	})

	t.Run("signal gate miss is swallowed", func(t *testing.T) {
		service := &fakeDreamService{shouldRun: true, runErr: memory.ErrDreamGateNotSatisfied}
		runtime := NewRuntime(true, service, func(context.Context, string, string, string) error {
			return nil
		}, time.Minute, discardLogger(), nil)

		runtime.runCheck(
			context.Background(),
			discardLogger(),
			service,
			func(context.Context, string, string, string) error {
				return nil
			},
			"manual",
			"ws-1",
		)
		if got := service.runCount(); got != 1 {
			t.Fatalf("run count = %d, want 1", got)
		}
	})

	t.Run("should run error skips spawn", func(t *testing.T) {
		service := &fakeDreamService{shouldRunErr: errors.New("gate failed")}
		spawnCalls := 0
		runtime := NewRuntime(true, service, func(context.Context, string, string, string) error {
			spawnCalls++
			return nil
		}, time.Minute, discardLogger(), nil)

		runtime.runCheck(
			context.Background(),
			discardLogger(),
			service,
			func(context.Context, string, string, string) error {
				spawnCalls++
				return nil
			},
			"manual",
			"ws-1",
		)
		if spawnCalls != 0 {
			t.Fatalf("spawn calls = %d, want 0", spawnCalls)
		}
	})
}

func TestNewSessionSpawnerCreatesDreamSession(t *testing.T) {
	t.Parallel()

	cfg := dreamConfig()
	sessions := &fakeSessionManager{}
	workspace := filepath.Join(t.TempDir(), "workspace")
	resolver := &fakeWorkspaceResolver{
		resolveOrRegisterResolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{ID: "ws-created", RootDir: workspace},
		},
	}

	spawner := NewSessionSpawner(sessions, resolver, &cfg, filepath.Join(t.TempDir(), "memory"))
	if spawner == nil {
		t.Fatal("NewSessionSpawner() = nil, want non-nil")
	}

	if err := spawner(
		context.Background(),
		"memory-consolidation",
		"summarize recent sessions",
		workspace,
	); err != nil {
		t.Fatalf("spawner() error = %v", err)
	}

	if got := sessions.createCount(); got != 1 {
		t.Fatalf("Create() calls = %d, want 1", got)
	}
	if got := sessions.createCall(0).Type; got != session.SessionTypeDream {
		t.Fatalf("Create() type = %q, want %q", got, session.SessionTypeDream)
	}
	if got := sessions.createCall(0).Provider; got != "" {
		t.Fatalf("Create() provider = %q, want explicit empty provider", got)
	}
	if got := sessions.createCall(0).AgentName; got != "memory-agent" {
		t.Fatalf("Create() agent = %q, want explicit configured memory-agent", got)
	}
	if got := sessions.createCall(0).Workspace; got != "ws-created" {
		t.Fatalf("Create() workspace = %q, want ws-created", got)
	}
	if got := sessions.createCall(0).WorkspacePath; got != "" {
		t.Fatalf("Create() workspace_path = %q, want empty", got)
	}
	if got := sessions.promptCount(); got != 1 || sessions.promptCall(0).msg != "summarize recent sessions" {
		t.Fatalf("Prompt() calls = %d, want one prompt payload", got)
	}
	if got := sessions.stopCount(); got != 1 || sessions.stopCall(0) != "dream-1" {
		t.Fatalf("Stop() calls = %d, want stop for created dream session", got)
	}
	if got := resolver.resolveOrRegisterCalls; got != 1 {
		t.Fatalf("ResolveOrRegister() calls = %d, want 1", got)
	}
}

func TestNewSessionSpawnerUsesDedicatedDreamingCuratorForDefaultAgent(t *testing.T) {
	t.Parallel()

	cfg := dreamConfig()
	cfg.Memory.Dream.Agent = aghconfig.DefaultAgentName
	sessions := &fakeSessionManager{}
	resolver := &fakeWorkspaceResolver{
		resolveResolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{ID: "ws-default", RootDir: filepath.Join(t.TempDir(), "workspace")},
		},
	}

	spawner := NewSessionSpawner(sessions, resolver, &cfg, filepath.Join(t.TempDir(), "memory"))
	if err := spawner(context.Background(), "memory-consolidation", "prompt", "ws-default"); err != nil {
		t.Fatalf("spawner() error = %v", err)
	}

	if got := sessions.createCall(0).AgentName; got != DreamingCuratorAgentName {
		t.Fatalf("Create() agent = %q, want %q", got, DreamingCuratorAgentName)
	}
}

func TestNewSessionSpawnerResolvesExplicitAliasWorkspace(t *testing.T) {
	t.Parallel()

	cfg := dreamConfig()
	sessions := &fakeSessionManager{}
	resolver := &fakeWorkspaceResolver{
		resolveResolved: workspacepkg.ResolvedWorkspace{
			Workspace: workspacepkg.Workspace{ID: "ws-alias", RootDir: filepath.Join(t.TempDir(), "workspace")},
		},
	}

	spawner := NewSessionSpawner(sessions, resolver, &cfg, filepath.Join(t.TempDir(), "memory"))
	if err := spawner(context.Background(), "memory-consolidation", "prompt", "workspace-alias"); err != nil {
		t.Fatalf("spawner() error = %v", err)
	}

	if got := resolver.resolveCalls; got != 1 {
		t.Fatalf("Resolve() calls = %d, want 1", got)
	}
	if got := resolver.lastResolveArg; got != "workspace-alias" {
		t.Fatalf("Resolve() arg = %q, want workspace-alias", got)
	}
	if got := sessions.createCall(0).Workspace; got != "ws-alias" {
		t.Fatalf("Create() workspace = %q, want ws-alias", got)
	}
	if got := sessions.createCall(0).Provider; got != "" {
		t.Fatalf("Create() provider = %q, want explicit empty provider", got)
	}
}

func TestNewSessionSpawnerPropagatesWorkspaceResolveErrors(t *testing.T) {
	t.Parallel()

	cfg := dreamConfig()
	expectedErr := errors.New("lookup failed")
	spawner := NewSessionSpawner(
		&fakeSessionManager{},
		&fakeWorkspaceResolver{resolveErr: expectedErr},
		&cfg,
		filepath.Join(t.TempDir(), "memory"),
	)

	err := spawner(context.Background(), "memory-consolidation", "prompt", "workspace-alias")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("spawner() error = %v, want %v", err, expectedErr)
	}
}

func TestIsPathLikeWorkspaceRefRecognizesSlashSeparatedRefs(t *testing.T) {
	t.Parallel()

	if !isPathLikeWorkspaceRef("subdir/workspace") {
		t.Fatal(`isPathLikeWorkspaceRef("subdir/workspace") = false, want true`)
	}
	if !isPathLikeWorkspaceRef(`subdir\workspace`) {
		t.Fatal(`isPathLikeWorkspaceRef("subdir\\workspace") = false, want true`)
	}
}

func TestNewSessionSpawnerDerivesRecentWorkspacesFromSessions(t *testing.T) {
	t.Parallel()

	cfg := dreamConfig()
	sessions := &fakeSessionManager{
		infos: []*session.Info{
			{
				ID:          "dream-old",
				WorkspaceID: "ws-a",
				Type:        session.SessionTypeDream,
				UpdatedAt:   time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC),
			},
			{
				ID:          "user-old",
				WorkspaceID: "ws-a",
				Type:        session.SessionTypeUser,
				UpdatedAt:   time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
			},
			{
				ID:          "user-new",
				WorkspaceID: "ws-b",
				Type:        session.SessionTypeUser,
				UpdatedAt:   time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC),
			},
			{
				ID:          "user-dup",
				WorkspaceID: "ws-a",
				Type:        session.SessionTypeUser,
				UpdatedAt:   time.Date(2026, 4, 4, 9, 0, 0, 0, time.UTC),
			},
		},
	}
	globalMemoryDir := filepath.Join(t.TempDir(), "memory")
	lockPath := memory.ConsolidationLockPath(globalMemoryDir)
	prior := time.Date(2026, 4, 4, 8, 0, 0, 0, time.UTC)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(lock dir) error = %v", err)
	}
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatalf("os.WriteFile(lock) error = %v", err)
	}
	if err := os.Chtimes(lockPath, prior, prior); err != nil {
		t.Fatalf("os.Chtimes(lock) error = %v", err)
	}

	spawner := NewSessionSpawner(sessions, &fakeWorkspaceResolver{}, &cfg, globalMemoryDir)
	if err := spawner(context.Background(), "memory-consolidation", "prompt", ""); err != nil {
		t.Fatalf("spawner() error = %v", err)
	}

	if got := sessions.createCount(); got != 2 {
		t.Fatalf("Create() calls = %d, want 2", got)
	}
	if got := sessions.createCall(0).Workspace; got != "ws-b" {
		t.Fatalf("Create() workspace[0] = %q, want ws-b", got)
	}
	if got := sessions.createCall(0).Provider; got != "" {
		t.Fatalf("Create() provider[0] = %q, want explicit empty provider", got)
	}
	if got := sessions.createCall(1).Workspace; got != "ws-a" {
		t.Fatalf("Create() workspace[1] = %q, want ws-a", got)
	}
	if got := sessions.createCall(1).Provider; got != "" {
		t.Fatalf("Create() provider[1] = %q, want explicit empty provider", got)
	}
}

func TestResolveWorkspaceRefValidatesInputs(t *testing.T) {
	t.Parallel()

	t.Run("blank ref is rejected", func(t *testing.T) {
		_, err := resolveWorkspaceRef(context.Background(), &fakeWorkspaceResolver{}, "   ")
		if err == nil {
			t.Fatal("resolveWorkspaceRef() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "dream workspace is required") {
			t.Fatalf("resolveWorkspaceRef() error = %v, want blank workspace error", err)
		}
	})

	t.Run("empty resolved id is rejected", func(t *testing.T) {
		_, err := resolveWorkspaceRef(context.Background(), &fakeWorkspaceResolver{
			resolveResolved: workspacepkg.ResolvedWorkspace{},
		}, "workspace-alias")
		if err == nil {
			t.Fatal("resolveWorkspaceRef() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "dream workspace id is required") {
			t.Fatalf("resolveWorkspaceRef() error = %v, want empty id error", err)
		}
	})
}

func TestNewSessionSpawnerReturnsNoRecentWorkspacesWhenSessionsAreOld(t *testing.T) {
	t.Parallel()

	cfg := dreamConfig()
	sessions := &fakeSessionManager{
		infos: []*session.Info{
			{
				ID:          "user-old",
				WorkspaceID: "ws-a",
				Type:        session.SessionTypeUser,
				UpdatedAt:   time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC),
			},
		},
	}
	globalMemoryDir := filepath.Join(t.TempDir(), "memory")
	lockPath := memory.ConsolidationLockPath(globalMemoryDir)
	prior := time.Date(2026, 4, 4, 8, 0, 0, 0, time.UTC)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("os.MkdirAll(lock dir) error = %v", err)
	}
	if err := os.WriteFile(lockPath, nil, 0o644); err != nil {
		t.Fatalf("os.WriteFile(lock) error = %v", err)
	}
	if err := os.Chtimes(lockPath, prior, prior); err != nil {
		t.Fatalf("os.Chtimes(lock) error = %v", err)
	}

	spawner := NewSessionSpawner(sessions, &fakeWorkspaceResolver{}, &cfg, globalMemoryDir)
	err := spawner(context.Background(), "memory-consolidation", "prompt", "")
	if err == nil {
		t.Fatal("spawner() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "no recent workspaces available") {
		t.Fatalf("spawner() error = %v, want no recent workspaces error", err)
	}
}

func TestSpawnSessionWrapsPromptAndStopErrors(t *testing.T) {
	t.Parallel()

	t.Run("prompt error is wrapped", func(t *testing.T) {
		sessions := &fakeSessionManager{promptErr: errors.New("prompt failed")}
		err := spawnSession(context.Background(), sessions, "memory-agent", "goal", "prompt", "ws-1", 0)
		if err == nil {
			t.Fatal("spawnSession() error = nil, want non-nil")
		}
		if !strings.Contains(err.Error(), "prompt dream session") {
			t.Fatalf("spawnSession() error = %v, want prompt context", err)
		}
	})

	t.Run("stop error is joined", func(t *testing.T) {
		stopErr := errors.New("stop failed")
		sessions := &fakeSessionManager{stopErr: stopErr}
		err := spawnSession(context.Background(), sessions, "memory-agent", "goal", "prompt", "ws-1", 0)
		if !errors.Is(err, stopErr) {
			t.Fatalf("spawnSession() error = %v, want stop failure", err)
		}
	})

	t.Run("prompt event errors are surfaced", func(t *testing.T) {
		sessions := &fakeSessionManager{
			promptEvents: []acp.AgentEvent{{Type: acp.EventTypeError, Error: "tool failed"}},
		}
		err := spawnSession(context.Background(), sessions, "memory-agent", "goal", "prompt", "ws-1", 0)
		if err == nil || !strings.Contains(err.Error(), "tool failed") {
			t.Fatalf("spawnSession() error = %v, want prompt event failure", err)
		}
	})

	t.Run("stop uses fresh context after caller cancellation", func(t *testing.T) {
		sessions := &fakeSessionManager{}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := spawnSession(ctx, sessions, "memory-agent", "goal", "prompt", "ws-1", 0); err != nil {
			t.Fatalf("spawnSession() error = %v", err)
		}
		if got, want := sessions.lastStopContextErr(), error(nil); got != want {
			t.Fatalf("Stop() context err = %v, want nil", got)
		}
	})
}

func dreamConfig() aghconfig.Config {
	cfg := aghconfig.DefaultWithHome(aghconfig.HomePaths{})
	cfg.Memory.Enabled = true
	cfg.Memory.Dream.Enabled = true
	cfg.Memory.Dream.Agent = "memory-agent"
	cfg.Memory.Dream.CheckInterval = time.Minute
	return cfg
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func waitForCondition(t *testing.T, label string, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", label)
}

type fakeDreamService struct {
	mu             sync.Mutex
	shouldRun      bool
	shouldRunErr   error
	runErr         error
	shouldRunCalls int
	runCalls       int
	workspaceRefs  []string
}

func (f *fakeDreamService) ShouldRun() (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shouldRunCalls++
	return f.shouldRun, f.shouldRunErr
}

func (f *fakeDreamService) Run(_ context.Context, _ memory.SessionSpawner, workspace string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.runCalls++
	f.workspaceRefs = append(f.workspaceRefs, workspace)
	return f.runErr
}

func (f *fakeDreamService) runCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.runCalls
}

func (f *fakeDreamService) lastWorkspace() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.workspaceRefs) == 0 {
		return ""
	}
	return f.workspaceRefs[len(f.workspaceRefs)-1]
}

type fakeSessionManager struct {
	mu           sync.Mutex
	infos        []*session.Info
	promptErr    error
	promptEvents []acp.AgentEvent
	stopErr      error
	createCalls  []session.CreateOpts
	promptCalls  []struct {
		id  string
		msg string
	}
	stopCalls  []string
	stopCtxErr []error
}

func (f *fakeSessionManager) Create(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls = append(f.createCalls, opts)
	sessionID := fmt.Sprintf("dream-%d", len(f.createCalls))
	return &session.Session{
		ID:          sessionID,
		AgentName:   opts.AgentName,
		WorkspaceID: strings.TrimSpace(opts.Workspace),
		Workspace:   strings.TrimSpace(opts.Workspace),
		Type:        opts.Type,
		State:       session.StateActive,
	}, nil
}

func (f *fakeSessionManager) ListAll(context.Context) ([]*session.Info, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*session.Info(nil), f.infos...), nil
}

func (f *fakeSessionManager) Prompt(_ context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	f.mu.Lock()
	f.promptCalls = append(f.promptCalls, struct {
		id  string
		msg string
	}{id: id, msg: msg})
	promptErr := f.promptErr
	f.mu.Unlock()
	if promptErr != nil {
		return nil, promptErr
	}

	ch := make(chan acp.AgentEvent, len(f.promptEvents))
	for _, event := range f.promptEvents {
		ch <- event
	}
	close(ch)
	return ch, nil
}

func (f *fakeSessionManager) Stop(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopCalls = append(f.stopCalls, id)
	if ctx != nil {
		f.stopCtxErr = append(f.stopCtxErr, ctx.Err())
	} else {
		f.stopCtxErr = append(f.stopCtxErr, context.Canceled)
	}
	return f.stopErr
}

func (f *fakeSessionManager) createCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.createCalls)
}

func (f *fakeSessionManager) createCall(index int) session.CreateOpts {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.createCalls[index]
}

func (f *fakeSessionManager) promptCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.promptCalls)
}

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

func (f *fakeSessionManager) stopCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.stopCalls)
}

func (f *fakeSessionManager) lastStopContextErr() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.stopCtxErr) == 0 {
		return nil
	}
	return f.stopCtxErr[len(f.stopCtxErr)-1]
}

func (f *fakeSessionManager) stopCall(index int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopCalls[index]
}

type fakeWorkspaceResolver struct {
	resolveResolved           workspacepkg.ResolvedWorkspace
	resolveOrRegisterResolved workspacepkg.ResolvedWorkspace
	resolveErr                error
	resolveOrRegisterErr      error
	lastResolveArg            string
	lastResolveOrRegisterArg  string
	resolveCalls              int
	resolveOrRegisterCalls    int
}

func (f *fakeWorkspaceResolver) Register(
	context.Context,
	workspacepkg.RegisterOptions,
) (workspacepkg.Workspace, error) {
	return workspacepkg.Workspace{}, errors.New("unexpected Register call")
}

func (f *fakeWorkspaceResolver) Unregister(context.Context, string) error {
	return errors.New("unexpected Unregister call")
}

func (f *fakeWorkspaceResolver) Update(context.Context, string, workspacepkg.UpdateOptions) error {
	return errors.New("unexpected Update call")
}

func (f *fakeWorkspaceResolver) List(context.Context) ([]workspacepkg.Workspace, error) {
	return nil, errors.New("unexpected List call")
}

func (f *fakeWorkspaceResolver) Get(context.Context, string) (workspacepkg.Workspace, error) {
	return workspacepkg.Workspace{}, errors.New("unexpected Get call")
}

func (f *fakeWorkspaceResolver) Resolve(
	_ context.Context,
	idOrNameOrPath string,
) (workspacepkg.ResolvedWorkspace, error) {
	f.resolveCalls++
	f.lastResolveArg = idOrNameOrPath
	if f.resolveErr != nil {
		return workspacepkg.ResolvedWorkspace{}, f.resolveErr
	}
	return f.resolveResolved, nil
}

func (f *fakeWorkspaceResolver) ResolveOrRegister(
	_ context.Context,
	path string,
) (workspacepkg.ResolvedWorkspace, error) {
	f.resolveOrRegisterCalls++
	f.lastResolveOrRegisterArg = path
	if f.resolveOrRegisterErr != nil {
		return workspacepkg.ResolvedWorkspace{}, f.resolveOrRegisterErr
	}
	return f.resolveOrRegisterResolved, nil
}
