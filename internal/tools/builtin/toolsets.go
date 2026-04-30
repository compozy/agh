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
				toolspkg.ToolIDNetworkStatus.String(),
				toolspkg.ToolIDNetworkChannels.String(),
				toolspkg.ToolIDNetworkInbox.String(),
				toolspkg.ToolIDNetworkPeers.String(),
				toolspkg.ToolIDNetworkSend.String(),
			},
		},
		toolspkg.Toolset{
			ID: toolspkg.ToolsetIDSessions,
			Tools: []string{
				toolspkg.ToolIDSessionList.String(),
				toolspkg.ToolIDSessionStatus.String(),
				toolspkg.ToolIDSessionHistory.String(),
				toolspkg.ToolIDSessionEvents.String(),
				toolspkg.ToolIDSessionDescribe.String(),
			},
		},
		toolspkg.Toolset{
			ID: toolspkg.ToolsetIDWorkspace,
			Tools: []string{
				toolspkg.ToolIDWorkspaceList.String(),
				toolspkg.ToolIDWorkspaceInfo.String(),
				toolspkg.ToolIDWorkspaceDescribe.String(),
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
