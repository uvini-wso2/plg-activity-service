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
