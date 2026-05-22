package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/diagnostics"
	mcpauth "github.com/compozy/agh/internal/mcp/auth"
)

func TestMCPAuthStatusReportsRedactedState(t *testing.T) {
	t.Parallel()

	t.Run("Should report redacted auth status", func(t *testing.T) {
		t.Parallel()

		deps := newMCPAuthTestDeps(t, &stubMCPAuthClient{
			statusFn: func(_ context.Context, cfg mcpauth.ServerConfig) (mcpauth.Status, error) {
				return mcpauth.Status{
					ServerName:   cfg.ServerName,
					Status:       mcpauth.StatusAuthenticated,
					ClientID:     cfg.ClientID,
					Scopes:       []string{"read"},
					TokenPresent: true,
					Refreshable:  true,
				}, nil
			},
		})

		stdout, _, err := executeRootCommand(t, deps, "mcp", "auth", "status", "linear", "-o", "json")
		if err != nil {
			t.Fatalf("executeRootCommand(mcp auth status) error = %v", err)
		}
		if strings.Contains(stdout, "access-token") || strings.Contains(stdout, "refresh-token") ||
			strings.Contains(stdout, "client-secret") {
			t.Fatalf("status output leaked secret material: %s", stdout)
		}
		var statuses []mcpauth.Status
		if err := json.Unmarshal([]byte(stdout), &statuses); err != nil {
			t.Fatalf("json.Unmarshal(status) error = %v", err)
		}
		if len(statuses) != 1 || statuses[0].Status != mcpauth.StatusAuthenticated {
			t.Fatalf("statuses = %#v", statuses)
		}
	})
}

func TestMCPAuthSecretResolverRegistersDynamicRedaction(t *testing.T) {
	t.Run("Should register resolved env secret for diagnostics redaction", func(t *testing.T) {
		// diagnostics redaction keeps a package-global registry, so this subtest must stay serial.
		secret := "mcp-client-secret-redaction-test"
		resolver := mcpAuthSecretResolver(aghconfig.HomePaths{}, func(key string) string {
			if key == "MCP_CLIENT_SECRET_REDACTION_TEST" {
				return secret
			}
			return ""
		})

		value, err := resolver(context.Background(), "env:MCP_CLIENT_SECRET_REDACTION_TEST")
		if err != nil {
			t.Fatalf("mcpAuthSecretResolver(env) error = %v", err)
		}
		if value != secret {
			t.Fatalf("mcpAuthSecretResolver(env) = %q, want secret", value)
		}
		redacted := diagnostics.Redact("leaked " + secret)
		if strings.Contains(redacted, secret) {
			t.Fatalf("diagnostics.Redact() leaked resolved MCP client secret: %q", redacted)
		}
	})
}

func TestMCPAuthStatusBundlesRenderHumanAndToon(t *testing.T) {
	t.Parallel()

	expiresAt := timePointer(fixedTestNow)
	status := mcpauth.Status{
		ServerName:  "linear",
		Status:      mcpauth.StatusAuthenticated,
		RemoteURL:   "https://mcp.example/sse",
		AuthType:    "oauth2_pkce",
		ClientID:    "client-id",
		Scopes:      []string{"read", "write"},
		ExpiresAt:   expiresAt,
		Refreshable: true,
		Diagnostic:  "ok",
	}

	t.Run("Should render single status in human format", func(t *testing.T) {
		t.Parallel()

		bundle := mcpAuthStatusBundle(status)
		human, err := bundle.human()
		if err != nil {
			t.Fatalf("mcpAuthStatusBundle.human() error = %v", err)
		}
		if !strings.Contains(human, "linear") || !strings.Contains(human, "Refreshable") {
			t.Fatalf("mcp auth status human = %q, want status rows", human)
		}
	})

	t.Run("Should render single status in toon format", func(t *testing.T) {
		t.Parallel()

		bundle := mcpAuthStatusBundle(status)
		toon, err := bundle.toon()
		if err != nil {
			t.Fatalf("mcpAuthStatusBundle.toon() error = %v", err)
		}
		if !strings.Contains(toon, "mcp_auth") || !strings.Contains(toon, "read|write") {
			t.Fatalf("mcp auth status toon = %q, want toon fields", toon)
		}
	})

	t.Run("Should render status list in human format", func(t *testing.T) {
		t.Parallel()

		listBundle := mcpAuthStatusListBundle([]mcpauth.Status{status})
		listHuman, err := listBundle.human()
		if err != nil {
			t.Fatalf("mcpAuthStatusListBundle.human() error = %v", err)
		}
		if !strings.Contains(listHuman, "MCP Auth") || !strings.Contains(listHuman, "linear") {
			t.Fatalf("mcp auth list human = %q, want status table", listHuman)
		}
	})

	t.Run("Should render status list in toon format", func(t *testing.T) {
		t.Parallel()

		listBundle := mcpAuthStatusListBundle([]mcpauth.Status{status})
		listToon, err := listBundle.toon()
		if err != nil {
			t.Fatalf("mcpAuthStatusListBundle.toon() error = %v", err)
		}
		if !strings.Contains(listToon, "mcp_auth[1]") {
			t.Fatalf("mcp auth list toon = %q, want toon table", listToon)
		}
	})
}

func TestMCPAuthLoginManualCodeExchangesWithoutPrintingVerifier(t *testing.T) {
	t.Parallel()

	t.Run("Should start manual login and persist state without printing verifier", func(t *testing.T) {
		t.Parallel()

		deps := newMCPAuthTestDeps(t, &stubMCPAuthClient{
			beginFn: func(_ context.Context, cfg mcpauth.ServerConfig, redirectURL string) (mcpauth.LoginState, error) {
				return mcpauth.LoginState{
					ServerName:       cfg.ServerName,
					RedirectURL:      redirectURL,
					State:            "state-1",
					Verifier:         "sensitive-verifier",
					AuthorizationURL: "https://auth.example/authorize?state=state-1",
					Config:           cfg,
				}, nil
			},
		})
		homePaths, err := deps.resolveHome()
		if err != nil {
			t.Fatalf("deps.resolveHome() error = %v", err)
		}

		stdout, stderr, err := executeRootCommand(
			t,
			deps,
			"mcp",
			"auth",
			"login",
			"linear",
			"--manual",
			"--redirect-url",
			defaultMCPAuthRedirectURL,
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("executeRootCommand(mcp auth login --manual) error = %v", err)
		}
		if strings.Contains(stdout+stderr, "sensitive-verifier") {
			t.Fatalf("manual login output leaked PKCE verifier: stdout=%q stderr=%q", stdout, stderr)
		}
		if !strings.Contains(stderr, "https://auth.example/authorize?state=state-1") {
			t.Fatalf("manual login stderr = %q, want authorization URL", stderr)
		}
		var status mcpauth.Status
		if err := json.Unmarshal([]byte(stdout), &status); err != nil {
			t.Fatalf("json.Unmarshal(login status) error = %v", err)
		}
		if status.Status != mcpauth.StatusNeedsLogin || status.AuthorizationURL == "" {
			t.Fatalf("status = %#v, want pending manual login", status)
		}
		path, err := mcpAuthPendingLoginPath(homePaths, "linear")
		if err != nil {
			t.Fatalf("mcpAuthPendingLoginPath() error = %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("os.Stat(pending login) error = %v", err)
		}
		assertMCPAuthPendingLoginFileMode(t, path, 0o600)
	})

	t.Run("Should atomically replace stale pending login files with private mode", func(t *testing.T) {
		t.Parallel()

		deps := newMCPAuthTestDeps(t, &stubMCPAuthClient{})
		homePaths, err := deps.resolveHome()
		if err != nil {
			t.Fatalf("deps.resolveHome() error = %v", err)
		}
		path, err := mcpAuthPendingLoginPath(homePaths, "linear")
		if err != nil {
			t.Fatalf("mcpAuthPendingLoginPath() error = %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(pending dir) error = %v", err)
		}
		if err := os.Chmod(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("Chmod(pending dir) error = %v", err)
		}
		if err := os.WriteFile(path, []byte("stale verifier"), 0o644); err != nil {
			t.Fatalf("WriteFile(stale pending login) error = %v", err)
		}

		if err := saveMCPAuthPendingLogin(homePaths, mcpauth.LoginState{
			ServerName:       "linear",
			RedirectURL:      defaultMCPAuthRedirectURL,
			State:            "state-replacement",
			Verifier:         "replacement-sensitive-verifier",
			AuthorizationURL: "https://auth.example/authorize?state=state-replacement",
		}); err != nil {
			t.Fatalf("saveMCPAuthPendingLogin() error = %v", err)
		}

		assertMCPAuthPendingLoginFileMode(t, filepath.Dir(path), 0o700)
		assertMCPAuthPendingLoginFileMode(t, path, 0o600)
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(pending login) error = %v", err)
		}
		text := string(payload)
		if strings.Contains(text, "stale verifier") || !strings.Contains(text, "replacement-sensitive-verifier") {
			t.Fatalf("pending login payload = %q, want replacement without stale verifier", text)
		}
	})

	t.Run("Should exchange manual code without printing verifier", func(t *testing.T) {
		t.Parallel()

		deps := newMCPAuthTestDeps(t, &stubMCPAuthClient{
			beginFn: func(_ context.Context, cfg mcpauth.ServerConfig, redirectURL string) (mcpauth.LoginState, error) {
				t.Fatalf("BeginLogin(%q, %q) was called for --manual-code", cfg.ServerName, redirectURL)
				return mcpauth.LoginState{}, nil
			},
			exchangeFn: func(_ context.Context, state mcpauth.LoginState, callbackURL string) (mcpauth.Status, error) {
				if state.State != "state-original" || state.Verifier != "sensitive-verifier" {
					t.Fatalf("state = %#v, want persisted manual login state", state)
				}
				if state.Config.ClientSecret != "client-secret" {
					t.Fatalf("state.Config.ClientSecret = %q, want resolved current secret", state.Config.ClientSecret)
				}
				if !strings.Contains(callbackURL, "code=manual-code") ||
					!strings.Contains(callbackURL, "state="+state.State) {
					t.Fatalf("callbackURL = %q", callbackURL)
				}
				return mcpauth.Status{ServerName: state.ServerName, Status: mcpauth.StatusAuthenticated}, nil
			},
		})
		homePaths, err := deps.resolveHome()
		if err != nil {
			t.Fatalf("deps.resolveHome() error = %v", err)
		}
		if err := saveMCPAuthPendingLogin(homePaths, mcpauth.LoginState{
			ServerName:       "linear",
			RedirectURL:      defaultMCPAuthRedirectURL,
			State:            "state-original",
			Verifier:         "sensitive-verifier",
			AuthorizationURL: "https://auth.example/authorize?state=state-original",
		}); err != nil {
			t.Fatalf("saveMCPAuthPendingLogin() error = %v", err)
		}

		stdout, stderr, err := executeRootCommand(
			t,
			deps,
			"mcp",
			"auth",
			"login",
			"linear",
			"--manual-code",
			"manual-code",
			"-o",
			"json",
		)
		if err != nil {
			t.Fatalf("executeRootCommand(mcp auth login) error = %v", err)
		}
		if strings.Contains(stdout+stderr, "sensitive-verifier") {
			t.Fatalf("login output leaked PKCE verifier: stdout=%q stderr=%q", stdout, stderr)
		}
		var status mcpauth.Status
		if err := json.Unmarshal([]byte(stdout), &status); err != nil {
			t.Fatalf("json.Unmarshal(login status) error = %v", err)
		}
		if status.Status != mcpauth.StatusAuthenticated {
			t.Fatalf("status = %#v", status)
		}
		path, err := mcpAuthPendingLoginPath(homePaths, "linear")
		if err != nil {
			t.Fatalf("mcpAuthPendingLoginPath() error = %v", err)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("pending login file error = %v, want not exists", err)
		}
	})
}

func assertMCPAuthPendingLoginFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("Stat(%q).Mode().Perm() = %#o, want %#o", path, got, want)
	}
}

func TestMCPAuthLogoutCallsAuthClient(t *testing.T) {
	t.Parallel()

	t.Run("Should call auth client logout", func(t *testing.T) {
		t.Parallel()

		called := false
		deps := newMCPAuthTestDeps(t, &stubMCPAuthClient{
			logoutFn: func(_ context.Context, cfg mcpauth.ServerConfig) (mcpauth.Status, error) {
				called = true
				return mcpauth.Status{ServerName: cfg.ServerName, Status: mcpauth.StatusNeedsLogin}, nil
			},
		})

		if _, _, err := executeRootCommand(t, deps, "mcp", "auth", "logout", "linear"); err != nil {
			t.Fatalf("executeRootCommand(mcp auth logout) error = %v", err)
		}
		if !called {
			t.Fatal("Logout was not called")
		}
	})
}

func TestListenForMCPAuthCallbackRequiresLoopbackRedirect(t *testing.T) {
	t.Parallel()

	t.Run("Should reject non-loopback redirect", func(t *testing.T) {
		t.Parallel()

		listener, _, err := listenForMCPAuthCallback(
			context.Background(),
			"http://0.0.0.0:0/callback",
		)
		if listener != nil {
			t.Cleanup(func() {
				if closeErr := listener.Close(); closeErr != nil {
					t.Errorf("listener.Close() error = %v", closeErr)
				}
			})
		}
		if err == nil {
			t.Fatal("listenForMCPAuthCallback(non-loopback) error = nil, want failure")
		}
		if !strings.Contains(err.Error(), "loopback") {
			t.Fatalf("listenForMCPAuthCallback(non-loopback) error = %v, want loopback failure", err)
		}
	})

	t.Run("Should replace zero port with bound listener port", func(t *testing.T) {
		t.Parallel()

		listener, actualRedirectURL, err := listenForMCPAuthCallback(
			context.Background(),
			"http://127.0.0.1:0/callback",
		)
		if err != nil {
			t.Fatalf("listenForMCPAuthCallback(loopback) error = %v", err)
		}
		t.Cleanup(func() {
			if err := listener.Close(); err != nil {
				t.Errorf("listener.Close() error = %v", err)
			}
		})
		if strings.Contains(actualRedirectURL, ":0/") {
			t.Fatalf("actualRedirectURL = %q, want bound listener port", actualRedirectURL)
		}
	})
}

func TestListenForMCPAuthCallbackNormalizesEmptyPath(t *testing.T) {
	t.Parallel()

	t.Run("Should normalize empty path to callback", func(t *testing.T) {
		t.Parallel()

		listener, actualRedirectURL, err := listenForMCPAuthCallback(
			context.Background(),
			"http://127.0.0.1:0",
		)
		if err != nil {
			t.Fatalf("listenForMCPAuthCallback(empty path) error = %v", err)
		}
		t.Cleanup(func() {
			if err := listener.Close(); err != nil {
				t.Errorf("listener.Close() error = %v", err)
			}
		})
		if !strings.HasSuffix(actualRedirectURL, "/callback") {
			t.Fatalf("actualRedirectURL = %q, want /callback path", actualRedirectURL)
		}
	})
}

func newMCPAuthTestDeps(t *testing.T, client *stubMCPAuthClient) commandDeps {
	t.Helper()

	deps := newTestDeps(t, &stubClient{})
	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	cfg := aghconfig.DefaultWithHome(homePaths)
	cfg.MCPServers = []aghconfig.MCPServer{{
		Name:      "linear",
		Transport: aghconfig.MCPServerTransportSSE,
		URL:       "https://mcp.example/sse",
		Auth: aghconfig.MCPAuthConfig{
			Type:             aghconfig.MCPAuthTypeOAuth2PKCE,
			AuthorizationURL: "https://auth.example/authorize",
			TokenURL:         "https://auth.example/token",
			ClientID:         "client-id",
			ClientSecretRef:  "env:LINEAR_CLIENT_SECRET",
			Scopes:           []string{"read"},
		},
	}}
	deps.loadConfig = func() (aghconfig.Config, error) {
		return cfg, nil
	}
	deps.resolveHome = func() (aghconfig.HomePaths, error) {
		return homePaths, nil
	}
	deps.getenv = func(key string) string {
		if key == "LINEAR_CLIENT_SECRET" {
			return "client-secret"
		}
		return ""
	}
	deps.newMCPAuthClient = func(
		context.Context,
		aghconfig.HomePaths,
	) (mcpAuthClient, func(context.Context) error, error) {
		return client, func(context.Context) error { return nil }, nil
	}
	return deps
}

type stubMCPAuthClient struct {
	beginFn    func(context.Context, mcpauth.ServerConfig, string) (mcpauth.LoginState, error)
	exchangeFn func(context.Context, mcpauth.LoginState, string) (mcpauth.Status, error)
	refreshFn  func(context.Context, mcpauth.ServerConfig) (mcpauth.Status, error)
	statusFn   func(context.Context, mcpauth.ServerConfig) (mcpauth.Status, error)
	logoutFn   func(context.Context, mcpauth.ServerConfig) (mcpauth.Status, error)
}

func (s *stubMCPAuthClient) BeginLogin(
	ctx context.Context,
	cfg mcpauth.ServerConfig,
	redirectURL string,
) (mcpauth.LoginState, error) {
	if s.beginFn != nil {
		return s.beginFn(ctx, cfg, redirectURL)
	}
	return mcpauth.LoginState{}, nil
}

func (s *stubMCPAuthClient) Exchange(
	ctx context.Context,
	state mcpauth.LoginState,
	callbackURL string,
) (mcpauth.Status, error) {
	if s.exchangeFn != nil {
		return s.exchangeFn(ctx, state, callbackURL)
	}
	return mcpauth.Status{}, nil
}

func (s *stubMCPAuthClient) Refresh(
	ctx context.Context,
	cfg mcpauth.ServerConfig,
) (mcpauth.Status, error) {
	if s.refreshFn != nil {
		return s.refreshFn(ctx, cfg)
	}
	return mcpauth.Status{ServerName: cfg.ServerName, Status: mcpauth.StatusAuthenticated}, nil
}

func (s *stubMCPAuthClient) Status(
	ctx context.Context,
	cfg mcpauth.ServerConfig,
) (mcpauth.Status, error) {
	if s.statusFn != nil {
		return s.statusFn(ctx, cfg)
	}
	return mcpauth.Status{ServerName: cfg.ServerName, Status: mcpauth.StatusNeedsLogin}, nil
}

func (s *stubMCPAuthClient) Logout(
	ctx context.Context,
	cfg mcpauth.ServerConfig,
) (mcpauth.Status, error) {
	if s.logoutFn != nil {
		return s.logoutFn(ctx, cfg)
	}
	return mcpauth.Status{ServerName: cfg.ServerName, Status: mcpauth.StatusNeedsLogin}, nil
}
