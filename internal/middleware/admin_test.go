package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func adminRouter(token string, reached *bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AdminAuth(token))
	r.GET("/x", func(c *gin.Context) { *reached = true; c.Status(http.StatusOK) })
	return r
}

func TestAdminAuth(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		header     string
		wantStatus int
		wantReach  bool
	}{
		{"valid", "secret", "Bearer secret", 200, true},
		{"case-insensitive scheme", "secret", "bearer secret", 200, true},
		{"missing header", "secret", "", 401, false},
		{"wrong token", "secret", "Bearer wrong!", 401, false},
		{"wrong length", "secret", "Bearer secret-longer", 401, false},
		{"wrong scheme", "secret", "Basic secret", 401, false},
		{"extra parts", "secret", "Bearer secret extra", 401, false},
		{"unconfigured token", "", "Bearer anything", 503, false},
		{"whitespace-only token", "   ", "Bearer anything", 503, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached := false
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			rec := httptest.NewRecorder()
			adminRouter(tt.token, &reached).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("expected %d, got %d", tt.wantStatus, rec.Code)
			}
			if reached != tt.wantReach {
				t.Fatalf("handler reached = %v, want %v", reached, tt.wantReach)
			}
		})
	}
}
