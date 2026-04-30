package builtin

import toolspkg "github.com/pedronauck/agh/internal/tools"

// ToolsetCatalog returns the built-in toolset definitions.
func ToolsetCatalog() (toolspkg.ToolsetCatalog, error) {
	return toolspkg.NewToolsetCatalog(builtinToolsets...)
}

var builtinToolsets = []toolspkg.Toolset{
	{
		ID: toolspkg.ToolsetIDBootstrap,
		Tools: []string{
			toolspkg.ToolIDToolList.String(),
			toolspkg.ToolIDToolSearch.String(),
			toolspkg.ToolIDToolInfo.String(),
		},
	},
	{
		ID:       toolspkg.ToolsetIDCatalog,
		Tools:    []string{"agh__skill_*"},
		Toolsets: []toolspkg.ToolsetID{toolspkg.ToolsetIDBootstrap},
	},
	{
		ID: toolspkg.ToolsetIDCoordination,
		Tools: []string{
			toolspkg.ToolIDNetworkStatus.String(),
			toolspkg.ToolIDNetworkChannels.String(),
			toolspkg.ToolIDNetworkInbox.String(),
			toolspkg.ToolIDNetworkPeers.String(),
			toolspkg.ToolIDNetworkSend.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDSessions,
		Tools: []string{
			toolspkg.ToolIDSessionList.String(),
			toolspkg.ToolIDSessionStatus.String(),
			toolspkg.ToolIDSessionHistory.String(),
			toolspkg.ToolIDSessionEvents.String(),
			toolspkg.ToolIDSessionDescribe.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDWorkspace,
		Tools: []string{
			toolspkg.ToolIDWorkspaceList.String(),
			toolspkg.ToolIDWorkspaceInfo.String(),
			toolspkg.ToolIDWorkspaceDescribe.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDMemory,
		Tools: []string{
			toolspkg.ToolIDMemoryList.String(),
			toolspkg.ToolIDMemoryRead.String(),
			toolspkg.ToolIDMemorySearch.String(),
			toolspkg.ToolIDMemoryHistory.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDObserve,
		Tools: []string{
			toolspkg.ToolIDObserveEvents.String(),
			toolspkg.ToolIDObserveMetrics.String(),
			toolspkg.ToolIDObserveSearch.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDBridges,
		Tools: []string{
			toolspkg.ToolIDBridgesList.String(),
			toolspkg.ToolIDBridgesStatus.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDTasks,
		Tools: []string{
			toolspkg.ToolIDTaskList.String(),
			toolspkg.ToolIDTaskRead.String(),
			toolspkg.ToolIDTaskCreate.String(),
			toolspkg.ToolIDTaskChildCreate.String(),
			toolspkg.ToolIDTaskUpdate.String(),
			toolspkg.ToolIDTaskCancel.String(),
			toolspkg.ToolIDTaskRunList.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDAutonomy,
		Tools: []string{
			toolspkg.ToolIDTaskRunClaimNext.String(),
			toolspkg.ToolIDTaskRunHeartbeat.String(),
			toolspkg.ToolIDTaskRunComplete.String(),
			toolspkg.ToolIDTaskRunFail.String(),
			toolspkg.ToolIDTaskRunRelease.String(),
		},
	},
	{ID: toolspkg.ToolsetIDConfig, Tools: []string{"agh__config_*"}},
	{ID: toolspkg.ToolsetIDHooks, Tools: []string{"agh__hooks_*"}},
	{ID: toolspkg.ToolsetIDAutomation, Tools: []string{"agh__automation_*"}},
	{ID: toolspkg.ToolsetIDExtensions, Tools: []string{"agh__extensions_*"}},
	{ID: toolspkg.ToolsetIDMCPAuth, Tools: []string{toolspkg.ToolIDMCPAuthStatus.String()}},
}
