package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/internal/enums"
	"github.com/mtariq99/dispatchai/models"
)

type ConversationStore struct {
	db *sql.DB
}

func NewConversationStore(db *sql.DB) (*ConversationStore, error) {
	if db == nil {
		return nil, errors.New("database cannot be nil")
	}

	return &ConversationStore{
		db: db,
	}, nil
}

func (s *ConversationStore) Lock(ctx context.Context, conversationID uuid.UUID) (func() error, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("conversation database is not configured")
	}
	if conversationID == uuid.Nil {
		return nil, fmt.Errorf("conversation id is required")
	}

	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire conversation database connection: %w", err)
	}

	if _, err := conn.ExecContext(
		ctx,
		"SELECT pg_advisory_lock(hashtextextended($1, 0))",
		conversationID.String(),
	); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("lock conversation: %w", err)
	}

	release := func() error {
		_, unlockErr := conn.ExecContext(
			context.Background(),
			"SELECT pg_advisory_unlock(hashtextextended($1, 0))",
			conversationID.String(),
		)
		closeErr := conn.Close()

		if unlockErr != nil {
			return fmt.Errorf("unlock conversation: %w", unlockErr)
		}
		if closeErr != nil {
			return fmt.Errorf("release conversation database connection: %w", closeErr)
		}
		return nil
	}

	return release, nil
}

func (s *ConversationStore) Load(conversationID uuid.UUID, integrationID uuid.UUID) ([]*models.Message, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("conversation database is not configured")
	}
	if conversationID == uuid.Nil {
		return nil, fmt.Errorf("conversation id is required")
	}
	if integrationID == uuid.Nil {
		return nil, fmt.Errorf("integration id is required")
	}

	ctx := context.Background()
	IsOwnedConvQuery := `
		SELECT EXISTS (
            SELECT 1
            FROM conversations
            WHERE id = $1 AND integration_id = $2
        )
		`
	var owned bool
	err := s.db.QueryRowContext(ctx, IsOwnedConvQuery, conversationID, integrationID).Scan(&owned)
	if err != nil {
		return nil, fmt.Errorf("check conversation ownership: %w", err)
	}
	if !owned {
		return nil, fmt.Errorf("conversation not found or does not belong to integration")
	}
	convMsgsQuery := `
		SELECT role, content, tool_calls, tool_call_id
         FROM conversation_messages
         WHERE conversation_id = $1
         ORDER BY sequence ASC
		`
	rows, err := s.db.QueryContext(ctx, convMsgsQuery, conversationID)
	if err != nil {
		return nil, fmt.Errorf("load conversation messages: %w", err)
	}
	defer rows.Close()

	messages := make([]*models.Message, 0)

	for rows.Next() {
		var (
			role         string
			content      sql.NullString
			toolCallsRaw []byte
			toolCallID   sql.NullString
		)

		if err := rows.Scan(
			&role,
			&content,
			&toolCallsRaw,
			&toolCallID,
		); err != nil {
			return nil, fmt.Errorf("scan conversation message: %w", err)
		}

		message := &models.Message{
			Role:       enums.Role(role),
			Content:    content.String,
			ToolCallID: toolCallID.String,
		}

		if len(toolCallsRaw) > 0 {
			if err := json.Unmarshal(toolCallsRaw, &message.ToolCalls); err != nil {
				return nil, fmt.Errorf("decode conversation tool calls: %w", err)
			}
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation messages: %w", err)
	}

	return messages, nil
}

func (s *ConversationStore) Save(conversationID uuid.UUID, integrationID uuid.UUID, messages []*models.Message) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("conversation database is not configured")
	}
	if conversationID == uuid.Nil {
		return fmt.Errorf("conversation id is required")
	}
	if integrationID == uuid.Nil {
		return fmt.Errorf("integration id is required")
	}

	for i, msg := range messages {
		if msg == nil {
			return fmt.Errorf("conversation message at index %d is nil", i)
		}
	}

	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save conversation transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Verify ownership and lock the conversation row for this transaction.
	var ownerIntegrationID uuid.UUID

	err = tx.QueryRowContext(
		ctx,
		`SELECT integration_id
		 FROM conversations
		 WHERE id = $1
		 FOR UPDATE`,
		conversationID,
	).Scan(&ownerIntegrationID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf(
				"conversation not found or does not belong to integration",
			)
		}
		return fmt.Errorf("load conversation ownership: %w", err)
	}

	if ownerIntegrationID != integrationID {
		return fmt.Errorf(
			"conversation not found or does not belong to integration",
		)
	}

	// Replace the existing conversation messages atomically.
	_, err = tx.ExecContext(
		ctx,
		`DELETE FROM conversation_messages WHERE conversation_id = $1`,
		conversationID,
	)
	if err != nil {
		return fmt.Errorf("delete previous conversation messages: %w", err)
	}

	for i, msg := range messages {
		toolCallsJSON := []byte("[]")

		if len(msg.ToolCalls) > 0 {
			toolCallsJSON, err = json.Marshal(msg.ToolCalls)
			if err != nil {
				return fmt.Errorf("marshal tool calls at index %d: %w", i, err)
			}
		}
		insertConvMsgQuery := `
			INSERT INTO conversation_messages (
				conversation_id,
				sequence,
				role,
				content,
				tool_calls,
				tool_call_id
			)
			VALUES ($1, $2, $3, $4, $5, $6)`
		_, err = tx.ExecContext(ctx, insertConvMsgQuery, conversationID, i, msg.Role, msg.Content, toolCallsJSON, msg.ToolCallID)
		if err != nil {
			return fmt.Errorf("insert conversation message at index %d: %w", i, err)
		}
	}
	updateConvQuery := `
		UPDATE conversations
		 SET updated_at = NOW()
		 WHERE id = $1 AND integration_id = $2
		 `
	_, err = tx.ExecContext(ctx, updateConvQuery, conversationID, integrationID)
	if err != nil {
		return fmt.Errorf("update conversation timestamp: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit conversation messages transaction: %w", err)
	}

	return nil
}

func (s *ConversationStore) CreateConversation(ctx context.Context, integrationID uuid.UUID, title string) (*models.Conversation, error) {
	if integrationID == uuid.Nil {
		return nil, fmt.Errorf("integration id is required")
	}

	query := `
		INSERT INTO conversations (
			id,
			integration_id,
			title
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			integration_id,
			title,
			status,
			created_at,
			updated_at,
			archived_at
	`

	conversation := &models.Conversation{}
	var nullableTitle sql.NullString
	var archivedAt sql.NullTime

	conversationID := uuid.New()

	err := s.db.QueryRowContext(
		ctx,
		query,
		conversationID,
		integrationID,
		title,
	).Scan(
		&conversation.ID,
		&conversation.IntegrationID,
		&nullableTitle,
		&conversation.Status,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
		&archivedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create conversation: %w", err)
	}

	if nullableTitle.Valid {
		conversation.Title = nullableTitle.String
	}

	if archivedAt.Valid {
		conversation.ArchivedAt = archivedAt.Time
	}

	return conversation, nil
}

func (s *ConversationStore) GetConversation(ctx context.Context, integrationID, conversationID uuid.UUID) (*models.Conversation, error) {
	if integrationID == uuid.Nil {
		return nil, fmt.Errorf("integration id is required")
	}

	if conversationID == uuid.Nil {
		return nil, fmt.Errorf("conversation id is required")
	}

	query := `
		SELECT
			id,
			integration_id,
			title,
			status,
			created_at,
			updated_at,
			archived_at
		FROM conversations
		WHERE id = $1
		  AND integration_id = $2
	`

	conversation := &models.Conversation{}
	var nullableTitle sql.NullString
	var archivedAt sql.NullTime

	err := s.db.QueryRowContext(
		ctx,
		query,
		conversationID,
		integrationID,
	).Scan(
		&conversation.ID,
		&conversation.IntegrationID,
		&nullableTitle,
		&conversation.Status,
		&conversation.CreatedAt,
		&conversation.UpdatedAt,
		&archivedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("conversation not found or does not belong to integration")
		}

		return nil, fmt.Errorf("get conversation: %w", err)
	}

	if nullableTitle.Valid {
		conversation.Title = nullableTitle.String
	}

	if archivedAt.Valid {
		conversation.ArchivedAt = archivedAt.Time
	}

	return conversation, nil
}
