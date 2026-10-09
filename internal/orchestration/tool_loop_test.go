package assistant

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/internal/conversation"
	"github.com/mtariq99/dispatchai/internal/idempotency"
	"github.com/mtariq99/dispatchai/internal/policy"
	"github.com/mtariq99/dispatchai/internal/project"
	"github.com/mtariq99/dispatchai/internal/tools"
	"github.com/mtariq99/dispatchai/models"
)

type testLLM struct {
	responses []*models.Response
	requests  []models.Request
}

func (f *testLLM) Generate(_ context.Context, req models.Request) (*models.Response, error) {
	f.requests = append(f.requests, req)
	if len(f.responses) == 0 {
		return nil, context.Canceled
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp, nil
}

type testGateway struct {
	calls []models.Call
}

func (f *testGateway) Execute(_ context.Context, call models.Call, _ models.ExecutionContext) *models.Result {
	f.calls = append(f.calls, call)
	return &models.Result{
		CallID:  call.ID,
		Success: true,
		Data:    json.RawMessage(`{"order_id":"ord-123","status":"delayed"}`),
	}
}

type testIdempotency struct{}

func (testIdempotency) Get(context.Context, uuid.UUID, string) (*models.Result, bool, error) {
	return nil, false, nil
}

func (testIdempotency) Reserve(context.Context, uuid.UUID, string, string, json.RawMessage) (bool, error) {
	return true, nil
}

func (testIdempotency) Complete(context.Context, uuid.UUID, string, *models.Result) error {
	return nil
}

func (testIdempotency) Fail(context.Context, uuid.UUID, string, *models.Error) error {
	return nil
}

func TestToolLoopToolRoundTrip(t *testing.T) {
	registry := tools.NewRegistry()
	err := registry.Register(tools.NewStaticTool(models.Definition{
		Name:        "find_order",
		Description: "Find an order by ID",
		Parameters:  map[string]any{"type": "object"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	gateway := &testGateway{}
	executor := tools.NewExecutor(
		registry,
		project.ToolGateway(gateway),
		policy.NewEngine(),
		idempotency.ToolIdempotency(testIdempotency{}),
	)

	fake := &testLLM{
		responses: []*models.Response{
			{
				ToolCalls: []models.ToolCall{
					{
						ID:   "call-1",
						Name: "find_order",
						Args: map[string]any{"order_id": "ord-123"},
					},
				},
			},
			{Content: "Order ord-123 is delayed."},
		},
	}

	loop := NewToolLoop(fake, registry, executor, "test-model", 1000, 0)
	conv := conversation.NewConversation()
	conv.AddUserMessage("Where is order ord-123?")

	execCtx := &models.ExecutionContext{
		RequestID:      "req-1",
		RunID:          uuid.New(),
		ConversationID: uuid.New(),
		Identity:       models.Identity{UserID: uuid.New()},
		Tenant:         models.TenantContext{TenantID: uuid.New()},
	}

	answer, err := loop.Run(context.Background(), conv, execCtx)
	if err != nil {
		t.Fatalf("ToolLoop.Run() error = %v", err)
	}
	if answer != "Order ord-123 is delayed." {
		t.Fatalf("unexpected answer: %q", answer)
	}
	if len(fake.requests) != 2 {
		t.Fatalf("expected 2 LLM requests, got %d", len(fake.requests))
	}
	if len(gateway.calls) != 1 {
		t.Fatalf("expected 1 gateway call, got %d", len(gateway.calls))
	}
	if gateway.calls[0].Name != "find_order" {
		t.Fatalf("unexpected tool: %q", gateway.calls[0].Name)
	}
	if len(conv.Messages) < 4 {
		t.Fatalf("expected conversation to contain user, tool call, tool result, and final answer; got %d messages", len(conv.Messages))
	}
}

func TestToolLoopNilLLMResponse(t *testing.T) {
	registry := tools.NewRegistry()
	fake := &testLLM{
		responses: []*models.Response{nil},
	}
	loop := NewToolLoop(fake, registry, nil, "test-model", 1000, 0)
	conv := conversation.NewConversation()
	conv.AddUserMessage("Hello")

	answer, err := loop.Run(context.Background(), conv, &models.ExecutionContext{})
	if err == nil {
		t.Fatal("expected an error for a nil LLM response")
	}
	if answer != "" {
		t.Fatalf("expected empty answer, got %q", answer)
	}
	if err.Error() != "llm generate returned a nil response" {
		t.Fatalf("unexpected error: %v", err)
	}
}
