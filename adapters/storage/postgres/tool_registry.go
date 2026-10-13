package storage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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

func ValidateDefinition(def *models.Definition) ([]byte, error) {
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

func (tr *ToolRegistry) CreateToolRegistry(ctx context.Context, integrationID string, defination *models.Definition, idempotencyKey string, requestHash []byte) (*tools.DefinitionRecord, bool, error) {
	if tr == nil || tr.db == nil {
		return nil, false, errors.New("tool registry database is not configured")
	}
	if integrationID == "" || idempotencyKey == "" || len(requestHash) == 0 {
		return nil, false, errors.New("missing required fields")
	}
	schema, err := ValidateDefinition(defination)
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
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, fmt.Errorf("check registration idempotency: %w", err)
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

func (tr *ToolRegistry) GetTool(ctx context.Context, integrationID, toolID string) (*tools.DefinitionRecord, error) {
	if tr == nil || tr.db == nil {
		return nil, errors.New("tool registry database is not configured")
	}

	toolSlctQuery := `SELECT ` + toolColumns + `
	FROM tool_definitions
	WHERE id = $1::uuid AND integration_id=$2::uuid AND deleted_at IS NULL`
	toolRow := tr.db.QueryRowContext(ctx, toolSlctQuery, toolID, integrationID)
	tool, err := ScanTool(toolRow)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.ErrToolNotFound
	}
	return tool, nil
}

func (r *ToolRegistry) ListEnabled(ctx context.Context, integrationID string) ([]*tools.DefinitionRecord, error) {
	enabledSlctQuery := `
		SELECT ` + toolColumns + `
		FROM tool_definitions d
		WHERE d.integration_id = $1::uuid
		  AND d.enabled = TRUE
		  AND d.deleted_at IS NULL
		  AND EXISTS (
			  SELECT 1 FROM host_integrations i
			  WHERE i.id = d.integration_id AND i.status = 'active'
		  )
		ORDER BY d.name
	`
	rows, err := r.db.QueryContext(ctx, enabledSlctQuery, integrationID)
	if err != nil {
		return nil, fmt.Errorf("list enabled tool definitions: %w", err)
	}
	defer rows.Close()

	var records []*tools.DefinitionRecord
	for rows.Next() {
		record, err := ScanTool(rows)
		if err != nil {
			return nil, fmt.Errorf("scan tool definition: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tool definitions: %w", err)
	}
	return records, nil
}

func (r *ToolRegistry) GetForExecution(ctx context.Context, integrationID, toolID string, expectedVersion int) (*tools.DefinitionRecord, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("tool registry database is not configured")
	}
	selectQuery := `
		SELECT ` + toolColumns + `
		FROM tool_definitions d
		JOIN host_integrations i ON i.id = d.integration_id
		WHERE d.integration_id = $1::uuid
		  AND d.id = $2::uuid
		  AND d.enabled = TRUE
		  AND d.deleted_at IS NULL
		  AND i.status = 'active'
	`
	SqlRow := r.db.QueryRowContext(ctx, selectQuery, integrationID, toolID)
	record, err := ScanTool(SqlRow)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errs.ErrToolUnavailable
	}
	if err != nil {
		return nil, fmt.Errorf("validate tool for execution: %w", err)
	}
	if expectedVersion <= 0 || record.Version != expectedVersion {
		return nil, errs.ErrVersionConflict
	}
	return record, nil
}

func (r *ToolRegistry) beginMutation(ctx context.Context, integrationID, toolID string, expectedVersion int) (*sql.Tx, *tools.DefinitionRecord, error) {
	if r == nil || r.db == nil {
		return nil, nil, errors.New("tool registry database is not configured")
	}
	if expectedVersion <= 0 {
		return nil, nil, errs.ErrVersionConflict
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("begin tool mutation: %w", err)
	}
	lockedSelectQuery := `
		SELECT ` + toolColumns + `
		FROM tool_definitions
		WHERE id = $1::uuid
		  AND integration_id = $2::uuid
		  AND deleted_at IS NULL
		FOR UPDATE
	`
	lockedSqlRow := tx.QueryRowContext(ctx, lockedSelectQuery, toolID, integrationID)
	current, err := ScanTool(lockedSqlRow)
	if errors.Is(err, sql.ErrNoRows) {
		tx.Rollback()
		return nil, nil, errs.ErrToolNotFound
	}
	if err != nil {
		tx.Rollback()
		return nil, nil, fmt.Errorf("load tool for mutation: %w", err)
	}
	if current.Version != expectedVersion {
		tx.Rollback()
		return nil, nil, errs.ErrVersionConflict
	}
	return tx, current, nil
}

func (r *ToolRegistry) commitMutation(ctx context.Context, tx *sql.Tx, record *tools.DefinitionRecord, changeType string) (*tools.DefinitionRecord, error) {
	if changeType != "" {
		if err := insertToolVersion(ctx, tx, record, changeType); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tool mutation: %w", err)
	}
	return record, nil
}

func (r *ToolRegistry) Update(ctx context.Context, integrationID, toolID string, expectedVersion int, definition *models.Definition) (*tools.DefinitionRecord, error) {
	schema, err := ValidateDefinition(definition)
	if err != nil {
		return nil, err
	}

	tx, current, err := r.beginMutation(ctx, integrationID, toolID, expectedVersion)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	selectQuery := `
			SELECT EXISTS (
				SELECT 1 FROM tool_definitions
				WHERE integration_id = $1::uuid
				  AND name = $2
				  AND deleted_at IS NULL
				  AND id <> $3::uuid
			)
		`
	if current.Definition.Name != definition.Name {
		var exists bool
		err := tx.QueryRowContext(ctx, selectQuery, integrationID, definition.Name, toolID).Scan(&exists)
		if err != nil {
			return nil, fmt.Errorf("mutate tool definition: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("mutate tool definition: %w", errors.New("tool name already exists in this integration"))
		}
	}

	record, changeType, err := updateDefinition(ctx, tx, integrationID, toolID, definition, schema, current.Version+1)
	if err != nil {
		return nil, fmt.Errorf("mutate tool definition: %w", err)
	}
	return r.commitMutation(ctx, tx, record, changeType)
}

func (r *ToolRegistry) SetEnabled(ctx context.Context, integrationID, toolID string, expectedVersion int, enabled bool) (*tools.DefinitionRecord, error) {
	tx, current, err := r.beginMutation(ctx, integrationID, toolID, expectedVersion)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if current.Enabled == enabled {
		return r.commitMutation(ctx, tx, current, "")
	}

	change := "disabled"
	if enabled {
		change = "enabled"
	}
	updateQuery := `
		UPDATE tool_definitions
		SET enabled = $4, version = version + 1, updated_at = NOW()
		WHERE id = $1::uuid AND integration_id = $2::uuid
		  AND version = $3 AND deleted_at IS NULL
		RETURNING ` + toolColumns
	sqlRow := tx.QueryRowContext(ctx, updateQuery, toolID, integrationID, current.Version, enabled)

	record, err := ScanTool(sqlRow)
	if err != nil {
		return nil, fmt.Errorf("mutate tool definition: %w", err)
	}
	return r.commitMutation(ctx, tx, record, change)
}

func (r *ToolRegistry) SoftDelete(ctx context.Context, integrationID, toolID string, expectedVersion int) error {
	tx, current, err := r.beginMutation(ctx, integrationID, toolID, expectedVersion)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	updateQuery := `
		UPDATE tool_definitions
		SET enabled = FALSE, deleted_at = NOW(),
		    version = version + 1, updated_at = NOW()
		WHERE id = $1::uuid AND integration_id = $2::uuid
		  AND version = $3 AND deleted_at IS NULL
		RETURNING ` + toolColumns
	updateSqlRow := tx.QueryRowContext(ctx, updateQuery, toolID, integrationID, current.Version)
	record, err := ScanTool(updateSqlRow)
	if err != nil {
		return fmt.Errorf("mutate tool definition: %w", err)
	}
	_, err = r.commitMutation(ctx, tx, record, "deleted")
	return err
}

func updateDefinition(ctx context.Context, tx *sql.Tx, integrationID, toolID string, definition *models.Definition, schema []byte, nextVersion int) (*tools.DefinitionRecord, string, error) {
	query := `   -- scanTool -> ScanTool
		UPDATE tool_definitions
		SET name = $3, description = $4, input_schema = $5::jsonb,
		    version = $6, updated_at = NOW()
		WHERE id = $1::uuid AND integration_id = $2::uuid AND deleted_at IS NULL
		RETURNING ` + toolColumns
	sqlRow := tx.QueryRowContext(ctx, query, toolID, integrationID, definition.Name, definition.Description, schema, nextVersion)
	record, err := ScanTool(sqlRow)
	return record, "updated", err
}

func insertToolVersion(ctx context.Context, tx *sql.Tx, record *tools.DefinitionRecord, changeType string) error {
	schema, err := json.Marshal(record.Definition.Parameters)
	if err != nil {
		return fmt.Errorf("encode versioned tool schema: %w", err)
	}
	insertQ := `
		INSERT INTO tool_definition_versions
			(tool_id, version, name, description, input_schema, enabled, change_type, changed_at)
		VALUES ($1::uuid, $2, $3, $4, $5::jsonb, $6, $7, $8)
	`
	_, err = tx.ExecContext(ctx, insertQ, record.ID, record.Version, record.Definition.Name, record.Definition.Description,
		schema, record.Enabled, changeType, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("persist tool definition version: %w", err)
	}
	return nil
}
