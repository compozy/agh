package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOAuthPKCELifecycleWithRefreshAndLogout(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
	store := newMemoryTokenStore()
	var (
		mu            sync.Mutex
		refreshCalled bool
		revokedToken  string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case metadataWellKnownPath:
			writeJSON(t, w, Metadata{
				Issuer:                        "https://issuer.example",
				AuthorizationEndpoint:         "http://" + r.Host + "/authorize",
				TokenEndpoint:                 "http://" + r.Host + "/token",
				RevocationEndpoint:            "http://" + r.Host + "/revoke",
				CodeChallengeMethodsSupported: []string{"S256"},
			})
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm() error = %v", err)
			}
			if r.Form.Get("code_verifier") == "auth-code" || strings.Contains(r.Form.Encode(), "access-token") {
				t.Fatalf("token request leaked sensitive values in unexpected field: %s", r.Form.Encode())
			}
			switch r.Form.Get("grant_type") {
			case "authorization_code":
				if r.Form.Get("code") != "auth-code" {
					t.Fatalf("authorization code = %q, want auth-code", r.Form.Get("code"))
				}
				if r.Form.Get("code_verifier") == "" {
					t.Fatal("code_verifier = empty")
				}
				writeJSON(t, w, map[string]any{
					"access_token":  "access-token-1",
					"refresh_token": "refresh-token-1",
					"token_type":    "Bearer",
					"expires_in":    3600,
					"scope":         "read write",
				})
			case "refresh_token":
				if r.Form.Get("refresh_token") != "refresh-token-1" {
					t.Fatalf("refresh_token = %q, want persisted token", r.Form.Get("refresh_token"))
				}
				mu.Lock()
				refreshCalled = true
				mu.Unlock()
				writeJSON(t, w, map[string]any{
					"access_token": "access-token-2",
					"token_type":   "Bearer",
					"expires_in":   7200,
				})
			default:
				t.Fatalf("grant_type = %q", r.Form.Get("grant_type"))
			}
		case "/revoke":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm(revoke) error = %v", err)
			}
			mu.Lock()
			revokedToken = r.Form.Get("token")
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service, err := NewService(store, WithHTTPClient(server.Client()), WithNow(func() time.Time { return now }))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	cfg := ServerConfig{
		ServerName:    "linear",
		RemoteURL:     "https://mcp.example/sse",
		Type:          "oauth2_pkce",
		IssuerURL:     server.URL,
		ClientID:      "client-1",
		Scopes:        []string{"read", "write"},
		RevocationURL: server.URL + "/revoke",
	}

	login, err := service.BeginLogin(ctx, cfg, "http://127.0.0.1/callback")
	if err != nil {
		t.Fatalf("BeginLogin() error = %v", err)
	}
	authURL, err := url.Parse(login.AuthorizationURL)
	if err != nil {
		t.Fatalf("Parse authorization URL error = %v", err)
	}
	if authURL.Query().Get("code_challenge") == "" || authURL.Query().Get("code_verifier") != "" {
		t.Fatalf("authorization URL query = %s, want challenge without verifier", authURL.RawQuery)
	}
	if _, err := service.Exchange(ctx, login, "http://127.0.0.1/callback?code=auth-code&state=wrong"); err == nil {
		t.Fatal("Exchange(wrong state) error = nil, want mismatch")
	}

	status, err := service.Exchange(ctx, login, "http://127.0.0.1/callback?code=auth-code&state="+login.State)
	if err != nil {
		t.Fatalf("Exchange() error = %v", err)
	}
	if status.Status != StatusAuthenticated || !status.Refreshable || !status.TokenPresent {
		t.Fatalf("Exchange() status = %#v", status)
	}
	statusJSON, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("json.Marshal(status) error = %v", err)
	}
	if strings.Contains(string(statusJSON), "access-token") || strings.Contains(string(statusJSON), "refresh-token") {
		t.Fatalf("redacted status leaked token material: %s", statusJSON)
	}

	refreshed, err := service.Refresh(ctx, cfg)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.Status != StatusAuthenticated {
		t.Fatalf("Refresh() status = %#v", refreshed)
	}
	mu.Lock()
	if !refreshCalled {
		t.Fatal("refresh endpoint was not called")
	}
	mu.Unlock()
	token, err := store.GetMCPAuthToken(ctx, "linear")
	if err != nil {
		t.Fatalf("GetMCPAuthToken() error = %v", err)
	}
	if token.AccessToken != "access-token-2" || token.RefreshToken != "refresh-token-1" {
		t.Fatalf("stored token after refresh = %#v", token)
	}

	loggedOut, err := service.Logout(ctx, cfg)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if loggedOut.Status != StatusNeedsLogin {
		t.Fatalf("Logout() status = %#v", loggedOut)
	}
	mu.Lock()
	if revokedToken != "refresh-token-1" {
		t.Fatalf("revoked token = %q, want refresh token", revokedToken)
	}
	mu.Unlock()
	if _, err := store.GetMCPAuthToken(ctx, "linear"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetMCPAuthToken(after logout) error = %v, want ErrTokenNotFound", err)
	}
}

func TestTokenResponseRejectsMalformedPayload(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newMemoryTokenStore()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			writeJSON(t, w, map[string]any{"refresh_token": "refresh"})
		default:
			writeJSON(t, w, Metadata{
				AuthorizationEndpoint: "http://" + r.Host + "/authorize",
				TokenEndpoint:         "http://" + r.Host + "/token",
			})
		}
	}))
	defer server.Close()

	service, err := NewService(store, WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	cfg := ServerConfig{
		ServerName:  "bad",
		Type:        "oauth2_pkce",
		MetadataURL: server.URL,
		ClientID:    "client",
	}
	login, err := service.BeginLogin(ctx, cfg, "http://127.0.0.1/callback")
	if err != nil {
		t.Fatalf("BeginLogin() error = %v", err)
	}
	_, err = service.Exchange(ctx, login, "http://127.0.0.1/callback?code=ok&state="+login.State)
	if err == nil || !strings.Contains(err.Error(), "access_token is required") {
		t.Fatalf("Exchange() error = %v, want malformed token response", err)
	}
}

type memoryTokenStore struct {
	mu     sync.Mutex
	tokens map[string]TokenRecord
}

func newMemoryTokenStore() *memoryTokenStore {
	return &memoryTokenStore{tokens: map[string]TokenRecord{}}
}

func (s *memoryTokenStore) SaveMCPAuthToken(_ context.Context, token TokenRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token.ServerName] = token
	return nil
}

func (s *memoryTokenStore) GetMCPAuthToken(_ context.Context, serverName string) (TokenRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, ok := s.tokens[serverName]
	if !ok {
		return TokenRecord{}, ErrTokenNotFound
	}
	return token, nil
}

func (s *memoryTokenStore) ListMCPAuthTokens(context.Context) ([]TokenRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tokens := make([]TokenRecord, 0, len(s.tokens))
	for _, token := range s.tokens {
		tokens = append(tokens, token)
	}
	return tokens, nil
}

func (s *memoryTokenStore) DeleteMCPAuthToken(_ context.Context, serverName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, serverName)
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("json.Encode() error = %v", err)
	}
}
