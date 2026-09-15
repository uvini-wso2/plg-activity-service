package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Config holds settings for the real Anthropic API client.
type Config struct {
	APIKey string
	// Model — CONFIRM the exact current model string against Anthropic's
	// docs before going live; not yet verified since API access is still
	// pending (2026-09-15). Using "claude-sonnet-4-5" as a placeholder.
	Model string
}

// Client is a real Generator implementation calling Anthropic's Messages API.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// systemPrompt instructs Claude to work ONLY from provided data — matches
// the "analyse only provided data, don't invent info" principle from the
// architecture doc.
const systemPrompt = `You are helping a Customer Success engineer write a short, genuine outreach email to a product signup.

Rules:
- Use ONLY the information provided below. Do not invent details about the person, their company, or their usage.
- If information is insufficient to personalize a claim, omit it rather than guessing.
- Keep the tone warm and low-pressure, not salesy.
- Base the email on the provided template, adapting it to reference the prospect's actual activity where relevant.`

func (c *Client) Generate(p Prompt) (string, error) {
	reqBody := map[string]interface{}{
		"model":      c.cfg.Model,
		"max_tokens": 1024,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": buildUserMessage(p)},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call anthropic: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anthropic returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	var sb strings.Builder
	for _, block := range result.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	return sb.String(), nil
}

func buildUserMessage(p Prompt) string {
	var sb strings.Builder
	sb.WriteString("Prospect organization: " + p.OrganizationName)
	sb.WriteString("\nClassification outcome: " + p.Outcome)
	if len(p.Tags) > 0 {
		sb.WriteString("\nReasoning tags: " + strings.Join(p.Tags, ", "))
	}
	sb.WriteString("\n\nKnown activity:\n" + p.ActivitySummary)
	sb.WriteString("\n\nTemplate to personalize:\n" + p.EmailTemplate)
	return sb.String()
}
