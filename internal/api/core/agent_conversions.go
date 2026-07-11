package core

import (
	"github.com/compozy/agh/internal/api/contract"
	aghconfig "github.com/compozy/agh/internal/config"
	workspacepkg "github.com/compozy/agh/internal/workspace"
)

// AgentPayloadFromDef converts an agent definition into the shared payload.
func AgentPayloadFromDef(agent aghconfig.AgentDef) contract.AgentPayload {
	return AgentPayloadFromEntry(AgentCatalogEntry{
		Def:    agent,
		Origin: contract.AgentOriginGlobal,
	})
}

// AgentCatalogEntryFromDef attaches origin truth to one disk-loaded definition.
func AgentCatalogEntryFromDef(
	home aghconfig.HomePaths,
	agent aghconfig.AgentDef,
	workspaceID string,
) AgentCatalogEntry {
	origin, _ := aghconfig.AgentOriginFor(home, agent.SourcePath)
	entry := AgentCatalogEntry{Def: aghconfig.CloneAgentDef(agent)}
	if origin == aghconfig.AgentOriginWorkspace || (origin == "" && workspaceID != "") {
		entry.Origin = contract.AgentOriginWorkspace
		entry.WorkspaceID = workspaceID
		return entry
	}
	entry.Origin = contract.AgentOriginGlobal
	return entry
}

// AgentPayloadFromEntry converts an origin-aware catalog entry into the shared payload.
func AgentPayloadFromEntry(entry AgentCatalogEntry) contract.AgentPayload {
	agent := entry.Def
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
	digest, digestErr := aghconfig.AgentDefinitionDigest(agent)
	diagnostics := make([]contract.AgentDiagnosticPayload, 0, 1)
	if digestErr != nil {
		diagnostics = append(diagnostics, contract.AgentDiagnosticPayload{
			Path:      agent.SourcePath,
			ErrorKind: "definition_digest",
			Message:   digestErr.Error(),
		})
	}
	disabledSkills := append([]string(nil), agent.Skills.Disabled...)
	return contract.AgentPayload{
		Name:             agent.Name,
		Provider:         agent.Provider,
		Command:          agent.Command,
		Model:            agent.Model,
		ReasoningEffort:  contract.ReasoningEffort(agent.ReasoningEffort),
		Tools:            append([]string(nil), agent.Tools...),
		Toolsets:         append([]string(nil), agent.Toolsets...),
		DenyTools:        append([]string(nil), agent.DenyTools...),
		Permissions:      agent.Permissions,
		CategoryPath:     append([]string(nil), agent.CategoryPath...),
		MCPServers:       mcpServers,
		Origin:           entry.Origin,
		WorkspaceID:      entry.WorkspaceID,
		Skills:           &contract.CreateAgentSkillsConfig{Disabled: disabledSkills},
		DefinitionDigest: digest,
		Prompt:           agent.Prompt,
		Diagnostics:      diagnostics,
	}
}

// AgentPayloadsFromEntries converts origin-aware catalog entries into response payloads.
func AgentPayloadsFromEntries(entries []AgentCatalogEntry) []contract.AgentPayload {
	payloads := make([]contract.AgentPayload, 0, len(entries))
	for _, entry := range entries {
		if aghconfig.IsPublicAgentDef(entry.Def) {
			payloads = append(payloads, AgentPayloadFromEntry(entry))
		}
	}
	return payloads
}

// AgentPayloadFromDiagnostic converts a malformed workspace agent diagnostic into a payload row.
func AgentPayloadFromDiagnostic(
	diagnostic workspacepkg.AgentDiagnostic,
	workspaceID ...string,
) contract.AgentPayload {
	resolvedWorkspaceID := ""
	if len(workspaceID) > 0 {
		resolvedWorkspaceID = workspaceID[0]
	}
	return contract.AgentPayload{
		Name:        diagnostic.Name,
		Provider:    "",
		Origin:      contract.AgentOriginWorkspace,
		WorkspaceID: resolvedWorkspaceID,
		Prompt:      "",
		Diagnostics: []contract.AgentDiagnosticPayload{{
			Path:      diagnostic.Path,
			ErrorKind: diagnostic.ErrorKind,
			Message:   diagnostic.Message,
		}},
	}
}
