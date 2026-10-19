package storage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	errs "github.com/mtariq99/dispatchai/internal/errors"
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

const toolColumns = `id::text,integration_id::text,name,description,input_schema,version,enabled,created_at,updated_at,deleted_at`

func ScanTool(row interface{ Scan(...any) error }) (*models.DefinitionRecord, error) {
	var record models.DefinitionRecord
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
		return nil, errs.Wrapf(errs.ErrDecodeToolSchema, err)
	}
	return &record, nil
}

func ValidateDefinition(def *models.Definition) ([]byte, error) {
	if def.Name == "" {
		return nil, errs.ErrToolNameRequired
	}
	if len(def.Description) == 0 {
		return nil, errs.ErrToolDescriptionRequired
	}

	if len(def.Name) > 100 {
		return nil, errs.ErrToolNameTooLong
	}
	if def.Parameters == nil {
		return nil, errs.ErrToolSchemaRequired
	}
	if def.Parameters["type"] != "object" {
		return nil, errs.ErrToolSchemaMustBeObject
	}

	schema, err := json.Marshal(def.Parameters)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrDecodeToolSchema, err)
	}
	return schema, nil
}

func (tr *ToolRegistry) CreateToolRegistry(ctx context.Context, integrationID string, defination *models.Definition, idempotencyKey string, requestHash []byte) (*models.DefinitionRecord, bool, error) {
	if tr == nil || tr.db == nil {
		return nil, false, errs.ErrDatabaseNotConfigured
	}
	if integrationID == "" || idempotencyKey == "" || len(requestHash) == 0 {
		return nil, false, errs.ErrFieldsRequired
	}
	schema, err := ValidateDefinition(defination)
	if err != nil {
		return nil, false, err
	}
	tx, err := tr.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, errs.Wrapf(errs.ErrBeginToolRegistration, err)
	}
	defer tx.Rollback()

	var oldHash []byte
	var oldToolID string
	selectQuery := `
	SELECT request_hash,tool_id::text
	FROM tool_registration_idempotency
	WHERE integration_id = $1::uuid AND idempotency_key = $2
	`
	err = tx.QueryRowContext(ctx, selectQuery, integrationID, idempotencyKey).Scan(&oldHash, &oldToolID)
	if err == nil {
		if !bytes.Equal(oldHash, requestHash) {
			return nil, false, errs.ErrIdempotencyConflict
		}
		getRecordQuery := `
			SELECT ` + toolColumns + `
			FROM tool_definitions
			WHERE id = $1::uuid AND integration_id = $2::uuid
	`
		sqlRow := tx.QueryRowContext(ctx, getRecordQuery, oldToolID, integrationID)
		record, getErr := ScanTool(sqlRow)
		if getErr != nil {
			return nil, false, errs.Wrapf(errs.ErrLoadRegisteredTool, err)
		}
		if err := tx.Commit(); err != nil {
			return nil, false, errs.Wrapf(errs.ErrCommitRegistrationReplay, err)
		}
		return record, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, errs.Wrapf(errs.ErrCheckRegistrationIdempotency, err)
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
		return nil, false, errs.Wrapf(errs.ErrIntegrationNotFound, err)
	}
	if integrationStatus != "active" {
		return nil, false, errs.ErrIntegrationInactive
	}
	toolInsertQuery := `
		INSERT INTO tool_definitions
			(integration_id, name, description, input_schema, enabled, version)
		VALUES ($1::uuid, $2, $3, $4::jsonb, TRUE, 1)
		RETURNING ` + toolColumns
	sqlInsertToolRow := tx.QueryRowContext(ctx, toolInsertQuery, integrationID, defination.Name, defination.Description, schema)
	record, err := ScanTool(sqlInsertToolRow)
	if err != nil {
		return nil, false, errs.Wrapf(errs.ErrInsertToolDefinition, err)
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
		return nil, false, errs.Wrapf(errs.ErrCheckRegistrationIdempotency, err)
	}

	if err := tx.Commit(); err != nil {
		return nil, false, errs.Wrapf(errs.ErrCommitToolRegistration, err)
	}
	return record, false, nil
}

func (tr *ToolRegistry) GetTool(ctx context.Context, integrationID, toolID string) (*models.DefinitionRecord, error) {
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

func (r *ToolRegistry) ListEnabled(ctx context.Context, integrationID uuid.UUID) ([]*models.DefinitionRecord, error) {
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
		return nil, errs.Wrapf(errs.ErrListEnabledTools, err)
	}
	defer rows.Close()

	var records []*models.DefinitionRecord
	for rows.Next() {
		record, err := ScanTool(rows)
		if err != nil {
			return nil, errs.Wrapf(errs.ErrScanToolDefinition, err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.Wrapf(errs.ErrIterateToolDefinitions, err)
	}
	return records, nil
}

func (r *ToolRegistry) GetForExecution(ctx context.Context, integrationID, toolID string, expectedVersion int) (*models.DefinitionRecord, error) {
	if r == nil || r.db == nil {
		return nil, errs.ErrDatabaseNotConfigured
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
		return nil, errs.Wrapf(errs.ErrValidateToolForExecution, err)
	}
	if expectedVersion <= 0 || record.Version != expectedVersion {
		return nil, errs.ErrVersionConflict
	}
	return record, nil
}

func (r *ToolRegistry) beginMutation(ctx context.Context, integrationID, toolID string, expectedVersion int) (*sql.Tx, *models.DefinitionRecord, error) {
	if r == nil || r.db == nil {
		return nil, nil, errs.ErrDatabaseNotConfigured
	}
	if expectedVersion <= 0 {
		return nil, nil, errs.ErrVersionConflict
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, errs.Wrapf(errs.ErrBeginToolMutation, err)
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
		return nil, nil, errs.Wrapf(errs.ErrLoadToolForMutation, err)
	}
	if current.Version != expectedVersion {
		tx.Rollback()
		return nil, nil, errs.ErrVersionConflict
	}
	return tx, current, nil
}

func (r *ToolRegistry) commitMutation(ctx context.Context, tx *sql.Tx, record *models.DefinitionRecord, changeType string) (*models.DefinitionRecord, error) {
	if changeType != "" {
		if err := insertToolVersion(ctx, tx, record, changeType); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, errs.Wrapf(errs.ErrCommitToolMutation, err)
	}
	return record, nil
}

func (r *ToolRegistry) Update(ctx context.Context, integrationID, toolID string, expectedVersion int, definition *models.Definition) (*models.DefinitionRecord, error) {
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
			return nil, errs.Wrapf(errs.ErrMutateToolDefinition, err)
		}
		if exists {
			return nil, errs.ErrToolNameConflict
		}
	}

	record, changeType, err := updateDefinition(ctx, tx, integrationID, toolID, definition, schema, current.Version+1)
	if err != nil {
		return nil, errs.Wrapf(errs.ErrMutateToolDefinition, err)
	}
	return r.commitMutation(ctx, tx, record, changeType)
}

func (r *ToolRegistry) SetEnabled(ctx context.Context, integrationID, toolID string, expectedVersion int, enabled bool) (*models.DefinitionRecord, error) {
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
		return nil, errs.Wrapf(errs.ErrMutateToolDefinition, err)
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
		return errs.Wrapf(errs.ErrMutateToolDefinition, err)
	}
	_, err = r.commitMutation(ctx, tx, record, "deleted")
	return err
}

func updateDefinition(ctx context.Context, tx *sql.Tx, integrationID, toolID string, definition *models.Definition, schema []byte, nextVersion int) (*models.DefinitionRecord, string, error) {
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

func insertToolVersion(ctx context.Context, tx *sql.Tx, record *models.DefinitionRecord, changeType string) error {
	schema, err := json.Marshal(record.Definition.Parameters)
	if err != nil {
		return errs.Wrapf(errs.ErrEncodeVersionedToolSchema, err)
	}
	insertQ := `
		INSERT INTO tool_definition_versions
			(tool_id, version, name, description, input_schema, enabled, change_type, changed_at)
		VALUES ($1::uuid, $2, $3, $4, $5::jsonb, $6, $7, $8)
	`
	_, err = tx.ExecContext(ctx, insertQ, record.ID, record.Version, record.Definition.Name, record.Definition.Description,
		schema, record.Enabled, changeType, time.Now().UTC())
	if err != nil {
		return errs.Wrapf(errs.ErrPersistToolDefinitionVersion, err)
	}
	return nil
}
