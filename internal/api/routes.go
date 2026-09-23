package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/models"
)

// This file registers the HTTP routes exposed by DispatchAI.
//
// Example routes:
//
//   POST /v1/assistant/chat
//   POST /v1/assistant/dispatch
//   POST /v1/assistant/events
//   GET  /v1/assistant/conversations/:id
//
// Route registration belongs here so the HTTP entry surface remains
// easy to discover and maintain.

type AssistantHandler struct {
	config  *models.Config
	handler *Handler
}

func NewAssistantHandler(cfg *models.Config, handler *Handler) (*AssistantHandler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}
	return &AssistantHandler{
		config:  cfg,
		handler: handler,
	}, nil
}

func (ah *AssistantHandler) RegisterRoutes(r *gin.Engine) {
	r.POST("/v1/assistant/chat")
}
