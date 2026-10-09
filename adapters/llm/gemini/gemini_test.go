package gemini

import (
	"testing"

	"github.com/mtariq99/dispatchai/models"
)

func TestNewGeminiClientRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name string
		cfg  *models.Config
	}{
		{
			name: "nil config",
			cfg:  nil,
		},
		{
			name: "missing generate URL",
			cfg: &models.Config{
				LLM: models.LLMConfig{
					GeminiAPIKey: "test-key",
					MaxTokens:    100,
				},
			},
		},
		{
			name: "missing API key",
			cfg: &models.Config{
				LLM: models.LLMConfig{
					GenerateURL: "https://example.invalid",
					MaxTokens:   100,
				},
			},
		},
		{
			name: "invalid max tokens",
			cfg: &models.Config{
				LLM: models.LLMConfig{
					GenerateURL:  "https://example.invalid",
					GeminiAPIKey: "test-key",
					MaxTokens:    0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewGeminiClient(tt.cfg)
			if err == nil {
				t.Fatal("expected constructor error")
			}
			if client != nil {
				t.Fatal("expected nil client for invalid config")
			}
		})
	}
}

func TestNewGeminiClientAcceptsValidConfig(t *testing.T) {
	cfg := &models.Config{
		LLM: models.LLMConfig{
			GenerateURL:  "https://example.invalid",
			GeminiAPIKey: "test-key",
			MaxTokens:    100,
			Model:        "test-model",
		},
	}

	client, err := NewGeminiClient(cfg)
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if client == nil {
		t.Fatal("expected a client")
	}
}
