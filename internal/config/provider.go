package config

import (
	"errors"
	"fmt"
	"maps"
	"strings"
)

// ProviderConfig describes how to launch a provider in ACP mode.
type ProviderConfig struct {
	Command      string      `toml:"command"`
	DefaultModel string      `toml:"default_model"`
	APIKeyEnv    string      `toml:"api_key_env"`
	MCPServers   []MCPServer `toml:"mcp_servers,omitempty"`
}

// MCPServerTransport identifies how AGH reaches an MCP server.
type MCPServerTransport string

const (
	// MCPServerTransportStdio launches a local subprocess and talks MCP over stdio.
	MCPServerTransportStdio MCPServerTransport = "stdio"
	// MCPServerTransportHTTP talks to a remote streamable HTTP MCP endpoint.
	MCPServerTransportHTTP MCPServerTransport = "http"
	// MCPServerTransportSSE talks to a remote SSE MCP endpoint.
	MCPServerTransportSSE MCPServerTransport = "sse"
)

// MCPAuthType identifies the remote MCP authentication mechanism.
type MCPAuthType string

const (
	// MCPAuthTypeOAuth2PKCE uses OAuth 2.1 authorization code with PKCE.
	MCPAuthTypeOAuth2PKCE MCPAuthType = "oauth2_pkce"
)

// MCPAuthConfig describes remote MCP OAuth configuration. It stores endpoint
// metadata and environment variable names only; token material is persisted in
// the auth token store.
type MCPAuthConfig struct {
	Type             MCPAuthType `json:"type,omitempty"              yaml:"type,omitempty"              toml:"type,omitempty"`
	IssuerURL        string      `json:"issuer_url,omitempty"        yaml:"issuer_url,omitempty"        toml:"issuer_url,omitempty"`
	MetadataURL      string      `json:"metadata_url,omitempty"      yaml:"metadata_url,omitempty"      toml:"metadata_url,omitempty"`
	AuthorizationURL string      `json:"authorization_url,omitempty" yaml:"authorization_url,omitempty" toml:"authorization_url,omitempty"`
	TokenURL         string      `json:"token_url,omitempty"         yaml:"token_url,omitempty"         toml:"token_url,omitempty"`
	RevocationURL    string      `json:"revocation_url,omitempty"    yaml:"revocation_url,omitempty"    toml:"revocation_url,omitempty"`
	ClientID         string      `json:"client_id,omitempty"         yaml:"client_id,omitempty"         toml:"client_id,omitempty"`
	ClientSecretEnv  string      `json:"client_secret_env,omitempty" yaml:"client_secret_env,omitempty" toml:"client_secret_env,omitempty"`
	Scopes           []string    `json:"scopes,omitempty"            yaml:"scopes,omitempty"            toml:"scopes,omitempty"`
}

// MCPServer describes an MCP server passed through to the agent runtime.
type MCPServer struct {
	Name      string             `json:"name"                yaml:"name"                toml:"name"`
	Transport MCPServerTransport `json:"transport,omitempty" yaml:"transport,omitempty" toml:"transport,omitempty"`
	Command   string             `json:"command,omitempty"   yaml:"command,omitempty"   toml:"command,omitempty"`
	Args      []string           `json:"args,omitempty"      yaml:"args,omitempty"      toml:"args,omitempty"`
	Env       map[string]string  `json:"env,omitempty"       yaml:"env,omitempty"       toml:"env,omitempty"`
	URL       string             `json:"url,omitempty"       yaml:"url,omitempty"       toml:"url,omitempty"`
	Auth      MCPAuthConfig      `json:"auth"                yaml:"auth,omitempty"      toml:"auth,omitempty"`
}

// ResolvedAgent is the effective runtime configuration for a parsed agent definition.
type ResolvedAgent struct {
	Name        string
	Provider    string
	Command     string
	Model       string
	Tools       []string
	Permissions string
	APIKeyEnv   string
	MCPServers  []MCPServer
	Prompt      string
}

// ErrProviderUnavailable reports that a requested provider cannot be resolved
// from the effective workspace/global config.
var ErrProviderUnavailable = errors.New("provider unavailable")

var builtinProviders = map[string]ProviderConfig{
	"claude": {
		Command:      "npx -y @agentclientprotocol/claude-agent-acp@0.24.2",
		DefaultModel: "claude-sonnet-4-20250514",
		APIKeyEnv:    envVarName("ANTHROPIC", "API", "KEY"),
	},
	"codex": {
		Command:      "npx @zed-industries/codex-acp@0.10.0",
		DefaultModel: "gpt-4o",
		APIKeyEnv:    envVarName("OPENAI", "API", "KEY"),
	},
	"gemini": {
		Command:      "gemini --acp",
		DefaultModel: "gemini-2.5-pro",
		APIKeyEnv:    envVarName("GEMINI", "API", "KEY"),
	},
	"opencode": {
		Command: "npx -y opencode-ai acp",
	},
	"copilot": {
		Command: "copilot --acp --stdio",
	},
	"cursor": {
		Command: "cursor-agent acp",
	},
	"kiro": {
		Command: "kiro-cli-chat acp",
	},
	"pi": {
		Command: "npx pi-acp@0.0.22",
	},
}

// BuiltinProviders returns a deep copy of the built-in provider registry.
func BuiltinProviders() map[string]ProviderConfig {
	return cloneProviders(builtinProviders)
}

func envVarName(parts ...string) string {
	return strings.Join(parts, "_")
}

// ResolveProvider resolves a provider using the built-in registry and config overrides.
func (c *Config) ResolveProvider(name string) (ProviderConfig, error) {
	providerName := strings.TrimSpace(name)
	if providerName == "" {
		return ProviderConfig{}, errors.New("provider name is required")
	}

	resolved, hasBuiltin := builtinProviders[providerName]
	if c != nil {
		if override, ok := c.Providers[providerName]; ok {
			resolved = mergeProvider(resolved, override)
		}
	}

	if !hasBuiltin {
		if c == nil {
			return ProviderConfig{}, newUnknownProviderError(providerName)
		}
		if _, ok := c.Providers[providerName]; !ok {
			return ProviderConfig{}, newUnknownProviderError(providerName)
		}
	}

	if err := validateResolvedProvider(providerName, resolved); err != nil {
		return ProviderConfig{}, fmt.Errorf("%w: %w", ErrProviderUnavailable, err)
	}

	return resolved, nil
}

// ResolveAgent resolves a parsed agent definition against provider config and global defaults.
func (c *Config) ResolveAgent(agent AgentDef) (ResolvedAgent, error) {
	if err := agent.Validate(); err != nil {
		return ResolvedAgent{}, err
	}

	var defaults DefaultsConfig
	var permissions PermissionsConfig
	var mcpServers []MCPServer
	if c != nil {
		defaults = c.Defaults
		permissions = c.Permissions
		mcpServers = c.MCPServers
	}

	providerName := strings.TrimSpace(agent.Provider)
	if providerName == "" {
		providerName = strings.TrimSpace(defaults.Provider)
	}
	if providerName == "" {
		return ResolvedAgent{}, errors.New(
			"agent provider is required; run `agh install` or set agent.provider/defaults.provider",
		)
	}

	provider, err := c.ResolveProvider(providerName)
	if err != nil {
		return ResolvedAgent{}, err
	}

	tools := cloneStrings(agent.Tools)
	if len(tools) == 0 {
		tools = []string{"*"}
	}

	resolvedPermissions := strings.TrimSpace(agent.Permissions)
	if resolvedPermissions == "" {
		resolvedPermissions = string(permissions.Mode)
	}

	command := strings.TrimSpace(agent.Command)
	if command == "" {
		command = strings.TrimSpace(provider.Command)
	}

	model := strings.TrimSpace(agent.Model)
	if model == "" {
		model = strings.TrimSpace(provider.DefaultModel)
	}

	resolved := ResolvedAgent{
		Name:        agent.Name,
		Provider:    providerName,
		Command:     command,
		Model:       model,
		Tools:       tools,
		Permissions: resolvedPermissions,
		APIKeyEnv:   provider.APIKeyEnv,
		MCPServers:  mergeMCPServerLayers(mcpServers, provider.MCPServers, agent.MCPServers),
		Prompt:      agent.Prompt,
	}

	if strings.TrimSpace(resolved.Command) == "" {
		return ResolvedAgent{}, fmt.Errorf("provider %q command is required", providerName)
	}
	if strings.TrimSpace(resolved.Permissions) != "" {
		if err := PermissionMode(resolved.Permissions).Validate("agent.permissions"); err != nil {
			return ResolvedAgent{}, err
		}
	}

	return resolved, nil
}

// ResolveSessionAgent resolves a parsed agent definition for one session.
// When providerOverride is set, the selected provider becomes canonical and
// provider-owned runtime fields are re-resolved from that provider to avoid
// mixed runtimes from the original agent definition.
func (c *Config) ResolveSessionAgent(agent AgentDef, providerOverride string) (ResolvedAgent, error) {
	override := strings.TrimSpace(providerOverride)
	if override == "" {
		return c.ResolveAgent(agent)
	}

	effectiveProvider := strings.TrimSpace(agent.Provider)
	if effectiveProvider == "" && c != nil {
		effectiveProvider = strings.TrimSpace(c.Defaults.Provider)
	}
	if override == effectiveProvider {
		return c.ResolveAgent(agent)
	}

	sessionAgent := agent
	sessionAgent.Provider = override
	sessionAgent.Command = ""
	sessionAgent.Model = ""

	resolved, err := c.ResolveAgent(sessionAgent)
	if err != nil {
		return ResolvedAgent{}, fmt.Errorf("resolve session agent with provider %q: %w", override, err)
	}

	return resolved, nil
}

func mergeProvider(base ProviderConfig, override ProviderConfig) ProviderConfig {
	merged := cloneProvider(base)
	if strings.TrimSpace(override.Command) != "" {
		merged.Command = override.Command
	}
	if strings.TrimSpace(override.DefaultModel) != "" {
		merged.DefaultModel = override.DefaultModel
	}
	if strings.TrimSpace(override.APIKeyEnv) != "" {
		merged.APIKeyEnv = override.APIKeyEnv
	}
	merged.MCPServers = MergeMCPServers(merged.MCPServers, override.MCPServers)

	return merged
}

func newUnknownProviderError(providerName string) error {
	return fmt.Errorf("%w: unknown provider %q", ErrProviderUnavailable, providerName)
}

// MergeMCPServers merges provider-level and agent-level MCP servers by name.
func MergeMCPServers(base []MCPServer, overlay []MCPServer) []MCPServer {
	return mergeMCPServerLayers(base, overlay)
}

// OverrideMCPServers overlays MCP servers by name, replacing the full server object
// on collision instead of field-merging it.
func OverrideMCPServers(base []MCPServer, overlay []MCPServer) []MCPServer {
	merged := cloneMCPServersWithCapacity(base, len(base)+len(overlay))
	index := indexMCPServersByName(merged)

	for _, server := range overlay {
		name := normalizeMCPServerName(server.Name)
		if idx, ok := index[name]; ok && name != "" {
			merged[idx] = cloneMCPServer(server)
			continue
		}

		merged = append(merged, cloneMCPServer(server))
		if name != "" {
			index[name] = len(merged) - 1
		}
	}

	return merged
}

func mergeMCPServerLayers(base []MCPServer, overlays ...[]MCPServer) []MCPServer {
	totalCapacity := len(base)
	for _, overlay := range overlays {
		totalCapacity += len(overlay)
	}

	merged := cloneMCPServersWithCapacity(base, totalCapacity)
	index := indexMCPServersByName(merged)

	for _, overlay := range overlays {
		for _, server := range overlay {
			name := normalizeMCPServerName(server.Name)
			if idx, ok := index[name]; ok && name != "" {
				merged[idx] = mergeMCPServer(merged[idx], server)
				continue
			}

			merged = append(merged, cloneMCPServer(server))
			if name != "" {
				index[name] = len(merged) - 1
			}
		}
	}

	return merged
}

func normalizeMCPServerName(name string) string {
	return strings.TrimSpace(name)
}

func indexMCPServersByName(servers []MCPServer) map[string]int {
	index := make(map[string]int, len(servers))
	for i, server := range servers {
		name := normalizeMCPServerName(server.Name)
		if name == "" {
			continue
		}
		index[name] = i
	}
	return index
}

// Validate ensures the MCP server entry is usable.
func (s MCPServer) Validate(path string) error {
	transport := s.EffectiveTransport()
	if err := transport.Validate(path + ".transport"); err != nil {
		return err
	}
	if err := s.Auth.Validate(path + ".auth"); err != nil {
		return err
	}
	switch {
	case strings.TrimSpace(s.Name) == "":
		return fmt.Errorf("%s.name is required", path)
	case transport == MCPServerTransportStdio && strings.TrimSpace(s.Command) == "":
		return fmt.Errorf("%s.command is required", path)
	case transport == MCPServerTransportStdio && strings.TrimSpace(s.URL) != "":
		return fmt.Errorf("%s.url requires remote transport", path)
	case transport != MCPServerTransportStdio && strings.TrimSpace(s.URL) == "":
		return fmt.Errorf("%s.url is required for %s transport", path, transport)
	case transport != MCPServerTransportStdio && strings.TrimSpace(s.Command) != "":
		return fmt.Errorf("%s.command is only valid for stdio transport", path)
	case transport == MCPServerTransportStdio && !s.Auth.IsZero():
		return fmt.Errorf("%s.auth is only valid for remote MCP servers", path)
	default:
		return nil
	}
}

// EffectiveTransport returns the explicit transport or the compatibility
// default. Local command servers remain stdio; servers with a URL default to
// streamable HTTP.
func (s MCPServer) EffectiveTransport() MCPServerTransport {
	if s.Transport != "" {
		return s.Transport
	}
	if strings.TrimSpace(s.URL) != "" {
		return MCPServerTransportHTTP
	}
	return MCPServerTransportStdio
}

// Validate reports whether the transport is supported.
func (t MCPServerTransport) Validate(path string) error {
	switch t {
	case "", MCPServerTransportStdio, MCPServerTransportHTTP, MCPServerTransportSSE:
		return nil
	default:
		return fmt.Errorf("%s must be one of stdio, http, or sse", path)
	}
}

// IsZero reports whether the auth config is empty.
func (a MCPAuthConfig) IsZero() bool {
	return strings.TrimSpace(string(a.Type)) == "" &&
		strings.TrimSpace(a.IssuerURL) == "" &&
		strings.TrimSpace(a.MetadataURL) == "" &&
		strings.TrimSpace(a.AuthorizationURL) == "" &&
		strings.TrimSpace(a.TokenURL) == "" &&
		strings.TrimSpace(a.RevocationURL) == "" &&
		strings.TrimSpace(a.ClientID) == "" &&
		strings.TrimSpace(a.ClientSecretEnv) == "" &&
		len(a.Scopes) == 0
}

// Enabled reports whether auth is configured.
func (a MCPAuthConfig) Enabled() bool {
	return !a.IsZero()
}

// Validate ensures remote MCP OAuth configuration has enough metadata to run
// the authorization-code flow without placing token material in config files.
func (a MCPAuthConfig) Validate(path string) error {
	if a.IsZero() {
		return nil
	}
	if a.Type != MCPAuthTypeOAuth2PKCE {
		return fmt.Errorf("%s.type must be %q", path, MCPAuthTypeOAuth2PKCE)
	}
	if strings.TrimSpace(a.ClientID) == "" {
		return fmt.Errorf("%s.client_id is required", path)
	}
	if strings.TrimSpace(a.MetadataURL) == "" &&
		strings.TrimSpace(a.IssuerURL) == "" &&
		(strings.TrimSpace(a.AuthorizationURL) == "" || strings.TrimSpace(a.TokenURL) == "") {
		return fmt.Errorf(
			"%s requires metadata_url, issuer_url, or both authorization_url and token_url",
			path,
		)
	}
	if strings.ContainsAny(strings.TrimSpace(a.ClientSecretEnv), " =\t\r\n") {
		return fmt.Errorf("%s.client_secret_env must be an environment variable name", path)
	}
	for idx, scope := range a.Scopes {
		if strings.TrimSpace(scope) == "" {
			return fmt.Errorf("%s.scopes[%d] is required", path, idx)
		}
	}
	return nil
}

func validateResolvedProvider(name string, provider ProviderConfig) error {
	if strings.TrimSpace(provider.Command) == "" {
		return fmt.Errorf("provider %q command is required", name)
	}

	for i, server := range provider.MCPServers {
		if err := server.Validate(fmt.Sprintf("providers.%s.mcp_servers[%d]", name, i)); err != nil {
			return err
		}
	}

	return nil
}

func mergeMCPServer(base MCPServer, override MCPServer) MCPServer {
	merged := cloneMCPServer(base)
	if strings.TrimSpace(override.Name) != "" {
		merged.Name = override.Name
	}
	if override.Transport != "" {
		merged.Transport = override.Transport
	}
	if strings.TrimSpace(override.Command) != "" {
		merged.Command = override.Command
	}
	if len(override.Args) > 0 {
		merged.Args = append([]string(nil), override.Args...)
	}
	if len(override.Env) > 0 {
		merged.Env = mergeStringMaps(merged.Env, override.Env)
	}
	if strings.TrimSpace(override.URL) != "" {
		merged.URL = override.URL
	}
	if !override.Auth.IsZero() {
		merged.Auth = mergeMCPAuthConfig(merged.Auth, override.Auth)
	}

	return merged
}

func mergeMCPAuthConfig(base MCPAuthConfig, override MCPAuthConfig) MCPAuthConfig {
	merged := cloneMCPAuthConfig(base)
	if override.Type != "" {
		merged.Type = override.Type
	}
	if strings.TrimSpace(override.IssuerURL) != "" {
		merged.IssuerURL = override.IssuerURL
	}
	if strings.TrimSpace(override.MetadataURL) != "" {
		merged.MetadataURL = override.MetadataURL
	}
	if strings.TrimSpace(override.AuthorizationURL) != "" {
		merged.AuthorizationURL = override.AuthorizationURL
	}
	if strings.TrimSpace(override.TokenURL) != "" {
		merged.TokenURL = override.TokenURL
	}
	if strings.TrimSpace(override.RevocationURL) != "" {
		merged.RevocationURL = override.RevocationURL
	}
	if strings.TrimSpace(override.ClientID) != "" {
		merged.ClientID = override.ClientID
	}
	if strings.TrimSpace(override.ClientSecretEnv) != "" {
		merged.ClientSecretEnv = override.ClientSecretEnv
	}
	if len(override.Scopes) > 0 {
		merged.Scopes = append([]string(nil), override.Scopes...)
	}
	return merged
}

func cloneProviders(src map[string]ProviderConfig) map[string]ProviderConfig {
	if len(src) == 0 {
		return map[string]ProviderConfig{}
	}

	cloned := make(map[string]ProviderConfig, len(src))
	for name, provider := range src {
		cloned[name] = cloneProvider(provider)
	}

	return cloned
}

func cloneProvider(src ProviderConfig) ProviderConfig {
	return ProviderConfig{
		Command:      src.Command,
		DefaultModel: src.DefaultModel,
		APIKeyEnv:    src.APIKeyEnv,
		MCPServers:   cloneMCPServers(src.MCPServers),
	}
}

func cloneMCPServers(src []MCPServer) []MCPServer {
	return cloneMCPServersWithCapacity(src, len(src))
}

func cloneMCPServersWithCapacity(src []MCPServer, capacity int) []MCPServer {
	if len(src) == 0 {
		return nil
	}

	if capacity < len(src) {
		capacity = len(src)
	}

	cloned := make([]MCPServer, len(src), capacity)
	for i, server := range src {
		cloned[i] = cloneMCPServer(server)
	}

	return cloned
}

func cloneMCPServer(src MCPServer) MCPServer {
	return MCPServer{
		Name:      src.Name,
		Transport: src.Transport,
		Command:   src.Command,
		Args:      append([]string(nil), src.Args...),
		Env:       mergeStringMaps(nil, src.Env),
		URL:       src.URL,
		Auth:      cloneMCPAuthConfig(src.Auth),
	}
}

func cloneMCPAuthConfig(src MCPAuthConfig) MCPAuthConfig {
	src.Scopes = append([]string(nil), src.Scopes...)
	return src
}

func mergeStringMaps(base map[string]string, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}

	merged := make(map[string]string, len(base)+len(overlay))
	maps.Copy(merged, base)
	maps.Copy(merged, overlay)

	return merged
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	return append([]string(nil), values...)
}
