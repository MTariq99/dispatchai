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

// EnrollIntegrationRequest is submitted by the host. PublicKey is standard
// Base64 (with padding) of the 65-byte uncompressed SEC1 P-256 point.
type EnrollIntegrationRequest struct {
	EnrollmentToken string `json:"enrollment_token"`
	KeyID           string `json:"key_id"`
	PublicKey       string `json:"public_key"`
}

type EnrollIntegrationResponse struct {
	ClientID string `json:"client_id"`
	KeyID    string `json:"key_id"`
	Status   string `json:"status"`
}

type EnrollKeyParams struct {
	TokenHash []byte
	KeyID     string
	PublicKey []byte
}

type EnrolledKey struct {
	IntegrationID string
	ClientID      string
	KeyID         string
}
