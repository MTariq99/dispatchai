package enums

import "regexp"

const (
	SignatureAlgorithm  = "ECDSA-P256-SHA256"
	CanonicalVersion    = "DISPATCHAI-REQUEST-V1"
	HeaderClientID      = "X-Client-Id"
	HeaderKeyID         = "X-Key-Id"
	HeaderTimestamp     = "X-Timestamp"
	HeaderNonce         = "X-Nonce"
	HeaderRequestID     = "X-Request-Id"
	HeaderContentSHA256 = "X-Content-SHA256"
	HeaderAlgorithm     = "X-Signature-Algorithm"
	HeaderSignature     = "X-Signature"
)

var (
	IdPattern        = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	NoncePattern     = regexp.MustCompile(`^[A-Za-z0-9_-]{16,64}$`)
	HashPattern      = regexp.MustCompile(`^[0-9a-f]{64}$`)
	TimestampPattern = regexp.MustCompile(`^[0-9]{1,12}$`)
	SignaturePattern = regexp.MustCompile(`^[A-Za-z0-9+/]{1,128}={0,2}$`)
)
