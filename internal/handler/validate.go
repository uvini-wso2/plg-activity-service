package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
	"github.com/uvini-wso2/plg-activity-service/internal/validation"
)

// Validate handles GET /asgardeo/validate?company_id=...&user_id=...&email=...
//
// UPDATED (2026-09-30): now calls the real classification API internally,
// given just an email — replacing the earlier temporary approach where
// the caller had to pass domain/category directly, since that API didn't
// exist yet.
func Validate(client eventsClient, classifier classificationClient) http.HandlerFunc {
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
		if email == "" {
			http.Error(w, `{"error":"email query parameter is required"}`, http.StatusBadRequest)
			return
		}

		classified, err := classifier.Classify(email)
		if err != nil {
			slog.Error("email classification failed", "error", err)
			http.Error(w, `{"error":"failed to classify email"}`, http.StatusBadGateway)
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
			Domain:   classified.Domain,
			Category: classified.Category,
		}

		validationResult := validation.Classify(ec, summary)

		response := struct {
			Outcome validation.Outcome `json:"outcome"`
			Tags    []string           `json:"tags"`
			Email   string             `json:"email"`
			Domain  string             `json:"domain"`
			moesif.Summary
			EventsFound int `json:"eventsFound"`
		}{
			Outcome:     validationResult.Outcome,
			Tags:        validationResult.Tags,
			Email:       email,
			Domain:      classified.Domain,
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
