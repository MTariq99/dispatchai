package models

type KeyRecord struct {
	IntegrationID string
	PublicKey     []byte
}

type SignedHeaders struct {
	ClientID      string
	KeyID         string
	Timestamp     string
	Nonce         string
	RequestID     string
	ContentSHA256 string
	Algorithm     string
}

type CanonicalRequest struct {
	Method  string
	Path    string
	Query   string
	Headers SignedHeaders
}
