package models

import (
	"encoding/json"
	"time"
)

type Call struct {
	ID                string
	Name              string
	Arguments         json.RawMessage
	ToolID            string `json:"-"`
	DefinitionVersion int    `json:"-"`
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

type UpdateToolRequest struct {
	ExpectedVersion int        `json:"expected_version"`
	Definition      Definition `json:"definition"`
}

type ToggleToolRequest struct {
	ExpectedVersion int  `json:"expected_version"`
	Enabled         bool `json:"enabled"`
}

type DefinitionRecord struct {
	ID            string
	IntegrationID string
	Definition    Definition
	Version       int
	Enabled       bool
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
