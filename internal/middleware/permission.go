package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/integrationctx"
)

type PermissionChecker interface {
	HasAccess(ctx context.Context, integrationID, permission string) (bool, error)
}

func RequirePermission(checker PermissionChecker, permission string) gin.HandlerFunc {
	requiredPermission := strings.TrimSpace(permission)
	return func(c *gin.Context) {
		if checker == nil || requiredPermission == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "permission authorization is not configured",
			})
			return
		}
		integrationID, ok := integrationctx.Get(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "integration authentication required",
			})
			return
		}
		granted, err := checker.HasAccess(c.Request.Context(), integrationID, requiredPermission)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "permission authorization temporarily unavailable",
			})
			return
		}

		if !granted {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "required permission not granted",
			})
			return
		}

		c.Next()
	}
}
