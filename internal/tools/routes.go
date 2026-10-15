package tools

import (
	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/middleware"
)

func (h *ToolRegistryHandler) RegisterRoutes(routes *gin.RouterGroup, permissionChecker middleware.PermissionChecker) {
	routes.POST("", middleware.RequirePermission(permissionChecker, "tools:register"), h.RegisterTool)
	routes.GET("/:toolID", middleware.RequirePermission(permissionChecker, "tools:read"), h.GetTool)
	routes.PATCH("/:toolID", middleware.RequirePermission(permissionChecker, "tools:update"), h.UpdateToolDefinition)
	routes.PUT("/:toolID/status", middleware.RequirePermission(permissionChecker, "tools:enable"), h.ToggleTool)
	routes.DELETE("/:toolID", middleware.RequirePermission(permissionChecker, "tools:delete"), h.DeleteTool)
}
