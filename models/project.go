package models

import (
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

// ToolRequest is sent from DispatchAI to the Host Project's /tools/execute endpoint.
type ToolRequest struct {
	RequestID      string          `json:"request_id"`
	RunID          uuid.UUID       `json:"run_id"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	IntegrationID  uuid.UUID       `json:"integration_id"`
	ToolName       string          `json:"tool_name"`
	CallID         string          `json:"call_id"`
	Arguments      json.RawMessage `json:"arguments"`
	Trace          TraceContext    `json:"trace"`
}

// AuthorizationContext carries the permissions DispatchAI has verified
// for the integration (from its granted permissions), not claims from the model or user.
type AuthorizationContext struct {
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type TraceContext struct {
	TraceID string `json:"trace_id"`
	SpanID  string `json:"span_id"`
}

type ToolResponse struct {
	RequestID string           `json:"request_id"`
	CallID    string           `json:"call_id"`
	Success   bool             `json:"success"`
	Data      json.RawMessage  `json:"data,omitempty"`
	Error     *ToolError       `json:"error,omitempty"`
	Metadata  ResponseMetadata `json:"metadata"`
}

type ToolError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type ResponseMetadata struct {
	ExecutionID string `json:"execution_id"`
	DurationMS  int64  `json:"duration_ms"`
}

// ExecutionContext is internal to DispatchAI and is never sent over the wire as-is.
type ExecutionContext struct {
	RequestID      string
	RunID          uuid.UUID
	ConversationID uuid.UUID
	IntegrationID  uuid.UUID
	Authorization  AuthorizationContext
	Trace          TraceContext
}

// Validate checks the fields every tool execution depends on.
func (c *ExecutionContext) Validate() error {
	switch {
	case c == nil:
		return errors.New("execution context is required")
	case c.RequestID == "":
		return errors.New("request id is required")
	case c.RunID == uuid.Nil:
		return errors.New("run id is required")
	case c.ConversationID == uuid.Nil:
		return errors.New("conversation id is required")
	case c.IntegrationID == uuid.Nil:
		return errors.New("integration id is required")
	}
	return nil
}
