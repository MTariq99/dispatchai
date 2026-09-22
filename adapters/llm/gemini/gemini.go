package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"

	"github.com/mtariq99/dispatchai/models"
)

// This package contains the concrete Gemini integration.
//
// It adapts Gemini's API to DispatchAI's generic LLM interface.
//
// All Gemini-specific request and response handling should remain
// inside this adapter.

type GeminiClient struct {
	httpClient   *http.Client
	generateURL  string
	apiKey       string
	maxTokens    int
	temperature  float64
	limiter      *rate.Limiter
	DefaultModel string
}

func NewGeminiClient(cfg *models.Config) *GeminiClient {
	if cfg.LLM.GenerateURL == "" {
		fmt.Println("cfg.LLM.GenerateURL is empty")
		return nil
	}

	if cfg.LLM.GeminiAPIKey == "" {
		fmt.Println("cfg.LLM.GeminiAPIKey is empty")
		return nil
	}

	if cfg.LLM.MaxTokens <= 0 {
		fmt.Println("cfg.LLM.MaxTokens must be greater than zero")
		return nil
	}

	return &GeminiClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		generateURL: cfg.LLM.GenerateURL,
		apiKey:      cfg.LLM.GeminiAPIKey,
		maxTokens:   cfg.LLM.MaxTokens,
		temperature: cfg.LLM.Temperature,
		limiter: rate.NewLimiter(
			rate.Every(13*time.Second),
			1,
		),
	}
}

func (c *GeminiClient) Generate(ctx context.Context, req models.Request) (*models.Response, error) {
	if err := c.ValidateRequest(req); err != nil {
		return nil, err
	}
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait failed: %w", err)
	}

	geminiReq, err := BuildGeminiRequest(req, c.maxTokens, c.temperature)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, err
	}
	requestURL, err := c.buildRequestURL(req.Model)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create gemini request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("gemini request canceled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("gemini HTTP request failed: %w", err)
	}

	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read gemini response: %w", err)
	}

	if err := handleHTTPError(resp.StatusCode, raw); err != nil {
		return nil, err
	}

	result, err := parseGeminiResponse(raw)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *GeminiClient) ValidateRequest(req models.Request) error {
	if len(req.Messages) == 0 {
		return errors.New(
			"gemini request requires at least one message",
		)
	}

	if req.Model == "" && c.DefaultModel == "" {
		return errors.New(
			"gemini model is required",
		)
	}

	if req.MaxTokens < 0 {
		return errors.New(
			"gemini max tokens cannot be negative",
		)
	}

	if req.Temperature < 0 {
		return errors.New(
			"gemini temperature cannot be negative",
		)
	}

	return nil
}

func (c *GeminiClient) buildRequestURL(model string) (string, error) {
	if strings.TrimSpace(model) == "" {
		model = c.DefaultModel
	}

	base := strings.TrimSpace(c.generateURL)

	if base == "" {
		return "", errors.New("gemini generate URL is empty")
	}
	if strings.Contains(base, "{model}") {
		base = strings.ReplaceAll(base, "{model}", url.PathEscape(model))
		return c.addAPIKey(base)
	}

	if strings.Contains(base, ":generateContent") {
		return c.addAPIKey(base)
	}
	base = strings.TrimRight(base, "/")

	base += "/" + url.PathEscape(model) + ":generateContent"

	return c.addAPIKey(base)
}

func (c *GeminiClient) addAPIKey(rawURL string) (string, error) {
	if strings.TrimSpace(c.apiKey) == "" {
		return "", errors.New("gemini api key is empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse gemini URL: %w", err)
	}
	query := parsed.Query()
	query.Set("key", c.apiKey)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func handleHTTPError(statusCode int, raw []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	parsed := parseGeminiError(raw)
	if statusCode == http.StatusTooManyRequests {
		return parseGeminiRateLimit(raw)
	}
	return &ProviderError{
		StatusCode: statusCode,
		Status:     parsed.Error.Status,
		Message:    parsed.Error.Message,
		Raw:        string(raw),
	}
}
