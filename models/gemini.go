package models

import "github.com/mtariq99/dispatchai/internal/enums"

type Request struct {
	Model       string
	Messages    []Message
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

// adapter structs
type GeminiRequest struct {
	SystemInstruction *GeminiContent `json:"systemInstruction,omitempty"`

	Contents []GeminiContent `json:"contents"`

	Tools []GeminiTool `json:"tools,omitempty"`

	GenerationConfig GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text,omitempty"`

	FunctionCall *GeminiFunctionCall `json:"functionCall,omitempty"`

	FunctionResponse *GeminiFunctionResponse `json:"functionResponse,omitempty"`
}

type GeminiFunctionCall struct {
	ID   string         `json:"id,omitempty"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type GeminiFunctionResponse struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type GeminiTool struct {
	FunctionDeclarations []GeminiFunctionDeclaration `json:"functionDeclarations"`
}

type GeminiFunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type GeminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type GeminiResponse struct {
	Candidates []*GeminiCandidate `json:"candidates"`

	UsageMetadata *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
}

type GeminiCandidate struct {
	Content *GeminiContent `json:"content,omitempty"`

	FinishReason string `json:"finishReason,omitempty"`
}

type GeminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type GeminiErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`

		Details []GeminiErrorDetail `json:"details"`
	} `json:"error"`
}

type GeminiErrorDetail struct {
	Type string `json:"@type"`

	RetryDelay string `json:"retryDelay"`

	QuotaMetric string `json:"quotaMetric"`
	QuotaID     string `json:"quotaId"`
}
