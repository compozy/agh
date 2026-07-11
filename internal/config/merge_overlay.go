package config

type configOverlay struct {
	Daemon        daemonOverlay              `toml:"daemon"`
	HTTP          httpOverlay                `toml:"http"`
	Defaults      defaultsOverlay            `toml:"defaults"`
	Agents        agentsOverlay              `toml:"agents"`
	Limits        limitsOverlay              `toml:"limits"`
	Session       sessionOverlay             `toml:"session"`
	Permissions   permissionsOverlay         `toml:"permissions"`
	MCPServers    []mcpServerOverlay         `toml:"mcp_servers"`
	Providers     map[string]providerOverlay `toml:"providers"`
	ModelCatalog  modelCatalogOverlay        `toml:"model_catalog"`
	Sandboxes     map[string]sandboxOverlay  `toml:"sandboxes"`
	Observability observabilityOverlay       `toml:"observability"`
	Log           logOverlay                 `toml:"log"`
	Memory        memoryOverlay              `toml:"memory"`
	Skills        skillsOverlay              `toml:"skills"`
	Extensions    extensionsOverlay          `toml:"extensions"`
	Tools         toolsOverlay               `toml:"tools"`
	Automation    automationOverlay          `toml:"automation"`
	Loops         loopsOverlay               `toml:"loops"`
	Goals         goalsOverlay               `toml:"goals"`
	Task          taskOverlay                `toml:"task"`
	Hooks         hooksOverlay               `toml:"hooks"`
	Network       networkOverlay             `toml:"network"`
	Autonomy      autonomyOverlay            `toml:"autonomy"`
}

func (o *configOverlay) Apply(dst *Config) error {
	o.Daemon.Apply(&dst.Daemon)
	o.HTTP.Apply(&dst.HTTP)
	o.Defaults.Apply(&dst.Defaults)
	o.Agents.Apply(&dst.Agents)
	o.Limits.Apply(&dst.Limits)
	o.Session.Apply(&dst.Session)
	o.Permissions.Apply(&dst.Permissions)
	if len(o.MCPServers) > 0 {
		dst.MCPServers = applyMCPServerOverlays(dst.MCPServers, o.MCPServers)
	}
	applyProviderOverlays(dst, o.Providers)
	o.ModelCatalog.Apply(&dst.ModelCatalog)
	applySandboxOverlays(dst, o.Sandboxes)
	o.Observability.Apply(&dst.Observability)
	o.Log.Apply(&dst.Log)
	o.Memory.Apply(&dst.Memory)
	o.Skills.Apply(&dst.Skills)
	o.Extensions.Apply(&dst.Extensions)
	o.Tools.Apply(&dst.Tools)
	if err := o.Automation.Apply(&dst.Automation); err != nil {
		return err
	}
	o.Loops.Apply(&dst.Loops)
	o.Goals.Apply(&dst.Goals)
	o.Task.Apply(&dst.Task)
	o.Network.Apply(&dst.Network)
	o.Autonomy.Apply(&dst.Autonomy)
	return o.Hooks.Apply(&dst.Hooks)
}
