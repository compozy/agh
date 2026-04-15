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

	createResp := mustUnixRequest(t, runtime.client, http.MethodPost, "http://unix/api/bridges", []byte(`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","dm_policy":"pairing","routing_policy":{"include_peer":true},"provider_config":{"mode":"bot","tenant":"acme"},"delivery_defaults":{"peer_id":"peer-default","mode":"reply"}}`), nil)
	if createResp.StatusCode != http.StatusCreated {
		body := mustReadAll(t, createResp.Body)
		t.Fatalf("create bridge status = %d, want %d; body=%s", createResp.StatusCode, http.StatusCreated, body)
	}

	var created contract.BridgeResponse
	decodeHTTPJSON(t, createResp, &created)
	if created.Bridge.ID == "" {
		t.Fatal("expected created bridge id")
	}
	if created.Bridge.DMPolicy != bridgepkg.BridgeDMPolicyPairing {
		t.Fatalf("created.Bridge.DMPolicy = %q, want %q", created.Bridge.DMPolicy, bridgepkg.BridgeDMPolicyPairing)
	}
	if got, want := string(created.Bridge.ProviderConfig), `{"mode":"bot","tenant":"acme"}`; got != want {
		t.Fatalf("created.Bridge.ProviderConfig = %s, want %s", got, want)
	}
	if got, want := string(created.Bridge.DeliveryDefaults), `{"peer_id":"peer-default","mode":"reply"}`; got != want {
		t.Fatalf("created.Bridge.DeliveryDefaults = %s, want %s", got, want)
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
	if got, want := string(fetched.Bridge.ProviderConfig), `{"mode":"bot","tenant":"acme"}`; got != want {
		t.Fatalf("fetched.Bridge.ProviderConfig = %s, want %s", got, want)
	}
	if got, want := string(fetched.Bridge.DeliveryDefaults), `{"peer_id":"peer-default","mode":"reply"}`; got != want {
		t.Fatalf("fetched.Bridge.DeliveryDefaults = %s, want %s", got, want)
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

func TestUDSBridgeProvidersExposeOperatorMetadata(t *testing.T) {
	runtime := newIntegrationRuntime(t)
	runtime.bridges.providers = []bridgepkg.BridgeProvider{{
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
	}}

	resp := mustUnixRequest(t, runtime.client, http.MethodGet, "http://unix/api/bridges/providers", nil, nil)
	if resp.StatusCode != http.StatusOK {
		body := mustReadAll(t, resp.Body)
		t.Fatalf("provider list status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, body)
	}

	var payload contract.BridgeProvidersResponse
	decodeHTTPJSON(t, resp, &payload)
	if got, want := len(payload.Providers), 1; got != want {
		t.Fatalf("len(providers) = %d, want %d", got, want)
	}
	if len(payload.Providers[0].SecretSlots) != 1 || payload.Providers[0].SecretSlots[0].Name != "bot_token" {
		t.Fatalf("providers[0].SecretSlots = %#v", payload.Providers[0].SecretSlots)
	}
	if payload.Providers[0].ConfigSchema == nil || payload.Providers[0].ConfigSchema.Schema != "agh.bridge.telegram" {
		t.Fatalf("providers[0].ConfigSchema = %#v", payload.Providers[0].ConfigSchema)
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
