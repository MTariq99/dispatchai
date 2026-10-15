package integration

import (
	"context"
	"errors"

	"github.com/google/uuid"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

type IntegrationAdminStore interface {
	GetHostIntegrationByClientID(ctx context.Context, clientID string) (*models.HostIntegration, error)
	UpateHostIntegrationStatus(ctx context.Context, clientID string, status string) error
	GetHostPermissions(ctx context.Context) ([]*models.HostPermissions, error)
	GetGrantedPermissions(ctx context.Context, clientID string) ([]string, error)
	GrantPermissions(ctx context.Context, clientID string, integrationID uuid.UUID, permissions []string, grantedBy string) (*models.GrantPermissions, error)
}

type AdminService struct {
	store IntegrationAdminStore
}

func NewAdminService(store IntegrationAdminStore) (*AdminService, error) {
	if store == nil {
		return nil, errors.New("integration admin store is required")
	}
	return &AdminService{store: store}, nil
}

// SetStatus only accepts active or suspended. The store validates the
// transition and the signing-key requirement under a row lock.
func (s *AdminService) SetStatus(ctx context.Context, clientID string, req *models.ToggleIntegrationReq) error {
	if s == nil || s.store == nil {
		return errors.New("integration admin service is not configured")
	}
	if clientID == "" || req == nil || (req.Status != "active" && req.Status != "suspended") {
		return errs.ErrInvalidOnboardingRequest
	}
	return s.store.UpateHostIntegrationStatus(ctx, clientID, req.Status)
}

func (s *AdminService) PermissionCatalog(ctx context.Context) ([]*models.HostPermissions, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("integration admin service is not configured")
	}
	return s.store.GetHostPermissions(ctx)
}

func (s *AdminService) GrantedPermissions(ctx context.Context, clientID string) ([]string, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("integration admin service is not configured")
	}
	if clientID == "" {
		return nil, errs.ErrInvalidOnboardingRequest
	}
	return s.store.GetGrantedPermissions(ctx, clientID)
}

func (s *AdminService) GrantPermissions(ctx context.Context, clientID, grantedBy string, req *models.GrantPermissionsReq) (*models.GrantPermissions, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("integration admin service is not configured")
	}
	if clientID == "" || grantedBy == "" || req == nil || len(req.Permissions) == 0 {
		return nil, errs.ErrInvalidOnboardingRequest
	}
	integ, err := s.store.GetHostIntegrationByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}
	return s.store.GrantPermissions(ctx, clientID, integ.ID, req.Permissions, grantedBy)
}
