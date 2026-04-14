package httpapi

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
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
	engine := gin.New()
	startedAt := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return startedAt.Add(time.Second) }
	customLoader := func(name string, homePaths aghconfig.HomePaths) (aghconfig.AgentDef, error) {
		return aghconfig.AgentDef{Name: name, Provider: "fake", Prompt: "hello"}, nil
	}
	store := memory.NewStore(filepath.Join(t.TempDir(), "memory"))
	dream := &stubDreamTrigger{}
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = freeTCPPort(t)

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithHost(cfg.HTTP.Host),
		WithPort(cfg.HTTP.Port),
		WithLogger(discardLogger()),
		WithStartedAt(startedAt),
		WithNow(now),
		WithPollInterval(25*time.Millisecond),
		WithSessionManager(stubSessionManager{}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{}),
		WithWorkspaceResolver(stubWorkspaceService{}),
		WithMemoryStore(store),
		WithDreamTrigger(dream),
		WithAgentLoader(customLoader),
		WithEngine(engine),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if server.Port() != cfg.HTTP.Port {
		t.Fatalf("Port() = %d, want %d", server.Port(), cfg.HTTP.Port)
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
}

func TestPortHandlesNilServer(t *testing.T) {
	var server *Server
	if server.Port() != 0 {
		t.Fatalf("Port(nil) = %d, want 0", server.Port())
	}
}

func TestNewRequiresSessionManagerTaskServiceObserverAndWorkspaceResolver(t *testing.T) {
	homePaths := newTestHomePaths(t)

	if _, err := New(WithHomePaths(homePaths), WithObserver(stubObserver{})); err == nil {
		t.Fatal("New() without session manager error = nil, want non-nil")
	}
	if _, err := New(WithHomePaths(homePaths), WithSessionManager(stubSessionManager{})); err == nil {
		t.Fatal("New() without task service error = nil, want non-nil")
	}
	if _, err := New(WithHomePaths(homePaths), WithSessionManager(stubSessionManager{}), WithTaskService(stubTaskManager{})); err == nil {
		t.Fatal("New() without observer error = nil, want non-nil")
	}
	if _, err := New(WithHomePaths(homePaths), WithSessionManager(stubSessionManager{}), WithTaskService(stubTaskManager{}), WithObserver(stubObserver{})); err == nil {
		t.Fatal("New() without workspace resolver error = nil, want non-nil")
	}
}

func TestServerStartAndShutdownServeRequests(t *testing.T) {
	homePaths := newTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = freeTCPPort(t)

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithHost(cfg.HTTP.Host),
		WithPort(cfg.HTTP.Port),
		WithLogger(discardLogger()),
		WithSessionManager(stubSessionManager{
			ListAllFn: func(context.Context) ([]*session.SessionInfo, error) { return nil, nil },
		}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{
			HealthFn: func(context.Context) (observe.Health, error) { return observe.Health{Status: "ok"}, nil },
		}),
		WithWorkspaceResolver(stubWorkspaceService{}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := server.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	resp, err := http.Get(mustURL(cfg.HTTP.Host, server.Port(), "/api/daemon/status"))
	if err != nil {
		t.Fatalf("http.Get() error = %v", err)
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

	_, err = http.Get(mustURL(cfg.HTTP.Host, server.Port(), "/api/daemon/status"))
	if err == nil {
		t.Fatal("expected request after shutdown to fail")
	}
}

func TestServerStartRejectsNilContextAndDuplicateStart(t *testing.T) {
	homePaths := newTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = freeTCPPort(t)

	server, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithHost(cfg.HTTP.Host),
		WithPort(cfg.HTTP.Port),
		WithLogger(discardLogger()),
		WithSessionManager(stubSessionManager{}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{}),
		WithWorkspaceResolver(stubWorkspaceService{}),
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

func TestWaitForServeDone(t *testing.T) {
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

func TestServerStartReportsListenFailure(t *testing.T) {
	homePaths := newTestHomePaths(t)
	port := freeTCPPort(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = port

	first, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithHost(cfg.HTTP.Host),
		WithPort(cfg.HTTP.Port),
		WithLogger(discardLogger()),
		WithSessionManager(stubSessionManager{}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{}),
		WithWorkspaceResolver(stubWorkspaceService{}),
	)
	if err != nil {
		t.Fatalf("first New() error = %v", err)
	}
	if err := first.Start(context.Background()); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	defer func() {
		_ = first.Shutdown(context.Background())
	}()

	second, err := New(
		WithHomePaths(homePaths),
		WithConfig(cfg),
		WithHost(cfg.HTTP.Host),
		WithPort(cfg.HTTP.Port),
		WithLogger(discardLogger()),
		WithSessionManager(stubSessionManager{}),
		WithTaskService(stubTaskManager{}),
		WithObserver(stubObserver{}),
		WithWorkspaceResolver(stubWorkspaceService{}),
	)
	if err != nil {
		t.Fatalf("second New() error = %v", err)
	}
	if err := second.Start(context.Background()); err == nil {
		t.Fatal("second Start() error = nil, want non-nil")
	}
}

func TestShutdownNilServerIsSafe(t *testing.T) {
	var server *Server
	if err := server.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown(nil) error = %v", err)
	}
}

func TestShutdownTimeoutIsReported(t *testing.T) {
	server := &Server{serveDone: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := server.Shutdown(ctx)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Shutdown(timeout) error = %v, want deadline exceeded", err)
	}
}
