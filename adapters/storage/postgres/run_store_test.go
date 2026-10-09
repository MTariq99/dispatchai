package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/internal/run"
)

func TestRunStoreIntegration(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run PostgreSQL integration tests")
	}

	db, err := sql.Open(
		"postgres",
		"postgres://dispatchai:dispatchai@localhost:5438/dispatch?sslmode=disable",
	)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}

	store := NewRunStore(db)

	tenantID := uuid.New()
	userID := uuid.New()
	conversationID := uuid.New()
	requestID := uuid.New()
	runID := uuid.New()

	now := time.Date(
		2026,
		time.October,
		8,
		12,
		0,
		0,
		0,
		time.UTC,
	)

	_, err = db.ExecContext(
		ctx,
		`
		INSERT INTO conversations (
			id,
			tenant_id,
			user_id,
			title,
			status,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		conversationID,
		tenantID,
		userID,
		"RunStore integration test",
		"active",
		now,
		now,
	)
	if err != nil {
		t.Fatalf("create test conversation: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(
			context.Background(),
			`DELETE FROM conversations WHERE id = $1`,
			conversationID,
		)
	})

	r := &run.Run{
		ID:             runID,
		TenantID:       tenantID,
		UserID:         userID,
		ConversationID: conversationID,
		RequestID:      requestID,
		Status:         run.StateCreated,
		ModelProvider:  "gemini",
		ModelName:      "gemini-2.5-flash",
		CreatedAt:      now,
		UpdatedAt:      now,
		InputTokens:    100,
		OutputTokens:   50,
		TotalTokens:    150,
	}

	t.Run("Create", func(t *testing.T) {
		err := store.Create(ctx, r)
		if err != nil {
			t.Fatalf("create run: %v", err)
		}
	})

	t.Run("Get", func(t *testing.T) {
		got, err := store.Get(ctx, runID)
		if err != nil {
			t.Fatalf("get run: %v", err)
		}

		assertRunEqual(t, r, got)
	})

	t.Run("GetByRequestID", func(t *testing.T) {
		got, err := store.GetByRequestID(ctx, requestID)
		if err != nil {
			t.Fatalf("get run by request id: %v", err)
		}

		assertRunEqual(t, r, got)
	})

	t.Run("Update", func(t *testing.T) {
		updatedAt := now.Add(2 * time.Minute)
		completedAt := now.Add(3 * time.Minute)

		r.Status = run.StateCompleted
		r.StartedAt = now.Add(30 * time.Second)
		r.CompletedAt = &completedAt
		r.UpdatedAt = updatedAt
		r.InputTokens = 200
		r.OutputTokens = 100
		r.TotalTokens = 300
		r.ErrorCode = ""
		r.ErrorMessage = ""

		err := store.Update(ctx, r)
		if err != nil {
			t.Fatalf("update run: %v", err)
		}

		got, err := store.Get(ctx, runID)
		if err != nil {
			t.Fatalf("get updated run: %v", err)
		}

		assertRunEqual(t, r, got)
	})

	t.Run("GetNotFound", func(t *testing.T) {
		_, err := store.Get(ctx, uuid.New())
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})

	t.Run("GetByRequestIDNotFound", func(t *testing.T) {
		_, err := store.GetByRequestID(ctx, uuid.New())
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected sql.ErrNoRows, got %v", err)
		}
	})
}

func assertRunEqual(t *testing.T, want, got *run.Run) {
	t.Helper()

	if want.ID != got.ID {
		t.Errorf("ID: want %s, got %s", want.ID, got.ID)
	}

	if want.TenantID != got.TenantID {
		t.Errorf("TenantID: want %s, got %s", want.TenantID, got.TenantID)
	}

	if want.UserID != got.UserID {
		t.Errorf("UserID: want %s, got %s", want.UserID, got.UserID)
	}

	if want.ConversationID != got.ConversationID {
		t.Errorf("ConversationID: want %s, got %s", want.ConversationID, got.ConversationID)
	}

	if want.RequestID != got.RequestID {
		t.Errorf("RequestID: want %s, got %s", want.RequestID, got.RequestID)
	}

	if want.Status != got.Status {
		t.Errorf("Status: want %s, got %s", want.Status, got.Status)
	}

	if want.ModelProvider != got.ModelProvider {
		t.Errorf("ModelProvider: want %s, got %s", want.ModelProvider, got.ModelProvider)
	}

	if want.ModelName != got.ModelName {
		t.Errorf("ModelName: want %s, got %s", want.ModelName, got.ModelName)
	}

	if !want.StartedAt.Equal(got.StartedAt) {
		t.Errorf("StartedAt: want %v, got %v", want.StartedAt, got.StartedAt)
	}

	if !equalTimePtr(want.CompletedAt, got.CompletedAt) {
		t.Errorf("CompletedAt: want %v, got %v", want.CompletedAt, got.CompletedAt)
	}

	if want.InputTokens != got.InputTokens {
		t.Errorf("InputTokens: want %d, got %d", want.InputTokens, got.InputTokens)
	}

	if want.OutputTokens != got.OutputTokens {
		t.Errorf("OutputTokens: want %d, got %d", want.OutputTokens, got.OutputTokens)
	}

	if want.TotalTokens != got.TotalTokens {
		t.Errorf("TotalTokens: want %d, got %d", want.TotalTokens, got.TotalTokens)
	}

	if want.ErrorCode != got.ErrorCode {
		t.Errorf("ErrorCode: want %q, got %q", want.ErrorCode, got.ErrorCode)
	}

	if want.ErrorMessage != got.ErrorMessage {
		t.Errorf("ErrorMessage: want %q, got %q", want.ErrorMessage, got.ErrorMessage)
	}

	if !want.CreatedAt.Equal(got.CreatedAt) {
		t.Errorf("CreatedAt: want %v, got %v", want.CreatedAt, got.CreatedAt)
	}

	if !want.UpdatedAt.Equal(got.UpdatedAt) {
		t.Errorf("UpdatedAt: want %v, got %v", want.UpdatedAt, got.UpdatedAt)
	}
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}

	return a.Equal(*b)
}
