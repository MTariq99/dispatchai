package authn

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

func BodySHA256(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// BuildCanonicalRequest serializes request fields in a stable, versioned format.
//
// Protocol details:
//   - Fields are separated by a single LF byte.
//   - There is no trailing LF.
//   - An empty query is represented by an empty line.
//   - Path must be URL.EscapedPath() from the incoming request.
//   - Query must be URL.RawQuery from the incoming request.
//   - The body hash must be lowercase hexadecimal SHA-256.

func BuildCanonicalReq(req models.CanonicalRequest) ([]byte, error) {
	h := req.Headers

	if req.Method == "" || req.Path == "" {
		return nil, errs.Wrapf(errs.ErrInvalidRequest, nil, " : method and path are required")
	}
	if req.Method != strings.ToUpper(req.Method) {
		return nil, errs.Wrapf(errs.ErrInvalidRequest, nil, " : method must be uppercase")
	}
	if !strings.HasPrefix(req.Path, "/") {
		return nil, errs.Wrapf(errs.ErrInvalidRequest, nil, ": path must be absolute")
	}

	if strings.ContainsAny(req.Path, "\r\n") || strings.ContainsAny(req.Query, "\r\n") {
		return nil, errs.Wrapf(errs.ErrInvalidRequest, nil, ": invalid path or query")
	}

	fields := []string{
		h.ClientID,
		h.KeyID,
		h.Timestamp,
		h.Nonce,
		h.RequestID,
	}
	for _, field := range fields {
		if strings.TrimSpace(field) == "" ||
			strings.ContainsAny(field, "\r\n") {
			return nil, fmt.Errorf("%w: missing or invalid signed header", errs.ErrInvalidRequest)
		}
	}
	if h.Algorithm != enums.SignatureAlgorithm {
		return nil, fmt.Errorf("%w: unsupported signature algorithm", errs.ErrInvalidRequest)
	}
	if len(h.ContentSHA256) != 64 || strings.ToLower(h.ContentSHA256) != h.ContentSHA256 {
		return nil, errs.ErrInvalidBodyHash
	}

	decodedHash, err := hex.DecodeString(h.ContentSHA256)
	if err != nil || len(decodedHash) != sha256.Size {
		return nil, errs.ErrInvalidBodyHash
	}

	canonical := strings.Join([]string{
		enums.CanonicalVersion,
		req.Method,
		req.Path,
		req.Query,
		h.ClientID,
		h.KeyID,
		h.Timestamp,
		h.Nonce,
		h.RequestID,
		h.ContentSHA256,
		h.Algorithm,
	}, "\n")

	return []byte(canonical), nil
}

func CanonicalRequestFromHTTP(r *http.Request, headers models.SignedHeaders) (models.CanonicalRequest, error) {
	if r == nil || r.URL == nil {
		return models.CanonicalRequest{}, fmt.Errorf("%w: request URL is missing", errs.ErrInvalidRequest)
	}

	path := r.URL.EscapedPath()
	if path == "" {
		path = "/"
	}

	return models.CanonicalRequest{
		Method:  r.Method,
		Path:    path,
		Query:   r.URL.RawQuery,
		Headers: headers,
	}, nil
}
