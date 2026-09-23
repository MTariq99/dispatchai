package models

type ChatRequest struct {
	Message        string   `json:"message" binding:"required"`
	ConversationID string   `json:"conversation_id"`
	UserID         string   `json:"user_id" binding:"required"`
	TenantID       string   `json:"tenant_id" binding:"required"`
	Roles          []string `json:"roles"`
	Permissions    []string `json:"permissions"`
}
type ChatResponse struct {
	RequestID string `json:"request_id"`
	Answer    string `json:"answer"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
