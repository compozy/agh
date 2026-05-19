package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/diagnostics"
	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/vault"
	"github.com/spf13/cobra"
)

const (
	mcpAuthExpiresValue = "Expires"
	mcpAuthAuthKey      = "auth"
	mcpAuthExpiresAtKey = "expires_at"
	mcpAuthMCPKey       = "mcp"
)

const defaultMCPAuthLoginTimeout = 2 * time.Minute
const mcpAuthPendingLoginDir = "mcp-auth"

type mcpAuthClient interface {
	BeginLogin(ctx context.Context, cfg mcpauth.ServerConfig, redirectURL string) (mcpauth.LoginState, error)
	Exchange(ctx context.Context, state mcpauth.LoginState, callbackURL string) (mcpauth.Status, error)
	Refresh(ctx context.Context, cfg mcpauth.ServerConfig) (mcpauth.Status, error)
	Status(ctx context.Context, cfg mcpauth.ServerConfig) (mcpauth.Status, error)
	Logout(ctx context.Context, cfg mcpauth.ServerConfig) (mcpauth.Status, error)
}

type newMCPAuthClientFunc func(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
) (mcpAuthClient, func(context.Context) error, error)

func (d commandDeps) withMCPAuthDefaults() commandDeps {
	if d.newMCPAuthClient == nil {
		d.newMCPAuthClient = defaultMCPAuthClient
	}
	return d
}

func defaultMCPAuthClient(
	ctx context.Context,
	homePaths aghconfig.HomePaths,
) (mcpAuthClient, func(context.Context) error, error) {
	db, err := globaldb.OpenGlobalDB(ctx, homePaths.DatabaseFile)
	if err != nil {
		return nil, nil, err
	}
	service, err := mcpauth.NewService(db)
	if err != nil {
		if closeErr := db.Close(ctx); closeErr != nil {
			return nil, nil, fmt.Errorf(
				"cli: initialize MCP auth service for %q: %w; close global DB: %v",
				homePaths.DatabaseFile,
				err,
				closeErr,
			)
		}
		return nil, nil, fmt.Errorf("cli: initialize MCP auth service for %q: %w", homePaths.DatabaseFile, err)
	}
	return service, db.Close, nil
}

func newMCPCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   mcpAuthMCPKey,
		Short: "Manage MCP integrations",
	}
	cmd.AddCommand(newMCPAuthCommand(deps))
	return cmd
}

func newMCPAuthCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   mcpAuthAuthKey,
		Short: "Authenticate remote MCP servers",
	}
	cmd.AddCommand(newMCPAuthLoginCommand(deps))
	cmd.AddCommand(newMCPAuthStatusCommand(deps))
	cmd.AddCommand(newMCPAuthLogoutCommand(deps))
	return cmd
}

func newMCPAuthLoginCommand(deps commandDeps) *cobra.Command {
	var (
		redirectURL string
		manualCode  string
		manual      bool
		timeout     time.Duration
	)
	cmd := &cobra.Command{
		Use:   "login <server>",
		Short: "Run OAuth login for a remote MCP server",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			runtime, client, cleanup, err := mcpAuthRuntime(cmd.Context(), deps)
			if err != nil {
				return err
			}
			defer func() {
				cleanupMCPAuthRuntime(cmd.Context(), cleanup, &runErr)
			}()

			cfg, err := resolveMCPAuthTarget(cmd.Context(), &runtime.Config, runtime.HomePaths, args[0], deps.getenv)
			if err != nil {
				return err
			}
			status, err := runMCPAuthLogin(
				cmd,
				client,
				runtime.HomePaths,
				cfg,
				redirectURL,
				manualCode,
				manual,
				timeout,
			)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, mcpAuthStatusBundle(status))
		},
	}
	cmd.Flags().StringVar(&redirectURL, "redirect-url", "", "OAuth redirect URL")
	cmd.Flags().StringVar(
		&manualCode,
		"manual-code",
		"",
		"Exchange an authorization code without starting the loopback listener",
	)
	cmd.Flags().BoolVar(&manual, "manual", false, "Print an authorization URL for manual code exchange")
	cmd.Flags().DurationVar(&timeout, "timeout", defaultMCPAuthLoginTimeout, "Loopback login timeout")
	return cmd
}

func newMCPAuthStatusCommand(deps commandDeps) *cobra.Command {
	var refresh bool
	cmd := &cobra.Command{
		Use:   "status [server]",
		Short: "Show redacted remote MCP auth status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			runtime, client, cleanup, err := mcpAuthRuntime(cmd.Context(), deps)
			if err != nil {
				return err
			}
			defer func() {
				cleanupMCPAuthRuntime(cmd.Context(), cleanup, &runErr)
			}()

			configs, err := listMCPAuthTargets(cmd.Context(), &runtime.Config, runtime.HomePaths, deps.getenv)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				cfg, err := resolveMCPAuthTarget(
					cmd.Context(),
					&runtime.Config,
					runtime.HomePaths,
					args[0],
					deps.getenv,
				)
				if err != nil {
					return err
				}
				configs = []mcpauth.ServerConfig{cfg}
			}

			statuses := make([]mcpauth.Status, 0, len(configs))
			for _, cfg := range configs {
				var status mcpauth.Status
				if refresh {
					status, err = client.Refresh(cmd.Context(), cfg)
				} else {
					status, err = client.Status(cmd.Context(), cfg)
				}
				if err != nil {
					return err
				}
				statuses = append(statuses, status)
			}
			return writeCommandOutput(cmd, mcpAuthStatusListBundle(statuses))
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "Refresh tokens before reporting status")
	return cmd
}

func newMCPAuthLogoutCommand(deps commandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "logout <server>",
		Short: "Revoke or delete remote MCP auth tokens",
		Args:  exactOneNonBlankArg(),
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			runtime, client, cleanup, err := mcpAuthRuntime(cmd.Context(), deps)
			if err != nil {
				return err
			}
			defer func() {
				cleanupMCPAuthRuntime(cmd.Context(), cleanup, &runErr)
			}()

			cfg, err := resolveMCPAuthTarget(cmd.Context(), &runtime.Config, runtime.HomePaths, args[0], deps.getenv)
			if err != nil {
				return err
			}
			status, err := client.Logout(cmd.Context(), cfg)
			if err != nil {
				return err
			}
			return writeCommandOutput(cmd, mcpAuthStatusBundle(status))
		},
	}
}

func mcpAuthRuntime(
	ctx context.Context,
	deps commandDeps,
) (*runtimeContext, mcpAuthClient, func(context.Context) error, error) {
	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := deps.ensureHome(runtime.HomePaths); err != nil {
		return nil, nil, nil, err
	}
	client, cleanup, err := deps.newMCPAuthClient(ctx, runtime.HomePaths)
	if err != nil {
		return nil, nil, nil, err
	}
	if cleanup == nil {
		cleanup = func(context.Context) error { return nil }
	}
	return runtime, client, cleanup, nil
}

func cleanupMCPAuthRuntime(
	ctx context.Context,
	cleanup func(context.Context) error,
	runErr *error,
) {
	if cleanup == nil {
		return
	}
	if ctx == nil {
		ctx = context.TODO()
	} else {
		ctx = context.WithoutCancel(ctx)
	}
	if err := cleanup(ctx); err != nil && runErr != nil && *runErr == nil {
		*runErr = fmt.Errorf("cli: close MCP auth runtime: %w", err)
	}
}

func runMCPAuthLogin(
	cmd *cobra.Command,
	client mcpAuthClient,
	homePaths aghconfig.HomePaths,
	cfg mcpauth.ServerConfig,
	redirectURL string,
	manualCode string,
	manual bool,
	timeout time.Duration,
) (status mcpauth.Status, runErr error) {
	if client == nil {
		return mcpauth.Status{}, errors.New("cli: MCP auth client is required")
	}
	if manual && strings.TrimSpace(manualCode) != "" {
		return mcpauth.Status{}, errors.New("cli: --manual and --manual-code are mutually exclusive")
	}
	if manual {
		return runMCPAuthManualLogin(cmd, client, homePaths, cfg, redirectURL)
	}
	if strings.TrimSpace(manualCode) != "" {
		return runMCPAuthManualCodeLogin(cmd, client, homePaths, cfg, manualCode)
	}

	if timeout <= 0 {
		return mcpauth.Status{}, errors.New("cli: login timeout must be positive")
	}
	listener, actualRedirectURL, err := listenForMCPAuthCallback(cmd.Context(), redirectURL)
	if err != nil {
		return mcpauth.Status{}, err
	}
	defer func() {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) && runErr == nil {
			runErr = fmt.Errorf("cli: close MCP auth listener: %w", err)
		}
	}()

	state, err := client.BeginLogin(cmd.Context(), cfg, actualRedirectURL)
	if err != nil {
		return mcpauth.Status{}, err
	}
	callbackCh, errCh, shutdown, err := serveMCPAuthCallback(listener, actualRedirectURL)
	if err != nil {
		return mcpauth.Status{}, err
	}
	defer func() {
		if err := shutdown(cmd.Context()); err != nil && runErr == nil {
			runErr = err
		}
	}()

	if _, err := fmt.Fprintf(
		cmd.ErrOrStderr(),
		"Open this URL to authenticate %s:\n%s\n",
		cfg.ServerName,
		state.AuthorizationURL,
	); err != nil {
		return mcpauth.Status{}, fmt.Errorf("cli: write MCP auth login instructions: %w", err)
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	select {
	case callbackURL := <-callbackCh:
		return client.Exchange(cmd.Context(), state, callbackURL)
	case err := <-errCh:
		return mcpauth.Status{}, err
	case <-ctx.Done():
		return mcpauth.Status{}, fmt.Errorf("cli: MCP auth login timed out: %w", ctx.Err())
	}
}

func runMCPAuthManualLogin(
	cmd *cobra.Command,
	client mcpAuthClient,
	homePaths aghconfig.HomePaths,
	cfg mcpauth.ServerConfig,
	redirectURL string,
) (mcpauth.Status, error) {
	if strings.TrimSpace(redirectURL) == "" {
		redirectURL = "http://127.0.0.1/callback"
	}
	state, err := client.BeginLogin(cmd.Context(), cfg, redirectURL)
	if err != nil {
		return mcpauth.Status{}, err
	}
	if err := saveMCPAuthPendingLogin(homePaths, state); err != nil {
		return mcpauth.Status{}, err
	}
	if _, err := fmt.Fprintf(
		cmd.ErrOrStderr(),
		"Open this URL to authenticate %s, then rerun with --manual-code <code>:\n%s\n",
		cfg.ServerName,
		state.AuthorizationURL,
	); err != nil {
		return mcpauth.Status{}, fmt.Errorf("cli: write MCP auth manual login instructions: %w", err)
	}
	return mcpauth.Status{
		ServerName:       cfg.ServerName,
		Status:           mcpauth.StatusNeedsLogin,
		RemoteURL:        cfg.RemoteURL,
		AuthType:         cfg.Type,
		ClientID:         cfg.ClientID,
		Scopes:           append([]string(nil), cfg.Scopes...),
		AuthorizationURL: state.AuthorizationURL,
		Diagnostic:       "manual_code_pending",
	}, nil
}

func runMCPAuthManualCodeLogin(
	cmd *cobra.Command,
	client mcpAuthClient,
	homePaths aghconfig.HomePaths,
	cfg mcpauth.ServerConfig,
	manualCode string,
) (mcpauth.Status, error) {
	state, err := loadMCPAuthPendingLogin(homePaths, cfg)
	if err != nil {
		return mcpauth.Status{}, err
	}
	callback := callbackURLWithCode(state.RedirectURL, manualCode, state.State)
	status, err := client.Exchange(cmd.Context(), state, callback)
	if err != nil {
		return status, err
	}
	if err := removeMCPAuthPendingLogin(homePaths, cfg.ServerName); err != nil {
		return status, err
	}
	return status, nil
}

func resolveMCPAuthTarget(
	ctx context.Context,
	cfg *aghconfig.Config,
	homePaths aghconfig.HomePaths,
	name string,
	getenv func(string) string,
) (mcpauth.ServerConfig, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return mcpauth.ServerConfig{}, errors.New("cli: MCP server name is required")
	}
	configs, err := listMCPAuthTargets(ctx, cfg, homePaths, getenv)
	if err != nil {
		return mcpauth.ServerConfig{}, err
	}
	for _, item := range configs {
		if item.ServerName == target {
			return item, nil
		}
	}
	return mcpauth.ServerConfig{}, fmt.Errorf("cli: remote MCP auth server %q not found", target)
}

func listMCPAuthTargets(
	ctx context.Context,
	cfg *aghconfig.Config,
	homePaths aghconfig.HomePaths,
	getenv func(string) string,
) ([]mcpauth.ServerConfig, error) {
	if cfg == nil {
		return nil, errors.New("cli: MCP auth config is required")
	}
	servers := append([]aghconfig.MCPServer(nil), cfg.MCPServers...)
	providerNames := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		providerNames = append(providerNames, name)
	}
	sort.Strings(providerNames)
	for _, name := range providerNames {
		servers = aghconfig.MergeMCPServers(servers, cfg.Providers[name].MCPServers)
	}

	return mcpauth.ServerConfigsFromMCP(ctx, servers, mcpAuthSecretResolver(homePaths, getenv))
}

func mcpAuthSecretResolver(homePaths aghconfig.HomePaths, getenv func(string) string) mcpauth.SecretRefResolver {
	lookupEnv := func(key string) (string, bool) {
		if getenv == nil {
			return "", false
		}
		value := getenv(key)
		return value, strings.TrimSpace(value) != ""
	}
	return func(ctx context.Context, ref string) (string, error) {
		normalized := vault.NormalizeRef(ref)
		if vault.IsEnvRef(normalized) {
			envName, err := vault.EnvNameFromRef(normalized)
			if err != nil {
				return "", err
			}
			value, ok := lookupEnv(envName)
			if !ok {
				return "", fmt.Errorf("%w: env:%s", vault.ErrMissingSecret, envName)
			}
			diagnostics.RegisterDynamicSecret(value)
			return value, nil
		}
		if !vault.IsSecretRef(normalized) {
			return "", fmt.Errorf("%w: %s", vault.ErrUnsupportedSecretRef, normalized)
		}
		db, err := globaldb.OpenGlobalDB(ctx, homePaths.DatabaseFile)
		if err != nil {
			return "", fmt.Errorf("cli: open global DB for MCP auth secret: %w", err)
		}
		service, err := vault.NewService(
			db,
			vault.NewFileKeyProvider(homePaths.HomeDir, lookupEnv),
			vault.WithLookupEnv(lookupEnv),
		)
		if err != nil {
			closeErr := db.Close(ctx)
			return "", errors.Join(fmt.Errorf("cli: initialize MCP auth secret resolver: %w", err), closeErr)
		}
		value, resolveErr := service.ResolveRef(ctx, normalized)
		closeErr := db.Close(ctx)
		if resolveErr != nil {
			return "", errors.Join(
				fmt.Errorf("cli: resolve MCP auth secret ref %q: %w", normalized, resolveErr),
				closeErr,
			)
		}
		if closeErr != nil {
			return "", fmt.Errorf("cli: close MCP auth secret store: %w", closeErr)
		}
		diagnostics.RegisterDynamicSecret(value)
		return value, nil
	}
}

func listenForMCPAuthCallback(ctx context.Context, redirectURL string) (net.Listener, string, error) {
	if ctx == nil {
		return nil, "", errors.New("cli: listen context is required")
	}
	var listenConfig net.ListenConfig
	if strings.TrimSpace(redirectURL) == "" {
		listener, err := listenConfig.Listen(ctx, "tcp", "127.0.0.1:0")
		if err != nil {
			return nil, "", fmt.Errorf("cli: listen for MCP auth callback: %w", err)
		}
		return listener, "http://" + listener.Addr().String() + "/callback", nil
	}

	parsed, err := url.Parse(strings.TrimSpace(redirectURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, "", errors.New("cli: redirect-url must be an absolute http URL")
	}
	if parsed.Scheme != "http" {
		return nil, "", errors.New("cli: redirect-url loopback listener requires http")
	}
	if !mcpAuthLoopbackHost(parsed.Hostname()) {
		return nil, "", errors.New("cli: redirect-url loopback listener requires localhost or loopback IP")
	}
	if parsed.Path == "" {
		parsed.Path = "/callback"
	}
	listener, err := listenConfig.Listen(ctx, "tcp", parsed.Host)
	if err != nil {
		return nil, "", fmt.Errorf("cli: listen for MCP auth callback: %w", err)
	}
	if parsed.Port() == "0" {
		parsed.Host = listener.Addr().String()
	}
	return listener, parsed.String(), nil
}

func mcpAuthLoopbackHost(host string) bool {
	normalized := strings.Trim(strings.TrimSpace(host), "[]")
	if strings.EqualFold(normalized, "localhost") {
		return true
	}
	ip := net.ParseIP(normalized)
	return ip != nil && ip.IsLoopback()
}

func serveMCPAuthCallback(
	listener net.Listener,
	redirectURL string,
) (<-chan string, <-chan error, func(context.Context) error, error) {
	callbackCh := make(chan string, 1)
	errCh := make(chan error, 1)
	parsed, err := url.Parse(redirectURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cli: parse MCP auth redirect URL: %w", err)
	}
	server := &http.Server{ReadHeaderTimeout: 5 * time.Second}
	mux := http.NewServeMux()
	server.Handler = mux
	mux.HandleFunc(parsed.Path, func(w http.ResponseWriter, r *http.Request) {
		callback := *parsed
		callback.RawQuery = r.URL.RawQuery
		callback.Fragment = r.URL.Fragment
		callbackCh <- callback.String()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = fmt.Fprintln(w, "AGH MCP authentication received. You can return to the terminal.")
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("cli: serve MCP auth callback: %w", err)
		}
	}()
	return callbackCh, errCh, func(parent context.Context) error {
		if parent == nil {
			parent = context.TODO()
		} else {
			parent = context.WithoutCancel(parent)
		}
		ctx, cancel := context.WithTimeout(parent, 2*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("cli: stop MCP auth callback server: %w", err)
		}
		<-done
		return nil
	}, nil
}

func callbackURLWithCode(redirectURL string, code string, state string) string {
	parsed, err := url.Parse(strings.TrimSpace(redirectURL))
	if err != nil {
		return strings.TrimSpace(redirectURL)
	}
	values := parsed.Query()
	values.Set("code", strings.TrimSpace(code))
	values.Set("state", strings.TrimSpace(state))
	parsed.RawQuery = values.Encode()
	return parsed.String()
}

type mcpAuthPendingLoginRecord struct {
	ServerName       string           `json:"server_name"`
	RedirectURL      string           `json:"redirect_url"`
	State            string           `json:"state"`
	Verifier         string           `json:"verifier"`
	AuthorizationURL string           `json:"authorization_url"`
	Metadata         mcpauth.Metadata `json:"metadata"`
	CreatedAt        time.Time        `json:"created_at"`
}

func saveMCPAuthPendingLogin(homePaths aghconfig.HomePaths, state mcpauth.LoginState) error {
	path, err := mcpAuthPendingLoginPath(homePaths, state.ServerName)
	if err != nil {
		return err
	}
	record := mcpAuthPendingLoginRecord{
		ServerName:       state.ServerName,
		RedirectURL:      state.RedirectURL,
		State:            state.State,
		Verifier:         state.Verifier,
		AuthorizationURL: state.AuthorizationURL,
		Metadata:         state.Metadata,
		CreatedAt:        time.Now().UTC(),
	}
	payload, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("cli: encode pending MCP auth login: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("cli: create pending MCP auth login directory: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return fmt.Errorf("cli: write pending MCP auth login: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("cli: protect pending MCP auth login: %w", err)
	}
	diagnostics.RegisterDynamicSecret(state.Verifier)
	return nil
}

func loadMCPAuthPendingLogin(
	homePaths aghconfig.HomePaths,
	cfg mcpauth.ServerConfig,
) (mcpauth.LoginState, error) {
	path, err := mcpAuthPendingLoginPath(homePaths, cfg.ServerName)
	if err != nil {
		return mcpauth.LoginState{}, err
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return mcpauth.LoginState{}, fmt.Errorf(
				"cli: pending MCP auth login for %q not found; run login with --manual first",
				cfg.ServerName,
			)
		}
		return mcpauth.LoginState{}, fmt.Errorf("cli: read pending MCP auth login: %w", err)
	}
	var record mcpAuthPendingLoginRecord
	if err := json.Unmarshal(payload, &record); err != nil {
		return mcpauth.LoginState{}, fmt.Errorf("cli: decode pending MCP auth login: %w", err)
	}
	if record.ServerName != cfg.ServerName {
		return mcpauth.LoginState{}, fmt.Errorf(
			"cli: pending MCP auth login server %q does not match %q",
			record.ServerName,
			cfg.ServerName,
		)
	}
	state := mcpauth.LoginState{
		ServerName:       record.ServerName,
		RedirectURL:      record.RedirectURL,
		State:            record.State,
		Verifier:         record.Verifier,
		AuthorizationURL: record.AuthorizationURL,
		Metadata:         record.Metadata,
		Config:           cfg,
	}
	diagnostics.RegisterDynamicSecret(state.Verifier)
	return state, nil
}

func removeMCPAuthPendingLogin(homePaths aghconfig.HomePaths, serverName string) error {
	path, err := mcpAuthPendingLoginPath(homePaths, serverName)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("cli: remove pending MCP auth login: %w", err)
	}
	return nil
}

func mcpAuthPendingLoginPath(homePaths aghconfig.HomePaths, serverName string) (string, error) {
	if strings.TrimSpace(homePaths.RestartsDir) == "" {
		return "", errors.New("cli: AGH home restarts directory is required for MCP auth manual login")
	}
	normalized := strings.TrimSpace(serverName)
	if normalized == "" {
		return "", errors.New("cli: MCP server name is required for pending auth login")
	}
	sum := sha256.Sum256([]byte(normalized))
	return filepath.Join(
		homePaths.RestartsDir,
		mcpAuthPendingLoginDir,
		hex.EncodeToString(sum[:])+".json",
	), nil
}

func mcpAuthStatusBundle(status mcpauth.Status) outputBundle {
	return outputBundle{
		jsonValue: status,
		human: func() (string, error) {
			return renderHumanSection("MCP Auth", mcpAuthStatusRows(status)), nil
		},
		toon: func() (string, error) {
			fields, values := mcpAuthStatusFields(status)
			return renderToonObject("mcp_auth", fields, values), nil
		},
	}
}

func mcpAuthStatusListBundle(statuses []mcpauth.Status) outputBundle {
	return listBundle(
		statuses,
		statuses,
		"MCP Auth",
		[]string{"Server", automationStatusValue, "Client", mcpAuthExpiresValue, "Diagnostic"},
		"mcp_auth",
		[]string{"server", automationStatusKey, "client", mcpAuthExpiresAtKey, "diagnostic"},
		func(item mcpauth.Status) []string {
			return []string{
				stringOrDash(item.ServerName),
				stringOrDash(string(item.Status)),
				stringOrDash(item.ClientID),
				stringOrDash(timePtrString(item.ExpiresAt)),
				stringOrDash(item.Diagnostic),
			}
		},
		func(item mcpauth.Status) []string {
			return []string{
				item.ServerName,
				string(item.Status),
				item.ClientID,
				timePtrString(item.ExpiresAt),
				item.Diagnostic,
			}
		},
	)
}

func mcpAuthStatusRows(status mcpauth.Status) []keyValue {
	return []keyValue{
		{Label: "Server", Value: stringOrDash(status.ServerName)},
		{Label: automationStatusValue, Value: stringOrDash(string(status.Status))},
		{Label: "Remote URL", Value: stringOrDash(status.RemoteURL)},
		{Label: "Auth Type", Value: stringOrDash(status.AuthType)},
		{Label: "Client ID", Value: stringOrDash(status.ClientID)},
		{Label: "Scopes", Value: stringOrDash(strings.Join(status.Scopes, ", "))},
		{Label: mcpAuthExpiresValue, Value: stringOrDash(timePtrString(status.ExpiresAt))},
		{Label: "Refreshable", Value: boolString(status.Refreshable)},
		{Label: "Diagnostic", Value: stringOrDash(status.Diagnostic)},
	}
}

func mcpAuthStatusFields(status mcpauth.Status) ([]string, []string) {
	fields := []string{
		"server",
		automationStatusKey,
		"remote_url",
		"auth_type",
		"client_id",
		"scopes",
		mcpAuthExpiresAtKey,
		"refreshable",
		"diagnostic",
	}
	values := []string{
		status.ServerName,
		string(status.Status),
		status.RemoteURL,
		status.AuthType,
		status.ClientID,
		strings.Join(status.Scopes, "|"),
		timePtrString(status.ExpiresAt),
		boolString(status.Refreshable),
		status.Diagnostic,
	}
	return fields, values
}

func timePtrString(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func boolString(value bool) string {
	return strconv.FormatBool(value)
}
