package core_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
)

func TestBridgeHandlersCreateListGetAndUpdate(t *testing.T) {
	t.Parallel()

	var createCalled, updateCalled bool
	_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
		CreateInstanceFn: func(_ context.Context, req bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
			createCalled = true
			if req.Scope != bridgepkg.ScopeGlobal || req.Platform != "telegram" || req.DisplayName != "Support" {
				t.Fatalf("CreateInstance() req = %#v", req)
			}
			return &bridgepkg.BridgeInstance{
				ID:            "brg-core",
				Scope:         req.Scope,
				Platform:      req.Platform,
				ExtensionName: req.ExtensionName,
				DisplayName:   req.DisplayName,
				Enabled:       req.Enabled,
				Status:        req.Status,
				RoutingPolicy: req.RoutingPolicy,
				CreatedAt:     time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
				UpdatedAt:     time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
			}, nil
		},
		ListInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
			return []bridgepkg.BridgeInstance{{
				ID:            "brg-core",
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			}}, nil
		},
		GetInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
			return &bridgepkg.BridgeInstance{
				ID:            id,
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			}, nil
		},
		UpdateInstanceFn: func(_ context.Context, req bridgepkg.UpdateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
			updateCalled = true
			if req.ID != "brg-core" || req.DisplayName == nil || *req.DisplayName != "Renamed" {
				t.Fatalf("UpdateInstance() req = %#v", req)
			}
			return &bridgepkg.BridgeInstance{
				ID:            req.ID,
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   *req.DisplayName,
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			}, nil
		},
	})

	createResp := performRequest(t, engine, http.MethodPost, "/bridges", []byte(`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`))
	if createResp.Code != http.StatusCreated || !createCalled {
		t.Fatalf("create status = %d createCalled=%v body=%s", createResp.Code, createCalled, createResp.Body.String())
	}

	listResp := performRequest(t, engine, http.MethodGet, "/bridges", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}
	var listPayload contract.BridgesResponse
	testutil.DecodeJSONResponse(t, listResp, &listPayload)
	if got, want := len(listPayload.Bridges), 1; got != want {
		t.Fatalf("len(bridges) = %d, want %d", got, want)
	}

	getResp := performRequest(t, engine, http.MethodGet, "/bridges/brg-core", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%s", getResp.Code, getResp.Body.String())
	}

	updateResp := performRequest(t, engine, http.MethodPatch, "/bridges/brg-core", []byte(`{"display_name":"Renamed"}`))
	if updateResp.Code != http.StatusOK || !updateCalled {
		t.Fatalf("update status = %d updateCalled=%v body=%s", updateResp.Code, updateCalled, updateResp.Body.String())
	}

}

func TestBridgeHandlersLifecycleTransitions(t *testing.T) {
	t.Parallel()

	_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
		StartInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
			return &bridgepkg.BridgeInstance{ID: id, Scope: bridgepkg.ScopeGlobal, Platform: "telegram", ExtensionName: "ext-telegram", DisplayName: "Support", Enabled: true, Status: bridgepkg.BridgeStatusStarting, RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true}}, nil
		},
		StopInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
			return &bridgepkg.BridgeInstance{ID: id, Scope: bridgepkg.ScopeGlobal, Platform: "telegram", ExtensionName: "ext-telegram", DisplayName: "Support", Enabled: false, Status: bridgepkg.BridgeStatusDisabled, RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true}}, nil
		},
		RestartInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
			return &bridgepkg.BridgeInstance{ID: id, Scope: bridgepkg.ScopeGlobal, Platform: "telegram", ExtensionName: "ext-telegram", DisplayName: "Support", Enabled: true, Status: bridgepkg.BridgeStatusStarting, RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true}}, nil
		},
	})

	for _, tc := range []struct {
		name   string
		path   string
		status bridgepkg.BridgeStatus
	}{
		{name: "Should enable bridge", path: "/bridges/brg-core/enable", status: bridgepkg.BridgeStatusStarting},
		{name: "Should disable bridge", path: "/bridges/brg-core/disable", status: bridgepkg.BridgeStatusDisabled},
		{name: "Should restart bridge", path: "/bridges/brg-core/restart", status: bridgepkg.BridgeStatusStarting},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp := performRequest(t, engine, http.MethodPost, tc.path, nil)
			if resp.Code != http.StatusOK {
				t.Fatalf("%s status = %d body=%s", tc.path, resp.Code, resp.Body.String())
			}
			var payload contract.BridgeResponse
			testutil.DecodeJSONResponse(t, resp, &payload)
			if payload.Bridge.Status != tc.status {
				t.Fatalf("%s status payload = %q, want %q", tc.path, payload.Bridge.Status, tc.status)
			}
		})
	}
}

func TestBridgeHandlersRoutesAndTestDelivery(t *testing.T) {
	t.Parallel()

	_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
		ListRoutesFn: func(_ context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeRoute, error) {
			return []bridgepkg.BridgeRoute{{
				RoutingKeyHash:   "hash-1",
				Scope:            bridgepkg.ScopeGlobal,
				BridgeInstanceID: bridgeInstanceID,
				PeerID:           "peer-1",
				ThreadID:         "thread-1",
				SessionID:        "sess-1",
				AgentName:        "coder",
				LastActivityAt:   time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
				CreatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
				UpdatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
		ResolveDeliveryTargetFn: func(_ context.Context, req bridgepkg.ResolveDeliveryTargetRequest) (*bridgepkg.DeliveryTarget, error) {
			return &bridgepkg.DeliveryTarget{
				BridgeInstanceID: req.BridgeInstanceID,
				PeerID:           "peer-default",
				ThreadID:         req.ThreadID,
				Mode:             bridgepkg.DeliveryModeReply,
			}, nil
		},
	})

	routesResp := performRequest(t, engine, http.MethodGet, "/bridges/brg-core/routes", nil)
	if routesResp.Code != http.StatusOK {
		t.Fatalf("routes status = %d body=%s", routesResp.Code, routesResp.Body.String())
	}
	var routes contract.BridgeRoutesResponse
	testutil.DecodeJSONResponse(t, routesResp, &routes)
	if got, want := len(routes.Routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}

	testResp := performRequest(t, engine, http.MethodPost, "/bridges/brg-core/test-delivery", []byte(`{"target":{"thread_id":"thread-1"}}`))
	if testResp.Code != http.StatusOK {
		t.Fatalf("test delivery status = %d body=%s", testResp.Code, testResp.Body.String())
	}
	var payload contract.BridgeTestDeliveryResponse
	testutil.DecodeJSONResponse(t, testResp, &payload)
	if payload.DeliveryTarget.BridgeInstanceID != "brg-core" || payload.DeliveryTarget.ThreadID != "thread-1" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestBridgeHandlersListProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		listProvidersFn func(context.Context) ([]bridgepkg.BridgeProvider, error)
		wantError       string
		wantHealth      string
		wantPlatform    string
		wantStatus      int
	}{
		{
			name: "Should list bridge providers",
			listProvidersFn: func(context.Context) ([]bridgepkg.BridgeProvider, error) {
				return []bridgepkg.BridgeProvider{{
					Platform:      "telegram",
					ExtensionName: "telegram-reference",
					DisplayName:   "Telegram",
					Description:   "Reference Telegram bridge adapter",
					Enabled:       true,
					State:         "active",
					Health:        "healthy",
					HealthMessage: "connected",
				}}, nil
			},
			wantHealth:   "healthy",
			wantPlatform: "telegram",
			wantStatus:   http.StatusOK,
		},
		{
			name: "Should map bridge provider errors through bridge status mapping",
			listProvidersFn: func(context.Context) ([]bridgepkg.BridgeProvider, error) {
				return nil, bridgepkg.ErrBridgeInstanceUnavailable
			},
			wantError:  bridgepkg.ErrBridgeInstanceUnavailable.Error(),
			wantStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
				ListProvidersFn: tt.listProvidersFn,
			})

			resp := performRequest(t, engine, http.MethodGet, "/bridges/providers", nil)
			if resp.Code != tt.wantStatus {
				t.Fatalf("providers status = %d, want %d body=%s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			if tt.wantError != "" {
				var payload contract.ErrorPayload
				testutil.DecodeJSONResponse(t, resp, &payload)
				if !strings.Contains(payload.Error, tt.wantError) {
					t.Fatalf("error payload = %#v, want %q", payload, tt.wantError)
				}
				return
			}

			var payload contract.BridgeProvidersResponse
			testutil.DecodeJSONResponse(t, resp, &payload)
			if got, want := len(payload.Providers), 1; got != want {
				t.Fatalf("len(providers) = %d, want %d", got, want)
			}
			if got, want := payload.Providers[0].Platform, tt.wantPlatform; got != want {
				t.Fatalf("provider platform = %q, want %q", got, want)
			}
			if got, want := payload.Providers[0].Health, tt.wantHealth; got != want {
				t.Fatalf("provider health = %q, want %q", got, want)
			}
		})
	}
}

func TestBridgeHandlersIncludeObservedHealthPayloads(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	bridge := bridgepkg.BridgeInstance{
		ID:            "brg-health",
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Support",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusReady,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	}

	handlers := core.NewBaseHandlers(core.BaseHandlerConfig{
		TransportName:                "api-core-test",
		MaskInternalErrors:           false,
		IncludeSessionWorkspaceInSSE: true,
		Sessions:                     testutil.StubSessionManager{},
		Observer: testutil.StubObserver{
			QueryBridgeHealthFn: func(context.Context) ([]observe.BridgeInstanceHealth, error) {
				return []observe.BridgeInstanceHealth{{
					BridgeInstanceID:      bridge.ID,
					Status:                bridgepkg.BridgeStatusDegraded,
					RouteCount:            2,
					DeliveryBacklog:       1,
					DeliveryFailuresTotal: 3,
					AuthFailuresTotal:     1,
					LastSuccessAt:         time.Date(2026, 4, 3, 11, 59, 0, 0, time.UTC),
					LastError:             "adapter unavailable",
				}}, nil
			},
		},
		Bridges: testutil.StubBridgeService{ListInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
			return []bridgepkg.BridgeInstance{bridge}, nil
		}, GetInstanceFn: func(context.Context, string) (*bridgepkg.BridgeInstance, error) { return &bridge, nil }},
		Workspaces: testutil.StubWorkspaceService{},
		HomePaths:  homePaths,
		Config:     cfg,
		Logger:     testutil.DiscardLogger(),
		StartedAt:  time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		Now: func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC)
		},
		HTTPPort: cfg.HTTP.Port,
	})

	engine := gin.New()
	engine.GET("/bridges", handlers.ListBridges)
	engine.GET("/bridges/:id", handlers.GetBridge)

	listResp := performRequest(t, engine, http.MethodGet, "/bridges", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}
	var listPayload contract.BridgesResponse
	testutil.DecodeJSONResponse(t, listResp, &listPayload)
	if got, want := listPayload.BridgeHealth[bridge.ID].DeliveryBacklog, 1; got != want {
		t.Fatalf("bridge_health backlog = %d, want %d", got, want)
	}

	getResp := performRequest(t, engine, http.MethodGet, "/bridges/"+bridge.ID, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%s", getResp.Code, getResp.Body.String())
	}
	var getPayload contract.BridgeResponse
	testutil.DecodeJSONResponse(t, getResp, &getPayload)
	if got, want := getPayload.Health.Status, bridgepkg.BridgeStatusDegraded; got != want {
		t.Fatalf("get health status = %q, want %q", got, want)
	}
	if got, want := getPayload.Health.RouteCount, 2; got != want {
		t.Fatalf("get health route_count = %d, want %d", got, want)
	}
	if getPayload.Health.LastSuccessAt == nil || !getPayload.Health.LastSuccessAt.Equal(time.Date(2026, 4, 3, 11, 59, 0, 0, time.UTC)) {
		t.Fatalf("get health last_success_at = %#v, want 2026-04-03T11:59:00Z", getPayload.Health.LastSuccessAt)
	}
}

func TestBridgeHandlersMutationReturnsBestEffortPayloadWhenHealthLookupFails(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	bridge := bridgepkg.BridgeInstance{
		ID:            "brg-core",
		Scope:         bridgepkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Support",
		Enabled:       true,
		Status:        bridgepkg.BridgeStatusStarting,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	}

	handlers := core.NewBaseHandlers(core.BaseHandlerConfig{
		TransportName:                "api-core-test",
		MaskInternalErrors:           false,
		IncludeSessionWorkspaceInSSE: true,
		Sessions:                     testutil.StubSessionManager{},
		Observer: testutil.StubObserver{
			QueryBridgeHealthFn: func(context.Context) ([]observe.BridgeInstanceHealth, error) {
				return nil, errors.New("observer unavailable")
			},
		},
		Bridges: testutil.StubBridgeService{
			CreateInstanceFn: func(context.Context, bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
				return &bridge, nil
			},
		},
		Workspaces: testutil.StubWorkspaceService{},
		HomePaths:  homePaths,
		Config:     cfg,
		Logger:     testutil.DiscardLogger(),
		StartedAt:  time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		Now: func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC)
		},
		HTTPPort: cfg.HTTP.Port,
	})

	engine := gin.New()
	engine.POST("/bridges", handlers.CreateBridge)

	resp := performRequest(t, engine, http.MethodPost, "/bridges", []byte(`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`))
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body=%s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var payload contract.BridgeResponse
	testutil.DecodeJSONResponse(t, resp, &payload)
	if payload.Bridge.ID != bridge.ID || payload.Bridge.Status != bridgepkg.BridgeStatusStarting {
		t.Fatalf("payload.Bridge = %#v, want created bridge payload", payload.Bridge)
	}
	if payload.Health.BridgeInstanceID != "" || payload.Health.Status != "" || payload.Health.RouteCount != 0 {
		t.Fatalf("payload.Health = %#v, want zero-value best-effort health payload", payload.Health)
	}
}

func TestBridgeHandlersReturnServiceUnavailableWhenNotConfigured(t *testing.T) {
	t.Parallel()

	_, engine := newBridgeHandlerFixture(t, nil)
	resp := performRequest(t, engine, http.MethodGet, "/bridges", nil)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusServiceUnavailable, resp.Body.String())
	}
}

func newBridgeHandlerFixture(t *testing.T, bridges core.BridgeService) (*core.BaseHandlers, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	handlers := core.NewBaseHandlers(core.BaseHandlerConfig{
		TransportName:                "api-core-test",
		MaskInternalErrors:           false,
		IncludeSessionWorkspaceInSSE: true,
		Sessions:                     testutil.StubSessionManager{},
		Observer:                     testutil.StubObserver{},
		Bridges:                      bridges,
		Workspaces:                   testutil.StubWorkspaceService{},
		HomePaths:                    homePaths,
		Config:                       cfg,
		Logger:                       testutil.DiscardLogger(),
		StartedAt:                    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		Now: func() time.Time {
			return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC)
		},
		HTTPPort: cfg.HTTP.Port,
	})

	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.GET("/bridges", handlers.ListBridges)
	engine.POST("/bridges", handlers.CreateBridge)
	engine.GET("/bridges/providers", handlers.ListBridgeProviders)
	engine.GET("/bridges/:id", handlers.GetBridge)
	engine.PATCH("/bridges/:id", handlers.UpdateBridge)
	engine.POST("/bridges/:id/enable", handlers.EnableBridge)
	engine.POST("/bridges/:id/disable", handlers.DisableBridge)
	engine.POST("/bridges/:id/restart", handlers.RestartBridge)
	engine.GET("/bridges/:id/routes", handlers.ListBridgeRoutes)
	engine.POST("/bridges/:id/test-delivery", handlers.TestBridgeDelivery)
	return handlers, engine
}
