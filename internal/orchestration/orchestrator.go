package assistant

import (
	"context"
	"fmt"

	"github.com/mtariq99/dispatchai/internal/conversation"
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
	store        conversation.Store
}

func NewOrchestrator(llm llm.Client, registry *tools.Registry, executer *tools.Executer, model string, maxTokens int, temperature float64, store conversation.Store) *Orchestrator {
	return &Orchestrator{
		toolloop:     NewToolLoop(llm, registry, executer, model, maxTokens, temperature),
		systemPrompt: DefaultSystemPrompt,
		store:        store,
	}
}

func (o *Orchestrator) Run(ctx context.Context, execCtx *models.ExecutionContext, userQuery string) (string, error) {
	fmt.Println("================================================")
	fmt.Println("ORCHESTRATOR RUN")
	fmt.Println("conversation:", execCtx.ConversationID)
	fmt.Println("user:", execCtx.Identity.UserID)
	fmt.Println("query:", userQuery)
	fmt.Println("================================================")
	priorMessages, err := o.store.Load(execCtx.ConversationID)
	if err != nil {
		return "", fmt.Errorf("load conversation Error : %w", err)
	}
	fmt.Println("LOADED MESSAGES:", len(priorMessages))

	for i, msg := range priorMessages {
		fmt.Printf(
			"[%d] role=%s content=%q toolCallID=%q toolCalls=%d\n",
			i,
			msg.Role,
			msg.Content,
			msg.ToolCallID,
			len(msg.ToolCalls),
		)
	}
	conv := &conversation.Conversation{Messages: priorMessages}

	if len(priorMessages) == 0 && o.systemPrompt != "" {
		conv.AddSystemMessage(o.systemPrompt)
	}
	conv.AddUserMessage(userQuery)
	fmt.Println("AFTER ADD USER MESSAGE:", len(conv.Messages))

	for i, msg := range conv.Messages {
		fmt.Printf(
			"[%d] role=%s content=%q toolCallID=%q toolCalls=%d\n",
			i,
			msg.Role,
			msg.Content,
			msg.ToolCallID,
			len(msg.ToolCalls),
		)
	}
	answer, err := o.toolloop.Run(ctx, conv, execCtx)
	if err != nil {
		return "", err
	}
	if saveErr := o.store.Save(execCtx.ConversationID, execCtx.Tenant.TenantID, execCtx.Identity.UserID, conv.Messages); saveErr != nil {
		return "", saveErr
	}
	return answer, nil
}
