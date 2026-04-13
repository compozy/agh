//go:build integration

package udsapi

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestUDSBridgeCreateGetAndRoutesMirrorHTTP(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/bridges", []byte(`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body := mustReadAll(t, createResp.Body)
		t.Fatalf("create bridge status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, body)
	}

	var created contract.BridgeResponse
	decodeHTTPJSON(t, createResp, &created)
	if created.Bridge.ID == "" {
		t.Fatal("expected created bridge id")
	}

	getResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/bridges/"+created.Bridge.ID, nil, nil)
	if getResp.StatusCode != http.StatusOK {
		body := mustReadAll(t, getResp.Body)
		t.Fatalf("get bridge status = %d, want %d; body=%s", getResp.StatusCode, http.StatusOK, body)
	}

	var fetched contract.BridgeResponse
	decodeHTTPJSON(t, getResp, &fetched)
	if fetched.Bridge.ID != created.Bridge.ID || fetched.Bridge.DisplayName != "Support" {
		t.Fatalf("fetched.Bridge = %#v", fetched.Bridge)
	}

	if _, err := runtime.bridges.UpdateInstanceState(testutil.Context(t), bridgepkg.UpdateInstanceStateRequest{
		ID:      created.Bridge.ID,
		Enabled: true,
		Status:  bridgepkg.BridgeStatusReady,
	}); err != nil {
		t.Fatalf("runtime.bridges.UpdateInstanceState() error = %v", err)
	}
	if _, err := runtime.bridges.UpsertRoute(testutil.Context(t), bridgepkg.BridgeRoute{
		BridgeInstanceID: created.Bridge.ID,
		Scope:            bridgepkg.ScopeGlobal,
		PeerID:           "peer-1",
		SessionID:        "sess-1",
		AgentName:        "coder",
		LastActivityAt:   time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("runtime.bridges.UpsertRoute() error = %v", err)
	}

	routesResp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/bridges/"+created.Bridge.ID+"/routes", nil, nil)
	if routesResp.StatusCode != http.StatusOK {
		body := mustReadAll(t, routesResp.Body)
		t.Fatalf("bridge routes status = %d, want %d; body=%s", routesResp.StatusCode, http.StatusOK, body)
	}

	var routes contract.BridgeRoutesResponse
	decodeHTTPJSON(t, routesResp, &routes)
	if got, want := len(routes.Routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	if routes.Routes[0].BridgeInstanceID != created.Bridge.ID || routes.Routes[0].PeerID != "peer-1" {
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
