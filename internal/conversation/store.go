package conversation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/mtariq99/dispatchai/internal/enums"
	"github.com/mtariq99/dispatchai/models"
)

type Store interface {
	Lock(ctx context.Context, conversationID uuid.UUID) (func() error, error)
	Load(conversationID, tenantID, userID uuid.UUID) ([]*models.Message, error)
	Save(conversationID, tenantID, userID uuid.UUID, messages []*models.Message) error
}

type MemoryStore struct {
	db *sql.DB
}

func NewMemoryStore(db *sql.DB) *MemoryStore {
	return &MemoryStore{
		db: db,
	}
}

// Lock serializes requests for the same conversation across application instances.
// The session-level advisory lock remains held until the returned function is called.
func (s *MemoryStore) Lock(ctx context.Context, conversationID uuid.UUID) (func() error, error) {
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

func (im *MemoryStore) Load(conversationID, tenantID, userID uuid.UUID) ([]*models.Message, error) {
	ctx := context.Background()
	var ownerTenantID, ownerUserID uuid.UUID

	err := im.db.QueryRowContext(ctx, `
    SELECT tenant_id, user_id
    FROM conversations
    WHERE id = $1
`, conversationID).Scan(&ownerTenantID, &ownerUserID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("load conversation owner: %w", err)
	}

	if err == nil && (ownerTenantID != tenantID || ownerUserID != userID) {
		return nil, fmt.Errorf("conversation not found or does not belong to caller")
	}

	if errors.Is(err, sql.ErrNoRows) {
		return []*models.Message{}, nil
	}
	query := `
		SELECT role, content, tool_calls, tool_call_id
		FROM conversation_messages
		WHERE conversation_id = $1
		ORDER BY sequence ASC`
	rows, err := im.db.QueryContext(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	var messages []*models.Message
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
				return nil, err
			}
		}
		messages = append(messages, message)

	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation messages: %w", err)
	}
	if messages == nil {
		messages = []*models.Message{}
	}

	return messages, nil
}

func (s *MemoryStore) Save(conversationID, tenantID, userID uuid.UUID, messages []*models.Message) error {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	query := `
INSERT INTO conversations (id, tenant_id, user_id)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE
SET updated_at = NOW()
WHERE conversations.tenant_id = EXCLUDED.tenant_id
  AND conversations.user_id = EXCLUDED.user_id
	`
	result, err := tx.ExecContext(ctx, query, conversationID, tenantID, userID)
	if err != nil {
		return fmt.Errorf("upsert conversation: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check conversation ownership: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("conversation not found or does not belong to caller")
	}

	deleteConvMgsQuery := `
	DELETE FROM conversation_messages WHERE conversation_id=$1
	`
	_, err = tx.ExecContext(ctx, deleteConvMgsQuery, conversationID)
	if err != nil {
		return fmt.Errorf("delete prior conversation messages : %w", err)
	}
	for i, msg := range messages {
		var toolCallsJSON []byte

		if len(msg.ToolCalls) > 0 {
			toolCallsJSON, err = json.Marshal(msg.ToolCalls)
			if err != nil {
				return fmt.Errorf("marshal tool calls: %w", err)
			}
		} else {
			toolCallsJSON = []byte(`[]`)
		}

		convMsgQuery := `
		INSERT INTO conversation_messages(
			conversation_id,
			sequence,
			role,
			content,
			tool_calls,
			tool_call_id
		)
		VALUES($1,$2,$3,$4,$5,$6)
	`

		_, err = tx.ExecContext(
			ctx,
			convMsgQuery,
			conversationID,
			i,
			msg.Role,
			msg.Content,
			toolCallsJSON,
			msg.ToolCallID,
		)
		if err != nil {
			return fmt.Errorf("insert conversation message: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("Error commiting conv messages tx : %w", err)
	}
	return nil
}
