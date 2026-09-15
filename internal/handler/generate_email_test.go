package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
)

func TestGenerateEmail_MissingIdentifiers(t *testing.T) {
	mock := &mockMoesifClient{}
	gen := &mockGenerator{}
	req := httptest.NewRequest(http.MethodGet, "/generate-email?email=a@b.com&domain=b.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	GenerateEmail(mock, gen)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestGenerateEmail_ExcludedSkipsGeneration(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{Result: moesif.HitsResult{Hits: []moesif.RawHit{}, Total: 5}},
	}
	gen := &mockGenerator{Response: "should not be used"}
	req := httptest.NewRequest(http.MethodGet, "/generate-email?company_id=company_456&email=jane@wso2.com&domain=wso2.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	GenerateEmail(mock, gen)(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["outcome"] != "PLG CS Excluded" {
		t.Errorf("expected outcome = PLG CS Excluded, got %v", body["outcome"])
	}
	if _, exists := body["generatedEmail"]; exists {
		t.Error("expected NO generatedEmail field for an Excluded prospect")
	}
	if body["note"] == nil {
		t.Error("expected a note explaining generation was skipped")
	}
}

func TestGenerateEmail_EligibleGeneratesEmail(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{Result: moesif.HitsResult{Hits: []moesif.RawHit{}, Total: 0}},
	}
	gen := &mockGenerator{Response: "Hi there, saw you signed up..."}
	req := httptest.NewRequest(http.MethodGet, "/generate-email?company_id=company_456&email=jane@acme.com&domain=acme.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	GenerateEmail(mock, gen)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["outcome"] != "PLG CS Eligible" {
		t.Errorf("expected outcome = PLG CS Eligible, got %v", body["outcome"])
	}
	if body["generatedEmail"] != "Hi there, saw you signed up..." {
		t.Errorf("expected generatedEmail to match generator's response, got %v", body["generatedEmail"])
	}
	if gen.LastCall.OrganizationName == "" && gen.LastCall.Outcome == "" {
		t.Error("expected the generator to receive a populated Prompt")
	}
}

func TestGenerateEmail_GeneratorError(t *testing.T) {
	mock := &mockMoesifClient{
		Response: moesif.SearchResponse{Result: moesif.HitsResult{Hits: []moesif.RawHit{}, Total: 0}},
	}
	gen := &mockGenerator{Err: errFake}
	req := httptest.NewRequest(http.MethodGet, "/generate-email?company_id=company_456&email=jane@acme.com&domain=acme.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	GenerateEmail(mock, gen)(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}
