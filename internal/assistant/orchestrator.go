package assistant

import (
	"context"

	"github.com/mtariq99/dispatchai/internal/llm"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

// This file contains the main orchestration logic of the AI Assistant.
//
// It coordinates the complete reasoning cycle between the LLM,
// conversation state, tool executor, and Host Project.
//
// The normal flow is:
//
//   User Request
//        ↓
//   Build Context
//        ↓
//   Send Request + Tools to LLM
//        ↓
//   LLM returns Tool Call
//        ↓
//   Forward Tool Call to Host Project
//        ↓
//   Receive Tool Result
//        ↓
//   Give Result back to LLM
//        ↓
//   More Tool Calls OR Final Answer
//
// This is the central control plane of the Assistant.
//
// It does NOT execute business operations itself.
//
// The Host Project remains responsible for:
//   - authorization
//   - business validation
//   - database access
//   - business services
//   - external business APIs
//   - actual notification delivery

type Orchestrator struct {
	toolloop     *ToolLoop
	systemPrompt string
}

func NewOrchestrator(llm llm.Client, registry *tools.Registry, executer *tools.Executer, model string, maxTokens int, temperature float64) *Orchestrator {
	return &Orchestrator{
		toolloop:     NewToolLoop(llm, registry, executer, model, maxTokens, temperature),
		systemPrompt: DefaultSystemPrompt,
	}
}

func (o *Orchestrator) Run(ctx context.Context, execCtx *models.ExecutionContext, userQuery string) (string, error) {
	conv := NewConversation()
	if o.systemPrompt != "" {
		conv.AddSystemMessage(o.systemPrompt)
	}
	conv.AddUserMessage(userQuery)
	return o.toolloop.Run(ctx, conv, execCtx)
}
