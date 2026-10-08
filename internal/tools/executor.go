package tools

import (
	"context"
	"fmt"

	"github.com/mtariq99/dispatchai/internal/idempotency"
	"github.com/mtariq99/dispatchai/internal/policy"
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
	registry        *Registry
	toolGateWay     project.ToolGateway
	policyEngine    *policy.Engine
	IdempotentStore *idempotency.IdempotencyStore
}

func NewExecutor(registry *Registry, gateway project.ToolGateway, policyEngine *policy.Engine, idem *idempotency.IdempotencyStore) *Executer {
	return &Executer{
		registry:        registry,
		toolGateWay:     gateway,
		policyEngine:    policyEngine,
		IdempotentStore: idem,
	}
}

func (e *Executer) Execute(ctx context.Context, call *models.Call, excCtx *models.ExecutionContext, callCounts map[string]int) *models.Result {
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
	key := e.IdempotentStore.BuildKey(excCtx.ConversationID, call)

	result, exists, err := e.IdempotentStore.Get(key)
	if err != nil {
		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:      "IDEMPOTENCY_STORE_ERROR",
				Message:   "could not verify whether this operation already ran",
				Retryable: true,
			},
		}
	}
	if exists {
		return result
	}
	reserved, err := e.IdempotentStore.Reserve(key, excCtx.ConversationID, call.Name)
	if !reserved {
		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:      "OPERATION_IN_PROGRESS",
				Message:   "this operation is already being processed",
				Retryable: true,
			},
		}
	}
	decision := e.policyEngine.Evaluate(call, excCtx, callCounts)

	switch decision.Outcome {
	case policy.Deny:
		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:      "POLICY_DENIED",
				Message:   decision.Reason,
				Retryable: false,
			},
		}
	case policy.RequireApproval:
		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:      "APPROVAL_REQUIRED",
				Message:   decision.Reason,
				Retryable: false,
			},
		}
	}
	finalResult := e.toolGateWay.Execute(ctx, *call, *excCtx)

	if setErr := e.IdempotentStore.Set(key, excCtx.ConversationID, call.Name, finalResult); setErr != nil {
		fmt.Printf("warning: failed to store idempotency key %s: %v\n", key, setErr)
	}

	return finalResult
}
