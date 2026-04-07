package apicore_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/apicore"
	"github.com/pedronauck/agh/internal/apitest"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/memory"
)

type stubDreamTrigger struct {
	Triggered bool
	Reason    string
	Err       error
	Last      time.Time
	LastErr   error
	EnabledFn bool
	Calls     int
	Workspace string
}

func (s *stubDreamTrigger) Trigger(_ context.Context, workspace string) (bool, string, error) {
	s.Calls++
	s.Workspace = workspace
	return s.Triggered, s.Reason, s.Err
}

func (s *stubDreamTrigger) LastConsolidatedAt() (time.Time, error) {
	return s.Last, s.LastErr
}

func (s *stubDreamTrigger) Enabled() bool {
	return s.EnabledFn
}

type handlerFixture struct {
	Handlers  *apicore.BaseHandlers
	Engine    *gin.Engine
	HomePaths aghconfig.HomePaths
}

func newHandlerFixture(
	t *testing.T,
	manager apitest.StubSessionManager,
	observer apitest.StubObserver,
	workspaces apitest.StubWorkspaceService,
	store *memory.Store,
	dream apicore.DreamTrigger,
) handlerFixture {
	t.Helper()

	gin.SetMode(gin.TestMode)
	homePaths := apitest.NewTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123
	cfg.Daemon.Socket = "/tmp/apicore-test.sock"

	handlers := apicore.NewBaseHandlers(apicore.BaseHandlerConfig{
		TransportName:                "apicore-test",
		MaskInternalErrors:           false,
		IncludeSessionWorkspaceInSSE: true,
		Sessions:                     manager,
		Observer:                     observer,
		Workspaces:                   workspaces,
		MemoryStore:                  store,
		DreamTrigger:                 dream,
		HomePaths:                    homePaths,
		Config:                       cfg,
		Logger:                       apitest.DiscardLogger(),
		StartedAt:                    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		Now: func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC)
		},
		PollInterval: 5 * time.Millisecond,
		HTTPPort:     cfg.HTTP.Port,
	})

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.GET("/sessions", handlers.ListSessions)
	engine.POST("/sessions", handlers.CreateSession)
	engine.GET("/sessions/:id", handlers.GetSession)
	engine.DELETE("/sessions/:id", handlers.StopSession)
	engine.POST("/sessions/:id/resume", handlers.ResumeSession)
	engine.GET("/sessions/:id/events", handlers.SessionEvents)
	engine.GET("/sessions/:id/history", handlers.SessionHistory)
	engine.GET("/sessions/:id/transcript", handlers.SessionTranscript)
	engine.GET("/sessions/:id/stream", handlers.StreamSession)
	engine.GET("/agents", handlers.ListAgents)
	engine.GET("/agents/:name", handlers.GetAgent)
	engine.GET("/observe/events", handlers.ObserveEvents)
	engine.GET("/observe/events/stream", handlers.StreamObserveEvents)
	engine.GET("/observe/health", handlers.Health)
	engine.GET("/daemon/status", handlers.DaemonStatus)
	engine.GET("/memory", handlers.ListMemory)
	engine.GET("/memory/:filename", handlers.ReadMemory)
	engine.PUT("/memory/:filename", handlers.WriteMemory)
	engine.DELETE("/memory/:filename", handlers.DeleteMemory)
	engine.POST("/memory/consolidate", handlers.ConsolidateMemory)
	engine.POST("/workspaces", handlers.CreateWorkspace)
	engine.GET("/workspaces", handlers.ListWorkspaces)
	engine.GET("/workspaces/:id", handlers.GetWorkspace)
	engine.PATCH("/workspaces/:id", handlers.UpdateWorkspace)
	engine.DELETE("/workspaces/:id", handlers.DeleteWorkspace)
	engine.POST("/workspaces/resolve", handlers.ResolveWorkspace)

	return handlerFixture{
		Handlers:  handlers,
		Engine:    engine,
		HomePaths: homePaths,
	}
}

func performRequest(t *testing.T, engine http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.PerformRequest(t, engine, method, path, body)
}
