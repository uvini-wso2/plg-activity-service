package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uvini-wso2/plg-activity-service/internal/classification"
	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
)

func TestValidate_MissingIdentifiers(t *testing.T) {
	mock := &mockMoesifClient{}
	classifier := &mockClassificationClient{}
	req := httptest.NewRequest(http.MethodGet, "/asgardeo/validate?email=a@b.com", nil)
	rec := httptest.NewRecorder()

	Validate(mock, classifier)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestValidate_MissingEmail(t *testing.T) {
	mock := &mockMoesifClient{}
	classifier := &mockClassificationClient{}
	req := httptest.NewRequest(http.MethodGet, "/asgardeo/validate?company_id=company_456", nil)
	rec := httptest.NewRecorder()

	Validate(mock, classifier)(rec, req)

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
	classifier := &mockClassificationClient{
		Response: classification.Response{
			Email: "jane@acme.com", Domain: "acme.com", Category: classification.CategoryCorporate,
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/asgardeo/validate?company_id=company_456&email=jane@acme.com", nil)
	rec := httptest.NewRecorder()

	Validate(mock, classifier)(rec, req)

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
	if body["domain"] != "acme.com" {
		t.Errorf("expected domain = acme.com (from classification API), got %v", body["domain"])
	}
	if classifier.LastCall != "jane@acme.com" {
		t.Errorf("expected classifier to be called with jane@acme.com, got %q", classifier.LastCall)
	}
}

func TestValidate_WSO2Domain(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{
			Result: moesif.HitsResult{Hits: []moesif.RawHit{}, Total: 5},
		},
	}
	classifier := &mockClassificationClient{
		Response: classification.Response{
			Email: "jane@wso2.com", Domain: "wso2.com", Category: classification.CategoryCorporate,
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/asgardeo/validate?company_id=company_456&email=jane@wso2.com", nil)
	rec := httptest.NewRecorder()

	Validate(mock, classifier)(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["outcome"] != "PLG CS Excluded" {
		t.Errorf("expected outcome = PLG CS Excluded, got %v", body["outcome"])
	}
}

func TestValidate_MoesifError(t *testing.T) {
	mock := &mockMoesifClient{Err: errFake}
	classifier := &mockClassificationClient{
		Response: classification.Response{Domain: "b.com", Category: classification.CategoryCorporate},
	}
	req := httptest.NewRequest(http.MethodGet, "/asgardeo/validate?company_id=company_456&email=a@b.com", nil)
	rec := httptest.NewRecorder()

	Validate(mock, classifier)(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

func TestValidate_ClassificationError(t *testing.T) {
	mock := &mockMoesifClient{}
	classifier := &mockClassificationClient{Err: errFake}
	req := httptest.NewRequest(http.MethodGet, "/asgardeo/validate?company_id=company_456&email=a@b.com", nil)
	rec := httptest.NewRecorder()

	Validate(mock, classifier)(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

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
	classifier := &mockClassificationClient{
		Response: classification.Response{Domain: "gmail.com", Category: classification.CategoryPersonal},
	}
	req := httptest.NewRequest(http.MethodGet, "/asgardeo/validate?company_id=company_456&email=jane@gmail.com", nil)
	rec := httptest.NewRecorder()

	Validate(mock, classifier)(rec, req)

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if body["organizationName"] != "acme-corp" {
		t.Errorf("expected organizationName = acme-corp, got %v", body["organizationName"])
	}
	if body["outcome"] != "PLG CS Eligible" {
		t.Errorf("expected outcome = PLG CS Eligible (meaningful activity), got %v", body["outcome"])
	}
}

// TestValidate_UsesAccountOwnerEmailWhenNoEmailGiven confirms the new
// company_id-only flow (2026-09-30): when no email is given, the real
// account owner email from Moesif's own company data is used instead.
func TestValidate_UsesAccountOwnerEmailWhenNoEmailGiven(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{
			Result: moesif.HitsResult{
				Hits: []moesif.RawHit{
					{
						Source: moesif.RawSource{
							ActionName: moesif.ActionNameOnboardingStepCompleted,
							Request:    moesif.RawRequest{Time: "2026-09-07T08:30:21.000"},
							Company: moesif.RawCompany{Metadata: moesif.RawCompanyMetadata{
								AccountName:       "wayfinderenterprise",
								AccountOwnerEmail: "anuradhak@wso2.com",
							}},
						},
					},
				},
				Total: 1,
			},
		},
	}
	classifier := &mockClassificationClient{
		Response: classification.Response{Domain: "wso2.com", Category: classification.CategoryCorporate},
	}
	req := httptest.NewRequest(http.MethodGet, "/asgardeo/validate?company_id=company_456", nil)
	rec := httptest.NewRecorder()

	Validate(mock, classifier)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if classifier.LastCall != "anuradhak@wso2.com" {
		t.Errorf("expected classifier called with the account owner email, got %q", classifier.LastCall)
	}
}

// TestValidate_NoEmailAndNoAccountOwnerEmail confirms a clean 400 error
// when neither an explicit email NOR a discoverable account owner email
// exists.
func TestValidate_NoEmailAndNoAccountOwnerEmail(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{
			Result: moesif.HitsResult{Hits: []moesif.RawHit{}, Total: 0},
		},
	}
	classifier := &mockClassificationClient{}
	req := httptest.NewRequest(http.MethodGet, "/asgardeo/validate?company_id=company_456", nil)
	rec := httptest.NewRecorder()

	Validate(mock, classifier)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
