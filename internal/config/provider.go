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

// MCPServer describes an MCP server passed through to the agent runtime.
type MCPServer struct {
	Name    string            `yaml:"name"           toml:"name"`
	Command string            `yaml:"command"        toml:"command"`
	Args    []string          `yaml:"args,omitempty" toml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"  toml:"env,omitempty"`
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
			return ProviderConfig{}, fmt.Errorf("unknown provider %q", providerName)
		}
		if _, ok := c.Providers[providerName]; !ok {
			return ProviderConfig{}, fmt.Errorf("unknown provider %q", providerName)
		}
	}

	if err := validateResolvedProvider(providerName, resolved); err != nil {
		return ProviderConfig{}, err
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
	switch {
	case strings.TrimSpace(s.Name) == "":
		return fmt.Errorf("%s.name is required", path)
	case strings.TrimSpace(s.Command) == "":
		return fmt.Errorf("%s.command is required", path)
	default:
		return nil
	}
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
	if strings.TrimSpace(override.Command) != "" {
		merged.Command = override.Command
	}
	if len(override.Args) > 0 {
		merged.Args = append([]string(nil), override.Args...)
	}
	if len(override.Env) > 0 {
		merged.Env = mergeStringMaps(merged.Env, override.Env)
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
		Name:    src.Name,
		Command: src.Command,
		Args:    append([]string(nil), src.Args...),
		Env:     mergeStringMaps(nil, src.Env),
	}
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
