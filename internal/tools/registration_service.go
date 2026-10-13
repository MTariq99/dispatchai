package tools

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/mtariq99/dispatchai/models"
)

var toolNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`)

type RegistrationService struct {
	repo Repository
}

func NewRegistrationService(repo Repository) (*RegistrationService, error) {
	if repo == nil {
		return nil, errors.New("tool repository cannot be nil")
	}

	return &RegistrationService{
		repo: repo,
	}, nil
}

func (rs *RegistrationService) CreateTool(ctx context.Context, integrationID string, idempotencyKey string, definition *models.Definition) (*DefinitionRecord, bool, error) {
	if err := validateRegistrationInput(integrationID, definition); err != nil {
		return nil, false, err
	}
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey == "" {
		return nil, false, errors.New("Idempotency-Key header is required")
	}
	if len(idempotencyKey) > 255 {
		return nil, false, errors.New("Idempotency-Key must not exceed 255 characters")
	}
	payload, err := json.Marshal(definition)
	if err != nil {
		return nil, false, fmt.Errorf("encode tool definition: %w", err)
	}
	hash := sha256.Sum256(payload)

	record, replayed, err := rs.repo.CreateToolRegistry(ctx, integrationID, definition, idempotencyKey, hash[:])
	if err != nil {
		return nil, false, fmt.Errorf("register tool: %w", err)
	}
	return record, replayed, nil
}

func validateRegistrationInput(integrationID string, definition *models.Definition) error {
	if strings.TrimSpace(integrationID) == "" {
		return errors.New("integration identity is required")
	}
	return validateDefinition(definition)
}

func validateDefinition(definition *models.Definition) error {
	name := strings.TrimSpace(definition.Name)
	if !toolNamePattern.MatchString(name) {
		return errors.New(
			"tool name must start with a letter and contain only letters, digits, underscores, or hyphens (maximum 64 characters)",
		)
	}
	if strings.TrimSpace(definition.Description) == "" {
		return errors.New("tool description is required")
	}
	if definition.Parameters == nil {
		return errors.New("tool input schema is required")
	}

	schemaType, ok := definition.Parameters["type"].(string)
	if !ok || schemaType != "object" {
		return errors.New("tool input schema must have type object")
	}

	if _, err := json.Marshal(definition.Parameters); err != nil {
		return fmt.Errorf("invalid tool input schema: %w", err)
	}

	return nil
}

func (s *RegistrationService) Get(ctx context.Context, integrationID string, toolID string) (*DefinitionRecord, error) {
	if strings.TrimSpace(integrationID) == "" {
		return nil, errors.New("integration identity is required")
	}
	if strings.TrimSpace(toolID) == "" {
		return nil, errors.New("tool ID is required")
	}

	record, err := s.repo.GetTool(ctx, integrationID, toolID)
	if err != nil {
		return nil, fmt.Errorf("get tool: %w", err)
	}
	return record, nil
}

func (s *RegistrationService) Update(ctx context.Context, integrationID string, toolID string, expectedVersion int, definition *models.Definition) (*DefinitionRecord, error) {
	if strings.TrimSpace(integrationID) == "" {
		return nil, errors.New("integration identity is required")
	}
	if strings.TrimSpace(toolID) == "" {
		return nil, errors.New("tool ID is required")
	}
	if expectedVersion < 1 {
		return nil, errors.New("expected version must be positive")
	}
	if err := validateDefinition(definition); err != nil {
		return nil, err
	}

	record, err := s.repo.Update(ctx, integrationID, toolID, expectedVersion, definition)
	if err != nil {
		return nil, fmt.Errorf("update tool: %w", err)
	}
	return record, nil
}

func (s *RegistrationService) SetEnabled(ctx context.Context, integrationID string, toolID string, expectedVersion int, enabled bool) (*DefinitionRecord, error) {
	if strings.TrimSpace(integrationID) == "" {
		return nil, errors.New("integration identity is required")
	}
	if strings.TrimSpace(toolID) == "" {
		return nil, errors.New("tool ID is required")
	}
	if expectedVersion < 1 {
		return nil, errors.New("expected version must be positive")
	}

	record, err := s.repo.SetEnabled(
		ctx, integrationID, toolID, expectedVersion, enabled,
	)
	if err != nil {
		return nil, fmt.Errorf("change tool status: %w", err)
	}
	return record, nil
}

func (s *RegistrationService) Delete(ctx context.Context, integrationID string, toolID string, expectedVersion int) error {
	if strings.TrimSpace(integrationID) == "" {
		return errors.New("integration identity is required")
	}
	if strings.TrimSpace(toolID) == "" {
		return errors.New("tool ID is required")
	}
	if expectedVersion < 1 {
		return errors.New("expected version must be positive")
	}

	if err := s.repo.SoftDelete(ctx, integrationID, toolID, expectedVersion); err != nil {
		return fmt.Errorf("delete tool: %w", err)
	}
	return nil
}
