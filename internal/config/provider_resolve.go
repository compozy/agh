package config

import (
	"errors"
	"fmt"

	"strings"
)

// ResolveProvider resolves a provider using the built-in registry and config overrides.
func (c *Config) ResolveProvider(name string) (ProviderConfig, error) {
	providerName := CanonicalProviderName(name)
	if providerName == "" {
		return ProviderConfig{}, errors.New("provider name is required")
	}

	resolved, hasBuiltin := builtinProviders[providerName]
	if hasBuiltin {
		resolved = cloneProvider(resolved)
	}
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
	resolved.Models.Default = strings.TrimSpace(resolved.Models.Default)

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
		model = strings.TrimSpace(provider.Models.Default)
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
		ReasoningEffort: strings.TrimSpace(agent.ReasoningEffort),
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
		NoneSecurity:    provider.EffectiveNoneSecurity(),
		AuthStatusCmd:   strings.TrimSpace(provider.AuthStatusCmd),
		AuthLoginCmd:    strings.TrimSpace(provider.AuthLoginCmd),
		SessionMCP:      provider.SessionMCPEnabled(),
		CredentialSlots: provider.EffectiveCredentialSlots(),
		MCPServers:      mergeMCPServerLayers(mcpServers, provider.MCPServers, agent.MCPServers),
		Reasoning:       provider.Models.Reasoning,
		Prompt:          agent.Prompt,
	}
}

// ResolveSessionAgent resolves a parsed agent definition for one session.
// When providerOverride is set, the selected provider becomes canonical and
// provider-owned runtime fields are re-resolved from that provider to avoid
// mixed runtimes from the original agent definition.
func (c *Config) ResolveSessionAgent(agent AgentDef, providerOverride string) (ResolvedAgent, error) {
	return c.ResolveSessionAgentWithRuntime(agent, providerOverride, "")
}

// ResolveSessionAgentWithRuntime resolves one session agent with runtime-level provider/model overrides.
func (c *Config) ResolveSessionAgentWithRuntime(
	agent AgentDef,
	providerOverride string,
	modelOverride string,
) (ResolvedAgent, error) {
	override := CanonicalProviderName(providerOverride)
	model := strings.TrimSpace(modelOverride)
	if override == "" && model == "" {
		return c.ResolveAgent(agent)
	}

	effectiveProvider := CanonicalProviderName(agent.Provider)
	if effectiveProvider == "" && c != nil {
		effectiveProvider = CanonicalProviderName(c.Defaults.Provider)
	}
	if override == "" || override == effectiveProvider {
		sessionAgent := agent
		if model != "" {
			sessionAgent.Model = model
		}
		return c.ResolveAgent(sessionAgent)
	}

	sessionAgent := agent
	sessionAgent.Provider = override
	sessionAgent.Command = ""
	sessionAgent.Model = ""
	sessionAgent.ReasoningEffort = ""
	if model != "" {
		sessionAgent.Model = model
	}

	resolved, err := c.ResolveAgent(sessionAgent)
	if err != nil {
		return ResolvedAgent{}, fmt.Errorf("resolve session agent with provider %q: %w", override, err)
	}

	return resolved, nil
}
