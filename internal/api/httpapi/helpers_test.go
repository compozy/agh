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
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	settingspkg "github.com/pedronauck/agh/internal/settings"
)

type stubSessionManager = testutil.StubSessionManager
type stubObserver = testutil.StubObserver
type stubTaskManager = testutil.StubTaskManager
type stubBridgeService = testutil.StubBridgeService
type stubResourceService = testutil.StubResourceService
type stubWorkspaceService = testutil.StubWorkspaceService
type sseRecord = testutil.SSERecord

type stubExtensionService struct {
	ListFn    func(context.Context) ([]contract.ExtensionPayload, error)
	InstallFn func(context.Context, contract.InstallExtensionRequest) (contract.ExtensionPayload, error)
	EnableFn  func(context.Context, string) (contract.ExtensionPayload, error)
	DisableFn func(context.Context, string) (contract.ExtensionPayload, error)
	StatusFn  func(context.Context, string) (contract.ExtensionPayload, error)
}

func (s stubExtensionService) List(ctx context.Context) ([]contract.ExtensionPayload, error) {
	if s.ListFn == nil {
		return nil, nil
	}
	return s.ListFn(ctx)
}

func (s stubExtensionService) Install(
	ctx context.Context,
	req contract.InstallExtensionRequest,
) (contract.ExtensionPayload, error) {
	if s.InstallFn == nil {
		return contract.ExtensionPayload{}, nil
	}
	return s.InstallFn(ctx, req)
}

func (s stubExtensionService) Enable(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if s.EnableFn == nil {
		return contract.ExtensionPayload{}, nil
	}
	return s.EnableFn(ctx, name)
}

func (s stubExtensionService) Disable(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if s.DisableFn == nil {
		return contract.ExtensionPayload{}, nil
	}
	return s.DisableFn(ctx, name)
}

func (s stubExtensionService) Status(ctx context.Context, name string) (contract.ExtensionPayload, error) {
	if s.StatusFn == nil {
		return contract.ExtensionPayload{}, nil
	}
	return s.StatusFn(ctx, name)
}

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

	cfg := testConfigWithDisabledNetwork(homePaths)
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
		boundHost:    cfg.HTTP.Host,
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

	cfg := testConfigWithDisabledNetwork(homePaths)
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
		boundHost:    cfg.HTTP.Host,
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
	cfg := testConfigWithDisabledNetwork(homePaths)
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
		boundHost:    cfg.HTTP.Host,
		logger:       discardLogger(),
		startedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:          func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval: 5 * time.Millisecond,
		agentLoader:  aghconfig.LoadAgentDef,
		httpPort:     cfg.HTTP.Port,
		resourceAuth: append([]gin.HandlerFunc(nil), auth...),
	})
}

func newTestHandlersWithSettingsAndExtensions(
	t *testing.T,
	boundHost string,
	settings core.SettingsService,
	restart core.SettingsRestartController,
	extensions ExtensionService,
	homePaths aghconfig.HomePaths,
) *Handlers {
	t.Helper()

	cfg := testConfigWithDisabledNetwork(homePaths)
	cfg.HTTP.Host = boundHost
	cfg.HTTP.Port = 2123

	return newHandlers(&handlerConfig{
		sessions:        stubSessionManager{},
		tasks:           stubTaskManager{},
		observer:        stubObserver{},
		workspaces:      stubWorkspaceService{},
		settings:        settings,
		settingsRestart: restart,
		extensions:      extensions,
		staticFS:        mustStaticFS(t),
		homePaths:       homePaths,
		config:          cfg,
		boundHost:       cfg.HTTP.Host,
		logger:          discardLogger(),
		startedAt:       time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		now:             func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
		pollInterval:    5 * time.Millisecond,
		agentLoader:     aghconfig.LoadAgentDef,
		httpPort:        cfg.HTTP.Port,
	})
}

func newTestRouter(t *testing.T, handlers *Handlers) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(requestLoggingMiddleware(discardLogger()))
	boundHost := "127.0.0.1"
	if handlers != nil && handlers.boundHost != "" {
		boundHost = handlers.boundHost
	}
	engine.Use(corsMiddleware(boundHost))
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

func testConfigWithDisabledNetwork(homePaths aghconfig.HomePaths) aghconfig.Config {
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.Network.Enabled = false
	return cfg
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

func mustJSONBody(t *testing.T, value any) []byte {
	t.Helper()
	return testutil.MustJSONBody(t, value)
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
