package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uvini-wso2/plg-activity-service/internal/apim"
)

func TestAPIMValidate_MissingIdentifiers(t *testing.T) {
	mock := &mockAPIMClient{}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?domain=acme.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestAPIMValidate_MissingDomainOrCategory(t *testing.T) {
	mock := &mockAPIMClient{}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestAPIMValidate_EmailOptional(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{Result: apim.HitsResult{Hits: []apim.RawHit{}, Total: 5}},
	}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789&domain=acme.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rec.Code, rec.Body.String())
	}

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if _, exists := body["email"]; exists {
		t.Error("expected NO email field when not provided")
	}
	if body["outcome"] != "PLG CS Eligible" {
		t.Errorf("expected outcome = PLG CS Eligible, got %v", body["outcome"])
	}
}

func TestAPIMValidate_MeaningfulActivity(t *testing.T) {
	var hits []apim.RawHit
	for i := 0; i < 400; i++ {
		hits = append(hits, apim.RawHit{Source: apim.RawSource{ActionName: apim.ActionNameAPIInvoked, Request: apim.RawRequest{Time: "2026-09-14T08:30:00.000"}}})
	}
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{Result: apim.HitsResult{Hits: hits, Total: 400}},
	}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789&domain=gmail.com&category=personal", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock)(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["outcome"] != "PLG CS Eligible" {
		t.Errorf("expected outcome = PLG CS Eligible, got %v", body["outcome"])
	}
}

func TestAPIMValidate_MoesifError(t *testing.T) {
	mock := &mockAPIMClient{Err: errFake}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789&domain=acme.com&category=corporate", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock)(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}
