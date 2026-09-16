package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/uvini-wso2/plg-activity-service/internal/email"
	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
	"github.com/uvini-wso2/plg-activity-service/internal/validation"
)

// generateEmailResponse is the shared response shape for both the
// successful and skipped-generation cases, so the JSON contract stays
// consistent either way (GeneratedEmail/Note use omitempty so only the
// relevant one appears).
type generateEmailResponse struct {
	Outcome        validation.Outcome `json:"outcome"`
	Tags           []string           `json:"tags"`
	GeneratedEmail string             `json:"generatedEmail,omitempty"`
	Note           string             `json:"note,omitempty"`
}

// GenerateEmail handles GET /generate-email?company_id=...&user_id=...&email=...&domain=...&category=...
//
// Runs the same Search -> Normalize -> Classify pipeline as /validate,
// then feeds the result into a Generator (Claude) to produce a
// personalized outreach email using the team's shared template.
//
// ASSUMPTION (2026-09-15, not yet confirmed with team): email generation
// is SKIPPED for Excluded prospects — no point drafting outreach for
// someone already decided against. Confirm if this should behave
// differently (e.g. still generate for audit/preview purposes).
func GenerateEmail(client eventsClient, generator email.Generator) http.HandlerFunc {
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
			Email:    emailAddr,
			Domain:   domain,
			Category: category,
		}
		validationResult := validation.Classify(ec, summary)

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
			ActivitySummary:  email.SummarizeActivity(summary),
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
