package project

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mtariq99/dispatchai/models"
)

// ProjectClient is the communication boundary between DispatchAI
// and the Host Project.
//
// It is responsible only for transport and communication.
// It does not contain business logic, authorization decisions,
// or tool execution logic.
type ProjectClient interface {
	ExecuteTool(ctx context.Context, req *models.ToolRequest) (*models.ToolResponse, error)
}

// Config contains configuration required by the Host Project client.

// Client communicates with the Host Project over HTTP.
type Client struct {
	BaseURL    string
	httpClient *http.Client
}

// NewClient creates a Host Project client.
func NewClient(cfg *models.Config) (*Client, error) {
	if strings.TrimSpace(cfg.ProjectConfig.BaseURL) == "" {
		return nil, fmt.Errorf("project base URL cannot be empty")
	}

	timeout := cfg.ProjectConfig.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &Client{
		BaseURL: strings.TrimRight(cfg.ProjectConfig.BaseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// ExecuteTool sends a tool execution request to the Host Project.
//
// The Host Project remains responsible for authentication,
// authorization, validation, business rules, and execution.
func (c *Client) ExecuteTool(ctx context.Context, req *models.ToolRequest) (*models.ToolResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("tool request cannot be nil")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal tool request: %w", err)
	}

	endpoint := c.BaseURL + "/api/v1/ai/tools/execute"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create project request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("project request failed: %w", err)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read project response: %w", err)
	}

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf(
			"project returned HTTP %d: %s",
			resp.StatusCode,
			string(responseBody),
		)
	}

	var result models.ToolResponse

	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode project response: %w", err)
	}

	return &result, nil
}
