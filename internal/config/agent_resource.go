package config

import (
	"context"
	"errors"
	"strings"

	"github.com/pedronauck/agh/internal/resources"
)

const (
	// AgentResourceKind is the canonical desired-state resource kind for agent definitions.
	AgentResourceKind     resources.ResourceKind = "agent"
	agentResourceMaxBytes                        = 512 << 10
)

// NewAgentResourceCodec builds the canonical agent resource codec.
func NewAgentResourceCodec() (resources.KindCodec[AgentDef], error) {
	return resources.NewJSONCodec(AgentResourceKind, agentResourceMaxBytes, validateAgentResourceSpec)
}

func validateAgentResourceSpec(
	_ context.Context,
	scope resources.ResourceScope,
	spec AgentDef,
) (AgentDef, error) {
	normalizedScope := scope.Normalize()
	if err := normalizedScope.Validate("scope"); err != nil {
		return AgentDef{}, err
	}

	normalized := AgentDef{
		Name:        strings.TrimSpace(spec.Name),
		Provider:    strings.TrimSpace(spec.Provider),
		Command:     strings.TrimSpace(spec.Command),
		Model:       strings.TrimSpace(spec.Model),
		Tools:       normalizeAgentToolRefs(spec.Tools),
		Permissions: strings.TrimSpace(spec.Permissions),
		MCPServers:  cloneMCPServers(spec.MCPServers),
		Hooks:       cloneHookDecls(spec.Hooks),
		Prompt:      strings.TrimSpace(spec.Prompt),
	}
	if spec.Capabilities != nil {
		capabilities, err := normalizeCapabilityCatalog(spec.Capabilities, "agent.capabilities")
		if err != nil {
			return AgentDef{}, errors.Join(resources.ErrValidation, err)
		}
		normalized.Capabilities = capabilities
	}
	for idx := range normalized.MCPServers {
		normalized.MCPServers[idx].Name = strings.TrimSpace(normalized.MCPServers[idx].Name)
		normalized.MCPServers[idx].Command = strings.TrimSpace(normalized.MCPServers[idx].Command)
	}

	if err := normalized.Validate(); err != nil {
		return AgentDef{}, errors.Join(resources.ErrValidation, err)
	}
	return normalized, nil
}

func normalizeAgentToolRefs(values []string) []string {
	refs := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		refs = append(refs, trimmed)
	}
	if len(refs) == 0 {
		return []string{"*"}
	}
	return refs
}
