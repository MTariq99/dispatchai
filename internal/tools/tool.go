package tools

import (
	"context"

	"github.com/mtariq99/dispatchai/models"
)

// This file defines the generic representation of a tool capability.
//
// IMPORTANT:
//
// A Tool in DispatchAI is a description of a capability provided by
// the Host Project.
//
// It is NOT necessarily the implementation of that capability.
//
// For example:
//
//   Name:
//       find_delayed_orders
//
//   Description:
//       Find orders delayed by a specified number of minutes.
//
// The Host Project owns the actual implementation.
//
// DispatchAI only needs enough information to let the LLM understand
// what capability is available.

type Tool interface {
	Name() string
	Definition() models.Definition
	Execute(ctx context.Context, call models.Call) models.Result
}
