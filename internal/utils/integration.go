package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
)

func ValidateKeyID(keyID string) error {
	if !enums.KeyIDPattern.MatchString(keyID) {
		return errs.ErrInvalidEnrollmentRequest
	}
	return nil
}

func ValidateStatusTransition(from, to enums.Status) error {
	switch from {
	case enums.StatusPending:
		if to == enums.StatusActive {
			return nil
		}
	case enums.StatusActive:
		if to == enums.StatusPending {
			return nil
		}
	case enums.StatusSuspended:
		if to == enums.StatusActive {
			return nil
		}
	}
	return fmt.Errorf("invalid integration status transition: %q -> %q", from, to)
}

func GenerateClientID() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return "cli_" + hex.EncodeToString(randomBytes), nil
}
