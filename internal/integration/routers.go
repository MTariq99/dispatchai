package integration

import (
	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/enums"
	"github.com/mtariq99/dispatchai/internal/middleware"
)

func (h *IntegrationOnboardingHandler) RegisterAdminRoutes(r gin.IRoutes) {
	r.POST("/integrations", middleware.MaxBodyBytes(enums.MaxOnboardingBodyBytes), h.Create)
	r.POST("/integrations/:client_id/toggle", h.ToggleIntegration)
}

func (h *IntegrationOnboardingHandler) RegisterPublicRoutes(r gin.IRoutes) {
	r.POST("/v1/integrations/enroll", middleware.MaxBodyBytes(enums.MaxEnrollmentBodyBytes), h.Enroll)
}
