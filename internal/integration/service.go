package integration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

const enrollmentTokenTTL = 15 * time.Minute

type PendingIntegrationStore interface {
	CreatePendingIntegration(ctx context.Context, clientID string, displayName string, metadata []byte, enrollmentTokenHash []byte, enrollmentTokenExpiresAt time.Time) error
}

type OnboardingService struct {
	store PendingIntegrationStore
}

func NewOnboardingService(store PendingIntegrationStore) (*OnboardingService, error) {
	if store == nil {
		return nil, errors.New("integration onboarding store is required")
	}

	return &OnboardingService{store: store}, nil
}

func (s *OnboardingService) CreatePending(ctx context.Context, request models.CreateIntegrationRequest) (models.CreateIntegrationResponse, error) {
	if s == nil || s.store == nil {
		return models.CreateIntegrationResponse{},
			errors.New("integration onboarding service is not configured")
	}

	displayName := strings.TrimSpace(request.DisplayName)
	if displayName == "" || !utf8.ValidString(displayName) || len(displayName) > 200 {
		return models.CreateIntegrationResponse{}, errs.ErrInvalidOnboardingRequest
	}

	metadata := request.Metadata
	if metadata == nil {
		metadata = map[string]json.RawMessage{}
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return models.CreateIntegrationResponse{}, errs.ErrInvalidOnboardingRequest
	}

	clientID, err := generateClientID()
	if err != nil {
		return models.CreateIntegrationResponse{},
			fmt.Errorf("generate integration client ID: %w", err)
	}

	rawToken := make([]byte, 32)
	if _, err := rand.Read(rawToken); err != nil {
		return models.CreateIntegrationResponse{},
			fmt.Errorf("generate enrollment token: %w", err)
	}

	enrollmentToken := base64.RawURLEncoding.EncodeToString(rawToken)
	tokenHash := sha256.Sum256([]byte(enrollmentToken))
	expiresAt := time.Now().UTC().Add(enrollmentTokenTTL)

	if err := s.store.CreatePendingIntegration(ctx, clientID, displayName, metadataJSON, tokenHash[:], expiresAt); err != nil {
		return models.CreateIntegrationResponse{},
			fmt.Errorf("persist pending integration and enrollment token: %w", err)
	}

	return models.CreateIntegrationResponse{
		ClientID:        clientID,
		Status:          "pending",
		EnrollmentToken: enrollmentToken,
		ExpiresAt:       expiresAt,
	}, nil
}

func generateClientID() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	return "cli_" + hex.EncodeToString(randomBytes), nil
}
