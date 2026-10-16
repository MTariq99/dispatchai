package models

import "github.com/google/uuid"

type EndUser struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
}

type Claims struct {
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type ChatRequest struct {
	Message        string    `json:"message"`
	ConversationID uuid.UUID `json:"conversation_id"`
	EndUser        EndUser   `json:"end_user"`
	Claims         Claims    `json:"claims"`
}

type ChatResponse struct {
	RequestID      string    `json:"request_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	RunID          uuid.UUID `json:"run_id"`
	Status         string    `json:"status"`
	Answer         string    `json:"answer,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
