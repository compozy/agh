package httpapi

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/apitest"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
)

type stubSessionManager = apitest.StubSessionManager
type stubObserver = apitest.StubObserver
type stubWorkspaceService = apitest.StubWorkspaceService
type sseRecord = apitest.SSERecord

func newTestHandlers(t *testing.T, manager SessionManager, observer Observer, homePaths aghconfig.HomePaths) *Handlers {
	t.Helper()
	return newTestHandlersWithWorkspace(t, manager, observer, stubWorkspaceService{}, homePaths)
}

func newTestHandlersWithWorkspace(t *testing.T, manager SessionManager, observer Observer, workspaces WorkspaceService, homePaths aghconfig.HomePaths) *Handlers {
	t.Helper()

	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	return newHandlers(handlerConfig{
		sessions:     manager,
		observer:     observer,
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
	return apitest.NewTestHomePaths(t)
}

func writeAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string) {
	t.Helper()
	apitest.WriteAgentDef(t, homePaths, name)
}

func newSessionInfo(id string) *session.SessionInfo {
	return apitest.NewSessionInfo(id)
}

func newSession(id string) *session.Session {
	return apitest.NewSession(id)
}

func performRequest(t *testing.T, engine http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.PerformRequest(t, engine, method, path, body)
}

func performRequestWithHeaders(t *testing.T, engine http.Handler, method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.PerformRequestWithHeaders(t, engine, method, path, body, headers)
}

func decodeJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()
	apitest.DecodeJSONResponse(t, recorder, dest)
}

func parseSSE(t *testing.T, body string) []sseRecord {
	t.Helper()
	return apitest.ParseSSE(t, body)
}

func freeTCPPort(t *testing.T) int {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
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
	return apitest.DiscardLogger()
}
