package authn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"

	errs "github.com/mtariq99/dispatchai/internal/errors"
)

func signForTest(t *testing.T, privateKey *ecdsa.PrivateKey, message []byte) string {
	t.Helper()

	digest := sha256.Sum256(message)
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	if err != nil {
		t.Fatalf("ecdsa.SignASN1() error = %v", err)
	}

	return base64.StdEncoding.EncodeToString(signature)
}

func TestVerifySignature(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	canonical, err := BuildCanonicalReq(validCanonicalRequest())
	if err != nil {
		t.Fatalf("BuildCanonicalRequest() error = %v", err)
	}

	signature := signForTest(t, privateKey, canonical)

	if err := VerifySignature(&privateKey.PublicKey, canonical, signature); err != nil {
		t.Fatalf("VerifySignature() error = %v", err)
	}
}

func TestVerifySignatureRejectsTampering(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	canonical, err := BuildCanonicalReq(validCanonicalRequest())
	if err != nil {
		t.Fatalf("BuildCanonicalRequest() error = %v", err)
	}

	signature := signForTest(t, privateKey, canonical)
	tampered := append([]byte(nil), canonical...)
	tampered[len(tampered)-1] ^= 1

	err = VerifySignature(&privateKey.PublicKey, tampered, signature)
	if !errors.Is(err, errs.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestVerifySignatureRejectsMalformedBase64(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	err = VerifySignature(
		&privateKey.PublicKey,
		[]byte("canonical request"),
		"%%%not-base64%%%",
	)
	if !errors.Is(err, errs.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestParseP256PublicKey(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	// Encode the original public key.
	encoded, err := privateKey.PublicKey.Bytes()
	if err != nil {
		t.Fatalf("PublicKey.Bytes() error = %v", err)
	}

	// Parse the encoded public key.
	parsed, err := ParseP256PublicKey(encoded)
	if err != nil {
		t.Fatalf("ParseP256PublicKey() error = %v", err)
	}

	// Encode the parsed key and compare it with the original.
	parsedBytes, err := parsed.Bytes()
	if err != nil {
		t.Fatalf("parsed.Bytes() error = %v", err)
	}

	if string(parsedBytes) != string(encoded) {
		t.Fatal("parsed public key does not match original key")
	}
}

func TestParseP256PublicKeyRejectsInvalidInput(t *testing.T) {
	_, err := ParseP256PublicKey([]byte("not-a-public-key"))
	if !errors.Is(err, errs.ErrInvalidPublicKey) {
		t.Fatalf("expected ErrInvalidPublicKey, got %v", err)
	}
}
