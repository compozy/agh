package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

func TestGChatClawpatchRegressions(t *testing.T) {
	// not parallel: these regressions pin process-level GChat endpoint env precedence.
	t.Run("Should resolve provider config API and token URLs when env overrides are unset", func(t *testing.T) {
		t.Setenv(gchatAPIBaseEnv, "")
		t.Setenv(gchatTokenURLEnv, "")

		provider := newGChatClawpatchProvider(t)
		managed := testBridgeRuntime(t, provider.now(), "brg-gchat")
		credentials := mustCredentials(t)
		credentials.TokenURI = "https://credentials.example.invalid/oauth2/token"
		cfg := gchatProviderConfig{
			APIBaseURL: "https://tenant.example.invalid/chat",
			TokenURL:   "https://tenant.example.invalid/oauth2/token",
		}

		resolved, err := provider.newResolvedGChatConfig(managed, cfg, credentials, nil)
		if err != nil {
			t.Fatalf("newResolvedGChatConfig() error = %v", err)
		}
		if got, want := resolved.apiBaseURL, "https://tenant.example.invalid/chat"; got != want {
			t.Fatalf("resolved.apiBaseURL = %q, want %q", got, want)
		}
		if got, want := resolved.tokenURL, "https://tenant.example.invalid/oauth2/token"; got != want {
			t.Fatalf("resolved.tokenURL = %q, want %q", got, want)
		}
	})

	t.Run("Should evict terminal delivery states after acknowledgement", func(t *testing.T) {
		provider := newGChatClawpatchProvider(t)
		api := &fakeGChatAPI{}
		provider.apiFactory = func(*resolvedInstanceConfig) gchatAPI { return api }
		installGChatClawpatchRoute(provider, &resolvedInstanceConfig{instanceID: "brg-gchat"})

		startReq := testDeliveryRequest("brg-gchat", "delivery-final", 1, bridgepkg.DeliveryEventTypeStart, false)
		if _, err := provider.handleBridgesDeliver(context.Background(), nil, startReq); err != nil {
			t.Fatalf("handleBridgesDeliver(start) error = %v", err)
		}
		if state := provider.deliveryState("brg-gchat", "delivery-final"); state.RemoteMessageID == "" {
			t.Fatalf("deliveryState(start) = %#v, want retained remote message id", state)
		}

		finalReq := testDeliveryRequest("brg-gchat", "delivery-final", 2, bridgepkg.DeliveryEventTypeFinal, true)
		if _, err := provider.handleBridgesDeliver(context.Background(), nil, finalReq); err != nil {
			t.Fatalf("handleBridgesDeliver(final) error = %v", err)
		}
		if state := provider.deliveryState("brg-gchat", "delivery-final"); state != (deliveryState{}) {
			t.Fatalf("deliveryState(final) = %#v, want evicted", state)
		}

		deleteStartReq := testDeliveryRequest(
			"brg-gchat",
			"delivery-delete",
			1,
			bridgepkg.DeliveryEventTypeStart,
			false,
		)
		deleteStartAck, err := provider.handleBridgesDeliver(context.Background(), nil, deleteStartReq)
		if err != nil {
			t.Fatalf("handleBridgesDeliver(delete start) error = %v", err)
		}
		deleteReq := testDeleteRequest("brg-gchat", "delivery-delete", 2, deleteStartAck.RemoteMessageID)
		if _, err := provider.handleBridgesDeliver(context.Background(), nil, deleteReq); err != nil {
			t.Fatalf("handleBridgesDeliver(delete) error = %v", err)
		}
		if state := provider.deliveryState("brg-gchat", "delivery-delete"); state != (deliveryState{}) {
			t.Fatalf("deliveryState(delete) = %#v, want evicted", state)
		}
	})

	t.Run("Should reuse provider API clients across deliveries", func(t *testing.T) {
		server := newGChatClawpatchAPIServer(t)
		defer server.Close()

		provider := newGChatClawpatchProvider(t)
		cfg := resolvedInstanceConfig{
			instanceID:      "brg-gchat",
			apiBaseURL:      server.URL(),
			tokenURL:        server.TokenURL(),
			credentials:     mustCredentials(t),
			dmPolicy:        bridgepkg.BridgeDMPolicyOpen,
			webhookPath:     "/gchat/brg-gchat",
			listenAddr:      "127.0.0.1:0",
			dedup:           nil,
			rateLimiter:     nil,
			inFlightLimiter: nil,
		}
		installGChatClawpatchRoute(provider, &cfg)

		firstReq := testDeliveryRequest("brg-gchat", "delivery-cache-1", 1, bridgepkg.DeliveryEventTypeStart, false)
		if _, err := provider.handleBridgesDeliver(context.Background(), nil, firstReq); err != nil {
			t.Fatalf("handleBridgesDeliver(first) error = %v", err)
		}
		secondReq := testDeliveryRequest("brg-gchat", "delivery-cache-2", 1, bridgepkg.DeliveryEventTypeStart, false)
		if _, err := provider.handleBridgesDeliver(context.Background(), nil, secondReq); err != nil {
			t.Fatalf("handleBridgesDeliver(second) error = %v", err)
		}

		if got, want := server.TokenHits(), 1; got != want {
			t.Fatalf("token endpoint hits = %d, want %d", got, want)
		}
	})
}

func newGChatClawpatchProvider(t *testing.T) *gchatProvider {
	t.Helper()

	provider, err := newGChatProvider(io.Discard)
	if err != nil {
		t.Fatalf("newGChatProvider() error = %v", err)
	}
	return provider
}

func installGChatClawpatchRoute(provider *gchatProvider, cfg *resolvedInstanceConfig) {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	provider.routes[cfg.instanceID] = *cfg
	provider.reportedStatus[cfg.instanceID] = bridgepkg.BridgeStatusReady
}

type gchatClawpatchAPIServer struct {
	server      *httptest.Server
	mu          sync.Mutex
	tokenHits   int
	messageHits int
}

func newGChatClawpatchAPIServer(t *testing.T) *gchatClawpatchAPIServer {
	t.Helper()

	s := &gchatClawpatchAPIServer{}
	s.server = httptest.NewServer(http.HandlerFunc(s.serveHTTP))
	return s
}

func (s *gchatClawpatchAPIServer) Close() { s.server.Close() }

func (s *gchatClawpatchAPIServer) URL() string { return s.server.URL }

func (s *gchatClawpatchAPIServer) TokenURL() string { return s.server.URL + "/oauth2/token" }

func (s *gchatClawpatchAPIServer) TokenHits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokenHits
}

func (s *gchatClawpatchAPIServer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/oauth2/token":
		s.mu.Lock()
		s.tokenHits++
		s.mu.Unlock()
		if err := json.NewEncoder(w).Encode(gchatTokenResponse{AccessToken: "token-123", ExpiresIn: 3600}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/") && strings.HasSuffix(r.URL.Path, "/messages"):
		s.mu.Lock()
		s.messageHits++
		messageName := "spaces/AAA/messages/msg-" + strconv.Itoa(s.messageHits)
		s.mu.Unlock()
		if err := json.NewEncoder(w).Encode(gchatSentMessage{Name: messageName}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	default:
		http.NotFound(w, r)
	}
}
