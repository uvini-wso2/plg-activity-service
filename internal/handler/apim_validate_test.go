package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uvini-wso2/plg-activity-service/internal/apim"
	"github.com/uvini-wso2/plg-activity-service/internal/classification"
)

func TestAPIMValidate_MissingIdentifiers(t *testing.T) {
	mock := &mockAPIMClient{}
	classifier := &mockClassificationClient{}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?email=a@b.com", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock, classifier)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestAPIMValidate_MissingEmail(t *testing.T) {
	mock := &mockAPIMClient{}
	classifier := &mockClassificationClient{}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock, classifier)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestAPIMValidate_MeaningfulActivity(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{
			Result: apim.HitsResult{
				Hits:  []apim.RawHit{{Source: apim.RawSource{ActionName: apim.ActionNameComponentDeployed, Request: apim.RawRequest{Time: "2026-09-14T08:30:00.000"}}}},
				Total: 1,
			},
		},
	}
	classifier := &mockClassificationClient{
		Response: classification.Response{Domain: "gmail.com", Category: classification.CategoryPersonal},
	}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789&email=jane@gmail.com", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock, classifier)(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["outcome"] != "PLG CS Eligible" {
		t.Errorf("expected outcome = PLG CS Eligible, got %v", body["outcome"])
	}
	if classifier.LastCall != "jane@gmail.com" {
		t.Errorf("expected classifier called with jane@gmail.com, got %q", classifier.LastCall)
	}
}

func TestAPIMValidate_MoesifError(t *testing.T) {
	mock := &mockAPIMClient{Err: errFake}
	classifier := &mockClassificationClient{
		Response: classification.Response{Domain: "acme.com", Category: classification.CategoryCorporate},
	}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789&email=a@acme.com", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock, classifier)(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

func TestAPIMValidate_ClassificationError(t *testing.T) {
	mock := &mockAPIMClient{}
	classifier := &mockClassificationClient{Err: errFake}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789&email=a@acme.com", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock, classifier)(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

// TestAPIMValidate_UsesFirstSeenUserEmailWhenNoEmailGiven confirms the new
// company_id-only flow (2026-09-30): when no email is given, APIM falls
// back to the earliest-seen user's email.
func TestAPIMValidate_UsesFirstSeenUserEmailWhenNoEmailGiven(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{
			Result: apim.HitsResult{
				Hits: []apim.RawHit{
					{
						Source: apim.RawSource{
							ActionName: apim.ActionNameGatewayActivated,
							Request:    apim.RawRequest{Time: "2026-09-14T08:30:00.000"},
							User:       apim.RawUser{Email: "jane@acme.com"},
						},
					},
				},
				Total: 1,
			},
		},
	}
	classifier := &mockClassificationClient{
		Response: classification.Response{Domain: "acme.com", Category: classification.CategoryCorporate},
	}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock, classifier)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if classifier.LastCall != "jane@acme.com" {
		t.Errorf("expected classifier called with the first-seen user's email, got %q", classifier.LastCall)
	}
}

// TestAPIMValidate_NoEmailAndNoUserEmailFound confirms a clean 400 when
// neither an explicit email NOR any user email exists in the data at all.
func TestAPIMValidate_NoEmailAndNoUserEmailFound(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{Result: apim.HitsResult{Hits: []apim.RawHit{}, Total: 0}},
	}
	classifier := &mockClassificationClient{}
	req := httptest.NewRequest(http.MethodGet, "/apim/validate?company_id=company_789", nil)
	rec := httptest.NewRecorder()

	APIMValidate(mock, classifier)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}
