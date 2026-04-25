package daemon

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
)

func TestSettingsRuntimeSurfaceTransportParityStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		host                   string
		wantHTTPMutationParity bool
	}{
		{
			name:                   "loopback ipv4",
			host:                   "127.0.0.1",
			wantHTTPMutationParity: true,
		},
		{
			name:                   "localhost",
			host:                   "localhost",
			wantHTTPMutationParity: true,
		},
		{
			name:                   "wildcard ipv4",
			host:                   "0.0.0.0",
			wantHTTPMutationParity: false,
		},
		{
			name:                   "non loopback",
			host:                   "192.168.1.25",
			wantHTTPMutationParity: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			surface := &settingsRuntimeSurface{
				config: aghconfig.Config{
					HTTP: aghconfig.HTTPConfig{Host: tc.host},
				},
			}

			status, err := surface.TransportParityStatus(context.Background())
			if err != nil {
				t.Fatalf("TransportParityStatus() error = %v", err)
			}

			want := settingspkg.TransportParityStatus{
				Known:          true,
				SettingsHTTP:   tc.wantHTTPMutationParity,
				SettingsUDS:    true,
				ExtensionsHTTP: tc.wantHTTPMutationParity,
				ExtensionsUDS:  true,
			}
			if status != want {
				t.Fatalf("TransportParityStatus() = %#v, want %#v", status, want)
			}
		})
	}
}

func TestSettingsRuntimeSurfaceMCPAuthStatusSurvivesStoreReopen(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	path := filepath.Join(t.TempDir(), store.GlobalDatabaseName)
	first, err := globaldb.OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB(first) error = %v", err)
	}

	expiresAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	if err := first.SaveMCPAuthToken(ctx, mcpauth.TokenRecord{
		ServerName:   "remote-docs",
		Issuer:       "https://issuer.example.com",
		ClientID:     "agh-cli",
		Scopes:       []string{"mcp.read", "mcp.write"},
		AccessToken:  "access-secret",
		RefreshToken: "refresh-secret",
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		ObtainedAt:   expiresAt.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("SaveMCPAuthToken() error = %v", err)
	}
	if err := first.Close(ctx); err != nil {
		t.Fatalf("Close(first) error = %v", err)
	}

	reopened, err := globaldb.OpenGlobalDB(ctx, path)
	if err != nil {
		t.Fatalf("OpenGlobalDB(reopen) error = %v", err)
	}
	defer func() {
		if err := reopened.Close(ctx); err != nil {
			t.Fatalf("Close(reopened) error = %v", err)
		}
	}()

	surface := &settingsRuntimeSurface{mcpAuthStore: reopened}
	status, err := surface.MCPAuthStatus(ctx, aghconfig.MCPServer{
		Name:      "remote-docs",
		Transport: aghconfig.MCPServerTransportHTTP,
		URL:       "https://mcp.example.com",
		Auth: aghconfig.MCPAuthConfig{
			Type:             aghconfig.MCPAuthTypeOAuth2PKCE,
			ClientID:         "agh-cli",
			AuthorizationURL: "https://issuer.example.com/oauth/authorize",
			TokenURL:         "https://issuer.example.com/oauth/token",
			Scopes:           []string{"mcp.read", "mcp.write"},
		},
	})
	if err != nil {
		t.Fatalf("MCPAuthStatus() error = %v", err)
	}
	if status.Status != mcpauth.StatusAuthenticated {
		t.Fatalf("MCPAuthStatus().Status = %q, want %q", status.Status, mcpauth.StatusAuthenticated)
	}
	if !status.TokenPresent || !status.Refreshable {
		t.Fatalf("MCPAuthStatus() = %#v, want token present and refreshable", status)
	}
	if status.ExpiresAt == nil || !status.ExpiresAt.Equal(expiresAt) {
		t.Fatalf("MCPAuthStatus().ExpiresAt = %v, want %v", status.ExpiresAt, expiresAt)
	}
}
