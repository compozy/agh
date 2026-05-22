package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/compozy/agh/internal/api/contract"
	bridgepkg "github.com/compozy/agh/internal/bridges"
)

func TestBridgeHandlersShouldHandleBridgeRoutes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		method  string
		path    string
		body    []byte
		bridges stubBridgeService
		assert  func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "ShouldListBridgeProviders",
			method: http.MethodGet,
			path:   "/api/bridges/providers",
			bridges: stubBridgeService{
				ListProvidersFn: func(context.Context) ([]bridgepkg.BridgeProvider, error) {
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
						Enabled: true,
						State:   "active",
						Health:  "healthy",
					}}, nil
				},
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()
				if recorder.Code != http.StatusOK {
					t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
				}

				var response contract.BridgeProvidersResponse
				decodeJSONResponse(t, recorder, &response)
				if got, want := len(response.Providers), 1; got != want {
					t.Fatalf("len(providers) = %d, want %d", got, want)
				}
				if response.Providers[0].ExtensionName != "telegram-reference" {
					t.Fatalf("provider = %#v", response.Providers[0])
				}
				if len(response.Providers[0].SecretSlots) != 1 ||
					response.Providers[0].SecretSlots[0].Name != "bot_token" {
					t.Fatalf("provider secret slots = %#v", response.Providers[0].SecretSlots)
				}
				if response.Providers[0].ConfigSchema == nil ||
					response.Providers[0].ConfigSchema.Schema != "agh.bridge.telegram" {
					t.Fatalf("provider config schema = %#v", response.Providers[0].ConfigSchema)
				}
			},
		},
		{
			name:   "ShouldCreateBridgeInstance",
			method: http.MethodPost,
			path:   "/api/bridges",
			body: []byte(
				`{"scope":"workspace","workspace_id":"ws-alpha","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"dm_policy":"pairing","routing_policy":{"include_peer":true},"provider_config":{"mode":"bot","tenant":"acme"},"delivery_defaults":{"peer_id":"peer-default","mode":"reply"}}`,
			),
			bridges: stubBridgeService{
				CreateInstanceFn: func(_ context.Context, req bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
					if req.Scope != bridgepkg.ScopeWorkspace || req.WorkspaceID != "ws-alpha" ||
						req.Platform != "telegram" ||
						req.ExtensionName != "ext-telegram" ||
						req.DisplayName != "Support" {
						t.Fatalf("CreateInstance() req = %#v", req)
					}
					if !req.Enabled || req.Status != bridgepkg.BridgeStatusStarting || !req.RoutingPolicy.IncludePeer {
						t.Fatalf("CreateInstance() lifecycle = %#v", req)
					}
					if req.DMPolicy != bridgepkg.BridgeDMPolicyPairing {
						t.Fatalf(
							"CreateInstance().DMPolicy = %q, want %q",
							req.DMPolicy,
							bridgepkg.BridgeDMPolicyPairing,
						)
					}
					if got, want := string(req.ProviderConfig), `{"mode":"bot","tenant":"acme"}`; got != want {
						t.Fatalf("CreateInstance().ProviderConfig = %s, want %s", got, want)
					}
					if got, want := string(
						req.DeliveryDefaults,
					), `{"peer_id":"peer-default","mode":"reply"}`; got != want {
						t.Fatalf("CreateInstance().DeliveryDefaults = %s, want %s", got, want)
					}
					return &bridgepkg.BridgeInstance{
						ID:               "brg-1",
						Scope:            req.Scope,
						WorkspaceID:      req.WorkspaceID,
						Platform:         req.Platform,
						ExtensionName:    req.ExtensionName,
						DisplayName:      req.DisplayName,
						Enabled:          req.Enabled,
						Status:           req.Status,
						DMPolicy:         req.DMPolicy,
						RoutingPolicy:    req.RoutingPolicy,
						ProviderConfig:   req.ProviderConfig,
						DeliveryDefaults: req.DeliveryDefaults,
						Degradation:      req.Degradation,
						CreatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
						UpdatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
					}, nil
				},
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()
				if recorder.Code != http.StatusCreated {
					t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
				}

				var response contract.BridgeResponse
				decodeJSONResponse(t, recorder, &response)
				if response.Bridge.ID != "brg-1" || response.Bridge.WorkspaceID != "ws-alpha" ||
					response.Bridge.Status != bridgepkg.BridgeStatusStarting {
					t.Fatalf("response.Bridge = %#v", response.Bridge)
				}
				if got, want := string(response.Bridge.ProviderConfig), `{"mode":"bot","tenant":"acme"}`; got != want {
					t.Fatalf("response.Bridge.ProviderConfig = %s, want %s", got, want)
				}
			},
		},
		{
			name:   "ShouldListRequestedBridgeRoutes",
			method: http.MethodGet,
			path:   "/api/bridges/brg-1/routes",
			bridges: stubBridgeService{
				ListRoutesFn: func(_ context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeRoute, error) {
					if bridgeInstanceID != "brg-1" {
						t.Fatalf("ListRoutes() bridgeInstanceID = %q, want brg-1", bridgeInstanceID)
					}
					return []bridgepkg.BridgeRoute{{
						RoutingKeyHash:   "hash-1",
						Scope:            bridgepkg.ScopeWorkspace,
						WorkspaceID:      "ws-alpha",
						BridgeInstanceID: "brg-1",
						PeerID:           "peer-1",
						ThreadID:         "thread-1",
						GroupID:          "group-1",
						SessionID:        "sess-1",
						AgentName:        "coder",
						LastActivityAt:   time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
						CreatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
						UpdatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
					}}, nil
				},
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()
				if recorder.Code != http.StatusOK {
					t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
				}

				var response contract.BridgeRoutesResponse
				decodeJSONResponse(t, recorder, &response)
				if got, want := len(response.Routes), 1; got != want {
					t.Fatalf("len(routes) = %d, want %d", got, want)
				}
				if response.Routes[0].BridgeInstanceID != "brg-1" || response.Routes[0].ThreadID != "thread-1" {
					t.Fatalf("route = %#v", response.Routes[0])
				}
			},
		},
		{
			name:   "ShouldResolveTypedDeliveryTarget",
			method: http.MethodPost,
			path:   "/api/bridges/brg-1/test-delivery",
			body: []byte(
				`{"message":"hello","target":{"peer_id":"peer-1","thread_id":"thread-1","group_id":"group-1","mode":"reply"}}`,
			),
			bridges: stubBridgeService{
				ResolveDeliveryTargetFn: func(_ context.Context, req bridgepkg.ResolveDeliveryTargetRequest) (*bridgepkg.DeliveryTarget, error) {
					if req.BridgeInstanceID != "brg-1" || req.PeerID != "peer-1" || req.ThreadID != "thread-1" ||
						req.GroupID != "group-1" ||
						req.Mode != bridgepkg.DeliveryModeReply {
						t.Fatalf("ResolveDeliveryTarget() req = %#v", req)
					}
					return &bridgepkg.DeliveryTarget{
						BridgeInstanceID: req.BridgeInstanceID,
						PeerID:           req.PeerID,
						ThreadID:         req.ThreadID,
						GroupID:          req.GroupID,
						Mode:             req.Mode,
					}, nil
				},
			},
			assert: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				t.Helper()
				if recorder.Code != http.StatusOK {
					t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
				}

				var response contract.BridgeTestDeliveryResponse
				decodeJSONResponse(t, recorder, &response)
				if response.Status != "resolved" || response.DeliveryTarget.BridgeInstanceID != "brg-1" ||
					response.DeliveryTarget.Mode != bridgepkg.DeliveryModeReply {
					t.Fatalf("response = %#v", response)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			homePaths := newTestHomePaths(t)
			engine := newTestRouter(
				t,
				newTestHandlersWithBridges(
					t,
					stubSessionManager{},
					stubObserver{},
					tc.bridges,
					stubWorkspaceService{},
					homePaths,
				),
			)
			recorder := performRequest(t, engine, tc.method, tc.path, tc.body)
			tc.assert(t, recorder)
		})
	}
}
