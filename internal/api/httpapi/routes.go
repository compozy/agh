package httpapi

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the shared AGH API routes on the supplied Gin router.
func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
	if handlers == nil {
		return
	}

	api := router.Group("/api")
	registerWebhookRoutes(api, handlers)

	if !isLoopbackHost(canonicalHost(handlers.boundHost)) {
		api = api.Group("", loopbackAPIGuard(handlers.boundHost))
	}

	registerBridgeRoutes(api, handlers)
	registerWorkspaceRoutes(api, handlers)
	registerSessionRoutes(api, handlers)
	registerAgentRoutes(api, handlers)
	registerObserveRoutes(api, handlers)
	registerHookRoutes(api, handlers)
	registerResourceRoutes(api, handlers)
	registerToolRoutes(api, handlers)
	registerAutomationRoutes(api, handlers)
	registerTaskRoutes(api, handlers)
	registerSkillRoutes(api, handlers)
	registerMemoryRoutes(api, handlers)
	registerDaemonRoutes(api, handlers)
	registerNetworkRoutes(api, handlers)
	registerBundleRoutes(api, handlers)
	registerExtensionRoutes(api, handlers)
	registerSettingsRoutes(api, handlers)
	registerVaultRoutes(api, handlers)

	if engine, ok := router.(*gin.Engine); ok {
		engine.NoRoute(handlers.serveStaticRoute)
	}
}

func registerBridgeRoutes(api gin.IRouter, handlers *Handlers) {
	bridges := api.Group("/bridges")
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
	bridges.GET("/:id/secret-bindings", handlers.ListBridgeSecretBindings)
	bridges.PUT("/:id/secret-bindings/:binding_name", handlers.PutBridgeSecretBinding)
	bridges.DELETE("/:id/secret-bindings/:binding_name", handlers.DeleteBridgeSecretBinding)
	bridges.POST("/:id/test-delivery", handlers.TestBridgeDelivery)
}

func registerWorkspaceRoutes(api gin.IRouter, handlers *Handlers) {
	workspaces := api.Group("/workspaces")
	workspaces.POST("", handlers.CreateWorkspace)
	workspaces.GET("", handlers.ListWorkspaces)
	workspaces.GET("/:id", handlers.GetWorkspace)
	workspaces.PATCH("/:id", handlers.UpdateWorkspace)
	workspaces.DELETE("/:id", handlers.DeleteWorkspace)
	workspaces.POST("/resolve", handlers.ResolveWorkspace)
}

func registerSessionRoutes(api gin.IRouter, handlers *Handlers) {
	sessions := api.Group("/sessions")
	sessions.GET("", handlers.ListSessions)
	sessions.POST("", handlers.CreateSession)
	sessions.GET("/:id", handlers.GetSession)
	sessions.POST("/:id/soul/refresh", handlers.RefreshSessionSoul)
	sessions.GET("/:id/health", handlers.GetSessionHealth)
	sessions.GET("/:id/status", handlers.GetSessionStatus)
	sessions.GET("/:id/inspect", handlers.InspectSession)
	sessions.DELETE("/:id", handlers.DeleteSession)
	sessions.POST("/:id/stop", handlers.StopSession)
	sessions.POST("/:id/resume", handlers.ResumeSession)
	sessions.POST("/:id/repair", handlers.RepairSession)
	sessions.POST("/:id/clear", handlers.ClearSessionConversation)
	sessions.POST("/:id/prompt", handlers.promptSession)
	sessions.POST("/:id/prompt/cancel", handlers.cancelSessionPrompt)
	sessions.GET("/:id/events", handlers.SessionEvents)
	sessions.GET("/:id/history", handlers.SessionHistory)
	sessions.GET("/:id/transcript", handlers.SessionTranscript)
	sessions.GET("/:id/stream", handlers.StreamSession)
	sessions.POST("/:id/approve", handlers.approveSession)
}

func registerAgentRoutes(api gin.IRouter, handlers *Handlers) {
	agent := api.Group("/agent")
	agent.GET("/context", handlers.AgentContext)
	agent.GET("/soul", handlers.AgentSoul)
	agent.POST("/soul/validate", handlers.ValidateAgentSoul)

	agents := api.Group("/agents")
	agents.GET("", handlers.ListAgents)
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

func registerObserveRoutes(api gin.IRouter, handlers *Handlers) {
	observeGroup := api.Group("/observe")
	observeGroup.GET("/events", handlers.ObserveEvents)
	observeGroup.GET("/events/stream", handlers.StreamObserveEvents)
	observeGroup.GET("/health", handlers.Health)

	taskObserveGroup := observeGroup.Group("/tasks")
	taskObserveGroup.GET("/dashboard", handlers.TaskDashboard)
	taskObserveGroup.GET("/inbox", handlers.TaskInbox)
}

func registerHookRoutes(api gin.IRouter, handlers *Handlers) {
	hooksGroup := api.Group("/hooks")
	hooksGroup.GET("/catalog", handlers.HookCatalog)
	hooksGroup.GET("/runs", handlers.HookRuns)
	hooksGroup.GET("/events", handlers.HookEvents)
}

func registerResourceRoutes(api gin.IRouter, handlers *Handlers) {
	if handlers == nil {
		return
	}

	auth := handlers.resourceAuthMiddleware()
	if len(auth) == 0 {
		return
	}

	resourcesGroup := api.Group("/resources", auth...)
	resourcesGroup.GET("", handlers.ListResources)
	resourcesGroup.GET("/:kind", handlers.ListResources)
	resourcesGroup.GET("/:kind/:id", handlers.GetResource)
	resourcesGroup.PUT("/:kind/:id", handlers.PutResource)
	resourcesGroup.DELETE("/:kind/:id", handlers.DeleteResource)
}

func registerToolRoutes(api gin.IRouter, handlers *Handlers) {
	privileged := handlers.privilegedMutationGuard()
	tools := api.Group("/tools")
	tools.GET("", handlers.ListTools)
	tools.POST("/search", handlers.SearchTools)
	tools.POST("/:id/approvals", privileged, handlers.CreateToolApproval)
	tools.POST("/:id/invoke", privileged, handlers.InvokeTool)
	tools.GET("/:id", handlers.GetTool)

	sessions := api.Group("/sessions")
	sessions.GET("/:id/tools", handlers.ListSessionTools)
	sessions.POST("/:id/tools/search", handlers.SearchSessionTools)

	toolsets := api.Group("/toolsets")
	toolsets.GET("", handlers.ListToolsets)
	toolsets.GET("/:id", handlers.GetToolset)
}

func registerAutomationRoutes(api gin.IRouter, handlers *Handlers) {
	automationGroup := api.Group("/automation")

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

func registerTaskRoutes(api gin.IRouter, handlers *Handlers) {
	tasks := api.Group("/tasks")
	tasks.POST("", handlers.CreateTask)
	tasks.GET("", handlers.ListTasks)
	tasks.GET("/:id", handlers.GetTask)
	tasks.DELETE("/:id", handlers.DeleteTask)
	tasks.PATCH("/:id", handlers.UpdateTask)
	tasks.POST("/:id/publish", handlers.PublishTask)
	tasks.POST("/:id/start", handlers.StartTask)
	tasks.POST("/:id/cancel", handlers.CancelTask)
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

	taskRuns := api.Group("/task-runs")
	taskRuns.GET("/:id", handlers.GetTaskRun)
	taskRuns.POST("/:id/claim", handlers.ClaimTaskRun)
	taskRuns.POST("/:id/start", handlers.StartTaskRun)
	taskRuns.POST("/:id/attach-session", handlers.AttachTaskRunSession)
	taskRuns.POST("/:id/complete", handlers.CompleteTaskRun)
	taskRuns.POST("/:id/fail", handlers.FailTaskRun)
	taskRuns.POST("/:id/cancel", handlers.CancelTaskRun)
}

func registerSkillRoutes(api gin.IRouter, handlers *Handlers) {
	skillsGroup := api.Group("/skills")
	skillsGroup.GET("", handlers.ListSkills)
	skillsGroup.GET("/:name", handlers.GetSkill)
	skillsGroup.GET("/:name/content", handlers.GetSkillContent)
	skillsGroup.POST("/:name/enable", handlers.EnableSkill)
	skillsGroup.POST("/:name/disable", handlers.DisableSkill)
}

func registerMemoryRoutes(api gin.IRouter, handlers *Handlers) {
	memoryGroup := api.Group("/memory")
	memoryGroup.GET("", handlers.ListMemory)
	memoryGroup.GET("/health", handlers.MemoryHealth)
	memoryGroup.GET("/history", handlers.MemoryHistory)
	memoryGroup.GET("/search", handlers.SearchMemory)
	memoryGroup.POST("/reindex", handlers.ReindexMemory)
	memoryGroup.POST("/consolidate", handlers.ConsolidateMemory)
	memoryGroup.GET("/:filename", handlers.ReadMemory)
	memoryGroup.PUT("/:filename", handlers.WriteMemory)
	memoryGroup.DELETE("/:filename", handlers.DeleteMemory)
}

func registerDaemonRoutes(api gin.IRouter, handlers *Handlers) {
	daemonGroup := api.Group("/daemon")
	daemonGroup.GET("/status", handlers.DaemonStatus)
}

func registerNetworkRoutes(api gin.IRouter, handlers *Handlers) {
	networkGroup := api.Group("/network")
	networkGroup.GET("/status", handlers.NetworkStatus)
	networkGroup.GET("/peers", handlers.NetworkPeers)
	networkGroup.GET("/peers/:peer_id", handlers.NetworkPeer)
	networkGroup.GET("/peers/:peer_id/messages", handlers.NetworkPeerMessages)
	networkGroup.GET("/channels", handlers.NetworkChannels)
	networkGroup.POST("/channels", handlers.CreateNetworkChannel)
	networkGroup.GET("/channels/:channel", handlers.NetworkChannel)
	networkGroup.GET("/channels/:channel/messages", handlers.NetworkChannelMessages)
	networkGroup.POST("/send", handlers.NetworkSend)
	networkGroup.GET("/inbox", handlers.NetworkInbox)
}

func registerBundleRoutes(api gin.IRouter, handlers *Handlers) {
	bundles := api.Group("/bundles")
	bundles.GET("/catalog", handlers.ListBundleCatalog)
	bundles.POST("/preview", handlers.PreviewBundleActivation)
	bundles.GET("/activations", handlers.ListBundleActivations)
	bundles.POST("/activations", handlers.ActivateBundle)
	bundles.GET("/activations/:id", handlers.GetBundleActivation)
	bundles.PATCH("/activations/:id", handlers.UpdateBundleActivation)
	bundles.DELETE("/activations/:id", handlers.DeleteBundleActivation)
	bundles.GET("/network/settings", handlers.BundleNetworkSettings)
}

func registerExtensionRoutes(api gin.IRouter, handlers *Handlers) {
	privileged := handlers.privilegedMutationGuard()
	extensions := api.Group("/extensions")
	extensions.GET("", handlers.ListExtensions)
	extensions.POST("", privileged, handlers.InstallExtension)
	extensions.GET("/:name", handlers.ExtensionStatus)
	extensions.POST("/:name/enable", privileged, handlers.EnableExtension)
	extensions.POST("/:name/disable", privileged, handlers.DisableExtension)
}

func registerSettingsRoutes(api gin.IRouter, handlers *Handlers) {
	privileged := handlers.privilegedMutationGuard()
	settings := api.Group("/settings")

	settings.GET("/general", handlers.GetSettingsGeneral)
	settings.PATCH("/general", privileged, handlers.UpdateSettingsGeneral)
	settings.GET("/memory", handlers.GetSettingsMemory)
	settings.PATCH("/memory", privileged, handlers.UpdateSettingsMemory)
	settings.GET("/skills", handlers.GetSettingsSkills)
	settings.PATCH("/skills", privileged, handlers.UpdateSettingsSkills)
	settings.GET("/automation", handlers.GetSettingsAutomation)
	settings.PATCH("/automation", privileged, handlers.UpdateSettingsAutomation)
	settings.GET("/network", handlers.GetSettingsNetwork)
	settings.PATCH("/network", privileged, handlers.UpdateSettingsNetwork)

	observability := settings.Group("/observability")
	observability.GET("", handlers.GetSettingsObservability)
	observability.PATCH("", privileged, handlers.UpdateSettingsObservability)
	observability.GET("/log-tail", privileged, handlers.StreamSettingsObservabilityLogTail)

	settings.GET("/hooks-extensions", handlers.GetSettingsHooksExtensions)
	settings.PATCH("/hooks-extensions", privileged, handlers.UpdateSettingsHooksExtensions)

	settings.GET("/providers", handlers.ListSettingsProviders)
	settings.GET("/providers/:name", handlers.GetSettingsProvider)
	settings.PUT("/providers/:name", privileged, handlers.PutSettingsProvider)
	settings.DELETE("/providers/:name", privileged, handlers.DeleteSettingsProvider)

	settings.GET("/mcp-servers", handlers.ListSettingsMCPServers)
	settings.PUT("/mcp-servers/:name", privileged, handlers.PutSettingsMCPServer)
	settings.DELETE("/mcp-servers/:name", privileged, handlers.DeleteSettingsMCPServer)

	settings.GET("/sandboxes", handlers.ListSettingsSandboxes)
	settings.GET("/sandboxes/:name", handlers.GetSettingsSandbox)
	settings.PUT("/sandboxes/:name", privileged, handlers.PutSettingsSandbox)
	settings.DELETE("/sandboxes/:name", privileged, handlers.DeleteSettingsSandbox)

	settings.GET("/hooks", handlers.ListSettingsHooks)
	settings.PUT("/hooks/:name", privileged, handlers.PutSettingsHook)
	settings.DELETE("/hooks/:name", privileged, handlers.DeleteSettingsHook)

	actions := settings.Group("/actions")
	actions.POST("/restart", privileged, handlers.TriggerSettingsRestart)
	actions.GET("/restart/:operation_id", handlers.GetSettingsRestartStatus)
}

func registerVaultRoutes(api gin.IRouter, handlers *Handlers) {
	privileged := handlers.privilegedMutationGuard()
	vaultGroup := api.Group("/vault")
	vaultGroup.GET("/secrets", privileged, handlers.ListVaultSecrets)
	vaultGroup.GET("/secrets/metadata", privileged, handlers.GetVaultSecretMetadata)
	vaultGroup.PUT("/secrets", privileged, handlers.PutVaultSecret)
	vaultGroup.DELETE("/secrets", privileged, handlers.DeleteVaultSecret)
}

func registerWebhookRoutes(api gin.IRouter, handlers *Handlers) {
	webhooks := api.Group("/webhooks")
	webhooks.POST("/global/:endpoint", handlers.DeliverGlobalWebhook)
	webhooks.POST("/workspaces/:workspace_id/:endpoint", handlers.DeliverWorkspaceWebhook)
}
