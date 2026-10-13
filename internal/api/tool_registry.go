package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

type ToolRegistryHandler struct {
	toolRegistry *tools.RegistrationService
}

func NewToolRegistryHandler(cfg *models.Config, toolRegistryService *tools.RegistrationService) (*ToolRegistryHandler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}

	if toolRegistryService == nil {
		return nil, fmt.Errorf("tool registration is nil")
	}

	return &ToolRegistryHandler{
		toolRegistry: toolRegistryService,
	}, nil
}

func (trh *ToolRegistryHandler) RegisterTool(c *gin.Context) {
}

func (trh *ToolRegistryHandler) GetToolsList(c *gin.Context) {}

func (trh *ToolRegistryHandler) UpdateToolDefinition(c *gin.Context) {}

func (trh *ToolRegistryHandler) ToggleTool(c *gin.Context) {}

func (trh *ToolRegistryHandler) DeleteTool(c *gin.Context) {}
