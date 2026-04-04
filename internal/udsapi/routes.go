package udsapi

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the shared AGH API routes on the supplied Gin router.
func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
	api := router.Group("/api")

	sessions := api.Group("/sessions")
	{
		sessions.GET("", handlers.listSessions)
		sessions.POST("", handlers.createSession)
		sessions.GET("/:id", handlers.getSession)
		sessions.DELETE("/:id", handlers.stopSession)
		sessions.POST("/:id/resume", handlers.resumeSession)
		sessions.POST("/:id/prompt", handlers.promptSession)
		sessions.GET("/:id/events", handlers.sessionEvents)
		sessions.GET("/:id/history", handlers.sessionHistory)
		sessions.GET("/:id/stream", handlers.streamSession)
		sessions.POST("/:id/approve", handlers.approveSession)
	}

	agents := api.Group("/agents")
	{
		agents.GET("", handlers.listAgents)
		agents.GET("/:name", handlers.getAgent)
	}

	observe := api.Group("/observe")
	{
		observe.GET("/events", handlers.observeEvents)
		observe.GET("/events/stream", handlers.streamObserveEvents)
		observe.GET("/health", handlers.health)
	}

	memoryGroup := api.Group("/memory")
	{
		memoryGroup.GET("", handlers.listMemory)
		memoryGroup.GET("/:filename", handlers.readMemory)
		memoryGroup.PUT("/:filename", handlers.writeMemory)
		memoryGroup.DELETE("/:filename", handlers.deleteMemory)
		memoryGroup.POST("/consolidate", handlers.consolidateMemory)
	}

	daemon := api.Group("/daemon")
	{
		daemon.GET("/status", handlers.daemonStatus)
	}
}
