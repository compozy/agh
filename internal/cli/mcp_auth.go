package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	aghconfig "github.com/pedronauck/agh/internal/config"
	mcpauth "github.com/pedronauck/agh/internal/mcp/auth"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/spf13/cobra"
)

const defaultMCPAuthLoginTimeout = 2 * time.Minute

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
		_ = db.Close(ctx)
		return nil, nil, err
	}
	return service, db.Close, nil
}

func newMCPCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP integrations",
	}
	cmd.AddCommand(newMCPAuthCommand(deps))
	return cmd
}

func newMCPAuthCommand(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
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
		timeout     time.Duration
	)
	cmd := &cobra.Command{
		Use:   "login <server>",
		Short: "Run OAuth login for a remote MCP server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			runtime, client, cleanup, err := mcpAuthRuntime(cmd.Context(), deps)
			if err != nil {
				return err
			}
			defer func() {
				cleanupMCPAuthRuntime(cmd.Context(), cleanup, &runErr)
			}()

			cfg, err := resolveMCPAuthTarget(&runtime.Config, args[0], deps.getenv)
			if err != nil {
				return err
			}
			status, err := runMCPAuthLogin(cmd, client, cfg, redirectURL, manualCode, timeout)
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

			configs, err := listMCPAuthTargets(&runtime.Config, deps.getenv)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				cfg, err := resolveMCPAuthTarget(&runtime.Config, args[0], deps.getenv)
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
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			runtime, client, cleanup, err := mcpAuthRuntime(cmd.Context(), deps)
			if err != nil {
				return err
			}
			defer func() {
				cleanupMCPAuthRuntime(cmd.Context(), cleanup, &runErr)
			}()

			cfg, err := resolveMCPAuthTarget(&runtime.Config, args[0], deps.getenv)
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
	cfg mcpauth.ServerConfig,
	redirectURL string,
	manualCode string,
	timeout time.Duration,
) (status mcpauth.Status, runErr error) {
	if client == nil {
		return mcpauth.Status{}, errors.New("cli: MCP auth client is required")
	}
	if strings.TrimSpace(manualCode) != "" {
		if strings.TrimSpace(redirectURL) == "" {
			redirectURL = "http://127.0.0.1/callback"
		}
		state, err := client.BeginLogin(cmd.Context(), cfg, redirectURL)
		if err != nil {
			return mcpauth.Status{}, err
		}
		callback := callbackURLWithCode(redirectURL, manualCode, state.State)
		return client.Exchange(cmd.Context(), state, callback)
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

	_, _ = fmt.Fprintf(
		cmd.ErrOrStderr(),
		"Open this URL to authenticate %s:\n%s\n",
		cfg.ServerName,
		state.AuthorizationURL,
	)
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

func resolveMCPAuthTarget(
	cfg *aghconfig.Config,
	name string,
	getenv func(string) string,
) (mcpauth.ServerConfig, error) {
	target := strings.TrimSpace(name)
	if target == "" {
		return mcpauth.ServerConfig{}, errors.New("cli: MCP server name is required")
	}
	configs, err := listMCPAuthTargets(cfg, getenv)
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
	cfg *aghconfig.Config,
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

	lookupSecret := func(key string) string {
		if getenv == nil {
			return ""
		}
		return getenv(key)
	}
	return mcpauth.ServerConfigsFromMCP(servers, lookupSecret)
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
		[]string{"Server", "Status", "Client", "Expires", "Diagnostic"},
		"mcp_auth",
		[]string{"server", "status", "client", "expires_at", "diagnostic"},
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
		{Label: "Status", Value: stringOrDash(string(status.Status))},
		{Label: "Remote URL", Value: stringOrDash(status.RemoteURL)},
		{Label: "Auth Type", Value: stringOrDash(status.AuthType)},
		{Label: "Client ID", Value: stringOrDash(status.ClientID)},
		{Label: "Scopes", Value: stringOrDash(strings.Join(status.Scopes, ", "))},
		{Label: "Expires", Value: stringOrDash(timePtrString(status.ExpiresAt))},
		{Label: "Refreshable", Value: boolString(status.Refreshable)},
		{Label: "Diagnostic", Value: stringOrDash(status.Diagnostic)},
	}
}

func mcpAuthStatusFields(status mcpauth.Status) ([]string, []string) {
	fields := []string{
		"server",
		"status",
		"remote_url",
		"auth_type",
		"client_id",
		"scopes",
		"expires_at",
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
