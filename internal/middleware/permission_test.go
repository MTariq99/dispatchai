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

type fakePermissionChecker struct {
	granted       bool
	err           error
	calls         int
	integrationID string
	permission    string
}

func (f *fakePermissionChecker) HasAccess(
	ctx context.Context,
	integrationID string,
	permission string,
) (bool, error) {
	f.calls++
	f.integrationID = integrationID
	f.permission = permission
	return f.granted, f.err
}

func newPermissionTestRouter(
	checker PermissionChecker,
	integrationID string,
	handlerCalled *bool,
) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	// Simulate the identity established by SignedRequestAuth.
	if integrationID != "" {
		router.Use(func(c *gin.Context) {
			integrationctx.Set(c, integrationID)
			c.Next()
		})
	}

	router.GET(
		"/protected",
		RequirePermission(checker, "tools:read"),
		func(c *gin.Context) {
			*handlerCalled = true
			c.Status(http.StatusOK)
		},
	)

	return router
}

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name            string
		integrationID   string
		checker         *fakePermissionChecker
		wantStatus      int
		wantCheckerCall bool
		wantHandler     bool
	}{
		{
			name:            "missing authenticated identity",
			integrationID:   "",
			checker:         &fakePermissionChecker{granted: true},
			wantStatus:      http.StatusUnauthorized,
			wantCheckerCall: false,
			wantHandler:     false,
		},
		{
			name:            "permission not granted",
			integrationID:   "integration-123",
			checker:         &fakePermissionChecker{granted: false},
			wantStatus:      http.StatusForbidden,
			wantCheckerCall: true,
			wantHandler:     false,
		},
		{
			name:            "permission store failure",
			integrationID:   "integration-123",
			checker:         &fakePermissionChecker{err: errors.New("database unavailable")},
			wantStatus:      http.StatusServiceUnavailable,
			wantCheckerCall: true,
			wantHandler:     false,
		},
		{
			name:            "permission granted",
			integrationID:   "integration-123",
			checker:         &fakePermissionChecker{granted: true},
			wantStatus:      http.StatusOK,
			wantCheckerCall: true,
			wantHandler:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlerCalled := false
			router := newPermissionTestRouter(
				tt.checker,
				tt.integrationID,
				&handlerCalled,
			)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s",
					rec.Code, tt.wantStatus, rec.Body.String())
			}

			if handlerCalled != tt.wantHandler {
				t.Fatalf("handlerCalled = %v, want %v",
					handlerCalled, tt.wantHandler)
			}

			if (tt.checker.calls > 0) != tt.wantCheckerCall {
				t.Fatalf("checker called = %v, want %v",
					tt.checker.calls > 0, tt.wantCheckerCall)
			}
		})
	}
}

func TestRequirePermissionUsesAuthenticatedIdentity(t *testing.T) {
	checker := &fakePermissionChecker{granted: true}
	handlerCalled := false

	router := newPermissionTestRouter(
		checker,
		"authenticated-integration",
		&handlerCalled,
	)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)

	// A caller-controlled header must not replace the authenticated identity.
	req.Header.Set("X-Integration-ID", "attacker-controlled-integration")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s",
			rec.Code, http.StatusOK, rec.Body.String())
	}

	if checker.integrationID != "authenticated-integration" {
		t.Fatalf("checker received integration ID %q, want authenticated identity",
			checker.integrationID)
	}

	if checker.permission != "tools:read" {
		t.Fatalf("checker received permission %q, want tools:read",
			checker.permission)
	}

	if !handlerCalled {
		t.Fatal("expected protected handler to execute")
	}
}

func TestRequirePermissionRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		checker    PermissionChecker
		permission string
	}{
		{
			name:       "nil checker",
			checker:    nil,
			permission: "tools:read",
		},
		{
			name:       "blank permission",
			checker:    &fakePermissionChecker{granted: true},
			permission: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			handlerCalled := false

			router.GET("/protected",
				RequirePermission(tt.checker, tt.permission),
				func(c *gin.Context) {
					handlerCalled = true
					c.Status(http.StatusOK)
				},
			)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d",
					rec.Code, http.StatusServiceUnavailable)
			}

			if handlerCalled {
				t.Fatal("protected handler executed with invalid configuration")
			}
		})
	}
}
