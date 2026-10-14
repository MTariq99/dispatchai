package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/mtariq99/dispatchai/internal/authn"
	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/internal/integration"
	"github.com/mtariq99/dispatchai/models"
)

const (
	pgUniqueViolation  = "23505"
	integrationKeyUniq = "uq_integration_keys_integration_key"
)

type IntegrationEnrollmentStore struct {
	db *sql.DB
}

var _ integration.EnrollmentStore = (*IntegrationEnrollmentStore)(nil)

func NewIntegrationEnrollmentStore(db *sql.DB) *IntegrationEnrollmentStore {
	return &IntegrationEnrollmentStore{db: db}
}

func (s *IntegrationEnrollmentStore) EnrollKey(ctx context.Context, p models.EnrollKeyParams) (*models.EnrolledKey, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("integration enrollment database is not configured")
	}
	if len(p.TokenHash) != sha256.Size {
		return nil, errs.ErrInvalidEnrollmentRequest
	}
	if err := integration.ValidateKeyID(p.KeyID); err != nil {
		return nil, err
	}
	if _, err := authn.ParseP256PublicKey(p.PublicKey); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin integration enrollment transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var integrationID string
	UpdateEnrollmentQuery := `
		UPDATE integration_enrollment_tokens
		SET consumed_at = NOW()
		WHERE token_hash = $1
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		RETURNING integration_id::text
	`
	err = tx.QueryRowContext(ctx, UpdateEnrollmentQuery, p.TokenHash).Scan(&integrationID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.ErrEnrollmentTokenInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("consume enrollment token: %w", err)
	}

	var clientID, status string
	selectIntegQuery := `
		SELECT integration_key, status
		FROM host_integrations
		WHERE id = $1::uuid
		FOR UPDATE
	`
	err = tx.QueryRowContext(ctx, selectIntegQuery, integrationID).Scan(&clientID, &status)
	if err != nil {
		return nil, fmt.Errorf("lock integration for enrollment: %w", err)
	}
	if status != "pending" {
		return nil, errs.ErrIntegrationNotEnrollable
	}
	insertIntegQuery := `
		INSERT INTO integration_keys (integration_id, key_id, algorithm, public_key)
		VALUES ($1::uuid, $2, $3, $4)
	`
	_, err = tx.ExecContext(ctx, insertIntegQuery, integrationID, p.KeyID, enums.SignatureAlgorithm, p.PublicKey)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pgUniqueViolation && pqErr.Constraint == integrationKeyUniq {
			return nil, errs.ErrKeyIDConflict
		}
		return nil, fmt.Errorf("insert integration key: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit integration enrollment transaction: %w", err)
	}

	return &models.EnrolledKey{
		IntegrationID: integrationID,
		ClientID:      clientID,
		KeyID:         p.KeyID,
	}, nil
}
