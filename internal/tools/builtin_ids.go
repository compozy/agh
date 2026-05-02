package tools

const (
	// BuiltinSourceOwner is the source owner for daemon-compiled AGH tools.
	BuiltinSourceOwner = "daemon"
)

const (
	// ToolIDToolList lists tools in the caller's effective registry projection.
	ToolIDToolList ToolID = "agh__tool_list"
	// ToolIDToolSearch searches tools in the caller's effective registry projection.
	ToolIDToolSearch ToolID = "agh__tool_search"
	// ToolIDToolInfo reads one tool descriptor and diagnostics view.
	ToolIDToolInfo ToolID = "agh__tool_info"
	// ToolIDSkillList lists skills through the existing skill registry.
	ToolIDSkillList ToolID = "agh__skill_list"
	// ToolIDSkillSearch searches skills through the existing skill registry.
	ToolIDSkillSearch ToolID = "agh__skill_search"
	// ToolIDSkillView reads one skill and its verified body.
	ToolIDSkillView ToolID = "agh__skill_view"
	// ToolIDNetworkPeers lists visible network peers.
	ToolIDNetworkPeers ToolID = "agh__network_peers"
	// ToolIDNetworkStatus reads daemon-owned network runtime status.
	ToolIDNetworkStatus ToolID = "agh__network_status"
	// ToolIDNetworkChannels lists active AGH network channels.
	ToolIDNetworkChannels ToolID = "agh__network_channels"
	// ToolIDNetworkInbox reads queued inbound network messages for one local session.
	ToolIDNetworkInbox ToolID = "agh__network_inbox"
	// ToolIDNetworkSend sends one network message through the existing network manager.
	ToolIDNetworkSend ToolID = "agh__network_send"
	// ToolIDSessionList lists runtime sessions.
	ToolIDSessionList ToolID = "agh__session_list"
	// ToolIDSessionStatus reads one runtime session snapshot.
	ToolIDSessionStatus ToolID = "agh__session_status"
	// ToolIDSessionHistory reads grouped turn history for one session.
	ToolIDSessionHistory ToolID = "agh__session_history"
	// ToolIDSessionEvents reads persisted events for one session.
	ToolIDSessionEvents ToolID = "agh__session_events"
	// ToolIDSessionDescribe reads a composite read-only session description.
	ToolIDSessionDescribe ToolID = "agh__session_describe"
	// ToolIDSessionHealth reads metadata-only session health and wake eligibility.
	ToolIDSessionHealth ToolID = "agh__session_health"
	// ToolIDAgentHeartbeatStatus reads resolved Heartbeat policy, wake state, health, and wake audit.
	ToolIDAgentHeartbeatStatus ToolID = "agh__agent_heartbeat_status"
	// ToolIDAgentHeartbeatWake requests one managed advisory Heartbeat wake decision.
	ToolIDAgentHeartbeatWake ToolID = "agh__agent_heartbeat_wake"
	// ToolIDWorkspaceList lists registered workspaces.
	ToolIDWorkspaceList ToolID = "agh__workspace_list"
	// ToolIDWorkspaceInfo reads one registered workspace record.
	ToolIDWorkspaceInfo ToolID = "agh__workspace_info"
	// ToolIDWorkspaceDescribe reads one resolved workspace detail projection.
	ToolIDWorkspaceDescribe ToolID = "agh__workspace_describe"
	// ToolIDMemoryList lists memory headers visible for a scope.
	ToolIDMemoryList ToolID = "agh__memory_list"
	// ToolIDMemoryRead reads one memory document through the current memory store.
	ToolIDMemoryRead ToolID = "agh__memory_read"
	// ToolIDMemorySearch searches memory documents through the current memory store.
	ToolIDMemorySearch ToolID = "agh__memory_search"
	// ToolIDMemoryHistory reads redacted memory operation history.
	ToolIDMemoryHistory ToolID = "agh__memory_history"
	// ToolIDObserveEvents reads redacted observability events.
	ToolIDObserveEvents ToolID = "agh__observe_events"
	// ToolIDObserveMetrics reads daemon observability health and metrics.
	ToolIDObserveMetrics ToolID = "agh__observe_metrics"
	// ToolIDObserveSearch searches redacted observability events.
	ToolIDObserveSearch ToolID = "agh__observe_search"
	// ToolIDBridgesList lists bridge instances without secret bindings.
	ToolIDBridgesList ToolID = "agh__bridges_list"
	// ToolIDBridgesStatus reads bridge status and health without credentials.
	ToolIDBridgesStatus ToolID = "agh__bridges_status"
	// ToolIDTaskList lists task summaries through the task service.
	ToolIDTaskList ToolID = "agh__task_list"
	// ToolIDTaskRead reads one task view through the task service.
	ToolIDTaskRead ToolID = "agh__task_read"
	// ToolIDTaskCreate creates one root task through the task service.
	ToolIDTaskCreate ToolID = "agh__task_create"
	// ToolIDTaskChildCreate creates one child task through the task service.
	ToolIDTaskChildCreate ToolID = "agh__task_child_create"
	// ToolIDTaskUpdate updates one task through the task service.
	ToolIDTaskUpdate ToolID = "agh__task_update"
	// ToolIDTaskCancel cancels one task through the task service.
	ToolIDTaskCancel ToolID = "agh__task_cancel"
	// ToolIDTaskRunList lists task runs through the task service.
	ToolIDTaskRunList ToolID = "agh__task_run_list"
	// ToolIDTaskRunClaimNext claims the next run for the caller session.
	ToolIDTaskRunClaimNext ToolID = "agh__task_run_claim_next"
	// ToolIDTaskRunHeartbeat extends the caller session's active run lease.
	ToolIDTaskRunHeartbeat ToolID = "agh__task_run_heartbeat"
	// ToolIDTaskRunComplete completes the caller session's active run lease.
	ToolIDTaskRunComplete ToolID = "agh__task_run_complete"
	// ToolIDTaskRunFail fails the caller session's active run lease.
	ToolIDTaskRunFail ToolID = "agh__task_run_fail"
	// ToolIDTaskRunRelease releases the caller session's active run lease.
	ToolIDTaskRunRelease ToolID = "agh__task_run_release"
	// ToolIDConfigShow shows the redacted effective config.
	ToolIDConfigShow ToolID = "agh__config_show"
	// ToolIDConfigList lists redacted effective config entries.
	ToolIDConfigList ToolID = "agh__config_list"
	// ToolIDConfigGet reads one redacted effective config entry.
	ToolIDConfigGet ToolID = "agh__config_get"
	// ToolIDConfigSet mutates one validated config overlay value.
	ToolIDConfigSet ToolID = "agh__config_set"
	// ToolIDConfigUnset removes one validated config overlay value.
	ToolIDConfigUnset ToolID = "agh__config_unset"
	// ToolIDConfigDiff compares defaults/global config against the effective view.
	ToolIDConfigDiff ToolID = "agh__config_diff"
	// ToolIDConfigPath reports resolved config paths.
	ToolIDConfigPath ToolID = "agh__config_path"
	// ToolIDHooksList lists resolved hooks.
	ToolIDHooksList ToolID = "agh__hooks_list"
	// ToolIDHooksInfo reads one resolved hook.
	ToolIDHooksInfo ToolID = "agh__hooks_info"
	// ToolIDHooksEvents lists supported hook events.
	ToolIDHooksEvents ToolID = "agh__hooks_events"
	// ToolIDHooksRuns lists hook run audit records.
	ToolIDHooksRuns ToolID = "agh__hooks_runs"
	// ToolIDHooksCreate creates one config-backed hook declaration.
	ToolIDHooksCreate ToolID = "agh__hooks_create"
	// ToolIDHooksUpdate updates one config-backed hook declaration.
	ToolIDHooksUpdate ToolID = "agh__hooks_update"
	// ToolIDHooksDelete deletes one config-backed hook declaration.
	ToolIDHooksDelete ToolID = "agh__hooks_delete"
	// ToolIDHooksEnable enables one config-backed hook declaration.
	ToolIDHooksEnable ToolID = "agh__hooks_enable"
	// ToolIDHooksDisable disables one config-backed hook declaration.
	ToolIDHooksDisable ToolID = "agh__hooks_disable"
	// ToolIDAutomationJobsList lists automation jobs through the automation manager.
	ToolIDAutomationJobsList ToolID = "agh__automation_jobs_list"
	// ToolIDAutomationJobsGet reads one automation job through the automation manager.
	ToolIDAutomationJobsGet ToolID = "agh__automation_jobs_get"
	// ToolIDAutomationJobsCreate creates one dynamic automation job through the automation manager.
	ToolIDAutomationJobsCreate ToolID = "agh__automation_jobs_create"
	// ToolIDAutomationJobsUpdate updates one automation job through the automation manager.
	ToolIDAutomationJobsUpdate ToolID = "agh__automation_jobs_update"
	// ToolIDAutomationJobsDelete deletes one dynamic automation job through the automation manager.
	ToolIDAutomationJobsDelete ToolID = "agh__automation_jobs_delete"
	// ToolIDAutomationJobsEnable enables one automation job through the automation manager.
	ToolIDAutomationJobsEnable ToolID = "agh__automation_jobs_enable"
	// ToolIDAutomationJobsDisable disables one automation job through the automation manager.
	ToolIDAutomationJobsDisable ToolID = "agh__automation_jobs_disable"
	// ToolIDAutomationJobsTrigger manually triggers one automation job through the automation manager.
	ToolIDAutomationJobsTrigger ToolID = "agh__automation_jobs_trigger"
	// ToolIDAutomationJobsHistory lists run history for one automation job.
	ToolIDAutomationJobsHistory ToolID = "agh__automation_jobs_history"
	// ToolIDAutomationTriggersList lists automation triggers through the automation manager.
	ToolIDAutomationTriggersList ToolID = "agh__automation_triggers_list"
	// ToolIDAutomationTriggersGet reads one automation trigger through the automation manager.
	ToolIDAutomationTriggersGet ToolID = "agh__automation_triggers_get"
	// ToolIDAutomationTriggersCreate creates one dynamic automation trigger through the automation manager.
	ToolIDAutomationTriggersCreate ToolID = "agh__automation_triggers_create"
	// ToolIDAutomationTriggersUpdate updates one automation trigger through the automation manager.
	ToolIDAutomationTriggersUpdate ToolID = "agh__automation_triggers_update"
	// ToolIDAutomationTriggersDelete deletes one dynamic automation trigger through the automation manager.
	ToolIDAutomationTriggersDelete ToolID = "agh__automation_triggers_delete"
	// ToolIDAutomationTriggersEnable enables one automation trigger through the automation manager.
	ToolIDAutomationTriggersEnable ToolID = "agh__automation_triggers_enable"
	// ToolIDAutomationTriggersDisable disables one automation trigger through the automation manager.
	ToolIDAutomationTriggersDisable ToolID = "agh__automation_triggers_disable"
	// ToolIDAutomationTriggersHistory lists run history for one automation trigger.
	ToolIDAutomationTriggersHistory ToolID = "agh__automation_triggers_history"
	// ToolIDAutomationRunsList lists automation run records through the automation manager.
	ToolIDAutomationRunsList ToolID = "agh__automation_runs_list"
	// ToolIDAutomationRunsGet reads one automation run record through the automation manager.
	ToolIDAutomationRunsGet ToolID = "agh__automation_runs_get"
	// ToolIDExtensionsSearch searches configured extension marketplace sources.
	ToolIDExtensionsSearch ToolID = "agh__extensions_search"
	// ToolIDExtensionsList lists installed extensions through the extension registry.
	ToolIDExtensionsList ToolID = "agh__extensions_list"
	// ToolIDExtensionsInfo reads one installed extension status.
	ToolIDExtensionsInfo ToolID = "agh__extensions_info"
	// ToolIDExtensionsInstall installs one extension through a managed local or marketplace source.
	ToolIDExtensionsInstall ToolID = "agh__extensions_install"
	// ToolIDExtensionsUpdate updates one or more marketplace-installed extensions.
	ToolIDExtensionsUpdate ToolID = "agh__extensions_update"
	// ToolIDExtensionsRemove removes one managed installed extension.
	ToolIDExtensionsRemove ToolID = "agh__extensions_remove"
	// ToolIDExtensionsEnable enables one installed extension.
	ToolIDExtensionsEnable ToolID = "agh__extensions_enable"
	// ToolIDExtensionsDisable disables one installed extension.
	ToolIDExtensionsDisable ToolID = "agh__extensions_disable"
	// ToolIDMCPAuthStatus reads redacted MCP auth diagnostics for one configured server.
	ToolIDMCPAuthStatus ToolID = "agh__mcp_auth_status"
)

const (
	// ToolsetIDBootstrap groups registry self-inspection tools.
	ToolsetIDBootstrap ToolsetID = "agh__bootstrap"
	// ToolsetIDCatalog groups registry and skill catalog tools.
	ToolsetIDCatalog ToolsetID = "agh__catalog"
	// ToolsetIDCoordination groups network coordination tools.
	ToolsetIDCoordination ToolsetID = "agh__coordination"
	// ToolsetIDTasks groups bounded task tools.
	ToolsetIDTasks ToolsetID = "agh__tasks"
	// ToolsetIDAutonomy groups session-bound task-run autonomy tools.
	ToolsetIDAutonomy ToolsetID = "agh__autonomy"
	// ToolsetIDSessions groups read-only runtime session tools.
	ToolsetIDSessions ToolsetID = "agh__sessions"
	// ToolsetIDAuthoredContext groups managed Soul/Heartbeat read and wake tools.
	ToolsetIDAuthoredContext ToolsetID = "agh__authored_context"
	// ToolsetIDWorkspace groups read-only workspace tools.
	ToolsetIDWorkspace ToolsetID = "agh__workspace"
	// ToolsetIDMemory groups read-only memory inspection tools.
	ToolsetIDMemory ToolsetID = "agh__memory"
	// ToolsetIDObserve groups read-only observability tools.
	ToolsetIDObserve ToolsetID = "agh__observe"
	// ToolsetIDBridges groups read-only bridge inspection tools.
	ToolsetIDBridges ToolsetID = "agh__bridges"
	// ToolsetIDConfig groups validated config tools.
	ToolsetIDConfig ToolsetID = "agh__config"
	// ToolsetIDHooks groups hook introspection and mutable config-backed hook tools.
	ToolsetIDHooks ToolsetID = "agh__hooks"
	// ToolsetIDAutomation groups automation lifecycle and run inspection tools.
	ToolsetIDAutomation ToolsetID = "agh__automation"
	// ToolsetIDExtensions groups extension discovery and lifecycle tools.
	ToolsetIDExtensions ToolsetID = "agh__extensions"
	// ToolsetIDMCPAuth groups redacted MCP auth diagnostics.
	ToolsetIDMCPAuth ToolsetID = "agh__mcp_auth"
)

// BuiltinSource returns the provenance shared by daemon-compiled AGH tools.
func BuiltinSource() SourceRef {
	return SourceRef{
		Kind:  SourceBuiltin,
		Owner: BuiltinSourceOwner,
		Scope: "daemon",
	}
}
