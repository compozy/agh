package udsapi

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the shared AGH API routes on the supplied Gin router.
func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
	api := router.Group("/api")

	registerStatusRoutes(api, handlers)
	registerBridgeRoutes(api, handlers)
	registerNotificationRoutes(api, handlers)
	registerWorkspaceRoutes(api, handlers)
	registerSessionRoutes(api, handlers)
	registerAgentRoutes(api, handlers)
	registerAgentKernelRoutes(api, handlers)
	registerLogsRoutes(api, handlers)
	registerSupportRoutes(api, handlers)
	registerObserveRoutes(api, handlers)
	registerHookRoutes(api, handlers)
	registerResourceRoutes(api, handlers)
	registerToolRoutes(api, handlers)
	registerAutomationRoutes(api, handlers)
	registerTaskRoutes(api, handlers)
	registerTaskRunRoutes(api, handlers)
	registerSkillRoutes(api, handlers)
	registerMemoryRoutes(api, handlers)
	registerNetworkRoutes(api, handlers)
	registerBundleRoutes(api, handlers)
	registerExtensionRoutes(api, handlers)
	registerSettingsRoutes(api, handlers)
	registerVaultRoutes(api, handlers)
	registerProviderRoutes(api, handlers)
	registerModelCatalogRoutes(api, handlers)
	registerHostedMCPRoutes(api, handlers)
}

func registerStatusRoutes(api gin.IRouter, handlers *Handlers) {
	api.GET("/status", handlers.GetStatus)
	api.GET("/doctor", handlers.GetDoctor)
}

func registerBridgeRoutes(api gin.IRouter, handlers *Handlers) {
	bridges := api.Group("/bridges")
	{
		bridges.GET("", handlers.ListBridges)
		bridges.POST("", handlers.CreateBridge)
		bridges.GET("/providers", handlers.ListBridgeProviders)
		bridges.GET("/health/stream", handlers.StreamBridgeHealth)
		bridges.GET("/:id", handlers.GetBridge)
		bridges.PATCH("/:id", handlers.UpdateBridge)
		bridges.POST("/:id/enable", handlers.EnableBridge)
		bridges.POST("/:id/disable", handlers.DisableBridge)
		bridges.POST("/:id/restart", handlers.RestartBridge)
		bridges.GET("/:id/routes", handlers.ListBridgeRoutes)
		bridges.GET("/:id/targets", handlers.ListBridgeTargets)
		bridges.POST("/:id/resolve", handlers.ResolveBridgeTarget)
		bridges.GET("/:id/secret-bindings", handlers.ListBridgeSecretBindings)
		bridges.PUT("/:id/secret-bindings/:binding_name", handlers.PutBridgeSecretBinding)
		bridges.DELETE("/:id/secret-bindings/:binding_name", handlers.DeleteBridgeSecretBinding)
		bridges.POST("/:id/test-delivery", handlers.TestBridgeDelivery)
	}
}

func registerNotificationRoutes(api gin.IRouter, handlers *Handlers) {
	notifications := api.Group("/notifications")
	presets := notifications.Group("/presets")
	presets.GET("", handlers.ListNotificationPresets)
	presets.POST("", handlers.CreateNotificationPreset)
	presets.GET("/:name", handlers.GetNotificationPreset)
	presets.PUT("/:name", handlers.UpdateNotificationPreset)
	presets.DELETE("/:name", handlers.DeleteNotificationPreset)
}

func registerWorkspaceRoutes(api gin.IRouter, handlers *Handlers) {
	workspaces := api.Group("/workspaces")
	{
		workspaces.POST("", handlers.CreateWorkspace)
		workspaces.GET("", handlers.ListWorkspaces)
		workspaces.GET("/:workspace_id", handlers.GetWorkspace)
		workspaces.PATCH("/:workspace_id", handlers.UpdateWorkspace)
		workspaces.DELETE("/:workspace_id", handlers.DeleteWorkspace)
		workspaces.POST("/resolve", handlers.ResolveWorkspace)
	}
}

func registerSessionRoutes(api gin.IRouter, handlers *Handlers) {
	sessions := api.Group("/sessions")
	{
		sessions.GET("", handlers.ListSessions)
		sessions.POST("", handlers.CreateSession)
	}
	workspaceSessions := api.Group("/workspaces/:workspace_id/sessions")
	{
		workspaceSessions.GET("/:session_id", handlers.GetSession)
		workspaceSessions.POST("/:session_id/soul/refresh", handlers.RefreshSessionSoul)
		workspaceSessions.GET("/:session_id/health", handlers.GetSessionHealth)
		workspaceSessions.GET("/:session_id/status", handlers.GetSessionStatus)
		workspaceSessions.GET("/:session_id/inspect", handlers.InspectSession)
		workspaceSessions.DELETE("/:session_id", handlers.DeleteSession)
		workspaceSessions.POST("/:session_id/stop", handlers.StopSession)
		workspaceSessions.POST("/:session_id/attach", handlers.AttachSession)
		workspaceSessions.POST("/:session_id/repair", handlers.RepairSession)
		workspaceSessions.POST("/:session_id/clear", handlers.ClearSessionConversation)
		workspaceSessions.POST("/:session_id/prompt", handlers.promptSession)
		workspaceSessions.POST("/:session_id/prompt/cancel", handlers.cancelSessionPrompt)
		workspaceSessions.POST("/:session_id/interrupt", handlers.interruptSessionPrompt)
		workspaceSessions.POST("/:session_id/steer", handlers.steerSessionPrompt)
		workspaceSessions.DELETE("/:session_id/prompt/queue/:queue_entry_id", handlers.cancelQueuedSessionPrompt)
		workspaceSessions.GET("/:session_id/events", handlers.SessionEvents)
		workspaceSessions.GET("/:session_id/history", handlers.SessionHistory)
		workspaceSessions.GET("/:session_id/transcript", handlers.SessionTranscript)
		workspaceSessions.GET("/:session_id/recap", handlers.SessionRecap)
		workspaceSessions.GET("/:session_id/stream", handlers.StreamSession)
		workspaceSessions.POST("/:session_id/approve", handlers.approveSession)
	}
}

func registerAgentRoutes(api gin.IRouter, handlers *Handlers) {
	agents := api.Group("/agents")
	{
		agents.GET("", handlers.ListAgents)
		agents.POST("", handlers.CreateAgent)
		agents.GET("/:name/soul", handlers.GetAgentSoul)
		agents.POST("/:name/soul/validate", handlers.ValidateAgentSoulDefinition)
		agents.PUT("/:name/soul", handlers.PutAgentSoul)
		agents.DELETE("/:name/soul", handlers.DeleteAgentSoul)
		agents.GET("/:name/soul/history", handlers.ListAgentSoulHistory)
		agents.POST("/:name/soul/rollback", handlers.RollbackAgentSoul)
		agents.GET("/:name/heartbeat", handlers.GetAgentHeartbeat)
		agents.POST("/:name/heartbeat/validate", handlers.ValidateAgentHeartbeat)
		agents.PUT("/:name/heartbeat", handlers.PutAgentHeartbeat)
		agents.DELETE("/:name/heartbeat", handlers.DeleteAgentHeartbeat)
		agents.GET("/:name/heartbeat/history", handlers.ListAgentHeartbeatHistory)
		agents.POST("/:name/heartbeat/rollback", handlers.RollbackAgentHeartbeat)
		agents.GET("/:name/heartbeat/status", handlers.GetAgentHeartbeatStatus)
		agents.POST("/:name/heartbeat/wake", handlers.WakeAgentHeartbeat)
		agents.GET("/:name", handlers.GetAgent)
	}
}

func registerAgentKernelRoutes(api gin.IRouter, handlers *Handlers) {
	agent := api.Group("/agent")
	{
		agent.GET("/me", handlers.AgentMe)
		agent.GET("/context", handlers.AgentContext)
		agent.GET("/soul", handlers.AgentSoul)
		agent.POST("/soul/validate", handlers.ValidateAgentSoul)
		agent.GET("/coordinator/config", handlers.AgentCoordinatorConfig)
		agent.POST("/spawn", handlers.AgentSpawn)
		agent.GET("/channels", handlers.AgentChannels)
		agent.GET("/channels/:channel/recv", handlers.AgentChannelRecv)
		agent.POST("/channels/:channel/send", handlers.AgentChannelSend)
		agent.POST("/channels/reply", handlers.AgentChannelReply)
		tasks := agent.Group("/tasks")
		{
			tasks.POST("/claim-next", handlers.AgentTaskClaimNext)
			tasks.POST("/:run_id/heartbeat", handlers.AgentTaskHeartbeat)
			tasks.POST("/:run_id/complete", handlers.AgentTaskComplete)
			tasks.POST("/:run_id/fail", handlers.AgentTaskFail)
			tasks.POST("/:run_id/release", handlers.AgentTaskRelease)
		}
	}
}

func registerLogsRoutes(api gin.IRouter, handlers *Handlers) {
	api.GET("/logs", handlers.ListLogs)
	api.GET("/logs/stream", handlers.StreamLogs)
}

func registerSupportRoutes(api gin.IRouter, handlers *Handlers) {
	support := api.Group("/support")
	{
		support.POST("/bundles", handlers.CreateSupportBundle)
		support.GET("/bundles/:operation_id", handlers.GetSupportBundle)
		support.GET("/bundles/:operation_id/download", handlers.DownloadSupportBundle)
	}
}

func registerObserveRoutes(api gin.IRouter, handlers *Handlers) {
	observe := api.Group("/observe")

	taskObserve := observe.Group("/tasks")
	{
		taskObserve.GET("/dashboard", handlers.TaskDashboard)
		taskObserve.GET("/inbox", handlers.TaskInbox)
	}
}

func registerHookRoutes(api gin.IRouter, handlers *Handlers) {
	hooksGroup := api.Group("/hooks")
	{
		hooksGroup.GET("/catalog", handlers.HookCatalog)
		hooksGroup.GET("/events", handlers.HookEvents)
	}
	workspaceHooksGroup := api.Group("/workspaces/:workspace_id/hooks")
	{
		workspaceHooksGroup.GET("/runs", handlers.HookRuns)
	}
}

func registerResourceRoutes(api gin.IRouter, handlers *Handlers) {
	resourcesGroup := api.Group("/resources")
	{
		resourcesGroup.GET("", handlers.ListResources)
		resourcesGroup.GET("/:kind", handlers.ListResources)
		resourcesGroup.GET("/:kind/:id", handlers.GetResource)
		resourcesGroup.PUT("/:kind/:id", handlers.PutResource)
		resourcesGroup.DELETE("/:kind/:id", handlers.DeleteResource)
	}
}

func registerToolRoutes(api gin.IRouter, handlers *Handlers) {
	tools := api.Group("/tools")
	{
		tools.GET("", handlers.ListTools)
		tools.POST("/search", handlers.SearchTools)
		tools.POST("/:id/approvals", handlers.CreateToolApproval)
		tools.POST("/:id/invoke", handlers.InvokeTool)
		tools.GET("/:id", handlers.GetTool)
	}

	workspaceSessions := api.Group("/workspaces/:workspace_id/sessions")
	{
		workspaceSessions.GET("/:session_id/tools", handlers.ListSessionTools)
		workspaceSessions.POST("/:session_id/tools/search", handlers.SearchSessionTools)
	}

	toolsets := api.Group("/toolsets")
	{
		toolsets.GET("", handlers.ListToolsets)
		toolsets.GET("/:id", handlers.GetToolset)
	}
}

func registerAutomationRoutes(api gin.IRouter, handlers *Handlers) {
	automationGroup := api.Group("/automation")
	{
		jobs := automationGroup.Group("/jobs")
		jobs.GET("", handlers.ListAutomationJobs)
		jobs.POST("", handlers.CreateAutomationJob)
		jobs.GET("/:id", handlers.GetAutomationJob)
		jobs.PATCH("/:id", handlers.UpdateAutomationJob)
		jobs.DELETE("/:id", handlers.DeleteAutomationJob)
		jobs.POST("/:id/trigger", handlers.TriggerAutomationJob)
		jobs.GET("/:id/runs", handlers.AutomationJobRuns)

		triggers := automationGroup.Group("/triggers")
		triggers.GET("", handlers.ListAutomationTriggers)
		triggers.POST("", handlers.CreateAutomationTrigger)
		triggers.GET("/:id", handlers.GetAutomationTrigger)
		triggers.PATCH("/:id", handlers.UpdateAutomationTrigger)
		triggers.DELETE("/:id", handlers.DeleteAutomationTrigger)
		triggers.GET("/:id/runs", handlers.AutomationTriggerRuns)

		runs := automationGroup.Group("/runs")
		runs.GET("", handlers.ListAutomationRuns)
		runs.GET("/:id", handlers.GetAutomationRun)
	}
}

func registerTaskRoutes(api gin.IRouter, handlers *Handlers) {
	tasks := api.Group("/tasks")
	{
		tasks.POST("", handlers.CreateTask)
		tasks.GET("", handlers.ListTasks)
		tasks.GET("/:id", handlers.GetTask)
		tasks.GET("/:id/inspect", handlers.InspectTask)
		tasks.DELETE("/:id", handlers.DeleteTask)
		tasks.PATCH("/:id", handlers.UpdateTask)
		tasks.GET("/:id/execution-profile", handlers.GetTaskExecutionProfile)
		tasks.PUT("/:id/execution-profile", handlers.SetTaskExecutionProfile)
		tasks.DELETE("/:id/execution-profile", handlers.DeleteTaskExecutionProfile)
		tasks.POST("/:id/notifications/bridges", handlers.CreateTaskBridgeNotificationSubscription)
		tasks.GET("/:id/notifications/bridges", handlers.ListTaskBridgeNotificationSubscriptions)
		tasks.GET("/:id/notifications/bridges/:subscription_id", handlers.GetTaskBridgeNotificationSubscription)
		deleteBridgeNotificationSubscription := handlers.DeleteTaskBridgeNotificationSubscription
		tasks.DELETE("/:id/notifications/bridges/:subscription_id", deleteBridgeNotificationSubscription)
		tasks.GET("/:id/reviews", handlers.ListTaskReviews)
		tasks.POST("/:id/publish", handlers.PublishTask)
		tasks.POST("/:id/start", handlers.StartTask)
		tasks.POST("/:id/cancel", handlers.CancelTask)
		tasks.POST("/:id/pause", handlers.PauseTask)
		tasks.POST("/:id/resume", handlers.ResumeTask)
		tasks.POST("/:id/children", handlers.CreateChildTask)
		tasks.POST("/:id/dependencies", handlers.AddTaskDependency)
		tasks.DELETE("/:id/dependencies/:depends_on_id", handlers.RemoveTaskDependency)
		tasks.GET("/:id/timeline", handlers.TaskTimeline)
		tasks.GET("/:id/stream", handlers.StreamTask)
		tasks.GET("/:id/tree", handlers.TaskTree)
		tasks.POST("/:id/approve", handlers.ApproveTask)
		tasks.POST("/:id/reject", handlers.RejectTask)
		tasks.POST("/:id/triage/read", handlers.MarkTaskRead)
		tasks.POST("/:id/triage/archive", handlers.ArchiveTask)
		tasks.POST("/:id/triage/dismiss", handlers.DismissTask)
		tasks.POST("/:id/runs", handlers.EnqueueTaskRun)
		tasks.GET("/:id/runs", handlers.ListTaskRuns)
	}
}

func registerTaskRunRoutes(api gin.IRouter, handlers *Handlers) {
	taskRuns := api.Group("/task-runs")
	{
		taskRuns.GET("/:id", handlers.GetTaskRun)
		taskRuns.POST("/:id/reviews", handlers.RequestTaskRunReview)
		taskRuns.GET("/:id/reviews", handlers.ListTaskRunReviews)
		taskRuns.POST("/:id/claim", handlers.ClaimTaskRun)
		taskRuns.POST("/:id/start", handlers.StartTaskRun)
		taskRuns.POST("/:id/attach-session", handlers.AttachTaskRunSession)
		taskRuns.POST("/:id/complete", handlers.CompleteTaskRun)
		taskRuns.POST("/:id/fail", handlers.FailTaskRun)
		taskRuns.POST("/:id/cancel", handlers.CancelTaskRun)
	}

	runs := api.Group("/runs")
	{
		runs.POST("/bulk/release", handlers.BulkForceReleaseTaskRuns)
		runs.POST("/bulk/fail", handlers.BulkForceFailTaskRuns)
		runs.GET("/:id/inspect", handlers.InspectRun)
		runs.POST("/:id/release", handlers.ForceReleaseTaskRun)
		runs.POST("/:id/fail", handlers.ForceFailTaskRun)
		runs.POST("/:id/retry", handlers.RetryTaskRun)
	}

	scheduler := api.Group("/scheduler")
	{
		scheduler.GET("", handlers.GetScheduler)
		scheduler.GET("/backlog", handlers.GetSchedulerBacklog)
		scheduler.POST("/pause", handlers.PauseScheduler)
		scheduler.POST("/resume", handlers.ResumeScheduler)
		scheduler.POST("/drain", handlers.DrainScheduler)
	}

	taskReviews := api.Group("/task-reviews")
	{
		taskReviews.GET("/:id", handlers.GetTaskRunReview)
		taskReviews.POST("/:id/verdict", handlers.SubmitTaskRunReviewVerdict)
	}
}

func registerSkillRoutes(api gin.IRouter, handlers *Handlers) {
	skillsGroup := api.Group("/skills")
	{
		skillsGroup.GET("", handlers.ListSkills)
		skillsGroup.GET("/marketplace/search", handlers.SearchSkillMarketplace)
		skillsGroup.GET("/marketplace/info", handlers.GetSkillMarketplaceInfo)
		skillsGroup.POST("/marketplace/install", handlers.InstallSkillMarketplace)
		skillsGroup.POST("/marketplace/update", handlers.UpdateSkillMarketplace)
		skillsGroup.DELETE("/marketplace/:name", handlers.RemoveSkillMarketplace)
		skillsGroup.GET("/:name", handlers.GetSkill)
		skillsGroup.GET("/:name/content", handlers.GetSkillContent)
		skillsGroup.GET("/:name/shadows", handlers.GetSkillShadows)
		skillsGroup.POST("/:name/enable", handlers.EnableSkill)
		skillsGroup.POST("/:name/disable", handlers.DisableSkill)
	}
}

func registerMemoryRoutes(api gin.IRouter, handlers *Handlers) {
	memoryGroup := api.Group("/memory")
	{
		memoryGroup.GET("", handlers.ListMemory)
		memoryGroup.GET("/health", handlers.MemoryHealth)
		memoryGroup.GET("/config", handlers.MemoryConfigMetadata)
		memoryGroup.GET("/history", handlers.MemoryHistory)
		memoryGroup.GET("/scope-show", handlers.MemoryScopeShow)
		memoryGroup.POST("", handlers.WriteMemory)
		memoryGroup.POST("/search", handlers.SearchMemory)
		memoryGroup.POST("/reindex", handlers.ReindexMemory)
		memoryGroup.POST("/promote", handlers.PromoteMemory)
		memoryGroup.POST("/reset", handlers.ResetMemory)
		memoryGroup.POST("/reload", handlers.ReloadMemory)
		memoryGroup.GET("/decisions", handlers.ListMemoryDecisions)
		memoryGroup.GET("/decisions/:decision_id", handlers.GetMemoryDecision)
		memoryGroup.POST("/decisions/:decision_id/revert", handlers.RevertMemoryDecision)
		memoryGroup.GET("/recall-traces/:session_id/:turn_seq", handlers.GetMemoryRecallTrace)
		memoryGroup.GET("/dreams/status", handlers.GetMemoryDreamStatus)
		memoryGroup.GET("/dreams", handlers.ListMemoryDreams)
		memoryGroup.POST("/dreams/trigger", handlers.TriggerMemoryDream)
		memoryGroup.GET("/dreams/:dream_id", handlers.GetMemoryDream)
		memoryGroup.POST("/dreams/:dream_id/retry", handlers.RetryMemoryDream)
		memoryGroup.GET("/daily", handlers.ListMemoryDailyLogs)
		memoryGroup.GET("/extractor/status", handlers.GetMemoryExtractorStatus)
		memoryGroup.GET("/extractor/failures", handlers.ListMemoryExtractorFailures)
		memoryGroup.POST("/extractor/retry", handlers.RetryMemoryExtractor)
		memoryGroup.POST("/extractor/drain", handlers.DrainMemoryExtractor)
		memoryGroup.GET("/providers", handlers.ListMemoryProviders)
		memoryGroup.POST("/providers/select", handlers.SelectMemoryProvider)
		memoryGroup.GET("/providers/:provider_name", handlers.GetMemoryProvider)
		memoryGroup.POST("/providers/:provider_name/enable", handlers.EnableMemoryProvider)
		memoryGroup.POST("/providers/:provider_name/disable", handlers.DisableMemoryProvider)
		memoryGroup.POST("/ad-hoc", handlers.CreateMemoryAdhocNote)
		memoryGroup.POST("/sessions/prune", handlers.PruneMemorySessions)
		memoryGroup.POST("/sessions/repair", handlers.RepairMemorySessions)
		memoryGroup.GET("/:filename", handlers.ReadMemory)
		memoryGroup.PATCH("/:filename", handlers.EditMemory)
		memoryGroup.DELETE("/:filename", handlers.DeleteMemory)
	}
	workspaceMemorySessions := api.Group("/workspaces/:workspace_id/memory/sessions")
	{
		workspaceMemorySessions.GET("/:session_id/ledger", handlers.GetMemorySessionLedger)
		workspaceMemorySessions.POST("/:session_id/replay", handlers.ReplayMemorySession)
	}
}

func registerNetworkRoutes(api gin.IRouter, handlers *Handlers) {
	network := api.Group("/network")
	{
		network.GET("/status", handlers.NetworkStatus)
	}
	workspaceNetwork := api.Group("/workspaces/:workspace_id/network")
	{
		workspaceNetwork.GET("/peers", handlers.NetworkPeers)
		workspaceNetwork.GET("/peers/:peer_id", handlers.NetworkPeer)
		workspaceNetwork.GET("/channels", handlers.NetworkChannels)
		workspaceNetwork.POST("/channels", handlers.CreateNetworkChannel)
		workspaceNetwork.GET("/channels/:channel", handlers.NetworkChannel)
		workspaceNetwork.GET("/channels/:channel/threads", handlers.NetworkThreads)
		workspaceNetwork.GET("/channels/:channel/threads/:thread_id", handlers.NetworkThread)
		workspaceNetwork.GET("/channels/:channel/threads/:thread_id/messages", handlers.NetworkThreadMessages)
		workspaceNetwork.GET("/channels/:channel/directs", handlers.NetworkDirectRooms)
		workspaceNetwork.POST("/channels/:channel/directs/resolve", handlers.ResolveNetworkDirectRoom)
		workspaceNetwork.GET("/channels/:channel/directs/:direct_id", handlers.NetworkDirectRoom)
		workspaceNetwork.GET("/channels/:channel/directs/:direct_id/messages", handlers.NetworkDirectRoomMessages)
		workspaceNetwork.GET("/work/:work_id", handlers.NetworkWork)
		workspaceNetwork.POST("/send", handlers.NetworkSend)
		workspaceNetwork.GET("/inbox", handlers.NetworkInbox)
	}
}

func registerBundleRoutes(api gin.IRouter, handlers *Handlers) {
	bundles := api.Group("/bundles")
	{
		bundles.GET("/catalog", handlers.ListBundleCatalog)
		bundles.POST("/preview", handlers.PreviewBundleActivation)
		bundles.GET("/activations", handlers.ListBundleActivations)
		bundles.POST("/activations", handlers.ActivateBundle)
		bundles.GET("/activations/:id", handlers.GetBundleActivation)
		bundles.PATCH("/activations/:id", handlers.UpdateBundleActivation)
		bundles.DELETE("/activations/:id", handlers.DeleteBundleActivation)
		bundles.GET("/network/settings", handlers.BundleNetworkSettings)
	}
}

func registerExtensionRoutes(api gin.IRouter, handlers *Handlers) {
	extensions := api.Group("/extensions")
	{
		extensions.GET("", handlers.ListExtensions)
		extensions.GET("/marketplace", handlers.SearchExtensionMarketplace)
		extensions.POST("", handlers.InstallExtension)
		extensions.PUT("/:name", handlers.UpdateExtension)
		extensions.DELETE("/:name", handlers.RemoveExtension)
		extensions.GET("/:name/provenance", handlers.ExtensionProvenance)
		extensions.GET("/:name", handlers.ExtensionStatus)
		extensions.POST("/:name/enable", handlers.EnableExtension)
		extensions.POST("/:name/disable", handlers.DisableExtension)
	}
}

func registerSettingsRoutes(api gin.IRouter, handlers *Handlers) {
	settings := api.Group("/settings")

	settings.GET("/apply", handlers.ListSettingsApplyRecords)
	settings.POST("/reload", handlers.ReloadSettings)
	settings.GET("/general", handlers.GetSettingsGeneral)
	settings.GET("/update", handlers.GetSettingsUpdate)
	settings.PATCH("/general", handlers.UpdateSettingsGeneral)
	settings.GET("/memory", handlers.GetSettingsMemory)
	settings.PATCH("/memory", handlers.UpdateSettingsMemory)
	settings.GET("/skills", handlers.GetSettingsSkills)
	settings.PATCH("/skills", handlers.UpdateSettingsSkills)
	settings.GET("/automation", handlers.GetSettingsAutomation)
	settings.PATCH("/automation", handlers.UpdateSettingsAutomation)
	settings.GET("/network", handlers.GetSettingsNetwork)
	settings.PATCH("/network", handlers.UpdateSettingsNetwork)

	observability := settings.Group("/observability")
	observability.GET("", handlers.GetSettingsObservability)
	observability.PATCH("", handlers.UpdateSettingsObservability)
	observability.GET("/log-tail", handlers.StreamSettingsObservabilityLogTail)

	settings.GET("/hooks-extensions", handlers.GetSettingsHooksExtensions)
	settings.PATCH("/hooks-extensions", handlers.UpdateSettingsHooksExtensions)

	settings.GET("/providers", handlers.ListSettingsProviders)
	settings.GET("/providers/:name", handlers.GetSettingsProvider)
	settings.PUT("/providers/:name", handlers.PutSettingsProvider)
	settings.DELETE("/providers/:name", handlers.DeleteSettingsProvider)

	settings.GET("/mcp-servers", handlers.ListSettingsMCPServers)
	settings.PUT("/mcp-servers/:name", handlers.PutSettingsMCPServer)
	settings.DELETE("/mcp-servers/:name", handlers.DeleteSettingsMCPServer)

	settings.GET("/sandboxes", handlers.ListSettingsSandboxes)
	settings.GET("/sandboxes/:name", handlers.GetSettingsSandbox)
	settings.PUT("/sandboxes/:name", handlers.PutSettingsSandbox)
	settings.DELETE("/sandboxes/:name", handlers.DeleteSettingsSandbox)

	settings.GET("/hooks", handlers.ListSettingsHooks)
	settings.PUT("/hooks/:name", handlers.PutSettingsHook)
	settings.DELETE("/hooks/:name", handlers.DeleteSettingsHook)

	actions := settings.Group("/actions")
	actions.POST("/restart", handlers.TriggerSettingsRestart)
	actions.GET("/restart/:operation_id", handlers.GetSettingsRestartStatus)
}

func registerVaultRoutes(api gin.IRouter, handlers *Handlers) {
	vaultGroup := api.Group("/vault")
	{
		vaultGroup.GET("/secrets", handlers.ListVaultSecrets)
		vaultGroup.GET("/secrets/metadata", handlers.GetVaultSecretMetadata)
		vaultGroup.PUT("/secrets", handlers.PutVaultSecret)
		vaultGroup.DELETE("/secrets", handlers.DeleteVaultSecret)
	}
}

func registerProviderRoutes(api gin.IRouter, handlers *Handlers) {
	providers := api.Group("/providers")
	{
		providers.GET("", handlers.ListProviders)
		providers.GET("/:provider_id", handlers.GetProvider)
		providers.POST("/:provider_id/auth/probe", handlers.ProbeProviderAuth)
	}
}

func registerModelCatalogRoutes(api gin.IRouter, handlers *Handlers) {
	modelCatalog := api.Group("/model-catalog")
	modelCatalog.GET("/*catalog_path", handlers.ModelCatalogRoute)
	modelCatalog.POST("/*catalog_path", handlers.ModelCatalogRoute)
}
