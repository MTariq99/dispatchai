package authn

import (
	"strings"
	"testing"

	"github.com/mtariq99/dispatchai/internal/enums"
	"github.com/mtariq99/dispatchai/models"
)

func validCanonicalRequest() models.CanonicalRequest {
	return models.CanonicalRequest{
		Method: "POST",
		Path:   "/v1/integrations/tools",
		Query:  "",
		Headers: models.SignedHeaders{
			ClientID:      "client-123",
			KeyID:         "key-456",
			Timestamp:     "1791900000",
			Nonce:         "a-random-nonce-value",
			RequestID:     "request-789",
			ContentSHA256: BodySHA256(nil),
			Algorithm:     enums.SignatureAlgorithm,
		},
	}
}

func TestBuildCanonicalRequest(t *testing.T) {
	got, err := BuildCanonicalReq(validCanonicalRequest())
	if err != nil {
		t.Fatalf("BuildCanonicalRequest() error = %v", err)
	}

	want := strings.Join([]string{
		"DISPATCHAI-REQUEST-V1",
		"POST",
		"/v1/integrations/tools",
		"",
		"client-123",
		"key-456",
		"1791900000",
		"a-random-nonce-value",
		"request-789",
		BodySHA256(nil),
		"ECDSA-P256-SHA256",
	}, "\n")

	if string(got) != want {
		t.Fatalf("canonical request mismatch:\ngot:\n%q\nwant:\n%q", got, want)
	}
}

func TestBuildCanonicalRequestRejectsUnsupportedAlgorithm(t *testing.T) {
	req := validCanonicalRequest()
	req.Headers.Algorithm = "none"

	if _, err := BuildCanonicalReq(req); err == nil {
		t.Fatal("expected unsupported algorithm to be rejected")
	}
}

func TestBuildCanonicalRequestRejectsInvalidBodyHash(t *testing.T) {
	req := validCanonicalRequest()
	req.Headers.ContentSHA256 = "not-a-hash"

	if _, err := BuildCanonicalReq(req); err == nil {
		t.Fatal("expected invalid body hash to be rejected")
	}
}

func TestBuildCanonicalRequestRejectsNewlineInClientID(t *testing.T) {
	req := validCanonicalRequest()
	req.Headers.ClientID = "client-123\nattacker"

	if _, err := BuildCanonicalReq(req); err == nil {
		t.Fatal("expected newline injection to be rejected")
	}
}

func TestBodySHA256EmptyBody(t *testing.T) {
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	if got := BodySHA256(nil); got != want {
		t.Fatalf("BodySHA256(nil) = %q, want %q", got, want)
	}
}
