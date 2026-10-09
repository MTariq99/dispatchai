package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/internal/idempotency"
	"github.com/mtariq99/dispatchai/internal/policy"
	"github.com/mtariq99/dispatchai/internal/project"
	"github.com/mtariq99/dispatchai/models"
)

type executorTestGateway struct {
	calls  int
	result *models.Result
}

func (g *executorTestGateway) Execute(_ context.Context, call models.Call, _ models.ExecutionContext) *models.Result {
	g.calls++
	return g.result
}

type executorTestIdempotency struct {
	exists       bool
	cached       *models.Result
	reserved     bool
	getErr       error
	reserveErr   error
	getCalls     int
	reserveCalls int
	failCalls    int
	failErr      error
	completeErr  error
}

func (s *executorTestIdempotency) Get(_ context.Context, _ uuid.UUID, _ string) (*models.Result, bool, error) {
	s.getCalls++
	return s.cached, s.exists, s.getErr
}

func (s *executorTestIdempotency) Reserve(_ context.Context, _ uuid.UUID, _ string, _ string, _ json.RawMessage) (bool, error) {
	s.reserveCalls++
	return s.reserved, s.reserveErr
}

func (s *executorTestIdempotency) Complete(context.Context, uuid.UUID, string, *models.Result) error {
	if s.completeErr != nil {
		s.reserved = false
		return s.completeErr
	}

	return nil
}

func (s *executorTestIdempotency) Fail(_ context.Context, _ uuid.UUID, _ string, _ *models.Error) error {
	s.failCalls++
	return s.failErr
}

var (
	_ project.ToolGateway         = (*executorTestGateway)(nil)
	_ idempotency.ToolIdempotency = (*executorTestIdempotency)(nil)
)

func newExecutorTestSetup(t *testing.T, engine *policy.Engine) (*Executer, *executorTestGateway, *executorTestIdempotency, *models.ExecutionContext) {
	t.Helper()

	registry := NewRegistry()
	err := registry.Register(NewStaticTool(models.Definition{
		Name:       "find_order",
		Parameters: map[string]any{"type": "object"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	gateway := &executorTestGateway{
		result: &models.Result{
			Success: true,
			Data:    json.RawMessage(`{"status":"delayed"}`),
		},
	}

	store := &executorTestIdempotency{reserved: true}
	executor := NewExecutor(registry, gateway, engine, store)

	execCtx := &models.ExecutionContext{
		RunID:          uuid.New(),
		ConversationID: uuid.New(),
		Identity:       models.Identity{UserID: uuid.New()},
		Tenant:         models.TenantContext{TenantID: uuid.New()},
	}

	return executor, gateway, store, execCtx
}

func TestExecutorRejectsInvalidCalls(t *testing.T) {
	executor, gateway, _, execCtx := newExecutorTestSetup(t, policy.NewEngine())

	result := executor.Execute(context.Background(), nil, execCtx, nil)
	if result.Success || result.Error == nil || result.Error.Code != "INVALID_TOOL_CALL" {
		t.Fatalf("expected INVALID_TOOL_CALL, got %+v", result)
	}
	if gateway.calls != 0 {
		t.Fatal("gateway must not execute a nil call")
	}

	result = executor.Execute(context.Background(), &models.Call{ID: "call-1"}, execCtx, nil)
	if result.Success || result.Error == nil || result.Error.Code != "INVALID_TOOL_CALL" {
		t.Fatalf("expected INVALID_TOOL_CALL for empty tool name, got %+v", result)
	}
	if gateway.calls != 0 {
		t.Fatal("gateway must not execute a call with an empty tool name")
	}
}

func TestExecutorRejectsUnknownTool(t *testing.T) {
	executor, gateway, store, execCtx := newExecutorTestSetup(t, policy.NewEngine())

	result := executor.Execute(context.Background(), &models.Call{
		ID:   "call-1",
		Name: "delete_everything",
	}, execCtx, nil)

	if result.Success || result.Error == nil || result.Error.Code != "UNKNOWN_TOOL" {
		t.Fatalf("expected UNKNOWN_TOOL, got %+v", result)
	}
	if gateway.calls != 0 || store.getCalls != 0 {
		t.Fatalf("unknown tool should not reach gateway or idempotency store: gateway=%d store=%d",
			gateway.calls, store.getCalls)
	}
}

func TestExecutorPolicyDenialBlocksGateway(t *testing.T) {
	deny := policy.Rule(func(
		_ *models.Call,
		_ *models.ExecutionContext,
		_ map[string]int,
	) policy.Decision {
		return policy.DenyDecision("test policy denial")
	})

	executor, gateway, store, execCtx := newExecutorTestSetup(t, policy.NewEngine(deny))

	result := executor.Execute(context.Background(), &models.Call{
		ID:   "call-1",
		Name: "find_order",
	}, execCtx, nil)

	if result.Success || result.Error == nil || result.Error.Code != "POLICY_DENIED" {
		t.Fatalf("expected POLICY_DENIED, got %+v", result)
	}
	if gateway.calls != 0 {
		t.Fatal("gateway must not execute a denied call")
	}
	if store.failCalls != 1 {
		t.Fatalf("expected denial to be recorded once, got %d", store.failCalls)
	}
}

func TestExecutorReservationConflictBlocksGateway(t *testing.T) {
	executor, gateway, store, execCtx := newExecutorTestSetup(t, policy.NewEngine())
	store.reserved = false

	result := executor.Execute(context.Background(), &models.Call{
		ID:   "call-1",
		Name: "find_order",
	}, execCtx, nil)

	if result.Success || result.Error == nil || result.Error.Code != "OPERATION_IN_PROGRESS" {
		t.Fatalf("expected OPERATION_IN_PROGRESS, got %+v", result)
	}
	if gateway.calls != 0 {
		t.Fatal("gateway must not execute when reservation fails")
	}
}

func TestExecutorReturnsCachedResult(t *testing.T) {
	executor, gateway, store, execCtx := newExecutorTestSetup(t, policy.NewEngine())
	store.exists = true
	store.cached = &models.Result{
		CallID:  "call-1",
		Success: true,
		Data:    json.RawMessage(`{"status":"already-completed"}`),
	}

	result := executor.Execute(context.Background(), &models.Call{
		ID:   "call-1",
		Name: "find_order",
	}, execCtx, nil)

	if !result.Success || string(result.Data) != `{"status":"already-completed"}` {
		t.Fatalf("expected cached result, got %+v", result)
	}
	if gateway.calls != 0 || store.reserveCalls != 0 {
		t.Fatalf("cached result must bypass execution and reservation: gateway=%d reserve=%d",
			gateway.calls, store.reserveCalls)
	}
}

func TestExecutorReturnsCachedFailure(t *testing.T) {
	executor, gateway, store, execCtx := newExecutorTestSetup(
		t,
		policy.NewEngine(),
	)

	store.exists = true
	store.cached = &models.Result{
		CallID:  "call-1",
		Success: false,
		Error: &models.Error{
			Code:      "ORDER_NOT_FOUND",
			Message:   "order does not exist",
			Retryable: false,
		},
	}

	result := executor.Execute(context.Background(), &models.Call{
		ID:   "call-1",
		Name: "find_order",
	}, execCtx, nil)

	if result.Success {
		t.Fatalf("expected cached failure, got %+v", result)
	}

	if result.Error == nil {
		t.Fatal("expected cached error")
	}

	if result.Error.Code != "ORDER_NOT_FOUND" {
		t.Fatalf("expected ORDER_NOT_FOUND, got %q", result.Error.Code)
	}

	if result.Error.Message != "order does not exist" {
		t.Fatalf("unexpected cached error message: %q", result.Error.Message)
	}

	if result.Error.Retryable {
		t.Fatal("expected cached failure to remain non-retryable")
	}

	if gateway.calls != 0 || store.reserveCalls != 0 {
		t.Fatalf(
			"cached failure must bypass execution and reservation: gateway=%d reserve=%d",
			gateway.calls,
			store.reserveCalls,
		)
	}
}

func TestExecutorHandlesNilGatewayResult(t *testing.T) {
	executor, gateway, _, execCtx := newExecutorTestSetup(t, policy.NewEngine())
	gateway.result = nil

	result := executor.Execute(context.Background(), &models.Call{
		ID:   "call-1",
		Name: "find_order",
	}, execCtx, nil)

	if result.Success || result.Error == nil || result.Error.Code != "TOOL_EXECUTION_FAILED" {
		t.Fatalf("expected TOOL_EXECUTION_FAILED, got %+v", result)
	}
	if gateway.calls != 1 {
		t.Fatalf("expected one gateway call, got %d", gateway.calls)
	}
}

func TestExecutorDoesNotRetryWhenCompletePersistenceFails(t *testing.T) {
	executor, gateway, store, execCtx := newExecutorTestSetup(
		t,
		policy.NewEngine(),
	)
	store.completeErr = context.DeadlineExceeded

	call := &models.Call{
		ID:        "call-complete-failure",
		Name:      "find_order",
		Arguments: json.RawMessage(`{"order_id":"ord-123"}`),
	}

	first := executor.Execute(context.Background(), call, execCtx, nil)

	if first.Success {
		t.Fatalf("expected persistence error, got success: %+v", first)
	}
	if first.Error == nil ||
		first.Error.Code != "TOOL_RESULT_PERSISTENCE_FAILED" {
		t.Fatalf("expected TOOL_RESULT_PERSISTENCE_FAILED, got %+v", first)
	}
	if first.Error.Retryable {
		t.Fatal("uncertain execution outcome must not be automatically retried")
	}
	if gateway.calls != 1 {
		t.Fatalf("expected one gateway execution, got %d", gateway.calls)
	}

	// A subsequent attempt must respect the existing pending reservation.
	second := executor.Execute(context.Background(), call, execCtx, nil)

	if second.Success {
		t.Fatalf("expected pending-operation error, got %+v", second)
	}
	if second.Error == nil ||
		second.Error.Code != "OPERATION_IN_PROGRESS" {
		t.Fatalf("expected OPERATION_IN_PROGRESS, got %+v", second)
	}
	if gateway.calls != 1 {
		t.Fatalf(
			"gateway must not execute again after uncertain outcome; calls=%d",
			gateway.calls,
		)
	}
}

func TestExecutorDoesNotRetryWhenFailurePersistenceFails(t *testing.T) {
	executor, gateway, store, execCtx := newExecutorTestSetup(
		t,
		policy.NewEngine(),
	)

	gateway.result = &models.Result{
		CallID:  "call-failure-persistence",
		Success: false,
		Error: &models.Error{
			Code:      "HOST_TOOL_FAILED",
			Message:   "host operation returned an error",
			Retryable: true,
		},
	}
	store.failErr = context.DeadlineExceeded

	call := &models.Call{
		ID:        "call-failure-persistence",
		Name:      "find_order",
		Arguments: json.RawMessage(`{"order_id":"ord-123"}`),
	}

	first := executor.Execute(context.Background(), call, execCtx, nil)

	if first.Success {
		t.Fatalf("expected failure, got %+v", first)
	}
	if first.Error == nil {
		t.Fatal("expected an error")
	}
	if store.failCalls != 1 {
		t.Fatalf("expected one failure-persistence attempt, got %d", store.failCalls)
	}
	if gateway.calls != 1 {
		t.Fatalf("expected one gateway execution, got %d", gateway.calls)
	}

	// The fake store still has no cached terminal result and its
	// reservation remains unavailable, simulating an unresolved operation.
	store.exists = false
	store.reserved = false

	second := executor.Execute(context.Background(), call, execCtx, nil)

	if second.Success {
		t.Fatalf("expected unresolved-operation failure, got %+v", second)
	}
	if second.Error == nil || second.Error.Code != "OPERATION_IN_PROGRESS" {
		t.Fatalf("expected OPERATION_IN_PROGRESS, got %+v", second)
	}
	if gateway.calls != 1 {
		t.Fatalf("gateway must not execute again; calls=%d", gateway.calls)
	}
}

func TestExecutorReturnsErrorWhenPolicyPersistenceFails(t *testing.T) {
	tests := []struct {
		name    string
		outcome policy.Outcome
		reason  string
	}{
		{
			name:    "denial",
			outcome: policy.Deny,
			reason:  "test policy denial",
		},
		{
			name:    "approval required",
			outcome: policy.RequireApproval,
			reason:  "approval is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule := policy.Rule(func(
				_ *models.Call,
				_ *models.ExecutionContext,
				_ map[string]int,
			) policy.Decision {
				return policy.Decision{
					Outcome: tt.outcome,
					Reason:  tt.reason,
				}
			})

			executor, gateway, store, execCtx := newExecutorTestSetup(
				t,
				policy.NewEngine(rule),
			)
			store.failErr = context.DeadlineExceeded

			result := executor.Execute(context.Background(), &models.Call{
				ID:   "call-policy-persistence-failure",
				Name: "find_order",
			}, execCtx, nil)

			if result.Success {
				t.Fatalf("expected failure, got %+v", result)
			}
			if result.Error == nil {
				t.Fatal("expected an error")
			}
			if result.Error.Code != "TOOL_POLICY_PERSISTENCE_FAILED" {
				t.Fatalf(
					"expected TOOL_POLICY_PERSISTENCE_FAILED, got %+v",
					result.Error,
				)
			}
			if result.Error.Retryable {
				t.Fatal("policy persistence failure must not be automatically retried")
			}
			if store.failCalls != 1 {
				t.Fatalf("expected one persistence attempt, got %d", store.failCalls)
			}
			if gateway.calls != 0 {
				t.Fatalf("gateway must not execute; calls=%d", gateway.calls)
			}
		})
	}
}

func TestExecutorPersistsFailureWhenGatewayReturnsUnsuccessfulResultWithoutError(t *testing.T) {
	executor, gateway, store, execCtx := newExecutorTestSetup(t, policy.NewEngine())
	gateway.result = &models.Result{
		CallID:  "call-missing-error",
		Success: false,
		Error:   nil,
	}

	call := &models.Call{
		ID:        "call-missing-error",
		Name:      "find_order",
		Arguments: json.RawMessage(`{"order_id":"ord-123"}`),
	}

	result := executor.Execute(context.Background(), call, execCtx, nil)

	if result.Success {
		t.Fatalf("expected unsuccessful result, got %+v", result)
	}
	if result.Error == nil || result.Error.Code != "TOOL_EXECUTION_FAILED" {
		t.Fatalf("expected normalized TOOL_EXECUTION_FAILED error, got %+v", result)
	}
	if store.failCalls != 1 {
		t.Fatalf("expected failure state to be persisted once, got %d calls", store.failCalls)
	}
	if gateway.calls != 1 {
		t.Fatalf("expected one gateway execution, got %d", gateway.calls)
	}
}

func TestExecutorRejectsEmptyCallIDBeforeIdempotencyAccess(t *testing.T) {
	executor, gateway, store, execCtx := newExecutorTestSetup(t, policy.NewEngine())

	result := executor.Execute(context.Background(), &models.Call{
		ID:   "",
		Name: "find_order",
	}, execCtx, nil)

	if result.Success {
		t.Fatalf("expected invalid tool call, got %+v", result)
	}
	if result.Error == nil || result.Error.Code != "INVALID_TOOL_CALL" {
		t.Fatalf("expected INVALID_TOOL_CALL, got %+v", result)
	}
	if store.getCalls != 0 || store.reserveCalls != 0 {
		t.Fatalf("idempotency store must not be accessed: Get=%d Reserve=%d",
			store.getCalls, store.reserveCalls)
	}
	if gateway.calls != 0 {
		t.Fatalf("gateway must not execute; calls=%d", gateway.calls)
	}
}

func TestExecutorRejectsNilExecutor(t *testing.T) {
	var executor *Executer

	result := executor.Execute(context.Background(), &models.Call{
		ID:   "call-1",
		Name: "find_order",
	}, &models.ExecutionContext{RunID: uuid.New()}, nil)

	if result == nil || result.Error == nil {
		t.Fatal("expected an explicit configuration error")
	}
	if result.Error.Code != "EXECUTOR_NOT_CONFIGURED" {
		t.Fatalf("expected EXECUTOR_NOT_CONFIGURED, got %q", result.Error.Code)
	}
}

func TestExecutorRejectsMissingDependencies(t *testing.T) {
	tests := []struct {
		name     string
		executor *Executer
	}{
		{
			name:     "nil registry",
			executor: &Executer{toolGateWay: &executorTestGateway{}, policyEngine: policy.NewEngine(), IdempotentStore: &executorTestIdempotency{}},
		},
		{
			name:     "nil gateway",
			executor: &Executer{registry: NewRegistry(), policyEngine: policy.NewEngine(), IdempotentStore: &executorTestIdempotency{}},
		},
		{
			name:     "nil policy engine",
			executor: &Executer{registry: NewRegistry(), toolGateWay: &executorTestGateway{}, IdempotentStore: &executorTestIdempotency{}},
		},
		{
			name:     "nil idempotency store",
			executor: &Executer{registry: NewRegistry(), toolGateWay: &executorTestGateway{}, policyEngine: policy.NewEngine()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.executor.Execute(context.Background(), &models.Call{
				ID:   "call-1",
				Name: "find_order",
			}, &models.ExecutionContext{RunID: uuid.New()}, nil)

			if result == nil || result.Error == nil {
				t.Fatal("expected an explicit configuration error")
			}
			if result.Error.Code != "EXECUTOR_NOT_CONFIGURED" {
				t.Fatalf("expected EXECUTOR_NOT_CONFIGURED, got %q", result.Error.Code)
			}
		})
	}
}
