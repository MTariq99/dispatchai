package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/internal/integrationctx"
)

type RequestVerifier interface {
	Verify(r *http.Request) (integrationID string, err error)
}

func SignedRequestAuth(verifier RequestVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		if verifier == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "integration authentication is not configured",
			})
			return
		}

		integrationID, err := verifier.Verify(c.Request)
		if err != nil {

			slog.Warn("signed request rejected", "client_id", c.GetHeader(enums.HeaderClientID), "reason", err.Error())
			if errors.Is(err, errs.ErrAuthUnavailable) {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error": "authentication temporarily unavailable",
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid request signature"})
			return
		}
		if strings.TrimSpace(integrationID) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid request signature"})
			return
		}

		integrationctx.Set(c, integrationID)
		c.Next()
	}
}
