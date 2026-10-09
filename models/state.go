package models

import (
	"github.com/google/uuid"
)

type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusCompleted RequestStatus = "completed"
	RequestStatusFailed    RequestStatus = "failed"
)

type RequestState struct {
	Status         RequestStatus
	RequestID      uuid.UUID
	RunID          *uuid.UUID
	ConversationID uuid.UUID
	Response       *ChatResponse
	Fingerprint    string
}
