package storage

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	errs "github.com/mtariq99/dispatchai/internal/errors"
)

func TestLookupActiveKey(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewSignedAuthStore(db)
	pub := newTestPublicKey(t)

	tests := []struct {
		name        string
		integStatus string
		keyStatus   string
		expiresSQL  string
		wantFound   bool
	}{
		{"active", "active", "active", "NULL", true},
		{"future expiry", "active", "active", "NOW() + interval '1 hour'", true},
		{"expired key", "active", "active", "NOW() - interval '1 hour'", false},
		{"revoked key", "active", "revoked", "NULL", false},
		{"pending integration", "pending", "active", "NULL", false},
		{"suspended integration", "suspended", "active", "NULL", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fx := seedEnrollment(t, db, tt.integStatus, time.Minute, 14*time.Minute)
			if _, err := db.Exec(`
				INSERT INTO integration_keys
					(integration_id, key_id, public_key, status, created_at, expires_at, revoked_at)
				VALUES ($1::uuid, 'k1', $2, $3::varchar, NOW() - interval '2 hours', `+tt.expiresSQL+`,
					CASE WHEN $3::text = 'revoked' THEN NOW() ELSE NULL END)`,
				fx.integrationID, pub, tt.keyStatus); err != nil {
				t.Fatalf("seed key: %v", err)
			}

			rec, err := store.LookupActiveKey(context.Background(), fx.clientID, "k1")
			if tt.wantFound {
				if err != nil {
					t.Fatalf("expected key, got %v", err)
				}
				if rec.IntegrationID != fx.integrationID || !bytes.Equal(rec.PublicKey, pub) {
					t.Fatalf("unexpected record: %+v", rec)
				}
				return
			}
			if !errors.Is(err, errs.ErrSigKeyNotFound) {
				t.Fatalf("expected ErrSigKeyNotFound, got %v", err)
			}
		})
	}
}

func TestLookupActiveKeyUnknowns(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewSignedAuthStore(db)
	fx := seedEnrollment(t, db, "active", time.Minute, 14*time.Minute)
	if _, err := db.Exec(`INSERT INTO integration_keys (integration_id, key_id, public_key)
		VALUES ($1::uuid, 'k1', $2)`, fx.integrationID, newTestPublicKey(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LookupActiveKey(context.Background(), fx.clientID, "other-key"); !errors.Is(err, errs.ErrSigKeyNotFound) {
		t.Fatalf("unknown key id: got %v", err)
	}
	if _, err := store.LookupActiveKey(context.Background(), "cli_does_not_exist", "k1"); !errors.Is(err, errs.ErrSigKeyNotFound) {
		t.Fatalf("unknown client: got %v", err)
	}
}

func TestRecordNonceConcurrentDuplicates(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewSignedAuthStore(db)
	fx := seedEnrollment(t, db, "active", time.Minute, 14*time.Minute)

	const attempts = 32
	start := make(chan struct{})
	results := make(chan bool, attempts)
	errsCh := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := store.RecordNonce(context.Background(), fx.integrationID, "same-nonce-value-0001", time.Now().Add(time.Hour))
			if err != nil {
				errsCh <- err
				return
			}
			results <- ok
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errsCh)

	for err := range errsCh {
		t.Errorf("unexpected error: %v", err)
	}
	wins := 0
	for ok := range results {
		if ok {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("want exactly 1 successful insert, got %d", wins)
	}
}

func TestPurgeExpiredNonces(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewSignedAuthStore(db)
	fx := seedEnrollment(t, db, "active", time.Minute, 14*time.Minute)
	ctx := context.Background()

	if ok, err := store.RecordNonce(ctx, fx.integrationID, "expired-nonce-000001", time.Now().Add(-time.Minute)); err != nil || !ok {
		t.Fatalf("seed expired nonce: ok=%v err=%v", ok, err)
	}
	if ok, err := store.RecordNonce(ctx, fx.integrationID, "live-nonce-0000000001", time.Now().Add(time.Hour)); err != nil || !ok {
		t.Fatalf("seed live nonce: ok=%v err=%v", ok, err)
	}
	if _, err := store.PurgeExpiredNonces(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM integration_request_nonces WHERE integration_id = $1::uuid`, fx.integrationID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected only the live nonce to remain, got %d rows", n)
	}
}
