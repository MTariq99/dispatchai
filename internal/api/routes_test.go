package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/integration"
	"github.com/mtariq99/dispatchai/internal/tools"
)

type denyVerifier struct{}

func (denyVerifier) Verify(*http.Request) (string, error) { return "", errors.New("denied") }

func routerWithAdminToken(token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	ah := &AssistantHandler{
		handler:             &Handler{},
		ToolRegistryHandler: &tools.ToolRegistryHandler{},
		OnboardingHandler:   &integration.IntegrationOnboardingHandler{},
	}
	r := gin.New()
	ah.RegisterRoutes(r, denyVerifier{}, token)
	return r
}

func post(r *gin.Engine, path string) int {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Code
}

func TestRouteGuards(t *testing.T) {
	r := routerWithAdminToken("admin-secret")

	if got := post(r, "/v1/admin/integrations"); got != http.StatusUnauthorized {
		t.Fatalf("admin route without token: want 401, got %d", got)
	}
	if got := post(r, "/v1/integrations/tools"); got != http.StatusUnauthorized {
		t.Fatalf("tool route without credentials: want 401, got %d", got)
	}
	// Enrollment must be reachable without any bearer credential. The handler
	// here has no service, so 503 proves we got past the (absent) auth layer.
	if got := post(r, "/v1/integrations/enroll"); got != http.StatusServiceUnavailable {
		t.Fatalf("enroll route: want 503 from unconfigured handler, got %d", got)
	}
}

func TestAdminRoutesFailClosedWithoutToken(t *testing.T) {
	r := routerWithAdminToken("")
	if got := post(r, "/v1/admin/integrations"); got != http.StatusServiceUnavailable {
		t.Fatalf("want 503 when admin token unset, got %d", got)
	}
}
