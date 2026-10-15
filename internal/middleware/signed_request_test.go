package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	errs "github.com/mtariq99/dispatchai/internal/errors"
	"github.com/mtariq99/dispatchai/internal/integrationctx"
)

type fakeVerifier struct {
	id  string
	err error
}

func (f fakeVerifier) Verify(*http.Request) (string, error) { return f.id, f.err }

func runSigned(v RequestVerifier) (*httptest.ResponseRecorder, string, bool) {
	gin.SetMode(gin.TestMode)
	var gotID string
	reached := false
	r := gin.New()
	r.Use(SignedRequestAuth(v))
	r.GET("/x", func(c *gin.Context) {
		reached = true
		gotID, _ = integrationctx.Get(c)
		c.Status(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	return rec, gotID, reached
}

func TestSignedRequestAuthSuccessSetsIdentity(t *testing.T) {
	rec, id, reached := runSigned(fakeVerifier{id: "int-1"})
	if rec.Code != 200 || !reached || id != "int-1" {
		t.Fatalf("code=%d reached=%v id=%q", rec.Code, reached, id)
	}
}

func TestSignedRequestAuthFailuresAreGeneric401(t *testing.T) {
	failures := []error{
		errs.ErrInvalidSignature, errs.ErrReplayedRequest, errs.ErrSigKeyNotFound,
		errs.ErrStaleTimestamp, errs.ErrInvalidRequest, errs.ErrInvalidBodyHash,
		fmt.Errorf("wrapped: %w", errs.ErrInvalidSignature),
	}
	var first string
	for i, e := range failures {
		rec, _, reached := runSigned(fakeVerifier{err: e})
		if rec.Code != http.StatusUnauthorized || reached {
			t.Fatalf("%v: code=%d reached=%v", e, rec.Code, reached)
		}
		if i == 0 {
			first = rec.Body.String()
		} else if rec.Body.String() != first {
			t.Fatalf("response bodies differ and would act as an oracle: %q vs %q", first, rec.Body.String())
		}
	}
}

func TestSignedRequestAuthFailsClosed(t *testing.T) {
	rec, _, reached := runSigned(fakeVerifier{err: fmt.Errorf("x: %w", errs.ErrAuthUnavailable)})
	if rec.Code != http.StatusServiceUnavailable || reached {
		t.Fatalf("backend failure: code=%d reached=%v", rec.Code, reached)
	}
	rec, _, reached = runSigned(nil)
	if rec.Code != http.StatusServiceUnavailable || reached {
		t.Fatalf("nil verifier: code=%d reached=%v", rec.Code, reached)
	}
	rec, _, reached = runSigned(fakeVerifier{id: "  ", err: nil})
	if rec.Code != http.StatusUnauthorized || reached {
		t.Fatalf("blank identity: code=%d reached=%v", rec.Code, reached)
	}
	_ = errors.New // keep errors import if edited
}
