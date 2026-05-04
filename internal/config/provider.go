package config

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"

	"github.com/pedronauck/agh/internal/vault"
)

var providerSecretRefSegmentPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]*$`)

// ProviderHarness identifies the runtime strategy used to launch a provider.
type ProviderHarness string

const (
	// ProviderHarnessACP launches the configured command directly as an ACP runtime.
	ProviderHarnessACP ProviderHarness = "acp"
	// ProviderHarnessPiACP launches pi through the pi-acp adapter and materializes provider settings.
	ProviderHarnessPiACP ProviderHarness = "pi_acp"
)

// ProviderAuthMode identifies who owns launch-time provider authentication.
type ProviderAuthMode string

const (
	// ProviderAuthModeNativeCLI lets the provider CLI use its own login/session state.
	ProviderAuthModeNativeCLI ProviderAuthMode = "native_cli"
	// ProviderAuthModeBoundSecret injects explicitly configured credential slots at launch.
	ProviderAuthModeBoundSecret ProviderAuthMode = "bound_secret"
	// ProviderAuthModeNone launches the provider without AGH-managed credentials.
	ProviderAuthModeNone ProviderAuthMode = "none"
)

// ProviderEnvPolicy identifies which daemon environment is inherited by a provider process.
type ProviderEnvPolicy string

const (
	// ProviderEnvPolicyFiltered removes secret-shaped daemon variables but keeps operator context.
	ProviderEnvPolicyFiltered ProviderEnvPolicy = "filtered"
	// ProviderEnvPolicyIsolated keeps only a fixed operational allowlist.
	ProviderEnvPolicyIsolated ProviderEnvPolicy = "isolated"
)

// ProviderHomePolicy identifies whether provider CLI state comes from the operator home or an isolated home.
type ProviderHomePolicy string

const (
	// ProviderHomePolicyOperator lets native CLIs read their existing operator login state.
	ProviderHomePolicyOperator ProviderHomePolicy = "operator"
	// ProviderHomePolicyIsolated points native CLIs at an AGH-owned provider home.
	ProviderHomePolicyIsolated ProviderHomePolicy = "isolated"
)

// ProviderCredentialSlot describes one launch-time secret binding needed by a provider.
type ProviderCredentialSlot struct {
	Name      string `toml:"name"`
	TargetEnv string `toml:"target_env"`
	SecretRef string `toml:"secret_ref"`
	Kind      string `toml:"kind,omitempty"`
	Required  bool   `toml:"required"`
}

// ProviderConfig describes how to launch a provider in ACP mode.
type ProviderConfig struct {
	Command         string                   `toml:"command"`
	DisplayName     string                   `toml:"display_name,omitempty"`
	DefaultModel    string                   `toml:"default_model"`
	Harness         ProviderHarness          `toml:"harness,omitempty"`
	RuntimeProvider string                   `toml:"runtime_provider,omitempty"`
	Transport       string                   `toml:"transport,omitempty"`
	BaseURL         string                   `toml:"base_url,omitempty"`
	AuthMode        ProviderAuthMode         `toml:"auth_mode,omitempty"`
	EnvPolicy       ProviderEnvPolicy        `toml:"env_policy,omitempty"`
	HomePolicy      ProviderHomePolicy       `toml:"home_policy,omitempty"`
	AuthStatusCmd   string                   `toml:"auth_status_command,omitempty"`
	AuthLoginCmd    string                   `toml:"auth_login_command,omitempty"`
	SessionMCP      *bool                    `toml:"session_mcp,omitempty"`
	Aliases         []string                 `toml:"aliases,omitempty"`
	CredentialSlots []ProviderCredentialSlot `toml:"credential_slots,omitempty"`
	MCPServers      []MCPServer              `toml:"mcp_servers,omitempty"`
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
// metadata and secret refs only; token material is persisted through the
// vault-backed auth token store.
type MCPAuthConfig struct {
	Type             MCPAuthType `json:"type,omitempty"              yaml:"type,omitempty"              toml:"type,omitempty"`
	IssuerURL        string      `json:"issuer_url,omitempty"        yaml:"issuer_url,omitempty"        toml:"issuer_url,omitempty"`
	MetadataURL      string      `json:"metadata_url,omitempty"      yaml:"metadata_url,omitempty"      toml:"metadata_url,omitempty"`
	AuthorizationURL string      `json:"authorization_url,omitempty" yaml:"authorization_url,omitempty" toml:"authorization_url,omitempty"`
	TokenURL         string      `json:"token_url,omitempty"         yaml:"token_url,omitempty"         toml:"token_url,omitempty"`
	RevocationURL    string      `json:"revocation_url,omitempty"    yaml:"revocation_url,omitempty"    toml:"revocation_url,omitempty"`
	ClientID         string      `json:"client_id,omitempty"         yaml:"client_id,omitempty"         toml:"client_id,omitempty"`
	ClientSecretRef  string      `json:"client_secret_ref,omitempty" yaml:"client_secret_ref,omitempty" toml:"client_secret_ref,omitempty"`
	Scopes           []string    `json:"scopes,omitempty"            yaml:"scopes,omitempty"            toml:"scopes,omitempty"`
}

// MCPServer describes an MCP server passed through to the agent runtime.
type MCPServer struct {
	Name      string             `json:"name"                 yaml:"name"                 toml:"name"`
	Transport MCPServerTransport `json:"transport,omitempty"  yaml:"transport,omitempty"  toml:"transport,omitempty"`
	Command   string             `json:"command,omitempty"    yaml:"command,omitempty"    toml:"command,omitempty"`
	Args      []string           `json:"args,omitempty"       yaml:"args,omitempty"       toml:"args,omitempty"`
	Env       map[string]string  `json:"env,omitempty"        yaml:"env,omitempty"        toml:"env,omitempty"`
	SecretEnv map[string]string  `json:"secret_env,omitempty" yaml:"secret_env,omitempty" toml:"secret_env,omitempty"`
	URL       string             `json:"url,omitempty"        yaml:"url,omitempty"        toml:"url,omitempty"`
	Auth      MCPAuthConfig      `json:"auth"                 yaml:"auth,omitempty"       toml:"auth,omitempty"`
}

// ResolvedAgent is the effective runtime configuration for a parsed agent definition.
type ResolvedAgent struct {
	Name            string
	Provider        string
	Command         string
	DisplayName     string
	Model           string
	Tools           []string
	Toolsets        []string
	DenyTools       []string
	Permissions     string
	Harness         ProviderHarness
	RuntimeProvider string
	Transport       string
	BaseURL         string
	AuthMode        ProviderAuthMode
	EnvPolicy       ProviderEnvPolicy
	HomePolicy      ProviderHomePolicy
	AuthStatusCmd   string
	AuthLoginCmd    string
	SessionMCP      bool
	CredentialSlots []ProviderCredentialSlot
	MCPServers      []MCPServer
	Prompt          string
}

// ErrProviderUnavailable reports that a requested provider cannot be resolved
// from the effective workspace/global config.
var ErrProviderUnavailable = errors.New("provider unavailable")

const (
	piACPCommand             = "npx -y pi-acp@latest"
	piACPAuthLoginCommand    = piACPCommand + " --terminal-login"
	providerAPIKeyCredential = "api_key"
)

var builtinProviderAliases = map[string]string{
	"blackbox-ai":        "blackbox",
	"blackboxai":         "blackbox",
	"cline-cli":          "cline",
	"goose-cli":          "goose",
	"hermes-agent":       "hermes",
	"junie-cli":          "junie",
	"ai-gateway":         "vercel-ai-gateway",
	"kimi":               "moonshot",
	"kimi cli":           "kimi-cli",
	"kimi-cli":           "kimi-cli",
	"kimi-code":          "kimi-cli",
	"kimi-coding":        "moonshot",
	"moonshotai":         "moonshot",
	"open-hands":         "openhands",
	"openhands-cli":      "openhands",
	"openclaw-cli":       "openclaw",
	"qoder-cli":          "qoder",
	"qwen":               "qwen-code",
	"qwen cli":           "qwen-code",
	"qwen code":          "qwen-code",
	"qwen-code":          "qwen-code",
	"vercel":             "vercel-ai-gateway",
	"vercel-gateway":     "vercel-ai-gateway",
	"vercel-ai":          "vercel-ai-gateway",
	"vercel-ai-gateway":  "vercel-ai-gateway",
	"z.ai":               "zai",
	"z-ai":               "zai",
	"z_ai":               "zai",
	"glm":                "zai",
	"openrouter-ai":      "openrouter",
	"openrouter-gateway": "openrouter",
	"minimax-ai":         "minimax",
	"minimax-cn":         "minimax",
	"grok":               "xai",
	"x-ai":               "xai",
	"mistralai":          "mistral",
	"mistral-ai":         "mistral",
}

var builtinProviders = map[string]ProviderConfig{
	"claude": {
		Command:      "npx -y @agentclientprotocol/claude-agent-acp@latest",
		DisplayName:  "Claude Code",
		Harness:      ProviderHarnessACP,
		DefaultModel: "claude-sonnet-4-6",
	},
	"codex": {
		Command:      "npx -y @zed-industries/codex-acp@latest",
		DisplayName:  "Codex",
		Harness:      ProviderHarnessACP,
		DefaultModel: "gpt-5.4",
	},
	"gemini": {
		Command:      "gemini --acp",
		DisplayName:  "Gemini CLI",
		Harness:      ProviderHarnessACP,
		DefaultModel: "gemini-3.1-pro-preview",
	},
	"opencode": {
		Command:     "npx -y opencode-ai@latest acp",
		DisplayName: "OpenCode",
		Harness:     ProviderHarnessACP,
	},
	"blackbox": {
		Command:     "blackbox --experimental-acp",
		DisplayName: "BLACKBOX AI",
		Harness:     ProviderHarnessACP,
	},
	"cline": {
		Command:     "npx -y cline@latest --acp",
		DisplayName: "Cline",
		Harness:     ProviderHarnessACP,
	},
	"goose": {
		Command:     "goose acp",
		DisplayName: "Goose",
		Harness:     ProviderHarnessACP,
	},
	"hermes": {
		Command:     "hermes acp",
		DisplayName: "Hermes",
		Harness:     ProviderHarnessACP,
	},
	"junie": {
		Command:     "junie --acp true",
		DisplayName: "Junie",
		Harness:     ProviderHarnessACP,
	},
	"kimi-cli": {
		Command:     "kimi acp",
		DisplayName: "Kimi CLI",
		Harness:     ProviderHarnessACP,
	},
	"openclaw": {
		Command:     "openclaw acp",
		DisplayName: "OpenClaw",
		Harness:     ProviderHarnessACP,
		SessionMCP:  boolRef(false),
	},
	"openhands": {
		Command:     "openhands acp",
		DisplayName: "OpenHands",
		Harness:     ProviderHarnessACP,
	},
	"qoder": {
		Command:     "npx -y @qoder-ai/qodercli@latest --acp",
		DisplayName: "Qoder CLI",
		Harness:     ProviderHarnessACP,
	},
	"qwen-code": {
		Command:      "npx -y @qwen-code/qwen-code@latest --acp --experimental-skills",
		DisplayName:  "Qwen Code",
		Harness:      ProviderHarnessACP,
		DefaultModel: "qwen3.6-plus",
	},
	"copilot": {
		Command:     "copilot --acp --stdio",
		DisplayName: "GitHub Copilot CLI",
		Harness:     ProviderHarnessACP,
	},
	"cursor": {
		Command:     "cursor-agent acp",
		DisplayName: "Cursor Agent",
		Harness:     ProviderHarnessACP,
	},
	"kiro": {
		Command:     "kiro-cli-chat acp",
		DisplayName: "Kiro CLI",
		Harness:     ProviderHarnessACP,
	},
	"pi": {
		Command:         piACPCommand,
		DisplayName:     "Pi",
		Harness:         ProviderHarnessPiACP,
		RuntimeProvider: "anthropic",
		DefaultModel:    "claude-opus-4-7",
		AuthLoginCmd:    piACPAuthLoginCommand,
	},
	"openrouter": {
		Command:         piACPCommand,
		DisplayName:     "OpenRouter",
		Harness:         ProviderHarnessPiACP,
		RuntimeProvider: "openrouter",
		DefaultModel:    "openai/gpt-5.4",
		CredentialSlots: []ProviderCredentialSlot{apiKeyCredentialSlot("OPENROUTER_API_KEY")},
	},
	"zai": {
		Command:         piACPCommand,
		DisplayName:     "z.ai",
		Harness:         ProviderHarnessPiACP,
		RuntimeProvider: "zai",
		DefaultModel:    "glm-4.6",
		CredentialSlots: []ProviderCredentialSlot{apiKeyCredentialSlot("ZAI_API_KEY")},
	},
	"moonshot": {
		Command:         piACPCommand,
		DisplayName:     "Moonshot / Kimi",
		Harness:         ProviderHarnessPiACP,
		RuntimeProvider: "kimi-coding",
		DefaultModel:    "kimi-k2-thinking",
		CredentialSlots: []ProviderCredentialSlot{apiKeyCredentialSlot("KIMI_API_KEY")},
	},
	"vercel-ai-gateway": {
		Command:         piACPCommand,
		DisplayName:     "Vercel AI Gateway",
		Harness:         ProviderHarnessPiACP,
		RuntimeProvider: "vercel-ai-gateway",
		DefaultModel:    "anthropic/claude-opus-4-7",
		CredentialSlots: []ProviderCredentialSlot{apiKeyCredentialSlot("AI_GATEWAY_API_KEY")},
	},
	"xai": {
		Command:         piACPCommand,
		DisplayName:     "xAI",
		Harness:         ProviderHarnessPiACP,
		RuntimeProvider: "xai",
		DefaultModel:    "grok-4-fast-non-reasoning",
		CredentialSlots: []ProviderCredentialSlot{apiKeyCredentialSlot("XAI_API_KEY")},
	},
	"minimax": {
		Command:         piACPCommand,
		DisplayName:     "MiniMax",
		Harness:         ProviderHarnessPiACP,
		RuntimeProvider: "minimax",
		DefaultModel:    "MiniMax-M2.1",
		CredentialSlots: []ProviderCredentialSlot{apiKeyCredentialSlot("MINIMAX_API_KEY")},
	},
	"mistral": {
		Command:         piACPCommand,
		DisplayName:     "Mistral",
		Harness:         ProviderHarnessPiACP,
		RuntimeProvider: "mistral",
		DefaultModel:    "devstral-medium-latest",
		CredentialSlots: []ProviderCredentialSlot{apiKeyCredentialSlot("MISTRAL_API_KEY")},
	},
	"groq": {
		Command:         piACPCommand,
		DisplayName:     "Groq",
		Harness:         ProviderHarnessPiACP,
		RuntimeProvider: "groq",
		DefaultModel:    "openai/gpt-oss-120b",
		CredentialSlots: []ProviderCredentialSlot{apiKeyCredentialSlot("GROQ_API_KEY")},
	},
}

// BuiltinProviders returns a deep copy of the built-in provider registry.
func BuiltinProviders() map[string]ProviderConfig {
	return cloneProviders(builtinProviders)
}

// CanonicalProviderName resolves known builtin aliases to the stable provider id.
func CanonicalProviderName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	if _, ok := builtinProviders[trimmed]; ok {
		return trimmed
	}
	if canonical, ok := builtinProviderAliases[strings.ToLower(trimmed)]; ok {
		return canonical
	}
	return trimmed
}

func apiKeyCredentialSlot(targetEnv string) ProviderCredentialSlot {
	return apiKeyCredentialSlotWithRequired(targetEnv, true)
}

func apiKeyCredentialSlotWithRequired(targetEnv string, required bool) ProviderCredentialSlot {
	return ProviderCredentialSlot{
		Name:      providerAPIKeyCredential,
		TargetEnv: targetEnv,
		SecretRef: "env:" + targetEnv,
		Kind:      providerAPIKeyCredential,
		Required:  required,
	}
}

// ResolveProvider resolves a provider using the built-in registry and config overrides.
func (c *Config) ResolveProvider(name string) (ProviderConfig, error) {
	providerName := CanonicalProviderName(name)
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

	providerName := CanonicalProviderName(agent.Provider)
	if providerName == "" {
		providerName = CanonicalProviderName(defaults.Provider)
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
	if model == "" && provider.RequiresRuntimeModel() {
		return ResolvedAgent{}, fmt.Errorf(
			"agent model is required when provider %q has no default model",
			providerName,
		)
	}

	resolved := resolvedAgentFromProvider(
		agent,
		providerName,
		provider,
		resolvedPermissions,
		command,
		model,
		mcpServers,
	)

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

func resolvedAgentFromProvider(
	agent AgentDef,
	providerName string,
	provider ProviderConfig,
	resolvedPermissions string,
	command string,
	model string,
	mcpServers []MCPServer,
) ResolvedAgent {
	return ResolvedAgent{
		Name:            agent.Name,
		Provider:        providerName,
		Command:         command,
		DisplayName:     provider.DisplayName,
		Model:           model,
		Tools:           cloneStrings(agent.Tools),
		Toolsets:        cloneStrings(agent.Toolsets),
		DenyTools:       cloneStrings(agent.DenyTools),
		Permissions:     resolvedPermissions,
		Harness:         provider.EffectiveHarness(),
		RuntimeProvider: provider.RuntimeProviderName(providerName),
		Transport:       strings.TrimSpace(provider.Transport),
		BaseURL:         strings.TrimSpace(provider.BaseURL),
		AuthMode:        provider.EffectiveAuthMode(),
		EnvPolicy:       provider.EffectiveEnvPolicy(),
		HomePolicy:      provider.EffectiveHomePolicy(),
		AuthStatusCmd:   strings.TrimSpace(provider.AuthStatusCmd),
		AuthLoginCmd:    strings.TrimSpace(provider.AuthLoginCmd),
		SessionMCP:      provider.SessionMCPEnabled(),
		CredentialSlots: provider.EffectiveCredentialSlots(),
		MCPServers:      mergeMCPServerLayers(mcpServers, provider.MCPServers, agent.MCPServers),
		Prompt:          agent.Prompt,
	}
}

// ResolveSessionAgent resolves a parsed agent definition for one session.
// When providerOverride is set, the selected provider becomes canonical and
// provider-owned runtime fields are re-resolved from that provider to avoid
// mixed runtimes from the original agent definition.
func (c *Config) ResolveSessionAgent(agent AgentDef, providerOverride string) (ResolvedAgent, error) {
	override := CanonicalProviderName(providerOverride)
	if override == "" {
		return c.ResolveAgent(agent)
	}

	effectiveProvider := CanonicalProviderName(agent.Provider)
	if effectiveProvider == "" && c != nil {
		effectiveProvider = CanonicalProviderName(c.Defaults.Provider)
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
	if strings.TrimSpace(override.DisplayName) != "" {
		merged.DisplayName = override.DisplayName
	}
	if strings.TrimSpace(override.DefaultModel) != "" {
		merged.DefaultModel = override.DefaultModel
	}
	if override.Harness != "" {
		merged.Harness = override.Harness
	}
	if strings.TrimSpace(override.RuntimeProvider) != "" {
		merged.RuntimeProvider = override.RuntimeProvider
	}
	if strings.TrimSpace(override.Transport) != "" {
		merged.Transport = override.Transport
	}
	if strings.TrimSpace(override.BaseURL) != "" {
		merged.BaseURL = override.BaseURL
	}
	if override.AuthMode != "" {
		merged.AuthMode = override.AuthMode
	}
	if override.EnvPolicy != "" {
		merged.EnvPolicy = override.EnvPolicy
	}
	if override.HomePolicy != "" {
		merged.HomePolicy = override.HomePolicy
	}
	if strings.TrimSpace(override.AuthStatusCmd) != "" {
		merged.AuthStatusCmd = override.AuthStatusCmd
	}
	if strings.TrimSpace(override.AuthLoginCmd) != "" {
		merged.AuthLoginCmd = override.AuthLoginCmd
	}
	if override.SessionMCP != nil {
		merged.SessionMCP = boolRef(*override.SessionMCP)
	}
	if len(override.Aliases) > 0 {
		merged.Aliases = cloneStrings(override.Aliases)
	}
	if len(override.CredentialSlots) > 0 {
		merged.CredentialSlots = cloneProviderCredentialSlots(override.CredentialSlots)
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
	case transport != MCPServerTransportStdio && len(s.SecretEnv) > 0:
		return fmt.Errorf("%s.secret_env is only valid for stdio transport", path)
	default:
		return validateStdioMCPEnv(path, transport, s.Env, s.SecretEnv)
	}
}

func validateStdioMCPEnv(
	path string,
	transport MCPServerTransport,
	env map[string]string,
	secretEnv map[string]string,
) error {
	if transport != MCPServerTransportStdio {
		return nil
	}
	for key := range env {
		if forbiddenStdioMCPEnvKey(key) {
			return fmt.Errorf("%s.env.%s is forbidden for stdio MCP servers", path, strings.TrimSpace(key))
		}
		if vault.SecretLikeEnvName(key) {
			return fmt.Errorf("%s.env.%s must move secret-like values to secret_env", path, strings.TrimSpace(key))
		}
	}
	return vault.ValidateSecretEnvMap(path, "mcp", secretEnv)
}

func forbiddenStdioMCPEnvKey(key string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(key))
	switch normalized {
	case "NODE_OPTIONS", "PYTHONPATH", "PYTHONHOME", "LD_PRELOAD":
		return true
	default:
		return strings.HasPrefix(normalized, "DYLD_")
	}
}

// EffectiveHarness returns the configured provider harness or the command-backed default.
func (p ProviderConfig) EffectiveHarness() ProviderHarness {
	if p.Harness != "" {
		return p.Harness
	}
	return ProviderHarnessACP
}

// RequiresRuntimeModel reports whether AGH must provide a model to start this provider.
func (p ProviderConfig) RequiresRuntimeModel() bool {
	return p.EffectiveHarness() == ProviderHarnessPiACP
}

// EffectiveAuthMode returns the configured auth owner or the slot-derived default.
func (p ProviderConfig) EffectiveAuthMode() ProviderAuthMode {
	if p.AuthMode != "" {
		return p.AuthMode
	}
	if len(p.EffectiveCredentialSlots()) > 0 {
		return ProviderAuthModeBoundSecret
	}
	return ProviderAuthModeNativeCLI
}

// EffectiveEnvPolicy returns the configured provider environment inheritance policy.
func (p ProviderConfig) EffectiveEnvPolicy() ProviderEnvPolicy {
	if p.EnvPolicy != "" {
		return p.EnvPolicy
	}
	return ProviderEnvPolicyFiltered
}

// EffectiveHomePolicy returns the configured provider home inheritance policy.
func (p ProviderConfig) EffectiveHomePolicy() ProviderHomePolicy {
	if p.HomePolicy != "" {
		return p.HomePolicy
	}
	return ProviderHomePolicyOperator
}

// RuntimeProviderName returns the downstream runtime provider id for harnesses that need one.
func (p ProviderConfig) RuntimeProviderName(providerName string) string {
	if runtimeProvider := strings.TrimSpace(p.RuntimeProvider); runtimeProvider != "" {
		return runtimeProvider
	}
	return strings.TrimSpace(providerName)
}

// EffectiveCredentialSlots returns explicit launch credential slots.
func (p ProviderConfig) EffectiveCredentialSlots() []ProviderCredentialSlot {
	if len(p.CredentialSlots) > 0 {
		return cloneProviderCredentialSlots(p.CredentialSlots)
	}
	return nil
}

// SessionMCPEnabled reports whether AGH should pass per-session MCP servers to the provider.
func (p ProviderConfig) SessionMCPEnabled() bool {
	if p.SessionMCP == nil {
		return true
	}
	return *p.SessionMCP
}

// Validate reports whether the harness is supported.
func (h ProviderHarness) Validate(path string) error {
	switch h {
	case "", ProviderHarnessACP, ProviderHarnessPiACP:
		return nil
	default:
		return fmt.Errorf("%s must be one of acp or pi_acp", path)
	}
}

// Validate reports whether the provider auth mode is supported.
func (m ProviderAuthMode) Validate(path string) error {
	switch m {
	case "", ProviderAuthModeNativeCLI, ProviderAuthModeBoundSecret, ProviderAuthModeNone:
		return nil
	default:
		return fmt.Errorf("%s must be one of native_cli, bound_secret, or none", path)
	}
}

// Validate reports whether the provider env policy is supported.
func (p ProviderEnvPolicy) Validate(path string) error {
	switch p {
	case "", ProviderEnvPolicyFiltered, ProviderEnvPolicyIsolated:
		return nil
	default:
		return fmt.Errorf("%s must be one of filtered or isolated", path)
	}
}

// Validate reports whether the provider home policy is supported.
func (p ProviderHomePolicy) Validate(path string) error {
	switch p {
	case "", ProviderHomePolicyOperator, ProviderHomePolicyIsolated:
		return nil
	default:
		return fmt.Errorf("%s must be one of operator or isolated", path)
	}
}

// Validate reports whether the provider credential slot can be resolved at launch.
func (s ProviderCredentialSlot) Validate(path string) error {
	switch {
	case strings.TrimSpace(s.Name) == "":
		return fmt.Errorf("%s.name is required", path)
	case strings.TrimSpace(s.TargetEnv) == "":
		return fmt.Errorf("%s.target_env is required", path)
	case !vault.EnvNamePattern.MatchString(strings.TrimSpace(s.TargetEnv)):
		return fmt.Errorf("%s.target_env must be an environment variable name", path)
	case strings.TrimSpace(s.SecretRef) == "":
		return fmt.Errorf("%s.secret_ref is required", path)
	case !validProviderSecretRef(s.SecretRef):
		return fmt.Errorf("%s.secret_ref must be env:VAR or vault:providers/<provider>/<slot>", path)
	default:
		return nil
	}
}

func validProviderSecretRef(ref string) bool {
	normalized := vault.NormalizeRef(ref)
	if vault.IsEnvRef(normalized) {
		return vault.ValidateRef(normalized) == nil
	}
	if err := vault.ValidateSecretRefNamespace(normalized, "providers"); err != nil {
		return false
	}
	path := strings.TrimPrefix(normalized, "vault:providers/")
	segments := strings.Split(path, "/")
	if len(segments) < 2 {
		return false
	}
	for _, segment := range segments {
		if !providerSecretRefSegmentPattern.MatchString(segment) {
			return false
		}
	}
	return true
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
		strings.TrimSpace(a.ClientSecretRef) == "" &&
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
	if strings.TrimSpace(a.ClientSecretRef) != "" {
		if err := vault.ValidateRefNamespace(a.ClientSecretRef, "mcp"); err != nil {
			return fmt.Errorf("%s.client_secret_ref is invalid: %w", path, err)
		}
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
	if err := provider.EffectiveHarness().Validate(fmt.Sprintf("providers.%s.harness", name)); err != nil {
		return err
	}
	if err := provider.EffectiveAuthMode().Validate(fmt.Sprintf("providers.%s.auth_mode", name)); err != nil {
		return err
	}
	if err := provider.EffectiveEnvPolicy().Validate(fmt.Sprintf("providers.%s.env_policy", name)); err != nil {
		return err
	}
	if err := provider.EffectiveHomePolicy().Validate(fmt.Sprintf("providers.%s.home_policy", name)); err != nil {
		return err
	}
	if provider.EffectiveHarness() == ProviderHarnessPiACP &&
		strings.TrimSpace(provider.RuntimeProviderName(name)) == "" {
		return fmt.Errorf("providers.%s.runtime_provider is required for pi_acp providers", name)
	}
	slots, err := validateProviderAuthSlots(name, provider)
	if err != nil {
		return err
	}
	for i, slot := range slots {
		if err := slot.Validate(fmt.Sprintf("providers.%s.credential_slots[%d]", name, i)); err != nil {
			return err
		}
	}

	for i, server := range provider.MCPServers {
		if err := server.Validate(fmt.Sprintf("providers.%s.mcp_servers[%d]", name, i)); err != nil {
			return err
		}
	}

	return nil
}

func validateProviderAuthSlots(name string, provider ProviderConfig) ([]ProviderCredentialSlot, error) {
	authMode := provider.EffectiveAuthMode()
	slots := provider.EffectiveCredentialSlots()
	if builtinNativeAuthProvider(name) && provider.AuthMode == "" && len(slots) > 0 {
		return nil, fmt.Errorf(
			"providers.%s.auth_mode must be %q before credential_slots can override native CLI authentication",
			name,
			ProviderAuthModeBoundSecret,
		)
	}
	switch authMode {
	case ProviderAuthModeBoundSecret:
		if len(slots) == 0 {
			return nil, fmt.Errorf("providers.%s.credential_slots is required when auth_mode is bound_secret", name)
		}
	case ProviderAuthModeNativeCLI:
		if len(slots) > 0 {
			return nil, fmt.Errorf(
				"providers.%s.credential_slots requires auth_mode = %q; native_cli uses provider-owned login state",
				name,
				ProviderAuthModeBoundSecret,
			)
		}
	case ProviderAuthModeNone:
		if len(slots) > 0 {
			return nil, fmt.Errorf("providers.%s.credential_slots cannot be set when auth_mode is none", name)
		}
	}
	return slots, nil
}

func builtinNativeAuthProvider(name string) bool {
	builtin, ok := builtinProviders[name]
	return ok && builtin.EffectiveAuthMode() == ProviderAuthModeNativeCLI
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
	if len(override.SecretEnv) > 0 {
		merged.SecretEnv = mergeStringMaps(merged.SecretEnv, override.SecretEnv)
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
	if strings.TrimSpace(override.ClientSecretRef) != "" {
		merged.ClientSecretRef = override.ClientSecretRef
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
		Command:         src.Command,
		DisplayName:     src.DisplayName,
		DefaultModel:    src.DefaultModel,
		Harness:         src.Harness,
		RuntimeProvider: src.RuntimeProvider,
		Transport:       src.Transport,
		BaseURL:         src.BaseURL,
		AuthMode:        src.AuthMode,
		EnvPolicy:       src.EnvPolicy,
		HomePolicy:      src.HomePolicy,
		AuthStatusCmd:   src.AuthStatusCmd,
		AuthLoginCmd:    src.AuthLoginCmd,
		SessionMCP:      cloneBoolRef(src.SessionMCP),
		Aliases:         cloneStrings(src.Aliases),
		CredentialSlots: cloneProviderCredentialSlots(src.CredentialSlots),
		MCPServers:      cloneMCPServers(src.MCPServers),
	}
}

func boolRef(value bool) *bool {
	return &value
}

func cloneBoolRef(src *bool) *bool {
	if src == nil {
		return nil
	}
	return boolRef(*src)
}

func cloneProviderCredentialSlots(src []ProviderCredentialSlot) []ProviderCredentialSlot {
	if len(src) == 0 {
		return nil
	}
	cloned := make([]ProviderCredentialSlot, len(src))
	copy(cloned, src)
	return cloned
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
		SecretEnv: mergeStringMaps(nil, src.SecretEnv),
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
