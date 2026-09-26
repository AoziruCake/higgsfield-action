package higgsfield

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the production Higgsfield API host.
	DefaultBaseURL = "https://api.higgsfield.ai"
	userAgent      = "higgsfield-action/0.1.0"
)

// Client calls the Higgsfield HTTP API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient builds a Client. apiKey must be KEY_ID:KEY_SECRET.
func NewClient(apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		baseURL:    DefaultBaseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// WithBaseURL overrides the API host (tests only).
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = strings.TrimRight(baseURL, "/")
	return c
}

// SubmitImage starts an asynchronous image generation job.
func (c *Client) SubmitImage(ctx context.Context, model string, req ImageRequest) (SubmitResponse, error) {
	var zero SubmitResponse
	if err := ctx.Err(); err != nil {
		return zero, err
	}

	endpoint, err := c.modelURL(model)
	if err != nil {
		return zero, err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return zero, fmt.Errorf("encode request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return zero, fmt.Errorf("submit image: %w", err)
	}
	defer resp.Body.Close()

	if err := decodeJSON(resp, &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

// GetStatus fetches the current state from statusURL returned by SubmitImage.
func (c *Client) GetStatus(ctx context.Context, statusURL string) (RequestStatus, error) {
	var zero RequestStatus
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if strings.TrimSpace(statusURL) == "" {
		return zero, fmt.Errorf("status url is required")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return zero, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return zero, fmt.Errorf("get status: %w", err)
	}
	defer resp.Body.Close()

	if err := decodeJSON(resp, &zero); err != nil {
		return zero, err
	}
	return zero, nil
}

func (c *Client) modelURL(model string) (string, error) {
	model = strings.Trim(model, "/")
	if model == "" {
		return "", fmt.Errorf("model is required")
	}
	return c.baseURL + "/" + model, nil
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Key "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
}

func decodeJSON(resp *http.Response, dst any) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody struct {
			Detail string `json:"detail"`
		}
		_ = json.Unmarshal(body, &errBody)
		return &APIError{StatusCode: resp.StatusCode, Detail: errBody.Detail}
	}

	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
