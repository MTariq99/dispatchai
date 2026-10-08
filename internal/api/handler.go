package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/llm"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

type Handler struct{}

func NewHandler(cfg *models.Config, llm llm.Client, registry *tools.Registry, executer *tools.Executer) (*Handler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}
	if llm == nil {
		return nil, fmt.Errorf("llm client cannot be empty")
	}
	if registry == nil {
		return nil, fmt.Errorf("no tools are registered")
	}
	if executer == nil {
		return nil, fmt.Errorf("executer is nil")
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "gemini-2.5-flash"
	}
	if cfg.LLM.MaxTokens <= 0 {
		cfg.LLM.MaxTokens = 1024
	}
	if cfg.LLM.Temperature < 0 {
		cfg.LLM.Temperature = 0.7
	}
	return &Handler{}, nil
}

func (h *Handler) Chat(c *gin.Context) {
	var req models.ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid request body: " + err.Error(),
		})
		return
	}

	// execCtx := models.ExecutionContext{
	// 	RequestID:      generateRequestID(),
	// 	ConversationID: req.ConversationID,
	// 	Identity:       models.Identity{UserID: req.UserID},
	// 	Tenant:         models.TenantContext{TenantID: req.TenantID},
	// 	Authorization: models.AuthorizationContext{
	// 		Roles:       req.Roles,
	// 		Permissions: req.Permissions,
	// 	},
	// }

	c.JSON(http.StatusNotImplemented, models.ErrorResponse{
		Error: "run orchestration is not wired yet",
	})
}
