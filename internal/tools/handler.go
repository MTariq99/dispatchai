package tools

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/integrationctx"
	"github.com/mtariq99/dispatchai/models"
)

type ToolRegistryHandler struct {
	toolRegistry *RegistrationService
}

func NewToolRegistryHandler(cfg *models.Config, toolRegistryService *RegistrationService) (*ToolRegistryHandler, error) {
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

func authenticatedIntegrationID(c *gin.Context) (string, bool) {
	return integrationctx.Get(c)
}

func (h *ToolRegistryHandler) RegisterTool(c *gin.Context) {
	integrationID, ok := authenticatedIntegrationID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration authentication required"})
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key header is required"})
		return
	}

	var definition models.Definition
	if err := c.ShouldBindJSON(&definition); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tool definition"})
		return
	}

	record, replayed, err := h.toolRegistry.CreateTool(
		c.Request.Context(),
		integrationID,
		idempotencyKey,
		&definition,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}

	c.JSON(status, gin.H{
		"tool":     record,
		"replayed": replayed,
	})
}

func (h *ToolRegistryHandler) GetTool(c *gin.Context) {
	integrationID, ok := authenticatedIntegrationID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration authentication required"})
		return
	}

	record, err := h.toolRegistry.Get(c.Request.Context(), integrationID, c.Param("toolID"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tool not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tool": record})
}

func (h *ToolRegistryHandler) UpdateToolDefinition(c *gin.Context) {
	integrationID, ok := authenticatedIntegrationID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration authentication required"})
		return
	}

	var req models.UpdateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid update request"})
		return
	}
	if req.ExpectedVersion < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expected_version must be positive"})
		return
	}

	record, err := h.toolRegistry.Update(
		c.Request.Context(),
		integrationID,
		c.Param("toolID"),
		req.ExpectedVersion,
		&req.Definition,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tool": record})
}

func (h *ToolRegistryHandler) ToggleTool(c *gin.Context) {
	integrationID, ok := authenticatedIntegrationID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration authentication required"})
		return
	}

	var req models.ToggleToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status request"})
		return
	}
	if req.ExpectedVersion < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expected_version must be positive"})
		return
	}

	record, err := h.toolRegistry.SetEnabled(
		c.Request.Context(),
		integrationID,
		c.Param("toolID"),
		req.ExpectedVersion,
		req.Enabled,
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tool": record})
}

func (h *ToolRegistryHandler) DeleteTool(c *gin.Context) {
	integrationID, ok := authenticatedIntegrationID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "integration authentication required"})
		return
	}

	version, err := strconv.Atoi(c.Query("expected_version"))
	if err != nil || version < 1 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "positive expected_version query parameter is required",
		})
		return
	}

	if err := h.toolRegistry.Delete(
		c.Request.Context(),
		integrationID,
		c.Param("toolID"),
		version,
	); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
