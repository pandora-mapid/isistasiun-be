package copilot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client (owned by Priyapta) forwards copilot queries to the AI service
// (owned by Firaz — see internal/copilot/service.go for the actual AI logic
// when it's implemented as an in-process service instead of a separate HTTP
// service; swap this out if that's how the team decides to run it).
type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeoutSeconds int) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
	}
}

func (c *Client) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/copilot/query", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai service error (%d): %s", resp.StatusCode, string(respBody))
	}

	var out QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode ai service response: %w", err)
	}
	return &out, nil
}
