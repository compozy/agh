//go:build integration

package udsapi

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestUDSChannelCreateGetAndRoutesMirrorHTTP(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/channels", []byte(`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body := mustReadAll(t, createResp.Body)
		t.Fatalf("create channel status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, body)
	}

	var created contract.ChannelResponse
	decodeHTTPJSON(t, createResp, &created)
	if created.Channel.ID == "" {
		t.Fatal("expected created channel id")
	}

	getResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/channels/"+created.Channel.ID, nil, nil)
	if getResp.StatusCode != http.StatusOK {
		body := mustReadAll(t, getResp.Body)
		t.Fatalf("get channel status = %d, want %d; body=%s", getResp.StatusCode, http.StatusOK, body)
	}

	var fetched contract.ChannelResponse
	decodeHTTPJSON(t, getResp, &fetched)
	if fetched.Channel.ID != created.Channel.ID || fetched.Channel.DisplayName != "Support" {
		t.Fatalf("fetched.Channel = %#v", fetched.Channel)
	}

	if _, err := runtime.channels.UpdateInstanceState(testutil.Context(t), channelspkg.UpdateInstanceStateRequest{
		ID:      created.Channel.ID,
		Enabled: true,
		Status:  channelspkg.ChannelStatusReady,
	}); err != nil {
		t.Fatalf("runtime.channels.UpdateInstanceState() error = %v", err)
	}
	if _, err := runtime.channels.UpsertRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: created.Channel.ID,
		Scope:             channelspkg.ScopeGlobal,
		PeerID:            "peer-1",
		SessionID:         "sess-1",
		AgentName:         "coder",
		LastActivityAt:    time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("runtime.channels.UpsertRoute() error = %v", err)
	}

	routesResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/channels/"+created.Channel.ID+"/routes", nil, nil)
	if routesResp.StatusCode != http.StatusOK {
		body := mustReadAll(t, routesResp.Body)
		t.Fatalf("channel routes status = %d, want %d; body=%s", routesResp.StatusCode, http.StatusOK, body)
	}

	var routes contract.ChannelRoutesResponse
	decodeHTTPJSON(t, routesResp, &routes)
	if got, want := len(routes.Routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	if routes.Routes[0].ChannelInstanceID != created.Channel.ID || routes.Routes[0].PeerID != "peer-1" {
		t.Fatalf("routes = %#v", routes.Routes)
	}
}

func mustReadAll(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	defer func() {
		if err := body.Close(); err != nil {
			t.Errorf("body.Close() error = %v", err)
		}
	}()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	return string(data)
}
