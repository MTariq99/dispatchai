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
	curve := elliptic.P256()
	if len(encoded) != 65 || encoded[0] != 0x04 {
		return nil, errs.ErrInvalidPublicKey
	}
	x, y := elliptic.Unmarshal(curve, encoded)
	if x == nil || y == nil || !curve.IsOnCurve(x, y) {
		return nil, errs.ErrInvalidPublicKey
	}
	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}

func VerifySignature(publicKey *ecdsa.PublicKey, canonicalReq []byte, signatureBase64 string) error {
	if publicKey == nil || publicKey.Curve == nil || publicKey.Curve.Params().Name != elliptic.P256().Params().Name || publicKey.X == nil || publicKey.Y == nil || !elliptic.P256().IsOnCurve(publicKey.X, publicKey.Y) {
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
	if !ecdsa.VerifyASN1(publicKey, digest[:], signature) {
		return errs.ErrInvalidSignature
	}

	return nil
}
