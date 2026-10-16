package storage

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

func openEnrollmentTestDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run PostgreSQL integration tests")
	}
	dsn := os.Getenv("INTEGRATION_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://dispatchai:dispatchai@localhost:5438/dispatch?sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	return db
}

type enrollmentFixture struct {
	integrationID string
	clientID      string
	tokenHash     []byte
}

// seedEnrollment inserts an integration with the given status and one
// enrollment token created createdAgo ago that expires in expiresIn.
func seedEnrollment(t *testing.T, db *sql.DB, status string, createdAgo, expiresIn time.Duration) enrollmentFixture {
	t.Helper()
	ctx := context.Background()

	clientID := "cli_test_" + uuid.NewString()
	var integrationID string
	if err := db.QueryRowContext(ctx, `
		INSERT INTO host_integrations (integration_key, display_name, status)
		VALUES ($1, 'enrollment test', $2)
		RETURNING id::text
	`, clientID, status).Scan(&integrationID); err != nil {
		t.Fatalf("seed integration: %v", err)
	}

	t.Cleanup(func() {
		bg := context.Background()
		_, _ = db.ExecContext(bg, `DELETE FROM integration_keys WHERE integration_id = $1::uuid`, integrationID)
		_, _ = db.ExecContext(bg, `DELETE FROM integration_enrollment_tokens WHERE integration_id = $1::uuid`, integrationID)
		_, _ = db.ExecContext(bg, `DELETE FROM host_integrations WHERE id = $1::uuid`, integrationID)
	})

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		t.Fatalf("random token: %v", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))

	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO integration_enrollment_tokens (integration_id, token_hash, created_at, expires_at)
		VALUES ($1::uuid, $2, $3, $4)
	`, integrationID, hash[:], now.Add(-createdAgo), now.Add(expiresIn)); err != nil {
		t.Fatalf("seed enrollment token: %v", err)
	}

	return enrollmentFixture{integrationID: integrationID, clientID: clientID, tokenHash: hash[:]}
}

func validEnrollment(t *testing.T, db *sql.DB) enrollmentFixture {
	t.Helper()
	return seedEnrollment(t, db, "pending", time.Minute, 14*time.Minute)
}

func newTestPublicKey(t *testing.T) []byte {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	encoded, err := priv.PublicKey.Bytes()
	if err != nil {
		t.Fatalf("encode public key: %v", err)
	}
	return encoded
}

func tokenConsumed(t *testing.T, db *sql.DB, hash []byte) bool {
	t.Helper()
	var consumed bool
	if err := db.QueryRow(
		`SELECT consumed_at IS NOT NULL FROM integration_enrollment_tokens WHERE token_hash = $1`, hash,
	).Scan(&consumed); err != nil {
		t.Fatalf("read token state: %v", err)
	}
	return consumed
}

func keyCount(t *testing.T, db *sql.DB, integrationID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM integration_keys WHERE integration_id = $1::uuid`, integrationID,
	).Scan(&n); err != nil {
		t.Fatalf("count keys: %v", err)
	}
	return n
}

func integrationStatus(t *testing.T, db *sql.DB, integrationID string) string {
	t.Helper()
	var s string
	if err := db.QueryRow(
		`SELECT status FROM host_integrations WHERE id = $1::uuid`, integrationID,
	).Scan(&s); err != nil {
		t.Fatalf("read integration status: %v", err)
	}
	return s
}

func TestEnrollKeySuccess(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewIntegrationEnrollmentStore(db)
	fx := validEnrollment(t, db)
	pub := newTestPublicKey(t)

	got, err := store.EnrollKey(context.Background(), models.EnrollKeyParams{
		TokenHash: fx.tokenHash, KeyID: "key-1", PublicKey: pub,
	})
	if err != nil {
		t.Fatalf("EnrollKey() error = %v", err)
	}
	if got.ClientID != fx.clientID || got.IntegrationID != fx.integrationID || got.KeyID != "key-1" {
		t.Fatalf("unexpected result: %+v", got)
	}

	var stored []byte
	var algorithm, keyStatus string
	if err := db.QueryRow(`
		SELECT public_key, algorithm, status FROM integration_keys
		WHERE integration_id = $1::uuid AND key_id = 'key-1'
	`, fx.integrationID).Scan(&stored, &algorithm, &keyStatus); err != nil {
		t.Fatalf("read stored key: %v", err)
	}
	if !bytes.Equal(stored, pub) || algorithm != "ECDSA-P256-SHA256" || keyStatus != "active" {
		t.Fatalf("stored key mismatch: algo=%s status=%s", algorithm, keyStatus)
	}
	if !tokenConsumed(t, db, fx.tokenHash) {
		t.Fatal("token must be consumed after success")
	}
	if s := integrationStatus(t, db, fx.integrationID); s != "pending" {
		t.Fatalf("enrollment must not activate the integration, status = %q", s)
	}
}

func TestEnrollKeyUnknownToken(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewIntegrationEnrollmentStore(db)
	unknown := sha256.Sum256([]byte("no-such-token-" + uuid.NewString()))

	_, err := store.EnrollKey(context.Background(), models.EnrollKeyParams{
		TokenHash: unknown[:], KeyID: "key-1", PublicKey: newTestPublicKey(t),
	})
	if !errors.Is(err, errs.ErrEnrollmentTokenInvalid) {
		t.Fatalf("expected ErrEnrollmentTokenInvalid, got %v", err)
	}
}

func TestEnrollKeyExpiredToken(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewIntegrationEnrollmentStore(db)
	fx := seedEnrollment(t, db, "pending", 2*time.Hour, -time.Hour)

	_, err := store.EnrollKey(context.Background(), models.EnrollKeyParams{
		TokenHash: fx.tokenHash, KeyID: "key-1", PublicKey: newTestPublicKey(t),
	})
	if !errors.Is(err, errs.ErrEnrollmentTokenInvalid) {
		t.Fatalf("expected ErrEnrollmentTokenInvalid, got %v", err)
	}
	if keyCount(t, db, fx.integrationID) != 0 {
		t.Fatal("no key may be stored for an expired token")
	}
}

func TestEnrollKeyReusedToken(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewIntegrationEnrollmentStore(db)
	fx := validEnrollment(t, db)
	ctx := context.Background()

	if _, err := store.EnrollKey(ctx, models.EnrollKeyParams{
		TokenHash: fx.tokenHash, KeyID: "key-1", PublicKey: newTestPublicKey(t),
	}); err != nil {
		t.Fatalf("first enrollment: %v", err)
	}
	_, err := store.EnrollKey(ctx, models.EnrollKeyParams{
		TokenHash: fx.tokenHash, KeyID: "key-2", PublicKey: newTestPublicKey(t),
	})
	if !errors.Is(err, errs.ErrEnrollmentTokenInvalid) {
		t.Fatalf("expected ErrEnrollmentTokenInvalid, got %v", err)
	}
	if n := keyCount(t, db, fx.integrationID); n != 1 {
		t.Fatalf("expected exactly 1 key, got %d", n)
	}
}

func TestEnrollKeyRejectsInvalidPublicKeyWithoutBurningToken(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewIntegrationEnrollmentStore(db)
	fx := validEnrollment(t, db)

	wrongPrefix := make([]byte, 65)
	wrongPrefix[0] = 0x02
	offCurve := make([]byte, 65)
	offCurve[0] = 0x04
	offCurve[32], offCurve[64] = 1, 1 // (1,1) is not on P-256

	cases := map[string][]byte{
		"empty":        nil,
		"too short":    {0x04},
		"compressed":   append([]byte{0x02}, make([]byte, 32)...),
		"wrong prefix": wrongPrefix,
		"off curve":    offCurve,
	}
	for name, pub := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := store.EnrollKey(context.Background(), models.EnrollKeyParams{
				TokenHash: fx.tokenHash, KeyID: "key-1", PublicKey: pub,
			})
			if !errors.Is(err, errs.ErrInvalidPublicKey) {
				t.Fatalf("expected ErrInvalidPublicKey, got %v", err)
			}
			if tokenConsumed(t, db, fx.tokenHash) {
				t.Fatal("invalid key must not consume the token")
			}
		})
	}
}

func TestEnrollKeyRejectsInvalidKeyID(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewIntegrationEnrollmentStore(db)
	fx := validEnrollment(t, db)

	for _, keyID := range []string{"", " lead", "has space", "new\nline", "-dash", strings.Repeat("a", 129)} {
		_, err := store.EnrollKey(context.Background(), models.EnrollKeyParams{
			TokenHash: fx.tokenHash, KeyID: keyID, PublicKey: newTestPublicKey(t),
		})
		if !errors.Is(err, errs.ErrInvalidEnrollmentRequest) {
			t.Fatalf("key ID %q: expected ErrInvalidEnrollmentRequest, got %v", keyID, err)
		}
	}
	if tokenConsumed(t, db, fx.tokenHash) {
		t.Fatal("invalid key ID must not consume the token")
	}
}

func TestEnrollKeyDuplicateKeyIDRollsBack(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewIntegrationEnrollmentStore(db)
	fx := validEnrollment(t, db)

	if _, err := db.Exec(`
		INSERT INTO integration_keys (integration_id, key_id, public_key)
		VALUES ($1::uuid, 'dup', $2)
	`, fx.integrationID, newTestPublicKey(t)); err != nil {
		t.Fatalf("seed existing key: %v", err)
	}

	_, err := store.EnrollKey(context.Background(), models.EnrollKeyParams{
		TokenHash: fx.tokenHash, KeyID: "dup", PublicKey: newTestPublicKey(t),
	})
	if !errors.Is(err, errs.ErrKeyIDConflict) {
		t.Fatalf("expected ErrKeyIDConflict, got %v", err)
	}
	if tokenConsumed(t, db, fx.tokenHash) {
		t.Fatal("rollback must leave the token unconsumed")
	}
	if n := keyCount(t, db, fx.integrationID); n != 1 {
		t.Fatalf("expected only the pre-existing key, got %d", n)
	}
}

func TestEnrollKeyIneligibleIntegrationRollsBack(t *testing.T) {
	for _, status := range []string{"active", "suspended"} {
		t.Run(status, func(t *testing.T) {
			db := openEnrollmentTestDB(t)
			store := NewIntegrationEnrollmentStore(db)
			fx := seedEnrollment(t, db, status, time.Minute, 14*time.Minute)

			_, err := store.EnrollKey(context.Background(), models.EnrollKeyParams{
				TokenHash: fx.tokenHash, KeyID: "key-1", PublicKey: newTestPublicKey(t),
			})
			if !errors.Is(err, errs.ErrIntegrationNotEnrollable) {
				t.Fatalf("expected ErrIntegrationNotEnrollable, got %v", err)
			}
			if tokenConsumed(t, db, fx.tokenHash) {
				t.Fatal("rollback must leave the token unconsumed")
			}
			if keyCount(t, db, fx.integrationID) != 0 {
				t.Fatal("no key may be stored for an ineligible integration")
			}
			if s := integrationStatus(t, db, fx.integrationID); s != status {
				t.Fatalf("integration status changed to %q", s)
			}
		})
	}
}

func TestEnrollKeyConcurrentSameToken(t *testing.T) {
	db := openEnrollmentTestDB(t)
	store := NewIntegrationEnrollmentStore(db)
	fx := validEnrollment(t, db)

	const attempts = 32
	keys := make([][]byte, attempts)
	for i := range keys {
		keys[i] = newTestPublicKey(t) // generated here: t.Fatal is unsafe in goroutines
	}

	start := make(chan struct{})
	results := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, err := store.EnrollKey(context.Background(), models.EnrollKeyParams{
				TokenHash: fx.tokenHash, KeyID: fmt.Sprintf("key-%d", i), PublicKey: keys[i],
			})
			results <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	var ok, invalid int
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, errs.ErrEnrollmentTokenInvalid):
			invalid++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 1 || invalid != attempts-1 {
		t.Fatalf("want 1 success and %d rejections, got %d and %d", attempts-1, ok, invalid)
	}
	if n := keyCount(t, db, fx.integrationID); n != 1 {
		t.Fatalf("one token must bind exactly one key, got %d", n)
	}
}
