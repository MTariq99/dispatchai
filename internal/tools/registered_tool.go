package tools

import (
	"context"
	"fmt"

	"github.com/mtariq99/dispatchai/models"
)

type RegisteredTool struct {
	record DefinitionRecord
}

var _ Tool = (*RegisteredTool)(nil)

func NewRegisteredTool(record *DefinitionRecord) *RegisteredTool {
	if record == nil {
		return nil
	}
	return &RegisteredTool{record: *record}
}

func (t *RegisteredTool) Name() string {
	return t.record.Definition.Name
}

func (t *RegisteredTool) Definition() models.Definition {
	return t.record.Definition
}

func (t *RegisteredTool) Record() DefinitionRecord {
	return t.record
}

func (t *RegisteredTool) Execute(ctx context.Context, call models.Call) models.Result {
	return models.Result{
		CallID:  call.ID,
		Success: false,
		Error: &models.Error{
			Code:      "INVALID_EXECUTION_PATH",
			Message:   fmt.Sprintf("tool %q must execute through the tool executor", t.Name()),
			Retryable: false,
		},
	}
}
