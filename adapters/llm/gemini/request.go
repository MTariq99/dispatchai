package gemini

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mtariq99/dispatchai/internal/enums"
	"github.com/mtariq99/dispatchai/models"
)

func BuildGeminiRequest(req models.Request, defaultMaxTokens int, defaultTemperature float64) (*models.GeminiRequest, error) {
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("llm request must contain atleast one message")
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}
	if maxTokens < 0 {
		return nil, fmt.Errorf("max tokens cannot be negative")
	}
	temperature := req.Temperature
	if temperature == 0 {
		temperature = defaultTemperature
	}
	if temperature < 0 {
		return nil, fmt.Errorf("temperature cannot be negative")
	}
	result := models.GeminiRequest{
		Contents: make([]models.GeminiContent, 0),
		GenerationConfig: models.GeminiGenerationConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     temperature,
		},
	}

	for _, message := range req.Messages {
		switch message.Role {
		case enums.RoleSystem:
			systemInstruction, err := convertSystemMessage(message)
			if err != nil {
				return nil, err
			}
			result.SystemInstruction = systemInstruction
		case enums.RoleUser:
			Geminicontent, err := convertUserMessage(message)
			if err != nil {
				return nil, err
			}
			result.Contents = append(result.Contents, Geminicontent)
		case enums.RoleAssistant:
			content, err := convertAssistantMessage(message)
			if err != nil {
				return nil, err
			}

			result.Contents = append(result.Contents, content)

		case enums.RoleTool:
			Geminicontent, err := convertToolMessage(req.Messages, message)
			if err != nil {
				return nil, err
			}

			result.Contents = append(result.Contents, Geminicontent)
		default:
			return nil, fmt.Errorf("unsupported message role %q", message.Role)
		}
	}
	if len(req.Tools) > 0 {
		tools := models.GeminiTool{
			FunctionDeclarations: make([]models.GeminiFunctionDeclaration, 0, len(req.Tools)),
		}
		for _, tool := range req.Tools {
			if strings.TrimSpace(tool.Name) == "" {
				return nil, fmt.Errorf("tool name cannot be empty")
			}
			extractedTool := models.GeminiFunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			}
			tools.FunctionDeclarations = append(tools.FunctionDeclarations, extractedTool)
		}
		result.Tools = append(result.Tools, tools)
	}
	return &result, nil
}

func convertSystemMessage(message models.Message) (*models.GeminiContent, error) {
	if strings.TrimSpace(message.Content) == "" {
		return nil, fmt.Errorf("system message cannot be empty")
	}
	return &models.GeminiContent{
		Parts: []models.GeminiPart{
			{
				Text: message.Content,
			},
		},
	}, nil
}

func convertUserMessage(message models.Message) (models.GeminiContent, error) {
	return models.GeminiContent{
		Role: "user",
		Parts: []models.GeminiPart{
			{
				Text: message.Content,
			},
		},
	}, nil
}

func convertAssistantMessage(message models.Message) (models.GeminiContent, error) {
	content := models.GeminiContent{
		Role:  "model",
		Parts: make([]models.GeminiPart, 0),
	}

	if message.Content != "" {
		content.Parts = append(
			content.Parts,
			models.GeminiPart{
				Text: message.Content,
			},
		)
	}

	for _, toolCall := range message.ToolCalls {
		if strings.TrimSpace(toolCall.Name) == "" {
			return models.GeminiContent{}, errors.New(
				"assistant tool call name cannot be empty",
			)
		}

		content.Parts = append(
			content.Parts,
			models.GeminiPart{
				FunctionCall: &models.GeminiFunctionCall{
					ID:   toolCall.ID,
					Name: toolCall.Name,
					Args: toolCall.Args,
				},
			},
		)
	}

	if len(content.Parts) == 0 {
		return models.GeminiContent{}, errors.New(
			"assistant message contains neither content nor tool calls",
		)
	}

	return content, nil
}

func convertToolMessage(messages []models.Message, message models.Message) (models.GeminiContent, error) {
	if strings.TrimSpace(message.ToolCallID) == "" {
		return models.GeminiContent{}, errors.New(
			"tool message requires tool call ID",
		)
	}

	toolName, found := findToolName(
		messages,
		message.ToolCallID,
	)

	if !found {
		return models.GeminiContent{}, fmt.Errorf("cannot find tool name for tool call %q", message.ToolCallID)
	}

	response := map[string]any{}

	if message.Content != "" {
		if err := json.Unmarshal([]byte(message.Content), &response); err != nil {
			response["content"] = message.Content
		}
	}

	return models.GeminiContent{
		Role: "user",
		Parts: []models.GeminiPart{
			{
				FunctionResponse: &models.GeminiFunctionResponse{
					ID:       message.ToolCallID,
					Name:     toolName,
					Response: response,
				},
			},
		},
	}, nil
}

func findToolName(messages []models.Message, toolCallID string) (string, bool) {
	for _, message := range messages {
		if message.Role != enums.RoleAssistant {
			continue
		}

		for _, toolCall := range message.ToolCalls {
			if toolCall.ID == toolCallID {
				return toolCall.Name, true
			}
		}
	}

	return "", false
}
