package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/uvini-wso2/plg-activity-service/internal/apim"
)

// apimClient is the interface APIMEvents depends on — mirrors eventsClient's
// pattern, kept separate since apim.Client has its own independent types.
type apimClient interface {
	Search(criteria apim.FilterCriteria) (apim.SearchResponse, error)
}

// APIMEvents handles GET /apim/events?company_id=...&user_id=...
//
// Mirrors Events() (the Asgardeo handler) in structure, but uses the
// internal/apim package's own types throughout, since APIM's real data
// shape differs from Asgardeo's in confirmed ways (see internal/apim's
// doc comments).
func APIMEvents(client apimClient) http.HandlerFunc {
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

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summary); err != nil {
			http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
			return
		}
	}
}
