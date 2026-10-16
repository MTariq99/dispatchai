package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

	"github.com/mtariq99/dispatchai/models"
)

func RequestFingerprint(req models.ChatRequest) (string, error) {
	roles := append([]string(nil), req.Claims.Roles...)
	permissions := append([]string(nil), req.Claims.Permissions...)

	sort.Strings(roles)
	sort.Strings(permissions)

	input := models.FingerprintInput{
		Message:        req.Message,
		UserID:         req.EndUser.UserID.String(),
		TenantID:       req.EndUser.TenantID.String(),
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
