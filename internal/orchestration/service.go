package assistant

import (
	"context"

	"github.com/mtariq99/dispatchai/models"
)

// This file is the entry point for the example Host Project.
//
// It demonstrates how an existing application can integrate with the
// generic AI Assistant.
//
// The example project owns the actual business operations and exposes
// selected capabilities as tools.
//
// For example:
//
//   find_delayed_orders
//   get_customer
//   get_customer_preferences
//   send_notification
//
// The AI Assistant never directly accesses this project's database.
//
// The flow is:
//
//   AI Assistant
//        ↓
//   Tool Request
//        ↓
//   Example Host Project
//        ↓
//   Business Service / Database
//        ↓
//   Tool Result
//
// This example should behave like a real external project integration,
// not like an implementation of business logic inside DispatchAI.

// Assistant is the only entrypoint the API layer (Phase 9) is allowed
// to call. Everything else in this package is internal machinery.
type Assistant struct {
	orchestrator *Orchestrator
}

func NewAssistant(orchestrator *Orchestrator) *Assistant {
	return &Assistant{orchestrator: orchestrator}
}

func (a *Assistant) HandleRequest(ctx context.Context, userMessage string, execCtx models.ExecutionContext) (string, error) {
	return a.orchestrator.Run(ctx, &execCtx, userMessage)
}
