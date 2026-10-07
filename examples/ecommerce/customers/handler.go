package customers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mtariq99/dispatchai/models"
)

type Customers struct {
	config *models.Config
}

func NewCustomers(cfg *models.Config) *Customers {
	return &Customers{
		config: cfg,
	}
}

func (c *Customers) GetCustomers(ctx context.Context, req *models.ToolRequest) (*models.ToolResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("tool request cannot be nil")
	}

	if req.ToolName != "get_customers" {
		return nil, fmt.Errorf("unexpected tool: %s", req.ToolName)
	}

	// 1. Validate identity
	if req.Identity.UserID == "" {
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
	if req.Tenant.TenantID == "" {
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
		if permission == "customer.read" {
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
				Message:   "user is not authorized to read customers",
				Retryable: false,
			},
		}, nil
	}

	// 4. Decode tool arguments
	var arguments struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}

	if err := json.Unmarshal(req.Arguments, &arguments); err != nil {
		return &models.ToolResponse{
			RequestID: req.RequestID,
			CallID:    req.CallID,
			Success:   false,
			Error: &models.ToolError{
				Code:      "INVALID_ARGUMENTS",
				Message:   "invalid get_customers arguments",
				Retryable: false,
			},
		}, nil
	}

	// 5. Apply Host Project business validation
	if arguments.Limit <= 0 {
		arguments.Limit = 50
	}
	if arguments.Limit > 100 {
		arguments.Limit = 100
	}
	if arguments.Offset < 0 {
		arguments.Offset = 0
	}

	allCustomers := []map[string]any{
		{"id": "cust_001", "name": "Alice Johnson", "email": "alice.johnson@example.com"},
		{"id": "cust_002", "name": "Brian Smith", "email": "brian.smith@example.com"},
		{"id": "cust_003", "name": "Carla Mendes", "email": "carla.mendes@example.com"},
		{"id": "cust_004", "name": "David Lee", "email": "david.lee@example.com"},
		{"id": "cust_005", "name": "Emma Wilson", "email": "emma.wilson@example.com"},
		{"id": "cust_006", "name": "Farhan Ali", "email": "farhan.ali@example.com"},
		{"id": "cust_007", "name": "Grace Kim", "email": "grace.kim@example.com"},
		{"id": "cust_008", "name": "Hassan Raza", "email": "hassan.raza@example.com"},
	}

	start := arguments.Offset
	if start > len(allCustomers) {
		start = len(allCustomers)
	}
	end := start + arguments.Limit
	if end > len(allCustomers) {
		end = len(allCustomers)
	}
	pagedCustomers := allCustomers[start:end]

	result := map[string]any{
		"customers": pagedCustomers,
		"limit":     arguments.Limit,
		"offset":    arguments.Offset,
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal customer result: %w", err)
	}

	return &models.ToolResponse{
		RequestID: req.RequestID,
		CallID:    req.CallID,
		Success:   true,
		Data:      data,
	}, nil
}
