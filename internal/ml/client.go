// Package ml is the HTTP client for the Python ML service (Phases 9-10).
package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"etl-telemetria-transformadores/internal/domain"
)

// SimilarResult is one candidate match with its similarity score.
type SimilarResult struct {
	TransformerID string  `json:"transformer_id"`
	Score         float64 `json:"score"`
}

// Client talks to the Python ML service.
type Client struct {
	baseURL string
	http    *http.Client
}

// New builds a client for the ML service base URL (e.g. http://localhost:8081).
func New(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Similar asks the ML service for the top-k similar transformers to target
// among candidates. The target itself should be excluded from candidates.
func (c *Client) Similar(ctx context.Context, target domain.Transformer, candidates []domain.Transformer, topK int) ([]SimilarResult, error) {
	body, err := json.Marshal(map[string]any{
		"target":     target,
		"candidates": candidates,
		"top_k":      topK,
	})
	if err != nil {
		return nil, fmt.Errorf("ml: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/similar", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ml: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ml: call /similar: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ml: /similar returned %d: %s", resp.StatusCode, truncate(raw))
	}
	var out struct {
		Results []SimilarResult `json:"results"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("ml: decode response: %w", err)
	}
	return out.Results, nil
}

func truncate(b []byte) string {
	const max = 300
	if len(b) > max {
		return string(b[:max]) + "..."
	}
	return string(b)
}
