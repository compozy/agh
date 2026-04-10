package httpapi

import "github.com/gin-gonic/gin"

// RegisterRoutes registers the shared AGH API routes on the supplied Gin router.
func RegisterRoutes(router gin.IRouter, handlers *Handlers) {
	if handlers == nil {
		return
	}

	api := router.Group("/api")

	registerWorkspaceRoutes(api, handlers)
	registerSessionRoutes(api, handlers)
	registerAgentRoutes(api, handlers)
	registerObserveRoutes(api, handlers)
	registerHookRoutes(api, handlers)
	registerSkillRoutes(api, handlers)
	registerMemoryRoutes(api, handlers)
	registerDaemonRoutes(api, handlers)

	if engine, ok := router.(*gin.Engine); ok {
		engine.NoRoute(handlers.serveStaticRoute)
	}
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
	sessions.DELETE("/:id", handlers.StopSession)
	sessions.POST("/:id/resume", handlers.ResumeSession)
	sessions.POST("/:id/prompt", handlers.promptSession)
	sessions.GET("/:id/events", handlers.SessionEvents)
	sessions.GET("/:id/history", handlers.SessionHistory)
	sessions.GET("/:id/transcript", handlers.SessionTranscript)
	sessions.GET("/:id/stream", handlers.StreamSession)
	sessions.POST("/:id/approve", handlers.approveSession)
}

func registerAgentRoutes(api gin.IRouter, handlers *Handlers) {
	agents := api.Group("/agents")
	agents.GET("", handlers.ListAgents)
	agents.GET("/:name", handlers.GetAgent)
}

func registerObserveRoutes(api gin.IRouter, handlers *Handlers) {
	observeGroup := api.Group("/observe")
	observeGroup.GET("/events", handlers.ObserveEvents)
	observeGroup.GET("/events/stream", handlers.StreamObserveEvents)
	observeGroup.GET("/health", handlers.Health)
}

func registerHookRoutes(api gin.IRouter, handlers *Handlers) {
	hooksGroup := api.Group("/hooks")
	hooksGroup.GET("/catalog", handlers.HookCatalog)
	hooksGroup.GET("/runs", handlers.HookRuns)
	hooksGroup.GET("/events", handlers.HookEvents)
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
	memoryGroup.GET("/:filename", handlers.ReadMemory)
	memoryGroup.PUT("/:filename", handlers.WriteMemory)
	memoryGroup.DELETE("/:filename", handlers.DeleteMemory)
	memoryGroup.POST("/consolidate", handlers.ConsolidateMemory)
}

func registerDaemonRoutes(api gin.IRouter, handlers *Handlers) {
	daemonGroup := api.Group("/daemon")
	daemonGroup.GET("/status", handlers.DaemonStatus)
}
