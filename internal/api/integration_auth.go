package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/integrationctx"
)

const integrationIdentityKey = "authenticated_integration_id"

var ErrInvalidIntegrationCredentials = errors.New(
	"invalid integration credentials",
)

type IntegrationAuthenticator interface {
	AuthenticateIntegration(ctx context.Context, token string) (integrationID string, err error)
}

func IntegrationAuthMiddleware(authenticator IntegrationAuthenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if authenticator == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "integration authentication is not configured",
			})
			return
		}

		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.Fields(header)

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
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

func integrationIDFromContext(c *gin.Context) (string, bool) {
	value, exists := c.Get(integrationIdentityKey)
	if !exists {
		return "", false
	}

	integrationID, ok := value.(string)
	if !ok || strings.TrimSpace(integrationID) == "" {
		return "", false
	}

	return integrationID, true
}
