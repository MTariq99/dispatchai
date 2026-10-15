package errs

import "errors"

var (
	ErrInvalidOnboardingRequest = errors.New("invalid integration onboarding request")
	ErrInvalidEnrollmentRequest = errors.New("invalid integration enrollment request")
	ErrEnrollmentTokenInvalid   = errors.New("enrollment token is invalid, expired, or already used")
	ErrIntegrationNotEnrollable = errors.New("integration is not eligible for key enrollment")
	ErrKeyIDConflict            = errors.New("key ID already exists for this integration")
	ErrIntegrationNotFound      = errors.New("integration not found")
	ErrInvalidStatusTransition  = errors.New("invalid integration status transition")
	ErrNoActiveKey              = errors.New("integration has no active signing key")
	ErrUnknownPermission        = errors.New("unknown permission")
)
