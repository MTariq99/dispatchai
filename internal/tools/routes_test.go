package tools

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/middleware"
)

type allowToolRouteVerifier struct{}

func (allowToolRouteVerifier) Verify(*http.Request) (string, error) {
	return "authenticated-integration-123", nil
}

type recordingPermissionChecker struct {
	permission    string
	integrationID string
	granted       bool
	err           error
	calls         int
}

func (f *recordingPermissionChecker) HasAccess(
	_ context.Context,
	integrationID string,
	permission string,
) (bool, error) {
	f.calls++
	f.integrationID = integrationID
	f.permission = permission
	return f.granted, f.err
}

func TestToolRoutesRequireTheirSpecificPermissions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		method     string
		path       string
		permission string
	}{
		{
			name:       "register tool",
			method:     http.MethodPost,
			path:       "/v1/integrations/tools",
			permission: "tools:register",
		},
		{
			name:       "read tool",
			method:     http.MethodGet,
			path:       "/v1/integrations/tools/tool-123",
			permission: "tools:read",
		},
		{
			name:       "update tool",
			method:     http.MethodPatch,
			path:       "/v1/integrations/tools/tool-123",
			permission: "tools:update",
		},
		{
			name:       "enable tool",
			method:     http.MethodPut,
			path:       "/v1/integrations/tools/tool-123/status",
			permission: "tools:enable",
		},
		{
			name:       "delete tool",
			method:     http.MethodDelete,
			path:       "/v1/integrations/tools/tool-123",
			permission: "tools:delete",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &recordingPermissionChecker{granted: false}
			router := gin.New()

			routes := router.Group(
				"/v1/integrations/tools",
				middleware.SignedRequestAuth(allowToolRouteVerifier{}),
			)
			(&ToolRegistryHandler{}).RegisterRoutes(routes, checker)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body: %s",
					rec.Code, http.StatusForbidden, rec.Body.String())
			}

			if checker.calls != 1 {
				t.Fatalf("permission checker calls = %d, want 1", checker.calls)
			}

			if checker.integrationID != "authenticated-integration-123" {
				t.Fatalf("integration ID = %q, want authenticated identity",
					checker.integrationID)
			}

			if checker.permission != tt.permission {
				t.Fatalf("permission = %q, want %q",
					checker.permission, tt.permission)
			}
		})
	}
}

func TestToolRoutesFailClosedWhenPermissionStoreFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	checker := &recordingPermissionChecker{
		err: errors.New("database unavailable"),
	}

	router := gin.New()
	routes := router.Group(
		"/v1/integrations/tools",
		middleware.SignedRequestAuth(allowToolRouteVerifier{}),
	)
	(&ToolRegistryHandler{}).RegisterRoutes(routes, checker)

	req := httptest.NewRequest(
		http.MethodGet,
		"/v1/integrations/tools/tool-123",
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body: %s",
			rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}

	if checker.calls != 1 {
		t.Fatalf("permission checker calls = %d, want 1", checker.calls)
	}
}
