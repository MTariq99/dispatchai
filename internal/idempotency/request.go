package idempotency

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

type RequestIdempotency interface {
	Reserve(ctx context.Context, tenantID uuid.UUID, key string, conversationID uuid.UUID, requestID uuid.UUID, runID uuid.UUID, operation string) (bool, error)
	Get(ctx context.Context, tenantID uuid.UUID, key string) (*models.Result, bool, error)
	Complete(ctx context.Context, tenantID uuid.UUID, key string, requestID uuid.UUID, runID uuid.UUID, result *models.Result) error
	Fail(ctx context.Context, tenantID uuid.UUID, key string, requestID uuid.UUID, runID uuid.UUID, result *models.Result) error
}

type RequestIdempotencyStore struct {
	mu sync.RWMutex
	// store map[string]*models.Result
	db *sql.DB
}

func NewIdempotencyStore(db *sql.DB) *RequestIdempotencyStore {
	return &RequestIdempotencyStore{
		// store: make(map[string]*models.Result, 0),
		db: db,
	}
}

func (is *RequestIdempotencyStore) Complete(ctx context.Context, tenantID uuid.UUID, key string, requestID uuid.UUID, runID uuid.UUID, result *models.Result) error {
	if key == "" {
		return fmt.Errorf("idempotency key cannot be empty")
	}

	if result == nil {
		return fmt.Errorf("idempotency result cannot be nil")
	}

	responseJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal idempotency result: %w", err)
	}

	query := `
        UPDATE idempotency_keys
        SET
            status = 'completed',
            response = $1
        WHERE tenant_id = $2
          AND key = $3
          AND request_id = $4
          AND run_id = $5
          AND status = 'pending'
    `

	resultExec, err := is.db.ExecContext(ctx, query, responseJSON, tenantID, key, requestID, runID)
	if err != nil {
		return fmt.Errorf("complete idempotency record: %w", err)
	}

	rows, err := resultExec.RowsAffected()
	if err != nil {
		return fmt.Errorf("check idempotency completion: %w", err)
	}

	if rows != 1 {
		return fmt.Errorf("idempotency record was not completed")
	}

	return nil
}

func (is *RequestIdempotencyStore) Get(ctx context.Context, tenantID uuid.UUID, key string) (*models.Result, bool, error) {
	if key == "" {
		return nil, false, fmt.Errorf("idempotency key cannot be empty")
	}

	query := `
        SELECT response
        FROM idempotency_keys
        WHERE tenant_id = $1
          AND key = $2
          AND status = 'completed'
    `

	var responseJSON []byte

	err := is.db.QueryRowContext(
		ctx,
		query,
		tenantID,
		key,
	).Scan(&responseJSON)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("get idempotency record: %w", err)
	}

	if len(responseJSON) == 0 {
		return nil, false, nil
	}

	var result models.Result

	if err := json.Unmarshal(responseJSON, &result); err != nil {
		return nil, false, fmt.Errorf("unmarshal idempotency response: %w", err)
	}

	return &result, true, nil
}

func (is *RequestIdempotencyStore) Reserve(ctx context.Context, tenantID uuid.UUID, key string, conversationID uuid.UUID, requestID uuid.UUID, runID uuid.UUID, operation string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("idempotency key cannot be empty")
	}
	var returnedKey string
	query := `
		INSERT INTO idempotency_keys (
			key,
			tenant_id,
			conversation_id, 
			request_id,
			run_id,
			operation, 
			status
		)VALUES (
            $1,
            $2,
            $3,
            $4,
            $5,
            $6,
            'pending'
		)ON CONFLICT (tenant_id, key)
        DO NOTHING
        RETURNING key
	`
	err := is.db.QueryRowContext(ctx, query, key, tenantID, conversationID, requestID, runID, operation).Scan(&returnedKey)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reserve  request idempotency record: %w", err)
	}
	return true, nil
}
