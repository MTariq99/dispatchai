package tools

import "github.com/gin-gonic/gin"

func (h *ToolRegistryHandler) RegisterRoutes(routes *gin.RouterGroup) {
	routes.POST("", h.RegisterTool)
	routes.GET("/:toolID", h.GetTool)
	routes.PATCH("/:toolID", h.UpdateToolDefinition)
	routes.PUT("/:toolID/status", h.ToggleTool)
	routes.DELETE("/:toolID", h.DeleteTool)
}
