package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type IntegrationOnboardingStore struct {
	db *sql.DB
}

func NewIntegrationOnboardingStore(db *sql.DB) *IntegrationOnboardingStore {
	return &IntegrationOnboardingStore{db: db}
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
