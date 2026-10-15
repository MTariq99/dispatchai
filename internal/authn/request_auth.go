package authn

import (
	"bytes"
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/mtariq99/dispatchai/internal/enums"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/models"
)

type KeyLookup interface {
	LookupActiveKey(ctx context.Context, clientID, keyID string) (*models.KeyRecord, error)
}

type NonceStore interface {
	RecordNonce(ctx context.Context, integrationID, nonce string, expiresAt time.Time) (bool, error)
}

type VerifierConfig struct {
	MaxClockSkew time.Duration
	MaxBodyBytes int64
}

type RequestVerifier struct {
	keys     KeyLookup
	nonces   NonceStore
	maxSkew  time.Duration
	nonceTTL time.Duration
	maxBody  int64
	now      func() time.Time
}

func NewRequestVerifier(keys KeyLookup, nonces NonceStore, cfg VerifierConfig) (*RequestVerifier, error) {
	if keys == nil || nonces == nil {
		return nil, errors.New("key lookup and nonce store are required")
	}
	if cfg.MaxClockSkew <= 0 {
		cfg.MaxClockSkew = 5 * time.Minute
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = 1 << 20
	}
	return &RequestVerifier{
		keys:     keys,
		nonces:   nonces,
		maxSkew:  cfg.MaxClockSkew,
		nonceTTL: 2*cfg.MaxClockSkew + time.Minute,
		maxBody:  cfg.MaxBodyBytes,
		now:      time.Now,
	}, nil
}

type parsedRequest struct {
	headers   SignedHeaders
	signature string
	unix      int64
}

func singleHeader(r *http.Request, name string) (string, error) {
	values := r.Header.Values(name)
	if len(values) != 1 || values[0] == "" {
		return "", fmt.Errorf("%w: header %s must appear exactly once", errs.ErrInvalidRequest, name)
	}
	return values[0], nil
}

func parseHeaders(r *http.Request) (*parsedRequest, error) {
	p := &parsedRequest{}

	fields := []struct {
		name    string
		dst     *string
		pattern *regexp.Regexp
	}{
		{enums.HeaderClientID, &p.headers.ClientID, enums.IdPattern},
		{enums.HeaderKeyID, &p.headers.KeyID, enums.IdPattern},
		{enums.HeaderTimestamp, &p.headers.Timestamp, enums.TimestampPattern},
		{enums.HeaderNonce, &p.headers.Nonce, enums.NoncePattern},
		{enums.HeaderRequestID, &p.headers.RequestID, enums.IdPattern},
		{enums.HeaderContentSHA256, &p.headers.ContentSHA256, enums.HashPattern},
		{enums.HeaderSignature, &p.signature, enums.SignaturePattern},
	}
	for _, f := range fields {
		if err := GetHeader(r, f.name, f.dst, f.pattern); err != nil {
			return nil, err
		}
	}

	alg, err := singleHeader(r, enums.HeaderAlgorithm)
	if err != nil {
		return nil, err
	}
	if alg != enums.SignatureAlgorithm {
		return nil, fmt.Errorf("%w: unsupported signature algorithm", errs.ErrInvalidRequest)
	}
	p.headers.Algorithm = alg

	p.unix, err = strconv.ParseInt(p.headers.Timestamp, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: bad timestamp", errs.ErrInvalidRequest)
	}
	return p, nil
}

func GetHeader(r *http.Request, name string, dst *string, pattern *regexp.Regexp) error {
	v, err := singleHeader(r, name)
	if err != nil {
		return err
	}
	if !pattern.MatchString(v) {
		return fmt.Errorf("%w: header %s is malformed", errs.ErrInvalidRequest, name)
	}
	*dst = v
	return nil
}

func (v *RequestVerifier) Verify(r *http.Request) (string, error) {
	if v == nil || r == nil {
		return "", errs.ErrInvalidRequest
	}
	ctx := r.Context()

	p, err := parseHeaders(r)
	if err != nil {
		return "", err
	}

	skew := v.now().Sub(time.Unix(p.unix, 0))
	if skew < 0 {
		skew = -skew
	}
	log.Printf("timestamp diagnostic: request=%d server=%d skew=%s max_skew=%s", p.unix, v.now().Unix(), skew, v.maxSkew)
	if skew > v.maxSkew {
		return "", errs.ErrStaleTimestamp
	}

	rec, err := v.keys.LookupActiveKey(ctx, p.headers.ClientID, p.headers.KeyID)
	if errors.Is(err, errs.ErrSigKeyNotFound) {
		return "", err
	}
	if err != nil {
		return "", fmt.Errorf("%w: key lookup: %v", errs.ErrAuthUnavailable, err)
	}
	var body []byte
	if r.Body != nil {
		body, err = io.ReadAll(io.LimitReader(r.Body, v.maxBody+1))
		if err != nil {
			return "", fmt.Errorf("%w: cannot read body", errs.ErrInvalidRequest)
		}
		if int64(len(body)) > v.maxBody {
			return "", fmt.Errorf("%w: body too large", errs.ErrInvalidRequest)
		}
	}
	r.Body = io.NopCloser(bytes.NewReader(body)) // handlers can still read it
	log.Printf(
		"body hash diagnostic: bytes=%d actual=%s header=%s",
		len(body),
		BodySHA256(body),
		p.headers.ContentSHA256,
	)
	if subtle.ConstantTimeCompare([]byte(BodySHA256(body)), []byte(p.headers.ContentSHA256)) != 1 {
		return "", errs.ErrInvalidBodyHash
	}

	publicKey, err := ParseP256PublicKey(rec.PublicKey)
	if err != nil {
		return "", err
	}
	canonicalReq, err := CanonicalRequestFromHTTP(r, p.headers)
	if err != nil {
		return "", err
	}
	canonical, err := BuildCanonicalReq(canonicalReq)
	if err != nil {
		return "", err
	}
	if err = VerifySignature(publicKey, canonical, p.signature); err != nil {
		return "", err
	}

	inserted, err := v.nonces.RecordNonce(ctx, rec.IntegrationID, p.headers.Nonce, v.now().Add(v.nonceTTL))
	if err != nil {
		return "", fmt.Errorf("%w: nonce store: %v", errs.ErrAuthUnavailable, err)
	}
	if !inserted {
		return "", errs.ErrReplayedRequest
	}
	return rec.IntegrationID, nil
}
