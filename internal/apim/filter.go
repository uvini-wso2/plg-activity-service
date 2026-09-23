package apim

// FilterCriteria holds the parameters used to build a Moesif search query.
type FilterCriteria struct {
	CompanyID string
	UserID    string
	From      string
	To        string
}

// BuildPostFilter constructs the Elasticsearch-style post_filter Moesif
// expects.
//
// PREFERS CompanyID over UserID when both are given (2026-09-22 team
// decision, based on a live, controlled investigation): most of APIM's
// detailed console-driven events (component created/deployed/tested,
// quick-start funnel steps, etc.) are tagged with company_id, and some do
// NOT carry a user_id at all — so combining both with AND can silently
// miss real events. If only UserID is given (no CompanyID), we still
// search by UserID alone, since that's all we have.
func BuildPostFilter(criteria FilterCriteria) map[string]interface{} {
	var must []map[string]interface{}

	if criteria.CompanyID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"company_id": criteria.CompanyID},
		})
	} else if criteria.UserID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"user_id": criteria.UserID},
		})
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{"must": must},
	}
}
