package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/integrationctx"
)

type IntegrationAuthenticator interface {
	AuthenticateIntegration(ctx context.Context, token string) (integrationID string, err error)
}

// IntegrationAuth is the legacy Bearer-token authentication for integration
// routes. It will be superseded by signed-request authentication.
func IntegrationAuth(authenticator IntegrationAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authenticator == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "integration authentication is not configured",
			})
			return
		}

		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "valid integration bearer token is required",
			})
			return
		}

		integrationID, err := authenticator.AuthenticateIntegration(c.Request.Context(), parts[1])
		if err != nil || strings.TrimSpace(integrationID) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid integration credentials",
			})
			return
		}

		integrationctx.Set(c, integrationID)
		c.Next()
	}
}
