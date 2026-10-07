package gemini

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mtariq99/dispatchai/models"
)

func parseGeminiResponse(raw []byte) (*models.Response, error) {
	var response models.GeminiResponse
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, fmt.Errorf("error decoding gemini response: %w", err)
	}
	if len(response.Candidates) == 0 {
		return nil, fmt.Errorf("gemini returned no candidates")
	}
	candidate := response.Candidates[0]
	if candidate == nil {
		return nil, fmt.Errorf("gemini returned nil candidate")
	}
	if candidate.Content == nil {
		return nil, fmt.Errorf("gemini candidate contains no content: finish_reason=%s", candidate.FinishReason)
	}

	result := models.Response{
		StopReason: candidate.FinishReason,
	}

	var textParts []string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
		}
		if part.FunctionCall != nil {
			id := part.FunctionCall.ID
			if strings.TrimSpace(id) == "" {
				id = generateToolCallID()
			}
			result.ToolCalls = append(result.ToolCalls, models.ToolCall{
				ID:   id,
				Name: part.FunctionCall.Name,
				Args: part.FunctionCall.Args,
			})
		}
	}

	result.Content = strings.Join(textParts, "\n")

	if response.UsageMetadata != nil {
		result.Usage = models.Usage{
			InputTokens:  response.UsageMetadata.PromptTokenCount,
			OutputTokens: response.UsageMetadata.CandidatesTokenCount,
			TotalTokens:  response.UsageMetadata.TotalTokenCount,
		}
	}

	if result.Content == "" && len(result.ToolCalls) == 0 {
		return nil, fmt.Errorf("gemini returned neither text nor tool calls: finish_reason=%s", candidate.FinishReason)
	}

	return &result, nil
}

func generateToolCallID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "call_" + hex.EncodeToString(b)
}
