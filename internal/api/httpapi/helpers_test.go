package httpapi

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
)

type stubSessionManager = testutil.StubSessionManager
type stubObserver = testutil.StubObserver
type stubTaskManager = testutil.StubTaskManager
type stubBridgeService = testutil.StubBridgeService
type stubResourceService = testutil.StubResourceService
type stubWorkspaceService = testutil.StubWorkspaceService
type sseRecord = testutil.SSERecord

func newTestHandlers(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()
	return newTestHandlersWithAutomationBridgesTasksAndWorkspace(
		t,
		manager,
		observer,
		nil,
		stubTaskManager{},
		nil,
		stubWorkspaceService{},
		homePaths,
	)
}

func newTestHandlersWithBridges(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	bridges core.BridgeService,
	workspaces core.WorkspaceService,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()
	return newTestHandlersWithAutomationBridgesTasksAndWorkspace(
		t,
		manager,
		observer,
		nil,
		stubTaskManager{},
		bridges,
		workspaces,
		homePaths,
	)
}

func newTestHandlersWithAutomationBridgesTasksAndWorkspace(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	automation core.AutomationManager,
	tasks core.TaskService,
	bridges core.BridgeService,
	workspaces core.WorkspaceService,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	return newHandlers(&handlerConfig{
		sessions:     manager,
		tasks:        tasks,
		observer:     observer,
		automation:   automation,
		bridges:      bridges,
		workspaces:   workspaces,
		staticFS:     mustStaticFS(t),
		homePaths:    homePaths,
		config:       cfg,
		logger:       discardLogger(),
		startedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:          func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval: 5 * time.Millisecond,
		agentLoader:  aghconfig.LoadAgentDef,
		httpPort:     cfg.HTTP.Port,
	})
}

func newTestHandlersWithWorkspace(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	workspaces core.WorkspaceService,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()

	return newTestHandlersWithBridges(t, manager, observer, nil, workspaces, homePaths)
}

func newTestHandlersWithResources(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	resources core.ResourceService,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	return newHandlers(&handlerConfig{
		sessions:     manager,
		tasks:        stubTaskManager{},
		observer:     observer,
		resources:    resources,
		workspaces:   stubWorkspaceService{},
		staticFS:     mustStaticFS(t),
		homePaths:    homePaths,
		config:       cfg,
		logger:       discardLogger(),
		startedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:          func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval: 5 * time.Millisecond,
		agentLoader:  aghconfig.LoadAgentDef,
		httpPort:     cfg.HTTP.Port,
	})
}

func newTestHandlersWithResourcesAndAuth(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	resources core.ResourceService,
	auth ...gin.HandlerFunc,
) *Handlers {
	t.Helper()

	homePaths := newTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	return newHandlers(&handlerConfig{
		sessions:     manager,
		tasks:        stubTaskManager{},
		observer:     observer,
		resources:    resources,
		workspaces:   stubWorkspaceService{},
		staticFS:     mustStaticFS(t),
		homePaths:    homePaths,
		config:       cfg,
		logger:       discardLogger(),
		startedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:          func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval: 5 * time.Millisecond,
		agentLoader:  aghconfig.LoadAgentDef,
		httpPort:     cfg.HTTP.Port,
		resourceAuth: append([]gin.HandlerFunc(nil), auth...),
	})
}

func newTestRouter(t *testing.T, handlers *Handlers) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestLoggingMiddleware(discardLogger()))
	engine.Use(corsMiddleware("127.0.0.1"))
	engine.Use(errorMiddleware())
	RegisterRoutes(engine, handlers)
	return engine
}

func mustStaticFS(t *testing.T) fs.FS {
	t.Helper()

	staticFS, err := newStaticFS()
	if err != nil {
		t.Fatalf("newStaticFS() error = %v", err)
	}

	return staticFS
}

func newTestHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()
	return testutil.NewTestHomePaths(t)
}

func writeAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string) {
	t.Helper()
	testutil.WriteAgentDef(t, homePaths, name)
}

func newSessionInfo(id string) *session.Info {
	return testutil.NewSessionInfo(id)
}

func newSession(id string) *session.Session {
	return testutil.NewSession(id)
}

func performRequest(t *testing.T, engine http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.PerformRequest(t, engine, method, path, body)
}

func performRequestWithHeaders(
	t *testing.T,
	engine http.Handler,
	method, path string,
	body []byte,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.PerformRequestWithHeaders(t, engine, method, path, body, headers)
}

func decodeJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()
	testutil.DecodeJSONResponse(t, recorder, dest)
}

func parseSSE(t *testing.T, body string) []sseRecord {
	t.Helper()
	return testutil.ParseSSE(t, body)
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	var listenConfig net.ListenConfig
	ln, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen(:0) error = %v", err)
	}
	defer func() {
		_ = ln.Close()
	}()

	tcpAddr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener addr type = %T, want *net.TCPAddr", ln.Addr())
	}
	return tcpAddr.Port
}

func mustURL(host string, port int, path string) string {
	return fmt.Sprintf("http://%s:%d%s", host, port, path)
}

func discardLogger() *slog.Logger {
	return testutil.DiscardLogger()
}
