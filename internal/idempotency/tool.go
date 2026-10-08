package idempotency

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

type ToolIdempotency interface {
	Get(ctx context.Context, runID uuid.UUID, callID string) (*models.Result, bool, error)
	Reserve(ctx context.Context, runID uuid.UUID, callID string, toolName string, arguments json.RawMessage) (bool, error)
	Complete(ctx context.Context, runID uuid.UUID, callID string, result *models.Result) error
	Fail(ctx context.Context, runID uuid.UUID, callID string, err *models.Error) error
	BuildKey(conversationId uuid.UUID, call *models.Call) string
}

type ToolIdempotencyStore struct {
	db *sql.DB
}

func NewToolIdempotency(db *sql.DB) *ToolIdempotencyStore {
	return &ToolIdempotencyStore{
		db: db,
	}
}

func (s *ToolIdempotencyStore) Reserve(ctx context.Context, runID uuid.UUID, callID string, toolName string, arguments json.RawMessage) (bool, error) {
	if runID == uuid.Nil {
		return false, fmt.Errorf("run ID cannot be nil")
	}
	if callID == "" {
		return false, fmt.Errorf("call ID cannot be empty")
	}
	if toolName == "" {
		return false, fmt.Errorf("tool name cannot be empty")
	}
	if len(arguments) == 0 {
		return false, fmt.Errorf("tool arguments cannot be empty")
	}
	if !json.Valid(arguments) {
		return false, fmt.Errorf("tool arguments must be valid JSON")
	}

	executionID := uuid.New()

	const query = `
		INSERT INTO tool_executions (
			id,
			run_id,
			call_id,
			tool_name,
			arguments,
			status,
			attempt
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			'pending',
			1
		)
		ON CONFLICT (run_id, call_id)
		DO NOTHING
	`

	result, err := s.db.ExecContext(
		ctx,
		query,
		executionID,
		runID,
		callID,
		toolName,
		arguments,
	)
	if err != nil {
		return false, fmt.Errorf("reserve tool execution: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check tool execution reservation: %w", err)
	}

	if rowsAffected == 0 {
		return false, nil
	}

	return true, nil
}

func (tis *ToolIdempotencyStore) Get(ctx context.Context, runID uuid.UUID, callID string) (*models.Result, bool, error) {
	if runID == uuid.Nil {
		return nil, false, fmt.Errorf("run ID cannot be nil")
	}
	if callID == "" {
		return nil, false, fmt.Errorf("call ID cannot be empty")
	}

	const query = `
		SELECT result
		FROM tool_executions
		WHERE run_id = $1 AND call_id = $2 AND status = 'completed'
	`

	var responseJSON []byte
	err := tis.db.QueryRowContext(
		ctx,
		query,
		runID,
		callID,
	).Scan(&responseJSON)

	if err == sql.ErrNoRows {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, fmt.Errorf("get tool execution: %w", err)
	}

	if len(responseJSON) == 0 {
		return nil, false, nil
	}

	var result models.Result

	if err := json.Unmarshal(responseJSON, &result); err != nil {
		return nil, false, fmt.Errorf("unmarshal tool execution response: %w", err)
	}

	return &result, true, nil
}

func (tis *ToolIdempotencyStore) Complete(ctx context.Context, runID uuid.UUID, callID string, result *models.Result) error {
	return nil
}

func (tis *ToolIdempotencyStore) Fail(ctx context.Context, runID uuid.UUID, callID string, err *models.Error) error {
	return nil
}
