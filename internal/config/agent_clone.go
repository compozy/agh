package config

import "strings"

// CloneAgentDef returns a deep copy of an agent definition.
func CloneAgentDef(agent AgentDef) AgentDef {
	return AgentDef{
		Name:         strings.TrimSpace(agent.Name),
		Provider:     strings.TrimSpace(agent.Provider),
		Command:      strings.TrimSpace(agent.Command),
		Model:        strings.TrimSpace(agent.Model),
		Tools:        cloneStrings(agent.Tools),
		Toolsets:     cloneStrings(agent.Toolsets),
		DenyTools:    cloneStrings(agent.DenyTools),
		Permissions:  strings.TrimSpace(agent.Permissions),
		Skills:       normalizeAgentSkillsConfig(agent.Skills),
		MCPServers:   cloneMCPServers(agent.MCPServers),
		Hooks:        cloneHookDecls(agent.Hooks),
		Capabilities: agent.Capabilities.Clone(),
		Prompt:       strings.TrimSpace(agent.Prompt),
		SourcePath:   strings.TrimSpace(agent.SourcePath),
	}
}
