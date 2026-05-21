package udsapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	core "github.com/pedronauck/agh/internal/api/core"
	apispec "github.com/pedronauck/agh/internal/api/spec"
	apitestutil "github.com/pedronauck/agh/internal/api/testutil"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/network"
	"github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

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

func TestRegisterRoutesCoversTechSpecEndpoints(t *testing.T) {
	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	routes := engine.Routes()
	got := make([]string, 0, len(routes))
	for _, route := range routes {
		got = append(got, route.Method+" "+route.Path)
	}
	sort.Strings(got)

	want := []string{
		"DELETE /api/agents/:name/heartbeat",
		"DELETE /api/agents/:name/soul",
		"DELETE /api/automation/jobs/:id",
		"DELETE /api/automation/triggers/:id",
		"DELETE /api/bridges/:id/secret-bindings/:binding_name",
		"DELETE /api/bundles/activations/:id",
		"DELETE /api/memory/:filename",
		"DELETE /api/settings/sandboxes/:name",
		"DELETE /api/settings/hooks/:name",
		"DELETE /api/settings/mcp-servers/:name",
		"DELETE /api/settings/providers/:name",
		"DELETE /api/resources/:kind/:id",
		"DELETE /api/skills/marketplace/:name",
		"DELETE /api/workspaces/:workspace_id/sessions/:session_id",
		"DELETE /api/tasks/:id",
		"DELETE /api/tasks/:id/notifications/bridges/:subscription_id",
		"DELETE /api/tasks/:id/dependencies/:depends_on_id",
		"DELETE /api/tasks/:id/execution-profile",
		"DELETE /api/vault/secrets",
		"DELETE /api/workspaces/:workspace_id",
		"GET /api/agents",
		"GET /api/agents/:name",
		"GET /api/agents/:name/heartbeat",
		"GET /api/agents/:name/heartbeat/history",
		"GET /api/agents/:name/heartbeat/status",
		"GET /api/agents/:name/soul",
		"GET /api/agents/:name/soul/history",
		"GET /api/agent/channels",
		"GET /api/agent/channels/:channel/recv",
		"GET /api/agent/context",
		"GET /api/agent/coordinator/config",
		"GET /api/agent/me",
		"GET /api/agent/soul",
		"GET /api/automation/jobs",
		"GET /api/automation/jobs/:id",
		"GET /api/automation/jobs/:id/runs",
		"GET /api/automation/runs",
		"GET /api/automation/runs/:id",
		"GET /api/automation/triggers",
		"GET /api/automation/triggers/:id",
		"GET /api/automation/triggers/:id/runs",
		"GET /api/bridges",
		"GET /api/bridges/:id",
		"GET /api/bridges/health/stream",
		"GET /api/bridges/:id/routes",
		"GET /api/bridges/:id/secret-bindings",
		"GET /api/bridges/providers",
		"GET /api/bundles/activations",
		"GET /api/bundles/activations/:id",
		"GET /api/bundles/catalog",
		"GET /api/bundles/network/settings",
		"GET /api/doctor",
		"GET /api/extensions",
		"GET /api/extensions/:name",
		"GET /api/hooks/catalog",
		"GET /api/hooks/events",
		"GET /api/workspaces/:workspace_id/hooks/runs",
		"GET /api/internal/hosted-mcp/projection",
		"GET /api/internal/hosted-mcp/projection/stream",
		"GET /api/logs",
		"GET /api/logs/stream",
		"GET /api/memory",
		"GET /api/memory/:filename",
		"GET /api/memory/config",
		"GET /api/memory/daily",
		"GET /api/memory/decisions",
		"GET /api/memory/decisions/:decision_id",
		"GET /api/memory/dreams",
		"GET /api/memory/dreams/:dream_id",
		"GET /api/memory/dreams/status",
		"GET /api/memory/extractor/failures",
		"GET /api/memory/extractor/status",
		"GET /api/memory/health",
		"GET /api/memory/history",
		"GET /api/memory/providers",
		"GET /api/memory/providers/:provider_name",
		"GET /api/memory/recall-traces/:session_id/:turn_seq",
		"GET /api/memory/scope-show",
		"GET /api/workspaces/:workspace_id/memory/sessions/:session_id/ledger",
		"GET /api/workspaces/:workspace_id/network/inbox",
		"GET /api/workspaces/:workspace_id/network/peers",
		"GET /api/workspaces/:workspace_id/network/peers/:peer_id",
		"GET /api/workspaces/:workspace_id/network/channels",
		"GET /api/workspaces/:workspace_id/network/channels/:channel",
		"GET /api/workspaces/:workspace_id/network/channels/:channel/directs",
		"GET /api/workspaces/:workspace_id/network/channels/:channel/directs/:direct_id",
		"GET /api/workspaces/:workspace_id/network/channels/:channel/directs/:direct_id/messages",
		"GET /api/workspaces/:workspace_id/network/channels/:channel/threads",
		"GET /api/workspaces/:workspace_id/network/channels/:channel/threads/:thread_id",
		"GET /api/workspaces/:workspace_id/network/channels/:channel/threads/:thread_id/messages",
		"GET /api/network/status",
		"GET /api/workspaces/:workspace_id/network/work/:work_id",
		"GET /api/status",
		"GET /api/observe/tasks/dashboard",
		"GET /api/observe/tasks/inbox",
		"GET /api/model-catalog/*catalog_path",
		"GET /api/providers",
		"GET /api/providers/:provider_id",
		"GET /api/resources",
		"GET /api/resources/:kind",
		"GET /api/resources/:kind/:id",
		"GET /api/sessions",
		"GET /api/workspaces/:workspace_id/sessions/:session_id",
		"GET /api/workspaces/:workspace_id/sessions/:session_id/events",
		"GET /api/workspaces/:workspace_id/sessions/:session_id/health",
		"GET /api/workspaces/:workspace_id/sessions/:session_id/history",
		"GET /api/workspaces/:workspace_id/sessions/:session_id/inspect",
		"GET /api/workspaces/:workspace_id/sessions/:session_id/recap",
		"GET /api/workspaces/:workspace_id/sessions/:session_id/status",
		"GET /api/workspaces/:workspace_id/sessions/:session_id/transcript",
		"GET /api/workspaces/:workspace_id/sessions/:session_id/stream",
		"GET /api/workspaces/:workspace_id/sessions/:session_id/tools",
		"GET /api/settings/actions/restart/:operation_id",
		"GET /api/settings/apply",
		"GET /api/settings/automation",
		"GET /api/settings/sandboxes",
		"GET /api/settings/sandboxes/:name",
		"GET /api/settings/general",
		"GET /api/settings/update",
		"GET /api/settings/hooks",
		"GET /api/settings/hooks-extensions",
		"GET /api/settings/mcp-servers",
		"GET /api/settings/memory",
		"GET /api/settings/network",
		"GET /api/settings/observability",
		"GET /api/settings/observability/log-tail",
		"GET /api/settings/providers",
		"GET /api/settings/providers/:name",
		"GET /api/settings/skills",
		"GET /api/support/bundles/:operation_id",
		"GET /api/support/bundles/:operation_id/download",
		"GET /api/skills",
		"GET /api/skills/:name",
		"GET /api/skills/:name/content",
		"GET /api/skills/marketplace/info",
		"GET /api/skills/marketplace/search",
		"GET /api/task-runs/:id",
		"GET /api/task-runs/:id/reviews",
		"GET /api/task-reviews/:id",
		"GET /api/tasks",
		"GET /api/tasks/:id",
		"GET /api/tasks/:id/notifications/bridges",
		"GET /api/tasks/:id/notifications/bridges/:subscription_id",
		"GET /api/tasks/:id/execution-profile",
		"GET /api/tasks/:id/reviews",
		"GET /api/tasks/:id/stream",
		"GET /api/tasks/:id/timeline",
		"GET /api/tasks/:id/tree",
		"GET /api/tasks/:id/runs",
		"GET /api/tools",
		"GET /api/tools/:id",
		"GET /api/toolsets",
		"GET /api/toolsets/:id",
		"GET /api/vault/secrets",
		"GET /api/vault/secrets/metadata",
		"GET /api/workspaces",
		"GET /api/workspaces/:workspace_id",
		"PATCH /api/automation/jobs/:id",
		"PATCH /api/automation/triggers/:id",
		"PATCH /api/bridges/:id",
		"PATCH /api/bundles/activations/:id",
		"PATCH /api/memory/:filename",
		"PATCH /api/settings/automation",
		"PATCH /api/settings/general",
		"PATCH /api/settings/hooks-extensions",
		"PATCH /api/settings/memory",
		"PATCH /api/settings/network",
		"PATCH /api/settings/observability",
		"PATCH /api/settings/skills",
		"PATCH /api/tasks/:id",
		"PATCH /api/workspaces/:workspace_id",
		"POST /api/automation/jobs",
		"POST /api/automation/jobs/:id/trigger",
		"POST /api/automation/triggers",
		"POST /api/bridges",
		"POST /api/bridges/:id/disable",
		"POST /api/bridges/:id/enable",
		"POST /api/bridges/:id/restart",
		"POST /api/bridges/:id/test-delivery",
		"POST /api/bundles/activations",
		"POST /api/bundles/preview",
		"POST /api/agent/channels/:channel/send",
		"POST /api/agent/channels/reply",
		"POST /api/agent/soul/validate",
		"POST /api/agent/spawn",
		"POST /api/agent/tasks/:run_id/complete",
		"POST /api/agent/tasks/:run_id/fail",
		"POST /api/agent/tasks/:run_id/heartbeat",
		"POST /api/agent/tasks/:run_id/release",
		"POST /api/agent/tasks/claim-next",
		"POST /api/agents",
		"POST /api/agents/:name/heartbeat/rollback",
		"POST /api/agents/:name/heartbeat/validate",
		"POST /api/agents/:name/heartbeat/wake",
		"POST /api/agents/:name/soul/rollback",
		"POST /api/agents/:name/soul/validate",
		"POST /api/extensions",
		"POST /api/extensions/:name/disable",
		"POST /api/extensions/:name/enable",
		"POST /api/internal/hosted-mcp/bind",
		"POST /api/internal/hosted-mcp/release",
		"POST /api/internal/hosted-mcp/tools/call",
		"POST /api/memory",
		"POST /api/memory/ad-hoc",
		"POST /api/memory/decisions/:decision_id/revert",
		"POST /api/memory/dreams/:dream_id/retry",
		"POST /api/memory/dreams/trigger",
		"POST /api/memory/extractor/drain",
		"POST /api/memory/extractor/retry",
		"POST /api/memory/promote",
		"POST /api/memory/providers/:provider_name/disable",
		"POST /api/memory/providers/:provider_name/enable",
		"POST /api/memory/providers/select",
		"POST /api/memory/reindex",
		"POST /api/memory/reload",
		"POST /api/memory/reset",
		"POST /api/memory/search",
		"POST /api/memory/sessions/prune",
		"POST /api/memory/sessions/repair",
		"POST /api/workspaces/:workspace_id/memory/sessions/:session_id/replay",
		"POST /api/model-catalog/*catalog_path",
		"POST /api/providers/:provider_id/auth/probe",
		"POST /api/workspaces/:workspace_id/network/channels",
		"POST /api/workspaces/:workspace_id/network/channels/:channel/directs/resolve",
		"POST /api/workspaces/:workspace_id/network/send",
		"POST /api/sessions",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/approve",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/clear",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/prompt",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/prompt/cancel",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/interrupt",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/steer",
		"DELETE /api/workspaces/:workspace_id/sessions/:session_id/prompt/queue/:queue_entry_id",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/repair",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/attach",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/soul/refresh",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/stop",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/tools/search",
		"POST /api/settings/actions/restart",
		"POST /api/settings/reload",
		"POST /api/support/bundles",
		"POST /api/skills/:name/disable",
		"POST /api/skills/:name/enable",
		"POST /api/skills/marketplace/install",
		"POST /api/skills/marketplace/update",
		"POST /api/task-runs/:id/attach-session",
		"POST /api/task-runs/:id/cancel",
		"POST /api/task-runs/:id/claim",
		"POST /api/task-runs/:id/complete",
		"POST /api/task-runs/:id/fail",
		"POST /api/task-runs/:id/reviews",
		"POST /api/task-runs/:id/start",
		"POST /api/task-reviews/:id/verdict",
		"POST /api/tasks",
		"POST /api/tasks/:id/approve",
		"POST /api/tasks/:id/notifications/bridges",
		"POST /api/tasks/:id/cancel",
		"POST /api/tasks/:id/children",
		"POST /api/tasks/:id/dependencies",
		"POST /api/tasks/:id/publish",
		"POST /api/tasks/:id/reject",
		"POST /api/tasks/:id/runs",
		"POST /api/tasks/:id/start",
		"POST /api/tasks/:id/triage/archive",
		"POST /api/tasks/:id/triage/dismiss",
		"POST /api/tasks/:id/triage/read",
		"POST /api/tools/:id/approvals",
		"POST /api/tools/:id/invoke",
		"POST /api/tools/search",
		"POST /api/workspaces",
		"POST /api/workspaces/resolve",
		"PUT /api/agents/:name/heartbeat",
		"PUT /api/agents/:name/soul",
		"PUT /api/bridges/:id/secret-bindings/:binding_name",
		"PUT /api/settings/sandboxes/:name",
		"PUT /api/settings/hooks/:name",
		"PUT /api/settings/mcp-servers/:name",
		"PUT /api/settings/providers/:name",
		"PUT /api/resources/:kind/:id",
		"PUT /api/tasks/:id/execution-profile",
		"PUT /api/vault/secrets",
	}
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("len(routes) = %d, want %d\nroutes=%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("route[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRegisterRoutesRejectsLegacyStatusSurfaces(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	for _, path := range []string{"/api/daemon/status", "/api/observe/health"} {
		resp := performRequest(t, engine, http.MethodGet, path, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("GET %s status = %d, want %d", path, resp.Code, http.StatusNotFound)
		}
	}
}

func TestRegisterRoutesRejectsLegacyProviderModelCatalogSurfaces(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/api/providers/models"},
		{method: http.MethodGet, path: "/api/providers/codex/models"},
		{method: http.MethodPost, path: "/api/providers/codex/models/refresh"},
		{method: http.MethodGet, path: "/api/providers/codex/models/status"},
	} {
		resp := performRequest(t, engine, tc.method, tc.path, nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, want %d", tc.method, tc.path, resp.Code, http.StatusNotFound)
		}
	}
}

func TestMemoryRoutesMatchV2Contract(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	apitestutil.AssertMemoryV2RouteParity(t, apitestutil.MemoryV2RouteKeysFromGin(engine.Routes()))
}

func TestSettingsRoutesUseSharedCoreHandlers(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	if err := os.WriteFile(homePaths.LogFile, []byte("daemon booted\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", homePaths.LogFile, err)
	}

	settingsService := &stubSettingsService{}
	restartController := &stubSettingsRestartController{}
	handlers := newTestHandlersWithSettingsAndExtensions(
		t,
		settingsService,
		restartController,
		stubExtensionService{},
		homePaths,
	)
	engine := newTestRouter(t, handlers)

	tests := []struct {
		name       string
		method     string
		path       string
		body       []byte
		wantStatus int
		assert     func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "get general section",
			method:     http.MethodGet,
			path:       "/api/settings/general",
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				var response contract.SettingsGeneralResponse
				decodeJSONResponse(t, recorder, &response)
				if response.Section != contract.SettingsSectionGeneral {
					t.Fatalf("response.Section = %q, want %q", response.Section, contract.SettingsSectionGeneral)
				}
				if settingsService.LastGetSectionRequest.Section != settingspkg.SectionGeneral {
					t.Fatalf(
						"LastGetSectionRequest.Section = %q, want %q",
						settingsService.LastGetSectionRequest.Section,
						settingspkg.SectionGeneral,
					)
				}
			},
		},
		{
			name:       "patch general section",
			method:     http.MethodPatch,
			path:       "/api/settings/general",
			wantStatus: http.StatusOK,
			body: mustJSONBody(t, contract.UpdateSettingsGeneralRequest{
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
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				var response contract.SettingsGlobalSectionMutationResult
				decodeJSONResponse(t, recorder, &response)
				if response.Section != contract.SettingsSectionGeneral {
					t.Fatalf("response.Section = %q, want %q", response.Section, contract.SettingsSectionGeneral)
				}
				if settingsService.LastUpdateSectionRequest.Section != settingspkg.SectionGeneral {
					t.Fatalf(
						"LastUpdateSectionRequest.Section = %q, want %q",
						settingsService.LastUpdateSectionRequest.Section,
						settingspkg.SectionGeneral,
					)
				}
				if settingsService.LastUpdateSectionRequest.General == nil ||
					settingsService.LastUpdateSectionRequest.General.Defaults.Agent != "coder" {
					t.Fatalf(
						"LastUpdateSectionRequest.General = %#v, want parsed payload",
						settingsService.LastUpdateSectionRequest.General,
					)
				}
			},
		},
		{
			name:       "list scoped mcp servers",
			method:     http.MethodGet,
			path:       "/api/settings/mcp-servers?scope=workspace&workspace_id=ws-1",
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				var response contract.SettingsMCPServersResponse
				decodeJSONResponse(t, recorder, &response)
				if response.Scope != contract.SettingsWorkspaceScopeWorkspace || response.WorkspaceID != "ws-1" {
					t.Fatalf(
						"response meta = %#v, want workspace ws-1",
						response.SettingsGlobalWorkspaceCollectionResponseMetaPayload,
					)
				}
				if settingsService.LastListCollectionRequest.Collection != settingspkg.CollectionMCPServers ||
					settingsService.LastListCollectionRequest.Scope != settingspkg.ScopeWorkspace ||
					settingsService.LastListCollectionRequest.WorkspaceID != "ws-1" {
					t.Fatalf("LastListCollectionRequest = %#v", settingsService.LastListCollectionRequest)
				}
			},
		},
		{
			name:       "put scoped mcp server",
			method:     http.MethodPut,
			path:       "/api/settings/mcp-servers/server-a?scope=workspace&workspace_id=ws-1&target=sidecar",
			wantStatus: http.StatusOK,
			body: mustJSONBody(t, contract.PutSettingsMCPServerRequest{
				Server: contract.SettingsMCPServerPayload{Name: "server-a", Command: "mcpd"},
			}),
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				var response contract.SettingsGlobalWorkspaceCollectionMutationResult
				decodeJSONResponse(t, recorder, &response)
				if response.Scope != contract.SettingsWorkspaceScopeWorkspace || response.WorkspaceID != "ws-1" {
					t.Fatalf("response = %#v, want workspace mutation metadata", response)
				}
				if settingsService.LastPutCollectionRequest.Collection != settingspkg.CollectionMCPServers ||
					settingsService.LastPutCollectionRequest.Scope != settingspkg.ScopeWorkspace ||
					settingsService.LastPutCollectionRequest.WorkspaceID != "ws-1" ||
					settingsService.LastPutCollectionRequest.Target != settingspkg.TargetSidecar {
					t.Fatalf("LastPutCollectionRequest = %#v", settingsService.LastPutCollectionRequest)
				}
				if settingsService.LastPutCollectionRequest.MCPServer == nil ||
					settingsService.LastPutCollectionRequest.MCPServer.Command != "mcpd" {
					t.Fatalf(
						"LastPutCollectionRequest.MCPServer = %#v, want parsed server payload",
						settingsService.LastPutCollectionRequest.MCPServer,
					)
				}
			},
		},
		{
			name:       "delete scoped mcp server",
			method:     http.MethodDelete,
			path:       "/api/settings/mcp-servers/server-a?scope=workspace&workspace_id=ws-1&target=sidecar",
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				var response contract.SettingsGlobalWorkspaceCollectionMutationResult
				decodeJSONResponse(t, recorder, &response)
				if response.Scope != contract.SettingsWorkspaceScopeWorkspace || response.WorkspaceID != "ws-1" {
					t.Fatalf("response = %#v, want workspace mutation metadata", response)
				}
				if settingsService.LastDeleteCollectionRequest.Collection != settingspkg.CollectionMCPServers ||
					settingsService.LastDeleteCollectionRequest.Scope != settingspkg.ScopeWorkspace ||
					settingsService.LastDeleteCollectionRequest.WorkspaceID != "ws-1" ||
					settingsService.LastDeleteCollectionRequest.Target != settingspkg.TargetSidecar {
					t.Fatalf("LastDeleteCollectionRequest = %#v", settingsService.LastDeleteCollectionRequest)
				}
			},
		},
		{
			name:       "trigger restart action",
			method:     http.MethodPost,
			path:       "/api/settings/actions/restart",
			body:       []byte(`{}`),
			wantStatus: http.StatusAccepted,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				var response contract.RestartActionResponse
				decodeJSONResponse(t, recorder, &response)
				if response.OperationID != "op-123" || response.StatusURL != "/api/settings/actions/restart/op-123" {
					t.Fatalf("response = %#v, want restart payload shape", response)
				}
				if restartController.RequestRestartCalls != 1 {
					t.Fatalf("RequestRestartCalls = %d, want 1", restartController.RequestRestartCalls)
				}
			},
		},
		{
			name:       "poll restart action",
			method:     http.MethodGet,
			path:       "/api/settings/actions/restart/op-123",
			wantStatus: http.StatusOK,
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()

				var response contract.RestartActionStatus
				decodeJSONResponse(t, recorder, &response)
				if response.OperationID != "op-123" || response.Status != contract.RestartOperationReady {
					t.Fatalf("response = %#v, want persisted restart status payload", response)
				}
				if restartController.GetRestartOperationID != "op-123" {
					t.Fatalf("GetRestartOperationID = %q, want %q", restartController.GetRestartOperationID, "op-123")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recorder := performRequest(t, engine, tc.method, tc.path, tc.body)
			if recorder.Code != tc.wantStatus {
				t.Fatalf(
					"%s %s status = %d, want %d; body=%s",
					tc.method,
					tc.path,
					recorder.Code,
					tc.wantStatus,
					recorder.Body.String(),
				)
			}
			if tc.assert != nil {
				tc.assert(t, recorder)
			}
		})
	}
}

func TestRegisterNetworkRoutesMatchDocumentedHTTPAndUDSSurface(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))

	got := registeredNetworkRoutesFromEngine(engine.Routes())
	want := documentedNetworkRoutesForTransport(apispec.TransportUDS)
	if !slices.Equal(got, want) {
		t.Fatalf("network routes = %v, want documented %s routes %v", got, apispec.TransportUDS, want)
	}
}

func registeredNetworkRoutesFromEngine(routes gin.RoutesInfo) []string {
	filtered := make([]string, 0)
	for _, route := range routes {
		if strings.HasPrefix(route.Path, "/api/network") ||
			strings.HasPrefix(route.Path, "/api/workspaces/:workspace_id/network") {
			filtered = append(filtered, route.Method+" "+route.Path)
		}
	}
	sort.Strings(filtered)
	return filtered
}

func documentedNetworkRoutesForTransport(transport apispec.Transport) []string {
	routes := make([]string, 0)
	for _, operation := range apispec.Operations() {
		if !slices.Contains(operation.Transports, transport) {
			continue
		}
		if !strings.HasPrefix(operation.Path, "/api/network") &&
			!strings.HasPrefix(operation.Path, "/api/workspaces/{workspace_id}/network") {
			continue
		}
		routes = append(routes, operation.Method+" "+normalizeNetworkSpecRoutePath(operation.Path))
	}
	sort.Strings(routes)
	return routes
}

func normalizeNetworkSpecRoutePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") && len(part) > 2 {
			parts[i] = ":" + part[1:len(part)-1]
		}
	}
	return strings.Join(parts, "/")
}

func TestRegisterTaskRoutesUseSharedHandlerBindings(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))

	expectedHandlers := map[string]string{
		"GET /api/observe/tasks/dashboard":                             "TaskDashboard",
		"GET /api/observe/tasks/inbox":                                 "TaskInbox",
		"GET /api/task-runs/:id":                                       "GetTaskRun",
		"GET /api/task-runs/:id/reviews":                               "ListTaskRunReviews",
		"GET /api/task-reviews/:id":                                    "GetTaskRunReview",
		"GET /api/tasks/:id/execution-profile":                         "GetTaskExecutionProfile",
		"GET /api/tasks/:id/notifications/bridges":                     "ListTaskBridgeNotificationSubscriptions",
		"GET /api/tasks/:id/notifications/bridges/:subscription_id":    "GetTaskBridgeNotificationSubscription",
		"GET /api/tasks/:id/reviews":                                   "ListTaskReviews",
		"GET /api/tasks/:id/stream":                                    "StreamTask",
		"GET /api/tasks/:id/timeline":                                  "TaskTimeline",
		"GET /api/tasks/:id/tree":                                      "TaskTree",
		"GET /api/agent/channels":                                      "AgentChannels",
		"GET /api/agent/channels/:channel/recv":                        "AgentChannelRecv",
		"GET /api/agent/context":                                       "AgentContext",
		"GET /api/agent/coordinator/config":                            "AgentCoordinatorConfig",
		"GET /api/agent/me":                                            "AgentMe",
		"POST /api/agent/channels/:channel/send":                       "AgentChannelSend",
		"POST /api/agent/channels/reply":                               "AgentChannelReply",
		"POST /api/agent/tasks/:run_id/complete":                       "AgentTaskComplete",
		"POST /api/agent/tasks/:run_id/fail":                           "AgentTaskFail",
		"POST /api/agent/tasks/:run_id/heartbeat":                      "AgentTaskHeartbeat",
		"POST /api/agent/tasks/:run_id/release":                        "AgentTaskRelease",
		"POST /api/agent/tasks/claim-next":                             "AgentTaskClaimNext",
		"POST /api/agent/spawn":                                        "AgentSpawn",
		"DELETE /api/tasks/:id":                                        "DeleteTask",
		"DELETE /api/tasks/:id/notifications/bridges/:subscription_id": "DeleteTaskBridgeNotificationSubscription",
		"DELETE /api/tasks/:id/execution-profile":                      "DeleteTaskExecutionProfile",
		"POST /api/workspaces/:workspace_id/sessions/:session_id/stop": "StopSession",
		"POST /api/tasks/:id/approve":                                  "ApproveTask",
		"POST /api/tasks/:id/notifications/bridges":                    "CreateTaskBridgeNotificationSubscription",
		"POST /api/tasks/:id/publish":                                  "PublishTask",
		"POST /api/tasks/:id/reject":                                   "RejectTask",
		"POST /api/tasks/:id/start":                                    "StartTask",
		"POST /api/tasks/:id/triage/archive":                           "ArchiveTask",
		"POST /api/tasks/:id/triage/dismiss":                           "DismissTask",
		"POST /api/tasks/:id/triage/read":                              "MarkTaskRead",
		"POST /api/task-runs/:id/reviews":                              "RequestTaskRunReview",
		"POST /api/task-reviews/:id/verdict":                           "SubmitTaskRunReviewVerdict",
		"PUT /api/tasks/:id/execution-profile":                         "SetTaskExecutionProfile",
	}

	routes := engine.Routes()
	for key, handlerName := range expectedHandlers {
		var matched *gin.RouteInfo
		for i := range routes {
			route := routes[i]
			if route.Method+" "+route.Path == key {
				matched = &route
				break
			}
		}
		if matched == nil {
			t.Fatalf("route %q not registered", key)
			return
		}
		if !strings.Contains(matched.Handler, handlerName) {
			t.Fatalf("route %q handler = %q, want substring %q", key, matched.Handler, handlerName)
		}
	}
}

func TestAgentChannelRecvRejectsInvalidPathAndQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "Should reject malformed channel identifiers before reading inbox",
			path: "/api/agent/channels/bad.channel/recv",
		},
		{
			name: "Should reject malformed wait query values before reading inbox",
			path: "/api/agent/channels/builders/recv?wait=maybe",
		},
		{
			name: "Should reject malformed limit query values before reading inbox",
			path: "/api/agent/channels/builders/recv?limit=abc",
		},
		{
			name: "Should reject non-positive limit query values before reading inbox",
			path: "/api/agent/channels/builders/recv?limit=0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handlers := newAgentChannelHandlers(t, stubNetworkService{
				InboxFn: func(context.Context, string) ([]network.Envelope, error) {
					t.Fatal("Inbox should not be called for invalid receive requests")
					return nil, nil
				},
				WaitInboxFn: func(context.Context, string, string) ([]network.Envelope, error) {
					t.Fatal("WaitInbox should not be called for invalid receive requests")
					return nil, nil
				},
			})
			recorder := performAgentKernelRequest(
				t,
				newTestRouter(t, handlers),
				http.MethodGet,
				tt.path,
				nil,
				agentKernelHeaders(),
			)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
			var payload contract.ErrorPayload
			decodeJSONResponse(t, recorder, &payload)
			if payload.Error == "" {
				t.Fatalf("error payload = %#v, want validation error", payload)
			}
		})
	}
}
func TestCreateSessionHandlerReturnsSessionID(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			if opts.AgentName != "coder" || opts.Name != "demo" || opts.Workspace != "alpha" ||
				opts.WorkspacePath != "" ||
				opts.Channel != "builders" {
				t.Fatalf("Create() opts = %#v", opts)
			}
			sess := newSession("sess-123")
			sess.Channel = "builders"
			return sess, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(
		t,
		engine,
		http.MethodPost,
		"/api/sessions",
		[]byte(`{"agent_name":"coder","name":"demo","workspace":"alpha","channel":"builders"}`),
	)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		Session sessionPayload `json:"session"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Session.ID != "sess-123" {
		t.Fatalf("session.id = %q, want %q", response.Session.ID, "sess-123")
	}
	if response.Session.WorkspaceID != "ws-workspace" || response.Session.WorkspacePath != "/workspace" {
		t.Fatalf("session workspace = %#v", response.Session)
	}
	if response.Session.Channel != "builders" {
		t.Fatalf("session channel = %q, want %q", response.Session.Channel, "builders")
	}
}

func TestCreateSessionHandlerAllowsMissingAgent(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		CreateFn: func(_ context.Context, opts session.CreateOpts) (*session.Session, error) {
			if opts.AgentName != "" {
				t.Fatalf("Create() AgentName = %q, want empty", opts.AgentName)
			}
			if opts.WorkspacePath == "" || opts.Workspace != "" {
				t.Fatalf("Create() workspace opts = %#v", opts)
			}
			return newSession("sess-default"), nil
		},
	}
	engine := newTestRouter(t, newTestHandlers(t, manager, stubObserver{}, homePaths))

	recorder := performRequest(
		t,
		engine,
		http.MethodPost,
		"/api/sessions",
		[]byte(`{"name":"demo","workspace_path":"/workspace"}`),
	)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
}

func TestListSessionsHandlerReturnsAllSessions(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			return []*session.Info{newSessionInfo("sess-a"), newSessionInfo("sess-b")}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(response.Sessions))
	}
}

func TestListSessionsHandlerFiltersByWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	infoA := newSessionInfo("sess-a")
	infoB := newSessionInfo("sess-b")
	infoB.WorkspaceID = "ws-beta"
	infoB.Workspace = "/other"

	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			return []*session.Info{infoA, infoB}, nil
		},
	}
	workspaces := stubWorkspaceService{
		GetFn: func(_ context.Context, ref string) (workspacepkg.Workspace, error) {
			if ref != "alpha" {
				t.Fatalf("Get() ref = %q, want alpha", ref)
			}
			return workspacepkg.Workspace{ID: "ws-workspace", Name: "alpha"}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, manager, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/sessions?workspace=alpha", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Sessions []sessionPayload `json:"sessions"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Sessions) != 1 || response.Sessions[0].ID != "sess-a" {
		t.Fatalf("sessions = %#v", response.Sessions)
	}
	if response.Sessions[0].WorkspaceID != "ws-workspace" {
		t.Fatalf("workspace_id = %q, want ws-workspace", response.Sessions[0].WorkspaceID)
	}
}

func TestHookEventsHandlerAvailableOnUDSRouter(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	observer := stubObserver{
		QueryHookEventsFn: func(_ context.Context, filter hookspkg.EventFilter) ([]hookspkg.EventDescriptor, error) {
			if filter.Family != hookspkg.HookEventFamilyTool {
				t.Fatalf("filter.Family = %q, want %q", filter.Family, hookspkg.HookEventFamilyTool)
			}
			if !filter.SyncOnly {
				t.Fatal("filter.SyncOnly = false, want true")
			}
			return []hookspkg.EventDescriptor{{
				Event:         hookspkg.HookToolPreCall,
				Family:        hookspkg.HookEventFamilyTool,
				SyncEligible:  true,
				PayloadSchema: "ToolPreCallPayload",
				PatchSchema:   "ToolCallPatch",
			}}, nil
		},
	}
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, observer, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/hooks/events?family=tool&sync_only=true", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Events []contract.HookEventPayload `json:"events"`
	}
	decodeJSONResponse(t, recorder, &response)
	if got, want := len(response.Events), 1; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if response.Events[0].Event != hookspkg.HookToolPreCall.String() {
		t.Fatalf("events[0].Event = %q, want %q", response.Events[0].Event, hookspkg.HookToolPreCall)
	}
}

func TestCreateWorkspaceHandlerRegistersWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	addDir := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(addDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(addDir) error = %v", err)
	}

	workspaces := stubWorkspaceService{
		RegisterFn: func(_ context.Context, opts workspacepkg.RegisterOptions) (workspacepkg.Workspace, error) {
			if opts.RootDir != rootDir || opts.Name != "alpha" || len(opts.AdditionalDirs) != 1 ||
				opts.AdditionalDirs[0] != addDir ||
				opts.DefaultAgent != "coder" ||
				opts.SandboxRef != "daytona-dev" {
				t.Fatalf("Register() opts = %#v", opts)
			}
			return workspacepkg.Workspace{
				ID:             "ws_alpha123",
				RootDir:        rootDir,
				AdditionalDirs: []string{addDir},
				Name:           "alpha",
				DefaultAgent:   "coder",
				SandboxRef:     "daytona-dev",
				CreatedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				UpdatedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}, nil
		},
	}
	engine := newTestRouter(
		t,
		newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths),
	)

	body := mustJSONBody(t, map[string]any{
		"root_dir":      rootDir,
		"name":          "alpha",
		"add_dirs":      []string{addDir},
		"default_agent": "coder",
		"sandbox_ref":   "daytona-dev",
	})
	recorder := performRequest(t, engine, http.MethodPost, "/api/workspaces", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload `json:"workspace"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.ID != "ws_alpha123" {
		t.Fatalf("workspace.id = %q, want ws_alpha123", response.Workspace.ID)
	}
}

func TestListWorkspacesHandlerReturnsRows(t *testing.T) {
	homePaths := newTestHomePaths(t)
	workspaces := stubWorkspaceService{
		ListFn: func(context.Context) ([]workspacepkg.Workspace, error) {
			return []workspacepkg.Workspace{{
				ID:        "ws_alpha",
				RootDir:   "/workspace",
				Name:      "alpha",
				CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 4, 3, 12, 5, 0, 0, time.UTC),
			}}, nil
		},
	}
	engine := newTestRouter(
		t,
		newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths),
	)

	recorder := performRequest(t, engine, http.MethodGet, "/api/workspaces", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspaces []workspacePayload `json:"workspaces"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Workspaces) != 1 || response.Workspaces[0].ID != "ws_alpha" {
		t.Fatalf("workspaces = %#v", response.Workspaces)
	}
}

func TestGetWorkspaceHandlerReturnsDetail(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	sharedSkillDir := filepath.Join(rootDir, ".agh", "skills", "review")
	resolved := workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:        "ws_alpha",
			RootDir:   rootDir,
			Name:      "alpha",
			CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		WorkspaceID: "ws_alpha",
		Config: aghconfig.Config{
			Providers: map[string]aghconfig.ProviderConfig{
				"alpha": {Command: "alpha --acp"},
			},
		},
		Agents: []aghconfig.AgentDef{{
			Name:     "coder",
			Provider: "fake",
			Prompt:   "hello",
		}},
		Skills: []workspacepkg.SkillPath{{
			Dir:    sharedSkillDir,
			Source: "workspace",
		}},
	}
	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			info := newSessionInfo("sess-a")
			info.WorkspaceID = "ws_alpha"
			return []*session.Info{info}, nil
		},
	}
	workspaces := stubWorkspaceService{
		ResolveFn: func(_ context.Context, ref string) (workspacepkg.ResolvedWorkspace, error) {
			if ref != "ws_alpha" {
				t.Fatalf("Resolve() ref = %q, want ws_alpha", ref)
			}
			return resolved, nil
		},
	}
	engine := newTestRouter(t, newTestHandlersWithWorkspace(t, manager, stubObserver{}, workspaces, homePaths))

	recorder := performRequest(t, engine, http.MethodGet, "/api/workspaces/ws_alpha", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.WorkspaceDetailPayload
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.ID != "ws_alpha" || len(response.Sessions) != 1 || len(response.Agents) != 1 ||
		len(response.Skills) != 1 {
		t.Fatalf("workspace detail = %#v", response)
	}
	if response.Skills[0].Name != "review" {
		t.Fatalf("skill name = %q, want review", response.Skills[0].Name)
	}
	expectedProviders := core.SessionProviderOptionPayloadsFromConfig(&resolved.Config)
	if len(response.Providers) != len(expectedProviders) {
		t.Fatalf("len(providers) = %d, want %d (%#v)", len(response.Providers), len(expectedProviders), response)
	}
	for i, want := range expectedProviders {
		if got := response.Providers[i]; !reflect.DeepEqual(got, want) {
			t.Fatalf("providers[%d] = %#v, want %#v", i, got, want)
		}
	}
}

func TestUpdateWorkspaceHandlerUpdatesWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	addDir := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(addDir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll(addDir) error = %v", err)
	}

	var updated bool
	workspaces := stubWorkspaceService{
		GetFn: func(_ context.Context, _ string) (workspacepkg.Workspace, error) {
			if !updated {
				return workspacepkg.Workspace{ID: "ws_alpha", RootDir: rootDir, Name: "alpha"}, nil
			}
			return workspacepkg.Workspace{
				ID:             "ws_alpha",
				RootDir:        rootDir,
				Name:           "beta",
				AdditionalDirs: []string{addDir},
				DefaultAgent:   "reviewer",
				SandboxRef:     "local-dev",
			}, nil
		},
		UpdateFn: func(_ context.Context, id string, opts workspacepkg.UpdateOptions) error {
			if id != "ws_alpha" || opts.Name == nil || *opts.Name != "beta" || opts.AdditionalDirs == nil ||
				len(*opts.AdditionalDirs) != 1 ||
				(*opts.AdditionalDirs)[0] != addDir ||
				opts.DefaultAgent == nil ||
				*opts.DefaultAgent != "reviewer" ||
				opts.SandboxRef == nil ||
				*opts.SandboxRef != "local-dev" {
				t.Fatalf("Update() id=%q opts=%#v", id, opts)
			}
			updated = true
			return nil
		},
	}
	engine := newTestRouter(
		t,
		newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths),
	)

	body := mustJSONBody(t, map[string]any{
		"name":          "beta",
		"add_dirs":      []string{addDir},
		"default_agent": "reviewer",
		"sandbox_ref":   "local-dev",
	})
	recorder := performRequest(t, engine, http.MethodPatch, "/api/workspaces/ws_alpha", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload `json:"workspace"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.Name != "beta" || len(response.Workspace.AddDirs) != 1 {
		t.Fatalf("workspace = %#v", response.Workspace)
	}
}

func TestDeleteWorkspaceHandlerReturnsNoContent(t *testing.T) {
	homePaths := newTestHomePaths(t)
	workspaces := stubWorkspaceService{
		GetFn: func(context.Context, string) (workspacepkg.Workspace, error) {
			return workspacepkg.Workspace{ID: "ws_alpha", Name: "alpha"}, nil
		},
		UnregisterFn: func(_ context.Context, id string) error {
			if id != "ws_alpha" {
				t.Fatalf("Unregister() id = %q, want ws_alpha", id)
			}
			return nil
		},
	}
	engine := newTestRouter(
		t,
		newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths),
	)

	recorder := performRequest(t, engine, http.MethodDelete, "/api/workspaces/ws_alpha", nil)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusNoContent, recorder.Body.String())
	}
}

func TestResolveWorkspaceHandlerReturnsWorkspace(t *testing.T) {
	homePaths := newTestHomePaths(t)
	rootDir := t.TempDir()
	workspaces := stubWorkspaceService{
		ResolveOrRegisterFn: func(_ context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
			if path != rootDir {
				t.Fatalf("ResolveOrRegister() path = %q, want %q", path, rootDir)
			}
			return workspacepkg.ResolvedWorkspace{
				Workspace: workspacepkg.Workspace{
					ID:        "ws_alpha",
					RootDir:   rootDir,
					Name:      "alpha",
					CreatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				},
				WorkspaceID: "ws_alpha",
			}, nil
		},
	}
	engine := newTestRouter(
		t,
		newTestHandlersWithWorkspace(t, stubSessionManager{}, stubObserver{}, workspaces, homePaths),
	)

	recorder := performRequest(
		t,
		engine,
		http.MethodPost,
		"/api/workspaces/resolve",
		mustJSONBody(t, map[string]string{"path": rootDir}),
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Workspace workspacePayload `json:"workspace"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Workspace.ID != "ws_alpha" {
		t.Fatalf("workspace.id = %q, want ws_alpha", response.Workspace.ID)
	}
}

func TestDeleteSessionHandlerReturnsNoContent(t *testing.T) {
	t.Parallel()

	t.Run("ShouldReturnNoContent", func(t *testing.T) {
		t.Parallel()

		homePaths := newTestHomePaths(t)
		manager := stubSessionManager{
			DeleteFn: func(_ context.Context, id string) error {
				if id != "sess-123" {
					t.Fatalf("Delete() id = %q, want sess-123", id)
				}
				return nil
			},
		}
		handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
		engine := newTestRouter(t, handlers)

		recorder := performRequest(t, engine, http.MethodDelete, "/api/workspaces/ws-workspace/sessions/sess-123", nil)
		if recorder.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
		}
		if got := recorder.Body.String(); got != "" {
			t.Fatalf("body = %q, want empty", got)
		}
	})
}

func TestStopSessionHandlerReturnsStopped(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		StopFn: func(_ context.Context, id string) error {
			if id != "sess-123" {
				t.Fatalf("Stop() id = %q, want sess-123", id)
			}
			return nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/workspaces/ws-workspace/sessions/sess-123/stop", nil)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Body.String(); got != "" {
		t.Fatalf("body = %q, want empty", got)
	}
}

func TestAttachSessionHandlerReturnsSession(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		AttachSessionFn: func(_ context.Context, req store.SessionAttachRequest) (store.SessionAttach, error) {
			if req.SessionID != "sess-123" {
				t.Fatalf("AttachSession() session id = %q, want sess-123", req.SessionID)
			}
			now := time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC)
			return store.SessionAttach{
				SessionID:       req.SessionID,
				AttachedTo:      req.AttachedTo,
				AttachedAt:      now,
				AttachExpiresAt: now.Add(time.Minute),
			}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodPost, "/api/workspaces/ws-workspace/sessions/sess-123/attach", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestPromptSessionHandlerReturnsSSEStream(t *testing.T) {
	t.Run("ShouldReturnAnSSEStream", func(t *testing.T) {
		homePaths := newTestHomePaths(t)
		manager := stubSessionManager{
			PromptFn: func(context.Context, string, string) (<-chan acp.AgentEvent, error) {
				ch := make(chan acp.AgentEvent, 2)
				ch <- acp.AgentEvent{
					Type:      "agent_message",
					TurnID:    "turn-1",
					Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
					Text:      "hello",
				}
				ch <- acp.AgentEvent{
					Type:       "done",
					TurnID:     "turn-1",
					Timestamp:  time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
					StopReason: "end_turn",
				}
				close(ch)
				return ch, nil
			},
		}
		handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
		engine := newTestRouter(t, handlers)

		recorder := performRequest(
			t,
			engine,
			http.MethodPost,
			"/api/workspaces/ws-workspace/sessions/sess-123/prompt",
			[]byte(`{"message":"hello"}`),
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
			t.Fatalf("Content-Type = %q, want text/event-stream", got)
		}
		if got := recorder.Header().Get("x-vercel-ai-ui-message-stream"); got != "v1" {
			t.Fatalf("x-vercel-ai-ui-message-stream = %q, want v1", got)
		}

		records := parseSSE(t, recorder.Body.String())
		if len(records) < 5 {
			t.Fatalf("len(records) = %d, want at least 5; body=%s", len(records), recorder.Body.String())
		}
		if string(records[len(records)-1].Data) != "[DONE]" {
			t.Fatalf("last data = %q, want [DONE]", string(records[len(records)-1].Data))
		}

		partTypes := make([]string, 0, len(records))
		for _, record := range records[:len(records)-1] {
			if len(record.Data) == 0 || string(record.Data) == "[DONE]" {
				continue
			}
			var payload struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(record.Data, &payload); err != nil {
				t.Fatalf("json.Unmarshal(prompt part) error = %v; data=%s", err, string(record.Data))
			}
			partTypes = append(partTypes, payload.Type)
		}
		hasType := func(target string) bool {
			return slices.Contains(partTypes, target)
		}
		if !hasType("start") || !hasType("text-start") || !hasType("text-delta") ||
			!hasType("text-end") || !hasType("finish") {
			t.Fatalf("prompt part types = %#v", partTypes)
		}
	})
}

func TestPromptSessionHandlerReturnsRawSSEStreamWhenRequested(t *testing.T) {
	t.Run("ShouldReturnRawAgentEventsForBufferedCLIPath", func(t *testing.T) {
		homePaths := newTestHomePaths(t)
		manager := stubSessionManager{
			PromptFn: func(context.Context, string, string) (<-chan acp.AgentEvent, error) {
				ch := make(chan acp.AgentEvent, 2)
				ch <- acp.AgentEvent{
					Type:      "agent_message",
					TurnID:    "turn-1",
					Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
					Text:      "hello",
				}
				ch <- acp.AgentEvent{
					Type:       "done",
					TurnID:     "turn-1",
					Timestamp:  time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
					StopReason: "end_turn",
				}
				close(ch)
				return ch, nil
			},
		}
		handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
		engine := newTestRouter(t, handlers)

		recorder := performRequest(
			t,
			engine,
			http.MethodPost,
			"/api/workspaces/ws-workspace/sessions/sess-123/prompt?format=raw",
			[]byte(`{"message":"hello"}`),
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
		}

		records := parseSSE(t, recorder.Body.String())
		if len(records) != 2 {
			t.Fatalf("len(records) = %d, want 2; body=%s", len(records), recorder.Body.String())
		}
		if records[0].Event != "agent_message" || records[1].Event != "done" {
			t.Fatalf("events = [%s %s], want [agent_message done]", records[0].Event, records[1].Event)
		}
	})
}

func TestPromptSessionRawHandlerPreservesBusyInputMode(t *testing.T) {
	t.Run("ShouldForwardInterruptModeOnRawCLIPath", func(t *testing.T) {
		t.Parallel()

		homePaths := newTestHomePaths(t)
		var gotOpts session.SendPromptOpts
		manager := stubSessionManager{
			SendPromptFn: func(_ context.Context, id string, opts session.SendPromptOpts) (session.SendPromptResult, error) {
				if id != "sess-123" {
					t.Fatalf("SendPrompt() id = %q, want sess-123", id)
				}
				gotOpts = opts
				return session.SendPromptResult{
					Status:      "interrupted",
					Mode:        opts.Mode,
					Interrupted: true,
					NewTurnID:   "turn-2",
				}, nil
			},
		}
		handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
		engine := newTestRouter(t, handlers)

		recorder := performRequest(
			t,
			engine,
			http.MethodPost,
			"/api/workspaces/ws-workspace/sessions/sess-123/prompt?format=raw",
			[]byte("{\"message\":\"replace\",\"mode\":\"interrupt\"}"),
		)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
		}
		if gotOpts.Message != "replace" || gotOpts.Mode != session.BusyInputModeInterrupt {
			t.Fatalf("SendPrompt() opts = %#v, want interrupt replace", gotOpts)
		}
		var decoded contract.SendPromptResultResponse
		if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("json.Unmarshal(prompt result) error = %v; body=%s", err, recorder.Body.String())
		}
		if decoded.Prompt.Mode != contract.PromptModeInterrupt || !decoded.Prompt.Interrupted {
			t.Fatalf("decoded prompt = %#v, want interrupted mode", decoded.Prompt)
		}
	})
}

func TestPromptSessionHandlerDrainsPromptAfterRequestCancellation(t *testing.T) {
	t.Run("ShouldCancelTheDetachedPromptContextWhenTheRequestEnds", func(t *testing.T) {
		homePaths := newTestHomePaths(t)
		promptCtxCh := make(chan context.Context, 1)
		events := make(chan acp.AgentEvent)
		manager := stubSessionManager{
			PromptFn: func(ctx context.Context, _ string, _ string) (<-chan acp.AgentEvent, error) {
				promptCtxCh <- ctx
				return events, nil
			},
		}
		handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
		engine := newTestRouter(t, handlers)

		requestCtx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequestWithContext(
			requestCtx,
			http.MethodPost,
			"/api/workspaces/ws-workspace/sessions/sess-123/prompt",
			strings.NewReader(`{"message":"hello"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			engine.ServeHTTP(recorder, req)
			close(done)
		}()

		var promptCtx context.Context
		select {
		case promptCtx = <-promptCtxCh:
		case <-time.After(time.Second):
			t.Fatal("Prompt() was not invoked")
		}

		cancel()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("handler did not return after request cancellation")
		}

		if err := promptCtx.Err(); err != nil {
			t.Fatalf("prompt context err = %v, want nil while detached drain is active", err)
		}

		close(events)

		select {
		case <-promptCtx.Done():
		case <-time.After(time.Second):
			t.Fatal("prompt context was not canceled after detached drain completed")
		}

		if !errors.Is(promptCtx.Err(), context.Canceled) {
			t.Fatalf("prompt context err = %v, want context.Canceled after detached drain completed", promptCtx.Err())
		}
	})

	t.Run("ShouldCancelTheDetachedPromptContextWhenStreamShutsDown", func(t *testing.T) {
		homePaths := newTestHomePaths(t)
		promptCtxCh := make(chan context.Context, 1)
		events := make(chan acp.AgentEvent)
		manager := stubSessionManager{
			PromptFn: func(ctx context.Context, _ string, _ string) (<-chan acp.AgentEvent, error) {
				promptCtxCh <- ctx
				return events, nil
			},
		}
		handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
		streamDone := make(chan struct{})
		handlers.setStreamDone(streamDone)
		engine := newTestRouter(t, handlers)

		requestCtx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequestWithContext(
			requestCtx,
			http.MethodPost,
			"/api/workspaces/ws-workspace/sessions/sess-123/prompt",
			strings.NewReader(`{"message":"hello"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			engine.ServeHTTP(recorder, req)
			close(done)
		}()

		var promptCtx context.Context
		select {
		case promptCtx = <-promptCtxCh:
		case <-time.After(time.Second):
			t.Fatal("Prompt() was not invoked")
		}

		cancel()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("handler did not return after request cancellation")
		}

		if err := promptCtx.Err(); err != nil {
			t.Fatalf("prompt context err = %v, want nil before stream shutdown", err)
		}

		close(streamDone)

		select {
		case <-promptCtx.Done():
		case <-time.After(time.Second):
			t.Fatal("prompt context was not canceled after stream shutdown")
		}

		if !errors.Is(promptCtx.Err(), context.Canceled) {
			t.Fatalf("prompt context err = %v, want context.Canceled after stream shutdown", promptCtx.Err())
		}
	})
}

func TestCancelSessionPromptHandlerReturnsOK(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		CancelPromptFn: func(_ context.Context, id string) error {
			if id != "sess-123" {
				t.Fatalf("CancelPrompt() id = %q, want sess-123", id)
			}
			return nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(
		t,
		engine,
		http.MethodPost,
		"/api/workspaces/ws-workspace/sessions/sess-123/prompt/cancel",
		nil,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "" {
		t.Fatalf("body = %q, want empty", got)
	}
}

func TestPromptSessionHandlerRejectsEmptyMessage(t *testing.T) {
	homePaths := newTestHomePaths(t)
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(
		t,
		engine,
		http.MethodPost,
		"/api/workspaces/ws-workspace/sessions/sess-123/prompt",
		[]byte(`{"message":""}`),
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestSessionEventsHandlerReturnsFilteredEvents(t *testing.T) {
	homePaths := newTestHomePaths(t)
	var gotQuery store.EventQuery
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.Info, error) {
			return newSessionInfo("sess-123"), nil
		},
		EventsFn: func(_ context.Context, _ string, query store.EventQuery) ([]store.SessionEvent, error) {
			gotQuery = query
			return []store.SessionEvent{{
				ID:        "ev-1",
				SessionID: "sess-123",
				Sequence:  7,
				TurnID:    "turn-1",
				Type:      "agent_message",
				AgentName: "coder",
				Content:   `{"text":"hello"}`,
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(
		t,
		engine,
		http.MethodGet,
		"/api/workspaces/ws-workspace/sessions/sess-123/events?type=agent_message&agent_name=coder&turn_id=turn-1&after_sequence=5&limit=10&since=2026-04-03T12:00:00Z",
		nil,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if gotQuery.Type != "agent_message" || gotQuery.AgentName != "coder" || gotQuery.TurnID != "turn-1" ||
		gotQuery.AfterSequence != 5 ||
		gotQuery.Limit != 10 {
		t.Fatalf("query = %#v", gotQuery)
	}

	var response struct {
		Events []sessionEventPayload `json:"events"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Events) != 1 || response.Events[0].Sequence != 7 {
		t.Fatalf("events = %#v", response.Events)
	}
	if response.Events[0].WorkspaceID != "ws-workspace" || response.Events[0].WorkspacePath != "/workspace" {
		t.Fatalf("event workspace = %#v", response.Events[0])
	}
}

func TestSessionHistoryHandlerReturnsTurns(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.Info, error) {
			return newSessionInfo("sess-123"), nil
		},
		HistoryFn: func(context.Context, string, store.EventQuery) ([]store.TurnHistory, error) {
			return []store.TurnHistory{{
				TurnID: "turn-1",
				Events: []store.SessionEvent{{
					ID:        "ev-1",
					SessionID: "sess-123",
					Sequence:  1,
					TurnID:    "turn-1",
					Type:      "agent_message",
					AgentName: "coder",
					Content:   `{"text":"hello"}`,
					Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
				}},
			}}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/workspaces/ws-workspace/sessions/sess-123/history", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		History []turnHistoryPayload `json:"history"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.History) != 1 || response.History[0].TurnID != "turn-1" {
		t.Fatalf("history = %#v", response.History)
	}
	if got := response.History[0].Events[0]; got.WorkspaceID != "ws-workspace" || got.WorkspacePath != "/workspace" {
		t.Fatalf("history event workspace = %#v", got)
	}
}

func TestSessionTranscriptHandlerReturnsMessages(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		TranscriptFn: func(context.Context, string) ([]transcript.UIMessage, error) {
			return []transcript.UIMessage{{
				ID:   "msg-1",
				Role: transcript.UIRoleAssistant,
				Parts: []transcript.UIMessagePart{{
					Type:  "text",
					Text:  "hello",
					State: "done",
				}},
			}}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(
		t,
		engine,
		http.MethodGet,
		"/api/workspaces/ws-workspace/sessions/sess-123/transcript",
		nil,
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Messages []transcript.UIMessage `json:"messages"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(response.Messages))
	}
	if got := response.Messages[0].Parts[0].Text; got != "hello" {
		t.Fatalf("messages[0].Parts[0].Text = %q, want %q", got, "hello")
	}
}

func TestStreamSessionHandlerUsesLastEventID(t *testing.T) {
	homePaths := newTestHomePaths(t)
	var gotQuery store.EventQuery
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.Info, error) {
			return newSessionInfo("sess-123"), nil
		},
		EventsFn: func(_ context.Context, _ string, query store.EventQuery) ([]store.SessionEvent, error) {
			gotQuery = query
			return []store.SessionEvent{{
				ID:        "ev-2",
				SessionID: "sess-123",
				Sequence:  2,
				TurnID:    "turn-1",
				Type:      "done",
				AgentName: "coder",
				Content:   `{"stop_reason":"end_turn"}`,
				Timestamp: time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
			}}, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	done := make(chan struct{})
	close(done)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/workspaces/ws-workspace/sessions/sess-123/stream",
		http.NoBody,
	)
	req.Header.Set("Last-Event-ID", "1")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if gotQuery.AfterSequence != 1 {
		t.Fatalf("AfterSequence = %d, want 1", gotQuery.AfterSequence)
	}
	records := parseSSE(t, recorder.Body.String())
	if len(records) != 1 || records[0].ID != "2" || records[0].Event != "done" {
		t.Fatalf("records = %#v", records)
	}
	var payload sessionEventPayload
	decodeSSEData(t, records[0], &payload)
	if payload.WorkspaceID != "ws-workspace" || payload.WorkspacePath != "/workspace" {
		t.Fatalf("stream payload workspace = %#v", payload)
	}
}

func TestStreamSessionHandlerSyntheticStoppedEventIncludesWorkspaceContext(t *testing.T) {
	homePaths := newTestHomePaths(t)
	stoppedAt := time.Date(2026, 4, 3, 12, 0, 5, 0, time.UTC)
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.Info, error) {
			info := newSessionInfo("sess-123")
			info.State = session.StateStopped
			info.UpdatedAt = stoppedAt
			return info, nil
		},
		EventsFn: func(context.Context, string, store.EventQuery) ([]store.SessionEvent, error) {
			return nil, nil
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/workspaces/ws-workspace/sessions/sess-123/stream",
		http.NoBody,
	)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) != 1 || records[0].Event != session.EventTypeSessionStopped {
		t.Fatalf("records = %#v", records)
	}
	var payload sessionEventPayload
	decodeSSEData(t, records[0], &payload)
	if payload.WorkspaceID != "ws-workspace" || payload.WorkspacePath != "/workspace" {
		t.Fatalf("stopped payload workspace = %#v", payload)
	}
	if payload.Timestamp != stoppedAt {
		t.Fatalf("stopped payload timestamp = %v, want %v", payload.Timestamp, stoppedAt)
	}
}

func TestListAgentsHandlerReturnsAvailableAgents(t *testing.T) {
	homePaths := newTestHomePaths(t)
	writeAgentDef(t, homePaths, "coder")
	writeAgentDef(t, homePaths, "researcher")
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/agents", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Agents []agentPayload `json:"agents"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Agents) != 2 {
		t.Fatalf("len(agents) = %d, want 2", len(response.Agents))
	}
	if response.Agents[0].Name != "coder" {
		t.Fatalf("first agent = %q, want coder", response.Agents[0].Name)
	}
}

func TestGetAgentHandlerReturnsAgent(t *testing.T) {
	homePaths := newTestHomePaths(t)
	writeAgentDef(t, homePaths, "coder")
	handlers := newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/agents/coder", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Agent agentPayload `json:"agent"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Agent.Name != "coder" || response.Agent.Provider != "fake" {
		t.Fatalf("agent = %#v", response.Agent)
	}
}

func TestListLogsHandlerReturnsEvents(t *testing.T) {
	homePaths := newTestHomePaths(t)
	observer := stubObserver{
		QueryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
			return []store.EventSummary{{
				ID:        "sum-1",
				SessionID: "sess-123",
				Type:      "agent_message",
				AgentName: "coder",
				Summary:   "hello",
				Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/logs?workspace_id=ws-workspace", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Events []logEventPayload `json:"events"`
	}
	decodeJSONResponse(t, recorder, &response)
	if len(response.Events) != 1 || response.Events[0].ID != "sum-1" {
		t.Fatalf("events = %#v", response.Events)
	}
}

func TestExtensionStatusHandlerTrimsName(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlersWithExtensions(t, stubSessionManager{}, stubObserver{}, stubExtensionService{
		StatusFn: func(_ context.Context, name string) (contract.ExtensionPayload, error) {
			if name != "ext-a" {
				t.Fatalf("Status() name = %q, want %q", name, "ext-a")
			}
			return contract.ExtensionPayload{Name: name, Enabled: true, State: "active"}, nil
		},
	}, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/extensions/%20ext-a%20", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Extension contract.ExtensionPayload `json:"extension"`
	}
	decodeJSONResponse(t, recorder, &response)
	if response.Extension.Name != "ext-a" {
		t.Fatalf("extension.name = %q, want %q", response.Extension.Name, "ext-a")
	}
}

func TestExtensionStatusHandlerRejectsBlankName(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	handlers := newTestHandlersWithExtensions(
		t,
		stubSessionManager{},
		stubObserver{},
		stubExtensionService{},
		homePaths,
	)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/extensions/%20%20%20", nil)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestHealthHandlerReturnsMetrics(t *testing.T) {
	homePaths := newTestHomePaths(t)
	observer := stubObserver{
		HealthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{
				Status:         "ok",
				ActiveSessions: 3,
			}, nil
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/status", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response contract.StatusPayload
	decodeJSONResponse(t, recorder, &response)
	if response.Health.ActiveSessions != 3 {
		t.Fatalf("health.active_sessions = %d, want 3", response.Health.ActiveSessions)
	}
}

func TestDaemonStatusHandlerReturnsRunningState(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		ListAllFn: func(context.Context) ([]*session.Info, error) {
			return []*session.Info{newSessionInfo("sess-1")}, nil
		},
	}
	observer := stubObserver{
		HealthFn: func(context.Context) (observe.Health, error) {
			return observe.Health{Status: "ok", ActiveSessions: 1, Version: "dev"}, nil
		},
	}
	handlers := newTestHandlers(t, manager, observer, homePaths)
	engine := newTestRouter(t, handlers)

	recorder := performRequest(t, engine, http.MethodGet, "/api/status", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.StatusPayload
	decodeJSONResponse(t, recorder, &response)
	if response.Daemon.Status != "running" {
		t.Fatalf("daemon.status = %q, want running", response.Daemon.Status)
	}
	if response.Daemon.TotalSessions != 1 {
		t.Fatalf("daemon.total_sessions = %d, want 1", response.Daemon.TotalSessions)
	}
}

func TestHelperParsersAndPayloads(t *testing.T) {
	if _, err := parseOptionalTime("bad-time"); err == nil {
		t.Fatal("parseOptionalTime() error = nil, want non-nil")
	}
	if _, err := parseOptionalInt("bad"); err == nil {
		t.Fatal("parseOptionalInt() error = nil, want non-nil")
	}
	if _, err := parseOptionalInt64("bad"); err == nil {
		t.Fatal("parseOptionalInt64() error = nil, want non-nil")
	}
	if _, err := parseLogsCursor("bad"); err == nil {
		t.Fatal("parseLogsCursor() error = nil, want non-nil")
	}
	if got := string(payloadJSON("not-json")); got != `"not-json"` {
		t.Fatalf("payloadJSON(non-json) = %s, want %q", got, `"not-json"`)
	}
	if tokenUsagePayloadFromUsage(nil) != nil {
		t.Fatal("tokenUsagePayloadFromUsage(nil) = non-nil, want nil")
	}
}

func TestSessionErrorMappingUsesNotFoundAndConflict(t *testing.T) {
	homePaths := newTestHomePaths(t)
	manager := stubSessionManager{
		StatusFn: func(context.Context, string) (*session.Info, error) {
			return nil, session.ErrSessionNotFound
		},
		CreateFn: func(context.Context, session.CreateOpts) (*session.Session, error) {
			return nil, session.ErrPromptInProgress
		},
	}
	handlers := newTestHandlers(t, manager, stubObserver{}, homePaths)
	engine := newTestRouter(t, handlers)

	getResp := performRequest(t, engine, http.MethodGet, "/api/workspaces/ws-workspace/sessions/missing", nil)
	if getResp.Code != http.StatusNotFound {
		t.Fatalf("GET /api/workspaces/:workspace_id/sessions/:session_id status = %d, want 404", getResp.Code)
	}

	postResp := performRequest(
		t,
		engine,
		http.MethodPost,
		"/api/sessions",
		[]byte(`{"agent_name":"coder","workspace":"alpha"}`),
	)
	if postResp.Code != http.StatusConflict {
		t.Fatalf("POST /api/sessions status = %d, want 409", postResp.Code)
	}
}

func TestObserveEventStreamUsesLastEventIDCursor(t *testing.T) {
	homePaths := newTestHomePaths(t)
	timestamp := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	observer := stubObserver{
		QueryEventsFn: func(context.Context, store.EventSummaryQuery) ([]store.EventSummary, error) {
			return []store.EventSummary{
				{
					ID:        "sum-a",
					SessionID: "sess-1",
					Sequence:  1,
					Type:      "agent_message",
					AgentName: "coder",
					Timestamp: timestamp,
				},
				{ID: "sum-b", SessionID: "sess-1", Sequence: 2, Type: "done", AgentName: "coder", Timestamp: timestamp},
			}, nil
		},
	}
	handlers := newTestHandlers(t, stubSessionManager{}, observer, homePaths)
	done := make(chan struct{})
	close(done)
	handlers.setStreamDone(done)
	engine := newTestRouter(t, handlers)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		"/api/logs/stream?workspace_id=ws-workspace",
		http.NoBody,
	)
	req.Header.Set("Last-Event-ID", timestamp.Format(time.RFC3339Nano)+"|00000000000000000001")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	records := parseSSE(t, recorder.Body.String())
	if len(records) == 0 {
		t.Fatalf("expected at least one SSE record, got body=%s", recorder.Body.String())
	}
	if records[0].ID != timestamp.Format(time.RFC3339Nano)+"|00000000000000000002" {
		t.Fatalf("record id = %q, want %q", records[0].ID, timestamp.Format(time.RFC3339Nano)+"|00000000000000000002")
	}
}
