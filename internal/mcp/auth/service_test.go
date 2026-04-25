package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	handlerErrors := newHandlerErrorRecorder()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case metadataWellKnownPath:
			if err := writeJSON(w, Metadata{
				Issuer:                        "https://issuer.example",
				AuthorizationEndpoint:         "http://" + r.Host + "/authorize",
				TokenEndpoint:                 "http://" + r.Host + "/token",
				RevocationEndpoint:            "http://" + r.Host + "/revoke",
				CodeChallengeMethodsSupported: []string{"S256"},
			}); err != nil {
				handlerErrors.record(err)
			}
		case "/token":
			if err := r.ParseForm(); err != nil {
				handlerErrors.record(fmt.Errorf("ParseForm() error = %w", err))
				http.Error(w, "parse form", http.StatusBadRequest)
				return
			}
			if r.Form.Get("code_verifier") == "auth-code" || strings.Contains(r.Form.Encode(), "access-token") {
				handlerErrors.record(
					fmt.Errorf("token request leaked sensitive values in unexpected field: %s", r.Form.Encode()),
				)
				http.Error(w, "token leak", http.StatusBadRequest)
				return
			}
			switch r.Form.Get("grant_type") {
			case "authorization_code":
				if r.Form.Get("code") != "auth-code" {
					handlerErrors.record(fmt.Errorf("authorization code = %q, want auth-code", r.Form.Get("code")))
					http.Error(w, "bad code", http.StatusBadRequest)
					return
				}
				if r.Form.Get("code_verifier") == "" {
					handlerErrors.record(errors.New("code_verifier = empty"))
					http.Error(w, "missing verifier", http.StatusBadRequest)
					return
				}
				if err := writeJSON(w, map[string]any{
					"access_token":  "access-token-1",
					"refresh_token": "refresh-token-1",
					"token_type":    "Bearer",
					"expires_in":    3600,
					"scope":         "read write",
				}); err != nil {
					handlerErrors.record(err)
				}
			case "refresh_token":
				if r.Form.Get("refresh_token") != "refresh-token-1" {
					handlerErrors.record(
						fmt.Errorf("refresh_token = %q, want persisted token", r.Form.Get("refresh_token")),
					)
					http.Error(w, "bad refresh token", http.StatusBadRequest)
					return
				}
				mu.Lock()
				refreshCalled = true
				mu.Unlock()
				if err := writeJSON(w, map[string]any{
					"access_token": "access-token-2",
					"token_type":   "Bearer",
					"expires_in":   7200,
				}); err != nil {
					handlerErrors.record(err)
				}
			default:
				handlerErrors.record(fmt.Errorf("grant_type = %q", r.Form.Get("grant_type")))
				http.Error(w, "bad grant type", http.StatusBadRequest)
			}
		case "/revoke":
			if err := r.ParseForm(); err != nil {
				handlerErrors.record(fmt.Errorf("ParseForm(revoke) error = %w", err))
				http.Error(w, "parse revoke form", http.StatusBadRequest)
				return
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
	handlerErrors.assertEmpty(t)
}

func TestTokenResponseRejectsMalformedPayload(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newMemoryTokenStore()
	handlerErrors := newHandlerErrorRecorder()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := writeJSON(w, map[string]any{"refresh_token": "refresh"}); err != nil {
				handlerErrors.record(err)
			}
		default:
			if err := writeJSON(w, Metadata{
				AuthorizationEndpoint:         "http://" + r.Host + "/authorize",
				TokenEndpoint:                 "http://" + r.Host + "/token",
				CodeChallengeMethodsSupported: []string{"S256"},
			}); err != nil {
				handlerErrors.record(err)
			}
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
	handlerErrors.assertEmpty(t)
}

func TestLogoutDeletesLocalTokenWhenRemoteRevocationFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newMemoryTokenStore()
	if err := store.SaveMCPAuthToken(ctx, TokenRecord{
		ServerName:   "linear",
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		ObtainedAt:   time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("SaveMCPAuthToken() error = %v", err)
	}
	handlerErrors := newHandlerErrorRecorder()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case metadataWellKnownPath:
			if err := writeJSON(w, Metadata{
				AuthorizationEndpoint:         "http://" + r.Host + "/authorize",
				TokenEndpoint:                 "http://" + r.Host + "/token",
				RevocationEndpoint:            "http://" + r.Host + "/revoke",
				CodeChallengeMethodsSupported: []string{"S256"},
			}); err != nil {
				handlerErrors.record(err)
			}
		case "/revoke":
			http.Error(w, "revocation unavailable", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service, err := NewService(store, WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	cfg := ServerConfig{
		ServerName: "linear",
		RemoteURL:  "https://mcp.example/sse",
		Type:       "oauth2_pkce",
		IssuerURL:  server.URL,
		ClientID:   "client-1",
	}

	status, err := service.Logout(ctx, cfg)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if status.Status != StatusNeedsLogin || !strings.Contains(status.Diagnostic, "remote revocation failed") {
		t.Fatalf("Logout() status = %#v, want local logout diagnostic", status)
	}
	if _, err := store.GetMCPAuthToken(ctx, "linear"); !errors.Is(err, ErrTokenNotFound) {
		t.Fatalf("GetMCPAuthToken(after failed revocation logout) error = %v, want ErrTokenNotFound", err)
	}
	handlerErrors.assertEmpty(t)
}

func TestSupportsS256RequiresAdvertisedMethod(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		methods []string
		want    bool
	}{
		{name: "Should reject missing metadata methods", methods: nil, want: false},
		{name: "Should reject non S256 methods", methods: []string{"plain"}, want: false},
		{name: "Should accept advertised S256 method", methods: []string{" plain ", " s256 "}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := supportsS256(tc.methods); got != tc.want {
				t.Fatalf("supportsS256(%#v) = %v, want %v", tc.methods, got, tc.want)
			}
		})
	}
}

func TestNewServiceDefaultsHTTPClientTimeout(t *testing.T) {
	t.Parallel()

	t.Run("Should configure bounded HTTP client by default", func(t *testing.T) {
		t.Parallel()

		service, err := NewService(newMemoryTokenStore())
		if err != nil {
			t.Fatalf("NewService() error = %v", err)
		}
		if service.client == nil || service.client.Timeout != defaultMetadataClientTimeout {
			t.Fatalf("NewService().client = %#v, want timeout %s", service.client, defaultMetadataClientTimeout)
		}
	})
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

type handlerErrorRecorder struct {
	mu   sync.Mutex
	errs []error
}

func newHandlerErrorRecorder() *handlerErrorRecorder {
	return &handlerErrorRecorder{}
}

func (r *handlerErrorRecorder) record(err error) {
	if err == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errs = append(r.errs, err)
}

func (r *handlerErrorRecorder) assertEmpty(t *testing.T) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.errs) > 0 {
		t.Fatalf("handler errors = %v", errors.Join(r.errs...))
	}
}

func writeJSON(w http.ResponseWriter, value any) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return fmt.Errorf("json.Encode() error = %w", err)
	}
	return nil
}
