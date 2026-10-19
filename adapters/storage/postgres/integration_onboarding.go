package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/internal/utils"
	"github.com/mtariq99/dispatchai/models"
)

type IntegrationOnboardingStore struct {
	db *sql.DB
}

func NewIntegrationOnboardingStore(db *sql.DB) *IntegrationOnboardingStore {
	return &IntegrationOnboardingStore{db: db}
}

func ScanHostIntegration(row interface{ Scan(...any) error }) (*models.HostIntegration, error) {
	var record models.HostIntegration
	err := row.Scan(
		&record.ID,
		&record.IntegrationKey,
		&record.DisplayName,
		&record.Status,
		&record.MetaData,
		&record.CreatedAt,
		&record.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (i *IntegrationOnboardingStore) CreatePendingIntegration(ctx context.Context, clientID string, displayName string, metadata []byte, enrollmentTokenHash []byte, enrollmentTokenExpiresAt time.Time) error {
	if i == nil || i.db == nil {
		return errs.ErrDatabaseNotConfigured
	}

	if len(enrollmentTokenHash) != 32 {
		return errs.ErrEnrollmentTokenInvalid
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return errs.ErrBeginOnboardingTransaction
	}
	defer func() {
		_ = tx.Rollback()
	}()

	const insertIntegration = `
		INSERT INTO host_integrations (
			integration_key,
			display_name,
			status,
			metadata
		)
		VALUES ($1, $2, 'pending', $3::jsonb)
		RETURNING id
	`

	var integrationID string
	err = tx.QueryRowContext(
		ctx,
		insertIntegration,
		clientID,
		displayName,
		metadata,
	).Scan(&integrationID)
	if err != nil {
		return errs.ErrCreatePendingIntegration
	}

	const insertEnrollmentToken = `
		INSERT INTO integration_enrollment_tokens (
			integration_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
	`

	_, err = tx.ExecContext(
		ctx,
		insertEnrollmentToken,
		integrationID,
		enrollmentTokenHash,
		enrollmentTokenExpiresAt,
	)
	if err != nil {
		return errs.Wrapf(errs.ErrPersistEnrollmentToken, err)
	}

	if err := tx.Commit(); err != nil {
		return errs.Wrapf(errs.ErrCommitOnboardingTransaction, err)
	}

	return nil
}

func (i *IntegrationOnboardingStore) GetHostIntegrationByClientID(ctx context.Context, clientID string) (*models.HostIntegration, error) {
	if i == nil || i.db == nil {
		return nil, errs.ErrDatabaseNotConfigured
	}
	const q = `
		SELECT id, integration_key, display_name, status, metadata, created_at, updated_at
		FROM host_integrations
		WHERE integration_key = $1`
	rec, err := ScanHostIntegration(i.db.QueryRowContext(ctx, q, clientID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.ErrIntegrationNotFound
	}
	return rec, err
}

func (i *IntegrationOnboardingStore) UpateHostIntegrationStatus(ctx context.Context, clientID string, status string) error {
	if i == nil || i.db == nil {
		return errs.ErrDatabaseNotConfigured
	}
	if clientID == "" || status == "" {
		return errs.ErrInvalidOnboardingRequest
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return errs.ErrBeginOnboardingTransaction
	}
	defer func() { _ = tx.Rollback() }()

	var integrationID, current string
	err = tx.QueryRowContext(ctx, `
		SELECT id::text, status FROM host_integrations
		WHERE integration_key = $1 FOR UPDATE`, clientID).Scan(&integrationID, &current)
	if errors.Is(err, sql.ErrNoRows) {
		return errs.ErrIntegrationNotFound
	}
	if err != nil {
		return errs.Wrapf(errs.ErrLockIntegration, err)
	}

	if err := utils.ValidateStatusTransition(enums.Status(current), enums.Status(status)); err != nil {
		return errs.Wrapf(errs.ErrInvalidStatusTransition, err)
	}

	if status == "active" {
		var usable bool
		if err := tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM integration_keys
				WHERE integration_id = $1::uuid
				  AND status = 'active' AND revoked_at IS NULL
				  AND (expires_at IS NULL OR expires_at > NOW()))`, integrationID).Scan(&usable); err != nil {
			return fmt.Errorf("check signing keys: %w", err)
		}
		if !usable {
			return errs.ErrNoActiveKey
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE host_integrations SET status = $1, updated_at = NOW()
		WHERE id = $2::uuid`, status, integrationID); err != nil {
		return fmt.Errorf("update integration status: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return errs.Wrapf(errs.ErrCommitOnboardingTransaction, err)
	}
	return nil
}

func (i *IntegrationOnboardingStore) GetHostPermissions(ctx context.Context) ([]*models.HostPermissions, error) {
	selectQuery := `
		SELECT permission, description
		FROM authorization_permissions
	`

	rows, err := i.db.QueryContext(ctx, selectQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions := make([]*models.HostPermissions, 0)

	for rows.Next() {
		var permission models.HostPermissions

		if err := rows.Scan(
			&permission.Permission,
			&permission.Description,
		); err != nil {
			return nil, err
		}

		permissions = append(permissions, &permission)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}

func (i *IntegrationOnboardingStore) GrantPermissions(ctx context.Context, clientID string, integrationID uuid.UUID, permissions []string, grantedBy string) (*models.GrantPermissions, error) {
	if clientID == "" {
		return nil, errs.ErrClientIDRequired
	}
	if integrationID == uuid.Nil {
		return nil, errs.ErrIntegrationIDRequired
	}
	if grantedBy == "" {
		return nil, errs.ErrGrantedByRequired
	}
	if len(permissions) == 0 {
		return nil, errs.ErrPermissionRequired
	}

	uniquePermissions := make([]string, 0, len(permissions))
	seen := make(map[string]struct{}, len(permissions))

	for _, permission := range permissions {
		if permission == "" {
			return nil, errs.ErrPermissionEmpty
		}

		if _, exists := seen[permission]; exists {
			continue
		}

		seen[permission] = struct{}{}
		uniquePermissions = append(uniquePermissions, permission)
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrBeginGrantPermissions, err)
	}
	defer tx.Rollback()

	var exists bool
	SelectHostIntegrationQuery := `
        SELECT EXISTS (
            SELECT 1
            FROM host_integrations
            WHERE id = $1
              AND integration_key = $2
        )
    `
	err = tx.QueryRowContext(ctx, SelectHostIntegrationQuery, integrationID, clientID).Scan(&exists)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrCheckIntegration, err)
	}
	if !exists {
		return nil, errs.ErrIntegrationNotFound
	}

	const insertQuery = `
        INSERT INTO integration_permission_grants (
            integration_id,
            permission,
            granted_by
        )
        VALUES ($1, $2, $3)
        ON CONFLICT (integration_id, permission) DO NOTHING
    `

	for _, permission := range uniquePermissions {
		if _, err := tx.ExecContext(ctx, insertQuery, integrationID, permission, grantedBy); err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23503" {
				return nil, errs.ErrUnknownPermission
			}
			return nil, errs.Wrapf(errs.ErrGrantPermission, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errs.Wrapf(errs.ErrCommitPermissionGrants, err)
	}

	return &models.GrantPermissions{
		IntegrationID: integrationID,
		Permissions:   uniquePermissions,
	}, nil
}

func (i *IntegrationOnboardingStore) GetGrantedPermissions(ctx context.Context, integrationID uuid.UUID) ([]string, error) {
	if i == nil || i.db == nil {
		return nil, errs.ErrDatabaseNotConfigured
	}

	var exists bool
	if err := i.db.QueryRowContext(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM host_integrations WHERE id = $1)`, integrationID,
	).Scan(&exists); err != nil {
		return nil, errs.Wrapf(errs.ErrCheckIntegration, err)
	}
	if !exists {
		return nil, errs.ErrIntegrationNotFound
	}

	rows, err := i.db.QueryContext(ctx, `
		SELECT permission
		FROM integration_permission_grants
		WHERE integration_id = $1
		ORDER BY permission`, integrationID)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrListGrantedPermissions, err)
	}
	defer rows.Close()

	perms := []string{}
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, errs.Wrapf(errs.ErrScanGrantedPermission, err)
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.Wrapf(errs.ErrListGrantedPermissions, err)
	}

	return perms, nil
}

func (i *IntegrationOnboardingStore) HasAccess(ctx context.Context, integrationID, permission string) (bool, error) {
	if i == nil || i.db == nil {
		return false, errs.ErrDatabaseNotConfigured
	}

	if integrationID == "" || permission == "" {
		return false, errs.ErrFieldsRequired
	}
	query := `
	SELECT EXISTS(
	SELECT 1
	FROM integration_permission_grants
	WHERE integration_id = $1 AND permission = $2
	)
	`
	var granted bool
	if err := i.db.QueryRowContext(
		ctx, query, integrationID, permission,
	).Scan(&granted); err != nil {
		return false, errs.Wrapf(errs.ErrCheckIntegrationPermission, err)
	}

	return granted, nil
}
