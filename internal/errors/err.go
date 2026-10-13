package errs

import "errors"

var (
	ErrToolNotFound        = errors.New("tool not found")
	ErrVersionConflict     = errors.New("tool definition version conflict")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different request")
	ErrIntegrationInactive = errors.New("integration is not active")
	ErrToolUnavailable     = errors.New("tool is disabled or deleted")
)
