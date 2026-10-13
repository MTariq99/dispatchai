package tools

import "github.com/gin-gonic/gin"

// RegisterRoutes registers tool management endpoints on a route group.
// The caller must supply a group protected by integration authentication.
func (h *ToolRegistryHandler) RegisterRoutes(routes *gin.RouterGroup) {
	routes.POST("", h.RegisterTool)
	routes.GET("", h.GetTool)
	routes.PATCH("/:toolID", h.UpdateToolDefinition)
	routes.PUT("/:toolID/status", h.ToggleTool)
	routes.DELETE("/:toolID", h.DeleteTool)
}
