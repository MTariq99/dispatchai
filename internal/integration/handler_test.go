package integration

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/middleware"
	"github.com/mtariq99/dispatchai/models"
)

type fakePendingIntegrationStore struct {
	clientID                 string
	displayName              string
	metadata                 []byte
	enrollmentTokenHash      []byte
	enrollmentTokenExpiresAt time.Time
	err                      error
	calls                    int
}
type fakeEnrollmentStore struct {
	params models.EnrollKeyParams
	err    error
	calls  int
}

func (s *fakePendingIntegrationStore) CreatePendingIntegration(
	_ context.Context,
	clientID string,
	displayName string,
	metadata []byte,
	enrollmentTokenHash []byte,
	enrollmentTokenExpiresAt time.Time,
) error {
	s.calls++
	s.clientID = clientID
	s.displayName = displayName
	s.metadata = append([]byte(nil), metadata...)
	s.enrollmentTokenHash = append([]byte(nil), enrollmentTokenHash...)
	s.enrollmentTokenExpiresAt = enrollmentTokenExpiresAt
	return s.err
}

func newTestHandler(t *testing.T, store PendingIntegrationStore, token string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	service, err := NewOnboardingService(store, &fakeEnrollmentStore{})
	if err != nil {
		t.Fatalf("create onboarding service: %v", err)
	}

	router := gin.New()
	admin := router.Group("/v1/admin", middleware.AdminAuth(token))
	NewIntegrationOnboardingHandler(service, nil).RegisterAdminRoutes(admin)
	return router
}

func TestCreateIntegrationSuccess(t *testing.T) {
	store := &fakePendingIntegrationStore{}
	router := newTestHandler(t, store, "test-admin-token")

	body := `{
		"display_name": "Example Host",
		"metadata": {"environment": "test"}
	}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/integrations",
		strings.NewReader(body),
	)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response models.CreateIntegrationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.Status != "pending" {
		t.Fatalf("expected pending status, got %q", response.Status)
	}

	if len(response.ClientID) != 68 ||
		!strings.HasPrefix(response.ClientID, "cli_") {
		t.Fatalf("unexpected client ID format: %q", response.ClientID)
	}

	if _, err := hex.DecodeString(strings.TrimPrefix(response.ClientID, "cli_")); err != nil {
		t.Fatalf("client ID suffix is not hexadecimal: %v", err)
	}

	if store.calls != 1 {
		t.Fatalf("expected one persistence call, got %d", store.calls)
	}

	if store.clientID != response.ClientID {
		t.Fatal("persisted client ID does not match response")
	}

	if store.displayName != "Example Host" {
		t.Fatalf("unexpected display name: %q", store.displayName)
	}

	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(store.metadata, &metadata); err != nil {
		t.Fatalf("decode persisted metadata: %v", err)
	}

	if string(metadata["environment"]) != `"test"` {
		t.Fatalf("unexpected persisted metadata: %s", store.metadata)
	}
}

func TestCreateIntegrationRejectsInvalidAuthorization(t *testing.T) {
	store := &fakePendingIntegrationStore{}
	router := newTestHandler(t, store, "test-admin-token")

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/integrations",
		strings.NewReader(`{"display_name":"Example Host"}`),
	)
	req.Header.Set("Authorization", "Bearer wrong-token")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d",
			http.StatusUnauthorized, recorder.Code)
	}

	if store.calls != 0 {
		t.Fatalf("store must not be called for unauthorized requests")
	}
}

func TestCreateIntegrationRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing display name",
			body: `{"metadata":{}}`,
		},
		{
			name: "unknown field",
			body: `{"display_name":"Example","unexpected":true}`,
		},
		{
			name: "malformed JSON",
			body: `{"display_name":`,
		},
		{
			name: "multiple JSON values",
			body: `{"display_name":"Example"} {"display_name":"Other"}`,
		},
		{
			name: "display name too long",
			body: `{"display_name":"` + strings.Repeat("a", 201) + `"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakePendingIntegrationStore{}
			router := newTestHandler(t, store, "test-admin-token")

			req := httptest.NewRequest(
				http.MethodPost,
				"/v1/admin/integrations",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Authorization", "Bearer test-admin-token")
			req.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s",
					http.StatusBadRequest, recorder.Code, recorder.Body.String())
			}

			if store.calls != 0 {
				t.Fatalf("store must not be called for invalid requests")
			}
		})
	}
}

func TestCreateIntegrationPersistenceFailure(t *testing.T) {
	store := &fakePendingIntegrationStore{
		err: errors.New("simulated database failure"),
	}
	router := newTestHandler(t, store, "test-admin-token")

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/integrations",
		strings.NewReader(`{"display_name":"Example Host"}`),
	)
	req.Header.Set("Authorization", "Bearer test-admin-token")
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	}

	if store.calls != 1 {
		t.Fatalf("expected one persistence attempt, got %d", store.calls)
	}

	if strings.Contains(recorder.Body.String(), `"status":"pending"`) {
		t.Fatal("failed persistence must never return a pending registration")
	}
}

func TestNewOnboardingServiceRejectsNilStores(t *testing.T) {
	if _, err := NewOnboardingService(nil, &fakeEnrollmentStore{}); err == nil {
		t.Fatal("expected an error for nil onboarding store")
	}
	if _, err := NewOnboardingService(&fakePendingIntegrationStore{}, nil); err == nil {
		t.Fatal("expected an error for nil enrollment store")
	}
}
