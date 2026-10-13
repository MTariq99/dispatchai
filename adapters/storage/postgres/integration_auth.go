package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"strings"
)

var ErrInvalidIntegrationToken = errors.New("invalid integration token")

type IntegrationAuthenticator struct {
	db *sql.DB
}

func NewIntegrationAuthenticator(db *sql.DB) *IntegrationAuthenticator {
	return &IntegrationAuthenticator{db: db}
}

func (a *IntegrationAuthenticator) AuthenticateIntegration(ctx context.Context, token string) (string, error) {
	if a == nil || a.db == nil || strings.TrimSpace(token) == "" {
		return "", ErrInvalidIntegrationToken
	}

	tokenHash := sha256.Sum256([]byte(token))

	var integrationID string
	err := a.db.QueryRowContext(ctx, `
		SELECT hi.id::text
		FROM host_integrations AS hi
		JOIN integration_credentials AS ic
		  ON ic.integration_id = hi.id
		WHERE ic.token_hash = $1
		  AND ic.revoked_at IS NULL
		  AND hi.status = 'active'
	`, tokenHash[:]).Scan(&integrationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidIntegrationToken
		}
		return "", err
	}

	return integrationID, nil
}
