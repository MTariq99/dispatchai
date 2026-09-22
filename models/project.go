package models

import "encoding/json"

type ToolRequest struct {
	RequestID      string
	ConversationID string
	ToolName       string
	CallID         string
	Arguments      json.RawMessage
	Identity       Identity
	Tenant         TenantContext
	Authorization  AuthorizationContext
	Trace          TraceContext
}

type Identity struct {
	UserID string
}

type TenantContext struct {
	TenantID string
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
	ConversationID string

	Identity      Identity
	Tenant        TenantContext
	Authorization AuthorizationContext
	Trace         TraceContext
}
