package models

import (
	"encoding/json"
	"time"
)

type CreateIntegrationRequest struct {
	DisplayName string                     `json:"display_name"`
	Metadata    map[string]json.RawMessage `json:"metadata"`
}

type CreateIntegrationResponse struct {
	ClientID        string    `json:"client_id"`
	Status          string    `json:"status"`
	EnrollmentToken string    `json:"enrollment_token"`
	ExpiresAt       time.Time `json:"expires_at"`
}
