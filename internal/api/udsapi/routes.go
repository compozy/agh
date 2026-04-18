package udsapi

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the shared AGH API routes on the supplied Gin router.
func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
	api := router.Group("/api")

	registerBridgeRoutes(api, handlers)
	registerWorkspaceRoutes(api, handlers)
	registerSessionRoutes(api, handlers)
	registerAgentRoutes(api, handlers)
	registerObserveRoutes(api, handlers)
	registerHookRoutes(api, handlers)
	registerResourceRoutes(api, handlers)
	registerAutomationRoutes(api, handlers)
	registerTaskRoutes(api, handlers)
	registerTaskRunRoutes(api, handlers)
	registerSkillRoutes(api, handlers)
	registerMemoryRoutes(api, handlers)
	registerDaemonRoutes(api, handlers)
	registerNetworkRoutes(api, handlers)
	registerBundleRoutes(api, handlers)
	registerExtensionRoutes(api, handlers)
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
		bridges.GET("/:id/secret-bindings", handlers.ListBridgeSecretBindings)
		bridges.PUT("/:id/secret-bindings/:binding_name", handlers.PutBridgeSecretBinding)
		bridges.DELETE("/:id/secret-bindings/:binding_name", handlers.DeleteBridgeSecretBinding)
		bridges.POST("/:id/test-delivery", handlers.TestBridgeDelivery)
	}
}

func registerWorkspaceRoutes(api gin.IRouter, handlers *Handlers) {
	workspaces := api.Group("/workspaces")
	{
		workspaces.POST("", handlers.CreateWorkspace)
		workspaces.GET("", handlers.ListWorkspaces)
		workspaces.GET("/:id", handlers.GetWorkspace)
		workspaces.PATCH("/:id", handlers.UpdateWorkspace)
		workspaces.DELETE("/:id", handlers.DeleteWorkspace)
		workspaces.POST("/resolve", handlers.ResolveWorkspace)
	}
}

func registerSessionRoutes(api gin.IRouter, handlers *Handlers) {
	sessions := api.Group("/sessions")
	{
		sessions.GET("", handlers.ListSessions)
		sessions.POST("", handlers.CreateSession)
		sessions.GET("/:id", handlers.GetSession)
		sessions.DELETE("/:id", handlers.StopSession)
		sessions.POST("/:id/resume", handlers.ResumeSession)
		sessions.POST("/:id/prompt", handlers.promptSession)
		sessions.GET("/:id/events", handlers.SessionEvents)
		sessions.GET("/:id/history", handlers.SessionHistory)
		sessions.GET("/:id/transcript", handlers.SessionTranscript)
		sessions.GET("/:id/stream", handlers.StreamSession)
		sessions.POST("/:id/approve", handlers.approveSession)
	}
}

func registerAgentRoutes(api gin.IRouter, handlers *Handlers) {
	agents := api.Group("/agents")
	{
		agents.GET("", handlers.ListAgents)
		agents.GET("/:name", handlers.GetAgent)
	}
}

func registerObserveRoutes(api gin.IRouter, handlers *Handlers) {
	observe := api.Group("/observe")
	{
		observe.GET("/events", handlers.ObserveEvents)
		observe.GET("/events/stream", handlers.StreamObserveEvents)
		observe.GET("/health", handlers.Health)
	}
}

func registerHookRoutes(api gin.IRouter, handlers *Handlers) {
	hooksGroup := api.Group("/hooks")
	{
		hooksGroup.GET("/catalog", handlers.HookCatalog)
		hooksGroup.GET("/runs", handlers.HookRuns)
		hooksGroup.GET("/events", handlers.HookEvents)
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
		tasks.PATCH("/:id", handlers.UpdateTask)
		tasks.POST("/:id/cancel", handlers.CancelTask)
		tasks.POST("/:id/children", handlers.CreateChildTask)
		tasks.POST("/:id/dependencies", handlers.AddTaskDependency)
		tasks.DELETE("/:id/dependencies/:depends_on_id", handlers.RemoveTaskDependency)
		tasks.POST("/:id/runs", handlers.EnqueueTaskRun)
		tasks.GET("/:id/runs", handlers.ListTaskRuns)
	}
}

func registerTaskRunRoutes(api gin.IRouter, handlers *Handlers) {
	taskRuns := api.Group("/task-runs")
	{
		taskRuns.POST("/:id/claim", handlers.ClaimTaskRun)
		taskRuns.POST("/:id/start", handlers.StartTaskRun)
		taskRuns.POST("/:id/attach-session", handlers.AttachTaskRunSession)
		taskRuns.POST("/:id/complete", handlers.CompleteTaskRun)
		taskRuns.POST("/:id/fail", handlers.FailTaskRun)
		taskRuns.POST("/:id/cancel", handlers.CancelTaskRun)
	}
}

func registerSkillRoutes(api gin.IRouter, handlers *Handlers) {
	skillsGroup := api.Group("/skills")
	{
		skillsGroup.GET("", handlers.ListSkills)
		skillsGroup.GET("/:name", handlers.GetSkill)
		skillsGroup.GET("/:name/content", handlers.GetSkillContent)
		skillsGroup.POST("/:name/enable", handlers.EnableSkill)
		skillsGroup.POST("/:name/disable", handlers.DisableSkill)
	}
}

func registerMemoryRoutes(api gin.IRouter, handlers *Handlers) {
	memoryGroup := api.Group("/memory")
	{
		memoryGroup.GET("", handlers.ListMemory)
		memoryGroup.GET("/search", handlers.SearchMemory)
		memoryGroup.POST("/reindex", handlers.ReindexMemory)
		memoryGroup.POST("/consolidate", handlers.ConsolidateMemory)
		memoryGroup.GET("/:filename", handlers.ReadMemory)
		memoryGroup.PUT("/:filename", handlers.WriteMemory)
		memoryGroup.DELETE("/:filename", handlers.DeleteMemory)
	}
}

func registerDaemonRoutes(api gin.IRouter, handlers *Handlers) {
	daemon := api.Group("/daemon")
	{
		daemon.GET("/status", handlers.DaemonStatus)
	}
}

func registerNetworkRoutes(api gin.IRouter, handlers *Handlers) {
	network := api.Group("/network")
	{
		network.GET("/status", handlers.NetworkStatus)
		network.GET("/peers", handlers.NetworkPeers)
		network.GET("/peers/:peer_id", handlers.NetworkPeer)
		network.GET("/channels", handlers.NetworkChannels)
		network.POST("/channels", handlers.CreateNetworkChannel)
		network.GET("/channels/:channel", handlers.NetworkChannel)
		network.GET("/channels/:channel/messages", handlers.NetworkChannelMessages)
		network.POST("/send", handlers.NetworkSend)
		network.GET("/inbox", handlers.NetworkInbox)
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
		extensions.POST("", handlers.InstallExtension)
		extensions.GET("/:name", handlers.ExtensionStatus)
		extensions.POST("/:name/enable", handlers.EnableExtension)
		extensions.POST("/:name/disable", handlers.DisableExtension)
	}
}
