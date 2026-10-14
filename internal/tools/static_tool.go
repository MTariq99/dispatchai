package tools

import (
	"context"

	"github.com/mtariq99/dispatchai/models"
)

type StaticTool struct {
	definition models.Definition
}

func NewStaticTool(def models.Definition) *StaticTool {
	return &StaticTool{definition: def}
}

func (t *StaticTool) Name() string {
	return t.definition.Name
}

func (t *StaticTool) Definition() models.Definition {
	return t.definition
}

func (t *StaticTool) Execute(ctx context.Context, call models.Call) models.Result {
	return models.Result{
		CallID:  call.ID,
		Success: false,
		Error: &models.Error{
			Code:    "NOT_DIRECTLY_EXECUTABLE",
			Message: "this tool must be executed through Executor, not directly",
		},
	}
}
