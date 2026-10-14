package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mtariq99/dispatchai/internal/enums"
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
		return fmt.Errorf("integration onboarding database is not configured")
	}

	if len(enrollmentTokenHash) != 32 {
		return fmt.Errorf("enrollment token hash must be 32 bytes")
	}

	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin integration onboarding transaction: %w", err)
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
		return fmt.Errorf("create pending integration: %w", err)
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
		return fmt.Errorf("persist integration enrollment token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit integration onboarding transaction: %w", err)
	}

	return nil
}

func (i *IntegrationOnboardingStore) GetHostIntegrationByClientID(ctx context.Context, clientID string) (*models.HostIntegration, error) {
	if clientID == "" {
		return nil, errors.New("client id is required")
	}
	HostIntegSlctQuery := `
	SELECT id,integration_key,display_name,status,meta_data,created_at,upated_at
	 FROM host_integrations
	 WHERE integration_key = $1
	`
	sqlRow := i.db.QueryRowContext(ctx, HostIntegSlctQuery, clientID)
	hostIntegration, err := ScanHostIntegration(sqlRow)
	if err != nil {
		return nil, err
	}
	return hostIntegration, nil
}

func (i *IntegrationOnboardingStore) UpateHostIntegrationStatus(ctx context.Context, clientID string, status string) error {
	if clientID == "" || status == "" {
		return errors.New("missing required fields")
	}
	tx, err := i.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	var currentStatus string
	StatusQuery := `
	SELECT status
	FROM host_integrations
	WHERE integration_key = $1
	FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, StatusQuery, clientID).Scan(currentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("integration not found: %s", clientID)
		}
		return fmt.Errorf("lock integration: %w", err)
	}
	if err := utils.ValidateStatusTransition(enums.Status(currentStatus), enums.Status(status)); err != nil {
		return err
	}
	upateStatusQuery := `
	UPDATE host_integrations
	SET status = $1,
	updated_at = NOW()
	WHERE integration_key = $2
	`
	_, err = tx.ExecContext(ctx, upateStatusQuery, status, clientID)
	if err != nil {
		return fmt.Errorf("update integration status: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit integration status update: %w", err)
	}
	return nil
}
