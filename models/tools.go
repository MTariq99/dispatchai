package models

import "encoding/json"

type Call struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

type Definition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type Result struct {
	CallID string

	Success bool

	Data json.RawMessage

	Error *Error
}

// Error represents a structured tool execution failure.
//
// This is intentionally different from Go's error interface because the
// result may need to be serialized and returned to the LLM.
type Error struct {
	Code    string
	Message string

	Retryable bool
}
