package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

type IdempotencyStore struct {
	db *sql.DB
}

func NewIdempotencyStore(db *sql.DB) *IdempotencyStore {
	return &IdempotencyStore{
		db: db,
	}
}

func (s *IdempotencyStore) Complete(ctx context.Context, integrationID uuid.UUID, key string, requestID uuid.UUID, runID uuid.UUID, response *models.ChatResponse) error {
	if integrationID == uuid.Nil {
		return errors.New("tenant id is required")
	}
	if key == "" {
		return errors.New("idempotency key is required")
	}
	if requestID == uuid.Nil {
		return errors.New("request id is required")
	}
	if runID == uuid.Nil {
		return errors.New("run id is required")
	}
	if response == nil {
		return errors.New("response is required")
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal chat response: %w", err)
	}

	query := `
	UPDATE idempotency_keys
	SET
		run_id = $4,
		status = 'completed',
		response = $5
	WHERE integration_id = $1
	  AND key = $2
	  AND request_id = $3
	  AND status = 'pending'
	`

	result, err := s.db.ExecContext(ctx, query, integrationID, key, requestID, runID, responseJSON)
	if err != nil {
		return fmt.Errorf("complete request idempotency: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rows != 1 {
		return errors.New("request idempotency record is not pending")
	}

	return nil
}

func (s *IdempotencyStore) Get(ctx context.Context, integrationID uuid.UUID, key string) (*models.ChatResponse, bool, error) {
	if integrationID == uuid.Nil {
		return nil, false, errors.New("integration id is required")
	}
	if key == "" {
		return nil, false, errors.New("idempotency key is required")
	}

	var responseJSON []byte

	query := `
		SELECT response
		FROM idempotency_keys
		WHERE integration_id = $1
		  AND key = $2
		  AND status = 'completed'
	`

	err := s.db.QueryRowContext(
		ctx,
		query,
		integrationID,
		key,
	).Scan(&responseJSON)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("get request idempotency: %w", err)
	}

	var response models.ChatResponse

	if err := json.Unmarshal(responseJSON, &response); err != nil {
		return nil, false, fmt.Errorf("unmarshal stored chat response: %w", err)
	}

	return &response, true, nil
}

func (s *IdempotencyStore) Reserve(ctx context.Context, integrationID uuid.UUID, key string, conversationID uuid.UUID, requestID uuid.UUID, fingerprint string) (bool, error) {
	if integrationID == uuid.Nil {
		return false, errors.New("integration id is required")
	}
	if key == "" {
		return false, errors.New("idempotency key is required")
	}
	if conversationID == uuid.Nil {
		return false, errors.New("conversation id is required")
	}
	if requestID == uuid.Nil {
		return false, errors.New("request id is required")
	}

	if fingerprint == "" {
		return false, errors.New("request fingerprint is required")
	}

	query := `
INSERT INTO idempotency_keys (
    key,
    integration_id,
    conversation_id,
    request_id,
    operation,
    status,
    request_fingerprint
)
VALUES ($1, $2, $3, $4, 'chat', 'pending', $5)
ON CONFLICT (integration_id, key) DO NOTHING
`

	result, err := s.db.ExecContext(ctx, query, key, integrationID, conversationID, requestID, fingerprint)
	if err != nil {
		return false, fmt.Errorf("reserve request idempotency: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("get affected rows: %w", err)
	}

	return rows == 1, nil
}

func (s *IdempotencyStore) Fail(ctx context.Context, integrationID uuid.UUID, key string, requestID uuid.UUID, runID uuid.UUID, response *models.ChatResponse) error {
	if integrationID == uuid.Nil {
		return errors.New("integration id is required")
	}
	if key == "" {
		return errors.New("idempotency key is required")
	}
	if requestID == uuid.Nil {
		return errors.New("request id is required")
	}
	if runID == uuid.Nil {
		return errors.New("run id is required")
	}
	if response == nil {
		return errors.New("response is required")
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal failed chat response: %w", err)
	}

	query := `
		UPDATE idempotency_keys
		SET
			run_id = $4,
			status = 'failed',
			response = $5
		WHERE integration_id = $1
		  AND key = $2
		  AND request_id = $3
		  AND status = 'pending'
	`

	result, err := s.db.ExecContext(ctx, query, integrationID, key, requestID, runID, responseJSON)
	if err != nil {
		return fmt.Errorf("fail request idempotency: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rows != 1 {
		return errors.New("request idempotency record is not pending")
	}

	return nil
}

func (s *IdempotencyStore) FailBeforeRun(ctx context.Context, integrationID uuid.UUID, key string, requestID uuid.UUID, response *models.ChatResponse) error {
	if integrationID == uuid.Nil {
		return errors.New("integration id is required")
	}

	if key == "" {
		return errors.New("idempotency key is required")
	}

	if requestID == uuid.Nil {
		return errors.New("request id is required")
	}

	if response == nil {
		return errors.New("response is required")
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal failed chat response: %w", err)
	}

	const query = `
		UPDATE idempotency_keys
		SET
			status = 'failed',
			response = $4
		WHERE integration_id = $1
		  AND key = $2
		  AND request_id = $3
		  AND status = 'pending'
		  AND run_id IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, integrationID, key, requestID, responseJSON)
	if err != nil {
		return fmt.Errorf("fail request before run: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rows != 1 {
		return errors.New("request idempotency record is not pending before run")
	}

	return nil
}

func (s *IdempotencyStore) GetState(ctx context.Context, integrationID uuid.UUID, key string) (*models.RequestState, bool, error) {
	if integrationID == uuid.Nil {
		return nil, false, errors.New("integration id is required")
	}

	if key == "" {
		return nil, false, errors.New("idempotency key is required")
	}

	var (
		status         string
		requestID      uuid.UUID
		runID          *uuid.UUID
		conversationID uuid.UUID
		responseJSON   []byte
		fingerprint    string
	)

	query := `
		SELECT
			status,
			request_id,
			run_id,
			conversation_id,
			response,
			COALESCE(request_fingerprint, '')
		FROM idempotency_keys
		WHERE integration_id = $1
		AND key = $2
	`

	err := s.db.QueryRowContext(ctx, query, integrationID, key).Scan(
		&status,
		&requestID,
		&runID,
		&conversationID,
		&responseJSON,
		&fingerprint,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("get request idempotency state: %w", err)
	}

	state := &models.RequestState{
		Status:         models.RequestStatus(status),
		RequestID:      requestID,
		RunID:          runID,
		ConversationID: conversationID,
		Fingerprint:    fingerprint,
	}

	if len(responseJSON) > 0 {
		var response models.ChatResponse

		if err := json.Unmarshal(responseJSON, &response); err != nil {
			return nil, false, fmt.Errorf("unmarshal stored chat response: %w", err)
		}

		state.Response = &response
	}

	return state, true, nil
}
