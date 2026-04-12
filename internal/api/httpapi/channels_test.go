package httpapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	channelspkg "github.com/pedronauck/agh/internal/channels"
)

func TestCreateChannelHandlerCreatesChannelInstance(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	channels := stubChannelService{
		CreateInstanceFn: func(_ context.Context, req channelspkg.CreateInstanceRequest) (*channelspkg.ChannelInstance, error) {
			if req.Scope != channelspkg.ScopeWorkspace || req.WorkspaceID != "ws-alpha" || req.Platform != "telegram" || req.ExtensionName != "ext-telegram" || req.DisplayName != "Support" {
				t.Fatalf("CreateInstance() req = %#v", req)
			}
			if !req.Enabled || req.Status != channelspkg.ChannelStatusStarting || !req.RoutingPolicy.IncludePeer {
				t.Fatalf("CreateInstance() lifecycle = %#v", req)
			}
			return &channelspkg.ChannelInstance{
				ID:               "chan-1",
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

	engine := newTestRouter(t, newTestHandlersWithChannels(t, stubSessionManager{}, stubObserver{}, channels, stubWorkspaceService{}, homePaths))
	body := []byte(`{"scope":"workspace","workspace_id":"ws-alpha","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`)
	recorder := performRequest(t, engine, http.MethodPost, "/api/channels", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response contract.ChannelResponse
	decodeJSONResponse(t, recorder, &response)
	if response.Channel.ID != "chan-1" || response.Channel.WorkspaceID != "ws-alpha" || response.Channel.Status != channelspkg.ChannelStatusStarting {
		t.Fatalf("response.Channel = %#v", response.Channel)
	}
}

func TestListChannelRoutesHandlerReturnsRequestedRouteSet(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	channels := stubChannelService{
		ListRoutesFn: func(_ context.Context, channelInstanceID string) ([]channelspkg.ChannelRoute, error) {
			if channelInstanceID != "chan-1" {
				t.Fatalf("ListRoutes() channelInstanceID = %q, want chan-1", channelInstanceID)
			}
			return []channelspkg.ChannelRoute{
				{
					RoutingKeyHash:    "hash-1",
					Scope:             channelspkg.ScopeWorkspace,
					WorkspaceID:       "ws-alpha",
					ChannelInstanceID: "chan-1",
					PeerID:            "peer-1",
					ThreadID:          "thread-1",
					GroupID:           "group-1",
					SessionID:         "sess-1",
					AgentName:         "coder",
					LastActivityAt:    time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
					CreatedAt:         time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
					UpdatedAt:         time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		},
	}

	engine := newTestRouter(t, newTestHandlersWithChannels(t, stubSessionManager{}, stubObserver{}, channels, stubWorkspaceService{}, homePaths))
	recorder := performRequest(t, engine, http.MethodGet, "/api/channels/chan-1/routes", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.ChannelRoutesResponse
	decodeJSONResponse(t, recorder, &response)
	if got, want := len(response.Routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	if response.Routes[0].ChannelInstanceID != "chan-1" || response.Routes[0].ThreadID != "thread-1" {
		t.Fatalf("route = %#v", response.Routes[0])
	}
}

func TestChannelTestDeliveryHandlerResolvesTypedTarget(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	channels := stubChannelService{
		ResolveDeliveryTargetFn: func(_ context.Context, req channelspkg.ResolveDeliveryTargetRequest) (*channelspkg.DeliveryTarget, error) {
			if req.ChannelInstanceID != "chan-1" || req.PeerID != "peer-1" || req.ThreadID != "thread-1" || req.GroupID != "group-1" || req.Mode != channelspkg.DeliveryModeReply {
				t.Fatalf("ResolveDeliveryTarget() req = %#v", req)
			}
			return &channelspkg.DeliveryTarget{
				ChannelInstanceID: req.ChannelInstanceID,
				PeerID:            req.PeerID,
				ThreadID:          req.ThreadID,
				GroupID:           req.GroupID,
				Mode:              req.Mode,
			}, nil
		},
	}

	engine := newTestRouter(t, newTestHandlersWithChannels(t, stubSessionManager{}, stubObserver{}, channels, stubWorkspaceService{}, homePaths))
	body := []byte(`{"message":"hello","target":{"peer_id":"peer-1","thread_id":"thread-1","group_id":"group-1","mode":"reply"}}`)
	recorder := performRequest(t, engine, http.MethodPost, "/api/channels/chan-1/test-delivery", body)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.ChannelTestDeliveryResponse
	decodeJSONResponse(t, recorder, &response)
	if response.Status != "resolved" || response.DeliveryTarget.ChannelInstanceID != "chan-1" || response.DeliveryTarget.Mode != channelspkg.DeliveryModeReply {
		t.Fatalf("response = %#v", response)
	}
}
