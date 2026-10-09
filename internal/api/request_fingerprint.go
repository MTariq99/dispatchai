package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/mtariq99/dispatchai/models"
)

type fingerprintInput struct {
	Message        string   `json:"message"`
	UserID         string   `json:"user_id"`
	TenantID       string   `json:"tenant_id"`
	ConversationID string   `json:"conversation_id"`
	Roles          []string `json:"roles"`
	Permissions    []string `json:"permissions"`
}

func requestFingerprint(req models.ChatRequest) (string, error) {
	roles := append([]string(nil), req.Roles...)
	permissions := append([]string(nil), req.Permissions...)

	// Authorization lists are treated as sets for fingerprinting.
	sort.Strings(roles)
	sort.Strings(permissions)

	input := fingerprintInput{
		Message:        req.Message,
		UserID:         req.UserID.String(),
		TenantID:       req.TenantID.String(),
		ConversationID: req.ConversationID.String(),
		Roles:          roles,
		Permissions:    permissions,
	}

	payload, err := json.Marshal(input)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}
