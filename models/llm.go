package models

import "github.com/mtariq99/dispatchai/internal/enums"

type Request struct {
	Model       string
	Messages    []*Message
	Tools       []ToolDefinition
	Temperature float64
	MaxTokens   int
}

type Message struct {
	Role       enums.Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

type Response struct {
	Content    string
	ToolCalls  []ToolCall
	Usage      Usage
	StopReason string
}

type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type ToolCall struct {
	ID   string
	Name string
	Args map[string]any
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}
