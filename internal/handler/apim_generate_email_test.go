package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uvini-wso2/plg-activity-service/internal/apim"
)

func TestAPIMGenerateEmail_MissingIdentifiers(t *testing.T) {
	mock := &mockAPIMClient{}
	gen := &mockGenerator{}
	req := httptest.NewRequest(http.MethodGet, "/apim/generate-email?domain=b.com&category=personal", nil)
	rec := httptest.NewRecorder()

	APIMGenerateEmail(mock, gen)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestAPIMGenerateEmail_ExcludedSkipsGeneration(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{Result: apim.HitsResult{Hits: []apim.RawHit{}, Total: 5}},
	}
	gen := &mockGenerator{Response: "should not be used"}
	req := httptest.NewRequest(http.MethodGet, "/apim/generate-email?company_id=company_456&domain=wso2.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	APIMGenerateEmail(mock, gen)(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["outcome"] != "PLG CS Excluded" {
		t.Errorf("expected outcome = PLG CS Excluded, got %v", body["outcome"])
	}
	if _, exists := body["generatedEmail"]; exists {
		t.Error("expected NO generatedEmail field for an Excluded prospect")
	}
}

func TestAPIMGenerateEmail_EligibleGeneratesEmail(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{
			Result: apim.HitsResult{
				Hits:  []apim.RawHit{{Source: apim.RawSource{ActionName: apim.ActionNameGatewayActivated, Request: apim.RawRequest{Time: "2026-09-14T08:30:00.000"}}}},
				Total: 1,
			},
		},
	}
	gen := &mockGenerator{Response: "Hi there, saw you activated your gateway..."}
	req := httptest.NewRequest(http.MethodGet, "/apim/generate-email?company_id=company_456&domain=acme.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	APIMGenerateEmail(mock, gen)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["outcome"] != "PLG CS Eligible" {
		t.Errorf("expected outcome = PLG CS Eligible, got %v", body["outcome"])
	}
	if body["generatedEmail"] != "Hi there, saw you activated your gateway..." {
		t.Errorf("expected generatedEmail to match generator's response, got %v", body["generatedEmail"])
	}
	if gen.LastCall.ActivitySummary == "" {
		t.Error("expected the generator to receive a non-empty APIM activity summary")
	}
}

func TestAPIMGenerateEmail_GeneratorError(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{Result: apim.HitsResult{Hits: []apim.RawHit{}, Total: 0}},
	}
	gen := &mockGenerator{Err: errFake}
	req := httptest.NewRequest(http.MethodGet, "/apim/generate-email?company_id=company_456&domain=acme.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	APIMGenerateEmail(mock, gen)(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}
