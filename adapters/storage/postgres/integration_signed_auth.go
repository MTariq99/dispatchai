package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/mtariq99/dispatchai/internal/authn"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

type SignedAuthStore struct {
	db *sql.DB
}

var (
	_ authn.KeyLookup  = (*SignedAuthStore)(nil)
	_ authn.NonceStore = (*SignedAuthStore)(nil)
)

func NewSignedAuthStore(db *sql.DB) *SignedAuthStore {
	return &SignedAuthStore{db: db}
}

func (s *SignedAuthStore) LookupActiveKey(ctx context.Context, clientID, keyID string) (*models.KeyRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("signed auth database is not configured")
	}
	const q = `
		SELECT hi.id::text, ik.public_key
		FROM host_integrations hi
		JOIN integration_keys ik ON ik.integration_id = hi.id
		WHERE hi.integration_key = $1
		  AND ik.key_id = $2
		  AND hi.status = 'active'
		  AND ik.status = 'active'
		  AND ik.revoked_at IS NULL
		  AND ik.algorithm = 'ECDSA-P256-SHA256'
		  AND (ik.expires_at IS NULL OR ik.expires_at > NOW())`

	var rec models.KeyRecord
	err := s.db.QueryRowContext(ctx, q, clientID, keyID).Scan(&rec.IntegrationID, &rec.PublicKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.ErrSigKeyNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lookup signing key: %w", err)
	}
	return &rec, nil
}

func (s *SignedAuthStore) RecordNonce(ctx context.Context, integrationID, nonce string, expiresAt time.Time) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("signed auth database is not configured")
	}
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO integration_request_nonces (integration_id, nonce, expires_at)
		VALUES ($1::uuid, $2, $3)
		ON CONFLICT (integration_id, nonce) DO NOTHING`, integrationID, nonce, expiresAt)
	if err != nil {
		return false, fmt.Errorf("record nonce: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("record nonce: %w", err)
	}
	return n == 1, nil
}

func (s *SignedAuthStore) PurgeExpiredNonces(ctx context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("signed auth database is not configured")
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM integration_request_nonces WHERE expires_at < NOW()`)
	if err != nil {
		return 0, fmt.Errorf("purge nonces: %w", err)
	}
	return res.RowsAffected()
}

func (s *SignedAuthStore) RunNonceCleanup(ctx context.Context, interval time.Duration, logger *zap.Logger) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n, err := s.PurgeExpiredNonces(ctx); err != nil {
				if logger != nil {
					logger.Warn("nonce cleanup failed", zap.Error(err))
				}
			} else if n > 0 && logger != nil {
				logger.Debug("expired nonces purged", zap.Int64("count", n))
			}
		}
	}
}
