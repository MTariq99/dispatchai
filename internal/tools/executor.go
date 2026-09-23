package tools

import (
	"context"

	"github.com/mtariq99/dispatchai/internal/project"
	"github.com/mtariq99/dispatchai/models"
)

//
// It does NOT execute tools directly.
//
// The execution path is:
//
//   LLM
//      ↓
//   Tool Call
//      ↓
//   Tool Executor
//      ↓
//   Project Client
//      ↓
//   Host Project
//      ↓
//   Business Service / Database
//      ↓
//   Tool Result
//
// The executor validates the call, sends it across the project boundary,
// handles communication errors, and converts the response into the
// internal ToolResult representation.
//
// This boundary is critical to keeping DispatchAI domain-agnostic.

type Executer struct {
	registry    *Registry
	toolGateWay project.ToolGateway
}

func NewExecuter(registry *Registry, tg project.ToolGateway) *Executer {
	return &Executer{
		registry:    registry,
		toolGateWay: tg,
	}
}

func (e *Executer) Execute(ctx context.Context, call *models.Call, excCtx *models.ExecutionContext) *models.Result {
	if call.Name == "" {
		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_TOOL_CALL",
				Message: "tool name cannot be empty",
			},
		}
	}

	if !e.registry.Has(call.Name) {
		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:    "UNKNOWN_TOOL",
				Message: "tool \"" + call.Name + "\" is not available in this request",
			},
		}
	}
	return e.toolGateWay.Execute(ctx, *call, *excCtx)
}
