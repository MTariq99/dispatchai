package http

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/models"
)

func testToolRequest() *models.ToolRequest {
	return &models.ToolRequest{
		RunID:    uuid.New(),
		CallID:   "call-1",
		ToolName: "send_notification",
		Arguments: []byte(
			`{"customer_id":"customer-1","message":"Your order shipped"}`,
		),
	}
}

func testToolResponse(req *models.ToolRequest) *models.ToolResponse {
	return &models.ToolResponse{
		RequestID: req.RequestID,
		CallID:    req.CallID,
		Success:   true,
		Data:      []byte(`{"sent":true}`),
	}
}

func TestIdempotencyStoreReturnsCachedResponse(t *testing.T) {
	store := newIdempotencyStore()
	req := testToolRequest()

	var executions atomic.Int32

	execute := func() (*models.ToolResponse, error) {
		executions.Add(1)
		return testToolResponse(req), nil
	}

	first, err := store.execute(context.Background(), req, execute)
	if err != nil {
		t.Fatalf("first execution failed: %v", err)
	}

	second, err := store.execute(context.Background(), req, execute)
	if err != nil {
		t.Fatalf("duplicate execution failed: %v", err)
	}

	if executions.Load() != 1 {
		t.Fatalf("expected one execution, got %d", executions.Load())
	}

	if string(first.Data) != string(second.Data) {
		t.Fatalf("expected cached response data %s, got %s", first.Data, second.Data)
	}
}

func TestIdempotencyStoreRejectsDifferentRequestForSameKey(t *testing.T) {
	store := newIdempotencyStore()
	req := testToolRequest()

	_, err := store.execute(context.Background(), req, func() (*models.ToolResponse, error) {
		return testToolResponse(req), nil
	})
	if err != nil {
		t.Fatalf("first execution failed: %v", err)
	}

	changed := *req
	changed.Arguments = []byte(`{"customer_id":"customer-2"}`)

	_, err = store.execute(context.Background(), &changed, func() (*models.ToolResponse, error) {
		t.Fatal("conflicting request must not execute")
		return nil, nil
	})

	if !errors.Is(err, errIdempotencyKeyConflict) {
		t.Fatalf("expected key conflict, got %v", err)
	}
}

func TestIdempotencyStoreExecutesConcurrentDuplicateOnce(t *testing.T) {
	store := newIdempotencyStore()
	req := testToolRequest()

	var executions atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})

	execute := func() (*models.ToolResponse, error) {
		executions.Add(1)
		close(started)
		<-release
		return testToolResponse(req), nil
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := store.execute(context.Background(), req, execute)
		errs <- err
	}()

	<-started

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := store.execute(context.Background(), req, execute)
		errs <- err
	}()

	close(release)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent execution failed: %v", err)
		}
	}

	if executions.Load() != 1 {
		t.Fatalf("expected one execution, got %d", executions.Load())
	}
}

func TestIdempotencyStoreAllowsRetryAfterHandlerError(t *testing.T) {
	store := newIdempotencyStore()
	req := testToolRequest()

	_, err := store.execute(context.Background(), req, func() (*models.ToolResponse, error) {
		return nil, errors.New("temporary failure")
	})
	if err == nil {
		t.Fatal("expected first execution to fail")
	}

	response, err := store.execute(context.Background(), req, func() (*models.ToolResponse, error) {
		return testToolResponse(req), nil
	})
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}

	if response == nil || !response.Success {
		t.Fatal("expected successful response after retry")
	}
}
