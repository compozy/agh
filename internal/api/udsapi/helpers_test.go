package udsapi

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

var errStubWorkspaceServiceNotImplemented = testutil.ErrStubWorkspaceServiceNotImplemented

type stubSessionManager = testutil.StubSessionManager
type stubObserver = testutil.StubObserver
type stubBridgeService = testutil.StubBridgeService
type stubNetworkService = testutil.StubNetworkService
type stubWorkspaceService = testutil.StubWorkspaceService
type stubSkillsRegistry = testutil.StubSkillsRegistry
type sseRecord = testutil.SSERecord

func newTestHandlers(t *testing.T, manager core.SessionManager, observer core.Observer, homePaths aghconfig.HomePaths) *Handlers {
	t.Helper()
	return newTestHandlersWithRuntime(t, manager, observer, nil, nil, stubWorkspaceService{}, nil, homePaths)
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
	return newTestHandlersWithRuntime(t, manager, observer, nil, bridges, workspaces, nil, homePaths)
}

func newTestHandlersWithExtensions(t *testing.T, manager core.SessionManager, observer core.Observer, extensions ExtensionService, homePaths aghconfig.HomePaths) *Handlers {
	t.Helper()
	return newTestHandlersWithRuntime(t, manager, observer, nil, nil, stubWorkspaceService{}, extensions, homePaths)
}

func newTestHandlersWithRuntime(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	automation core.AutomationManager,
	bridges core.BridgeService,
	workspaces core.WorkspaceService,
	extensions ExtensionService,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()

	return newHandlers(handlerConfig{
		sessions:     manager,
		observer:     observer,
		automation:   automation,
		bridges:      bridges,
		workspaces:   workspaces,
		homePaths:    homePaths,
		config:       aghconfig.DefaultWithHome(homePaths),
		logger:       discardLogger(),
		startedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:          func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval: 5 * time.Millisecond,
		agentLoader:  aghconfig.LoadAgentDef,
		extensions:   extensions,
	})
}

func newTestHandlersWithWorkspace(t *testing.T, manager core.SessionManager, observer core.Observer, workspaces core.WorkspaceService, homePaths aghconfig.HomePaths) *Handlers {
	t.Helper()

	return newTestHandlersWithBridges(t, manager, observer, nil, workspaces, homePaths)
}

func newTestRouter(t *testing.T, handlers *Handlers) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	RegisterRoutes(engine, handlers)
	return engine
}

func newTestHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()
	return testutil.NewTestHomePaths(t)
}

func shortSocketPath(t *testing.T) string {
	t.Helper()

	path := filepath.Join(os.TempDir(), "udsapi-"+strconv.FormatInt(time.Now().UTC().UnixNano(), 10)+".sock")
	t.Cleanup(func() {
		_ = os.Remove(path)
	})
	return path
}

func writeAgentDef(t *testing.T, homePaths aghconfig.HomePaths, name string) {
	t.Helper()
	testutil.WriteAgentDef(t, homePaths, name)
}

func newSessionInfo(id string) *session.SessionInfo {
	return testutil.NewSessionInfo(id)
}

func newSession(id string) *session.Session {
	return testutil.NewSession(id)
}

func performRequest(t *testing.T, engine http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	return testutil.PerformRequest(t, engine, method, path, body)
}

func decodeJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder, dest any) {
	t.Helper()
	testutil.DecodeJSONResponse(t, recorder, dest)
}

func decodeSSEData(t *testing.T, record sseRecord, dest any) {
	t.Helper()
	testutil.DecodeSSEData(t, record, dest)
}

func mustJSONBody(t *testing.T, value any) []byte {
	t.Helper()
	return testutil.MustJSONBody(t, value)
}

func parseSSE(t *testing.T, body string) []sseRecord {
	t.Helper()
	return testutil.ParseSSE(t, body)
}

func TestStubWorkspaceServiceDefaultsReportUnconfiguredMethods(t *testing.T) {
	t.Parallel()

	service := stubWorkspaceService{}

	if _, err := service.Register(context.Background(), workspacepkg.RegisterOptions{}); !errors.Is(err, errStubWorkspaceServiceNotImplemented) {
		t.Fatalf("Register() error = %v, want %v", err, errStubWorkspaceServiceNotImplemented)
	}
	if _, err := service.ResolveOrRegister(context.Background(), "/workspace"); !errors.Is(err, errStubWorkspaceServiceNotImplemented) {
		t.Fatalf("ResolveOrRegister() error = %v, want %v", err, errStubWorkspaceServiceNotImplemented)
	}
}

func newUnixClient(t *testing.T, socketPath string) *http.Client {
	t.Helper()

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{Transport: transport}
}

func discardLogger() *slog.Logger {
	return testutil.DiscardLogger()
}
