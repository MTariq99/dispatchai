package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMaxBodyBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for name, tc := range map[string]struct {
		size    int
		wantErr bool
	}{"at limit": {10, false}, "over limit": {11, true}} {
		t.Run(name, func(t *testing.T) {
			var readErr error
			r := gin.New()
			r.Use(MaxBodyBytes(10))
			r.POST("/x", func(c *gin.Context) {
				_, readErr = io.ReadAll(c.Request.Body)
				c.Status(http.StatusOK)
			})
			req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(strings.Repeat("a", tc.size)))
			r.ServeHTTP(httptest.NewRecorder(), req)
			if (readErr != nil) != tc.wantErr {
				t.Fatalf("read error = %v, wantErr %v", readErr, tc.wantErr)
			}
		})
	}
}
