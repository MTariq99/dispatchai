package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/integrationctx"
)

type stubAuthenticator struct {
	id       string
	err      error
	gotToken string
}

func (s *stubAuthenticator) AuthenticateIntegration(_ context.Context, token string) (string, error) {
	s.gotToken = token
	return s.id, s.err
}

func run(auth IntegrationAuthenticator, header string) (*httptest.ResponseRecorder, string, bool) {
	gin.SetMode(gin.TestMode)
	var gotID string
	reached := false
	r := gin.New()
	r.Use(IntegrationAuth(auth))
	r.GET("/x", func(c *gin.Context) {
		reached = true
		gotID, _ = integrationctx.Get(c)
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec, gotID, reached
}

func TestIntegrationAuthSuccessSetsIdentity(t *testing.T) {
	stub := &stubAuthenticator{id: "integration-1"}
	rec, id, reached := run(stub, "Bearer tok123")
	if rec.Code != 200 || !reached || id != "integration-1" || stub.gotToken != "tok123" {
		t.Fatalf("code=%d reached=%v id=%q token=%q", rec.Code, reached, id, stub.gotToken)
	}
}

func TestIntegrationAuthRejects(t *testing.T) {
	tests := []struct {
		name   string
		auth   IntegrationAuthenticator
		header string
		status int
	}{
		{"nil authenticator", nil, "Bearer t", 503},
		{"missing header", &stubAuthenticator{id: "x"}, "", 401},
		{"wrong scheme", &stubAuthenticator{id: "x"}, "Basic t", 401},
		{"authenticator error", &stubAuthenticator{err: errors.New("no")}, "Bearer t", 401},
		{"blank identity", &stubAuthenticator{id: "  "}, "Bearer t", 401},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, _, reached := run(tt.auth, tt.header)
			if rec.Code != tt.status || reached {
				t.Fatalf("code=%d reached=%v, want %d and not reached", rec.Code, reached, tt.status)
			}
		})
	}
}
