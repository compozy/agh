package core_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/api/core"
	"github.com/pedronauck/agh/internal/api/testutil"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/observe"
)

func TestChannelHandlersCreateListGetAndUpdate(t *testing.T) {
	t.Parallel()

	var createCalled, updateCalled bool
	_, engine := newChannelHandlerFixture(t, testutil.StubChannelService{
		CreateInstanceFn: func(_ context.Context, req channelspkg.CreateInstanceRequest) (*channelspkg.ChannelInstance, error) {
			createCalled = true
			if req.Scope != channelspkg.ScopeGlobal || req.Platform != "telegram" || req.DisplayName != "Support" {
				t.Fatalf("CreateInstance() req = %#v", req)
			}
			return &channelspkg.ChannelInstance{
				ID:            "chan-core",
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
		ListInstancesFn: func(context.Context) ([]channelspkg.ChannelInstance, error) {
			return []channelspkg.ChannelInstance{{
				ID:            "chan-core",
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			}}, nil
		},
		GetInstanceFn: func(_ context.Context, id string) (*channelspkg.ChannelInstance, error) {
			return &channelspkg.ChannelInstance{
				ID:            id,
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			}, nil
		},
		UpdateInstanceFn: func(_ context.Context, req channelspkg.UpdateInstanceRequest) (*channelspkg.ChannelInstance, error) {
			updateCalled = true
			if req.ID != "chan-core" || req.DisplayName == nil || *req.DisplayName != "Renamed" {
				t.Fatalf("UpdateInstance() req = %#v", req)
			}
			return &channelspkg.ChannelInstance{
				ID:            req.ID,
				Scope:         channelspkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   *req.DisplayName,
				Enabled:       true,
				Status:        channelspkg.ChannelStatusReady,
				RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
			}, nil
		},
	})

	createResp := performRequest(t, engine, http.MethodPost, "/channels", []byte(`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`))
	if createResp.Code != http.StatusCreated || !createCalled {
		t.Fatalf("create status = %d createCalled=%v body=%s", createResp.Code, createCalled, createResp.Body.String())
	}

	listResp := performRequest(t, engine, http.MethodGet, "/channels", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}
	var listPayload contract.ChannelsResponse
	testutil.DecodeJSONResponse(t, listResp, &listPayload)
	if got, want := len(listPayload.Channels), 1; got != want {
		t.Fatalf("len(channels) = %d, want %d", got, want)
	}

	getResp := performRequest(t, engine, http.MethodGet, "/channels/chan-core", nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%s", getResp.Code, getResp.Body.String())
	}

	updateResp := performRequest(t, engine, http.MethodPatch, "/channels/chan-core", []byte(`{"display_name":"Renamed"}`))
	if updateResp.Code != http.StatusOK || !updateCalled {
		t.Fatalf("update status = %d updateCalled=%v body=%s", updateResp.Code, updateCalled, updateResp.Body.String())
	}

}

func TestChannelHandlersLifecycleTransitions(t *testing.T) {
	t.Parallel()

	_, engine := newChannelHandlerFixture(t, testutil.StubChannelService{
		StartInstanceFn: func(_ context.Context, id string) (*channelspkg.ChannelInstance, error) {
			return &channelspkg.ChannelInstance{ID: id, Scope: channelspkg.ScopeGlobal, Platform: "telegram", ExtensionName: "ext-telegram", DisplayName: "Support", Enabled: true, Status: channelspkg.ChannelStatusStarting, RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true}}, nil
		},
		StopInstanceFn: func(_ context.Context, id string) (*channelspkg.ChannelInstance, error) {
			return &channelspkg.ChannelInstance{ID: id, Scope: channelspkg.ScopeGlobal, Platform: "telegram", ExtensionName: "ext-telegram", DisplayName: "Support", Enabled: false, Status: channelspkg.ChannelStatusDisabled, RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true}}, nil
		},
		RestartInstanceFn: func(_ context.Context, id string) (*channelspkg.ChannelInstance, error) {
			return &channelspkg.ChannelInstance{ID: id, Scope: channelspkg.ScopeGlobal, Platform: "telegram", ExtensionName: "ext-telegram", DisplayName: "Support", Enabled: true, Status: channelspkg.ChannelStatusStarting, RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true}}, nil
		},
	})

	for _, tc := range []struct {
		name   string
		path   string
		status channelspkg.ChannelStatus
	}{
		{name: "Should enable channel", path: "/channels/chan-core/enable", status: channelspkg.ChannelStatusStarting},
		{name: "Should disable channel", path: "/channels/chan-core/disable", status: channelspkg.ChannelStatusDisabled},
		{name: "Should restart channel", path: "/channels/chan-core/restart", status: channelspkg.ChannelStatusStarting},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resp := performRequest(t, engine, http.MethodPost, tc.path, nil)
			if resp.Code != http.StatusOK {
				t.Fatalf("%s status = %d body=%s", tc.path, resp.Code, resp.Body.String())
			}
			var payload contract.ChannelResponse
			testutil.DecodeJSONResponse(t, resp, &payload)
			if payload.Channel.Status != tc.status {
				t.Fatalf("%s status payload = %q, want %q", tc.path, payload.Channel.Status, tc.status)
			}
		})
	}
}

func TestChannelHandlersRoutesAndTestDelivery(t *testing.T) {
	t.Parallel()

	_, engine := newChannelHandlerFixture(t, testutil.StubChannelService{
		ListRoutesFn: func(_ context.Context, channelInstanceID string) ([]channelspkg.ChannelRoute, error) {
			return []channelspkg.ChannelRoute{{
				RoutingKeyHash:    "hash-1",
				Scope:             channelspkg.ScopeGlobal,
				ChannelInstanceID: channelInstanceID,
				PeerID:            "peer-1",
				ThreadID:          "thread-1",
				SessionID:         "sess-1",
				AgentName:         "coder",
				LastActivityAt:    time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
				CreatedAt:         time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
				UpdatedAt:         time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
			}}, nil
		},
		ResolveDeliveryTargetFn: func(_ context.Context, req channelspkg.ResolveDeliveryTargetRequest) (*channelspkg.DeliveryTarget, error) {
			return &channelspkg.DeliveryTarget{
				ChannelInstanceID: req.ChannelInstanceID,
				PeerID:            "peer-default",
				ThreadID:          req.ThreadID,
				Mode:              channelspkg.DeliveryModeReply,
			}, nil
		},
	})

	routesResp := performRequest(t, engine, http.MethodGet, "/channels/chan-core/routes", nil)
	if routesResp.Code != http.StatusOK {
		t.Fatalf("routes status = %d body=%s", routesResp.Code, routesResp.Body.String())
	}
	var routes contract.ChannelRoutesResponse
	testutil.DecodeJSONResponse(t, routesResp, &routes)
	if got, want := len(routes.Routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}

	testResp := performRequest(t, engine, http.MethodPost, "/channels/chan-core/test-delivery", []byte(`{"target":{"thread_id":"thread-1"}}`))
	if testResp.Code != http.StatusOK {
		t.Fatalf("test delivery status = %d body=%s", testResp.Code, testResp.Body.String())
	}
	var payload contract.ChannelTestDeliveryResponse
	testutil.DecodeJSONResponse(t, testResp, &payload)
	if payload.DeliveryTarget.ChannelInstanceID != "chan-core" || payload.DeliveryTarget.ThreadID != "thread-1" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestChannelHandlersIncludeObservedHealthPayloads(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	channel := channelspkg.ChannelInstance{
		ID:            "chan-health",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Support",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	}

	handlers := core.NewBaseHandlers(core.BaseHandlerConfig{
		TransportName:                "api-core-test",
		MaskInternalErrors:           false,
		IncludeSessionWorkspaceInSSE: true,
		Sessions:                     testutil.StubSessionManager{},
		Observer: testutil.StubObserver{
			QueryChannelHealthFn: func(context.Context) ([]observe.ChannelInstanceHealth, error) {
				return []observe.ChannelInstanceHealth{{
					ChannelInstanceID:     channel.ID,
					Status:                channelspkg.ChannelStatusDegraded,
					RouteCount:            2,
					DeliveryBacklog:       1,
					DeliveryFailuresTotal: 3,
					AuthFailuresTotal:     1,
					LastError:             "adapter unavailable",
				}}, nil
			},
		},
		Channels: testutil.StubChannelService{ListInstancesFn: func(context.Context) ([]channelspkg.ChannelInstance, error) {
			return []channelspkg.ChannelInstance{channel}, nil
		}, GetInstanceFn: func(context.Context, string) (*channelspkg.ChannelInstance, error) { return &channel, nil }},
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
	engine.GET("/channels", handlers.ListChannels)
	engine.GET("/channels/:id", handlers.GetChannel)

	listResp := performRequest(t, engine, http.MethodGet, "/channels", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResp.Code, listResp.Body.String())
	}
	var listPayload contract.ChannelsResponse
	testutil.DecodeJSONResponse(t, listResp, &listPayload)
	if got, want := listPayload.ChannelHealth[channel.ID].DeliveryBacklog, 1; got != want {
		t.Fatalf("channel_health backlog = %d, want %d", got, want)
	}

	getResp := performRequest(t, engine, http.MethodGet, "/channels/"+channel.ID, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("get status = %d body=%s", getResp.Code, getResp.Body.String())
	}
	var getPayload contract.ChannelResponse
	testutil.DecodeJSONResponse(t, getResp, &getPayload)
	if got, want := getPayload.Health.Status, channelspkg.ChannelStatusDegraded; got != want {
		t.Fatalf("get health status = %q, want %q", got, want)
	}
	if got, want := getPayload.Health.RouteCount, 2; got != want {
		t.Fatalf("get health route_count = %d, want %d", got, want)
	}
}

func TestChannelHandlersReturnServiceUnavailableWhenNotConfigured(t *testing.T) {
	t.Parallel()

	_, engine := newChannelHandlerFixture(t, nil)
	resp := performRequest(t, engine, http.MethodGet, "/channels", nil)
	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusServiceUnavailable, resp.Body.String())
	}
}

func newChannelHandlerFixture(t *testing.T, channels core.ChannelService) (*core.BaseHandlers, *gin.Engine) {
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
		Channels:                     channels,
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
	engine.GET("/channels", handlers.ListChannels)
	engine.POST("/channels", handlers.CreateChannel)
	engine.GET("/channels/:id", handlers.GetChannel)
	engine.PATCH("/channels/:id", handlers.UpdateChannel)
	engine.POST("/channels/:id/enable", handlers.EnableChannel)
	engine.POST("/channels/:id/disable", handlers.DisableChannel)
	engine.POST("/channels/:id/restart", handlers.RestartChannel)
	engine.GET("/channels/:id/routes", handlers.ListChannelRoutes)
	engine.POST("/channels/:id/test-delivery", handlers.TestChannelDelivery)
	return handlers, engine
}
