package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
	"github.com/uvini-wso2/plg-activity-service/internal/validation"
)

// Validate handles GET /validate?company_id=...&user_id=...&email=...&domain=...&category=...
//
// Returns the validation outcome + reasoning tags, ALONGSIDE the full
// prospect activity summary (org info, tenure, product activity) — per
// the "Validation Insight Delivery" requirement: a CS engineer needs to
// see not just the decision, but the underlying data that produced it.
//
// TEMPORARY (2026-09-11): since the real classification API doesn't exist
// yet, this endpoint accepts the classification result directly as query
// params, rather than calling that API internally. Once the classification
// API is ready, replace the param-parsing below with a real API call, and
// the rest of this handler (Search → Classify → response) stays the same.
func Validate(client eventsClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		companyID, errMsg := validateParam("company_id", r.URL.Query().Get("company_id"))
		if errMsg != "" {
			http.Error(w, `{"error":"invalid company_id"}`, http.StatusBadRequest)
			return
		}
		userID, errMsg := validateParam("user_id", r.URL.Query().Get("user_id"))
		if errMsg != "" {
			http.Error(w, `{"error":"invalid user_id"}`, http.StatusBadRequest)
			return
		}
		if companyID == "" && userID == "" {
			http.Error(w, `{"error":"at least one of company_id or user_id query parameters is required"}`, http.StatusBadRequest)
			return
		}

		email := strings.TrimSpace(r.URL.Query().Get("email"))
		domain := strings.TrimSpace(r.URL.Query().Get("domain"))
		category := strings.TrimSpace(r.URL.Query().Get("category"))
		if domain == "" || category == "" {
			http.Error(w, `{"error":"domain and category query parameters are required"}`, http.StatusBadRequest)
			return
		}

		criteria := moesif.FilterCriteria{
			CompanyID: companyID,
			UserID:    userID,
			From:      "-30d",
			To:        "now",
		}

		result, err := client.Search(criteria)
		if err != nil {
			slog.Error("moesif search failed", "error", err)
			http.Error(w, `{"error":"failed to fetch activity data"}`, http.StatusBadGateway)
			return
		}

		summary := moesif.Normalize(result.Result.Hits)

		ec := validation.EmailClassification{
			Email:    email,
			Domain:   domain,
			Category: category,
		}

		validationResult := validation.Classify(ec, summary)

		// Response combines the validation decision with the full prospect
		// summary (org info, tenure, product activity) — same
		// Summary-embedding pattern used by /events, plus the classification
		// inputs and the decision itself layered on top.
		response := struct {
			Outcome validation.Outcome `json:"outcome"`
			Tags    []string           `json:"tags"`
			Email   string             `json:"email,omitempty"`
			Domain  string             `json:"domain"`
			moesif.Summary
			EventsFound int `json:"eventsFound"`
		}{
			Outcome:     validationResult.Outcome,
			Tags:        validationResult.Tags,
			Email:       email,
			Domain:      domain,
			Summary:     summary,
			EventsFound: result.Result.Total,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
			return
		}
	}
}
