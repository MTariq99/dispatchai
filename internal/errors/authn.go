package errs

import (
	"errors"
)

var (
	ErrInvalidRequest   = errors.New("invalid signed request")
	ErrInvalidBodyHash  = errors.New("invalid content SHA-256")
	ErrInvalidSignature = errors.New("invalid request signature")
	ErrInvalidPublicKey = errors.New("invalid ECDSA P-256 public key")
	ErrSigKeyNotFound   = errors.New("signing key not found or inactive")
	ErrReplayedRequest  = errors.New("request replayed")
	ErrStaleTimestamp   = errors.New("request timestamp outside allowed window")
	ErrAuthUnavailable  = errors.New("authentication backend unavailable")
)
