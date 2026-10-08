package models

import "github.com/google/uuid"

type ChatRequest struct {
	Message        string    `json:"message" binding:"required"`
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id" binding:"required"`
	TenantID       uuid.UUID `json:"tenant_id" binding:"required"`
	Roles          []string  `json:"roles"`
	Permissions    []string  `json:"permissions"`
}
type ChatResponse struct {
	RequestID      string     `json:"request_id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	RunID          uuid.UUID  `json:"run_id"`
	Status         string     `json:"status"`
	Answer         string     `json:"answer,omitempty"`
	ToolCalls      []ToolCall `json:"tool_calls,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
