package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mtariq99/dispatchai/internal/authn"
	"github.com/mtariq99/dispatchai/internal/enums"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: dispatchai-sign keygen -out FILE | request [flags]")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = runKeygen(os.Args[2:])
	case "request":
		err = runRequest(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// keygen writes the private key (0600, never overwritten) and prints the
// Base64 public key to stdout for the enrollment request.
func runKeygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	out := fs.String("out", "", "private key file to create")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return errors.New("-out is required")
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: "PRIVATE KEY", Bytes: der}); err != nil {
		return err
	}
	pub, err := priv.PublicKey.ECDH()
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "private key written to", *out, "(keep it secret; it is never sent anywhere)")
	fmt.Println(base64.StdEncoding.EncodeToString(pub.Bytes()))
	return nil
}

func loadKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("no PEM block in key file")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || priv.Curve != elliptic.P256() {
		return nil, errors.New("key must be an ECDSA P-256 private key")
	}
	return priv, nil
}

func runRequest(args []string) error {
	fs := flag.NewFlagSet("request", flag.ContinueOnError)
	keyFile := fs.String("key", "", "private key file")
	clientID := fs.String("client", "", "client id (cli_...)")
	keyID := fs.String("key-id", "", "enrolled key id")
	method := fs.String("method", "GET", "HTTP method")
	rawURL := fs.String("url", "", "full URL")
	bodyStr := fs.String("body", "", "request body")
	bodyFile := fs.String("body-file", "", "read body from file")
	idem := fs.String("idem", "", "Idempotency-Key header value")
	repeat := fs.Int("repeat", 1, "send the identical signed request N times (same nonce; tests replay)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *keyFile == "" || *clientID == "" || *keyID == "" || *rawURL == "" {
		return errors.New("-key, -client, -key-id and -url are required")
	}

	priv, err := loadKey(*keyFile)
	if err != nil {
		return err
	}
	u, err := url.Parse(*rawURL)
	if err != nil {
		return err
	}
	body := []byte(*bodyStr)
	if *bodyFile != "" {
		if body, err = os.ReadFile(*bodyFile); err != nil {
			return err
		}
	}

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	headers := authn.SignedHeaders{
		ClientID:      *clientID,
		KeyID:         *keyID,
		Timestamp:     strconv.FormatInt(time.Now().Unix(), 10),
		Nonce:         base64.RawURLEncoding.EncodeToString(nonce),
		RequestID:     uuid.NewString(),
		ContentSHA256: authn.BodySHA256(body),
		Algorithm:     enums.SignatureAlgorithm,
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	m := strings.ToUpper(*method)
	canonical, err := authn.BuildCanonicalReq(authn.CanonicalRequest{
		Method: m, Path: path, Query: u.RawQuery, Headers: headers,
	})
	if err != nil {
		return err
	}
	digest := sha256.Sum256(canonical)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		return err
	}
	signature := base64.StdEncoding.EncodeToString(sig)

	client := &http.Client{Timeout: 15 * time.Second}
	for i := 1; i <= *repeat; i++ {
		req, err := http.NewRequest(m, *rawURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		if len(body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if *idem != "" {
			req.Header.Set("Idempotency-Key", *idem)
		}
		req.Header.Set(enums.HeaderClientID, headers.ClientID)
		req.Header.Set(enums.HeaderKeyID, headers.KeyID)
		req.Header.Set(enums.HeaderTimestamp, headers.Timestamp)
		req.Header.Set(enums.HeaderNonce, headers.Nonce)
		req.Header.Set(enums.HeaderRequestID, headers.RequestID)
		req.Header.Set(enums.HeaderContentSHA256, headers.ContentSHA256)
		req.Header.Set(enums.HeaderAlgorithm, headers.Algorithm)
		req.Header.Set(enums.HeaderSignature, signature)

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("[%d/%d] HTTP %d\n%s\n", i, *repeat, resp.StatusCode, respBody)
	}
	return nil
}
