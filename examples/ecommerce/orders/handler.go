package orders

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

type Orders struct {
	config *models.Config
}

func NewOrders(cfg *models.Config) *Orders {
	return &Orders{config: cfg}
}

// FindDelayedCustomers is a stub — it returns a fixed dummy dataset
// instead of querying a real order service, so we can prove the
// multi-tool loop (Phase 12) without building real business logic yet.
func (o *Orders) FindDelayedCustomers(ctx context.Context, req *models.ToolRequest) (*models.ToolResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("tool request cannot be nil")
	}

	if req.ToolName != "find_delayed_customers" {
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
		if permission == "order.read" {
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
				Message:   "user is not authorized to read orders",
				Retryable: false,
			},
		}, nil
	}

	// 4. Decode tool arguments
	var arguments struct {
		DelayMinutes int `json:"delay_minutes"`
	}

	if err := json.Unmarshal(req.Arguments, &arguments); err != nil {
		return &models.ToolResponse{
			RequestID: req.RequestID,
			CallID:    req.CallID,
			Success:   false,
			Error: &models.ToolError{
				Code:      "INVALID_ARGUMENTS",
				Message:   "invalid find_delayed_customers arguments",
				Retryable: false,
			},
		}, nil
	}

	if arguments.DelayMinutes <= 0 {
		arguments.DelayMinutes = 30
	}

	// 5. Execute actual business operation
	//
	// orders, err := o.orderService.FindDelayed(ctx, req.Tenant.TenantID, arguments.DelayMinutes)
	//
	// Replaced with a hardcoded dummy dataset for now.
	delayedCustomers := []map[string]any{
		{
			"customer_id":   "cust_002",
			"customer_name": "Brian Smith",
			"email":         "brian.smith@example.com",
			"order_id":      "ORD-1001",
			"delay_minutes": 47,
		},
		{
			"customer_id":   "cust_005",
			"customer_name": "Emma Wilson",
			"email":         "emma.wilson@example.com",
			"order_id":      "ORD-1007",
			"delay_minutes": 62,
		},
	}

	result := map[string]any{
		"delayed_customers": delayedCustomers,
		"delay_minutes":     arguments.DelayMinutes,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal delayed customers result: %w", err)
	}

	// 6. Return structured ToolResponse
	return &models.ToolResponse{
		RequestID: req.RequestID,
		CallID:    req.CallID,
		Success:   true,
		Data:      data,
	}, nil
}
