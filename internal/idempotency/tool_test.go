package idempotency

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/mtariq99/dispatchai/models"
)

func openToolIdempotencyTestDB(t *testing.T) *sql.DB {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run PostgreSQL integration tests")
	}

	db, err := sql.Open(
		"postgres",
		"postgres://dispatchai:dispatchai@localhost:5438/dispatch?sslmode=disable",
	)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close PostgreSQL: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}

	return db
}

func newToolIdempotencyTestRun(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()

	ctx := context.Background()
	conversationID := uuid.New()
	runID := uuid.New()

	_, err := db.ExecContext(ctx, `
		INSERT INTO conversations (id, tenant_id, user_id, title)
		VALUES ($1, $2, $3, $4)
	`, conversationID, uuid.New(), uuid.New(), "tool idempotency integration test")
	if err != nil {
		t.Fatalf("create test conversation: %v", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO conversation_runs (id, conversation_id, request_id, status)
		VALUES ($1, $2, $3, $4)
	`, runID, conversationID, uuid.New(), "running")
	if err != nil {
		_, _ = db.ExecContext(ctx,
			`DELETE FROM conversations WHERE id = $1`, conversationID)
		t.Fatalf("create test run: %v", err)
	}

	t.Cleanup(func() {
		_, err := db.ExecContext(context.Background(),
			`DELETE FROM conversations WHERE id = $1`, conversationID)
		if err != nil {
			t.Errorf("delete test conversation %s: %v", conversationID, err)
		}
	})

	return runID
}

func TestToolIdempotencyReserveIntegration(t *testing.T) {
	db := openToolIdempotencyTestDB(t)
	store := NewToolIdempotency(db)

	tests := []struct {
		name       string
		firstTool  string
		firstArgs  string
		secondTool string
		secondArgs string
		wantFirst  bool
		wantSecond bool
		wantErr    error
	}{
		{
			name:       "first reservation succeeds",
			firstTool:  "create_order",
			firstArgs:  `{"sku":"book","quantity":1}`,
			secondTool: "create_order",
			secondArgs: `{"sku":"book","quantity":1}`,
			wantFirst:  true,
			wantSecond: false,
		},
		{
			name:       "same key with different tool conflicts",
			firstTool:  "create_order",
			firstArgs:  `{"sku":"book","quantity":1}`,
			secondTool: "cancel_order",
			secondArgs: `{"sku":"book","quantity":1}`,
			wantFirst:  true,
			wantErr:    ErrToolCallConflict,
		},
		{
			name:       "same key with different arguments conflicts",
			firstTool:  "create_order",
			firstArgs:  `{"sku":"book","quantity":1}`,
			secondTool: "create_order",
			secondArgs: `{"sku":"book","quantity":2}`,
			wantFirst:  true,
			wantErr:    ErrToolCallConflict,
		},
		{
			name:       "JSON object key order is equivalent",
			firstTool:  "create_order",
			firstArgs:  `{"sku":"book","quantity":1}`,
			secondTool: "create_order",
			secondArgs: `{"quantity":1,"sku":"book"}`,
			wantFirst:  true,
			wantSecond: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runID := newToolIdempotencyTestRun(t, db)
			callID := fmt.Sprintf("call-%s", uuid.NewString())

			first, err := store.Reserve(
				context.Background(),
				runID,
				callID,
				tt.firstTool,
				json.RawMessage(tt.firstArgs),
			)
			if err != nil {
				t.Fatalf("first Reserve() error = %v", err)
			}
			if first != tt.wantFirst {
				t.Fatalf("first Reserve() = %v, want %v", first, tt.wantFirst)
			}

			second, err := store.Reserve(
				context.Background(),
				runID,
				callID,
				tt.secondTool,
				json.RawMessage(tt.secondArgs),
			)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("second Reserve() error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("second Reserve() error = %v", err)
			}
			if second != tt.wantSecond {
				t.Fatalf("second Reserve() = %v, want %v", second, tt.wantSecond)
			}
		})
	}
}

func TestToolIdempotencyGetTerminalStatesIntegration(t *testing.T) {
	db := openToolIdempotencyTestDB(t)
	store := NewToolIdempotency(db)
	ctx := context.Background()

	t.Run("completed result is replayed", func(t *testing.T) {
		runID := newToolIdempotencyTestRun(t, db)
		callID := "completed-" + uuid.NewString()

		reserved, err := store.Reserve(
			ctx,
			runID,
			callID,
			"find_order",
			json.RawMessage(`{"order_id":"ord-123"}`),
		)
		if err != nil || !reserved {
			t.Fatalf("Reserve() = (%v, %v), want (true, nil)", reserved, err)
		}

		want := &models.Result{
			CallID:  callID,
			Success: true,
			Data:    json.RawMessage(`{"status":"shipped"}`),
		}

		if err := store.Complete(ctx, runID, callID, want); err != nil {
			t.Fatalf("Complete() error = %v", err)
		}

		got, exists, err := store.Get(ctx, runID, callID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if !exists {
			t.Fatal("Get() exists = false, want true")
		}
		if got == nil || !got.Success || got.CallID != callID {
			t.Fatalf("Get() result = %+v, want successful result", got)
		}
		var gotData, wantData any

		if err := json.Unmarshal(got.Data, &gotData); err != nil {
			t.Fatalf("unmarshal returned data: %v", err)
		}
		if err := json.Unmarshal(want.Data, &wantData); err != nil {
			t.Fatalf("unmarshal expected data: %v", err)
		}

		gotJSON, err := json.Marshal(gotData)
		if err != nil {
			t.Fatalf("marshal returned data: %v", err)
		}
		wantJSON, err := json.Marshal(wantData)
		if err != nil {
			t.Fatalf("marshal expected data: %v", err)
		}

		if !bytes.Equal(gotJSON, wantJSON) {
			t.Fatalf("Get() data = %s, want %s", gotJSON, wantJSON)
		}
	})

	t.Run("failed result is replayed", func(t *testing.T) {
		runID := newToolIdempotencyTestRun(t, db)
		callID := "failed-" + uuid.NewString()

		reserved, err := store.Reserve(
			ctx,
			runID,
			callID,
			"create_order",
			json.RawMessage(`{"sku":"book"}`),
		)
		if err != nil || !reserved {
			t.Fatalf("Reserve() = (%v, %v), want (true, nil)", reserved, err)
		}

		want := &models.Error{
			Code:      "INSUFFICIENT_STOCK",
			Message:   "requested quantity is unavailable",
			Retryable: false,
		}

		if err := store.Fail(ctx, runID, callID, want); err != nil {
			t.Fatalf("Fail() error = %v", err)
		}

		got, exists, err := store.Get(ctx, runID, callID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if !exists {
			t.Fatal("Get() exists = false, want true")
		}
		if got == nil || got.Success || got.Error == nil {
			t.Fatalf("Get() result = %+v, want failed result", got)
		}

		if got.CallID != callID {
			t.Errorf("CallID = %q, want %q", got.CallID, callID)
		}
		if got.Error.Code != want.Code {
			t.Errorf("error code = %q, want %q", got.Error.Code, want.Code)
		}
		if got.Error.Message != want.Message {
			t.Errorf("error message = %q, want %q", got.Error.Message, want.Message)
		}
		if got.Error.Retryable != want.Retryable {
			t.Errorf("Retryable = %v, want %v", got.Error.Retryable, want.Retryable)
		}
	})

	t.Run("pending execution is not terminal", func(t *testing.T) {
		runID := newToolIdempotencyTestRun(t, db)
		callID := "pending-" + uuid.NewString()

		reserved, err := store.Reserve(
			ctx,
			runID,
			callID,
			"find_order",
			json.RawMessage(`{"order_id":"ord-456"}`),
		)
		if err != nil || !reserved {
			t.Fatalf("Reserve() = (%v, %v), want (true, nil)", reserved, err)
		}

		got, exists, err := store.Get(ctx, runID, callID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if exists || got != nil {
			t.Fatalf("Get() = (%+v, %v), want (nil, false)", got, exists)
		}

		// The existing reservation must still block a duplicate execution.
		reserved, err = store.Reserve(
			ctx,
			runID,
			callID,
			"find_order",
			json.RawMessage(`{"order_id":"ord-456"}`),
		)
		if err != nil {
			t.Fatalf("second Reserve() error = %v", err)
		}
		if reserved {
			t.Fatal("second Reserve() = true; pending operation must not be re-executed")
		}
	})

	t.Run("missing execution is not found", func(t *testing.T) {
		runID := newToolIdempotencyTestRun(t, db)

		got, exists, err := store.Get(
			ctx,
			runID,
			"missing-"+uuid.NewString(),
		)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if exists || got != nil {
			t.Fatalf("Get() = (%+v, %v), want (nil, false)", got, exists)
		}
	})
}

func TestToolIdempotencyFailTerminalStateIntegration(t *testing.T) {
	db := openToolIdempotencyTestDB(t)
	store := NewToolIdempotency(db)
	ctx := context.Background()

	t.Run("failure is persisted and cannot overwrite terminal state", func(t *testing.T) {
		runID := newToolIdempotencyTestRun(t, db)
		callID := "fail-terminal-" + uuid.NewString()

		reserved, err := store.Reserve(
			ctx,
			runID,
			callID,
			"create_order",
			json.RawMessage(`{"sku":"book"}`),
		)
		if err != nil || !reserved {
			t.Fatalf(
				"Reserve() = (%v, %v), want (true, nil)",
				reserved,
				err,
			)
		}

		firstFailure := &models.Error{
			Code:      "ORDER_REJECTED",
			Message:   "order was rejected",
			Retryable: false,
		}

		if err := store.Fail(ctx, runID, callID, firstFailure); err != nil {
			t.Fatalf("first Fail() error = %v", err)
		}

		got, exists, err := store.Get(ctx, runID, callID)
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if !exists || got == nil || got.Error == nil {
			t.Fatalf("Get() = (%+v, %v), want persisted failure", got, exists)
		}
		if got.Error.Code != firstFailure.Code {
			t.Fatalf("error code = %q, want %q",
				got.Error.Code, firstFailure.Code)
		}
		if got.Error.Message != firstFailure.Message {
			t.Fatalf("error message = %q, want %q",
				got.Error.Message, firstFailure.Message)
		}
		if got.Error.Retryable != firstFailure.Retryable {
			t.Fatalf("Retryable = %v, want %v",
				got.Error.Retryable, firstFailure.Retryable)
		}

		secondFailure := &models.Error{
			Code:      "DIFFERENT_ERROR",
			Message:   "this must not overwrite the original failure",
			Retryable: true,
		}

		err = store.Fail(ctx, runID, callID, secondFailure)
		if err == nil {
			t.Fatal("second Fail() succeeded; expected terminal state to be immutable")
		}

		got, exists, err = store.Get(ctx, runID, callID)
		if err != nil {
			t.Fatalf("Get() after second Fail() error = %v", err)
		}
		if !exists || got == nil || got.Error == nil {
			t.Fatalf("Get() after second Fail() = (%+v, %v), want original failure",
				got, exists)
		}
		if got.Error.Code != firstFailure.Code ||
			got.Error.Message != firstFailure.Message ||
			got.Error.Retryable != firstFailure.Retryable {
			t.Fatalf("original failure was overwritten: got %+v", got.Error)
		}

		err = store.Complete(ctx, runID, callID, &models.Result{
			CallID:  callID,
			Success: true,
			Data:    json.RawMessage(`{"status":"created"}`),
		})
		if err == nil {
			t.Fatal("Complete() succeeded after failure; expected terminal state to be immutable")
		}

		got, exists, err = store.Get(ctx, runID, callID)
		if err != nil {
			t.Fatalf("final Get() error = %v", err)
		}
		if !exists || got == nil || got.Success || got.Error == nil {
			t.Fatalf("final state = (%+v, %v), want original failure", got, exists)
		}
		if got.Error.Code != firstFailure.Code {
			t.Fatalf("final error code = %q, want %q",
				got.Error.Code, firstFailure.Code)
		}
	})
}

func TestToolIdempotencyConcurrentReserveIntegration(t *testing.T) {
	db := openToolIdempotencyTestDB(t)
	store := NewToolIdempotency(db)

	runID := newToolIdempotencyTestRun(t, db)
	callID := "concurrent-" + uuid.NewString()
	arguments := json.RawMessage(`{"sku":"book","quantity":1}`)

	const workers = 20

	start := make(chan struct{})
	type reserveResult struct {
		reserved bool
		err      error
	}

	results := make(chan reserveResult, workers)

	for i := 0; i < workers; i++ {
		go func() {
			<-start

			reserved, err := store.Reserve(
				context.Background(),
				runID,
				callID,
				"create_order",
				arguments,
			)

			results <- reserveResult{
				reserved: reserved,
				err:      err,
			}
		}()
	}

	// Release all goroutines together to increase contention.
	close(start)

	successCount := 0
	conflictCount := 0

	for i := 0; i < workers; i++ {
		result := <-results

		if result.err != nil {
			t.Errorf("Reserve() returned unexpected error: %v", result.err)
			continue
		}

		if result.reserved {
			successCount++
		} else {
			conflictCount++
		}
	}

	if successCount != 1 {
		t.Errorf("successful reservations = %d, want exactly 1", successCount)
	}

	if conflictCount != workers-1 {
		t.Errorf(
			"unsuccessful reservations = %d, want %d",
			conflictCount,
			workers-1,
		)
	}

	// Verify PostgreSQL contains exactly one execution for this key.
	var count int
	err := db.QueryRowContext(
		context.Background(),
		`SELECT COUNT(*)
		 FROM tool_executions
		 WHERE run_id = $1 AND call_id = $2`,
		runID,
		callID,
	).Scan(&count)
	if err != nil {
		t.Fatalf("count persisted executions: %v", err)
	}

	if count != 1 {
		t.Errorf("persisted executions = %d, want exactly 1", count)
	}
}
