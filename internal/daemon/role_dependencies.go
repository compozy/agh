package daemon

func roleResolverForState(state *bootState) *roleResolver {
	if state == nil {
		return nil
	}
	if state.roleResolver == nil {
		state.roleResolver = newRoleResolver(
			&state.cfg,
			state.workspaceResolver,
			agentCatalogDependency(state.agentCatalog, agentSidecarCatalogs{
				soul: state.soulCatalog, heartbeat: state.heartbeatCatalog,
			}),
			state.registry,
		)
	}
	return state.roleResolver
}
