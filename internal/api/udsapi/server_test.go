package udsapi

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
)

func TestNewHonorsOptionsAndDefaults(t *testing.T) {
	homePaths := newTestHomePaths(t)
	socketPath := shortSocketPath(t)
	engine := gin.New()
	startedAt := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return startedAt.Add(time.Second) }
	customLoader := func(name string, _ aghconfig.HomePaths) (aghconfig.AgentDef, error) {
		return aghconfig.AgentDef{Name: name, Provider: "fake", Prompt: "hello"}, nil
	}
	store := memory.NewStore(filepath.Join(t.TempDir(), "memory"))
	dream := &stubDreamTrigger{}
	bridgeService := &stubBridgeService{}
	extensionService := &stubExtensionService{}
	cfg := testConfigWithDisabledNetwork(homePaths)
	cfg.Daemon.Socket = socketPath

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(&cfg),
		WithSocketPath(socketPath),
		WithLogger(discardLogger()),
		WithStartedAt(startedAt),
		WithNow(now),
		WithPollInterval(25*time.Millisecond),
		WithSessionManager(stubSessionManager{}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{}),
		WithBridgeService(bridgeService),
		WithWorkspaceResolver(stubWorkspaceService{}),
		WithSkillsRegistry(stubSkillsRegistry{}),
		WithMemoryStore(store),
		WithDreamTrigger(dream),
		WithAgentLoader(customLoader),
		WithExtensionService(extensionService),
		WithEngine(engine),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if server.Path() != socketPath {
		t.Fatalf("Path() = %q, want %q", server.Path(), socketPath)
	}
	if server.engine != engine {
		t.Fatal("expected custom gin engine to be used")
	}
	if server.startedAt != startedAt {
		t.Fatalf("startedAt = %v, want %v", server.startedAt, startedAt)
	}
	if server.now == nil || !server.now().Equal(now()) {
		t.Fatalf("now() = %v, want %v", server.now(), now())
	}
	if server.pollInterval != 25*time.Millisecond {
		t.Fatalf("pollInterval = %v, want 25ms", server.pollInterval)
	}
	if server.handlers.AgentLoader == nil {
		t.Fatal("expected custom agent loader to be installed")
	}
	if server.handlers.MemoryStore != store {
		t.Fatal("expected memory store option to be installed")
	}
	if server.handlers.DreamTrigger != dream {
		t.Fatal("expected dream trigger option to be installed")
	}
	if server.handlers.Bridges != bridgeService {
		t.Fatal("expected bridge service option to be installed")
	}
	if server.handlers.Extensions != extensionService {
		t.Fatal("expected extension service option to be installed")
	}
	if server.extensions == nil || server.handlers.Extensions == nil {
		t.Fatal("expected extension service option to be installed")
	}
}

func TestPathHandlesNilServer(t *testing.T) {
	var server *Server
	if server.Path() != "" {
		t.Fatalf("Path(nil) = %q, want empty string", server.Path())
	}
}

func TestNewRejectsOverlongSocketPath(t *testing.T) {
	t.Parallel()

	t.Run("Should reject socket paths that exceed the portable Unix limit", func(t *testing.T) {
		t.Parallel()

		homePaths := newTestHomePaths(t)
		socketPath := "/tmp/" + strings.Repeat("a", maxSocketPathBytes)
		cfg := testConfigWithDisabledNetwork(homePaths)
		cfg.Daemon.Socket = socketPath

		_, err := New(
			WithHomePaths(homePaths),
			WithConfig(&cfg),
			WithSocketPath(socketPath),
			WithSessionManager(stubSessionManager{}),
			WithTaskService(stubTaskManager{}),
			WithObserver(stubObserver{}),
			WithWorkspaceResolver(stubWorkspaceService{}),
		)
		if !errors.Is(err, ErrSocketPathTooLong) {
			t.Fatalf("New() error = %v, want ErrSocketPathTooLong", err)
		}
	})
}

func TestNewRequiresSessionManagerTaskServiceObserverAndWorkspaceResolver(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	testCases := []struct {
		name    string
		opts    []Option
		wantErr error
	}{
		{
			name: "Should require a session manager",
			opts: []Option{
				WithHomePaths(homePaths),
				WithTaskService(stubTaskManager{}),
				WithObserver(stubObserver{}),
				WithWorkspaceResolver(stubWorkspaceService{}),
			},
			wantErr: ErrSessionManagerRequired,
		},
		{
			name: "Should require a task service",
			opts: []Option{
				WithHomePaths(homePaths),
				WithSessionManager(stubSessionManager{}),
				WithObserver(stubObserver{}),
				WithWorkspaceResolver(stubWorkspaceService{}),
			},
			wantErr: ErrTaskServiceRequired,
		},
		{
			name: "Should require an observer",
			opts: []Option{
				WithHomePaths(homePaths),
				WithSessionManager(stubSessionManager{}),
				WithTaskService(stubTaskManager{}),
				WithWorkspaceResolver(stubWorkspaceService{}),
			},
			wantErr: ErrObserverRequired,
		},
		{
			name: "Should require a workspace resolver",
			opts: []Option{
				WithHomePaths(homePaths),
				WithSessionManager(stubSessionManager{}),
				WithTaskService(stubTaskManager{}),
				WithObserver(stubObserver{}),
			},
			wantErr: ErrWorkspaceResolverRequired,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := New(tc.opts...); err == nil || !errors.Is(err, tc.wantErr) {
				t.Fatalf("New() error = %v, want %v", err, tc.wantErr)
			}
		})
	}

	t.Run("Should allow missing skills registry", func(t *testing.T) {
		t.Parallel()

		if _, err := New(
			WithHomePaths(homePaths),
			WithSessionManager(stubSessionManager{}),
			WithTaskService(stubTaskManager{}),
			WithObserver(stubObserver{}),
			WithWorkspaceResolver(stubWorkspaceService{}),
		); err != nil {
			t.Fatalf("New() without skills registry error = %v, want nil", err)
		}
	})
}

func TestServerStartAndShutdownCreatesAndRemovesSocket(t *testing.T) {
	homePaths := newTestHomePaths(t)
	socketPath := shortSocketPath(t)
	cfg := testConfigWithDisabledNetwork(homePaths)
	cfg.Daemon.Socket = socketPath

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(&cfg),
		WithSocketPath(socketPath),
		WithLogger(discardLogger()),
		WithSessionManager(stubSessionManager{
			ListAllFn: func(context.Context) ([]*session.Info, error) { return nil, nil },
		}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{
			HealthFn: func(context.Context) (observe.Health, error) { return observe.Health{Status: "ok"}, nil },
		}),
		WithWorkspaceResolver(stubWorkspaceService{}),
		WithSkillsRegistry(stubSkillsRegistry{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	info, err := os.Lstat(socketPath)
	if err != nil {
		t.Fatalf("os.Lstat(socket) error = %v", err)
	}
	if info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("socket mode = %v, want unix socket", info.Mode())
	}

	client := newUnixClient(t, socketPath)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"http://unix/api/daemon/status",
		http.NoBody,
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error = %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
	if _, err := os.Stat(socketPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket after shutdown: stat error = %v, want os.ErrNotExist", err)
	}
}

func TestServerStartRejectsNilContextAndDuplicateStart(t *testing.T) {
	homePaths := newTestHomePaths(t)
	socketPath := shortSocketPath(t)
	cfg := testConfigWithDisabledNetwork(homePaths)
	cfg.Daemon.Socket = socketPath

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(&cfg),
		WithSocketPath(socketPath),
		WithLogger(discardLogger()),
		WithSessionManager(stubSessionManager{}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{}),
		WithWorkspaceResolver(stubWorkspaceService{}),
		WithSkillsRegistry(stubSkillsRegistry{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() {
		_ = server.Shutdown(context.Background())
	}()
	if err := server.Start(context.Background()); err == nil {
		t.Fatal("Start(second) error = nil, want non-nil")
	}
}

func TestServerStartRejectsRegularFileAtSocketPath(t *testing.T) {
	homePaths := newTestHomePaths(t)
	socketPath := shortSocketPath(t)
	cfg := testConfigWithDisabledNetwork(homePaths)
	cfg.Daemon.Socket = socketPath
	if err := os.WriteFile(socketPath, []byte("not-a-socket"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(socketPath) error = %v", err)
	}

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(&cfg),
		WithSocketPath(socketPath),
		WithLogger(discardLogger()),
		WithSessionManager(stubSessionManager{}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{}),
		WithWorkspaceResolver(stubWorkspaceService{}),
		WithSkillsRegistry(stubSkillsRegistry{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := server.Start(context.Background()); err == nil {
		t.Fatal("Start() error = nil, want non-nil")
	}
}

func TestEnsureSocketParentDirAndWaitForServeDone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "daemon.sock")
	if err := ensureSocketParentDir(path); err != nil {
		t.Fatalf("ensureSocketParentDir() error = %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("os.Stat(parent dir) error = %v", err)
	}

	done := make(chan struct{})
	close(done)
	if err := waitForServeDone(context.Background(), done); err != nil {
		t.Fatalf("waitForServeDone(done) error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := waitForServeDone(ctx, make(chan struct{})); err == nil {
		t.Fatal("waitForServeDone(timeout) error = nil, want non-nil")
	}
}
