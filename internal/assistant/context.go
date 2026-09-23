package assistant

import (
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

// This file builds the context supplied to the LLM.
//
// Context may include:
//
//   - authenticated user identity
//   - tenant information
//   - roles or permission context
//   - conversation history
//   - request metadata
//   - available Host Project capabilities
//   - optional knowledge retrieved through RAG
//
// The identity and authorization context comes from the Host Project.
// The LLM must never be allowed to invent or modify those values.
//
// This file prepares information for reasoning; it does not make
// authorization decisions.

func BuildRequest(conv *Conversation, registry *tools.Registry, model string, maxTokens int, temperature float64) *models.Request {
	ToolsDefinations := registry.Definitions()
	toolDef := make([]models.ToolDefinition, 0, len(ToolsDefinations))
	for _, def := range ToolsDefinations {
		toolDef = append(toolDef, models.ToolDefinition{
			Name:        def.Name,
			Description: def.Description,
			Parameters:  def.Parameters,
		})
	}
	return &models.Request{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Messages:    conv.Messages,
		Tools:       toolDef,
	}
}
