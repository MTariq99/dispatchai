package tools

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/mtariq99/dispatchai/internal/idempotency"
	"github.com/mtariq99/dispatchai/internal/policy"
	"github.com/mtariq99/dispatchai/internal/project"
	"github.com/mtariq99/dispatchai/models"
)

type Executer struct {
	registry        *Registry
	toolGateWay     project.ToolGateway
	policyEngine    *policy.Engine
	IdempotentStore idempotency.ToolIdempotency
}

func NewExecutor(registry *Registry, gateway project.ToolGateway, policyEngine *policy.Engine, idem idempotency.ToolIdempotency) *Executer {
	return &Executer{
		registry:        registry,
		toolGateWay:     gateway,
		policyEngine:    policyEngine,
		IdempotentStore: idem,
	}
}

func (e *Executer) Execute(ctx context.Context, call *models.Call, excCtx *models.ExecutionContext, callCounts map[string]int) *models.Result {
	if e == nil {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:      "EXECUTOR_NOT_CONFIGURED",
				Message:   "tool executor is not configured",
				Retryable: false,
			},
		}
	}
	if e.registry == nil || e.toolGateWay == nil || e.policyEngine == nil || e.IdempotentStore == nil {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:      "EXECUTOR_NOT_CONFIGURED",
				Message:   "tool executor dependencies are not fully configured",
				Retryable: false,
			},
		}
	}
	if call == nil {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_TOOL_CALL",
				Message: "tool call cannot be nil",
			},
		}
	}

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

	if call.ID == "" {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:      "INVALID_TOOL_CALL",
				Message:   "tool call ID cannot be empty",
				Retryable: false,
			},
		}
	}

	if excCtx == nil {
		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_EXECUTION_CONTEXT",
				Message: "execution context cannot be nil",
			},
		}
	}

	if excCtx.RunID == uuid.Nil {
		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_EXECUTION_CONTEXT",
				Message: "run id cannot be empty",
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

	result, exists, err := e.IdempotentStore.Get(ctx, excCtx.RunID, call.ID)
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

	reserved, err := e.IdempotentStore.Reserve(ctx, excCtx.RunID, call.ID, call.Name, call.Arguments)
	if err != nil {
		if errors.Is(err, idempotency.ErrToolCallConflict) {
			return &models.Result{
				CallID:  call.ID,
				Success: false,
				Error: &models.Error{
					Code:      "TOOL_CALL_CONFLICT",
					Message:   "call ID was reused with different tool name or arguments",
					Retryable: false,
				},
			}
		}

		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:      "IDEMPOTENCY_STORE_ERROR",
				Message:   "could not reserve this operation",
				Retryable: true,
			},
		}
	}

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
		result := &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:      "POLICY_DENIED",
				Message:   decision.Reason,
				Retryable: false,
			},
		}

		if err := e.IdempotentStore.Fail(ctx, excCtx.RunID, call.ID, result.Error); err != nil {
			return &models.Result{
				CallID:  call.ID,
				Success: false,
				Error: &models.Error{
					Code: "TOOL_POLICY_PERSISTENCE_FAILED",
					Message: "policy denied the tool call, but the decision could not be persisted; " +
						"the operation must not be automatically re-executed until reconciled",
					Retryable: false,
				},
			}
		}

		return result

	case policy.RequireApproval:
		result := &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:      "APPROVAL_REQUIRED",
				Message:   decision.Reason,
				Retryable: false,
			},
		}

		if err := e.IdempotentStore.Fail(ctx, excCtx.RunID, call.ID, result.Error); err != nil {
			return &models.Result{
				CallID:  call.ID,
				Success: false,
				Error: &models.Error{
					Code: "TOOL_POLICY_PERSISTENCE_FAILED",
					Message: "approval is required, but the decision could not be persisted; " +
						"the operation must not be automatically re-executed until reconciled",
					Retryable: false,
				},
			}
		}

		return result
	}

	finalResult := e.toolGateWay.Execute(ctx, *call, *excCtx)

	if finalResult == nil {
		finalResult = &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code:      "TOOL_EXECUTION_FAILED",
				Message:   "tool gateway returned nil result",
				Retryable: true,
			},
		}
	}

	if finalResult.Success {
		if err := e.IdempotentStore.Complete(ctx, excCtx.RunID, call.ID, finalResult); err != nil {
			return &models.Result{
				CallID:  call.ID,
				Success: false,
				Error: &models.Error{
					Code: "TOOL_RESULT_PERSISTENCE_FAILED",
					Message: "tool execution may have succeeded, but its result " +
						"could not be persisted; automatic re-execution is unsafe",
					Retryable: false,
				},
			}
		}

		return finalResult
	}

	if finalResult.Error == nil {
		finalResult.Error = &models.Error{
			Code:      "TOOL_EXECUTION_FAILED",
			Message:   "tool gateway returned an unsuccessful result without an error",
			Retryable: false,
		}
	}

	if err := e.IdempotentStore.Fail(ctx, excCtx.RunID, call.ID, finalResult.Error); err != nil {
		return &models.Result{
			CallID:  call.ID,
			Success: false,
			Error: &models.Error{
				Code: "TOOL_FAILURE_PERSISTENCE_FAILED",
				Message: "tool execution failed, but its failure state could not be persisted; " +
					"the operation must not be automatically re-executed until reconciled",
				Retryable: false,
			},
		}
	}

	return finalResult
}
