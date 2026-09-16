package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
)

func TestValidate_MissingIdentifiers(t *testing.T) {
	mock := &mockMoesifClient{}
	req := httptest.NewRequest(http.MethodGet, "/validate?email=a@b.com&domain=b.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	Validate(mock)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestValidate_MissingClassificationParams(t *testing.T) {
	mock := &mockMoesifClient{}
	req := httptest.NewRequest(http.MethodGet, "/validate?company_id=company_456", nil)
	rec := httptest.NewRecorder()

	Validate(mock)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestValidate_CorporateEmail(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{
			Result: moesif.HitsResult{Hits: []moesif.RawHit{}, Total: 0},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/validate?company_id=company_456&email=jane@acme.com&domain=acme.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	Validate(mock)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if body["outcome"] != "PLG CS Eligible" {
		t.Errorf("expected outcome = PLG CS Eligible, got %v", body["outcome"])
	}
	if body["email"] != "jane@acme.com" {
		t.Errorf("expected email echoed back, got %v", body["email"])
	}
	if body["domain"] != "acme.com" {
		t.Errorf("expected domain echoed back, got %v", body["domain"])
	}
}

func TestValidate_WSO2Domain(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{
			Result: moesif.HitsResult{Hits: []moesif.RawHit{}, Total: 5},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/validate?company_id=company_456&email=jane@wso2.com&domain=wso2.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	Validate(mock)(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["outcome"] != "PLG CS Excluded" {
		t.Errorf("expected outcome = PLG CS Excluded, got %v", body["outcome"])
	}
}

func TestValidate_MoesifError(t *testing.T) {
	mock := &mockMoesifClient{
		Err: errFake,
	}
	req := httptest.NewRequest(http.MethodGet, "/validate?company_id=company_456&email=a@b.com&domain=b.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	Validate(mock)(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

// TestValidate_IncludesFullSummary confirms the "Validation Insight
// Delivery" requirement: the response must include the underlying
// prospect data (org info, activity), not just the bare decision.
func TestValidate_IncludesFullSummary(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{
			Result: moesif.HitsResult{
				Hits: []moesif.RawHit{
					{
						Source: moesif.RawSource{
							ActionName: moesif.ActionNameOnboardingStepCompleted,
							Request:    moesif.RawRequest{Time: "2026-09-07T08:30:21.000"},
							Company:    moesif.RawCompany{Metadata: moesif.RawCompanyMetadata{AccountName: "acme-corp"}},
						},
					},
				},
				Total: 1,
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/validate?company_id=company_456&email=jane@gmail.com&domain=gmail.com&category=personal", nil)
	rec := httptest.NewRecorder()

	Validate(mock)(rec, req)

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if body["organizationName"] != "acme-corp" {
		t.Errorf("expected organizationName = acme-corp, got %v", body["organizationName"])
	}
	if body["eventsFound"] != float64(1) {
		t.Errorf("expected eventsFound = 1, got %v", body["eventsFound"])
	}
	productActivity, ok := body["productActivity"].(map[string]interface{})
	if !ok {
		t.Fatal("expected productActivity to be present in the response")
	}
	if productActivity["applicationCreated"] != true {
		t.Errorf("expected productActivity.applicationCreated = true, got %v", productActivity["applicationCreated"])
	}
	if body["outcome"] != "PLG CS Eligible" {
		t.Errorf("expected outcome = PLG CS Eligible (meaningful activity), got %v", body["outcome"])
	}
}

// TestValidate_EmailOptional confirms the fix (2026-09-16): domain and
// category are required, but email is NOT — Classify() never actually
// uses the email address itself for any decision, only category and
// domain, so requiring it was stricter than necessary.
func TestValidate_EmailOptional(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{
			Result: moesif.HitsResult{Hits: []moesif.RawHit{}, Total: 3},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/validate?company_id=company_456&domain=acme.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	Validate(mock)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 (email should be optional), got %d, body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if body["outcome"] != "PLG CS Eligible" {
		t.Errorf("expected outcome = PLG CS Eligible, got %v", body["outcome"])
	}
	if _, exists := body["email"]; exists {
		t.Error("expected NO email field in response when email wasn't provided (omitempty)")
	}
}

// TestValidate_MissingDomainOrCategory confirms domain and category are
// still genuinely required, even though email no longer is.
func TestValidate_MissingDomainOrCategory(t *testing.T) {
	mock := &mockMoesifClient{}
	req := httptest.NewRequest(http.MethodGet, "/validate?company_id=company_456&email=a@b.com", nil)
	rec := httptest.NewRecorder()

	Validate(mock)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 (domain/category still required), got %d", rec.Code)
	}
}
