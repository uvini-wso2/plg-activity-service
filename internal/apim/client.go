package apim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config holds the settings needed to construct a Client — same
// Config/Client/NewClient pattern as internal/moesif, using APIM's own
// separate Moesif credentials.
type Config struct {
	APIKey  string
	BaseURL string
}

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

const (
	pageSize = 100
	maxPages = 10
)

// Search fetches events matching criteria, automatically paginating up to
// maxPages (a safety cap of 1000 events) — same keyset/seek pagination
// approach as internal/moesif's Search().
func (c *Client) Search(criteria FilterCriteria) (SearchResponse, error) {
	var allHits []RawHit
	var lastSort []interface{}
	var total int

	for page := 0; page < maxPages; page++ {
		reqBody := map[string]interface{}{
			"post_filter": BuildPostFilter(criteria),
			"size":        pageSize,
			"sort": []map[string]interface{}{
				{"request.time": map[string]string{"order": "desc"}},
			},
		}
		if lastSort != nil {
			reqBody["search_after"] = lastSort
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return SearchResponse{}, fmt.Errorf("marshal request: %w", err)
		}

		url := fmt.Sprintf("%s/search/~/search/events?from=%s&to=%s", c.cfg.BaseURL, criteria.From, criteria.To)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return SearchResponse{}, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return SearchResponse{}, fmt.Errorf("call moesif: %w", err)
		}

		respBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return SearchResponse{}, fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			return SearchResponse{}, fmt.Errorf("moesif returned status %d: %s", resp.StatusCode, string(respBytes))
		}

		var pageResp SearchResponse
		if err := json.Unmarshal(respBytes, &pageResp); err != nil {
			return SearchResponse{}, fmt.Errorf("parse response: %w", err)
		}

		allHits = append(allHits, pageResp.Result.Hits...)
		total = pageResp.Result.Total

		if len(pageResp.Result.Hits) < pageSize {
			break // reached the last page
		}
		lastSort = pageResp.Result.Hits[len(pageResp.Result.Hits)-1].Sort
	}

	return SearchResponse{Result: HitsResult{Hits: allHits, Total: total}}, nil
}
