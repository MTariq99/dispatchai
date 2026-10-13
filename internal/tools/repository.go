package tools

import (
	"context"
	"time"

	"github.com/mtariq99/dispatchai/models"
)

type DefinitionRecord struct {
	ID            string
	IntegrationID string
	Definition    models.Definition
	Version       int
	Enabled       bool
	DeletedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Repository interface {
	Create(ctx context.Context, integrationID string, definition models.Definition, idempotencyKey string, requestHash []byte) (record DefinitionRecord, replayed bool, err error)
	Get(ctx context.Context, integrationID string, toolID string) (DefinitionRecord, error)
	ListEnabled(ctx context.Context, integrationID string) ([]DefinitionRecord, error)
	Update(ctx context.Context, integrationID string, toolID string, expectedVersion int, definition models.Definition) (DefinitionRecord, error)
	SetEnabled(ctx context.Context, integrationID string, toolID string, expectedVersion int, enabled bool) (DefinitionRecord, error)
	SoftDelete(ctx context.Context, integrationID string, toolID string, expectedVersion int) error
	GetForExecution(ctx context.Context, integrationID string, toolID string, expectedVersion int) (DefinitionRecord, error)
}
