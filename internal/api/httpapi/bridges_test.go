package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

func TestCreateBridgeHandlerCreatesBridgeInstance(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	bridges := stubBridgeService{
		CreateInstanceFn: func(_ context.Context, req bridgepkg.CreateInstanceRequest) (*bridgepkg.BridgeInstance, error) {
			if req.Scope != bridgepkg.ScopeWorkspace || req.WorkspaceID != "ws-alpha" || req.Platform != "telegram" || req.ExtensionName != "ext-telegram" || req.DisplayName != "Support" {
				t.Fatalf("CreateInstance() req = %#v", req)
			}
			if !req.Enabled || req.Status != bridgepkg.BridgeStatusStarting || !req.RoutingPolicy.IncludePeer {
				t.Fatalf("CreateInstance() lifecycle = %#v", req)
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
				RoutingPolicy:    req.RoutingPolicy,
				DeliveryDefaults: req.DeliveryDefaults,
				CreatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
				UpdatedAt:        time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
			}, nil
		},
	}

	engine := newTestRouter(t, newTestHandlersWithBridges(t, stubSessionManager{}, stubObserver{}, bridges, stubWorkspaceService{}, homePaths))
	body := []byte(`{"scope":"workspace","workspace_id":"ws-alpha","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`)
	recorder := performRequest(t, engine, http.MethodPost, "/api/bridges", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response contract.BridgeResponse
	decodeJSONResponse(t, recorder, &response)
	if response.Bridge.ID != "brg-1" || response.Bridge.WorkspaceID != "ws-alpha" || response.Bridge.Status != bridgepkg.BridgeStatusStarting {
		t.Fatalf("response.Bridge = %#v", response.Bridge)
	}
}

func TestListBridgeRoutesHandlerReturnsRequestedRouteSet(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	bridges := stubBridgeService{
		ListRoutesFn: func(_ context.Context, bridgeInstanceID string) ([]bridgepkg.BridgeRoute, error) {
			if bridgeInstanceID != "brg-1" {
				t.Fatalf("ListRoutes() bridgeInstanceID = %q, want brg-1", bridgeInstanceID)
			}
			return []bridgepkg.BridgeRoute{
				{
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
				},
			}, nil
		},
	}

	engine := newTestRouter(t, newTestHandlersWithBridges(t, stubSessionManager{}, stubObserver{}, bridges, stubWorkspaceService{}, homePaths))
	recorder := performRequest(t, engine, http.MethodGet, "/api/bridges/brg-1/routes", nil)
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
}

func TestBridgeTestDeliveryHandlerResolvesTypedTarget(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	bridges := stubBridgeService{
		ResolveDeliveryTargetFn: func(_ context.Context, req bridgepkg.ResolveDeliveryTargetRequest) (*bridgepkg.DeliveryTarget, error) {
			if req.BridgeInstanceID != "brg-1" || req.PeerID != "peer-1" || req.ThreadID != "thread-1" || req.GroupID != "group-1" || req.Mode != bridgepkg.DeliveryModeReply {
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
	}

	engine := newTestRouter(t, newTestHandlersWithBridges(t, stubSessionManager{}, stubObserver{}, bridges, stubWorkspaceService{}, homePaths))
	body := []byte(`{"message":"hello","target":{"peer_id":"peer-1","thread_id":"thread-1","group_id":"group-1","mode":"reply"}}`)
	recorder := performRequest(t, engine, http.MethodPost, "/api/bridges/brg-1/test-delivery", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.BridgeTestDeliveryResponse
	decodeJSONResponse(t, recorder, &response)
	if response.Status != "resolved" || response.DeliveryTarget.BridgeInstanceID != "brg-1" || response.DeliveryTarget.Mode != bridgepkg.DeliveryModeReply {
		t.Fatalf("response = %#v", response)
	}
}
