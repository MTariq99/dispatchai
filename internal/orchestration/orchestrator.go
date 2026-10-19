package assistant

import (
	"context"
	"fmt"

	"github.com/mtariq99/dispatchai/internal/conversation"
	"github.com/mtariq99/dispatchai/internal/llm"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

type Orchestrator struct {
	toolloop     *ToolLoop
	systemPrompt string
	store        *conversation.ConversationService
}

func NewOrchestrator(llm llm.Client, toolService *tools.RegistrationService, executer *tools.Executer, model string, maxTokens int, temperature float64, store *conversation.ConversationService) *Orchestrator {
	return &Orchestrator{
		toolloop:     NewToolLoop(llm, toolService, executer, model, maxTokens, temperature),
		systemPrompt: DefaultSystemPrompt,
		store:        store,
	}
}

func (o *Orchestrator) Run(ctx context.Context, execCtx *models.ExecutionContext, userQuery string) (answer string, retErr error) {
	if o == nil || o.store == nil {
		return "", fmt.Errorf("conversation store is not configured")
	}
	if execCtx == nil {
		return "", fmt.Errorf("execution context is required")
	}

	unlock, err := o.store.LockConversation(ctx, execCtx.ConversationID)
	if err != nil {
		return "", fmt.Errorf("lock conversation: %w", err)
	}
	defer func() {
		if unlockErr := unlock(); unlockErr != nil && retErr == nil {
			answer = ""
			retErr = fmt.Errorf("release conversation lock: %w", unlockErr)
		}
	}()

	priorMessages, err := o.store.LoadConversation(execCtx.ConversationID, execCtx.IntegrationID)
	if err != nil {
		return "", fmt.Errorf("load conversation Error : %w", err)
	}

	conv := &conversation.Conversation{Messages: priorMessages}

	if len(priorMessages) == 0 && o.systemPrompt != "" {
		conv.AddSystemMessage(o.systemPrompt)
	}
	conv.AddUserMessage(userQuery)
	answer, err = o.toolloop.Run(ctx, conv, execCtx)
	if err != nil {
		return "", err
	}
	if saveErr := o.store.SaveConversation(execCtx.ConversationID, execCtx.IntegrationID, conv.Messages); saveErr != nil {
		return "", saveErr
	}
	return answer, nil
}
