package assistant

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/internal/conversation"
	"github.com/mtariq99/dispatchai/models"
)

func (tl *ToolLoop) BuildRequest(conv *conversation.Conversation, integrationID uuid.UUID) (*models.Request, error) {
	if integrationID == uuid.Nil {
		return nil, errors.New("integration ID is required")
	}
	definitionRecords, err := tl.toolService.GetToolsList(context.Background(), integrationID)
	if err != nil {
		return nil, errors.New(("Error getting enabled tools list for build request: " + err.Error()))
	}

	toolDefinitions := make([]models.ToolDefinition, 0, len(definitionRecords))
	for _, def := range definitionRecords {
		toolDefinitions = append(toolDefinitions, models.ToolDefinition(def.Definition))
	}
	return &models.Request{
		Model:       tl.model,
		MaxTokens:   tl.maxTokens,
		Temperature: tl.temperature,
		Messages:    conv.Messages,
		Tools:       toolDefinitions,
	}, nil
}
