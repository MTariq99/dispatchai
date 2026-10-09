package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/internal/idempotency"
	"github.com/mtariq99/dispatchai/models"
)

type requestIdempotencyStub struct {
	state *models.RequestState
	found bool
}

func (s *requestIdempotencyStub) GetState(
	_ context.Context,
	_ uuid.UUID,
	_ string,
) (*models.RequestState, bool, error) {
	return s.state, s.found, nil
}

func (s *requestIdempotencyStub) Reserve(
	context.Context,
	uuid.UUID,
	string,
	uuid.UUID,
	uuid.UUID,
	string,
) (bool, error) {
	panic("Reserve must not be called for a fingerprint mismatch")
}

func (s *requestIdempotencyStub) Get(
	context.Context,
	uuid.UUID,
	string,
) (*models.ChatResponse, bool, error) {
	panic("Get must not be called")
}

func (s *requestIdempotencyStub) Complete(
	context.Context,
	uuid.UUID,
	string,
	uuid.UUID,
	uuid.UUID,
	*models.ChatResponse,
) error {
	panic("Complete must not be called")
}

func (s *requestIdempotencyStub) Fail(
	context.Context,
	uuid.UUID,
	string,
	uuid.UUID,
	uuid.UUID,
	*models.ChatResponse,
) error {
	panic("Fail must not be called")
}

func (s *requestIdempotencyStub) FailBeforeRun(
	context.Context,
	uuid.UUID,
	string,
	uuid.UUID,
	*models.ChatResponse,
) error {
	panic("FailBeforeRun must not be called")
}

var _ idempotency.RequestIdempotency = (*requestIdempotencyStub)(nil)

func TestChatRejectsIdempotencyKeyWithDifferentFingerprint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tenantID := uuid.New()
	userID := uuid.New()

	store := &requestIdempotencyStub{
		found: true,
		state: &models.RequestState{
			Status:      models.RequestStatusCompleted,
			RequestID:   uuid.New(),
			Fingerprint: "fingerprint-original",
			Response: &models.ChatResponse{
				Answer: "This is the original response",
				Status: string(models.RequestStatusCompleted),
			},
		},
	}

	handler := &Handler{
		requestIdempotency: store,
	}

	body := `{
		"user_id": "` + userID.String() + `",
		"tenant_id": "` + tenantID.String() + `",
		"message": "Cancel my order"
	}`

	req := httptest.NewRequest(http.MethodPost, "/chat", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "chat-123")

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = req

	handler.Chat(ctx)

	if recorder.Code != http.StatusConflict {
		t.Fatalf(
			"status code: want %d, got %d; body: %s",
			http.StatusConflict,
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if strings.Contains(recorder.Body.String(), "This is the original response") {
		t.Fatal("handler returned the response from a different request")
	}

	if !strings.Contains(
		recorder.Body.String(),
		"Idempotency-Key was already used for a different or unverifiable request",
	) {
		t.Fatalf("unexpected response body: %s", recorder.Body.String())
	}
}
