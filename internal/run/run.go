package run

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidTenantID     = errors.New("tenant id is required")
	ErrInvalidUserID       = errors.New("user id is required")
	ErrInvalidConversation = errors.New("conversation id is required")
	ErrInvalidRequestID    = errors.New("request id is required")
	ErrInvalidTransition   = errors.New("invalid state transition")
)

type Run struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	UserID         uuid.UUID
	ConversationID uuid.UUID
	RequestID      uuid.UUID
	Status         State
	ModelProvider  string
	ModelName      string
	StartedAt      time.Time
	CompletedAt    *time.Time
	InputTokens    int64
	OutputTokens   int64
	TotalTokens    int64
	ErrorCode      string
	ErrorMessage   string
	Answer         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewRun(
	tenantID uuid.UUID,
	userID uuid.UUID,
	conversationID uuid.UUID,
	requestID uuid.UUID,
	now time.Time,
) (*Run, error) {
	if tenantID == uuid.Nil {
		return nil, ErrInvalidTenantID
	}

	if userID == uuid.Nil {
		return nil, ErrInvalidUserID
	}

	if conversationID == uuid.Nil {
		return nil, ErrInvalidConversation
	}

	if requestID == uuid.Nil {
		return nil, ErrInvalidRequestID
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	return &Run{
		ID:             uuid.New(),
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
		RequestID:      requestID,
		Status:         StateCreated,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (r *Run) TransitionTo(next State, now time.Time) error {
	if r.Status == next {
		return nil
	}

	if !r.Status.CanTransitionTo(next) {
		return ErrInvalidTransition
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	r.Status = next
	r.UpdatedAt = now

	if next == StateRunning && r.StartedAt.IsZero() {
		r.StartedAt = now
	}

	if next.IsTerminal() && r.CompletedAt == nil {
		completedAt := now
		r.CompletedAt = &completedAt
	}

	return nil
}
