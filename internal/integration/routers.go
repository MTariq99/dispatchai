package integration

import (
	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/enums"
	"github.com/mtariq99/dispatchai/internal/middleware"
)

func (h *IntegrationOnboardingHandler) RegisterAdminRoutes(r gin.IRoutes) {
	body := middleware.MaxBodyBytes(enums.MaxOnboardingBodyBytes)

	r.POST("/integrations", body, h.Create)
	r.POST("/integrations/:client_id/toggle", body, h.ToggleIntegration)
	r.GET("/integrations/permissions", h.GetGrantedPermissions)
	r.POST("/integrations/:client_id/permissions", body, h.UpsertGrantedPermission)
	r.GET("/permissions", h.GetPermissions)
}

func (h *IntegrationOnboardingHandler) RegisterPublicRoutes(r gin.IRoutes) {
	r.POST("/v1/integrations/enroll", middleware.MaxBodyBytes(enums.MaxEnrollmentBodyBytes), h.Enroll)
}
