package integrationctx

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const integrationIDKey = "authenticated_integration_id"

func Set(c *gin.Context, integrationID string) {
	c.Set(integrationIDKey, integrationID)
}

func Get(c *gin.Context) (string, bool) {
	value, exists := c.Get(integrationIDKey)
	if !exists {
		return "", false
	}

	id, ok := value.(string)
	if !ok || strings.TrimSpace(id) == "" {
		return "", false
	}

	return id, true
}
