package builtin

import toolspkg "github.com/compozy/agh/internal/tools"

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
			toolspkg.ToolIDNetworkChannelCreate.String(),
			toolspkg.ToolIDNetworkThreads.String(),
			toolspkg.ToolIDNetworkThreadMessages.String(),
			toolspkg.ToolIDNetworkDirects.String(),
			toolspkg.ToolIDNetworkDirectResolve.String(),
			toolspkg.ToolIDNetworkDirectMessages.String(),
			toolspkg.ToolIDNetworkWork.String(),
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
			toolspkg.ToolIDSessionHealth.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDAuthoredContext,
		Tools: []string{
			toolspkg.ToolIDSessionHealth.String(),
			toolspkg.ToolIDAgentHeartbeatStatus.String(),
			toolspkg.ToolIDAgentHeartbeatWake.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDWorkspace,
		Tools: []string{
			toolspkg.ToolIDWorkspaceList.String(),
			toolspkg.ToolIDWorkspaceInfo.String(),
			toolspkg.ToolIDWorkspaceDescribe.String(),
			toolspkg.ToolIDAgentCreate.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDProviderModels,
		Tools: []string{
			toolspkg.ToolIDProviderModelsList.String(),
			toolspkg.ToolIDProviderModelsRefresh.String(),
			toolspkg.ToolIDProviderModelsStatus.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDMemory,
		Tools: []string{
			toolspkg.ToolIDMemoryList.String(),
			toolspkg.ToolIDMemoryShow.String(),
			toolspkg.ToolIDMemorySearch.String(),
			toolspkg.ToolIDMemoryPropose.String(),
			toolspkg.ToolIDMemoryNote.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDMemoryAdmin,
		Tools: []string{
			toolspkg.ToolIDMemoryHealth.String(),
			toolspkg.ToolIDMemoryScopeShow.String(),
			toolspkg.ToolIDMemoryAdminHistory.String(),
			toolspkg.ToolIDMemoryReindex.String(),
			toolspkg.ToolIDMemoryPromote.String(),
			toolspkg.ToolIDMemoryReset.String(),
			toolspkg.ToolIDMemoryReload.String(),
			toolspkg.ToolIDMemoryDecisionsList.String(),
			toolspkg.ToolIDMemoryDecisionsShow.String(),
			toolspkg.ToolIDMemoryDecisionsRevert.String(),
			toolspkg.ToolIDMemoryRecallTrace.String(),
			toolspkg.ToolIDMemoryDreamStatus.String(),
			toolspkg.ToolIDMemoryDreamList.String(),
			toolspkg.ToolIDMemoryDreamShow.String(),
			toolspkg.ToolIDMemoryDreamTrigger.String(),
			toolspkg.ToolIDMemoryDreamRetry.String(),
			toolspkg.ToolIDMemoryDailyList.String(),
			toolspkg.ToolIDMemoryExtractorStatus.String(),
			toolspkg.ToolIDMemoryExtractorFailures.String(),
			toolspkg.ToolIDMemoryExtractorRetry.String(),
			toolspkg.ToolIDMemoryExtractorDrain.String(),
			toolspkg.ToolIDMemoryProviderList.String(),
			toolspkg.ToolIDMemoryProviderGet.String(),
			toolspkg.ToolIDMemoryProviderSelect.String(),
			toolspkg.ToolIDMemoryProviderEnable.String(),
			toolspkg.ToolIDMemoryProviderDisable.String(),
			toolspkg.ToolIDMemorySessionLedger.String(),
			toolspkg.ToolIDMemorySessionReplay.String(),
			toolspkg.ToolIDMemorySessionsPrune.String(),
			toolspkg.ToolIDMemorySessionsRepair.String(),
		},
	},
	{
		ID: toolspkg.ToolsetIDObserve,
		Tools: []string{
			toolspkg.ToolIDListLogs.String(),
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
			toolspkg.ToolIDTaskRunReviewRequest.String(),
			toolspkg.ToolIDTaskRunReviewList.String(),
			toolspkg.ToolIDTaskRunReviewShow.String(),
			toolspkg.ToolIDTaskExecutionProfileGet.String(),
			toolspkg.ToolIDTaskExecutionProfileSet.String(),
			toolspkg.ToolIDTaskExecutionProfileDelete.String(),
			toolspkg.ToolIDTaskNotificationSubscribe.String(),
			toolspkg.ToolIDTaskNotificationList.String(),
			toolspkg.ToolIDTaskNotificationShow.String(),
			toolspkg.ToolIDTaskNotificationDelete.String(),
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
			toolspkg.ToolIDTaskRunBlock.String(),
			toolspkg.ToolIDTaskRunReviewSubmit.String(),
		},
	},
	{ID: toolspkg.ToolsetIDConfig, Tools: []string{"agh__config_*"}},
	{ID: toolspkg.ToolsetIDHooks, Tools: []string{"agh__hooks_*"}},
	{ID: toolspkg.ToolsetIDAutomation, Tools: []string{"agh__automation_*"}},
	{ID: toolspkg.ToolsetIDExtensions, Tools: []string{"agh__extensions_*"}},
	{ID: toolspkg.ToolsetIDBundles, Tools: []string{"agh__bundles_*"}},
	{ID: toolspkg.ToolsetIDResources, Tools: []string{"agh__resources_*"}},
	{ID: toolspkg.ToolsetIDMCP, Tools: []string{toolspkg.ToolIDMCPStatus.String()}},
	{ID: toolspkg.ToolsetIDMCPAuth, Tools: []string{toolspkg.ToolIDMCPAuthStatus.String()}},
}
