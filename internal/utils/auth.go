package utils

import (
	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/integrationctx"
)

func AuthenticatedIntegrationID(c *gin.Context) (string, bool) {
	return integrationctx.Get(c)
}
