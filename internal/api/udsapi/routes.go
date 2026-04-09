package udsapi

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the shared AGH API routes on the supplied Gin router.
func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
	api := router.Group("/api")

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

	skillsGroup := api.Group("/skills")
	{
		skillsGroup.GET("", handlers.ListSkills)
		skillsGroup.GET("/:name", handlers.GetSkill)
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
}
