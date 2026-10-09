package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type ToolRequest struct {
	RequestID      string
	RunID          uuid.UUID
	ConversationID uuid.UUID
	ToolName       string
	CallID         string
	Arguments      json.RawMessage
	Identity       Identity
	Tenant         TenantContext
	Authorization  AuthorizationContext
	Trace          TraceContext
}

type Identity struct {
	UserID uuid.UUID
}

type TenantContext struct {
	TenantID uuid.UUID
}
type AuthorizationContext struct {
	Roles       []string
	Permissions []string
}

type TraceContext struct {
	TraceID string
	SpanID  string
}

type ToolResponse struct {
	RequestID string
	CallID    string

	Success bool
	Data    json.RawMessage

	Error *ToolError

	Metadata ResponseMetadata
}

type ToolError struct {
	Code      string
	Message   string
	Retryable bool
}

type ResponseMetadata struct {
	ExecutionID string
	DurationMS  int64
}
type ExecutionContext struct {
	RequestID      string
	RunID          uuid.UUID
	ConversationID uuid.UUID
	Identity       Identity
	Tenant         TenantContext
	Authorization  AuthorizationContext
	Trace          TraceContext
}
