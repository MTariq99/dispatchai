package integration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mtariq99/dispatchai/internal/authn"
	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/internal/utils"
	"github.com/mtariq99/dispatchai/models"
)

type PendingIntegrationStore interface {
	CreatePendingIntegration(ctx context.Context, clientID string, displayName string, metadata []byte, enrollmentTokenHash []byte, enrollmentTokenExpiresAt time.Time) error
}

type EnrollmentStore interface {
	EnrollKey(ctx context.Context, params models.EnrollKeyParams) (*models.EnrolledKey, error)
}

type OnboardingService struct {
	store      PendingIntegrationStore
	enrollment EnrollmentStore
}

func NewOnboardingService(store PendingIntegrationStore, enrollment EnrollmentStore) (*OnboardingService, error) {
	if store == nil {
		return nil, errors.New("integration onboarding store is required")
	}
	if enrollment == nil {
		return nil, errors.New("integration enrollment store is required")
	}
	return &OnboardingService{store: store, enrollment: enrollment}, nil
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

	clientID, err := utils.GenerateClientID()
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
	expiresAt := time.Now().UTC().Add(enums.EnrollmentTokenTTL)

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

func (s *OnboardingService) Enroll(ctx context.Context, req *models.EnrollIntegrationRequest) (*models.EnrollIntegrationResponse, error) {
	if s == nil || s.enrollment == nil {
		return nil, errors.New("integration enrollment service is not configured")
	}
	if req == nil {
		return nil, errs.ErrInvalidEnrollmentRequest
	}
	if !enums.EnrollmentTokenPattern.MatchString(req.EnrollmentToken) {
		return nil, errs.ErrInvalidEnrollmentRequest
	}
	if err := utils.ValidateKeyID(req.KeyID); err != nil {
		return nil, err
	}
	publicKey, err := base64.StdEncoding.Strict().DecodeString(req.PublicKey)
	if err != nil {
		return nil, errs.ErrInvalidEnrollmentRequest
	}

	if _, err = authn.ParseP256PublicKey(publicKey); err != nil {
		return nil, err
	}

	tokenHash := sha256.Sum256([]byte(req.EnrollmentToken))
	enrolled, err := s.enrollment.EnrollKey(ctx, models.EnrollKeyParams{
		TokenHash: tokenHash[:],
		KeyID:     req.KeyID,
		PublicKey: publicKey,
	})
	if err != nil {
		return nil, err
	}

	return &models.EnrollIntegrationResponse{
		ClientID: enrolled.ClientID,
		KeyID:    enrolled.KeyID,
		Status:   "pending",
	}, nil
}
