package integration

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

func (s *fakeEnrollmentStore) EnrollKey(_ context.Context, p models.EnrollKeyParams) (*models.EnrolledKey, error) {
	s.calls++
	s.params = models.EnrollKeyParams{
		TokenHash: append([]byte(nil), p.TokenHash...),
		KeyID:     p.KeyID,
		PublicKey: append([]byte(nil), p.PublicKey...),
	}
	if s.err != nil {
		return nil, s.err
	}
	return &models.EnrolledKey{IntegrationID: "integration-id", ClientID: "cli_test_client", KeyID: p.KeyID}, nil
}

func (s *fakePendingIntegrationStore) GetHostIntegrationByClientID(_ context.Context, clientID string) (*models.HostIntegration, error) {
	return nil, nil
}

func (s *fakePendingIntegrationStore) UpateHostIntegrationStatus(_ context.Context, _ string, _ string) error {
	return nil
}

func newEnrollRouter(t *testing.T, enrollment EnrollmentStore) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	service, err := NewOnboardingService(&fakePendingIntegrationStore{}, enrollment)
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	router := gin.New()
	NewIntegrationOnboardingHandler(service).RegisterPublicRoutes(router)
	return router
}

func testToken(t *testing.T) string {
	t.Helper()
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func testPublicKey(t *testing.T) []byte {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return priv.PublicKey().Bytes()
}

func enrollJSON(t *testing.T, token, keyID, pubB64 string) string {
	t.Helper()
	b, err := json.Marshal(map[string]string{"enrollment_token": token, "key_id": keyID, "public_key": pubB64})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func postEnroll(router *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/enroll", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestEnrollSuccess(t *testing.T) {
	store := &fakeEnrollmentStore{}
	router := newEnrollRouter(t, store)
	token, pub := testToken(t), testPublicKey(t)

	rec := postEnroll(router, enrollJSON(t, token, "acme-prod-1", base64.StdEncoding.EncodeToString(pub)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp models.EnrollIntegrationResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ClientID != "cli_test_client" || resp.KeyID != "acme-prod-1" || resp.Status != "pending" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	want := sha256.Sum256([]byte(token))
	if store.calls != 1 || !bytes.Equal(store.params.TokenHash, want[:]) ||
		!bytes.Equal(store.params.PublicKey, pub) || store.params.KeyID != "acme-prod-1" {
		t.Fatalf("store received wrong input: %+v (calls=%d)", store.params, store.calls)
	}
	if strings.Contains(rec.Body.String(), token) {
		t.Fatal("response must not contain the token")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("expected Cache-Control: no-store")
	}
}

func TestEnrollRejectsInvalidRequestsWithoutCallingStore(t *testing.T) {
	token := testToken(t)
	pubB64 := base64.StdEncoding.EncodeToString(testPublicKey(t))
	offCurve := make([]byte, 65)
	offCurve[0], offCurve[32], offCurve[64] = 0x04, 1, 1
	compressed := append([]byte{0x02}, make([]byte, 32)...)

	tests := []struct{ name, body string }{
		{"malformed JSON", `{"enrollment_token":`},
		{"empty body", ``},
		{"unknown field", `{"enrollment_token":"` + token + `","key_id":"k1","public_key":"` + pubB64 + `","extra":1}`},
		{"trailing JSON", enrollJSON(t, token, "k1", pubB64) + ` {"a":1}`},
		{"short token", enrollJSON(t, "short", "k1", pubB64)},
		{"bad token chars", enrollJSON(t, strings.Repeat("!", 43), "k1", pubB64)},
		{"empty key id", enrollJSON(t, token, "", pubB64)},
		{"key id with space", enrollJSON(t, token, "has space", pubB64)},
		{"key id with newline", enrollJSON(t, token, "new\nline", pubB64)},
		{"key id too long", enrollJSON(t, token, strings.Repeat("a", 129), pubB64)},
		{"not base64", enrollJSON(t, token, "k1", "not base64!!")},
		{"missing padding", enrollJSON(t, token, "k1", strings.TrimRight(pubB64, "="))},
		{"empty public key", enrollJSON(t, token, "k1", "")},
		{"too short key", enrollJSON(t, token, "k1", base64.StdEncoding.EncodeToString([]byte{4}))},
		{"compressed key", enrollJSON(t, token, "k1", base64.StdEncoding.EncodeToString(compressed))},
		{"off-curve key", enrollJSON(t, token, "k1", base64.StdEncoding.EncodeToString(offCurve))},
		{"body over limit", enrollJSON(t, token, strings.Repeat("a", enums.MaxEnrollmentBodyBytes+1), pubB64)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeEnrollmentStore{}
			rec := postEnroll(newEnrollRouter(t, store), tt.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
			}
			if store.calls != 0 {
				t.Fatalf("store must not be called, got %d", store.calls)
			}
		})
	}
}

func TestEnrollMapsStoreErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
	}{
		{"invalid token", errs.ErrEnrollmentTokenInvalid, http.StatusUnauthorized},
		{"wrapped invalid token", fmt.Errorf("outer: %w", errs.ErrEnrollmentTokenInvalid), http.StatusUnauthorized},
		{"not enrollable", errs.ErrIntegrationNotEnrollable, http.StatusUnauthorized},
		{"key conflict", errs.ErrKeyIDConflict, http.StatusConflict},
		{"store rejects key", errs.ErrInvalidPublicKey, http.StatusBadRequest},
		{"unexpected", errors.New("pq: connection refused to 10.0.0.5:5432"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeEnrollmentStore{err: tt.err}
			token := testToken(t)
			rec := postEnroll(newEnrollRouter(t, store),
				enrollJSON(t, token, "k1", base64.StdEncoding.EncodeToString(testPublicKey(t))))
			if rec.Code != tt.status {
				t.Fatalf("expected %d, got %d: %s", tt.status, rec.Code, rec.Body.String())
			}
			if store.calls != 1 {
				t.Fatalf("expected 1 store call, got %d", store.calls)
			}
			body := rec.Body.String()
			if strings.Contains(body, token) {
				t.Fatal("response must not contain the token")
			}
			if tt.status == http.StatusInternalServerError && (strings.Contains(body, "pq") || strings.Contains(body, "10.0.0.5")) {
				t.Fatalf("internal error leaked: %s", body)
			}
		})
	}
}

func TestEnrollUnauthorizedBodiesIndistinguishable(t *testing.T) {
	body := enrollJSON(t, testToken(t), "k1", base64.StdEncoding.EncodeToString(testPublicKey(t)))
	var bodies []string
	for _, e := range []error{errs.ErrEnrollmentTokenInvalid, errs.ErrIntegrationNotEnrollable} {
		rec := postEnroll(newEnrollRouter(t, &fakeEnrollmentStore{err: e}), body)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
		bodies = append(bodies, rec.Body.String())
	}
	if bodies[0] != bodies[1] {
		t.Fatalf("401 bodies differ:\n%s\n%s", bodies[0], bodies[1])
	}
}

func TestEnrollServiceNilSafety(t *testing.T) {
	svc, err := NewOnboardingService(&fakePendingIntegrationStore{}, &fakeEnrollmentStore{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Enroll(context.Background(), nil); !errors.Is(err, errs.ErrInvalidEnrollmentRequest) {
		t.Fatalf("expected ErrInvalidEnrollmentRequest for nil request, got %v", err)
	}
	if _, err := (&OnboardingService{}).Enroll(context.Background(), &models.EnrollIntegrationRequest{}); err == nil {
		t.Fatal("expected an error from an unconfigured service")
	}
}
