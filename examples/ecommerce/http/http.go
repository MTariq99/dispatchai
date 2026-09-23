package http

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/models"
)

// ToolHandler is the signature every Host Project tool implementation
// must satisfy. Framework-agnostic on purpose — Gin never appears here.
type ToolHandler func(ctx context.Context, req *models.ToolRequest) (*models.ToolResponse, error)

type Handler struct {
	config *models.Config
	tools  map[string]ToolHandler
}

func NewHandler(cfg *models.Config, tools map[string]ToolHandler) *Handler {
	return &Handler{
		config: cfg,
		tools:  tools,
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/v1/ai/tools/execute", h.Execute)
}

func (h *Handler) Execute(c *gin.Context) {
	var req models.ToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	tool, ok := h.tools[req.ToolName]
	if !ok {
		c.JSON(http.StatusOK, &models.ToolResponse{
			RequestID: req.RequestID,
			CallID:    req.CallID,
			Success:   false,
			Error: &models.ToolError{
				Code:    "UNKNOWN_TOOL",
				Message: "no handler registered for tool: " + req.ToolName,
			},
		})
		return
	}

	resp, err := tool(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}
