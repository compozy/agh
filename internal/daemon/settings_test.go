package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"
	core "github.com/pedronauck/agh/internal/api/core"
	aghconfig "github.com/pedronauck/agh/internal/config"
	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	settingspkg "github.com/pedronauck/agh/internal/settings"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	aghupdate "github.com/pedronauck/agh/internal/update"
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
	t.Run("Should preserve MCP auth status after reopening the backing store", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		path := filepath.Join(t.TempDir(), store.GlobalDatabaseName)
		first, err := globaldb.OpenGlobalDB(ctx, path)
		if err != nil {
			t.Fatalf("OpenGlobalDB(first) error = %v", err)
		}

		expiresAt := time.Date(2126, 5, 1, 12, 0, 0, 0, time.UTC)
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
	})
}

func TestSettingsRuntimeSurfaceMCPAuthStatusResolvesClientSecretRef(t *testing.T) {
	t.Run("Should resolve MCP client_secret_ref before computing auth status", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		called := false
		surface := &settingsRuntimeSurface{
			secretResolver: func(_ context.Context, ref string) (string, error) {
				called = true
				if ref != "vault:mcp/remote-docs/oauth/client-secret" {
					t.Fatalf("secret resolver ref = %q, want remote-docs client secret ref", ref)
				}
				return "client-secret", nil
			},
		}

		status, err := surface.MCPAuthStatus(ctx, aghconfig.MCPServer{
			Name:      "remote-docs",
			Transport: aghconfig.MCPServerTransportHTTP,
			URL:       "https://mcp.example.com",
			Auth: aghconfig.MCPAuthConfig{
				Type:             aghconfig.MCPAuthTypeOAuth2PKCE,
				ClientID:         "agh-cli",
				ClientSecretRef:  "vault:mcp/remote-docs/oauth/client-secret",
				AuthorizationURL: "https://issuer.example.com/oauth/authorize",
				TokenURL:         "https://issuer.example.com/oauth/token",
			},
		})
		if err != nil {
			t.Fatalf("MCPAuthStatus() error = %v", err)
		}
		if !called {
			t.Fatal("MCPAuthStatus() did not resolve client_secret_ref")
		}
		if got, want := status.Status, mcpauth.StatusNeedsLogin; got != want {
			t.Fatalf("MCPAuthStatus().Status = %q, want %q", got, want)
		}
	})
}

func TestSettingsRuntimeSurfaceMCPServerRuntimeStatus(t *testing.T) {
	t.Run("Should default MCP runtime probe timeout to five seconds", func(t *testing.T) {
		t.Parallel()

		surface := &settingsRuntimeSurface{}
		if got, want := surface.mcpProbeTimeout(), 5*time.Second; got != want {
			t.Fatalf("mcpProbeTimeout() = %s, want %s", got, want)
		}
	})

	t.Run("Should use the configured observability probe timeout for MCP runtime probes", func(t *testing.T) {
		t.Parallel()

		surface := &settingsRuntimeSurface{
			config: aghconfig.Config{
				Observability: aghconfig.ObservabilityConfig{
					AgentProbeTimeout: 9 * time.Second,
				},
			},
		}
		if got, want := surface.mcpProbeTimeout(), 9*time.Second; got != want {
			t.Fatalf("mcpProbeTimeout() = %s, want %s", got, want)
		}
	})

	t.Run("Should probe a reachable MCP server through the real executor", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		server := mcpsrv.NewTestStreamableHTTPServer(newSettingsMCPTestServer())
		t.Cleanup(server.Close)

		surface := &settingsRuntimeSurface{}
		status, err := surface.MCPServerRuntimeStatus(ctx, aghconfig.MCPServer{
			Name:      "docs",
			Transport: aghconfig.MCPServerTransportHTTP,
			URL:       server.URL,
		})
		if err != nil {
			t.Fatalf("MCPServerRuntimeStatus() error = %v", err)
		}
		if got, want := status.State, settingspkg.MCPServerRuntimeStateReady; got != want {
			t.Fatalf("MCPServerRuntimeStatus().State = %q, want %q", got, want)
		}
		if got, want := status.Probe, settingspkg.MCPServerProbeSucceeded; got != want {
			t.Fatalf("MCPServerRuntimeStatus().Probe = %q, want %q", got, want)
		}
		if !status.Initialized || status.ToolCount != 1 {
			t.Fatalf("MCPServerRuntimeStatus() = %#v, want initialized with one tool", status)
		}
	})

	t.Run("Should skip probing when remote MCP auth needs login", func(t *testing.T) {
		t.Parallel()

		surface := &settingsRuntimeSurface{}
		status, err := surface.MCPServerRuntimeStatus(context.Background(), aghconfig.MCPServer{
			Name:      "linear",
			Transport: aghconfig.MCPServerTransportHTTP,
			URL:       "https://mcp.linear.example/mcp",
			Auth: aghconfig.MCPAuthConfig{
				Type:             aghconfig.MCPAuthTypeOAuth2PKCE,
				AuthorizationURL: "https://auth.linear.example/authorize",
				TokenURL:         "https://auth.linear.example/token",
				ClientID:         "agh-desktop",
			},
		})
		if err != nil {
			t.Fatalf("MCPServerRuntimeStatus(auth) error = %v", err)
		}
		if got, want := status.State, settingspkg.MCPServerRuntimeStateAuthRequired; got != want {
			t.Fatalf("MCPServerRuntimeStatus(auth).State = %q, want %q", got, want)
		}
		if got, want := status.Probe, settingspkg.MCPServerProbeSkipped; got != want {
			t.Fatalf("MCPServerRuntimeStatus(auth).Probe = %q, want %q", got, want)
		}
		if status.Initialized || status.ToolCount != 0 {
			t.Fatalf("MCPServerRuntimeStatus(auth) = %#v, want no initialization or tools", status)
		}
	})

	t.Run("Should report config errors without fabricating a probe", func(t *testing.T) {
		t.Parallel()

		surface := &settingsRuntimeSurface{}
		status, err := surface.MCPServerRuntimeStatus(context.Background(), aghconfig.MCPServer{
			Name:      "broken",
			Transport: aghconfig.MCPServerTransportHTTP,
		})
		if err != nil {
			t.Fatalf("MCPServerRuntimeStatus(config error) error = %v", err)
		}
		if got, want := status.State, settingspkg.MCPServerRuntimeStateConfigError; got != want {
			t.Fatalf("MCPServerRuntimeStatus(config error).State = %q, want %q", got, want)
		}
		if got, want := status.Probe, settingspkg.MCPServerProbeSkipped; got != want {
			t.Fatalf("MCPServerRuntimeStatus(config error).Probe = %q, want %q", got, want)
		}
		if status.Diagnostic == "" {
			t.Fatal("MCPServerRuntimeStatus(config error).Diagnostic is empty")
		}
	})
}

func newSettingsMCPTestServer() *mcpsrv.MCPServer {
	server := mcpsrv.NewMCPServer("settings-test", "1.0.0", mcpsrv.WithToolCapabilities(true))
	server.AddTool(
		mcpsdk.NewTool(
			"lookup",
			mcpsdk.WithDescription("Lookup documentation"),
			mcpsdk.WithString("query"),
			mcpsdk.WithRawOutputSchema(json.RawMessage(
				"{\"type\":\"object\",\"properties\":{\"answer\":{\"type\":\"string\"}}}",
			)),
			mcpsdk.WithReadOnlyHintAnnotation(true),
		),
		func(context.Context, mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
			return mcpsdk.NewToolResultText("ok"), nil
		},
	)
	return server
}

type stubSettingsUpdateManager struct {
	checkFn func(context.Context, aghupdate.CheckOptions) (aghupdate.State, *aghupdate.Release, error)
}

func (s stubSettingsUpdateManager) Check(
	ctx context.Context,
	opts aghupdate.CheckOptions,
) (aghupdate.State, *aghupdate.Release, error) {
	if s.checkFn != nil {
		return s.checkFn(ctx, opts)
	}
	return aghupdate.State{}, nil, nil
}

func TestSettingsUpdateControllerGetUpdate(t *testing.T) {
	t.Run("Should translate the cached update snapshot from the shared manager", func(t *testing.T) {
		t.Parallel()

		checkedAt := time.Date(2026, 5, 3, 19, 0, 0, 0, time.UTC)
		controller := settingsUpdateController{
			manager: stubSettingsUpdateManager{
				checkFn: func(_ context.Context, opts aghupdate.CheckOptions) (aghupdate.State, *aghupdate.Release, error) {
					if opts.ForceRefresh {
						t.Fatal("CheckOptions.ForceRefresh = true, want false")
					}
					if !opts.AllowCachedOnFailure {
						t.Fatal("CheckOptions.AllowCachedOnFailure = false, want true")
					}
					return aghupdate.State{
						Supported:      true,
						Managed:        false,
						InstallMethod:  string(aghupdate.InstallMethodDirectBinary),
						CurrentVersion: "v1.0.0",
						LatestVersion:  "v1.1.0",
						Available:      true,
						Status:         aghupdate.StatusAvailable,
						Recommendation: "Run agh update.",
						ReleaseURL:     "https://github.com/compozy/agh/releases/tag/v1.1.0",
						CheckedAt:      &checkedAt,
						LastError:      "cached upstream failure",
					}, &aghupdate.Release{Version: "v1.1.0"}, nil
				},
			},
		}

		got, err := controller.GetUpdate(context.Background())
		if err != nil {
			t.Fatalf("GetUpdate() error = %v", err)
		}

		want := core.SettingsUpdateStatus{
			Supported:      true,
			Managed:        false,
			InstallMethod:  string(aghupdate.InstallMethodDirectBinary),
			CurrentVersion: "v1.0.0",
			LatestVersion:  "v1.1.0",
			Available:      true,
			Status:         string(aghupdate.StatusAvailable),
			Recommendation: "Run agh update.",
			ReleaseURL:     "https://github.com/compozy/agh/releases/tag/v1.1.0",
			CheckedAt:      &checkedAt,
			LastError:      "cached upstream failure",
		}
		if got != want {
			t.Fatalf("GetUpdate() = %#v, want %#v", got, want)
		}
	})

	t.Run("Should reject a missing settings update manager", func(t *testing.T) {
		t.Parallel()

		_, err := (settingsUpdateController{}).GetUpdate(context.Background())
		if err == nil {
			t.Fatal("GetUpdate() error = nil, want missing manager error")
		}
	})

	t.Run("Should surface raw manager errors when no state message is available", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("upstream unavailable")
		controller := settingsUpdateController{
			manager: stubSettingsUpdateManager{
				checkFn: func(context.Context, aghupdate.CheckOptions) (aghupdate.State, *aghupdate.Release, error) {
					return aghupdate.State{}, nil, wantErr
				},
			},
		}

		_, err := controller.GetUpdate(context.Background())
		if !errors.Is(err, wantErr) {
			t.Fatalf("GetUpdate() error = %v, want %v", err, wantErr)
		}
	})
}
