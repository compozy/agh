package udsapi

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the shared AGH API routes on the supplied Gin router.
func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
	api := router.Group("/api")

	channels := api.Group("/channels")
	{
		channels.GET("", handlers.ListChannels)
		channels.POST("", handlers.CreateChannel)
		channels.GET("/:id", handlers.GetChannel)
		channels.PATCH("/:id", handlers.UpdateChannel)
		channels.POST("/:id/enable", handlers.EnableChannel)
		channels.POST("/:id/disable", handlers.DisableChannel)
		channels.POST("/:id/restart", handlers.RestartChannel)
		channels.GET("/:id/routes", handlers.ListChannelRoutes)
		channels.POST("/:id/test-delivery", handlers.TestChannelDelivery)
	}

	workspaces := api.Group("/workspaces")
	{
		workspaces.POST("", handlers.CreateWorkspace)
		workspaces.GET("", handlers.ListWorkspaces)
		workspaces.GET("/:id", handlers.GetWorkspace)
		workspaces.PATCH("/:id", handlers.UpdateWorkspace)
		workspaces.DELETE("/:id", handlers.DeleteWorkspace)
		workspaces.POST("/resolve", handlers.ResolveWorkspace)
	}

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

	agents := api.Group("/agents")
	{
		agents.GET("", handlers.ListAgents)
		agents.GET("/:name", handlers.GetAgent)
	}

	observe := api.Group("/observe")
	{
		observe.GET("/events", handlers.ObserveEvents)
		observe.GET("/events/stream", handlers.StreamObserveEvents)
		observe.GET("/health", handlers.Health)
	}

	hooksGroup := api.Group("/hooks")
	{
		hooksGroup.GET("/catalog", handlers.HookCatalog)
		hooksGroup.GET("/runs", handlers.HookRuns)
		hooksGroup.GET("/events", handlers.HookEvents)
	}

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

	skillsGroup := api.Group("/skills")
	{
		skillsGroup.GET("", handlers.ListSkills)
		skillsGroup.GET("/:name", handlers.GetSkill)
		skillsGroup.GET("/:name/content", handlers.GetSkillContent)
		skillsGroup.POST("/:name/enable", handlers.EnableSkill)
		skillsGroup.POST("/:name/disable", handlers.DisableSkill)
	}

	memoryGroup := api.Group("/memory")
	{
		memoryGroup.GET("", handlers.ListMemory)
		memoryGroup.GET("/:filename", handlers.ReadMemory)
		memoryGroup.PUT("/:filename", handlers.WriteMemory)
		memoryGroup.DELETE("/:filename", handlers.DeleteMemory)
		memoryGroup.POST("/consolidate", handlers.ConsolidateMemory)
	}

	daemon := api.Group("/daemon")
	{
		daemon.GET("/status", handlers.DaemonStatus)
	}

	extensions := api.Group("/extensions")
	{
		extensions.GET("", handlers.ListExtensions)
		extensions.POST("", handlers.InstallExtension)
		extensions.GET("/:name", handlers.ExtensionStatus)
		extensions.POST("/:name/enable", handlers.EnableExtension)
		extensions.POST("/:name/disable", handlers.DisableExtension)
	}
}
