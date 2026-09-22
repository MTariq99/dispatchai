package llm

import (
	"context"

	"github.com/mtariq99/dispatchai/models"
)

type MockClient struct {
	Responses []models.Response

	index int
}

// NewMockClient creates a mock LLM client.
func NewMockClient(responses ...models.Response) *MockClient {
	return &MockClient{
		Responses: responses,
	}
}

// Generate returns the next predefined response.
func (m *MockClient) Generate(_ context.Context, _ models.Request) (models.Response, error) {
	if m.index >= len(m.Responses) {
		return models.Response{}, nil
	}

	response := m.Responses[m.index]
	m.index++

	return response, nil
}
