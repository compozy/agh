package udsapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

func TestCreateBridgeHandlerReturnsPersistedPayload(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	bridges := stubBridgeService{
		CreateInstanceFn: func(_ context.Context, req bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
			if req.Scope != bridgepkg.ScopeGlobal || req.Platform != "telegram" || req.ExtensionName != "ext-telegram" || req.DisplayName != "Support" {
				t.Fatalf("CreateInstance() req = %#v", req)
			}
			return &bridgepkg.BridgeInstance{
				ID:            "brg-uds",
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
	}

	engine := newTestRouter(t, newTestHandlersWithBridges(t, stubSessionManager{}, stubObserver{}, bridges, stubWorkspaceService{}, homePaths))
	body := []byte(`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`)
	recorder := performRequest(t, engine, http.MethodPost, "/api/bridges", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response contract.BridgeResponse
	decodeJSONResponse(t, recorder, &response)
	if response.Bridge.ID != "brg-uds" || response.Bridge.Scope != bridgepkg.ScopeGlobal {
		t.Fatalf("response.Bridge = %#v", response.Bridge)
	}
}

func TestGetBridgeHandlerReturnsPersistedPayload(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	bridges := stubBridgeService{
		GetInstanceFn: func(_ context.Context, id string) (*bridgepkg.BridgeInstance, error) {
			if id != "brg-uds" {
				t.Fatalf("GetInstance() id = %q, want brg-uds", id)
			}
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
	}

	engine := newTestRouter(t, newTestHandlersWithBridges(t, stubSessionManager{}, stubObserver{}, bridges, stubWorkspaceService{}, homePaths))
	recorder := performRequest(t, engine, http.MethodGet, "/api/bridges/brg-uds", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.BridgeResponse
	decodeJSONResponse(t, recorder, &response)
	if response.Bridge.ID != "brg-uds" || response.Bridge.Status != bridgepkg.BridgeStatusReady {
		t.Fatalf("response.Bridge = %#v", response.Bridge)
	}
}

func TestListBridgeRoutesHandlerReturnsRequestedPayload(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	bridges := stubBridgeService{
		ListRoutesFn: func(_ context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeRoute, error) {
			if bridgeInstanceID != "brg-uds" {
				t.Fatalf("ListRoutes() bridgeInstanceID = %q, want brg-uds", bridgeInstanceID)
			}
			return []bridgepkg.BridgeRoute{
				{
					RoutingKeyHash:   "hash-uds",
					Scope:            bridgepkg.ScopeGlobal,
					BridgeInstanceID: "brg-uds",
					PeerID:           "peer-1",
					SessionID:        "sess-1",
					AgentName:        "coder",
					LastActivityAt:   time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
					CreatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
					UpdatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		},
	}

	engine := newTestRouter(t, newTestHandlersWithBridges(t, stubSessionManager{}, stubObserver{}, bridges, stubWorkspaceService{}, homePaths))
	recorder := performRequest(t, engine, http.MethodGet, "/api/bridges/brg-uds/routes", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.BridgeRoutesResponse
	decodeJSONResponse(t, recorder, &response)
	if got, want := len(response.Routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	if response.Routes[0].BridgeInstanceID != "brg-uds" || response.Routes[0].PeerID != "peer-1" {
		t.Fatalf("route = %#v", response.Routes[0])
	}
}

func TestListBridgeProvidersHandlerReturnsRequestedPayload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "Should return requested payload"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			homePaths := newTestHomePaths(t)
			bridges := stubBridgeService{
				ListProvidersFn: func(_ context.Context) ([]bridgepkg.BridgeProvider, error) {
					return []bridgepkg.BridgeProvider{{
						Platform:      "telegram",
						ExtensionName: "telegram-reference",
						DisplayName:   "Telegram",
						Description:   "Reference Telegram bridge adapter",
						Enabled:       true,
						State:         "active",
						Health:        "healthy",
					}}, nil
				},
			}

			engine := newTestRouter(t, newTestHandlersWithBridges(t, stubSessionManager{}, stubObserver{}, bridges, stubWorkspaceService{}, homePaths))
			recorder := performRequest(t, engine, http.MethodGet, "/api/bridges/providers", nil)
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
		})
	}
}
