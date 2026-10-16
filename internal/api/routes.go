package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/conversation"
	"github.com/mtariq99/dispatchai/internal/integration"
	"github.com/mtariq99/dispatchai/internal/middleware"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

type AssistantHandler struct {
	config              *models.Config
	ToolRegistryHandler *tools.ToolRegistryHandler
	OnboardingHandler   *integration.IntegrationOnboardingHandler
	conversationHandler *conversation.ConversationHandler
}

func NewAssistantHandler(cfg *models.Config, trh *tools.ToolRegistryHandler, onboardingHandler *integration.IntegrationOnboardingHandler, conversationHandler *conversation.ConversationHandler) (*AssistantHandler, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is empty")
	}
	return &AssistantHandler{
		config:              cfg,
		ToolRegistryHandler: trh,
		OnboardingHandler:   onboardingHandler,
		conversationHandler: conversationHandler,
	}, nil
}

func (ah *AssistantHandler) RegisterRoutes(r *gin.Engine, verifier middleware.RequestVerifier, adminToken string, permissionChecker middleware.PermissionChecker) {
	toolRoutes := r.Group("/v1/integrations/tools", middleware.SignedRequestAuth(verifier))
	ah.ToolRegistryHandler.RegisterRoutes(toolRoutes, permissionChecker)

	adminRoutes := r.Group("/v1/admin", middleware.AdminAuth(adminToken))
	ah.OnboardingHandler.RegisterAdminRoutes(adminRoutes)
	ah.OnboardingHandler.RegisterPublicRoutes(r)

	conversationRoutes := r.Group("/v1/conversation", middleware.SignedRequestAuth(verifier), middleware.RequirePermission(permissionChecker, "assistant:chat"))
	ah.conversationHandler.RegisterConversationRoutes(conversationRoutes)
}
