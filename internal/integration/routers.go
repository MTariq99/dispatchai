package integration

import "github.com/gin-gonic/gin"

func (h *IntegrationOnboardingHandler) RegisterRoutes(r gin.IRoutes) {
	r.POST("/v1/admin/integrations", h.Create)  // admin-token checked inside Create
	r.POST("/v1/integrations/enroll", h.Enroll) // no admin token; the one-time token is the credential
}
