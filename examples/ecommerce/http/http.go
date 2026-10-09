package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/models"
)

type ToolHandler func(
	ctx context.Context,
	req *models.ToolRequest,
) (*models.ToolResponse, error)

type Handler struct {
	config      *models.Config
	tools       map[string]ToolHandler
	idempotency *idempotencyStore
}

func NewHandler(
	cfg *models.Config,
	tools map[string]ToolHandler,
) *Handler {
	return &Handler{
		config:      cfg,
		tools:       tools,
		idempotency: newIdempotencyStore(),
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/api/v1/ai/tools/execute", h.Execute)
}

func (h *Handler) Execute(c *gin.Context) {
	var req models.ToolRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
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

	resp, err := h.idempotency.execute(
		c.Request.Context(),
		&req,
		func() (*models.ToolResponse, error) {
			return tool(c.Request.Context(), &req)
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, errIdempotencyKeyConflict):
			c.JSON(http.StatusConflict, gin.H{
				"error": "idempotency key reused with different request",
			})

		case errors.Is(err, errInvalidIdempotencyRequest):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid idempotency request",
			})

		case errors.Is(err, context.Canceled),
			errors.Is(err, context.DeadlineExceeded):
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": "request cancelled or deadline exceeded",
			})

		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}
