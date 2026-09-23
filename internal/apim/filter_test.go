package apim

import "testing"

func TestBuildPostFilter_CompanyIDOnly(t *testing.T) {
	criteria := FilterCriteria{CompanyID: "company_789"}
	filter := BuildPostFilter(criteria)

	must := filter["bool"].(map[string]interface{})["must"].([]map[string]interface{})
	if len(must) != 1 {
		t.Fatalf("expected exactly 1 filter clause, got %d", len(must))
	}
	term := must[0]["term"].(map[string]interface{})
	if term["company_id"] != "company_789" {
		t.Errorf("expected company_id = company_789, got %v", term["company_id"])
	}
}

func TestBuildPostFilter_UserIDOnly(t *testing.T) {
	criteria := FilterCriteria{UserID: "user_123"}
	filter := BuildPostFilter(criteria)

	must := filter["bool"].(map[string]interface{})["must"].([]map[string]interface{})
	if len(must) != 1 {
		t.Fatalf("expected exactly 1 filter clause, got %d", len(must))
	}
	term := must[0]["term"].(map[string]interface{})
	if term["user_id"] != "user_123" {
		t.Errorf("expected user_id = user_123, got %v", term["user_id"])
	}
}

// TestBuildPostFilter_BothIdentifiers_PrefersCompanyID confirms the
// 2026-09-22 team decision: when BOTH are given, only company_id is used
// — user_id is dropped entirely, not combined with AND. This is a
// deliberate behavior difference from internal/moesif (Asgardeo), where
// both are combined.
func TestBuildPostFilter_BothIdentifiers_PrefersCompanyID(t *testing.T) {
	criteria := FilterCriteria{CompanyID: "company_789", UserID: "user_123"}
	filter := BuildPostFilter(criteria)

	must := filter["bool"].(map[string]interface{})["must"].([]map[string]interface{})
	if len(must) != 1 {
		t.Fatalf("expected exactly 1 filter clause (company_id only, user_id dropped), got %d: %v", len(must), must)
	}
	term := must[0]["term"].(map[string]interface{})
	if _, hasCompanyID := term["company_id"]; !hasCompanyID {
		t.Error("expected the single clause to filter on company_id")
	}
	if _, hasUserID := term["user_id"]; hasUserID {
		t.Error("expected user_id to be dropped entirely when company_id is also present")
	}
}

func TestBuildPostFilter_NoIdentifiers(t *testing.T) {
	filter := BuildPostFilter(FilterCriteria{})

	must := filter["bool"].(map[string]interface{})["must"].([]map[string]interface{})
	if len(must) != 0 {
		t.Errorf("expected no filter clauses when neither identifier is given, got %d", len(must))
	}
}
