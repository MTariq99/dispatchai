package authn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"fmt"

	errs "github.com/mtariq99/dispatchai/internal/errors"
)

func ParseP256PublicKey(encoded []byte) (*ecdsa.PublicKey, error) {
	if len(encoded) != 65 || encoded[0] != 0x04 {
		return nil, errs.ErrInvalidPublicKey
	}

	publicKey, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), encoded)
	if err != nil {
		return nil, errs.ErrInvalidPublicKey
	}

	return publicKey, nil
}

func VerifySignature(publicKey *ecdsa.PublicKey, canonicalReq []byte, signatureBase64 string) error {
	if publicKey == nil || publicKey.Curve == nil ||
		publicKey.Curve.Params().Name != elliptic.P256().Params().Name {
		return errs.ErrInvalidPublicKey
	}

	encodedKey, err := publicKey.Bytes()
	if err != nil {
		return errs.ErrInvalidPublicKey
	}

	validatedKey, err := ecdsa.ParseUncompressedPublicKey(
		elliptic.P256(),
		encodedKey,
	)
	if err != nil {
		return errs.ErrInvalidPublicKey
	}

	if len(canonicalReq) == 0 || signatureBase64 == "" {
		return errs.ErrInvalidSignature
	}

	signature, err := base64.StdEncoding.Strict().DecodeString(signatureBase64)
	if err != nil {
		return fmt.Errorf("%w: invalid Base64 encoding", errs.ErrInvalidSignature)
	}

	digest := sha256.Sum256(canonicalReq)
	if !ecdsa.VerifyASN1(validatedKey, digest[:], signature) {
		return errs.ErrInvalidSignature
	}

	return nil
}
