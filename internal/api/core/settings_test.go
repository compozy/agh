package core_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	automationmodel "github.com/pedronauck/agh/internal/automation/model"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/config/lifecycle"
	diagnosticcontract "github.com/pedronauck/agh/internal/diagnosticcontract"
	"github.com/pedronauck/agh/internal/diagnostics"
	extensionpkg "github.com/pedronauck/agh/internal/extension"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/resources"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

type stubSettingsService struct {
	GetSectionFn        func(context.Context, settingspkg.SectionRequest) (settingspkg.SectionEnvelope, error)
	UpdateSectionFn     func(context.Context, settingspkg.SectionUpdateRequest) (settingspkg.MutationResult, error)
	ApplySectionFn      func(context.Context, settingspkg.SectionUpdateRequest) (settingspkg.ApplyResult, error)
	ListCollectionFn    func(context.Context, settingspkg.CollectionRequest) (settingspkg.CollectionEnvelope, error)
	PutCollectionItemFn func(context.Context, settingspkg.CollectionItemPutRequest) (settingspkg.MutationResult, error)
	ApplyCollectionFn   func(context.Context, settingspkg.CollectionItemPutRequest) (settingspkg.ApplyResult, error)
	DeleteItemFn        func(context.Context, settingspkg.CollectionItemDeleteRequest) (settingspkg.MutationResult, error)
	ApplyDeleteFn       func(context.Context, settingspkg.CollectionItemDeleteRequest) (settingspkg.ApplyResult, error)
	ReloadFn            func(context.Context) (settingspkg.ApplyResult, error)
	ListApplyRecordsFn  func(context.Context, settingspkg.ApplyRecordFilter) ([]settingspkg.ApplyRecord, error)

	LastGetSectionRequest     settingspkg.SectionRequest
	LastUpdateSectionRequest  settingspkg.SectionUpdateRequest
	LastListCollectionRequest settingspkg.CollectionRequest
	LastPutCollectionRequest  settingspkg.CollectionItemPutRequest
	LastDeleteRequest         settingspkg.CollectionItemDeleteRequest
	LastApplyRecordFilter     settingspkg.ApplyRecordFilter

	GetSectionCalls        int
	UpdateSectionCalls     int
	ApplySectionCalls      int
	ListCollectionCalls    int
	PutCollectionItemCalls int
	ApplyCollectionCalls   int
	DeleteItemCalls        int
	ApplyDeleteCalls       int
	ReloadCalls            int
	ListApplyRecordsCalls  int
}

func (s *stubSettingsService) GetSection(
	ctx context.Context,
	req settingspkg.SectionRequest,
) (settingspkg.SectionEnvelope, error) {
	s.GetSectionCalls++
	s.LastGetSectionRequest = req
	if s.GetSectionFn != nil {
		return s.GetSectionFn(ctx, req)
	}
	return settingspkg.SectionEnvelope{}, nil
}

func (s *stubSettingsService) UpdateSection(
	ctx context.Context,
	req settingspkg.SectionUpdateRequest,
) (settingspkg.MutationResult, error) {
	s.UpdateSectionCalls++
	s.LastUpdateSectionRequest = req
	if s.UpdateSectionFn != nil {
		return s.UpdateSectionFn(ctx, req)
	}
	return settingspkg.MutationResult{}, nil
}

func (s *stubSettingsService) ApplySection(
	ctx context.Context,
	req settingspkg.SectionUpdateRequest,
) (settingspkg.ApplyResult, error) {
	s.ApplySectionCalls++
	s.LastUpdateSectionRequest = req
	if s.ApplySectionFn != nil {
		return s.ApplySectionFn(ctx, req)
	}
	return defaultApplyResult(req.Section), nil
}

func (s *stubSettingsService) ListCollection(
	ctx context.Context,
	req settingspkg.CollectionRequest,
) (settingspkg.CollectionEnvelope, error) {
	s.ListCollectionCalls++
	s.LastListCollectionRequest = req
	if s.ListCollectionFn != nil {
		return s.ListCollectionFn(ctx, req)
	}
	return settingspkg.CollectionEnvelope{}, nil
}

func (s *stubSettingsService) PutCollectionItem(
	ctx context.Context,
	req settingspkg.CollectionItemPutRequest,
) (settingspkg.MutationResult, error) {
	s.PutCollectionItemCalls++
	s.LastPutCollectionRequest = req
	if s.PutCollectionItemFn != nil {
		return s.PutCollectionItemFn(ctx, req)
	}
	return settingspkg.MutationResult{}, nil
}

func (s *stubSettingsService) ApplyCollectionItem(
	ctx context.Context,
	req settingspkg.CollectionItemPutRequest,
) (settingspkg.ApplyResult, error) {
	s.ApplyCollectionCalls++
	s.LastPutCollectionRequest = req
	if s.ApplyCollectionFn != nil {
		return s.ApplyCollectionFn(ctx, req)
	}
	return defaultApplyResult(settingspkg.SectionName(req.Collection)), nil
}

func (s *stubSettingsService) DeleteCollectionItem(
	ctx context.Context,
	req settingspkg.CollectionItemDeleteRequest,
) (settingspkg.MutationResult, error) {
	s.DeleteItemCalls++
	s.LastDeleteRequest = req
	if s.DeleteItemFn != nil {
		return s.DeleteItemFn(ctx, req)
	}
	return settingspkg.MutationResult{}, nil
}

func (s *stubSettingsService) ApplyCollectionDelete(
	ctx context.Context,
	req settingspkg.CollectionItemDeleteRequest,
) (settingspkg.ApplyResult, error) {
	s.ApplyDeleteCalls++
	s.LastDeleteRequest = req
	if s.ApplyDeleteFn != nil {
		return s.ApplyDeleteFn(ctx, req)
	}
	return defaultApplyResult(settingspkg.SectionName(req.Collection)), nil
}

func (s *stubSettingsService) Reload(ctx context.Context) (settingspkg.ApplyResult, error) {
	s.ReloadCalls++
	if s.ReloadFn != nil {
		return s.ReloadFn(ctx)
	}
	return defaultApplyResult(""), nil
}

func (s *stubSettingsService) ListApplyRecords(
	ctx context.Context,
	filter settingspkg.ApplyRecordFilter,
) ([]settingspkg.ApplyRecord, error) {
	s.ListApplyRecordsCalls++
	s.LastApplyRecordFilter = filter
	if s.ListApplyRecordsFn != nil {
		return s.ListApplyRecordsFn(ctx, filter)
	}
	return nil, nil
}

func defaultApplyResult(section settingspkg.SectionName) settingspkg.ApplyResult {
	return settingspkg.ApplyResult{
		Section:    section,
		Scope:      settingspkg.ScopeGlobal,
		Applied:    true,
		NextAction: "none",
		Record: settingspkg.ApplyRecord{
			ID:         "cfgapp-test",
			ActiveHash: "sha256:test",
			Generation: 1,
			DiffClass:  "live",
			Status:     "applied",
			Lifecycle:  "live",
			NextAction: "none",
			CreatedAt:  time.Unix(1, 0).UTC(),
			UpdatedAt:  time.Unix(1, 0).UTC(),
		},
	}
}

type unavailableSettingsMemoryProviderService struct {
	err error
}

func (s unavailableSettingsMemoryProviderService) List(
	context.Context,
	string,
) ([]contract.MemoryProviderPayload, error) {
	return nil, s.err
}

func (s unavailableSettingsMemoryProviderService) Get(
	context.Context,
	string,
	string,
) (contract.MemoryProviderPayload, error) {
	return contract.MemoryProviderPayload{}, s.err
}

func (s unavailableSettingsMemoryProviderService) Select(
	context.Context,
	string,
	string,
) (contract.MemoryProviderPayload, error) {
	return contract.MemoryProviderPayload{}, s.err
}

func (s unavailableSettingsMemoryProviderService) Enable(
	context.Context,
	string,
	string,
	string,
) (contract.MemoryProviderLifecycleResponse, error) {
	return contract.MemoryProviderLifecycleResponse{}, s.err
}

func (s unavailableSettingsMemoryProviderService) Disable(
	context.Context,
	string,
	string,
	string,
) (contract.MemoryProviderLifecycleResponse, error) {
	return contract.MemoryProviderLifecycleResponse{}, s.err
}

type stubSettingsRestartController struct {
	RequestFn func(context.Context) (core.SettingsRestartOperation, error)
	StatusFn  func(context.Context, string) (core.SettingsRestartOperation, error)

	LastOperationID string
	RequestCalls    int
	StatusCalls     int
}

func (s *stubSettingsRestartController) RequestRestart(
	ctx context.Context,
) (core.SettingsRestartOperation, error) {
	s.RequestCalls++
	if s.RequestFn != nil {
		return s.RequestFn(ctx)
	}
	return core.SettingsRestartOperation{}, nil
}

type stubSettingsUpdateController struct {
	GetFn    func(context.Context) (core.SettingsUpdateStatus, error)
	GetCalls int
}

func (s *stubSettingsUpdateController) GetUpdate(ctx context.Context) (core.SettingsUpdateStatus, error) {
	s.GetCalls++
	if s.GetFn != nil {
		return s.GetFn(ctx)
	}
	return core.SettingsUpdateStatus{}, nil
}

func (s *stubSettingsRestartController) GetRestartOperation(
	ctx context.Context,
	operationID string,
) (core.SettingsRestartOperation, error) {
	s.StatusCalls++
	s.LastOperationID = operationID
	if s.StatusFn != nil {
		return s.StatusFn(ctx, operationID)
	}
	return core.SettingsRestartOperation{}, nil
}

type settingsHandlerFixture struct {
	Handlers   *core.BaseHandlers
	Engine     *gin.Engine
	HomePaths  aghconfig.HomePaths
	StreamDone chan struct{}
	Service    *stubSettingsService
	Restart    *stubSettingsRestartController
	Update     *stubSettingsUpdateController
}

func newSettingsHandlerFixture(
	t *testing.T,
	transport string,
	settingsService *stubSettingsService,
	restartController *stubSettingsRestartController,
) settingsHandlerFixture {
	t.Helper()

	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123
	cfg.Daemon.Socket = "/tmp/settings-api-core.sock"
	streamDone := make(chan struct{})

	if settingsService == nil {
		settingsService = &stubSettingsService{}
	}
	if restartController == nil {
		restartController = &stubSettingsRestartController{}
	}
	updateController := &stubSettingsUpdateController{}

	handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
		TransportName:      transport,
		MaskInternalErrors: false,
		Settings:           settingsService,
		SettingsRestart:    restartController,
		SettingsUpdate:     updateController,
		HomePaths:          homePaths,
		Config:             cfg,
		Logger:             testutil.DiscardLogger(),
		Now: func() time.Time {
			return time.Date(2026, 4, 17, 18, 45, 0, 0, time.UTC)
		},
		PollInterval: 5 * time.Millisecond,
		StreamDone:   streamDone,
	})

	engine := gin.New()
	engine.Use(gin.Recovery())
	registerSettingsRoutes(engine, handlers)

	return settingsHandlerFixture{
		Handlers:   handlers,
		Engine:     engine,
		HomePaths:  homePaths,
		StreamDone: streamDone,
		Service:    settingsService,
		Restart:    restartController,
		Update:     updateController,
	}
}

func registerSettingsRoutes(engine *gin.Engine, handlers *core.BaseHandlers) {
	settings := engine.Group("/api/settings")
	settings.GET("/apply", handlers.ListSettingsApplyRecords)
	settings.POST("/reload", handlers.ReloadSettings)
	settings.GET("/general", handlers.GetSettingsGeneral)
	settings.GET("/update", handlers.GetSettingsUpdate)
	settings.PATCH("/general", handlers.UpdateSettingsGeneral)
	settings.GET("/memory", handlers.GetSettingsMemory)
	settings.PATCH("/memory", handlers.UpdateSettingsMemory)
	settings.GET("/skills", handlers.GetSettingsSkills)
	settings.PATCH("/skills", handlers.UpdateSettingsSkills)
	settings.GET("/automation", handlers.GetSettingsAutomation)
	settings.PATCH("/automation", handlers.UpdateSettingsAutomation)
	settings.GET("/network", handlers.GetSettingsNetwork)
	settings.PATCH("/network", handlers.UpdateSettingsNetwork)
	settings.GET("/observability", handlers.GetSettingsObservability)
	settings.PATCH("/observability", handlers.UpdateSettingsObservability)
	settings.GET("/hooks-extensions", handlers.GetSettingsHooksExtensions)
	settings.PATCH("/hooks-extensions", handlers.UpdateSettingsHooksExtensions)
	settings.GET("/providers", handlers.ListSettingsProviders)
	settings.GET("/providers/:name", handlers.GetSettingsProvider)
	settings.PUT("/providers/:name", handlers.PutSettingsProvider)
	settings.DELETE("/providers/:name", handlers.DeleteSettingsProvider)
	settings.GET("/mcp-servers", handlers.ListSettingsMCPServers)
	settings.PUT("/mcp-servers/:name", handlers.PutSettingsMCPServer)
	settings.DELETE("/mcp-servers/:name", handlers.DeleteSettingsMCPServer)
	settings.GET("/sandboxes", handlers.ListSettingsSandboxes)
	settings.GET("/sandboxes/:name", handlers.GetSettingsSandbox)
	settings.PUT("/sandboxes/:name", handlers.PutSettingsSandbox)
	settings.DELETE("/sandboxes/:name", handlers.DeleteSettingsSandbox)
	settings.GET("/hooks", handlers.ListSettingsHooks)
	settings.PUT("/hooks/:name", handlers.PutSettingsHook)
	settings.DELETE("/hooks/:name", handlers.DeleteSettingsHook)
	settings.POST("/actions/restart", handlers.TriggerSettingsRestart)
	settings.GET("/actions/restart/:operation_id", handlers.GetSettingsRestartStatus)
	settings.GET("/observability/log-tail", handlers.StreamSettingsObservabilityLogTail)
}

func TestStatusForSettingsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "nil", err: nil, want: http.StatusOK},
		{
			name: "validation sentinel",
			err:  core.NewSettingsValidationError(errors.New("bad payload")),
			want: http.StatusBadRequest,
		},
		{
			name: "not found sentinel",
			err:  core.NewSettingsNotFoundError(errors.New("missing")),
			want: http.StatusNotFound,
		},
		{
			name: "conflict sentinel",
			err:  core.NewSettingsConflictError(errors.New("conflict")),
			want: http.StatusConflict,
		},
		{
			name: "forbidden sentinel",
			err:  core.NewSettingsForbiddenError(errors.New("forbidden")),
			want: http.StatusForbidden,
		},
		{name: "workspace missing", err: workspacepkg.ErrWorkspaceNotFound, want: http.StatusNotFound},
		{
			name: "settings conflict sentinel",
			err: fmt.Errorf(
				"%w: %s",
				settingspkg.ErrConflict,
				`settings: section "general" does not support workspace scope`,
			),
			want: http.StatusConflict,
		},
		{
			name: "settings validation sentinel",
			err: fmt.Errorf(
				"%w: %s",
				settingspkg.ErrValidation,
				"settings: decode network settings request: bad json",
			),
			want: http.StatusBadRequest,
		},
		{
			name: "settings forbidden sentinel",
			err:  fmt.Errorf("%w: %s", settingspkg.ErrForbidden, "settings mutations are forbidden for this transport"),
			want: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := core.StatusForSettingsError(tc.err); got != tc.want {
				t.Fatalf("StatusForSettingsError(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

func TestGetSettingsUpdateReturnsCurrentSnapshot(t *testing.T) {
	t.Parallel()

	fixture := newSettingsHandlerFixture(t, "api-core-http", &stubSettingsService{}, nil)
	fixture.Update.GetFn = func(context.Context) (core.SettingsUpdateStatus, error) {
		checkedAt := time.Date(2026, 5, 3, 19, 0, 0, 0, time.UTC)
		return core.SettingsUpdateStatus{
			Supported:      true,
			Managed:        false,
			InstallMethod:  "direct-binary",
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.1.0",
			Available:      true,
			Status:         "available",
			Recommendation: "Run `agh update`.",
			ReleaseURL:     "https://github.com/compozy/agh/releases/tag/v1.1.0",
			CheckedAt:      &checkedAt,
		}, nil
	}

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/api/settings/update", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET /api/settings/update status = %d, want 200", resp.Code)
	}

	var payload contract.SettingsUpdateResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(update response) error = %v", err)
	}
	if payload.Status != contract.SettingsUpdateStatusAvailable || payload.InstallMethod != "direct-binary" {
		t.Fatalf("update payload = %#v, want available direct-binary", payload)
	}
	if fixture.Update.GetCalls != 1 {
		t.Fatalf("GetCalls = %d, want 1", fixture.Update.GetCalls)
	}
}

func TestSettingsSectionAndCollectionConversions(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 4, 17, 18, 0, 0, 0, time.UTC)
	lastConsolidatedAt := time.Date(2026, 4, 17, 17, 0, 0, 0, time.UTC)
	nextFire := time.Date(2026, 4, 17, 19, 0, 0, 0, time.UTC)
	lastSyncedAt := time.Date(2026, 4, 17, 18, 30, 0, 0, time.UTC)
	readOnly := true

	sectionEnvelopes := []settingspkg.SectionEnvelope{
		{
			Section:         settingspkg.SectionGeneral,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			General: &settingspkg.GeneralSection{
				Runtime: settingspkg.DaemonRuntimeStatus{
					Available:      true,
					Status:         "running",
					PID:            4100,
					StartedAt:      startedAt,
					UptimeSeconds:  900,
					Socket:         "/tmp/agh.sock",
					HTTPHost:       "127.0.0.1",
					HTTPPort:       2123,
					ActiveSessions: 2,
					ActiveAgents:   1,
					TotalSessions:  3,
					Version:        "test-version",
				},
				ConfigPaths: settingspkg.ConfigPaths{
					HomeDir:          "/tmp/home",
					GlobalConfig:     "/tmp/home/config.toml",
					GlobalMCPSidecar: "/tmp/home/mcp.json",
					LogFile:          "/tmp/home/agh.log",
					DaemonInfo:       "/tmp/home/daemon.json",
				},
				Settings: settingspkg.GeneralSettings{
					Defaults: aghconfig.DefaultsConfig{Agent: "coder", Provider: "openai", Sandbox: "local"},
					Limits:   aghconfig.LimitsConfig{MaxConcurrentAgents: 2},
					Permissions: aghconfig.PermissionsConfig{
						Mode: aghconfig.PermissionModeApproveReads,
					},
					SessionTimeout: 30 * time.Minute,
					HTTP:           aghconfig.HTTPConfig{Host: "127.0.0.1", Port: 2123},
					Daemon:         aghconfig.DaemonConfig{Socket: "/tmp/agh.sock"},
				},
				Actions: settingspkg.GeneralActions{
					Restart: settingspkg.ActionMetadata{
						Name:      "restart",
						Available: true,
						Behavior:  settingspkg.MutationBehaviorActionTrigger,
					},
				},
			},
		},
		{
			Section:         settingspkg.SectionMemory,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			Memory: &settingspkg.MemorySection{
				Config: aghconfig.MemoryConfig{
					Enabled:   true,
					GlobalDir: "/tmp/home/memory",
					Dream: aghconfig.DreamConfig{
						Enabled:       true,
						Agent:         "dreamer",
						MinHours:      1.5,
						MinSessions:   2,
						CheckInterval: time.Hour,
					},
				},
				Health: settingspkg.MemoryHealthStatus{
					Available:          true,
					FileCount:          4,
					DreamEnabled:       true,
					LastConsolidatedAt: &lastConsolidatedAt,
				},
				Actions: settingspkg.MemoryActions{
					Consolidate: settingspkg.ActionMetadata{
						Name:      "consolidate",
						Available: true,
						Behavior:  settingspkg.MutationBehaviorActionTrigger,
					},
				},
			},
		},
		{
			Section:         settingspkg.SectionSkills,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			Skills: &settingspkg.SkillsSection{
				Config: aghconfig.SkillsConfig{
					Enabled:      true,
					PollInterval: time.Minute,
					Marketplace: aghconfig.MarketplaceConfig{
						Registry: "registry.example",
						BaseURL:  "https://registry.example",
					},
				},
				DiscoveredCount:  5,
				DisabledCount:    1,
				RuntimeAvailable: true,
				Links:            []settingspkg.OperationalLink{{Label: "skills", Path: "/skills"}},
			},
		},
		{
			Section:         settingspkg.SectionAutomation,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			Automation: &settingspkg.AutomationSection{
				Config: settingspkg.AutomationSettings{
					Enabled:           true,
					Timezone:          "UTC",
					MaxConcurrentJobs: 2,
					DefaultFireLimit:  automationmodel.FireLimitConfig{Max: 5, Window: "1m"},
				},
				Runtime: settingspkg.AutomationRuntimeStatus{
					Available:        true,
					Running:          true,
					SchedulerRunning: true,
					JobTotal:         3,
					JobEnabled:       2,
					TriggerTotal:     4,
					TriggerEnabled:   3,
					NextFire:         &nextFire,
					LastSyncedAt:     &lastSyncedAt,
				},
				Links: []settingspkg.OperationalLink{{Label: "automation", Path: "/automation"}},
			},
		},
		{
			Section:         settingspkg.SectionNetwork,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			Network: &settingspkg.NetworkSection{
				Config: aghconfig.NetworkConfig{
					Enabled:        true,
					DefaultChannel: "builders",
					Port:           4222,
					MaxPayload:     1024,
					GreetInterval:  5,
					MaxReplayAge:   10,
					MaxQueueDepth:  32,
				},
				Runtime: settingspkg.NetworkRuntimeStatus{
					Available:       true,
					Enabled:         true,
					Status:          "running",
					ListenerHost:    "127.0.0.1",
					ListenerPort:    4222,
					LocalPeers:      1,
					RemotePeers:     2,
					Channels:        3,
					QueuedMessages:  4,
					QueuedSessions:  5,
					DeliveryWorkers: 2,
				},
				Links: []settingspkg.OperationalLink{{Label: "network", Path: "/network"}},
			},
		},
		{
			Section:         settingspkg.SectionObservability,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			Observability: &settingspkg.ObservabilitySection{
				Config: aghconfig.ObservabilityConfig{
					Enabled:        true,
					RetentionDays:  7,
					MaxGlobalBytes: 2048,
					Transcripts: aghconfig.ObservabilityTranscriptConfig{
						Enabled:            true,
						SegmentBytes:       4096,
						MaxBytesPerSession: 8192,
					},
				},
				Runtime: settingspkg.ObservabilityRuntimeStatus{
					Available:          true,
					Status:             "ok",
					GlobalDBSizeBytes:  10,
					SessionDBSizeBytes: 20,
					ActiveSessions:     2,
					ActiveAgents:       1,
					UptimeSeconds:      600,
				},
				LogTailSupport: settingspkg.CapabilityStatus{Available: true},
			},
		},
		{
			Section:         settingspkg.SectionHooksExtensions,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			HooksExtensions: &settingspkg.HooksExtensionsSection{
				Hooks: []settingspkg.HookItem{{
					Name: "capture-tool-call",
					Declaration: hookspkg.HookDecl{
						Name:         "capture-tool-call",
						Event:        hookspkg.HookToolPreCall,
						Mode:         hookspkg.HookModeAsync,
						ExecutorKind: hookspkg.HookExecutorSubprocess,
						Command:      "/bin/capture",
						Matcher: hookspkg.HookMatcher{
							ToolID: "agh__read",
						},
					},
					SourceMetadata: settingspkg.SourceMetadata{
						EffectiveSource: settingspkg.SourceRef{
							Kind:  settingspkg.SourceKindGlobalConfig,
							Scope: settingspkg.ScopeGlobal,
						},
						AvailableTargets: []settingspkg.WriteTargetKind{settingspkg.WriteTargetGlobalConfig},
					},
				}},
				Extensions: aghconfig.ExtensionsConfig{
					Marketplace: aghconfig.ExtensionsMarketplaceConfig{
						Registry: "extensions.example",
						BaseURL:  "https://extensions.example",
					},
					Resources: aghconfig.ExtensionsResourcesConfig{
						AllowedKinds: []resources.ResourceKind{resources.ResourceKind("tool")},
						MaxScope:     resources.ResourceScopeKindWorkspace,
						SnapshotRateLimit: aghconfig.ExtensionsResourceRateLimitConfig{
							Requests: 10,
							Window:   time.Minute,
							Queue:    2,
						},
						OperatorWriteRateLimit: aghconfig.ExtensionsResourceRateLimitConfig{
							Requests: 20,
							Window:   time.Minute,
							Queue:    4,
						},
					},
				},
				Installed: []settingspkg.InstalledExtension{{
					Name:          "ext-a",
					Version:       "1.0.0",
					Enabled:       true,
					State:         "active",
					Health:        "ok",
					HealthMessage: "healthy",
				}},
				TransportParity: settingspkg.TransportParityStatus{
					Known:          true,
					SettingsHTTP:   true,
					SettingsUDS:    true,
					ExtensionsHTTP: true,
					ExtensionsUDS:  true,
				},
			},
		},
	}

	for _, envelope := range sectionEnvelopes {
		t.Run("Should section/"+string(envelope.Section), func(t *testing.T) {
			t.Parallel()
			if _, err := core.SettingsSectionResponseFromEnvelope(envelope); err != nil {
				t.Fatalf("SettingsSectionResponseFromEnvelope(%s) error = %v", envelope.Section, err)
			}
		})
	}

	collectionEnvelopes := []settingspkg.CollectionEnvelope{
		{
			Collection:      settingspkg.CollectionProviders,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			Providers: []settingspkg.ProviderItem{
				{
					Name: "openai",
					Settings: settingspkg.ProviderSettings{
						Command: "codex",
						Models: aghconfig.ProviderModelsConfig{
							Default: "gpt-5.4",
						},
						CredentialSlots: []aghconfig.ProviderCredentialSlot{
							{
								Name:      "api_key",
								TargetEnv: "OPENAI_API_KEY",
								SecretRef: "env:OPENAI_API_KEY",
								Kind:      "api_key",
								Required:  true,
							},
						},
					},
					Default:          true,
					CommandAvailable: true,
					Credentials: []settingspkg.ProviderCredentialStatus{
						{
							Name:      "api_key",
							TargetEnv: "OPENAI_API_KEY",
							SecretRef: "env:OPENAI_API_KEY",
							Kind:      "api_key",
							Required:  true,
							Present:   true,
							Source:    "env",
						},
					},
					SourceMetadata: settingspkg.SourceMetadata{
						EffectiveSource: settingspkg.SourceRef{
							Kind:  settingspkg.SourceKindGlobalConfig,
							Scope: settingspkg.ScopeGlobal,
						},
						AvailableTargets: []settingspkg.WriteTargetKind{
							settingspkg.WriteTargetGlobalConfig,
						},
					},
					Fallback: &settingspkg.ProviderFallback{
						Source: settingspkg.SourceRef{
							Kind:  settingspkg.SourceKindBuiltinProvider,
							Scope: settingspkg.ScopeGlobal,
						},
						Settings: settingspkg.ProviderSettings{
							Command: "codex",
							Models: aghconfig.ProviderModelsConfig{
								Default: "gpt-5.4",
							},
						},
					},
				},
			},
		},
		{
			Collection:      settingspkg.CollectionMCPServers,
			Scope:           settingspkg.ScopeWorkspace,
			WorkspaceID:     "ws-1",
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal, settingspkg.ScopeWorkspace},
			MCPServers: []settingspkg.MCPServerItem{{
				Name:        "memory",
				Command:     "memoryd",
				Args:        []string{"serve"},
				Env:         map[string]string{"TOKEN": "abc"},
				Scope:       settingspkg.ScopeWorkspace,
				WorkspaceID: "ws-1",
				SourceMetadata: settingspkg.SourceMetadata{
					EffectiveSource: settingspkg.SourceRef{
						Kind:        settingspkg.SourceKindWorkspaceMCPSidecar,
						Scope:       settingspkg.ScopeWorkspace,
						WorkspaceID: "ws-1",
					},
					AvailableTargets: []settingspkg.WriteTargetKind{settingspkg.WriteTargetWorkspaceMCPSidecar},
				},
			}},
		},
		{
			Collection:      settingspkg.CollectionSandboxes,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			Sandboxes: []settingspkg.SandboxItem{{
				Name: "local",
				Profile: aghconfig.SandboxProfile{
					Backend:     "local",
					SyncMode:    "session-bidirectional",
					Persistence: "reuse",
					RuntimeRoot: "/workspace",
				},
				WorkspaceUsageCount: 2,
				SourceMetadata: settingspkg.SourceMetadata{
					EffectiveSource: settingspkg.SourceRef{
						Kind:  settingspkg.SourceKindGlobalConfig,
						Scope: settingspkg.ScopeGlobal,
					},
					AvailableTargets: []settingspkg.WriteTargetKind{
						settingspkg.WriteTargetGlobalConfig,
					},
				},
			}},
		},
		{
			Collection:      settingspkg.CollectionHooks,
			Scope:           settingspkg.ScopeGlobal,
			AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			Hooks: []settingspkg.HookItem{{
				Name: "capture",
				Declaration: hookspkg.HookDecl{
					Name:         "capture",
					Event:        hookspkg.HookToolPreCall,
					Mode:         hookspkg.HookModeAsync,
					ExecutorKind: hookspkg.HookExecutorSubprocess,
					Command:      "/bin/capture",
					Matcher: hookspkg.HookMatcher{
						ToolID:           "agh__read",
						ToolReadOnly:     &readOnly,
						MessageRole:      "assistant",
						MessageDeltaType: "text",
					},
				},
				SourceMetadata: settingspkg.SourceMetadata{
					EffectiveSource: settingspkg.SourceRef{
						Kind:  settingspkg.SourceKindGlobalConfig,
						Scope: settingspkg.ScopeGlobal,
					},
					AvailableTargets: []settingspkg.WriteTargetKind{
						settingspkg.WriteTargetGlobalConfig,
					},
				},
			}},
		},
	}

	for _, envelope := range collectionEnvelopes {
		t.Run("Should collection/"+string(envelope.Collection), func(t *testing.T) {
			t.Parallel()
			if _, err := core.SettingsCollectionResponseFromEnvelope(envelope); err != nil {
				t.Fatalf("SettingsCollectionResponseFromEnvelope(%s) error = %v", envelope.Collection, err)
			}
		})
	}
}

func TestUpdateSettingsGeneralRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	fixture := newSettingsHandlerFixture(t, "api-core-http", &stubSettingsService{}, nil)
	body := mustJSON(t, map[string]any{
		"config": map[string]any{
			"defaults": map[string]any{
				"agent": "coder",
			},
			"limits": map[string]any{
				"max_concurrent_agents": 2,
			},
			"permissions": map[string]any{
				"mode": "approve-reads",
			},
			"session_timeout": "not-a-duration",
			"http": map[string]any{
				"host": "127.0.0.1",
				"port": 2123,
			},
			"daemon": map[string]any{
				"socket": "/tmp/agh.sock",
			},
		},
	})

	resp := performRequest(t, fixture.Engine, http.MethodPatch, "/api/settings/general", body)
	if got, want := resp.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
	}
	if fixture.Service.UpdateSectionCalls != 0 {
		t.Fatalf("UpdateSectionCalls = %d, want 0", fixture.Service.UpdateSectionCalls)
	}

	var payload contract.ErrorPayload
	decodeJSON(t, resp.Body.Bytes(), &payload)
	if !strings.Contains(payload.Error, "general.config.session_timeout") {
		t.Fatalf("payload.Error = %q, want section-specific context", payload.Error)
	}
}

func TestUpdateSettingsSectionHandlersRejectMissingConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "general", path: "/api/settings/general", want: "general.config is required"},
		{name: "memory", path: "/api/settings/memory", want: "memory.config is required"},
		{name: "skills", path: "/api/settings/skills", want: "skills.config is required"},
		{name: "automation", path: "/api/settings/automation", want: "automation.config is required"},
		{name: "network", path: "/api/settings/network", want: "network.config is required"},
		{name: "observability", path: "/api/settings/observability", want: "observability.config is required"},
		{name: "hooks extensions", path: "/api/settings/hooks-extensions", want: "hooks-extensions.config is required"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &stubSettingsService{}
			fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

			resp := performRequest(t, fixture.Engine, http.MethodPatch, tc.path, []byte(`{}`))
			if got, want := resp.Code, http.StatusBadRequest; got != want {
				t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
			}
			if service.UpdateSectionCalls != 0 {
				t.Fatalf("UpdateSectionCalls = %d, want 0", service.UpdateSectionCalls)
			}

			var payload contract.ErrorPayload
			decodeJSON(t, resp.Body.Bytes(), &payload)
			if !strings.Contains(payload.Error, tc.want) {
				t.Fatalf("payload.Error = %q, want substring %q", payload.Error, tc.want)
			}
		})
	}
}

func TestUpdateSettingsMemoryRejectsUnavailableProvider(t *testing.T) {
	t.Parallel()

	t.Run("Should reject unknown provider names as validation errors", func(t *testing.T) {
		t.Parallel()

		service := &stubSettingsService{}
		fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)
		fixture.Handlers.MemoryProviders = unavailableSettingsMemoryProviderService{
			err: extensionpkg.ErrMemoryProviderNotFound,
		}

		memoryPayload := validSettingsMemoryConfigPayload()
		memoryPayload.Provider.Name = "qa-missing-provider"
		body := contract.UpdateSettingsMemoryRequest{Config: memoryPayload}

		resp := performRequest(t, fixture.Engine, http.MethodPatch, "/api/settings/memory", mustJSON(t, body))
		if got, want := resp.Code, http.StatusBadRequest; got != want {
			t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
		}
		if service.UpdateSectionCalls != 0 {
			t.Fatalf("UpdateSectionCalls = %d, want 0", service.UpdateSectionCalls)
		}

		var payload contract.ErrorPayload
		decodeJSON(t, resp.Body.Bytes(), &payload)
		if !strings.Contains(payload.Error, "qa-missing-provider") {
			t.Fatalf("payload.Error = %q, want provider name", payload.Error)
		}
	})

	t.Run("Should preserve provider lookup infrastructure failures as server errors", func(t *testing.T) {
		t.Parallel()

		service := &stubSettingsService{}
		fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)
		fixture.Handlers.MemoryProviders = unavailableSettingsMemoryProviderService{
			err: errors.New("catalog backend offline"),
		}

		memoryPayload := validSettingsMemoryConfigPayload()
		memoryPayload.Provider.Name = "qa-provider"
		body := contract.UpdateSettingsMemoryRequest{Config: memoryPayload}

		resp := performRequest(t, fixture.Engine, http.MethodPatch, "/api/settings/memory", mustJSON(t, body))
		if got, want := resp.Code, http.StatusInternalServerError; got != want {
			t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
		}
		if service.UpdateSectionCalls != 0 {
			t.Fatalf("UpdateSectionCalls = %d, want 0", service.UpdateSectionCalls)
		}

		var payload contract.ErrorPayload
		decodeJSON(t, resp.Body.Bytes(), &payload)
		if !strings.Contains(payload.Error, "catalog backend offline") {
			t.Fatalf("payload.Error = %q, want backend failure", payload.Error)
		}
	})
}

func TestUpdateSettingsSectionHandlersDelegateValidPayloads(t *testing.T) {
	t.Parallel()

	memoryPayload := validSettingsMemoryConfigPayload()
	memoryPayload.GlobalDir = "/tmp/memory"
	memoryPayload.Dream.Agent = "dreamer"
	memoryPayload.Dream.MinHours = 1.5
	memoryPayload.Dream.MinSessions = 2
	memoryPayload.Dream.CheckInterval = "1h"

	tests := []struct {
		name   string
		path   string
		body   any
		assert func(t *testing.T, req settingspkg.SectionUpdateRequest)
	}{
		{
			name: "general",
			path: "/api/settings/general",
			body: contract.UpdateSettingsGeneralRequest{
				Config: contract.SettingsGeneralConfigPayload{
					Defaults: contract.SettingsDefaultsPayload{
						Agent:    "coder",
						Provider: "openai",
						Sandbox:  "local",
					},
					Limits: contract.SettingsLimitsPayload{MaxConcurrentAgents: 2},
					Permissions: contract.SettingsPermissionsPayload{
						Mode: contract.SettingsPermissionModeApproveReads,
					},
					SessionTimeout: "30m",
					HTTP:           contract.SettingsHTTPPayload{Host: "127.0.0.1", Port: 2123},
					Daemon:         contract.SettingsDaemonPayload{Socket: "/tmp/agh.sock"},
				},
			},
			assert: func(t *testing.T, req settingspkg.SectionUpdateRequest) {
				t.Helper()
				if req.General == nil || req.General.Defaults.Agent != "coder" {
					t.Fatalf("req.General = %#v, want populated general settings", req.General)
				}
			},
		},
		{
			name: "memory",
			path: "/api/settings/memory",
			body: contract.UpdateSettingsMemoryRequest{
				Config: memoryPayload,
			},
			assert: func(t *testing.T, req settingspkg.SectionUpdateRequest) {
				t.Helper()
				if req.Memory == nil || req.Memory.Dream.Agent != "dreamer" {
					t.Fatalf("req.Memory = %#v, want populated memory config", req.Memory)
				}
			},
		},
		{
			name: "skills",
			path: "/api/settings/skills",
			body: contract.UpdateSettingsSkillsRequest{
				Config: contract.SettingsSkillsConfigPayload{
					Enabled:      true,
					PollInterval: "1m",
					Marketplace: contract.SettingsMarketplacePayload{
						Registry: "clawhub",
						BaseURL:  "https://registry.example",
					},
				},
			},
			assert: func(t *testing.T, req settingspkg.SectionUpdateRequest) {
				t.Helper()
				if req.Skills == nil || req.Skills.Marketplace.Registry != "clawhub" {
					t.Fatalf("req.Skills = %#v, want populated skills config", req.Skills)
				}
			},
		},
		{
			name: "automation",
			path: "/api/settings/automation",
			body: contract.UpdateSettingsAutomationRequest{
				Config: contract.SettingsAutomationConfigPayload{
					Enabled:           true,
					Timezone:          "UTC",
					MaxConcurrentJobs: 2,
					DefaultFireLimit:  automationmodel.FireLimitConfig{Max: 5, Window: "1m"},
				},
			},
			assert: func(t *testing.T, req settingspkg.SectionUpdateRequest) {
				t.Helper()
				if req.Automation == nil || req.Automation.DefaultFireLimit.Max != 5 {
					t.Fatalf("req.Automation = %#v, want populated automation config", req.Automation)
				}
			},
		},
		{
			name: "network",
			path: "/api/settings/network",
			body: contract.UpdateSettingsNetworkRequest{
				Config: contract.SettingsNetworkConfigPayload{
					Enabled:        true,
					DefaultChannel: "builders",
					Port:           4222,
					MaxPayload:     1024,
					GreetInterval:  5,
					MaxReplayAge:   10,
					MaxQueueDepth:  32,
				},
			},
			assert: func(t *testing.T, req settingspkg.SectionUpdateRequest) {
				t.Helper()
				if req.Network == nil || req.Network.DefaultChannel != "builders" {
					t.Fatalf("req.Network = %#v, want populated network config", req.Network)
				}
			},
		},
		{
			name: "observability",
			path: "/api/settings/observability",
			body: contract.UpdateSettingsObservabilityRequest{
				Config: contract.SettingsObservabilityConfigPayload{
					Enabled:        true,
					RetentionDays:  7,
					MaxGlobalBytes: 4096,
					Transcripts: contract.SettingsObservabilityTranscriptPayload{
						Enabled:            true,
						SegmentBytes:       2048,
						MaxBytesPerSession: 8192,
					},
				},
			},
			assert: func(t *testing.T, req settingspkg.SectionUpdateRequest) {
				t.Helper()
				if req.Observability == nil || req.Observability.RetentionDays != 7 {
					t.Fatalf("req.Observability = %#v, want populated observability config", req.Observability)
				}
			},
		},
		{
			name: "hooks extensions",
			path: "/api/settings/hooks-extensions",
			body: contract.UpdateSettingsHooksExtensionsRequest{
				Config: contract.SettingsExtensionsConfigPayload{
					Marketplace: contract.SettingsMarketplacePayload{
						Registry: "github",
						BaseURL:  "https://extensions.example",
					},
					Resources: contract.SettingsExtensionResourcesPayload{
						AllowedKinds: []string{"tool"},
						MaxScope:     resources.ResourceScopeKindWorkspace,
						SnapshotRateLimit: contract.SettingsExtensionRateLimitPayload{
							Requests: 10,
							Window:   "1m",
							Queue:    2,
						},
						OperatorWriteRateLimit: contract.SettingsExtensionRateLimitPayload{
							Requests: 20,
							Window:   "1m",
							Queue:    4,
						},
					},
				},
			},
			assert: func(t *testing.T, req settingspkg.SectionUpdateRequest) {
				t.Helper()
				if req.HooksExtensions == nil ||
					req.HooksExtensions.Resources.MaxScope != resources.ResourceScopeKindWorkspace {
					t.Fatalf("req.HooksExtensions = %#v, want populated extensions config", req.HooksExtensions)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &stubSettingsService{
				UpdateSectionFn: func(_ context.Context, req settingspkg.SectionUpdateRequest) (settingspkg.MutationResult, error) {
					return settingspkg.MutationResult{
						Section:  req.Section,
						Scope:    req.Scope,
						Behavior: settingspkg.MutationBehaviorAppliedNow,
						Applied:  true,
					}, nil
				},
			}
			fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

			resp := performRequest(t, fixture.Engine, http.MethodPatch, tc.path, mustJSON(t, tc.body))
			if got, want := resp.Code, http.StatusOK; got != want {
				t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
			}
			tc.assert(t, service.LastUpdateSectionRequest)
		})
	}
}

func validSettingsMemoryConfigPayload() contract.SettingsMemoryConfigPayload {
	return contract.SettingsMemoryConfigPayload{
		Enabled:   true,
		GlobalDir: "/tmp/agh-memory",
		Controller: contract.SettingsMemoryControllerPayload{
			Mode:            "hybrid",
			MaxLatency:      "300ms",
			DefaultOpOnFail: "noop",
			LLM: contract.SettingsMemoryControllerLLMPayload{
				Enabled:       true,
				Model:         "anthropic/claude-haiku-4",
				TopK:          5,
				PromptVersion: "v1",
				Timeout:       "250ms",
				MaxTokensOut:  256,
			},
			Policy: contract.SettingsMemoryControllerPolicyPayload{
				MaxContentChars: 4096,
				MaxWritesPerMin: 60,
				AllowOrigins: []string{
					"cli",
					"http",
					"uds",
					"tool",
					"extractor",
					"dreaming",
					"file",
					"provider",
				},
			},
		},
		Recall: contract.SettingsMemoryRecallPayload{
			TopK:          5,
			RawCandidates: 50,
			Fusion:        "weighted",
			Weights: contract.SettingsMemoryRecallWeightsPayload{
				BM25Unicode:  0.55,
				BM25Trigram:  0.20,
				Recency:      0.15,
				RecallSignal: 0.10,
			},
			Freshness: contract.SettingsMemoryRecallFreshnessPayload{
				BannerAfterDays: 1,
			},
			Signals: contract.SettingsMemoryRecallSignalsPayload{
				QueueCapacity:  256,
				WorkerRetryMax: 3,
				MetricsEnabled: true,
			},
		},
		Decisions: contract.SettingsMemoryDecisionsPayload{
			PruneAfterAppliedDays: 90,
			KeepAuditSummary:      true,
			MaxPostContentBytes:   65536,
		},
		Extractor: contract.SettingsMemoryExtractorPayload{
			Enabled:          true,
			Mode:             "post_message",
			ThrottleTurns:    1,
			Deadline:         "60s",
			SandboxInboxOnly: true,
			InboxPath:        "/tmp/agh-memory/_inbox",
			DLQPath:          "/tmp/agh-memory/_system/extractor/failures",
			Queue: contract.SettingsMemoryExtractorQueuePayload{
				Capacity:    1,
				CoalesceMax: 16,
			},
		},
		Dream: contract.SettingsMemoryDreamPayload{
			Enabled:       true,
			Agent:         "dreaming-curator",
			MinHours:      24,
			MinSessions:   3,
			Debounce:      "10m",
			PromptVersion: "v1",
			CheckInterval: "30m",
			Gates: contract.SettingsMemoryDreamGatesPayload{
				MinUnpromoted:  5,
				MinRecallCount: 2,
				MinScore:       0.75,
			},
			Scoring: contract.SettingsMemoryDreamScoringPayload{
				RecencyHalfLifeDays: 14,
				Weights: contract.SettingsMemoryDreamScoringWeightsPayload{
					Frequency: 0.30,
					Relevance: 0.35,
					Recency:   0.20,
					Freshness: 0.15,
				},
			},
		},
		Session: contract.SettingsMemorySessionPayload{
			LedgerFormat:     "jsonl",
			LedgerRoot:       "/tmp/agh-sessions",
			EventsPurgeGrace: "24h",
			ColdArchiveDays:  30,
			MaxArchiveBytes:  10737418240,
			UnboundPartition: "_unbound",
		},
		Daily: contract.SettingsMemoryDailyPayload{
			MaxBytes:        1048576,
			MaxLines:        5000,
			RotateFormat:    "{date}.{seq}.md",
			DreamingWindow:  7,
			ColdArchiveDays: 30,
			MaxArchiveBytes: 1073741824,
			SweepHour:       3,
			ArchivePath:     "_system/archive",
		},
		File: contract.SettingsMemoryFilePayload{
			MaxLines: 200,
			MaxBytes: 25600,
		},
		Provider: contract.SettingsMemoryProviderPayload{
			Timeout:          "2s",
			FailureThreshold: 5,
			Cooldown:         "30s",
		},
		Workspace: contract.SettingsMemoryWorkspacePayload{
			TOMLPath:   "<workspace>/.agh/workspace.toml",
			AutoCreate: true,
		},
	}
}

func TestSettingsCollectionHandlersDelegateValidPayloads(t *testing.T) {
	t.Parallel()

	readOnly := true
	tests := []struct {
		name           string
		method         string
		path           string
		body           any
		assert         func(t *testing.T, service *stubSettingsService)
		assertResponse func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name:   "list providers",
			method: http.MethodGet,
			path:   "/api/settings/providers",
			assert: func(t *testing.T, service *stubSettingsService) {
				t.Helper()
				if service.LastListCollectionRequest.Collection != settingspkg.CollectionProviders {
					t.Fatalf("Collection = %q, want providers", service.LastListCollectionRequest.Collection)
				}
			},
			assertResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				t.Helper()
				var payload contract.SettingsProvidersResponse
				testutil.DecodeJSONResponse(t, resp, &payload)
				if len(payload.Providers) != 1 || payload.Providers[0].Name != "openai" {
					t.Fatalf("providers payload = %#v, want openai provider", payload)
				}
			},
		},
		{
			name:   "list mcp servers",
			method: http.MethodGet,
			path:   "/api/settings/mcp-servers?scope=workspace&workspace_id=ws-1",
			assert: func(t *testing.T, service *stubSettingsService) {
				t.Helper()
				if service.LastListCollectionRequest.Collection != settingspkg.CollectionMCPServers ||
					service.LastListCollectionRequest.Scope != settingspkg.ScopeWorkspace ||
					service.LastListCollectionRequest.WorkspaceID != "ws-1" {
					t.Fatalf("LastListCollectionRequest = %#v", service.LastListCollectionRequest)
				}
			},
			assertResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				t.Helper()
				var payload contract.SettingsMCPServersResponse
				testutil.DecodeJSONResponse(t, resp, &payload)
				if len(payload.MCPServers) != 1 || payload.MCPServers[0].Name != "memory" {
					t.Fatalf("mcp servers payload = %#v, want memory server", payload)
				}
			},
		},
		{
			name:   "get sandbox",
			method: http.MethodGet,
			path:   "/api/settings/sandboxes/local",
			assert: func(t *testing.T, service *stubSettingsService) {
				t.Helper()
				if service.LastListCollectionRequest.Collection != settingspkg.CollectionSandboxes {
					t.Fatalf("Collection = %q, want sandboxes", service.LastListCollectionRequest.Collection)
				}
			},
			assertResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				t.Helper()
				var payload contract.SettingsSandboxResponse
				testutil.DecodeJSONResponse(t, resp, &payload)
				if payload.Sandbox.Name != "local" || payload.Sandbox.Profile.Backend != "local" {
					t.Fatalf("sandbox payload = %#v, want local sandbox profile", payload)
				}
			},
		},
		{
			name:   "list hooks",
			method: http.MethodGet,
			path:   "/api/settings/hooks",
			assert: func(t *testing.T, service *stubSettingsService) {
				t.Helper()
				if service.LastListCollectionRequest.Collection != settingspkg.CollectionHooks {
					t.Fatalf("Collection = %q, want hooks", service.LastListCollectionRequest.Collection)
				}
			},
			assertResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				t.Helper()
				var payload contract.SettingsHooksResponse
				testutil.DecodeJSONResponse(t, resp, &payload)
				if len(payload.Hooks) != 1 || payload.Hooks[0].Name != "capture" {
					t.Fatalf("hooks payload = %#v, want capture hook", payload)
				}
			},
		},
		{
			name:   "put provider",
			method: http.MethodPut,
			path:   "/api/settings/providers/openai",
			body: contract.PutSettingsProviderRequest{
				Settings: contract.SettingsProviderSettingsPayload{
					Command: "codex",
					Models: &contract.SettingsProviderModelsPayload{
						Default: "gpt-5.4",
						Curated: []contract.SettingsProviderModelPayload{
							{
								ID:                     "gpt-5.4",
								DisplayName:            "GPT-5.4",
								SupportsReasoning:      new(true),
								ReasoningEfforts:       []string{"low", "high"},
								DefaultReasoningEffort: "high",
							},
							{ID: "gpt-5.4-mini", DisplayName: "GPT-5.4 Mini"},
						},
					},
					CredentialSlots: []contract.SettingsProviderCredentialSlotPayload{
						{
							Name:      "api_key",
							TargetEnv: "OPENAI_API_KEY",
							SecretRef: "env:OPENAI_API_KEY",
							Kind:      "api_key",
							Required:  true,
						},
					},
				},
			},
			assert: func(t *testing.T, service *stubSettingsService) {
				t.Helper()
				if service.LastPutCollectionRequest.Provider == nil ||
					service.LastPutCollectionRequest.Provider.Models.Default != "gpt-5.4" {
					t.Fatalf("LastPutCollectionRequest.Provider = %#v", service.LastPutCollectionRequest.Provider)
				}
				if got := service.LastPutCollectionRequest.Provider.Models.Curated; len(got) != 2 ||
					got[0].ID != "gpt-5.4" ||
					got[1].ID != "gpt-5.4-mini" {
					t.Fatalf("Provider.Models.Curated = %#v", got)
				}
				model := service.LastPutCollectionRequest.Provider.Models.Curated[0]
				if model.SupportsReasoning == nil || !*model.SupportsReasoning {
					t.Fatalf(
						"Provider.Models.Curated[0].SupportsReasoning = %#v, want true",
						model.SupportsReasoning,
					)
				}
				if got, want := model.DefaultReasoningEffort, "high"; got != want {
					t.Fatalf("Provider.Models.Curated[0].DefaultReasoningEffort = %q, want %q", got, want)
				}
			},
			assertResponse: assertAppliedSettingsMutation,
		},
		{
			name:   "put sandbox",
			method: http.MethodPut,
			path:   "/api/settings/sandboxes/local",
			body: contract.PutSettingsSandboxRequest{
				Profile: contract.SettingsSandboxProfilePayload{
					Backend:     "local",
					SyncMode:    "session-bidirectional",
					Persistence: "reuse",
					RuntimeRoot: "/workspace",
				},
			},
			assert: func(t *testing.T, service *stubSettingsService) {
				t.Helper()
				if service.LastPutCollectionRequest.Sandbox == nil ||
					service.LastPutCollectionRequest.Sandbox.Backend != "local" {
					t.Fatalf("LastPutCollectionRequest.Sandbox = %#v", service.LastPutCollectionRequest.Sandbox)
				}
			},
			assertResponse: assertAppliedSettingsMutation,
		},
		{
			name:   "put hook",
			method: http.MethodPut,
			path:   "/api/settings/hooks/capture",
			body: contract.PutSettingsHookRequest{
				Declaration: contract.SettingsHookDeclarationPayload{
					Name:         "capture",
					Event:        hookspkg.HookToolPreCall,
					Mode:         hookspkg.HookModeAsync,
					ExecutorKind: hookspkg.HookExecutorSubprocess,
					Command:      "/bin/capture",
					Matcher: hookspkg.HookMatcher{
						ToolID:       "agh__read",
						ToolReadOnly: &readOnly,
					},
				},
			},
			assert: func(t *testing.T, service *stubSettingsService) {
				t.Helper()
				if service.LastPutCollectionRequest.Hook == nil ||
					service.LastPutCollectionRequest.Hook.Name != "capture" {
					t.Fatalf("LastPutCollectionRequest.Hook = %#v", service.LastPutCollectionRequest.Hook)
				}
			},
			assertResponse: assertAppliedSettingsMutation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &stubSettingsService{
				ListCollectionFn: func(_ context.Context, req settingspkg.CollectionRequest) (settingspkg.CollectionEnvelope, error) {
					envelope := settingspkg.CollectionEnvelope{
						Collection:      req.Collection,
						Scope:           req.Scope,
						WorkspaceID:     req.WorkspaceID,
						AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal, settingspkg.ScopeWorkspace},
					}
					switch req.Collection {
					case settingspkg.CollectionProviders:
						envelope.Providers = []settingspkg.ProviderItem{
							{
								Name: "openai",
								Settings: settingspkg.ProviderSettings{
									Command: "codex",
									Models: aghconfig.ProviderModelsConfig{
										Default: "gpt-5.4",
									},
									CredentialSlots: []aghconfig.ProviderCredentialSlot{
										{
											Name:      "api_key",
											TargetEnv: "OPENAI_API_KEY",
											SecretRef: "env:OPENAI_API_KEY",
											Kind:      "api_key",
											Required:  true,
										},
									},
								},
								CommandAvailable: true,
								SourceMetadata: settingspkg.SourceMetadata{
									EffectiveSource: settingspkg.SourceRef{
										Kind:  settingspkg.SourceKindGlobalConfig,
										Scope: settingspkg.ScopeGlobal,
									},
									AvailableTargets: []settingspkg.WriteTargetKind{
										settingspkg.WriteTargetGlobalConfig,
									},
								},
							},
						}
					case settingspkg.CollectionMCPServers:
						envelope.MCPServers = []settingspkg.MCPServerItem{{
							Name:        "memory",
							Command:     "memoryd",
							Scope:       req.Scope,
							WorkspaceID: req.WorkspaceID,
							SourceMetadata: settingspkg.SourceMetadata{
								EffectiveSource: settingspkg.SourceRef{
									Kind:        settingspkg.SourceKindWorkspaceMCPSidecar,
									Scope:       req.Scope,
									WorkspaceID: req.WorkspaceID,
								},
								AvailableTargets: []settingspkg.WriteTargetKind{
									settingspkg.WriteTargetWorkspaceMCPSidecar,
								},
							},
						}}
					case settingspkg.CollectionSandboxes:
						envelope.Sandboxes = []settingspkg.SandboxItem{{
							Name: "local",
							Profile: aghconfig.SandboxProfile{
								Backend: "local",
							},
							SourceMetadata: settingspkg.SourceMetadata{
								EffectiveSource: settingspkg.SourceRef{
									Kind:  settingspkg.SourceKindGlobalConfig,
									Scope: settingspkg.ScopeGlobal,
								},
								AvailableTargets: []settingspkg.WriteTargetKind{
									settingspkg.WriteTargetGlobalConfig,
								},
							},
						}}
					case settingspkg.CollectionHooks:
						envelope.Hooks = []settingspkg.HookItem{{
							Name: "capture",
							Declaration: hookspkg.HookDecl{
								Name:         "capture",
								Event:        hookspkg.HookToolPreCall,
								Mode:         hookspkg.HookModeAsync,
								ExecutorKind: hookspkg.HookExecutorSubprocess,
								Command:      "/bin/capture",
								Matcher:      hookspkg.HookMatcher{ToolID: "agh__read"},
							},
							SourceMetadata: settingspkg.SourceMetadata{
								EffectiveSource: settingspkg.SourceRef{
									Kind:  settingspkg.SourceKindGlobalConfig,
									Scope: settingspkg.ScopeGlobal,
								},
								AvailableTargets: []settingspkg.WriteTargetKind{
									settingspkg.WriteTargetGlobalConfig,
								},
							},
						}}
					}
					return envelope, nil
				},
				PutCollectionItemFn: func(_ context.Context, req settingspkg.CollectionItemPutRequest) (settingspkg.MutationResult, error) {
					return settingspkg.MutationResult{
						Section:  settingspkg.SectionName(req.Collection),
						Scope:    req.Scope,
						Behavior: settingspkg.MutationBehaviorAppliedNow,
						Applied:  true,
					}, nil
				},
			}
			fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

			var body []byte
			if tc.body != nil {
				body = mustJSON(t, tc.body)
			}
			resp := performRequest(t, fixture.Engine, tc.method, tc.path, body)
			if got, want := resp.Code, http.StatusOK; got != want {
				t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
			}
			tc.assertResponse(t, resp)
			tc.assert(t, service)
		})
	}
}

func assertAppliedSettingsMutation(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()

	var payload contract.SettingsApplyResponse
	testutil.DecodeJSONResponse(t, resp, &payload)
	if !payload.Applied || payload.Lifecycle != contract.SettingsApplyLifecycleLive {
		t.Fatalf("settings mutation payload = %#v, want live", payload)
	}
}

func TestSettingsCollectionMutationHandlersRejectInvalidPayloads(t *testing.T) {
	t.Parallel()

	readOnly := true
	tests := []struct {
		name              string
		method            string
		path              string
		body              []byte
		want              string
		expectPutCalls    int
		expectDeleteCalls int
	}{
		{
			name:   "provider missing settings",
			method: http.MethodPut,
			path:   "/api/settings/providers/openai",
			body:   []byte(`{}`),
			want:   "provider.settings is required",
		},
		{
			name:   "mcp server missing body",
			method: http.MethodPut,
			path:   "/api/settings/mcp-servers/server-a",
			body:   []byte(`{}`),
			want:   "mcp-servers.server is required",
		},
		{
			name:   "sandbox missing profile",
			method: http.MethodPut,
			path:   "/api/settings/sandboxes/local",
			body:   []byte(`{}`),
			want:   "sandboxes.profile is required",
		},
		{
			name:   "hook missing declaration",
			method: http.MethodPut,
			path:   "/api/settings/hooks/capture",
			body:   []byte(`{}`),
			want:   "hooks.declaration is required",
		},
		{
			name:   "delete invalid target",
			method: http.MethodDelete,
			path:   "/api/settings/mcp-servers/server-a?target=invalid",
			want:   "settings.target must be one of",
		},
		{
			name:   "hook invalid timeout",
			method: http.MethodPut,
			path:   "/api/settings/hooks/capture",
			body: mustJSON(t, contract.PutSettingsHookRequest{
				Declaration: contract.SettingsHookDeclarationPayload{
					Name:         "capture",
					Event:        hookspkg.HookToolPreCall,
					Mode:         hookspkg.HookModeAsync,
					ExecutorKind: hookspkg.HookExecutorSubprocess,
					Command:      "/bin/capture",
					Timeout:      "bad",
					Matcher: hookspkg.HookMatcher{
						ToolID:       "agh__read",
						ToolReadOnly: &readOnly,
					},
				},
			}),
			want: "hooks.declaration.timeout",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &stubSettingsService{}
			fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

			resp := performRequest(t, fixture.Engine, tc.method, tc.path, tc.body)
			if got, want := resp.Code, http.StatusBadRequest; got != want {
				t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
			}
			if service.PutCollectionItemCalls != 0 {
				t.Fatalf("PutCollectionItemCalls = %d, want 0", service.PutCollectionItemCalls)
			}
			if service.DeleteItemCalls != 0 {
				t.Fatalf("DeleteItemCalls = %d, want 0", service.DeleteItemCalls)
			}

			var payload contract.ErrorPayload
			decodeJSON(t, resp.Body.Bytes(), &payload)
			if !strings.Contains(payload.Error, tc.want) {
				t.Fatalf("payload.Error = %q, want substring %q", payload.Error, tc.want)
			}
		})
	}
}

func TestSettingsRemainingReadAndDeleteHandlers(t *testing.T) {
	t.Parallel()

	service := &stubSettingsService{
		GetSectionFn: func(_ context.Context, req settingspkg.SectionRequest) (settingspkg.SectionEnvelope, error) {
			envelope := settingspkg.SectionEnvelope{
				Section:         req.Section,
				Scope:           req.Scope,
				WorkspaceID:     req.WorkspaceID,
				AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			}
			switch req.Section {
			case settingspkg.SectionMemory:
				envelope.Memory = &settingspkg.MemorySection{
					Config: aghconfig.MemoryConfig{
						Enabled: true,
						Dream: aghconfig.DreamConfig{
							Agent:         "dreamer",
							CheckInterval: time.Hour,
						},
					},
				}
			case settingspkg.SectionSkills:
				envelope.Skills = &settingspkg.SkillsSection{
					Config: aghconfig.SkillsConfig{
						Enabled:      true,
						PollInterval: time.Minute,
						Marketplace:  aghconfig.MarketplaceConfig{Registry: "clawhub"},
					},
				}
			case settingspkg.SectionAutomation:
				envelope.Automation = &settingspkg.AutomationSection{
					Config: settingspkg.AutomationSettings{
						Enabled:           true,
						Timezone:          "UTC",
						MaxConcurrentJobs: 1,
						DefaultFireLimit:  automationmodel.FireLimitConfig{Max: 1, Window: "1m"},
					},
				}
			case settingspkg.SectionNetwork:
				envelope.Network = &settingspkg.NetworkSection{
					Config: aghconfig.NetworkConfig{
						Enabled: true,
					},
				}
			case settingspkg.SectionHooksExtensions:
				envelope.HooksExtensions = &settingspkg.HooksExtensionsSection{
					Extensions: aghconfig.ExtensionsConfig{
						Marketplace: aghconfig.ExtensionsMarketplaceConfig{Registry: "github"},
					},
				}
			default:
				return settingspkg.SectionEnvelope{}, errors.New("unexpected section")
			}
			return envelope, nil
		},
		ListCollectionFn: func(_ context.Context, req settingspkg.CollectionRequest) (settingspkg.CollectionEnvelope, error) {
			envelope := settingspkg.CollectionEnvelope{
				Collection:      req.Collection,
				Scope:           req.Scope,
				WorkspaceID:     req.WorkspaceID,
				AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
			}
			switch req.Collection {
			case settingspkg.CollectionSandboxes:
				envelope.Sandboxes = []settingspkg.SandboxItem{{
					Name: "local",
					Profile: aghconfig.SandboxProfile{
						Backend: "local",
					},
					SourceMetadata: settingspkg.SourceMetadata{
						EffectiveSource: settingspkg.SourceRef{
							Kind:  settingspkg.SourceKindGlobalConfig,
							Scope: settingspkg.ScopeGlobal,
						},
						AvailableTargets: []settingspkg.WriteTargetKind{
							settingspkg.WriteTargetGlobalConfig,
						},
					},
				}}
			default:
				return settingspkg.CollectionEnvelope{}, errors.New("unexpected collection")
			}
			return envelope, nil
		},
		DeleteItemFn: func(_ context.Context, req settingspkg.CollectionItemDeleteRequest) (settingspkg.MutationResult, error) {
			return settingspkg.MutationResult{
				Section:  settingspkg.SectionName(req.Collection),
				Scope:    req.Scope,
				Behavior: settingspkg.MutationBehaviorAppliedNow,
				Applied:  true,
			}, nil
		},
	}
	fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

	for _, path := range []string{
		"/api/settings/memory",
		"/api/settings/skills",
		"/api/settings/automation",
		"/api/settings/network",
		"/api/settings/hooks-extensions",
		"/api/settings/sandboxes",
	} {
		resp := performRequest(t, fixture.Engine, http.MethodGet, path, nil)
		if got, want := resp.Code, http.StatusOK; got != want {
			t.Fatalf("%s status = %d, want %d; body=%s", path, got, want, resp.Body.String())
		}
	}

	for _, path := range []string{
		"/api/settings/providers/openai",
		"/api/settings/sandboxes/local",
		"/api/settings/hooks/capture",
	} {
		resp := performRequest(t, fixture.Engine, http.MethodDelete, path, nil)
		if got, want := resp.Code, http.StatusOK; got != want {
			t.Fatalf("%s status = %d, want %d; body=%s", path, got, want, resp.Body.String())
		}
	}
}

func TestSettingsRejectsAgentNameOutsideSkills(t *testing.T) {
	t.Parallel()

	service := &stubSettingsService{}
	fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

	resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/api/settings/general?agent_name=coder",
		nil,
	)
	if got, want := resp.Code, http.StatusBadRequest; got != want {
		t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
	}
	if service.GetSectionCalls != 0 {
		t.Fatalf("GetSectionCalls = %d, want 0", service.GetSectionCalls)
	}

	var payload contract.ErrorPayload
	decodeJSON(t, resp.Body.Bytes(), &payload)
	if !strings.Contains(payload.Error, "agent_name is only supported for skills") {
		t.Fatalf("payload.Error = %q, want agent_name validation", payload.Error)
	}
}

func TestSettingsHandlersReturnServiceUnavailableWithoutInjectedDependencies(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	engine := gin.New()
	engine.Use(gin.Recovery())
	registerSettingsRoutes(engine, core.NewBaseHandlers(&core.BaseHandlerConfig{
		TransportName: "api-core-http",
		HomePaths:     homePaths,
		Config:        cfg,
		Logger:        testutil.DiscardLogger(),
	}))

	tests := []struct {
		method string
		path   string
		body   []byte
	}{
		{method: http.MethodGet, path: "/api/settings/general"},
		{
			method: http.MethodPatch,
			path:   "/api/settings/general",
			body: mustJSON(t, contract.UpdateSettingsGeneralRequest{
				Config: contract.SettingsGeneralConfigPayload{
					Defaults: contract.SettingsDefaultsPayload{Agent: "coder"},
					Limits:   contract.SettingsLimitsPayload{MaxConcurrentAgents: 2},
					Permissions: contract.SettingsPermissionsPayload{
						Mode: contract.SettingsPermissionModeApproveReads,
					},
					SessionTimeout: "30m",
					HTTP:           contract.SettingsHTTPPayload{Host: "127.0.0.1", Port: 2123},
					Daemon:         contract.SettingsDaemonPayload{Socket: "/tmp/agh.sock"},
				},
			}),
		},
		{method: http.MethodGet, path: "/api/settings/providers"},
		{
			method: http.MethodPut,
			path:   "/api/settings/mcp-servers/server-a",
			body: mustJSON(t, contract.PutSettingsMCPServerRequest{
				Server: contract.SettingsMCPServerPayload{Name: "server-a", Command: "mcpd"},
			}),
		},
		{method: http.MethodDelete, path: "/api/settings/hooks/capture"},
		{method: http.MethodPost, path: "/api/settings/actions/restart", body: []byte(`{}`)},
		{method: http.MethodGet, path: "/api/settings/actions/restart/op-123"},
	}

	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			t.Parallel()

			resp := performRequest(t, engine, tc.method, tc.path, tc.body)
			if got, want := resp.Code, http.StatusServiceUnavailable; got != want {
				t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
			}
		})
	}
}

func TestGetSettingsProviderMissingResourceReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &stubSettingsService{
		ListCollectionFn: func(context.Context, settingspkg.CollectionRequest) (settingspkg.CollectionEnvelope, error) {
			return settingspkg.CollectionEnvelope{
				Collection: settingspkg.CollectionProviders,
				Scope:      settingspkg.ScopeGlobal,
			}, nil
		},
	}
	fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/api/settings/providers/missing", nil)
	if got, want := resp.Code, http.StatusNotFound; got != want {
		t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
	}
}

func TestGetSettingsGeneralUnsupportedScopeReturnsConflict(t *testing.T) {
	t.Parallel()

	service := &stubSettingsService{
		GetSectionFn: func(context.Context, settingspkg.SectionRequest) (settingspkg.SectionEnvelope, error) {
			return settingspkg.SectionEnvelope{}, fmt.Errorf(
				"%w: %s",
				settingspkg.ErrConflict,
				`settings: section "general" does not support workspace scope`,
			)
		},
	}
	fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

	resp := performRequest(
		t,
		fixture.Engine,
		http.MethodGet,
		"/api/settings/general?scope=workspace&workspace_id=ws-1",
		nil,
	)
	if got, want := resp.Code, http.StatusConflict; got != want {
		t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
	}
	if got := service.LastGetSectionRequest.Scope; got != settingspkg.ScopeWorkspace {
		t.Fatalf("LastGetSectionRequest.Scope = %q, want %q", got, settingspkg.ScopeWorkspace)
	}
	if got := service.LastGetSectionRequest.WorkspaceID; got != "ws-1" {
		t.Fatalf("LastGetSectionRequest.WorkspaceID = %q, want ws-1", got)
	}
}

func TestTriggerSettingsRestartUsesActionHandlerInsteadOfSettingsMutation(t *testing.T) {
	t.Parallel()

	service := &stubSettingsService{}
	restartController := &stubSettingsRestartController{
		RequestFn: func(context.Context) (core.SettingsRestartOperation, error) {
			return core.SettingsRestartOperation{
				OperationID:        "op-123",
				Status:             "stopping",
				ActiveSessionCount: 3,
			}, nil
		},
	}
	fixture := newSettingsHandlerFixture(t, "api-core-http", service, restartController)

	resp := performRequest(t, fixture.Engine, http.MethodPost, "/api/settings/actions/restart", []byte(`{}`))
	if got, want := resp.Code, http.StatusAccepted; got != want {
		t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
	}
	if restartController.RequestCalls != 1 {
		t.Fatalf("RequestCalls = %d, want 1", restartController.RequestCalls)
	}
	if service.UpdateSectionCalls != 0 {
		t.Fatalf("UpdateSectionCalls = %d, want 0", service.UpdateSectionCalls)
	}

	var payload contract.RestartActionResponse
	decodeJSON(t, resp.Body.Bytes(), &payload)
	if payload.OperationID != "op-123" {
		t.Fatalf("payload.OperationID = %q, want op-123", payload.OperationID)
	}
	if payload.StatusURL != "/api/settings/actions/restart/op-123" {
		t.Fatalf("payload.StatusURL = %q, want restart polling URL", payload.StatusURL)
	}
	if payload.ActiveSessionCount != 3 {
		t.Fatalf("payload.ActiveSessionCount = %d, want 3", payload.ActiveSessionCount)
	}
}

func TestGetSettingsRestartStatusReturnsPersistedOperationShape(t *testing.T) {
	t.Parallel()

	completedAt := time.Date(2026, 4, 17, 18, 50, 0, 0, time.UTC)
	restartController := &stubSettingsRestartController{
		StatusFn: func(_ context.Context, operationID string) (core.SettingsRestartOperation, error) {
			return core.SettingsRestartOperation{
				OperationID:        operationID,
				Status:             "ready",
				OldPID:             4100,
				OldStartedAt:       time.Date(2026, 4, 17, 18, 0, 0, 0, time.UTC),
				OldSocketPath:      "/tmp/agh.sock",
				NewPID:             4200,
				ActiveSessionCount: 5,
				StartedAt:          time.Date(2026, 4, 17, 18, 45, 0, 0, time.UTC),
				UpdatedAt:          completedAt,
				CompletedAt:        &completedAt,
			}, nil
		},
	}
	fixture := newSettingsHandlerFixture(t, "api-core-http", &stubSettingsService{}, restartController)

	resp := performRequest(t, fixture.Engine, http.MethodGet, "/api/settings/actions/restart/op-ready", nil)
	if got, want := resp.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d; body=%s", got, want, resp.Body.String())
	}
	if restartController.LastOperationID != "op-ready" {
		t.Fatalf("LastOperationID = %q, want op-ready", restartController.LastOperationID)
	}

	var payload contract.RestartActionStatus
	decodeJSON(t, resp.Body.Bytes(), &payload)
	if payload.OperationID != "op-ready" || payload.NewPID != 4200 || payload.ActiveSessionCount != 5 {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.CompletedAt == nil || !payload.CompletedAt.Equal(completedAt) {
		t.Fatalf("payload.CompletedAt = %v, want %v", payload.CompletedAt, completedAt)
	}
}

func TestSettingsMCPServerMutationsPreserveScopeWorkspaceTargetAndMutationMetadata(t *testing.T) {
	t.Parallel()

	service := &stubSettingsService{
		PutCollectionItemFn: func(_ context.Context, req settingspkg.CollectionItemPutRequest) (settingspkg.MutationResult, error) {
			return settingspkg.MutationResult{
				Section:         settingspkg.SectionName(req.Collection),
				Scope:           req.Scope,
				WriteTarget:     settingspkg.WriteTargetWorkspaceMCPSidecar,
				WorkspaceID:     req.WorkspaceID,
				Behavior:        settingspkg.MutationBehaviorRestartRequired,
				Applied:         false,
				RestartRequired: true,
				RestartScope:    "daemon",
				Warnings:        []string{"restart required"},
			}, nil
		},
		DeleteItemFn: func(_ context.Context, req settingspkg.CollectionItemDeleteRequest) (settingspkg.MutationResult, error) {
			return settingspkg.MutationResult{
				Section:         settingspkg.SectionName(req.Collection),
				Scope:           req.Scope,
				WriteTarget:     settingspkg.WriteTargetWorkspaceMCPSidecar,
				WorkspaceID:     req.WorkspaceID,
				Behavior:        settingspkg.MutationBehaviorRestartRequired,
				Applied:         false,
				RestartRequired: true,
				RestartScope:    "daemon",
			}, nil
		},
	}
	fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

	putBody := mustJSON(t, contract.PutSettingsMCPServerRequest{
		Server: contract.SettingsMCPServerPayload{
			Name:    "server-a",
			Command: "mcpd",
			Args:    []string{"serve"},
			SecretEnv: map[string]string{
				"TOKEN": "vault:mcp/server-a/env/TOKEN",
			},
		},
		SecretValues: &contract.SettingsMCPSecretValuesPayload{
			SecretEnv: map[string]string{"TOKEN": "server-token"},
		},
	})
	putResp := performRequest(
		t,
		fixture.Engine,
		http.MethodPut,
		"/api/settings/mcp-servers/server-a?scope=workspace&workspace_id=ws-123&target=sidecar",
		putBody,
	)
	if got, want := putResp.Code, http.StatusOK; got != want {
		t.Fatalf("PUT status = %d, want %d; body=%s", got, want, putResp.Body.String())
	}
	if got := service.LastPutCollectionRequest.Scope; got != settingspkg.ScopeWorkspace {
		t.Fatalf("LastPutCollectionRequest.Scope = %q, want %q", got, settingspkg.ScopeWorkspace)
	}
	if got := service.LastPutCollectionRequest.WorkspaceID; got != "ws-123" {
		t.Fatalf("LastPutCollectionRequest.WorkspaceID = %q, want ws-123", got)
	}
	if got := service.LastPutCollectionRequest.Target; got != settingspkg.TargetSidecar {
		t.Fatalf("LastPutCollectionRequest.Target = %q, want %q", got, settingspkg.TargetSidecar)
	}
	if service.LastPutCollectionRequest.MCPServer == nil ||
		service.LastPutCollectionRequest.MCPServer.Name != "server-a" {
		t.Fatalf(
			"LastPutCollectionRequest.MCPServer = %#v, want populated request payload",
			service.LastPutCollectionRequest.MCPServer,
		)
	}
	if got, want := service.LastPutCollectionRequest.MCPSecrets.SecretEnv["TOKEN"], "server-token"; got != want {
		t.Fatalf("LastPutCollectionRequest.MCPSecrets.SecretEnv[TOKEN] = %q, want %q", got, want)
	}
	if strings.Contains(putResp.Body.String(), "server-token") {
		t.Fatalf("PUT response leaked raw secret value: %s", putResp.Body.String())
	}

	var putPayload contract.SettingsApplyResponse
	decodeJSON(t, putResp.Body.Bytes(), &putPayload)
	if putPayload.ApplyRecordID == "" || !putPayload.Applied {
		t.Fatalf("putPayload = %#v", putPayload)
	}

	deleteResp := performRequest(
		t,
		fixture.Engine,
		http.MethodDelete,
		"/api/settings/mcp-servers/server-a?scope=workspace&workspace_id=ws-123&target=sidecar",
		nil,
	)
	if got, want := deleteResp.Code, http.StatusOK; got != want {
		t.Fatalf("DELETE status = %d, want %d; body=%s", got, want, deleteResp.Body.String())
	}
	if got := service.LastDeleteRequest.Scope; got != settingspkg.ScopeWorkspace {
		t.Fatalf("LastDeleteRequest.Scope = %q, want %q", got, settingspkg.ScopeWorkspace)
	}
	if got := service.LastDeleteRequest.WorkspaceID; got != "ws-123" {
		t.Fatalf("LastDeleteRequest.WorkspaceID = %q, want ws-123", got)
	}
	if got := service.LastDeleteRequest.Target; got != settingspkg.TargetSidecar {
		t.Fatalf("LastDeleteRequest.Target = %q, want %q", got, settingspkg.TargetSidecar)
	}
}

func TestListSettingsApplyRecordsReturnsBlockedDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("Should return blocked apply records with redacted diagnostics", func(t *testing.T) {
		t.Parallel()

		diagnostic := diagnostics.NewItem(
			"config.apply.restart_required",
			diagnosticcontract.CodeConfigRestartRequired,
			diagnosticcontract.CategoryConfig,
			"Daemon restart required",
			"Restart required for token=super-secret",
			diagnosticcontract.SeverityWarn,
			diagnosticcontract.FreshnessLive,
		)
		service := &stubSettingsService{
			ListApplyRecordsFn: func(
				_ context.Context,
				filter settingspkg.ApplyRecordFilter,
			) ([]settingspkg.ApplyRecord, error) {
				if got, want := filter.Status, lifecycle.StatusBlocked; got != want {
					t.Fatalf("filter.Status = %q, want %q", got, want)
				}
				return []settingspkg.ApplyRecord{{
					ID:          "cfgapp-1",
					DesiredHash: "sha256:desired",
					ActiveHash:  "sha256:active",
					Generation:  7,
					Actor:       "http",
					DiffClass:   lifecycle.DiffClassRestartRequired,
					Status:      lifecycle.StatusBlocked,
					Lifecycle:   lifecycle.RestartRequired,
					NextAction:  lifecycle.NextActionRestartDaemon,
					Diagnostics: []diagnosticcontract.DiagnosticItem{diagnostic},
					CreatedAt:   time.Unix(1, 0).UTC(),
					UpdatedAt:   time.Unix(2, 0).UTC(),
				}}, nil
			},
		}
		fixture := newSettingsHandlerFixture(t, "api-core-http", service, nil)

		resp := performRequest(t, fixture.Engine, http.MethodGet, "/api/settings/apply?status=blocked", nil)
		if got, want := resp.Code, http.StatusOK; got != want {
			t.Fatalf("GET status = %d, want %d; body=%s", got, want, resp.Body.String())
		}
		var payload contract.ConfigApplyRecordsResponse
		decodeJSON(t, resp.Body.Bytes(), &payload)
		if len(payload.Entries) != 1 {
			t.Fatalf("entries len = %d, want 1", len(payload.Entries))
		}
		entry := payload.Entries[0]
		if got, want := entry.Status, contract.ConfigApplyStatusBlocked; got != want {
			t.Fatalf("entry.Status = %q, want %q", got, want)
		}
		if len(entry.Diagnostics) != 1 {
			t.Fatalf("entry.Diagnostics len = %d, want 1", len(entry.Diagnostics))
		}
		if strings.Contains(entry.Diagnostics[0].Message, "super-secret") {
			t.Fatalf("diagnostic message leaked secret: %q", entry.Diagnostics[0].Message)
		}
	})
}

func TestSettingsHandlersBehaveIdenticallyAcrossTransportShims(t *testing.T) {
	t.Parallel()

	observabilityEnvelope := settingspkg.SectionEnvelope{
		Section:         settingspkg.SectionObservability,
		Scope:           settingspkg.ScopeGlobal,
		AvailableScopes: []settingspkg.ScopeKind{settingspkg.ScopeGlobal},
		Observability: &settingspkg.ObservabilitySection{
			Config: aghconfig.ObservabilityConfig{
				Enabled:        true,
				RetentionDays:  7,
				MaxGlobalBytes: 1024,
				Transcripts: aghconfig.ObservabilityTranscriptConfig{
					Enabled:            true,
					SegmentBytes:       2048,
					MaxBytesPerSession: 4096,
				},
			},
			Runtime: settingspkg.ObservabilityRuntimeStatus{
				Available:          true,
				Status:             "ok",
				GlobalDBSizeBytes:  10,
				SessionDBSizeBytes: 20,
				ActiveSessions:     2,
				ActiveAgents:       1,
				UptimeSeconds:      300,
			},
			LogTailSupport: settingspkg.CapabilityStatus{Available: true},
		},
	}
	serviceFactory := func() *stubSettingsService {
		return &stubSettingsService{
			GetSectionFn: func(_ context.Context, req settingspkg.SectionRequest) (settingspkg.SectionEnvelope, error) {
				if req.Section != settingspkg.SectionObservability {
					return settingspkg.SectionEnvelope{}, errors.New("unexpected section")
				}
				return observabilityEnvelope, nil
			},
		}
	}
	restartFactory := func() *stubSettingsRestartController {
		return &stubSettingsRestartController{
			RequestFn: func(context.Context) (core.SettingsRestartOperation, error) {
				return core.SettingsRestartOperation{
					OperationID:        "op-shared",
					Status:             "stopping",
					ActiveSessionCount: 2,
				}, nil
			},
			StatusFn: func(_ context.Context, operationID string) (core.SettingsRestartOperation, error) {
				return core.SettingsRestartOperation{
					OperationID:        operationID,
					Status:             "starting",
					OldPID:             100,
					OldStartedAt:       time.Date(2026, 4, 17, 18, 0, 0, 0, time.UTC),
					OldSocketPath:      "/tmp/agh.sock",
					ActiveSessionCount: 2,
					StartedAt:          time.Date(2026, 4, 17, 18, 44, 0, 0, time.UTC),
					UpdatedAt:          time.Date(2026, 4, 17, 18, 44, 30, 0, time.UTC),
				}, nil
			},
		}
	}

	httpFixture := newSettingsHandlerFixture(t, "httpapi", serviceFactory(), restartFactory())
	udsFixture := newSettingsHandlerFixture(t, "udsapi", serviceFactory(), restartFactory())

	for _, path := range []string{
		"/api/settings/observability",
		"/api/settings/actions/restart",
		"/api/settings/actions/restart/op-shared",
	} {
		var body []byte
		method := http.MethodGet
		if path == "/api/settings/actions/restart" {
			method = http.MethodPost
			body = []byte(`{}`)
		}

		httpResp := performRequest(t, httpFixture.Engine, method, path, body)
		udsResp := performRequest(t, udsFixture.Engine, method, path, body)
		if httpResp.Code != udsResp.Code {
			t.Fatalf("%s status mismatch: http=%d uds=%d", path, httpResp.Code, udsResp.Code)
		}
		if httpResp.Body.String() != udsResp.Body.String() {
			t.Fatalf("%s body mismatch:\nhttp=%s\nuds=%s", path, httpResp.Body.String(), udsResp.Body.String())
		}
	}

	var observability contract.SettingsObservabilityResponse
	decodeJSON(
		t,
		performRequest(t, httpFixture.Engine, http.MethodGet, "/api/settings/observability", nil).Body.Bytes(),
		&observability,
	)
	if !observability.LogTail.Available || observability.LogTail.StreamURL != "/api/settings/observability/log-tail" {
		t.Fatalf("observability.LogTail = %#v, want SSE metadata", observability.LogTail)
	}
	if observability.LogTail.Transport != contract.SettingsStreamTransportSSE {
		t.Fatalf(
			"observability.LogTail.Transport = %q, want %q",
			observability.LogTail.Transport,
			contract.SettingsStreamTransportSSE,
		)
	}
}

func TestStreamSettingsObservabilityLogTailEmitsSSEEvent(t *testing.T) {
	t.Parallel()

	fixture := newSettingsHandlerFixture(t, "api-core-http", &stubSettingsService{}, nil)
	if err := os.WriteFile(fixture.HomePaths.LogFile, []byte{}, 0o644); err != nil {
		t.Fatalf("WriteFile(log) error = %v", err)
	}

	server := httptest.NewServer(fixture.Engine)
	defer server.Close()

	reqCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodGet,
		server.URL+"/api/settings/observability/log-tail",
		http.NoBody,
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("client.Do() error = %v", err)
	}
	defer resp.Body.Close()

	appendLine(t, fixture.HomePaths.LogFile, "daemon restarted cleanly\n")

	reader := bufio.NewReader(resp.Body)
	first := readStreamLine(t, reader)
	second := readStreamLine(t, reader)
	cancel()

	if first != "event: log" {
		t.Fatalf("first SSE line = %q, want event header", first)
	}
	if !strings.Contains(second, `"line":"daemon restarted cleanly"`) {
		t.Fatalf("second SSE line = %q, want log payload", second)
	}
	if got := resp.Header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return payload
}

func decodeJSON(t *testing.T, body []byte, dest any) {
	t.Helper()
	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", string(body), err)
	}
}

func appendLine(t *testing.T, path string, line string) {
	t.Helper()

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatalf("OpenFile(%s) error = %v", path, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, bytes.NewBufferString(line)); err != nil {
		t.Fatalf("write log line error = %v", err)
	}
	if err := file.Sync(); err != nil {
		t.Fatalf("file.Sync() error = %v", err)
	}
}

func readStreamLine(t *testing.T, reader *bufio.Reader) string {
	t.Helper()

	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := reader.ReadString('\n')
		ch <- result{line: strings.TrimRight(line, "\r\n"), err: err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("ReadString() error = %v", res.err)
		}
		return res.line
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for SSE line")
		return ""
	}
}
