package project

import (
	"context"
	"encoding/json"

	"github.com/mtariq99/dispatchai/models"
)

// This file defines the protocol boundary used when DispatchAI asks
// the Host Project to execute a capability.
//
// The gateway is deliberately generic.
//
// DispatchAI says:
//
//   "Execute tool X with these arguments."
//
// The Host Project decides:
//
//   - whether the caller is authorized
//   - whether the arguments are valid
//   - whether business rules permit the operation
//   - how to access its database/services
//   - how to perform the operation
//
// This file therefore represents the most important integration
// boundary in the system.

type ToolGateway interface {
	Execute(ctx context.Context, call models.Call, execCtx models.ExecutionContext) *models.Result
}

type ProjectToolGateway struct {
	client ProjectClient
}

func NewProjectToolGateway(client ProjectClient) *ProjectToolGateway {
	return &ProjectToolGateway{
		client: client,
	}
}

func (ptg *ProjectToolGateway) Execute(ctx context.Context, call models.Call, execCtx models.ExecutionContext) *models.Result {
	if call.ID == "" {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_TOOL_CALL",
				Message: "tool call ID cannot be empty",
			},
		}
	}
	if call.Name == "" {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_TOOL_CALL",
				Message: "tool call name cannot be empty",
			},
		}
	}
	if !json.Valid(call.Arguments) {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_TOOL_CALL",
				Message: "tool call arguments cannot be empty",
			},
		}
	}
	if execCtx.RequestID == "" {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_TOOL_CALL",
				Message: "tool call Request ID cannot be empty",
			},
		}
	}
	if execCtx.RequestID == "" {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_TOOL_CALL",
				Message: "tool call Conversation ID cannot be empty",
			},
		}
	}
	if execCtx.Identity.UserID == "" {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_TOOL_CALL",
				Message: "tool call User ID cannot be empty",
			},
		}
	}
	if execCtx.Tenant.TenantID == "" {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:    "INVALID_TOOL_CALL",
				Message: "tool call Tenant ID cannot be empty",
			},
		}
	}

	ToolRequest := &models.ToolRequest{
		RequestID:      execCtx.RequestID,
		ConversationID: execCtx.ConversationID,
		ToolName:       call.Name,
		CallID:         call.ID,
		Arguments:      call.Arguments,
		Identity:       execCtx.Identity,
		Tenant:         execCtx.Tenant,
		Authorization:  execCtx.Authorization,
		Trace:          execCtx.Trace,
	}
	resp, err := ptg.client.ExecuteTool(ctx, ToolRequest)
	if err != nil {
		return &models.Result{
			Success: false,
			Error: &models.Error{
				Code:    "Error From Client Project",
				Message: "Error From Client Project : " + err.Error(),
			},
		}
	}
	Res := &models.Result{
		CallID:  resp.CallID,
		Success: true,
		Data:    resp.Data,
		Error:   nil,
	}
	return Res
}
