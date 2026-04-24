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
	settingspkg "github.com/pedronauck/agh/internal/settings"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

var errStubWorkspaceServiceNotImplemented = testutil.ErrStubWorkspaceServiceNotImplemented

type stubSessionManager = testutil.StubSessionManager
type stubObserver = testutil.StubObserver
type stubTaskManager = testutil.StubTaskManager
type stubBridgeService = testutil.StubBridgeService
type stubNetworkService = testutil.StubNetworkService
type stubResourceService = testutil.StubResourceService
type stubWorkspaceService = testutil.StubWorkspaceService
type stubSkillsRegistry = testutil.StubSkillsRegistry
type sseRecord = testutil.SSERecord

type stubSettingsService struct {
	GetSectionFn                func(context.Context, settingspkg.SectionRequest) (settingspkg.SectionEnvelope, error)
	UpdateSectionFn             func(context.Context, settingspkg.SectionUpdateRequest) (settingspkg.MutationResult, error)
	ListCollectionFn            func(context.Context, settingspkg.CollectionRequest) (settingspkg.CollectionEnvelope, error)
	PutCollectionItemFn         func(context.Context, settingspkg.CollectionItemPutRequest) (settingspkg.MutationResult, error)
	DeleteCollectionItemFn      func(context.Context, settingspkg.CollectionItemDeleteRequest) (settingspkg.MutationResult, error)
	LastGetSectionRequest       settingspkg.SectionRequest
	LastUpdateSectionRequest    settingspkg.SectionUpdateRequest
	LastListCollectionRequest   settingspkg.CollectionRequest
	LastPutCollectionRequest    settingspkg.CollectionItemPutRequest
	LastDeleteCollectionRequest settingspkg.CollectionItemDeleteRequest
}

func (s *stubSettingsService) GetSection(
	ctx context.Context,
	req settingspkg.SectionRequest,
) (settingspkg.SectionEnvelope, error) {
	s.LastGetSectionRequest = req
	if s.GetSectionFn == nil {
		return settingsTestSectionEnvelope(req.Section, req.Scope, req.WorkspaceID), nil
	}
	return s.GetSectionFn(ctx, req)
}

func (s *stubSettingsService) UpdateSection(
	ctx context.Context,
	req settingspkg.SectionUpdateRequest,
) (settingspkg.MutationResult, error) {
	s.LastUpdateSectionRequest = req
	if s.UpdateSectionFn == nil {
		return settingspkg.MutationResult{
			Section:         req.Section,
			Scope:           req.Scope,
			WorkspaceID:     req.WorkspaceID,
			Behavior:        settingspkg.MutationBehaviorRestartRequired,
			RestartRequired: true,
		}, nil
	}
	return s.UpdateSectionFn(ctx, req)
}

func (s *stubSettingsService) ListCollection(
	ctx context.Context,
	req settingspkg.CollectionRequest,
) (settingspkg.CollectionEnvelope, error) {
	s.LastListCollectionRequest = req
	if s.ListCollectionFn == nil {
		return settingsTestCollectionEnvelope(req.Collection, req.Scope, req.WorkspaceID), nil
	}
	return s.ListCollectionFn(ctx, req)
}

func (s *stubSettingsService) PutCollectionItem(
	ctx context.Context,
	req settingspkg.CollectionItemPutRequest,
) (settingspkg.MutationResult, error) {
	s.LastPutCollectionRequest = req
	if s.PutCollectionItemFn == nil {
		return settingspkg.MutationResult{
			Section:         settingspkg.SectionName(req.Collection),
			Scope:           req.Scope,
			WorkspaceID:     req.WorkspaceID,
			Behavior:        settingspkg.MutationBehaviorRestartRequired,
			RestartRequired: true,
		}, nil
	}
	return s.PutCollectionItemFn(ctx, req)
}

func (s *stubSettingsService) DeleteCollectionItem(
	ctx context.Context,
	req settingspkg.CollectionItemDeleteRequest,
) (settingspkg.MutationResult, error) {
	s.LastDeleteCollectionRequest = req
	if s.DeleteCollectionItemFn == nil {
		return settingspkg.MutationResult{
			Section:         settingspkg.SectionName(req.Collection),
			Scope:           req.Scope,
			WorkspaceID:     req.WorkspaceID,
			Behavior:        settingspkg.MutationBehaviorRestartRequired,
			RestartRequired: true,
		}, nil
	}
	return s.DeleteCollectionItemFn(ctx, req)
}

type stubSettingsRestartController struct {
	RequestRestartFn      func(context.Context) (core.SettingsRestartOperation, error)
	GetRestartOperationFn func(context.Context, string) (core.SettingsRestartOperation, error)
	RequestRestartCalls   int
	GetRestartOperationID string
}

func (s *stubSettingsRestartController) RequestRestart(ctx context.Context) (core.SettingsRestartOperation, error) {
	s.RequestRestartCalls++
	if s.RequestRestartFn == nil {
		return core.SettingsRestartOperation{
			OperationID:        "op-123",
			Status:             "pending",
			ActiveSessionCount: 1,
		}, nil
	}
	return s.RequestRestartFn(ctx)
}

func (s *stubSettingsRestartController) GetRestartOperation(
	ctx context.Context,
	operationID string,
) (core.SettingsRestartOperation, error) {
	s.GetRestartOperationID = operationID
	if s.GetRestartOperationFn == nil {
		return core.SettingsRestartOperation{
			OperationID:        operationID,
			Status:             "ready",
			ActiveSessionCount: 1,
			StartedAt:          time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			UpdatedAt:          time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
		}, nil
	}
	return s.GetRestartOperationFn(ctx, operationID)
}

func newTestHandlers(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()
	return newTestHandlersWithRuntime(
		t,
		manager,
		observer,
		nil,
		stubTaskManager{},
		nil,
		stubWorkspaceService{},
		nil,
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
	return newTestHandlersWithRuntime(t, manager, observer, nil, stubTaskManager{}, bridges, workspaces, nil, homePaths)
}

func newTestHandlersWithExtensions(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	extensions ExtensionService,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()
	return newTestHandlersWithRuntime(
		t,
		manager,
		observer,
		nil,
		stubTaskManager{},
		nil,
		stubWorkspaceService{},
		extensions,
		homePaths,
	)
}

func newTestHandlersWithSettingsAndExtensions(
	t *testing.T,
	settings core.SettingsService,
	restart core.SettingsRestartController,
	extensions ExtensionService,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()

	cfg := testConfigWithDisabledNetwork(homePaths)
	return newHandlers(&handlerConfig{
		sessions:        stubSessionManager{},
		tasks:           stubTaskManager{},
		observer:        stubObserver{},
		workspaces:      stubWorkspaceService{},
		settings:        settings,
		settingsRestart: restart,
		extensions:      extensions,
		homePaths:       homePaths,
		config:          cfg,
		logger:          discardLogger(),
		startedAt:       time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:             func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval:    5 * time.Millisecond,
		agentLoader:     aghconfig.LoadAgentDef,
	})
}

func newTestHandlersWithRuntime(
	t *testing.T,
	manager core.SessionManager,
	observer core.Observer,
	automation core.AutomationManager,
	tasks core.TaskService,
	bridges core.BridgeService,
	workspaces core.WorkspaceService,
	extensions ExtensionService,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()

	cfg := testConfigWithDisabledNetwork(homePaths)
	return newHandlers(&handlerConfig{
		sessions:     manager,
		tasks:        tasks,
		observer:     observer,
		automation:   automation,
		bridges:      bridges,
		workspaces:   workspaces,
		homePaths:    homePaths,
		config:       cfg,
		logger:       discardLogger(),
		startedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:          func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval: 5 * time.Millisecond,
		agentLoader:  aghconfig.LoadAgentDef,
		extensions:   extensions,
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

	cfg := testConfigWithDisabledNetwork(homePaths)
	return newHandlers(&handlerConfig{
		sessions:     manager,
		tasks:        stubTaskManager{},
		observer:     observer,
		resources:    resources,
		workspaces:   stubWorkspaceService{},
		homePaths:    homePaths,
		config:       cfg,
		logger:       discardLogger(),
		startedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:          func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval: 5 * time.Millisecond,
		agentLoader:  aghconfig.LoadAgentDef,
	})
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

func testConfigWithDisabledNetwork(homePaths aghconfig.HomePaths) aghconfig.Config {
	return testutil.ConfigWithDisabledNetwork(homePaths)
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

	if _, err := service.Register(
		context.Background(),
		workspacepkg.RegisterOptions{},
	); !errors.Is(
		err,
		errStubWorkspaceServiceNotImplemented,
	) {
		t.Fatalf("Register() error = %v, want %v", err, errStubWorkspaceServiceNotImplemented)
	}
	if _, err := service.ResolveOrRegister(
		context.Background(),
		"/workspace",
	); !errors.Is(
		err,
		errStubWorkspaceServiceNotImplemented,
	) {
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

func settingsTestSectionEnvelope(
	section settingspkg.SectionName,
	scope settingspkg.ScopeKind,
	workspaceID string,
) settingspkg.SectionEnvelope {
	envelope := settingspkg.SectionEnvelope{
		Section:         section,
		Scope:           scope,
		WorkspaceID:     workspaceID,
		AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
	}
	switch section {
	case settingspkg.SectionGeneral:
		envelope.General = &settingspkg.GeneralSection{}
	case settingspkg.SectionMemory:
		envelope.Memory = &settingspkg.MemorySection{}
	case settingspkg.SectionSkills:
		envelope.Skills = &settingspkg.SkillsSection{}
	case settingspkg.SectionAutomation:
		envelope.Automation = &settingspkg.AutomationSection{}
	case settingspkg.SectionNetwork:
		envelope.Network = &settingspkg.NetworkSection{}
	case settingspkg.SectionObservability:
		envelope.Observability = &settingspkg.ObservabilitySection{}
	case settingspkg.SectionHooksExtensions:
		envelope.HooksExtensions = &settingspkg.HooksExtensionsSection{}
	}
	return envelope
}

func settingsTestCollectionEnvelope(
	collection settingspkg.CollectionName,
	scope settingspkg.ScopeKind,
	workspaceID string,
) settingspkg.CollectionEnvelope {
	envelope := settingspkg.CollectionEnvelope{
		Collection:      collection,
		Scope:           scope,
		WorkspaceID:     workspaceID,
		AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
	}
	switch collection {
	case settingspkg.CollectionProviders:
		envelope.Providers = []settingspkg.ProviderItem{{
			Name:     "demo",
			Settings: settingspkg.ProviderSettings{Command: "codex"},
		}}
	case settingspkg.CollectionMCPServers:
		envelope.MCPServers = []settingspkg.MCPServerItem{{
			Name:    "server-a",
			Command: "mcpd",
			Scope:   scope,
		}}
	case settingspkg.CollectionEnvironments:
		envelope.Environments = []settingspkg.EnvironmentItem{{
			Name:    "demo",
			Profile: aghconfig.EnvironmentProfile{Backend: "local"},
		}}
	case settingspkg.CollectionHooks:
		envelope.Hooks = []settingspkg.HookItem{}
	}
	return envelope
}
