package apim

// FilterCriteria holds the parameters used to build a Moesif search query.
type FilterCriteria struct {
	CompanyID string
	UserID    string
	From      string
	To        string
}

// BuildPostFilter constructs the Elasticsearch-style post_filter Moesif
// expects — same query-building approach as internal/moesif.
func BuildPostFilter(criteria FilterCriteria) map[string]interface{} {
	var must []map[string]interface{}

	if criteria.CompanyID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"company_id": criteria.CompanyID},
		})
	}
	if criteria.UserID != "" {
		must = append(must, map[string]interface{}{
			"term": map[string]interface{}{"user_id": criteria.UserID},
		})
	}

	return map[string]interface{}{
		"bool": map[string]interface{}{"must": must},
	}
}
