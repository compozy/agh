package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	bridgepkg "github.com/compozy/agh/internal/bridges"
)

func TestTeamsContractHealthAggregation(t *testing.T) {
	t.Run("Should remain unhealthy while any reported instance is auth required", func(t *testing.T) {
		runtime, err := newTeamsProvider(io.Discard)
		if err != nil {
			t.Fatalf("newTeamsProvider() error = %v", err)
		}
		runtime.mu.Lock()
		runtime.reportedStatus["brg-ready"] = bridgepkg.BridgeStatusReady
		runtime.reportedStatus["brg-auth"] = bridgepkg.BridgeStatusAuthRequired
		runtime.mu.Unlock()

		err = runtime.healthCheck()
		if err == nil || !strings.Contains(err.Error(), "brg-auth") {
			t.Fatalf("healthCheck() error = %v, want auth-required instance error", err)
		}

		runtime.clearLastError()
		err = runtime.healthCheck()
		if err == nil || !strings.Contains(err.Error(), "brg-auth") {
			t.Fatalf("healthCheck() after clearLastError() error = %v, want auth-required instance error", err)
		}
	})
}

func TestTeamsContractAuthorizationCache(t *testing.T) {
	// not parallel: loopback credentialed URL opt-in uses process environment.

	t.Run("Should reuse OpenID metadata and JWKS across authenticated webhooks", func(t *testing.T) {
		identity := newCountingTeamsIdentityServer(t)
		cfg := resolvedInstanceConfig{
			appID:             "app-id",
			serviceURL:        identity.ServiceURL(),
			openIDMetadataURL: identity.MetadataURL(),
		}
		body := []byte(teamsMessageWebhook(identity.ServiceURL(), "Need a summary"))
		for range 3 {
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				"http://127.0.0.1/teams/brg-1",
				strings.NewReader(string(body)),
			)
			if err != nil {
				t.Fatalf("http.NewRequestWithContext() error = %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+identity.SignedToken(t, cfg.appID, cfg.serviceURL))
			if err := verifyTeamsAuthorization(context.Background(), req, body, cfg); err != nil {
				t.Fatalf("verifyTeamsAuthorization() error = %v", err)
			}
		}

		metadataCalls, jwksCalls := identity.Counts()
		if metadataCalls != 1 || jwksCalls != 1 {
			t.Fatalf("identity calls = metadata:%d jwks:%d, want 1/1", metadataCalls, jwksCalls)
		}
	})
}

func TestTeamsContractDeliveryReferences(t *testing.T) {
	t.Run("Should edit the explicitly referenced remote message before current state", func(t *testing.T) {
		api := &fakeTeamsAPI{nextActivityID: 900}
		targetRemote := testTeamsRemoteMessageID("conversation-a", "https://service.test", "activity-a")
		currentRemote := testTeamsRemoteMessageID("conversation-b", "https://service.test", "activity-b")
		request := bridgepkg.DeliveryRequest{
			Event: bridgepkg.DeliveryEvent{
				DeliveryID:       "delivery-edit",
				BridgeInstanceID: "brg-teams",
				RoutingKey:       testTeamsRoutingKey(),
				DeliveryTarget:   testTeamsDeliveryTarget(),
				Seq:              2,
				EventType:        bridgepkg.DeliveryEventTypeFinal,
				Final:            true,
				Operation:        bridgepkg.DeliveryOperationEdit,
				Reference: &bridgepkg.DeliveryMessageReference{
					RemoteMessageID: targetRemote,
				},
				Content: bridgepkg.MessageContent{Text: "updated"},
			},
		}

		_, _, err := executeTeamsDelivery(
			context.Background(),
			api,
			resolvedInstanceConfig{instanceID: "brg-teams"},
			request,
			deliveryState{LastSeq: 1, RemoteMessageID: currentRemote},
			nil,
			func(string, string) (teamsUserContext, bool) {
				return teamsUserContext{}, false
			},
		)
		if err != nil {
			t.Fatalf("executeTeamsDelivery(edit) error = %v", err)
		}
		if got, want := len(api.updateCalls), 1; got != want {
			t.Fatalf("len(api.updateCalls) = %d, want %d", got, want)
		}
		if got, want := api.updateCalls[0].ActivityID, "activity-a"; got != want {
			t.Fatalf("api.updateCalls[0].ActivityID = %q, want %q", got, want)
		}
	})

	t.Run("Should delete a delivery-id reference from stored state", func(t *testing.T) {
		api := &fakeTeamsAPI{nextActivityID: 901}
		referencedRemote := testTeamsRemoteMessageID("conversation-ref", "https://service.test", "activity-ref")
		request := bridgepkg.DeliveryRequest{
			Event: bridgepkg.DeliveryEvent{
				DeliveryID:       "delivery-delete",
				BridgeInstanceID: "brg-teams",
				RoutingKey:       testTeamsRoutingKey(),
				DeliveryTarget:   testTeamsDeliveryTarget(),
				Seq:              2,
				Final:            true,
				EventType:        bridgepkg.DeliveryEventTypeDelete,
				Operation:        bridgepkg.DeliveryOperationDelete,
				Reference: &bridgepkg.DeliveryMessageReference{
					DeliveryID: "delivery-original",
				},
			},
		}

		_, _, err := executeTeamsDelivery(
			context.Background(),
			api,
			resolvedInstanceConfig{instanceID: "brg-teams"},
			request,
			deliveryState{LastSeq: 1},
			func(deliveryID string) (deliveryState, bool) {
				if deliveryID != "delivery-original" {
					return deliveryState{}, false
				}
				return deliveryState{RemoteMessageID: referencedRemote}, true
			},
			func(string, string) (teamsUserContext, bool) {
				return teamsUserContext{}, false
			},
		)
		if err != nil {
			t.Fatalf("executeTeamsDelivery(delete) error = %v", err)
		}
		if got, want := len(api.deleteCalls), 1; got != want {
			t.Fatalf("len(api.deleteCalls) = %d, want %d", got, want)
		}
		if got, want := api.deleteCalls[0].ActivityID, "activity-ref"; got != want {
			t.Fatalf("api.deleteCalls[0].ActivityID = %q, want %q", got, want)
		}
	})
}

type countingTeamsIdentityServer struct {
	server     *httptest.Server
	privateKey *rsa.PrivateKey
	keyID      string

	mu            sync.Mutex
	metadataCalls int
	jwksCalls     int
}

func newCountingTeamsIdentityServer(t *testing.T) *countingTeamsIdentityServer {
	t.Helper()
	enableTeamsLoopbackCredentialedURLsForTesting(t)

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	identity := &countingTeamsIdentityServer{privateKey: privateKey, keyID: "teams-cache-key"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/openid/.well-known/openidconfiguration":
			identity.mu.Lock()
			identity.metadataCalls++
			identity.mu.Unlock()
			if err := json.NewEncoder(w).Encode(map[string]any{
				"issuer":   "https://api.botframework.com",
				"jwks_uri": identity.server.URL + "/openid/keys",
			}); err != nil {
				t.Errorf("encode Teams OpenID metadata error = %v", err)
			}
		case r.Method == http.MethodGet && r.URL.Path == "/openid/keys":
			identity.mu.Lock()
			identity.jwksCalls++
			identity.mu.Unlock()
			pub := privateKey.PublicKey
			if err := json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]any{{
					"kty":          "RSA",
					"kid":          identity.keyID,
					"x5t":          identity.keyID,
					"n":            base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
					"e":            base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
					"endorsements": []string{"msteams"},
				}},
			}); err != nil {
				t.Errorf("encode Teams JWKS error = %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	identity.server = server
	t.Cleanup(server.Close)
	return identity
}

func (s *countingTeamsIdentityServer) ServiceURL() string {
	return s.server.URL
}

func (s *countingTeamsIdentityServer) MetadataURL() string {
	return s.server.URL + "/openid/.well-known/openidconfiguration"
}

func (s *countingTeamsIdentityServer) Counts() (int, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.metadataCalls, s.jwksCalls
}

func (s *countingTeamsIdentityServer) SignedToken(t *testing.T, appID string, serviceURL string) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, teamsAuthClaims{
		ServiceURL: serviceURL,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://api.botframework.com",
			Audience:  jwt.ClaimStrings{appID},
			Subject:   "botframework",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	token.Header["kid"] = s.keyID
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		t.Fatalf("token.SignedString() error = %v", err)
	}
	return signed
}

func testTeamsRemoteMessageID(conversationID string, serviceURL string, activityID string) string {
	return encodeRemoteMessageID(teamsRemoteMessageRef{
		ConversationID: conversationID,
		ServiceURL:     serviceURL,
		ActivityID:     activityID,
	})
}

func testTeamsRoutingKey() bridgepkg.RoutingKey {
	return bridgepkg.RoutingKey{
		Scope:            bridgepkg.ScopeWorkspace,
		WorkspaceID:      "ws-teams",
		BridgeInstanceID: "brg-teams",
		GroupID:          "19:channel@thread.tacv2",
	}
}

func testTeamsDeliveryTarget() bridgepkg.DeliveryTarget {
	return bridgepkg.DeliveryTarget{
		BridgeInstanceID: "brg-teams",
		GroupID:          "19:channel@thread.tacv2",
		Mode:             bridgepkg.DeliveryModeReply,
	}
}
