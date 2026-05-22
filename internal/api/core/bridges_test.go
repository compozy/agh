package core_test

import (
	"context"
	"errors"
	"fmt"
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

	t.Run("Should create, list, get, and update bridge instances", func(t *testing.T) {
		t.Parallel()

		var createCalled, updateCalled bool
		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			CreateInstanceFn: func(_ context.Context, req bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
				createCalled = true
				if req.Scope != bridgepkg.ScopeGlobal || req.Platform != "telegram" || req.DisplayName != "Support" {
					t.Fatalf("CreateInstance() req = %#v", req)
				}
				if req.Status != bridgepkg.BridgeStatusStarting {
					t.Fatalf("CreateInstance().Status = %q, want %q", req.Status, bridgepkg.BridgeStatusStarting)
				}
				if req.DMPolicy != bridgepkg.BridgeDMPolicyPairing {
					t.Fatalf("CreateInstance().DMPolicy = %q, want %q", req.DMPolicy, bridgepkg.BridgeDMPolicyPairing)
				}
				if got, want := string(req.ProviderConfig), `{"mode":"bot","tenant":"acme"}`; got != want {
					t.Fatalf("CreateInstance().ProviderConfig = %s, want %s", got, want)
				}
				if got, want := string(req.DeliveryDefaults), `{"peer_id":"peer-default","mode":"reply"}`; got != want {
					t.Fatalf("CreateInstance().DeliveryDefaults = %s, want %s", got, want)
				}
				return &bridgepkg.BridgeInstance{
					ID:               "brg-core",
					Scope:            req.Scope,
					Platform:         req.Platform,
					ExtensionName:    req.ExtensionName,
					DisplayName:      req.DisplayName,
					Enabled:          req.Enabled,
					Status:           req.Status,
					DMPolicy:         req.DMPolicy,
					RoutingPolicy:    req.RoutingPolicy,
					ProviderConfig:   req.ProviderConfig,
					DeliveryDefaults: req.DeliveryDefaults,
					CreatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
					UpdatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
				}, nil
			},
			ListInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
				return []bridgepkg.BridgeInstance{{
					ID:               "brg-core",
					Scope:            bridgepkg.ScopeGlobal,
					Platform:         "telegram",
					ExtensionName:    "ext-telegram",
					DisplayName:      "Support",
					Enabled:          true,
					Status:           bridgepkg.BridgeStatusReady,
					DMPolicy:         bridgepkg.BridgeDMPolicyOpen,
					RoutingPolicy:    bridgepkg.RoutingPolicy{IncludePeer: true},
					ProviderConfig:   []byte(`{"mode":"bot","tenant":"acme"}`),
					DeliveryDefaults: []byte(`{"peer_id":"peer-default","mode":"reply"}`),
				}}, nil
			},
			GetInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
				return &bridgepkg.BridgeInstance{
					ID:               id,
					Scope:            bridgepkg.ScopeGlobal,
					Platform:         "telegram",
					ExtensionName:    "ext-telegram",
					DisplayName:      "Support",
					Enabled:          true,
					Status:           bridgepkg.BridgeStatusReady,
					DMPolicy:         bridgepkg.BridgeDMPolicyOpen,
					RoutingPolicy:    bridgepkg.RoutingPolicy{IncludePeer: true},
					ProviderConfig:   []byte(`{"mode":"bot","tenant":"acme"}`),
					DeliveryDefaults: []byte(`{"peer_id":"peer-default","mode":"reply"}`),
					Degradation: &bridgepkg.BridgeDegradation{
						Reason: bridgepkg.BridgeDegradationReasonProviderTimeout,
					},
				}, nil
			},
			UpdateInstanceFn: func(_ context.Context, req bridgepkg.UpdateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
				updateCalled = true
				if req.ID != "brg-core" || req.DisplayName == nil || *req.DisplayName != "Renamed" {
					t.Fatalf("UpdateInstance() req = %#v", req)
				}
				if req.DMPolicy == nil || *req.DMPolicy != bridgepkg.BridgeDMPolicyAllowlist {
					t.Fatalf("UpdateInstance().DMPolicy = %#v", req.DMPolicy)
				}
				if req.ProviderConfig == nil || string(*req.ProviderConfig) != `{"mode":"comments"}` {
					t.Fatalf("UpdateInstance().ProviderConfig = %#v", req.ProviderConfig)
				}
				if req.DeliveryDefaults == nil ||
					string(*req.DeliveryDefaults) != `{"group_id":"ops","mode":"direct-send"}` {
					t.Fatalf("UpdateInstance().DeliveryDefaults = %#v", req.DeliveryDefaults)
				}
				if req.Degradation == nil || req.Degradation.Reason != bridgepkg.BridgeDegradationReasonAuthFailed {
					t.Fatalf("UpdateInstance().Degradation = %#v", req.Degradation)
				}
				return &bridgepkg.BridgeInstance{
					ID:               req.ID,
					Scope:            bridgepkg.ScopeGlobal,
					Platform:         "telegram",
					ExtensionName:    "ext-telegram",
					DisplayName:      *req.DisplayName,
					Enabled:          true,
					Status:           bridgepkg.BridgeStatusReady,
					DMPolicy:         *req.DMPolicy,
					RoutingPolicy:    bridgepkg.RoutingPolicy{IncludePeer: true},
					ProviderConfig:   *req.ProviderConfig,
					DeliveryDefaults: *req.DeliveryDefaults,
					Degradation:      req.Degradation,
				}, nil
			},
		})

		createResp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/bridges",
			[]byte(
				`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"dm_policy":"pairing","routing_policy":{"include_peer":true},"provider_config":{"mode":"bot","tenant":"acme"},"delivery_defaults":{"peer_id":"peer-default","mode":"reply"}}`,
			),
		)
		if createResp.Code != http.StatusCreated || !createCalled {
			t.Fatalf(
				"create status = %d createCalled=%v body=%s",
				createResp.Code,
				createCalled,
				createResp.Body.String(),
			)
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
		if got, want := string(listPayload.Bridges[0].ProviderConfig), `{"mode":"bot","tenant":"acme"}`; got != want {
			t.Fatalf("list provider_config = %s, want %s", got, want)
		}
		if got, want := string(
			listPayload.Bridges[0].DeliveryDefaults,
		), `{"peer_id":"peer-default","mode":"reply"}`; got != want {
			t.Fatalf("list delivery_defaults = %s, want %s", got, want)
		}

		getResp := performRequest(t, engine, http.MethodGet, "/bridges/brg-core", nil)
		if getResp.Code != http.StatusOK {
			t.Fatalf("get status = %d body=%s", getResp.Code, getResp.Body.String())
		}
		var getPayload contract.BridgeResponse
		testutil.DecodeJSONResponse(t, getResp, &getPayload)
		if getPayload.Bridge.DMPolicy != bridgepkg.BridgeDMPolicyOpen {
			t.Fatalf("get bridge dm_policy = %q, want %q", getPayload.Bridge.DMPolicy, bridgepkg.BridgeDMPolicyOpen)
		}
		if getPayload.Bridge.Degradation == nil ||
			getPayload.Bridge.Degradation.Reason != bridgepkg.BridgeDegradationReasonProviderTimeout {
			t.Fatalf("get bridge degradation = %#v", getPayload.Bridge.Degradation)
		}

		updateResp := performRequest(
			t,
			engine,
			http.MethodPatch,
			"/bridges/brg-core",
			[]byte(
				`{"display_name":"Renamed","dm_policy":"allowlist","provider_config":{"mode":"comments"},"delivery_defaults":{"group_id":"ops","mode":"direct-send"},"degradation":{"reason":"auth_failed"}}`,
			),
		)
		if updateResp.Code != http.StatusOK || !updateCalled {
			t.Fatalf(
				"update status = %d updateCalled=%v body=%s",
				updateResp.Code,
				updateCalled,
				updateResp.Body.String(),
			)
		}
	})
}

func TestBridgeHandlersListFiltersActiveWorkspaceScope(t *testing.T) {
	t.Parallel()

	t.Run("Should return global and active-workspace bridges without leaking other workspaces", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			ListInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
				return []bridgepkg.BridgeInstance{
					{
						ID:            "brg-global",
						Scope:         bridgepkg.ScopeGlobal,
						Platform:      "telegram",
						ExtensionName: "ext-telegram",
						DisplayName:   "Global",
						Enabled:       true,
						Status:        bridgepkg.BridgeStatusReady,
						RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
					},
					{
						ID:            "brg-alpha",
						Scope:         bridgepkg.ScopeWorkspace,
						WorkspaceID:   "ws-alpha",
						Platform:      "telegram",
						ExtensionName: "ext-telegram",
						DisplayName:   "Alpha",
						Enabled:       true,
						Status:        bridgepkg.BridgeStatusReady,
						RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
					},
					{
						ID:            "brg-beta",
						Scope:         bridgepkg.ScopeWorkspace,
						WorkspaceID:   "ws-beta",
						Platform:      "telegram",
						ExtensionName: "ext-telegram",
						DisplayName:   "Beta",
						Enabled:       true,
						Status:        bridgepkg.BridgeStatusReady,
						RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
					},
				}, nil
			},
		})

		resp := performRequest(t, engine, http.MethodGet, "/bridges?scope=all&workspace_id=ws-alpha", nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("list status = %d body=%s", resp.Code, resp.Body.String())
		}
		var payload contract.BridgesResponse
		testutil.DecodeJSONResponse(t, resp, &payload)
		if got, want := len(payload.Bridges), 2; got != want {
			t.Fatalf("len(bridges) = %d, want %d: %#v", got, want, payload.Bridges)
		}
		gotIDs := []string{payload.Bridges[0].ID, payload.Bridges[1].ID}
		wantIDs := []string{"brg-global", "brg-alpha"}
		for index, want := range wantIDs {
			if gotIDs[index] != want {
				t.Fatalf("bridge ids = %#v, want %#v", gotIDs, wantIDs)
			}
		}
	})
}

func TestBridgeHandlersHealthStreamFiltersActiveWorkspaceScope(t *testing.T) {
	t.Parallel()

	t.Run("Should stream only global and active-workspace bridge health", func(t *testing.T) {
		t.Parallel()

		gin.SetMode(gin.TestMode)
		streamDone := make(chan struct{})
		close(streamDone)
		homePaths := testutil.NewTestHomePaths(t)
		cfg := aghconfig.DefaultWithHome(homePaths)
		cfg.HTTP.Host = "127.0.0.1"
		cfg.HTTP.Port = 2123
		instances := []bridgepkg.BridgeInstance{
			{
				ID:            "brg-global",
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Global",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			},
			{
				ID:            "brg-alpha",
				Scope:         bridgepkg.ScopeWorkspace,
				WorkspaceID:   "ws-alpha",
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Alpha",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			},
			{
				ID:            "brg-beta",
				Scope:         bridgepkg.ScopeWorkspace,
				WorkspaceID:   "ws-beta",
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Beta",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusReady,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			},
		}
		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
			TransportName:                "api-core-test",
			MaskInternalErrors:           false,
			IncludeSessionWorkspaceInSSE: true,
			Sessions:                     testutil.StubSessionManager{},
			Observer: testutil.StubObserver{
				QueryBridgeHealthFn: func(context.Context) ([]observe.BridgeInstanceHealth, error) {
					return []observe.BridgeInstanceHealth{
						{
							BridgeInstanceID: "brg-global",
							Status:           bridgepkg.BridgeStatusReady,
							RouteCount:       1,
						},
						{
							BridgeInstanceID: "brg-alpha",
							Status:           bridgepkg.BridgeStatusReady,
							RouteCount:       2,
						},
						{
							BridgeInstanceID: "brg-beta",
							Status:           bridgepkg.BridgeStatusReady,
							RouteCount:       3,
						},
					}, nil
				},
			},
			Bridges: testutil.StubBridgeService{
				ListInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
					return instances, nil
				},
			},
			Workspaces:   testutil.StubWorkspaceService{},
			HomePaths:    homePaths,
			Config:       cfg,
			Logger:       testutil.DiscardLogger(),
			StartedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			Now:          func() time.Time { return time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC) },
			PollInterval: time.Millisecond,
			StreamDone:   streamDone,
			HTTPPort:     cfg.HTTP.Port,
		})
		engine := gin.New()
		engine.GET("/bridges/health/stream", handlers.StreamBridgeHealth)

		resp := performRequest(
			t,
			engine,
			http.MethodGet,
			"/bridges/health/stream?scope=all&workspace_id=ws-alpha",
			nil,
		)
		if resp.Code != http.StatusOK {
			t.Fatalf("stream status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
		}
		records := testutil.ParseSSE(t, resp.Body.String())
		if got, want := len(records), 1; got != want {
			t.Fatalf("stream records = %d, want %d; body=%s", got, want, resp.Body.String())
		}
		var payload contract.BridgeHealthStreamPayload
		testutil.DecodeSSEData(t, records[0], &payload)
		if _, ok := payload.BridgeHealth["brg-global"]; !ok {
			t.Fatalf("stream bridge_health missing global bridge: %#v", payload.BridgeHealth)
		}
		if _, ok := payload.BridgeHealth["brg-alpha"]; !ok {
			t.Fatalf("stream bridge_health missing active workspace bridge: %#v", payload.BridgeHealth)
		}
		if _, ok := payload.BridgeHealth["brg-beta"]; ok {
			t.Fatalf("stream bridge_health leaked inactive workspace bridge: %#v", payload.BridgeHealth)
		}
	})
}

func TestBridgeHandlersLifecycleTransitions(t *testing.T) {
	t.Parallel()

	_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
		StartInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
			return &bridgepkg.BridgeInstance{
				ID:            id,
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusStarting,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			}, nil
		},
		StopInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
			return &bridgepkg.BridgeInstance{
				ID:            id,
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       false,
				Status:        bridgepkg.BridgeStatusDisabled,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			}, nil
		},
		RestartInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
			return &bridgepkg.BridgeInstance{
				ID:            id,
				Scope:         bridgepkg.ScopeGlobal,
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        bridgepkg.BridgeStatusStarting,
				RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			}, nil
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

	t.Run("Should list routes and resolve test delivery targets", func(t *testing.T) {
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

		testResp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/bridges/brg-core/test-delivery",
			[]byte(`{"target":{"thread_id":"thread-1"}}`),
		)
		if testResp.Code != http.StatusOK {
			t.Fatalf("test delivery status = %d body=%s", testResp.Code, testResp.Body.String())
		}
		var payload contract.BridgeTestDeliveryResponse
		testutil.DecodeJSONResponse(t, testResp, &payload)
		if payload.DeliveryTarget.BridgeInstanceID != "brg-core" || payload.DeliveryTarget.ThreadID != "thread-1" {
			t.Fatalf("payload = %#v", payload)
		}
	})
}

func TestBridgeHandlersTargetDirectory(t *testing.T) {
	t.Run("Should list targets and return structured ambiguity diagnostics", func(t *testing.T) {
		t.Parallel()

		var listQuery bridgepkg.BridgeTargetQuery
		target := bridgepkg.BridgeTarget{
			BridgeID:       "brg-core",
			CanonicalRoute: "slack:channel:C123",
			DisplayName:    "#ops",
			Normalized:     "ops",
			TargetType:     bridgepkg.BridgeTargetTypeChannel,
			Qualifier:      "slack",
			UpdatedAt:      time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			LastSeenAt:     time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		}
		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			ListBridgeTargetsFn: func(_ context.Context, query bridgepkg.BridgeTargetQuery) (bridgepkg.BridgeTargetsResult, error) {
				listQuery = query
				return bridgepkg.BridgeTargetsResult{
					BridgeID:    query.BridgeID,
					Items:       []bridgepkg.BridgeTarget{target},
					Total:       1,
					GeneratedAt: time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
				}, nil
			},
			ResolveBridgeTargetFn: func(
				_ context.Context,
				bridgeID string,
				query string,
			) (bridgepkg.ResolveBridgeTargetResult, error) {
				return bridgepkg.ResolveBridgeTargetResult{
					Step:       4,
					Ambiguous:  true,
					Candidates: []bridgepkg.BridgeTarget{target, target},
				}, fmt.Errorf("bridge %q query %q: %w", bridgeID, query, bridgepkg.ErrBridgeTargetAmbiguous)
			},
		})

		listResp := performRequest(t, engine, http.MethodGet, "/bridges/brg-core/targets?q=ops&limit=5", nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf("target list status = %d body=%s", listResp.Code, listResp.Body.String())
		}
		if listQuery.BridgeID != "brg-core" || listQuery.Query != "ops" || listQuery.Limit != 5 {
			t.Fatalf("ListBridgeTargets() query = %#v", listQuery)
		}
		var listPayload contract.BridgeTargetsResponse
		testutil.DecodeJSONResponse(t, listResp, &listPayload)
		if got, want := len(listPayload.Targets), 1; got != want {
			t.Fatalf("len(targets) = %d, want %d", got, want)
		}

		resolveResp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/bridges/brg-core/resolve",
			[]byte("{\"name\":\"ops\"}"),
		)
		if resolveResp.Code != http.StatusUnprocessableEntity {
			t.Fatalf("target resolve status = %d body=%s", resolveResp.Code, resolveResp.Body.String())
		}
		var resolvePayload contract.BridgeResolveTargetResponse
		testutil.DecodeJSONResponse(t, resolveResp, &resolvePayload)
		if resolvePayload.Diagnostic == nil || resolvePayload.Diagnostic.Code != contract.CodeTargetAmbiguous {
			t.Fatalf("resolve diagnostic = %#v, want target_ambiguous", resolvePayload.Diagnostic)
		}
		if got, want := len(resolvePayload.Result.Candidates), 2; got != want {
			t.Fatalf("len(resolve candidates) = %d, want %d", got, want)
		}
	})

	t.Run("Should reject blank target names before resolver dispatch", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			ResolveBridgeTargetFn: func(context.Context, string, string) (bridgepkg.ResolveBridgeTargetResult, error) {
				t.Fatal("ResolveBridgeTarget() should not be called for blank target names")
				return bridgepkg.ResolveBridgeTargetResult{}, nil
			},
		})

		response := performRequest(
			t,
			engine,
			http.MethodPost,
			"/bridges/brg-core/resolve",
			[]byte(`{"name":"   "}`),
		)
		if response.Code != http.StatusBadRequest {
			t.Fatalf(
				"target resolve status = %d body=%s, want %d",
				response.Code,
				response.Body.String(),
				http.StatusBadRequest,
			)
		}
		if !strings.Contains(response.Body.String(), "bridge target name is required") {
			t.Fatalf("target resolve body = %q, want blank-name error", response.Body.String())
		}
	})
}

func TestBridgeHandlersSecretBindingsCRUD(t *testing.T) {
	t.Parallel()

	t.Run("Should manage bridge secret bindings without leaking write-only values", func(t *testing.T) {
		t.Parallel()

		var (
			putBinding       bridgepkg.BridgeSecretBinding
			putSecretValue   *string
			deleteInstanceID string
			deleteName       string
		)

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			ListSecretBindingsFn: func(_ context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeSecretBinding, error) {
				if bridgeInstanceID != "brg-core" {
					t.Fatalf("ListSecretBindings() bridgeInstanceID = %q, want brg-core", bridgeInstanceID)
				}
				return []bridgepkg.BridgeSecretBinding{{
					BridgeInstanceID: bridgeInstanceID,
					BindingName:      "bot_token",
					SecretRef:        "vault:bridges/brg-core/bot_token",
					Kind:             "token",
				}}, nil
			},
			PutSecretBindingFn: func(_ context.Context, binding bridgepkg.BridgeSecretBinding, secretValue *string) error {
				putBinding = binding
				putSecretValue = secretValue
				return nil
			},
			DeleteSecretBindingFn: func(_ context.Context, bridgeInstanceID string, bindingName string) error {
				deleteInstanceID = bridgeInstanceID
				deleteName = bindingName
				return nil
			},
		})

		listResp := performRequest(t, engine, http.MethodGet, "/bridges/brg-core/secret-bindings", nil)
		if listResp.Code != http.StatusOK {
			t.Fatalf("list secret bindings status = %d body=%s", listResp.Code, listResp.Body.String())
		}
		var listPayload contract.BridgeSecretBindingsResponse
		testutil.DecodeJSONResponse(t, listResp, &listPayload)
		if got, want := len(listPayload.Bindings), 1; got != want {
			t.Fatalf("len(bindings) = %d, want %d", got, want)
		}
		if listPayload.Bindings[0].BindingName != "bot_token" ||
			listPayload.Bindings[0].SecretRef != "vault:bridges/brg-core/bot_token" ||
			listPayload.Bindings[0].Kind != "token" {
			t.Fatalf("binding = %#v", listPayload.Bindings[0])
		}
		if strings.Contains(listResp.Body.String(), "secret_value") {
			t.Fatalf("list response leaked write-only secret field: %s", listResp.Body.String())
		}

		putResp := performRequest(
			t,
			engine,
			http.MethodPut,
			"/bridges/brg-core/secret-bindings/bot_token",
			[]byte(`{"secret_ref":"vault:bridges/brg-core/bot_token","kind":"token","secret_value":"telegram-token"}`),
		)
		if putResp.Code != http.StatusOK {
			t.Fatalf("put secret binding status = %d body=%s", putResp.Code, putResp.Body.String())
		}
		if putBinding.BridgeInstanceID != "brg-core" || putBinding.BindingName != "bot_token" ||
			putBinding.SecretRef != "vault:bridges/brg-core/bot_token" ||
			putBinding.Kind != "token" {
			t.Fatalf("put binding = %#v", putBinding)
		}
		if putSecretValue == nil || *putSecretValue != "telegram-token" {
			t.Fatalf("put secret value = %v, want write-only value", putSecretValue)
		}
		if strings.Contains(putResp.Body.String(), "telegram-token") {
			t.Fatalf("put response leaked secret_value: %s", putResp.Body.String())
		}

		deleteResp := performRequest(t, engine, http.MethodDelete, "/bridges/brg-core/secret-bindings/bot_token", nil)
		if deleteResp.Code != http.StatusNoContent {
			t.Fatalf("delete secret binding status = %d body=%s", deleteResp.Code, deleteResp.Body.String())
		}
		if deleteInstanceID != "brg-core" || deleteName != "bot_token" {
			t.Fatalf("delete args = %q/%q, want brg-core/bot_token", deleteInstanceID, deleteName)
		}
	})
}

func TestBridgeHandlersLifecycleAndSecretBindingErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("Should lifecycle transition maps bridge errors", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			StartInstanceFn: func(context.Context, string) (*bridgepkg.BridgeInstance, error) {
				return nil, bridgepkg.ErrBridgeInstanceNotFound
			},
		})

		resp := performRequest(t, engine, http.MethodPost, "/bridges/brg-core/enable", nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("enable status = %d, want %d body=%s", resp.Code, http.StatusNotFound, resp.Body.String())
		}
	})

	t.Run("Should service unavailable covers lifecycle and secret bindings", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, nil)
		tests := []struct {
			method string
			path   string
			body   []byte
		}{
			{
				method: http.MethodPost,
				path:   "/bridges",
				body: []byte(
					`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"routing_policy":{"include_peer":true}}`,
				),
			},
			{method: http.MethodGet, path: "/bridges/brg-core"},
			{method: http.MethodPatch, path: "/bridges/brg-core", body: []byte(`{"display_name":"Renamed"}`)},
			{method: http.MethodGet, path: "/bridges/brg-core/routes"},
			{
				method: http.MethodPost,
				path:   "/bridges/brg-core/test-delivery",
				body:   []byte(`{"target":{"peer_id":"peer-1"}}`),
			},
			{method: http.MethodPost, path: "/bridges/brg-core/enable"},
			{method: http.MethodPost, path: "/bridges/brg-core/disable"},
			{method: http.MethodPost, path: "/bridges/brg-core/restart"},
			{method: http.MethodGet, path: "/bridges/brg-core/secret-bindings"},
			{
				method: http.MethodPut,
				path:   "/bridges/brg-core/secret-bindings/bot_token",
				body:   []byte(`{"secret_ref":"env:TG_TOKEN","kind":"env"}`),
			},
			{method: http.MethodDelete, path: "/bridges/brg-core/secret-bindings/bot_token"},
		}

		for _, tc := range tests {
			resp := performRequest(t, engine, tc.method, tc.path, tc.body)
			if resp.Code != http.StatusServiceUnavailable {
				t.Fatalf(
					"%s %s status = %d, want %d body=%s",
					tc.method,
					tc.path,
					resp.Code,
					http.StatusServiceUnavailable,
					resp.Body.String(),
				)
			}
		}
	})

	t.Run("Should invalid secret binding payload is rejected before service call", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			PutSecretBindingFn: func(context.Context, bridgepkg.BridgeSecretBinding, *string) error {
				t.Fatal("PutSecretBinding() should not be called for invalid payload")
				return nil
			},
		})

		resp := performRequest(
			t,
			engine,
			http.MethodPut,
			"/bridges/brg-core/secret-bindings/bot_token",
			[]byte(`{"secret_ref":"env:TG_TOKEN","kind":7}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"put invalid secret binding status = %d, want %d body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should invalid bridge secret binding maps to bad request", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			PutSecretBindingFn: func(context.Context, bridgepkg.BridgeSecretBinding, *string) error {
				return fmt.Errorf(
					"%w: bridge secret refs must use vault:bridges/<path>",
					bridgepkg.ErrInvalidBridgeSecretBinding,
				)
			},
		})

		resp := performRequest(
			t,
			engine,
			http.MethodPut,
			"/bridges/brg-core/secret-bindings/bot_token",
			[]byte(`{"secret_ref":"env:TG_TOKEN","kind":"token"}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"put invalid secret binding service error status = %d, want %d body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})
}

func TestBridgeHandlersRequestDecodeAndServiceErrorPaths(t *testing.T) {
	t.Parallel()

	t.Run("Should create rejects malformed json", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			CreateInstanceFn: func(context.Context, bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
				t.Fatal("CreateInstance() should not be called for malformed JSON")
				return nil, nil
			},
		})

		resp := performRequest(t, engine, http.MethodPost, "/bridges", []byte(`{"scope":"global"`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"create malformed json status = %d, want %d body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should reject client-owned operational state fields", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			CreateInstanceFn: func(context.Context, bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
				t.Fatal("CreateInstance() should not be called when create payload includes operational state")
				return nil, nil
			},
		})

		tests := []struct {
			name      string
			body      []byte
			wantField string
		}{
			{
				name:      "Should reject status",
				wantField: "status",
				body: []byte(
					`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"ready","routing_policy":{"include_peer":true}}`,
				),
			},
			{
				name:      "Should reject degradation",
				wantField: "degradation",
				body: []byte(
					`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"degradation":{"reason":"rate_limited"},"routing_policy":{"include_peer":true}}`,
				),
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				resp := performRequest(t, engine, http.MethodPost, "/bridges", tt.body)
				if resp.Code != http.StatusBadRequest {
					t.Fatalf(
						"create status = %d, want %d body=%s",
						resp.Code,
						http.StatusBadRequest,
						resp.Body.String(),
					)
				}
				var payload contract.ErrorPayload
				testutil.DecodeJSONResponse(t, resp, &payload)
				if !strings.Contains(payload.Error, tt.wantField) {
					t.Fatalf("error payload = %#v, want detail mentioning %q", payload, tt.wantField)
				}
			})
		}
	})

	t.Run("Should create maps service errors", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			CreateInstanceFn: func(context.Context, bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
				return nil, bridgepkg.ErrBridgeInstanceUnavailable
			},
		})

		resp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/bridges",
			[]byte(
				`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"routing_policy":{"include_peer":true}}`,
			),
		)
		if resp.Code != http.StatusConflict {
			t.Fatalf(
				"create service error status = %d, want %d body=%s",
				resp.Code,
				http.StatusConflict,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should get maps not found", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			GetInstanceFn: func(context.Context, string) (*bridgepkg.BridgeInstance, error) {
				return nil, bridgepkg.ErrBridgeInstanceNotFound
			},
		})

		resp := performRequest(t, engine, http.MethodGet, "/bridges/missing", nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("get missing status = %d, want %d body=%s", resp.Code, http.StatusNotFound, resp.Body.String())
		}
	})

	t.Run("Should update rejects malformed json", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			UpdateInstanceFn: func(context.Context, bridgepkg.UpdateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
				t.Fatal("UpdateInstance() should not be called for malformed JSON")
				return nil, nil
			},
		})

		resp := performRequest(t, engine, http.MethodPatch, "/bridges/brg-core", []byte(`{"display_name":"broken"`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"update malformed json status = %d, want %d body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should update maps service errors", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			UpdateInstanceFn: func(context.Context, bridgepkg.UpdateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
				return nil, bridgepkg.ErrBridgeInstanceReadOnly
			},
		})

		resp := performRequest(t, engine, http.MethodPatch, "/bridges/brg-core", []byte(`{"display_name":"Renamed"}`))
		if resp.Code != http.StatusConflict {
			t.Fatalf(
				"update service error status = %d, want %d body=%s",
				resp.Code,
				http.StatusConflict,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should update rejects semantically invalid payload", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			UpdateInstanceFn: func(context.Context, bridgepkg.UpdateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
				t.Fatal("UpdateInstance() should not be called for invalid payload")
				return nil, nil
			},
		})

		resp := performRequest(
			t,
			engine,
			http.MethodPatch,
			"/bridges/brg-core",
			[]byte(`{"delivery_defaults":{"thread_id":"thr-1"}}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"update invalid payload status = %d, want %d body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should routes map not found", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			ListRoutesFn: func(context.Context, string) ([]bridgepkg.BridgeRoute, error) {
				return nil, bridgepkg.ErrBridgeRouteNotFound
			},
		})

		resp := performRequest(t, engine, http.MethodGet, "/bridges/brg-core/routes", nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("routes missing status = %d, want %d body=%s", resp.Code, http.StatusNotFound, resp.Body.String())
		}
	})

	t.Run("Should secret binding put maps service errors", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			PutSecretBindingFn: func(context.Context, bridgepkg.BridgeSecretBinding, *string) error {
				return bridgepkg.ErrBridgeInstanceReadOnly
			},
		})

		resp := performRequest(
			t,
			engine,
			http.MethodPut,
			"/bridges/brg-core/secret-bindings/bot_token",
			[]byte(`{"secret_ref":"vault:bridges/brg-core/bot_token","kind":"token"}`),
		)
		if resp.Code != http.StatusConflict {
			t.Fatalf(
				"put secret binding service error status = %d, want %d body=%s",
				resp.Code,
				http.StatusConflict,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should secret binding put rejects malformed json", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			PutSecretBindingFn: func(context.Context, bridgepkg.BridgeSecretBinding, *string) error {
				t.Fatal("PutSecretBinding() should not be called for malformed JSON")
				return nil
			},
		})

		resp := performRequest(
			t,
			engine,
			http.MethodPut,
			"/bridges/brg-core/secret-bindings/bot_token",
			[]byte(`{"secret_ref"`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"put secret binding malformed json status = %d, want %d body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should secret binding delete maps missing binding", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			DeleteSecretBindingFn: func(context.Context, string, string) error {
				return bridgepkg.ErrBridgeSecretBindingNotFound
			},
		})

		resp := performRequest(t, engine, http.MethodDelete, "/bridges/brg-core/secret-bindings/bot_token", nil)
		if resp.Code != http.StatusNotFound {
			t.Fatalf(
				"delete secret binding missing status = %d, want %d body=%s",
				resp.Code,
				http.StatusNotFound,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should test delivery rejects malformed json", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			ResolveDeliveryTargetFn: func(context.Context, bridgepkg.ResolveDeliveryTargetRequest) (*bridgepkg.DeliveryTarget, error) {
				t.Fatal("ResolveDeliveryTarget() should not be called for malformed JSON")
				return nil, nil
			},
		})

		resp := performRequest(t, engine, http.MethodPost, "/bridges/brg-core/test-delivery", []byte(`{"target"`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"test delivery malformed json status = %d, want %d body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should test delivery maps service errors", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			ResolveDeliveryTargetFn: func(context.Context, bridgepkg.ResolveDeliveryTargetRequest) (*bridgepkg.DeliveryTarget, error) {
				return nil, bridgepkg.ErrDeliveryQueueSaturated
			},
		})

		resp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/bridges/brg-core/test-delivery",
			[]byte(`{"target":{"peer_id":"peer-1"}}`),
		)
		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf(
				"test delivery service error status = %d, want %d body=%s",
				resp.Code,
				http.StatusServiceUnavailable,
				resp.Body.String(),
			)
		}
	})

	t.Run("Should test delivery rejects mismatched bridge instance id", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, testutil.StubBridgeService{
			ResolveDeliveryTargetFn: func(context.Context, bridgepkg.ResolveDeliveryTargetRequest) (*bridgepkg.DeliveryTarget, error) {
				t.Fatal("ResolveDeliveryTarget() should not be called for mismatched bridge id")
				return nil, nil
			},
		})

		resp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/bridges/brg-core/test-delivery",
			[]byte(`{"target":{"bridge_instance_id":"brg-other","peer_id":"peer-1"}}`),
		)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf(
				"test delivery mismatched bridge id status = %d, want %d body=%s",
				resp.Code,
				http.StatusBadRequest,
				resp.Body.String(),
			)
		}
	})
}

func TestBridgeHandlersLifecycleHelperErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		status  int
		bridges testutil.StubBridgeService
	}{
		{
			name:   "disable maps not found",
			path:   "/bridges/brg-core/disable",
			status: http.StatusNotFound,
			bridges: testutil.StubBridgeService{
				StopInstanceFn: func(context.Context, string) (*bridgepkg.BridgeInstance, error) {
					return nil, bridgepkg.ErrBridgeInstanceNotFound
				},
			},
		},
		{
			name:   "restart maps conflict",
			path:   "/bridges/brg-core/restart",
			status: http.StatusConflict,
			bridges: testutil.StubBridgeService{
				RestartInstanceFn: func(context.Context, string) (*bridgepkg.BridgeInstance, error) {
					return nil, bridgepkg.ErrInvalidBridgeStateTransition
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, engine := newBridgeHandlerFixture(t, tt.bridges)
			resp := performRequest(t, engine, http.MethodPost, tt.path, nil)
			if resp.Code != tt.status {
				t.Fatalf("%s status = %d, want %d body=%s", tt.path, resp.Code, tt.status, resp.Body.String())
			}
		})
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
					SecretSlots: []bridgepkg.BridgeSecretSlot{{
						Name:        "bot_token",
						Description: "Bot token",
						Required:    true,
					}},
					ConfigSchema: &bridgepkg.BridgeProviderConfigSchema{
						Schema:  "agh.bridge.telegram",
						Version: "v1",
					},
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
			if len(payload.Providers[0].SecretSlots) != 1 || payload.Providers[0].SecretSlots[0].Name != "bot_token" {
				t.Fatalf("provider secret_slots = %#v", payload.Providers[0].SecretSlots)
			}
			if payload.Providers[0].ConfigSchema == nil ||
				payload.Providers[0].ConfigSchema.Schema != "agh.bridge.telegram" {
				t.Fatalf("provider config_schema = %#v", payload.Providers[0].ConfigSchema)
			}
		})
	}
}

func TestBridgeHandlersIncludeObservedHealthPayloads(t *testing.T) {
	t.Parallel()

	t.Run("Should include observed health payloads in list and get responses", func(t *testing.T) {
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
			Status:        bridgepkg.BridgeStatusDegraded,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
			Degradation: &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonRateLimited,
				Message: "provider throttled",
			},
		}

		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
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
			Bridges: testutil.StubBridgeService{
				ListInstancesFn: func(context.Context) ([]bridgepkg.BridgeInstance, error) {
					return []bridgepkg.BridgeInstance{bridge}, nil
				},
				GetInstanceFn: func(context.Context, string) (*bridgepkg.BridgeInstance, error) {
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
		if listPayload.BridgeHealth[bridge.ID].Degradation == nil ||
			listPayload.BridgeHealth[bridge.ID].Degradation.Reason != bridgepkg.BridgeDegradationReasonRateLimited {
			t.Fatalf("bridge_health degradation = %#v", listPayload.BridgeHealth[bridge.ID].Degradation)
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
		if getPayload.Health.Degradation == nil ||
			getPayload.Health.Degradation.Reason != bridgepkg.BridgeDegradationReasonRateLimited {
			t.Fatalf("get health degradation = %#v", getPayload.Health.Degradation)
		}
		if getPayload.Health.LastSuccessAt == nil ||
			!getPayload.Health.LastSuccessAt.Equal(time.Date(2026, 4, 3, 11, 59, 0, 0, time.UTC)) {
			t.Fatalf("get health last_success_at = %#v, want 2026-04-03T11:59:00Z", getPayload.Health.LastSuccessAt)
		}
	})
}

func TestBridgeHandlersMutationReturnsBestEffortPayloadWhenHealthLookupFails(t *testing.T) {
	t.Parallel()

	t.Run("Should return best-effort bridge health payloads when observer lookup fails", func(t *testing.T) {
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

		handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
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

		resp := performRequest(
			t,
			engine,
			http.MethodPost,
			"/bridges",
			[]byte(
				`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"routing_policy":{"include_peer":true}}`,
			),
		)
		if resp.Code != http.StatusCreated {
			t.Fatalf("create status = %d, want %d; body=%s", resp.Code, http.StatusCreated, resp.Body.String())
		}

		var payload contract.BridgeResponse
		testutil.DecodeJSONResponse(t, resp, &payload)
		if payload.Bridge.ID != bridge.ID || payload.Bridge.Status != bridgepkg.BridgeStatusStarting {
			t.Fatalf("payload.Bridge = %#v, want created bridge payload", payload.Bridge)
		}
		if payload.Health.BridgeInstanceID != bridge.ID || payload.Health.Status != bridgepkg.BridgeStatusStarting ||
			payload.Health.RouteCount != 0 {
			t.Fatalf("payload.Health = %#v, want best-effort bridge identity and zero counters", payload.Health)
		}
	})
}

func TestBridgeHandlersReturnServiceUnavailableWhenNotConfigured(t *testing.T) {
	t.Parallel()

	t.Run("Should return service unavailable when bridge handlers are not configured", func(t *testing.T) {
		t.Parallel()

		_, engine := newBridgeHandlerFixture(t, nil)
		resp := performRequest(t, engine, http.MethodGet, "/bridges", nil)
		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusServiceUnavailable, resp.Body.String())
		}
	})
}

func newBridgeHandlerFixture(t *testing.T, bridges core.BridgeService) (*core.BaseHandlers, *gin.Engine) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	homePaths := testutil.NewTestHomePaths(t)
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.HTTP.Host = "127.0.0.1"
	cfg.HTTP.Port = 2123

	handlers := core.NewBaseHandlers(&core.BaseHandlerConfig{
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
	engine.GET("/bridges/:id/targets", handlers.ListBridgeTargets)
	engine.POST("/bridges/:id/resolve", handlers.ResolveBridgeTarget)
	engine.GET("/bridges/:id/secret-bindings", handlers.ListBridgeSecretBindings)
	engine.PUT("/bridges/:id/secret-bindings/:binding_name", handlers.PutBridgeSecretBinding)
	engine.DELETE("/bridges/:id/secret-bindings/:binding_name", handlers.DeleteBridgeSecretBinding)
	engine.POST("/bridges/:id/test-delivery", handlers.TestBridgeDelivery)
	return handlers, engine
}
