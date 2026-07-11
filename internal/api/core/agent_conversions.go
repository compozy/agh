package core

import (
	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

// AgentPayloadFromDef converts an agent definition into the shared payload.
func AgentPayloadFromDef(agent aghconfig.AgentDef) contract.AgentPayload {
	mcpServers := make([]contract.AgentMCPServerJSON, 0, len(agent.MCPServers))
	for _, server := range agent.MCPServers {
		redacted := aghconfig.RedactedMCPServer(server)
		mcpServers = append(mcpServers, contract.AgentMCPServerJSON{
			Name:      redacted.Name,
			Transport: string(redacted.Transport),
			Command:   redacted.Command,
			Args:      append([]string(nil), redacted.Args...),
			Env:       redacted.Env,
			SecretEnv: redacted.SecretEnv,
			URL:       redacted.URL,
			Auth:      settingsMCPAuthConfigPayload(redacted.Auth),
		})
	}
	return contract.AgentPayload{
		Name:            agent.Name,
		Provider:        agent.Provider,
		Command:         agent.Command,
		Model:           agent.Model,
		ReasoningEffort: contract.ReasoningEffort(agent.ReasoningEffort),
		Tools:           append([]string(nil), agent.Tools...),
		Toolsets:        append([]string(nil), agent.Toolsets...),
		DenyTools:       append([]string(nil), agent.DenyTools...),
		Permissions:     agent.Permissions,
		CategoryPath:    append([]string(nil), agent.CategoryPath...),
		MCPServers:      mcpServers,
		Prompt:          agent.Prompt,
	}
}

// AgentPayloadFromDiagnostic converts a malformed workspace agent diagnostic into a payload row.
func AgentPayloadFromDiagnostic(diagnostic workspacepkg.AgentDiagnostic) contract.AgentPayload {
	return contract.AgentPayload{
		Name:     diagnostic.Name,
		Provider: "",
		Prompt:   "",
		Diagnostics: []contract.AgentDiagnosticPayload{{
			Path:      diagnostic.Path,
			ErrorKind: diagnostic.ErrorKind,
			Message:   diagnostic.Message,
		}},
	}
}
