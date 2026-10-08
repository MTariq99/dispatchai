package notifications

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

type Notifications struct {
	config *models.Config
}

func NewNotifications(cfg *models.Config) *Notifications {
	return &Notifications{config: cfg}
}

// SendNotification is a stub — it "sends" a notification by just
// logging it and returning success, rather than calling a real
// notification service, so we can prove the multi-tool loop end to end.
func (n *Notifications) SendNotification(ctx context.Context, req *models.ToolRequest) (*models.ToolResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("tool request cannot be nil")
	}

	if req.ToolName != "send_notification" {
		return nil, fmt.Errorf("unexpected tool: %s", req.ToolName)
	}

	// 1. Validate identity
	if req.Identity.UserID == uuid.Nil {
		return &models.ToolResponse{
			RequestID: req.RequestID,
			CallID:    req.CallID,
			Success:   false,
			Error: &models.ToolError{
				Code:      "UNAUTHENTICATED",
				Message:   "user identity is required",
				Retryable: false,
			},
		}, nil
	}

	// 2. Validate tenant
	if req.Tenant.TenantID == uuid.Nil {
		return &models.ToolResponse{
			RequestID: req.RequestID,
			CallID:    req.CallID,
			Success:   false,
			Error: &models.ToolError{
				Code:      "TENANT_REQUIRED",
				Message:   "tenant context is required",
				Retryable: false,
			},
		}, nil
	}

	// 3. Host Project authorization
	authorized := false
	for _, permission := range req.Authorization.Permissions {
		if permission == "notification.send" {
			authorized = true
			break
		}
	}

	if !authorized {
		return &models.ToolResponse{
			RequestID: req.RequestID,
			CallID:    req.CallID,
			Success:   false,
			Error: &models.ToolError{
				Code:      "FORBIDDEN",
				Message:   "user is not authorized to send notifications",
				Retryable: false,
			},
		}, nil
	}

	// 4. Decode tool arguments
	var arguments struct {
		CustomerID string `json:"customer_id"`
		Message    string `json:"message"`
	}

	if err := json.Unmarshal(req.Arguments, &arguments); err != nil {
		return &models.ToolResponse{
			RequestID: req.RequestID,
			CallID:    req.CallID,
			Success:   false,
			Error: &models.ToolError{
				Code:      "INVALID_ARGUMENTS",
				Message:   "invalid send_notification arguments",
				Retryable: false,
			},
		}, nil
	}

	if arguments.CustomerID == "" {
		return &models.ToolResponse{
			RequestID: req.RequestID,
			CallID:    req.CallID,
			Success:   false,
			Error: &models.ToolError{
				Code:      "INVALID_ARGUMENTS",
				Message:   "customer_id is required",
				Retryable: false,
			},
		}, nil
	}

	// 5. Execute actual business operation
	//
	// err := n.notificationService.Send(ctx, req.Tenant.TenantID, arguments.CustomerID, arguments.Message)
	//
	// Stubbed: just log it and report success.
	fmt.Printf("[STUB NOTIFICATION] tenant=%s customer=%s message=%q\n",
		req.Tenant.TenantID, arguments.CustomerID, arguments.Message)

	result := map[string]any{
		"customer_id": arguments.CustomerID,
		"status":      "sent",
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal notification result: %w", err)
	}

	// 6. Return structured ToolResponse
	return &models.ToolResponse{
		RequestID: req.RequestID,
		CallID:    req.CallID,
		Success:   true,
		Data:      data,
	}, nil
}
