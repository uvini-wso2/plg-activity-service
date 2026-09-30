package classification

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config holds settings for the classification API client.
//
// AuthHeader / AuthValue: the REAL production authentication method is
// still unconfirmed as of 2026-09-30 — the only credential we've tested
// against is a "Test-Key" header carrying a SANDBOX-type token that
// expires in ~10 minutes. Kept configurable (rather than hardcoding
// "Test-Key") so switching to the real production method later is a
// config change, not a code change.
type Config struct {
	BaseURL    string // e.g. "https://apis-stg.wso2.com/llkq/plg-email-classifier/v1.0"
	AuthHeader string // e.g. "Test-Key" — CONFIRM the real production header name
	AuthValue  string
}

type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Classify calls the real classification API for a given email address.
func (c *Client) Classify(email string) (Response, error) {
	reqBody, err := json.Marshal(map[string]string{"email": email})
	if err != nil {
		return Response{}, fmt.Errorf("marshal request: %w", err)
	}

	url := c.cfg.BaseURL + "/classify-email"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return Response{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")
	if c.cfg.AuthHeader != "" {
		req.Header.Set(c.cfg.AuthHeader, c.cfg.AuthValue)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, fmt.Errorf("call classification api: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("classification api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var result Response
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return Response{}, fmt.Errorf("parse response: %w", err)
	}

	return result, nil
}
