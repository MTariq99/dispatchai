package idempotency_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/google/uuid"
	postgresstore "github.com/mtariq99/dispatchai/adapters/storage/postgres"
)

func TestRequestIdempotencyStoreIntegration(t *testing.T) {
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

	store := postgresstore.NewIdempotencyStore(db)
	var integrationID uuid.UUID

	integrationKey := "cli_test_" + uuid.NewString()

	err = db.QueryRowContext(ctx, `
    INSERT INTO host_integrations (
        integration_key,
        display_name,
        status
    )
    VALUES ($1, $2, $3)
    RETURNING id
`, integrationKey, "request idempotency integration test", "active").
		Scan(&integrationID)
	if err != nil {
		t.Fatalf("create test integration: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(
			context.Background(),
			`DELETE FROM host_integrations WHERE id = $1`,
			integrationID,
		)
	})

	key := "request-fingerprint-" + uuid.NewString()
	conversationID := uuid.New()
	requestID := uuid.New()

	t.Cleanup(func() {
		_, _ = db.ExecContext(
			context.Background(),
			`DELETE FROM idempotency_keys WHERE integration_id = $1 AND key = $2`,
			integrationID,
			key,
		)
	})

	// First request: reserve the key with its fingerprint.
	reserved, err := store.Reserve(
		ctx,
		integrationID,
		key,
		conversationID,
		requestID,
		"fingerprint-original",
	)
	if err != nil {
		t.Fatalf("reserve initial request: %v", err)
	}
	if !reserved {
		t.Fatal("expected initial reservation to succeed")
	}

	// Verify that the fingerprint was persisted and can be read back.
	state, found, err := store.GetState(ctx, integrationID, key)
	if err != nil {
		t.Fatalf("get initial request state: %v", err)
	}
	if !found {
		t.Fatal("expected reserved request state to exist")
	}
	if state.Fingerprint != "fingerprint-original" {
		t.Fatalf(
			"initial fingerprint: want %q, got %q",
			"fingerprint-original",
			state.Fingerprint,
		)
	}
	if state.RequestID != requestID {
		t.Fatalf("request ID: want %s, got %s", requestID, state.RequestID)
	}

	// Second request: reuse the same key with a different fingerprint.
	reserved, err = store.Reserve(
		ctx,
		integrationID,
		key,
		uuid.New(),
		uuid.New(),
		"fingerprint-different",
	)
	if err != nil {
		t.Fatalf("reserve duplicate key: %v", err)
	}
	if reserved {
		t.Fatal("expected duplicate reservation to be rejected")
	}

	// Verify that the original request and fingerprint remain unchanged.
	state, found, err = store.GetState(ctx, integrationID, key)
	if err != nil {
		t.Fatalf("get request state after duplicate: %v", err)
	}
	if !found {
		t.Fatal("expected original request state to remain")
	}
	if state.Fingerprint != "fingerprint-original" {
		t.Fatalf(
			"fingerprint after duplicate: want %q, got %q",
			"fingerprint-original",
			state.Fingerprint,
		)
	}
	if state.RequestID != requestID {
		t.Fatalf(
			"request ID after duplicate: want %s, got %s",
			requestID,
			state.RequestID,
		)
	}
}
