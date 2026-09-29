package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/uvini-wso2/plg-activity-service/internal/apim"
	"github.com/uvini-wso2/plg-activity-service/internal/email"
	"github.com/uvini-wso2/plg-activity-service/internal/validation"
)

// APIMGenerateEmail handles GET /apim/generate-email?company_id=...&user_id=...&email=...&domain=...&category=...
//
// Mirrors GenerateEmail() (the Asgardeo handler) in structure, but uses
// apim.Search/Normalize/Classify and email.SummarizeAPIMActivity, so it
// correctly reads APIM's own Summary shape instead of Asgardeo's.
func APIMGenerateEmail(client apimClient, generator email.Generator) http.HandlerFunc {
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

		emailAddr := strings.TrimSpace(r.URL.Query().Get("email"))
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
			Email:    emailAddr,
			Domain:   domain,
			Category: category,
		}
		validationResult := apim.Classify(ec, summary)

		w.Header().Set("Content-Type", "application/json")

		if validationResult.Outcome == validation.OutcomeExcluded {
			json.NewEncoder(w).Encode(generateEmailResponse{
				Outcome: validationResult.Outcome,
				Tags:    validationResult.Tags,
				Note:    "email generation skipped for Excluded prospects",
			})
			return
		}

		prompt := email.Prompt{
			OrganizationName: summary.OrganizationName,
			Outcome:          string(validationResult.Outcome),
			Tags:             validationResult.Tags,
			ActivitySummary:  email.SummarizeAPIMActivity(summary),
			EmailTemplate:    email.DefaultTemplate,
		}

		generated, err := generator.Generate(prompt)
		if err != nil {
			slog.Error("email generation failed", "error", err)
			http.Error(w, `{"error":"failed to generate email"}`, http.StatusBadGateway)
			return
		}

		json.NewEncoder(w).Encode(generateEmailResponse{
			Outcome:        validationResult.Outcome,
			Tags:           validationResult.Tags,
			GeneratedEmail: generated,
		})
	}
}
