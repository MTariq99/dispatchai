package gemini

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mtariq99/dispatchai/models"
)

type RateLimitError struct {
	StatusCode int
	Message    string

	RetryAfter time.Duration
	DailyQuota bool

	Raw string
}

func (e *RateLimitError) Error() string {
	switch {
	case e.RetryAfter > 0 && e.DailyQuota:
		return fmt.Sprintf(
			"gemini rate limited: retry after %s; daily quota exhausted",
			e.RetryAfter,
		)

	case e.RetryAfter > 0:
		return fmt.Sprintf(
			"gemini rate limited: retry after %s",
			e.RetryAfter,
		)

	case e.DailyQuota:
		return "gemini rate limited: daily quota exhausted"

	case e.Message != "":
		return fmt.Sprintf(
			"gemini rate limited: %s",
			e.Message,
		)

	default:
		return "gemini rate limited"
	}
}

// ProviderError represents a non-429 provider failure.
//
// The caller can inspect StatusCode instead of parsing error strings.
//
// Raw is retained for diagnostics but must not be logged blindly.
type ProviderError struct {
	StatusCode int
	Status     string
	Message    string
	Raw        string
}

func (e *ProviderError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf(
			"gemini provider error: status=%d message=%s",
			e.StatusCode,
			e.Message,
		)
	}

	return fmt.Sprintf(
		"gemini provider error: status=%d",
		e.StatusCode,
	)
}

// geminiErrorResponse represents the common Gemini REST error envelope.

func parseGeminiError(raw []byte) models.GeminiErrorResponse {
	var response models.GeminiErrorResponse

	// We intentionally ignore the unmarshal error here.
	//
	// The original HTTP response is preserved in ProviderError.Raw so that
	// callers still receive useful diagnostic information.
	_ = json.Unmarshal(raw, &response)

	return response
}

func parseGeminiRateLimit(raw []byte) *RateLimitError {
	parsed := parseGeminiError(raw)

	result := &RateLimitError{
		StatusCode: 429,
		Message:    parsed.Error.Message,
		Raw:        string(raw),
	}

	for _, detail := range parsed.Error.Details {
		switch detail.Type {
		case "type.googleapis.com/google.rpc.RetryInfo":
			if duration, err := parseRetryDelay(detail.RetryDelay); err == nil {
				result.RetryAfter = duration
			}

		case "type.googleapis.com/google.rpc.QuotaFailure":
			if strings.Contains(
				detail.QuotaID,
				"PerDay",
			) {
				result.DailyQuota = true
			}
		}
	}

	return result
}

// parseRetryDelay handles Google's duration representation.
//
// Examples:
//
//	42s
//	1.5s
//	250ms
//	2m
func parseRetryDelay(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return 0, errors.New("empty retry delay")
	}

	if strings.HasSuffix(value, "s") {
		seconds := strings.TrimSuffix(value, "s")

		n, err := strconv.ParseFloat(seconds, 64)
		if err != nil {
			return 0, fmt.Errorf(
				"invalid retry delay %q: %w",
				value,
				err,
			)
		}

		return time.Duration(
			n * float64(time.Second),
		), nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid retry delay %q: %w",
			value,
			err,
		)
	}

	return duration, nil
}
