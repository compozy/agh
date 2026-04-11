package udsapi

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	channelspkg "github.com/pedronauck/agh/internal/channels"
)

func TestCreateChannelHandlerReturnsPersistedPayload(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	channels := stubChannelService{
		CreateInstanceFn: func(_ context.Context, req channelspkg.CreateInstanceRequest) (*channelspkg.ChannelInstance, error) {
			if req.Scope != channelspkg.ScopeGlobal || req.Platform != "telegram" || req.ExtensionName != "ext-telegram" || req.DisplayName != "Support" {
				t.Fatalf("CreateInstance() req = %#v", req)
			}
			return &channelspkg.ChannelInstance{
				ID:            "chan-uds",
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

	engine := newTestRouter(t, newTestHandlersWithChannels(t, stubSessionManager{}, stubObserver{}, channels, stubWorkspaceService{}, homePaths))
	body := []byte(`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`)
	recorder := performRequest(t, engine, http.MethodPost, "/api/channels", body)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}

	var response contract.ChannelResponse
	decodeJSONResponse(t, recorder, &response)
	if response.Channel.ID != "chan-uds" || response.Channel.Scope != channelspkg.ScopeGlobal {
		t.Fatalf("response.Channel = %#v", response.Channel)
	}
}

func TestGetChannelHandlerReturnsPersistedPayload(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	channels := stubChannelService{
		GetInstanceFn: func(_ context.Context, id string) (*channelspkg.ChannelInstance, error) {
			if id != "chan-uds" {
				t.Fatalf("GetInstance() id = %q, want chan-uds", id)
			}
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
	}

	engine := newTestRouter(t, newTestHandlersWithChannels(t, stubSessionManager{}, stubObserver{}, channels, stubWorkspaceService{}, homePaths))
	recorder := performRequest(t, engine, http.MethodGet, "/api/channels/chan-uds", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.ChannelResponse
	decodeJSONResponse(t, recorder, &response)
	if response.Channel.ID != "chan-uds" || response.Channel.Status != channelspkg.ChannelStatusReady {
		t.Fatalf("response.Channel = %#v", response.Channel)
	}
}

func TestListChannelRoutesHandlerReturnsRequestedPayload(t *testing.T) {
	t.Parallel()

	homePaths := newTestHomePaths(t)
	channels := stubChannelService{
		ListRoutesFn: func(_ context.Context, channelInstanceID string) ([]channelspkg.ChannelRoute, error) {
			if channelInstanceID != "chan-uds" {
				t.Fatalf("ListRoutes() channelInstanceID = %q, want chan-uds", channelInstanceID)
			}
			return []channelspkg.ChannelRoute{
				{
					RoutingKeyHash:    "hash-uds",
					Scope:             channelspkg.ScopeGlobal,
					ChannelInstanceID: "chan-uds",
					PeerID:            "peer-1",
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
	recorder := performRequest(t, engine, http.MethodGet, "/api/channels/chan-uds/routes", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response contract.ChannelRoutesResponse
	decodeJSONResponse(t, recorder, &response)
	if got, want := len(response.Routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	if response.Routes[0].ChannelInstanceID != "chan-uds" || response.Routes[0].PeerID != "peer-1" {
		t.Fatalf("route = %#v", response.Routes[0])
	}
}
