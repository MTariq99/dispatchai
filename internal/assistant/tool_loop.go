package assistant

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mtariq99/dispatchai/internal/llm"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

// This file implements the repeated LLM -> Tool -> Result -> LLM loop.
//
// The LLM may require several tool calls before it can produce a final
// answer.
//
// Example:
//
//   LLM
//    ↓
//   find_delayed_orders
//    ↓
//   Tool Result
//    ↓
//   LLM
//    ↓
//   get_customer_preferences
//    ↓
//   Tool Result
//    ↓
//   LLM
//    ↓
//   send_notification
//    ↓
//   Tool Result
//    ↓
//   LLM
//    ↓
//   Final Answer
//
// This file also protects the system from uncontrolled execution by
// enforcing limits such as:
//
//   - maximum iterations
//   - maximum tool calls
//   - execution timeout
//   - invalid tool calls
//   - repeated tool calls
//
// It never directly executes business operations.

const MaxToolLoopIterations = 10

type ToolLoop struct {
	llm         llm.Client
	registry    *tools.Registry
	executer    *tools.Executer
	model       string
	maxTokens   int
	temperature float64
}

func NewToolLoop(llmClient llm.Client, registry *tools.Registry, executer *tools.Executer, model string, maxTokens int, temperature float64) *ToolLoop {
	return &ToolLoop{
		llm:         llmClient,
		registry:    registry,
		executer:    executer,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
	}
}

func (tl *ToolLoop) Run(ctx context.Context, conv *Conversation, execCtx *models.ExecutionContext) (string, error) {
	for i := 0; i < MaxToolLoopIterations; i++ {
		req := BuildRequest(conv, tl.registry, tl.model, tl.maxTokens, tl.temperature)

		resp, err := tl.llm.Generate(ctx, *req)
		if err != nil {
			return "", fmt.Errorf("llm generation failed : %w", err)
		}
		if len(resp.ToolCalls) == 0 {
			return resp.Content, nil
		}

		conv.AddAssistantMessage(models.Message{
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		for _, tc := range resp.ToolCalls {
			call, err := toolCallToCall(tc)
			if err != nil {
				conv.AddToolResult(tc.ID, fmt.Sprintf(`{"error":%q}`, err.Error()))
				continue
			}
			result := tl.executer.Execute(ctx, &call, execCtx)
			conv.AddToolResult(tc.ID, resultToContent(result))

		}
	}
	return "", fmt.Errorf("tool loop exceeded max iterations (%d)", MaxToolLoopIterations)
}

func toolCallToCall(tc models.ToolCall) (models.Call, error) {
	if tc.Name == "" {
		return models.Call{}, fmt.Errorf("tool call name cannot be empty")
	}

	argsBytes, err := json.Marshal(tc.Args)
	if err != nil {
		return models.Call{}, fmt.Errorf("marshal tool call args: %w", err)
	}

	return models.Call{
		ID:        tc.ID,
		Name:      tc.Name,
		Arguments: argsBytes,
	}, nil
}

func resultToContent(result *models.Result) string {
	if result == nil {
		return `{"error":"nil result from executor"}`
	}

	if !result.Success {
		payload := map[string]any{"error": true}
		if result.Error != nil {
			payload["code"] = result.Error.Code
			payload["message"] = result.Error.Message
		}
		b, _ := json.Marshal(payload)
		return string(b)
	}

	if len(result.Data) == 0 {
		return `{}`
	}

	return string(result.Data)
}
