package http

import (
	"bytes"
	"context"
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mtariq99/dispatchai/models"
)

func newHandlerTestRouter(tools map[string]ToolHandler) *gin.Engine {
	router := gin.New()
	NewHandler(nil, tools).RegisterRoutes(router)
	return router
}

func performToolRequest(t *testing.T, router *gin.Engine, req *models.ToolRequest) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	httpRequest := httptest.NewRequest(stdhttp.MethodPost, "/api/v1/ai/tools/execute", bytes.NewReader(body))
	httpRequest.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httpRequest)

	return recorder
}

func TestHandlerReturnsCachedResponse(t *testing.T) {
	req := testToolRequest()
	var executions atomic.Int32

	router := newHandlerTestRouter(map[string]ToolHandler{
		req.ToolName: func(
			_ context.Context,
			r *models.ToolRequest,
		) (*models.ToolResponse, error) {
			executions.Add(1)
			return testToolResponse(r), nil
		},
	})

	first := performToolRequest(t, router, req)
	second := performToolRequest(t, router, req)

	if first.Code != stdhttp.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s",
			first.Code, first.Body.String())
	}

	if second.Code != stdhttp.StatusOK {
		t.Fatalf("second request: expected 200, got %d: %s",
			second.Code, second.Body.String())
	}

	if executions.Load() != 1 {
		t.Fatalf("expected one tool execution, got %d", executions.Load())
	}

	var firstResponse models.ToolResponse
	if err := json.Unmarshal(first.Body.Bytes(), &firstResponse); err != nil {
		t.Fatalf("decode first response: %v", err)
	}

	var secondResponse models.ToolResponse
	if err := json.Unmarshal(second.Body.Bytes(), &secondResponse); err != nil {
		t.Fatalf("decode second response: %v", err)
	}

	if string(firstResponse.Data) != string(secondResponse.Data) {
		t.Fatalf("expected cached response data %s, got %s",
			firstResponse.Data, secondResponse.Data)
	}
}

func TestHandlerRejectsConflictingRequest(t *testing.T) {
	req := testToolRequest()
	var executions atomic.Int32

	router := newHandlerTestRouter(map[string]ToolHandler{
		req.ToolName: func(
			_ context.Context,
			r *models.ToolRequest,
		) (*models.ToolResponse, error) {
			executions.Add(1)
			return testToolResponse(r), nil
		},
	})

	first := performToolRequest(t, router, req)
	if first.Code != stdhttp.StatusOK {
		t.Fatalf("first request: expected 200, got %d: %s",
			first.Code, first.Body.String())
	}

	conflicting := *req
	conflicting.Arguments = json.RawMessage(
		`{"customer_id":"different-customer"}`,
	)

	second := performToolRequest(t, router, &conflicting)

	if second.Code != stdhttp.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d: %s",
			second.Code, second.Body.String())
	}

	if executions.Load() != 1 {
		t.Fatalf("conflicting request must not execute tool; got %d executions",
			executions.Load())
	}
}

func TestHandlerExecutesConcurrentDuplicatesOnce(t *testing.T) {
	req := testToolRequest()

	var executions atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	start := make(chan struct{})

	router := newHandlerTestRouter(map[string]ToolHandler{
		req.ToolName: func(
			_ context.Context,
			r *models.ToolRequest,
		) (*models.ToolResponse, error) {
			executions.Add(1)
			close(started)
			<-release
			return testToolResponse(r), nil
		},
	})

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	send := func() *httptest.ResponseRecorder {
		httpRequest := httptest.NewRequest(
			stdhttp.MethodPost,
			"/api/v1/ai/tools/execute",
			bytes.NewReader(body),
		)
		httpRequest.Header.Set("Content-Type", "application/json")

		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httpRequest)
		return recorder
	}

	var wg sync.WaitGroup
	results := make(chan *httptest.ResponseRecorder, 2)

	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- send()
		}()
	}

	close(start)

	// Wait until one request has started executing the tool.
	<-started

	// Allow the tool to finish so the duplicate can receive its result.
	close(release)

	wg.Wait()
	close(results)

	for result := range results {
		if result.Code != stdhttp.StatusOK {
			t.Errorf("expected 200, got %d: %s",
				result.Code, result.Body.String())
		}
	}

	if executions.Load() != 1 {
		t.Fatalf("expected one tool execution, got %d", executions.Load())
	}
}

func TestHandlerRejectsInvalidIdempotencyRequest(t *testing.T) {
	req := testToolRequest()
	req.CallID = ""

	var executions atomic.Int32

	router := newHandlerTestRouter(map[string]ToolHandler{
		req.ToolName: func(
			_ context.Context,
			r *models.ToolRequest,
		) (*models.ToolResponse, error) {
			executions.Add(1)
			return testToolResponse(r), nil
		},
	})

	recorder := performToolRequest(t, router, req)

	if recorder.Code != stdhttp.StatusBadRequest {
		t.Fatalf(
			"expected 400 Bad Request, got %d: %s",
			recorder.Code,
			recorder.Body.String(),
		)
	}

	if executions.Load() != 0 {
		t.Fatalf(
			"invalid request must not execute the tool; got %d executions",
			executions.Load(),
		)
	}
}
