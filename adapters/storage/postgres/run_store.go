package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/mtariq99/dispatchai/internal/run"
)

type RunStore struct {
	db *sql.DB
}

func NewRunStore(db *sql.DB) *RunStore {
	return &RunStore{
		db: db,
	}
}

func (s *RunStore) Create(ctx context.Context, r *run.Run) error {
	const query = `
	INSERT INTO conversation_runs (
		id,
		conversation_id,
		request_id,
		status,
		model_provider,
		model_name,
		started_at,
		completed_at,
		input_tokens,
		output_tokens,
		total_tokens,
		error_code,
		error_message,
		created_at,
		updated_at
	)
	VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8,
		$9, $10, $11, $12, $13, $14, $15
	)
`

	var startedAt any
	if !r.StartedAt.IsZero() {
		startedAt = r.StartedAt
	}

	_, err := s.db.ExecContext(
		ctx,
		query,
		r.ID,
		r.ConversationID,
		r.RequestID,
		r.Status,
		r.ModelProvider,
		r.ModelName,
		startedAt,
		r.CompletedAt,
		r.InputTokens,
		r.OutputTokens,
		r.TotalTokens,
		nullableString(r.ErrorCode),
		nullableString(r.ErrorMessage),
		r.CreatedAt,
		r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create run: %w", err)
	}

	return nil
}

func (s *RunStore) Get(ctx context.Context, runID uuid.UUID) (*run.Run, error) {
	const query = `
		SELECT
			cr.id,
			c.tenant_id,
			c.user_id,
			cr.conversation_id,
			cr.request_id,
			cr.status,
			cr.model_provider,
			cr.model_name,
			cr.started_at,
			cr.completed_at,
			cr.input_tokens,
			cr.output_tokens,
			cr.total_tokens,
			cr.error_code,
			cr.error_message,
			cr.answer,
			cr.created_at,
			cr.updated_at
		FROM conversation_runs cr
		INNER JOIN conversations c
			ON c.id = cr.conversation_id
		WHERE cr.id = $1
	`

	return s.scanRun(ctx, query, runID)
}

func (s *RunStore) GetByRequestID(ctx context.Context, requestID uuid.UUID) (*run.Run, error) {
	const query = `
	SELECT
		cr.id,
		c.tenant_id,
		c.user_id,
		cr.conversation_id,
		cr.request_id,
		cr.status,
		cr.model_provider,
		cr.model_name,
		cr.started_at,
		cr.completed_at,
		cr.input_tokens,
		cr.output_tokens,
		cr.total_tokens,
		cr.error_code,
		cr.error_message,
		cr.answer,
		cr.created_at,
		cr.updated_at
	FROM conversation_runs cr
	INNER JOIN conversations c
		ON c.id = cr.conversation_id
	WHERE cr.request_id = $1
	`

	return s.scanRun(ctx, query, requestID)
}

func (s *RunStore) Update(ctx context.Context, r *run.Run) error {
	const query = `
	UPDATE conversation_runs
	SET
		status = $2,
		model_provider = $3,
		model_name = $4,
		started_at = $5,
		completed_at = $6,
		input_tokens = $7,
		output_tokens = $8,
		total_tokens = $9,
		error_code = $10,
		error_message = $11,
		answer = $12,
		updated_at = $13
	WHERE id = $1
	`

	var startedAt any
	if !r.StartedAt.IsZero() {
		startedAt = r.StartedAt
	}

	result, err := s.db.ExecContext(
		ctx,
		query,
		r.ID,
		r.Status,
		r.ModelProvider,
		r.ModelName,
		startedAt,
		r.CompletedAt,
		r.InputTokens,
		r.OutputTokens,
		r.TotalTokens,
		nullableString(r.ErrorCode),
		nullableString(r.ErrorMessage),
		r.Answer,
		r.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update run: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get updated run rows: %w", err)
	}

	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *RunStore) scanRun(ctx context.Context, query string, id uuid.UUID) (*run.Run, error) {
	var r run.Run

	var (
		modelProvider sql.NullString
		modelName     sql.NullString
		startedAt     sql.NullTime
		completedAt   sql.NullTime
		errorCode     sql.NullString
		errorMessage  sql.NullString
		answer        sql.NullString
	)

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&r.ID,
		&r.TenantID,
		&r.UserID,
		&r.ConversationID,
		&r.RequestID,
		&r.Status,
		&modelProvider,
		&modelName,
		&startedAt,
		&completedAt,
		&r.InputTokens,
		&r.OutputTokens,
		&r.TotalTokens,
		&errorCode,
		&errorMessage,
		&answer,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}

		return nil, fmt.Errorf("get run: %w", err)
	}

	r.ModelProvider = modelProvider.String
	r.ModelName = modelName.String

	if startedAt.Valid {
		r.StartedAt = startedAt.Time
	}

	if completedAt.Valid {
		value := completedAt.Time
		r.CompletedAt = &value
	}

	r.ErrorCode = errorCode.String
	r.ErrorMessage = errorMessage.String
	r.Answer = answer.String

	return &r, nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}

	return value
}
