package errs

import (
	"errors"
)

var (
	ErrInvalidRequest   = errors.New("invalid signed request")
	ErrInvalidBodyHash  = errors.New("invalid content SHA-256")
	ErrInvalidSignature = errors.New("invalid request signature")
	ErrInvalidPublicKey = errors.New("invalid ECDSA P-256 public key")
)
