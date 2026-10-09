package idempotency

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

var ErrToolCallConflict = errors.New(
	"tool call ID reused with different tool name or arguments",
)

type ToolIdempotency interface {
	Get(ctx context.Context, runID uuid.UUID, callID string) (*models.Result, bool, error)
	Reserve(ctx context.Context, runID uuid.UUID, callID string, toolName string, arguments json.RawMessage) (bool, error)
	Complete(ctx context.Context, runID uuid.UUID, callID string, result *models.Result) error
	Fail(ctx context.Context, runID uuid.UUID, callID string, err *models.Error) error
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

	const insertQuery = `
		INSERT INTO tool_executions (
			id, run_id, call_id, tool_name, arguments, status, attempt
		)
		VALUES ($1, $2, $3, $4, $5, 'pending', 1)
		ON CONFLICT (run_id, call_id) DO NOTHING
	`

	result, err := s.db.ExecContext(
		ctx,
		insertQuery,
		uuid.New(),
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

	if rowsAffected == 1 {
		return true, nil
	}

	const existingQuery = `
		SELECT tool_name, arguments
		FROM tool_executions
		WHERE run_id = $1 AND call_id = $2
	`

	var existingToolName string
	var existingArguments []byte

	err = s.db.QueryRowContext(
		ctx,
		existingQuery,
		runID,
		callID,
	).Scan(&existingToolName, &existingArguments)
	if err != nil {
		return false, fmt.Errorf("inspect existing tool execution: %w", err)
	}

	var existingJSON any
	var requestedJSON any

	if err := json.Unmarshal(existingArguments, &existingJSON); err != nil {
		return false, fmt.Errorf("decode existing tool arguments: %w", err)
	}
	if err := json.Unmarshal(arguments, &requestedJSON); err != nil {
		return false, fmt.Errorf("decode requested tool arguments: %w", err)
	}

	existingCanonical, err := json.Marshal(existingJSON)
	if err != nil {
		return false, fmt.Errorf("canonicalize existing tool arguments: %w", err)
	}
	requestedCanonical, err := json.Marshal(requestedJSON)
	if err != nil {
		return false, fmt.Errorf("canonicalize requested tool arguments: %w", err)
	}

	if existingToolName != toolName || string(existingCanonical) != string(requestedCanonical) {
		return false, ErrToolCallConflict
	}

	return false, nil
}

func (s *ToolIdempotencyStore) Get(ctx context.Context, runID uuid.UUID, callID string) (*models.Result, bool, error) {
	if runID == uuid.Nil {
		return nil, false, fmt.Errorf("run ID cannot be nil")
	}
	if callID == "" {
		return nil, false, fmt.Errorf("call ID cannot be empty")
	}

	const query = `
		SELECT status, result, error_code, error_message, retryable
		FROM tool_executions
		WHERE run_id = $1 AND call_id = $2
	`

	var (
		status       string
		resultJSON   []byte
		errorCode    sql.NullString
		errorMessage sql.NullString
		retryable    bool
	)

	err := s.db.QueryRowContext(ctx, query, runID, callID).Scan(
		&status,
		&resultJSON,
		&errorCode,
		&errorMessage,
		&retryable,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get tool execution: %w", err)
	}

	switch status {
	case "completed":
		if len(resultJSON) == 0 {
			return nil, false, errors.New("completed tool execution has no persisted result")
		}

		var result models.Result
		if err := json.Unmarshal(resultJSON, &result); err != nil {
			return nil, false, fmt.Errorf("unmarshal tool execution result: %w", err)
		}

		return &result, true, nil

	case "failed":
		if !errorCode.Valid || !errorMessage.Valid {
			return nil, false, errors.New("failed tool execution has incomplete error details")
		}

		return &models.Result{
			CallID:  callID,
			Success: false,
			Error: &models.Error{
				Code:      errorCode.String,
				Message:   errorMessage.String,
				Retryable: retryable,
			},
		}, true, nil

	case "pending":
		return nil, false, nil

	default:
		return nil, false, fmt.Errorf(
			"unknown tool execution status %q",
			status,
		)
	}
}

func (s *ToolIdempotencyStore) Complete(ctx context.Context, runID uuid.UUID, callID string, result *models.Result) error {
	if runID == uuid.Nil {
		return errors.New("run id is required")
	}
	if callID == "" {
		return errors.New("call id is required")
	}
	if result == nil {
		return errors.New("result is required")
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal tool result: %w", err)
	}

	query := `
		UPDATE tool_executions
		SET
			status = 'completed',
			result = $3,
			completed_at = NOW()
		WHERE run_id = $1
		  AND call_id = $2
		  AND status = 'pending'
	`

	res, err := s.db.ExecContext(ctx, query, runID, callID, resultJSON)
	if err != nil {
		return fmt.Errorf("complete tool execution: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rows != 1 {
		return errors.New("tool execution is not pending")
	}

	return nil
}

func (s *ToolIdempotencyStore) Fail(ctx context.Context, runID uuid.UUID, callID string, errResult *models.Error) error {
	if runID == uuid.Nil {
		return errors.New("run id is required")
	}
	if callID == "" {
		return errors.New("call id is required")
	}
	if errResult == nil {
		return errors.New("error result is required")
	}

	query := `
		UPDATE tool_executions
		SET
			status = 'failed',
			error_code = $3,
			error_message = $4,
			retryable = $5,
			completed_at = NOW()
		WHERE run_id = $1
		  AND call_id = $2
		  AND status = 'pending'
	`

	res, err := s.db.ExecContext(
		ctx,
		query,
		runID,
		callID,
		errResult.Code,
		errResult.Message,
		errResult.Retryable,
	)
	if err != nil {
		return fmt.Errorf("fail tool execution: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rows != 1 {
		return errors.New("tool execution is not pending")
	}

	return nil
}
