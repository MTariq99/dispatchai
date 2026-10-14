package integration

import "github.com/gin-gonic/gin"

func (h *IntegrationOnboardingHandler) RegisterRoutes(r gin.IRoutes) {
	r.POST("/integrations", h.Create)
}
