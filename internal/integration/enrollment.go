package integration

import (
	"context"
	"regexp"

	errs "github.com/mtariq99/dispatchai/internal/errors"
)

var keyIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

// EnrollKeyParams carries validated-at-the-boundary enrollment input.
// TokenHash is SHA-256 of the raw enrollment token; PublicKey is the raw
// 65-byte uncompressed SEC1 P-256 point.
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

type EnrollmentStore interface {
	EnrollKey(ctx context.Context, params EnrollKeyParams) (*EnrolledKey, error)
}

func ValidateKeyID(keyID string) error {
	if !keyIDPattern.MatchString(keyID) {
		return errs.ErrInvalidEnrollmentRequest
	}
	return nil
}
