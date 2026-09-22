package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/uvini-wso2/plg-activity-service/internal/apim"
	"github.com/uvini-wso2/plg-activity-service/internal/validation"
)

// APIMValidate handles GET /apim/validate?company_id=...&user_id=...&email=...&domain=...&category=...
//
// Mirrors Validate() (the Asgardeo handler) in structure, but uses
// apim.Search/Normalize/Classify throughout. See apim.Classify's doc
// comment — this reuses the shared validation rule structure, with only
// the meaningful-activity signal and the isWSO2User check being
// APIM-specific. Not yet fully confirmed with the team (2026-09-22).
func APIMValidate(client apimClient) http.HandlerFunc {
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

		criteria := apim.FilterCriteria{
			CompanyID: companyID,
			UserID:    userID,
			From:      "-30d",
			To:        "now",
		}

		result, err := client.Search(criteria)
		if err != nil {
			slog.Error("apim moesif search failed", "error", err)
			http.Error(w, `{"error":"failed to fetch activity data"}`, http.StatusBadGateway)
			return
		}

		summary := apim.Normalize(result.Result.Hits, result.Result.Total)

		ec := validation.EmailClassification{
			Email:    email,
			Domain:   domain,
			Category: category,
		}

		validationResult := apim.Classify(ec, summary)

		response := struct {
			Outcome validation.Outcome `json:"outcome"`
			Tags    []string           `json:"tags"`
			Email   string             `json:"email,omitempty"`
			Domain  string             `json:"domain"`
			apim.Summary
		}{
			Outcome: validationResult.Outcome,
			Tags:    validationResult.Tags,
			Email:   email,
			Domain:  domain,
			Summary: summary,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
			return
		}
	}
}
