package models

import (
	"time"

	"github.com/google/uuid"
)

type CreateConversationReq struct {
	Name string `json:"name"`
}

type Conversation struct {
	ID            uuid.UUID `json:"id"`
	IntegrationID uuid.UUID `json:"integration_id"`
	Status        string    `json:"status"`
	Title         string    `json:"name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	ArchivedAt    time.Time `json:"archived_at"`
}

type FingerprintInput struct {
	Message        string   `json:"message"`
	UserID         string   `json:"user_id"`
	TenantID       string   `json:"tenant_id"`
	ConversationID string   `json:"conversation_id"`
	Roles          []string `json:"roles"`
	Permissions    []string `json:"permissions"`
}
