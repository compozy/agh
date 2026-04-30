package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

// ToolsetCatalog returns the built-in toolset definitions.
func ToolsetCatalog() (toolspkg.ToolsetCatalog, error) {
	return toolspkg.NewToolsetCatalog(
		toolspkg.Toolset{
			ID: toolspkg.ToolsetIDBootstrap,
			Tools: []string{
				toolspkg.ToolIDToolList.String(),
				toolspkg.ToolIDToolSearch.String(),
				toolspkg.ToolIDToolInfo.String(),
			},
		},
		toolspkg.Toolset{
			ID:       toolspkg.ToolsetIDCatalog,
			Tools:    []string{"agh__skill_*"},
			Toolsets: []toolspkg.ToolsetID{toolspkg.ToolsetIDBootstrap},
		},
		toolspkg.Toolset{
			ID: toolspkg.ToolsetIDCoordination,
			Tools: []string{
				toolspkg.ToolIDNetworkPeers.String(),
				toolspkg.ToolIDNetworkSend.String(),
			},
		},
		toolspkg.Toolset{
			ID:    toolspkg.ToolsetIDTasks,
			Tools: []string{"agh__task_*"},
		},
		toolspkg.Toolset{
			ID:    toolspkg.ToolsetIDConfig,
			Tools: []string{"agh__config_*"},
		},
		toolspkg.Toolset{
			ID:    toolspkg.ToolsetIDHooks,
			Tools: []string{"agh__hooks_*"},
		},
	)
}
