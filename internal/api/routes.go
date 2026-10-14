package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/integration"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

type AssistantHandler struct {
	config              *models.Config
	handler             *Handler
	ToolRegistryHandler *tools.ToolRegistryHandler
	OnboardingHandler   *integration.IntegrationOnboardingHandler
}

func NewAssistantHandler(cfg *models.Config, handler *Handler, trh *tools.ToolRegistryHandler, onboardingHandler *integration.IntegrationOnboardingHandler) (*AssistantHandler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}
	return &AssistantHandler{
		config:              cfg,
		handler:             handler,
		ToolRegistryHandler: trh,
		OnboardingHandler:   onboardingHandler,
	}, nil
}

func (ah *AssistantHandler) RegisterRoutes(r *gin.Engine, authenticator IntegrationAuthenticator) {
	r.POST("/v1/assistant/chat", ah.handler.Chat)

	toolRoutes := r.Group("/v1/integrations/tools", IntegrationAuthMiddleware(authenticator))
	ah.ToolRegistryHandler.RegisterRoutes(toolRoutes)
	OnboardingRoutes := r.Group("/v1/admin")
	ah.OnboardingHandler.RegisterRoutes(OnboardingRoutes)
}
