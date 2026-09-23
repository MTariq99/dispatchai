package api

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/assistant"
	"github.com/mtariq99/dispatchai/internal/llm"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

type Handler struct {
	assistant *assistant.Assistant
}

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
	orchestrator := assistant.NewOrchestrator(llm, registry, executer, cfg.LLM.Model, cfg.LLM.MaxTokens, cfg.LLM.Temperature)

	return &Handler{
		assistant: assistant.NewAssistant(orchestrator),
	}, nil
}

func (h *Handler) Chat(c *gin.Context) {
	var req models.ChatRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid request body: " + err.Error(),
		})
		return
	}

	execCtx := models.ExecutionContext{
		RequestID:      generateRequestID(),
		ConversationID: req.ConversationID,
		Identity:       models.Identity{UserID: req.UserID},
		Tenant:         models.TenantContext{TenantID: req.TenantID},
		Authorization: models.AuthorizationContext{
			Roles:       req.Roles,
			Permissions: req.Permissions,
		},
	}

	answer, err := h.assistant.HandleRequest(c.Request.Context(), req.Message, execCtx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.ChatResponse{
		RequestID: execCtx.RequestID,
		Answer:    answer,
	})
}

// generateRequestID produces a short random hex ID. Good enough for now;
// swap for github.com/google/uuid later if you want standard UUIDs.
func generateRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
