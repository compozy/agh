package httpapi

import "github.com/gin-gonic/gin"

func registerDesktopStateRoutes(api gin.IRouter, handlers *Handlers) {
	desktopState := api.Group("/workspaces/:workspace_id/desktop-state")
	desktopState.GET("", handlers.ListDesktopState)
	desktopState.POST("/apply", handlers.ApplyDesktopState)
	desktopState.GET("/stream", handlers.StreamDesktopState)
	desktopState.GET("/:key", handlers.GetDesktopState)
	desktopState.PUT("/:key", handlers.PutDesktopState)
	desktopState.DELETE("/:key", handlers.DeleteDesktopState)
}
