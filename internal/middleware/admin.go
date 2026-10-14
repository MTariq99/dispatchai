package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AdminAuth(adminToken string) gin.HandlerFunc {
	expected := strings.TrimSpace(adminToken)

	return func(c *gin.Context) {
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "integration onboarding is not configured",
			})
			return
		}

		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") ||
			subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "valid onboarding credentials are required",
			})
			return
		}

		c.Next()
	}
}
