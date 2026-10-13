package storage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/cockroachdb/errors"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

type ToolRegistry struct {
	db *sql.DB
}

func NewToolRegistry(db *sql.DB) *ToolRegistry {
	return &ToolRegistry{
		db: db,
	}
}

const toolColumns = `id::text,integration_id::text,name,description,input_schema,version,enabled,deleted_at,created_at,updated_at`

func ScanTool(row interface{ Scan(...any) error }) (*tools.DefinitionRecord, error) {
	var record tools.DefinitionRecord
	var schema []byte

	err := row.Scan(
		&record.ID,
		&record.IntegrationID,
		&record.Definition.Name,
		&record.Definition.Description,
		&schema,
		&record.Version,
		&record.Enabled,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(schema, &record.Definition.Parameters); err != nil {
		return nil, fmt.Errorf("decode tool input schema: %w", err)
	}
	return &record, nil
}

func ValidateDefination(def models.Definition) ([]byte, error) {
	if def.Name == "" {
		return nil, errors.New("tool name cannot be empty")
	}
	if len(def.Description) == 0 {
		return nil, errors.New("tool description cannot be empty")
	}

	if len(def.Name) > 100 {
		return nil, errors.New("tool name exceeds 100 characters")
	}
	if def.Parameters == nil {
		return nil, errors.New("tool parameters schema cannot be nil")
	}
	if def.Parameters["type"] != "object" {
		return nil, errors.New("tool parameters schema must have type object")
	}

	schema, err := json.Marshal(def.Parameters)
	if err != nil {
		return nil, fmt.Errorf("encode tool schema: %w", err)
	}
	return schema, nil
}

func (tr *ToolRegistry) CreateToolRegistry(ctx context.Context, integrationID string, defination models.Definition, idempotencyKey string, requestHash []byte) (*tools.DefinitionRecord, bool, error) {
	if tr == nil || tr.db == nil {
		return nil, false, errors.New("tool registry database is not configured")
	}
	if integrationID == "" || idempotencyKey == "" || len(requestHash) == 0 {
		return nil, false, errors.New("missing required fields")
	}
	schema, err := ValidateDefination(defination)
	if err != nil {
		return nil, false, err
	}
	tx, err := tr.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, fmt.Errorf("begin tool registration: %w", err)
	}
	defer tx.Rollback()

	var oldHash []byte
	var oldToolID string
	selectQuery := `
	SELECT request_hash,tool_id::text
	FROM tool_registration_idempotency
	WHERE integration_id = $1::uuid AND idempotency_key = $2
	`
	err = tx.QueryRowContext(ctx, selectQuery, integrationID, idempotencyKey).Scan(&oldHash, oldToolID)
	if err == nil {
		if !bytes.Equal(oldHash, requestHash) {
			return nil, false, errs.ErrIdempotencyConflict
		}
		getRecordQuery := `
	SELECT` + toolColumns + `
	FROM tool_definations
	WHERE id = $1::uuid AND integration_id = $2::uuid
	`
		sqlRow := tx.QueryRowContext(ctx, getRecordQuery, oldToolID, integrationID)
		record, getErr := ScanTool(sqlRow)
		if getErr != nil {
			return nil, false, fmt.Errorf("load registered tool: %w", getErr)
		}
		if err := tx.Commit(); err != nil {
			return nil, false, fmt.Errorf("commit registration replay: %w", err)
		}
		return record, true, nil
	}

	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, false, fmt.Errorf("check registration idempotency: %w", err)
		}
	}

	var integrationStatus string
	integrationStatusQuery := `
	SELECT status FROM host_integrations WHERE id = $1::uuid FOR SHARE
	`
	err = tx.QueryRowContext(ctx, integrationStatusQuery, integrationID).Scan(&integrationStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, errs.ErrIntegrationInactive
	}
	if err != nil {
		return nil, false, fmt.Errorf("load integration: %w", err)
	}
	if integrationStatus != "active" {
		return nil, false, errs.ErrIntegrationInactive
	}
	toolInsertQuery := `
		INSERT INTO tool_definitions
			(integration_id, name, description, input_schema, enabled, version)
		VALUES ($1::uuid, $2, $3, $4::jsonb, TRUE, 1)
		RETURNING` + toolColumns
	sqlInsertToolRow := tx.QueryRowContext(ctx, toolInsertQuery, integrationID, defination.Name, defination.Description, schema)
	record, err := ScanTool(sqlInsertToolRow)
	if err != nil {
		return nil, false, fmt.Errorf("insert tool definition: %w", err)
	}

	if err := insertToolVersion(ctx, tx, record, "created"); err != nil {
		return nil, false, err
	}
	InsertIdempotencyQuery := `
		INSERT INTO tool_registration_idempotency
			(integration_id, idempotency_key, request_hash, tool_id)
		VALUES ($1::uuid, $2, $3, $4::uuid)
	`
	_, err = tx.ExecContext(ctx, InsertIdempotencyQuery, integrationID, idempotencyKey, requestHash, record.ID)
	if err != nil {
		return nil, false, fmt.Errorf("persist registration idempotency: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit tool registration: %w", err)
	}
	return record, false, nil
}

func insertToolVersion(ctx context.Context, tx *sql.Tx, record *tools.DefinitionRecord, changeType string) error {
	schema, err := json.Marshal(record.Definition.Parameters)
	if err != nil {
		return fmt.Errorf("marshal tool schema: %w", err)
	}
	if changeType == "" {
		errors.New("tool version insertion requires change type")
	}
	insertToolVersionQuery := `
	INSERT INTO(tool_id,version,name,description,input_schema,enabled,change_type)
	VALUES($1,$2,$3,$4,$5::jsonb,$6$7)
	`
	_, err = tx.ExecContext(
		ctx,
		insertToolVersionQuery,
		record.ID,
		record.Version,
		record.Definition.Name,
		record.Definition.Description,
		string(schema),
		changeType,
	)
	if err != nil {
		return err
	}
	return nil
}
