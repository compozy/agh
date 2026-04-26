package workspace

import (
	"maps"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/environment"
	"github.com/pedronauck/agh/internal/filesnap"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
)

func cloneSnapshots(snapshots map[string]filesnap.Snapshot) map[string]filesnap.Snapshot {
	return filesnap.Clone(snapshots)
}

func cloneResolvedWorkspace(src *ResolvedWorkspace) ResolvedWorkspace {
	return ResolvedWorkspace{
		Workspace:   cloneWorkspace(src.Workspace),
		Config:      cloneConfig(&src.Config),
		Agents:      cloneAgentDefs(src.Agents),
		Skills:      cloneSkillPaths(src.Skills),
		Environment: cloneEnvironmentResolved(src.Environment),
		ResolvedAt:  src.ResolvedAt,
	}
}

func cloneWorkspace(src Workspace) Workspace {
	return Workspace{
		ID:             src.ID,
		RootDir:        src.RootDir,
		AdditionalDirs: append([]string(nil), src.AdditionalDirs...),
		Name:           src.Name,
		DefaultAgent:   src.DefaultAgent,
		EnvironmentRef: src.EnvironmentRef,
		CreatedAt:      src.CreatedAt,
		UpdatedAt:      src.UpdatedAt,
	}
}

func cloneWorkspaces(src []Workspace) []Workspace {
	if len(src) == 0 {
		return nil
	}

	cloned := make([]Workspace, 0, len(src))
	for _, ws := range src {
		cloned = append(cloned, cloneWorkspace(ws))
	}
	return cloned
}

func cloneConfig(src *aghconfig.Config) aghconfig.Config {
	return aghconfig.Config{
		Daemon:        src.Daemon,
		HTTP:          src.HTTP,
		Defaults:      src.Defaults,
		Limits:        src.Limits,
		Session:       src.Session,
		Permissions:   src.Permissions,
		MCPServers:    cloneMCPServers(src.MCPServers),
		Providers:     cloneProviders(src.Providers),
		Environments:  cloneEnvironmentProfiles(src.Environments),
		Observability: src.Observability,
		Log:           src.Log,
		Memory:        src.Memory,
		Skills: aghconfig.SkillsConfig{
			Enabled:                 src.Skills.Enabled,
			DisabledSkills:          append([]string(nil), src.Skills.DisabledSkills...),
			PollInterval:            src.Skills.PollInterval,
			AllowedMarketplaceMCP:   append([]string(nil), src.Skills.AllowedMarketplaceMCP...),
			AllowedMarketplaceHooks: append([]string(nil), src.Skills.AllowedMarketplaceHooks...),
			Marketplace:             src.Skills.Marketplace,
		},
		Extensions: src.Extensions,
		Automation: src.Automation,
		Autonomy:   src.Autonomy,
		Hooks: aghconfig.HooksConfig{
			Declarations: cloneHookDecls(src.Hooks.Declarations),
		},
		Network: src.Network,
	}
}

func cloneEnvironmentProfiles(src map[string]aghconfig.EnvironmentProfile) map[string]aghconfig.EnvironmentProfile {
	if len(src) == 0 {
		return map[string]aghconfig.EnvironmentProfile{}
	}

	cloned := make(map[string]aghconfig.EnvironmentProfile, len(src))
	for name, profile := range src {
		cloned[name] = cloneEnvironmentProfile(profile)
	}
	return cloned
}

func cloneEnvironmentProfile(src aghconfig.EnvironmentProfile) aghconfig.EnvironmentProfile {
	return aghconfig.EnvironmentProfile{
		Backend:     src.Backend,
		SyncMode:    src.SyncMode,
		Persistence: src.Persistence,
		RuntimeRoot: src.RuntimeRoot,
		Env:         cloneStringMap(src.Env),
		Network: aghconfig.NetworkProfile{
			AllowPublicIngress: src.Network.AllowPublicIngress,
			AllowOutbound:      src.Network.AllowOutbound,
			AllowList:          append([]string(nil), src.Network.AllowList...),
			DenyList:           append([]string(nil), src.Network.DenyList...),
		},
		Daytona: src.Daytona,
	}
}

func cloneEnvironmentResolved(src environment.Resolved) environment.Resolved {
	cloned := src
	cloned.Env = cloneStringMap(src.Env)
	cloned.Network.AllowList = append([]string(nil), src.Network.AllowList...)
	cloned.Network.DenyList = append([]string(nil), src.Network.DenyList...)
	if src.Daytona != nil {
		daytona := *src.Daytona
		cloned.Daytona = &daytona
	}
	return cloned
}

func cloneProviders(src map[string]aghconfig.ProviderConfig) map[string]aghconfig.ProviderConfig {
	if len(src) == 0 {
		return map[string]aghconfig.ProviderConfig{}
	}

	cloned := make(map[string]aghconfig.ProviderConfig, len(src))
	for name, provider := range src {
		cloned[name] = cloneProvider(provider)
	}
	return cloned
}

func cloneProvider(src aghconfig.ProviderConfig) aghconfig.ProviderConfig {
	return aghconfig.ProviderConfig{
		Command:      src.Command,
		DefaultModel: src.DefaultModel,
		APIKeyEnv:    src.APIKeyEnv,
		MCPServers:   cloneMCPServers(src.MCPServers),
	}
}

func cloneAgentDefs(src []aghconfig.AgentDef) []aghconfig.AgentDef {
	if len(src) == 0 {
		return nil
	}

	cloned := make([]aghconfig.AgentDef, 0, len(src))
	for _, agent := range src {
		cloned = append(cloned, aghconfig.AgentDef{
			Name:         agent.Name,
			Provider:     agent.Provider,
			Command:      agent.Command,
			Model:        agent.Model,
			Tools:        append([]string(nil), agent.Tools...),
			Permissions:  agent.Permissions,
			MCPServers:   cloneMCPServers(agent.MCPServers),
			Hooks:        cloneHookDecls(agent.Hooks),
			Capabilities: agent.Capabilities.Clone(),
			Prompt:       agent.Prompt,
		})
	}

	return cloned
}

func cloneSkillPaths(src []SkillPath) []SkillPath {
	if len(src) == 0 {
		return nil
	}

	return append([]SkillPath(nil), src...)
}

func cloneMCPServers(src []aghconfig.MCPServer) []aghconfig.MCPServer {
	if len(src) == 0 {
		return nil
	}

	cloned := make([]aghconfig.MCPServer, 0, len(src))
	for _, server := range src {
		cloned = append(cloned, aghconfig.MCPServer{
			Name:    server.Name,
			Command: server.Command,
			Args:    append([]string(nil), server.Args...),
			Env:     cloneStringMap(server.Env),
		})
	}

	return cloned
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(src))
	maps.Copy(cloned, src)
	return cloned
}

func cloneHookDecls(src []hookspkg.HookDecl) []hookspkg.HookDecl {
	if len(src) == 0 {
		return nil
	}

	cloned := make([]hookspkg.HookDecl, 0, len(src))
	for _, decl := range src {
		cloned = append(cloned, cloneHookDecl(decl))
	}

	return cloned
}

func cloneHookDecl(src hookspkg.HookDecl) hookspkg.HookDecl {
	cloned := src
	cloned.Args = append([]string(nil), src.Args...)
	cloned.Env = cloneStringMap(src.Env)
	cloned.Metadata = cloneStringMap(src.Metadata)
	if src.Matcher.ToolReadOnly != nil {
		value := *src.Matcher.ToolReadOnly
		cloned.Matcher.ToolReadOnly = &value
	}
	return cloned
}
