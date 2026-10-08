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

type Idempotency interface {
	Get(key string) (*models.Result, bool, error)
	Set(key string, res *models.Result) error
	BuildKey(conversationId uuid.UUID, call *models.Call) string
}

type IdempotencyStore struct {
	mu sync.RWMutex
	// store map[string]*models.Result
	db *sql.DB
}

func NewIdempotencyStore(db *sql.DB) *IdempotencyStore {
	return &IdempotencyStore{
		// store: make(map[string]*models.Result, 0),
		db: db,
	}
}

func (is *IdempotencyStore) Set(key string, conversationID uuid.UUID, operation string, res *models.Result) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	responseJSON, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	upsertQuery := `
		INSERT INTO idempotency_keys (key, conversation_id, operation, status, response)
		VALUES ($1, $2, $3, 'completed', $4)
		ON CONFLICT (key) DO UPDATE
		SET status = 'completed', response = $4
	`

	result, err := is.db.ExecContext(
		context.Background(),
		upsertQuery,
		key,
		conversationID,
		operation,
		responseJSON,
	)
	if err != nil {
		return fmt.Errorf("upsert idempotency record: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("idempotency record for key %q was not written", key)
	}

	return nil
}

func (is *IdempotencyStore) Get(key string) (*models.Result, bool, error) {
	if key == "" {
		return nil, false, fmt.Errorf("idempotency key cannot be empty")
	}

	getQuery := `
		SELECT response
		FROM idempotency_keys
		WHERE key = $1 AND status = 'completed'
	`

	var responseJSON []byte
	err := is.db.QueryRowContext(context.Background(), getQuery, key).Scan(&responseJSON)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("query idempotency record: %w", err)
	}

	if len(responseJSON) == 0 {
		return nil, false, nil
	}

	var result models.Result
	if err := json.Unmarshal(responseJSON, &result); err != nil {
		return nil, false, fmt.Errorf("unmarshal stored result: %w", err)
	}

	return &result, true, nil
}

func (is *IdempotencyStore) Reserve(key string, conversationID uuid.UUID, operation string) (bool, error) {
	var returnedKey string
	err := is.db.QueryRow(`
		INSERT INTO idempotency_keys (
			id,
			conversation_id, 
			operation, 
			status
		)VALUES (
			$1, 
			$2, 
			$3, 
			'pending'
		)ON CONFLICT (key) DO NOTHING
		RETURNING key
	`, key, conversationID, operation).Scan(&returnedKey)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
