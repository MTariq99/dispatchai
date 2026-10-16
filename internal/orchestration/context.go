package assistant

import (
	"github.com/mtariq99/dispatchai/internal/conversation"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

func BuildRequest(conv *conversation.Conversation, registry *tools.Registry, model string, maxTokens int, temperature float64) *models.Request {
	definitions := registry.Definitions()

	toolDefinitions := make([]models.ToolDefinition, 0, len(definitions))
	for _, def := range definitions {
		toolDefinitions = append(toolDefinitions, models.ToolDefinition(def))
	}
	return &models.Request{
		Model:       model,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		Messages:    conv.Messages,
		Tools:       toolDefinitions,
	}
}
