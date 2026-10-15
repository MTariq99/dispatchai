package authn

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

var testNow = time.Unix(1791900000, 0)

type fakeKeys struct {
	rec       *models.KeyRecord
	err       error
	gotClient string
	gotKey    string
}

func (f *fakeKeys) LookupActiveKey(_ context.Context, clientID, keyID string) (*models.KeyRecord, error) {
	f.gotClient, f.gotKey = clientID, keyID
	if f.err != nil {
		return nil, f.err
	}
	return f.rec, nil
}

type fakeNonces struct {
	seen  map[string]bool
	err   error
	calls int
}

func (f *fakeNonces) RecordNonce(_ context.Context, integrationID, nonce string, _ time.Time) (bool, error) {
	f.calls++
	if f.err != nil {
		return false, f.err
	}
	k := integrationID + "/" + nonce
	if f.seen[k] {
		return false, nil
	}
	f.seen[k] = true
	return true, nil
}

func newFixture(t *testing.T) (*RequestVerifier, *fakeKeys, *fakeNonces, *ecdsa.PrivateKey) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keys := &fakeKeys{rec: &models.KeyRecord{
		IntegrationID: "int-1",
		PublicKey:     elliptic.Marshal(elliptic.P256(), priv.PublicKey.X, priv.PublicKey.Y),
	}}
	nonces := &fakeNonces{seen: map[string]bool{}}
	v, err := NewRequestVerifier(keys, nonces, VerifierConfig{})
	if err != nil {
		t.Fatal(err)
	}
	v.now = func() time.Time { return testNow }
	return v, keys, nonces, priv
}

const testBody = `{"a":1}`

func signedRequest(t *testing.T, priv *ecdsa.PrivateKey, ts int64) *http.Request {
	t.Helper()
	headers := SignedHeaders{
		ClientID:      "cli_test",
		KeyID:         "key-1",
		Timestamp:     strconv.FormatInt(ts, 10),
		Nonce:         "nonce-aaaaaaaaaaaaaaaa",
		RequestID:     "req-1",
		ContentSHA256: BodySHA256([]byte(testBody)),
		Algorithm:     enums.SignatureAlgorithm,
	}
	r := httptest.NewRequest(http.MethodPost, "/v1/integrations/tools?x=1", strings.NewReader(testBody))
	cr, err := CanonicalRequestFromHTTP(r, headers)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := BuildCanonicalReq(cr)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set(enums.HeaderClientID, headers.ClientID)
	r.Header.Set(enums.HeaderKeyID, headers.KeyID)
	r.Header.Set(enums.HeaderTimestamp, headers.Timestamp)
	r.Header.Set(enums.HeaderNonce, headers.Nonce)
	r.Header.Set(enums.HeaderRequestID, headers.RequestID)
	r.Header.Set(enums.HeaderContentSHA256, headers.ContentSHA256)
	r.Header.Set(enums.HeaderAlgorithm, headers.Algorithm)
	r.Header.Set(enums.HeaderSignature, signForTest(t, priv, canonical))
	return r
}

func TestVerifyValidRequest(t *testing.T) {
	v, keys, nonces, priv := newFixture(t)
	r := signedRequest(t, priv, testNow.Unix())

	id, err := v.Verify(r)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if id != "int-1" || keys.gotClient != "cli_test" || keys.gotKey != "key-1" || nonces.calls != 1 {
		t.Fatalf("id=%q client=%q key=%q nonceCalls=%d", id, keys.gotClient, keys.gotKey, nonces.calls)
	}
	body, _ := io.ReadAll(r.Body)
	if string(body) != testBody {
		t.Fatalf("handler can no longer read the body: %q", body)
	}
}

func TestVerifyRejectsTampering(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(r *http.Request)
		want   error
	}{
		{"body", func(r *http.Request) { r.Body = io.NopCloser(strings.NewReader(`{"a":2}`)) }, errs.ErrInvalidBodyHash},
		{"content hash header", func(r *http.Request) { r.Header.Set(enums.HeaderContentSHA256, BodySHA256([]byte("x"))) }, errs.ErrInvalidBodyHash},
		{"method", func(r *http.Request) { r.Method = http.MethodPut }, errs.ErrInvalidSignature},
		{"path", func(r *http.Request) { r.URL.Path = "/v1/integrations/other" }, errs.ErrInvalidSignature},
		{"query", func(r *http.Request) { r.URL.RawQuery = "x=2" }, errs.ErrInvalidSignature},
		{"request id", func(r *http.Request) { r.Header.Set(enums.HeaderRequestID, "req-2") }, errs.ErrInvalidSignature},
		{"nonce", func(r *http.Request) { r.Header.Set(enums.HeaderNonce, "nonce-bbbbbbbbbbbbbbbb") }, errs.ErrInvalidSignature},
		{"timestamp", func(r *http.Request) {
			r.Header.Set(enums.HeaderTimestamp, strconv.FormatInt(testNow.Unix()+1, 10))
		}, errs.ErrInvalidSignature},
		{"key id", func(r *http.Request) { r.Header.Set(enums.HeaderKeyID, "key-2") }, errs.ErrInvalidSignature},
		{"client id", func(r *http.Request) { r.Header.Set(enums.HeaderClientID, "cli_other") }, errs.ErrInvalidSignature},
		{"signature not base64", func(r *http.Request) { r.Header.Set(enums.HeaderSignature, "abc") }, errs.ErrInvalidSignature},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, _, nonces, priv := newFixture(t)
			r := signedRequest(t, priv, testNow.Unix())
			tt.mutate(r)
			if _, err := v.Verify(r); !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			if nonces.calls != 0 {
				t.Fatal("nonce must not be stored when verification fails")
			}
		})
	}
}

func TestVerifyRejectsWrongKeySignature(t *testing.T) {
	v, _, nonces, _ := newFixture(t)
	other, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if _, err := v.Verify(signedRequest(t, other, testNow.Unix())); !errors.Is(err, errs.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
	if nonces.calls != 0 {
		t.Fatal("nonce must not be stored")
	}
}

func TestVerifyRejectsMissingOrMalformedHeaders(t *testing.T) {
	all := []string{
		enums.HeaderClientID, enums.HeaderKeyID, enums.HeaderTimestamp, enums.HeaderNonce,
		enums.HeaderRequestID, enums.HeaderContentSHA256, enums.HeaderAlgorithm, enums.HeaderSignature,
	}
	for _, h := range all {
		t.Run("missing "+h, func(t *testing.T) {
			v, _, nonces, priv := newFixture(t)
			r := signedRequest(t, priv, testNow.Unix())
			r.Header.Del(h)
			if _, err := v.Verify(r); !errors.Is(err, errs.ErrInvalidRequest) {
				t.Fatalf("expected ErrInvalidRequest, got %v", err)
			}
			if nonces.calls != 0 {
				t.Fatal("nonce must not be stored")
			}
		})
	}

	malformed := map[string]func(r *http.Request){
		"duplicate nonce":    func(r *http.Request) { r.Header.Add(enums.HeaderNonce, "nonce-cccccccccccccccc") },
		"short nonce":        func(r *http.Request) { r.Header.Set(enums.HeaderNonce, "short") },
		"non-digit ts":       func(r *http.Request) { r.Header.Set(enums.HeaderTimestamp, "12ab") },
		"bad algorithm":      func(r *http.Request) { r.Header.Set(enums.HeaderAlgorithm, "none") },
		"bad sig characters": func(r *http.Request) { r.Header.Set(enums.HeaderSignature, "%%%") },
		"uppercase hash": func(r *http.Request) {
			r.Header.Set(enums.HeaderContentSHA256, strings.ToUpper(BodySHA256([]byte(testBody))))
		},
	}
	for name, mutate := range malformed {
		t.Run(name, func(t *testing.T) {
			v, _, _, priv := newFixture(t)
			r := signedRequest(t, priv, testNow.Unix())
			mutate(r)
			if _, err := v.Verify(r); !errors.Is(err, errs.ErrInvalidRequest) {
				t.Fatalf("expected ErrInvalidRequest, got %v", err)
			}
		})
	}
}

func TestVerifyTimestampWindow(t *testing.T) {
	for name, tc := range map[string]struct {
		offset time.Duration
		ok     bool
	}{
		"now":             {0, true},
		"4m59s old":       {-(5*time.Minute - time.Second), true},
		"4m59s in future": {5*time.Minute - time.Second, true},
		"stale":           {-6 * time.Minute, false},
		"far future":      {6 * time.Minute, false},
	} {
		t.Run(name, func(t *testing.T) {
			v, _, _, priv := newFixture(t)
			_, err := v.Verify(signedRequest(t, priv, testNow.Add(tc.offset).Unix()))
			if tc.ok && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.ok && !errors.Is(err, errs.ErrStaleTimestamp) {
				t.Fatalf("expected ErrStaleTimestamp, got %v", err)
			}
		})
	}
}

func TestVerifyReplayRejected(t *testing.T) {
	v, _, _, priv := newFixture(t)
	if _, err := v.Verify(signedRequest(t, priv, testNow.Unix())); err != nil {
		t.Fatalf("first request: %v", err)
	}
	if _, err := v.Verify(signedRequest(t, priv, testNow.Unix())); !errors.Is(err, errs.ErrReplayedRequest) {
		t.Fatalf("expected ErrReplayedRequest, got %v", err)
	}
}

func TestVerifyKeyProblems(t *testing.T) {
	v, keys, _, priv := newFixture(t)
	keys.err = errs.ErrSigKeyNotFound
	_, err := v.Verify(signedRequest(t, priv, testNow.Unix()))
	if !errors.Is(err, errs.ErrSigKeyNotFound) || errors.Is(err, errs.ErrAuthUnavailable) {
		t.Fatalf("expected plain ErrSigKeyNotFound, got %v", err)
	}

	v, keys, _, priv = newFixture(t)
	keys.rec.PublicKey = []byte("bad")
	if _, err := v.Verify(signedRequest(t, priv, testNow.Unix())); !errors.Is(err, errs.ErrInvalidPublicKey) {
		t.Fatalf("expected ErrInvalidPublicKey, got %v", err)
	}
}

func TestVerifyFailsClosedOnBackendErrors(t *testing.T) {
	v, keys, _, priv := newFixture(t)
	keys.err = errors.New("db down")
	if _, err := v.Verify(signedRequest(t, priv, testNow.Unix())); !errors.Is(err, errs.ErrAuthUnavailable) {
		t.Fatalf("lookup failure: expected ErrAuthUnavailable, got %v", err)
	}

	v, _, nonces, priv := newFixture(t)
	nonces.err = errors.New("db down")
	id, err := v.Verify(signedRequest(t, priv, testNow.Unix()))
	if !errors.Is(err, errs.ErrAuthUnavailable) || id != "" {
		t.Fatalf("nonce failure: expected ErrAuthUnavailable and no identity, got id=%q err=%v", id, err)
	}
}

func TestVerifyRejectsOversizedBody(t *testing.T) {
	v, _, _, priv := newFixture(t)
	v.maxBody = 3
	if _, err := v.Verify(signedRequest(t, priv, testNow.Unix())); !errors.Is(err, errs.ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}
