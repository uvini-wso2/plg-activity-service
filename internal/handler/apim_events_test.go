package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/uvini-wso2/plg-activity-service/internal/apim"
)

func TestAPIMEvents_MissingIdentifiers(t *testing.T) {
	mock := &mockAPIMClient{}
	req := httptest.NewRequest(http.MethodGet, "/apim/events", nil)
	rec := httptest.NewRecorder()

	APIMEvents(mock)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rec.Code)
	}
}

func TestAPIMEvents_CompanyIDOnly(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{
			Result: apim.HitsResult{Hits: []apim.RawHit{}, Total: 600},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/apim/events?company_id=company_789", nil)
	rec := httptest.NewRecorder()

	APIMEvents(mock)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if mock.LastCriteria.CompanyID != "company_789" {
		t.Errorf("expected CompanyID = company_789, got %q", mock.LastCriteria.CompanyID)
	}
}

func TestAPIMEvents_UserIDOnly(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{Result: apim.HitsResult{Hits: []apim.RawHit{}, Total: 10}},
	}
	req := httptest.NewRequest(http.MethodGet, "/apim/events?user_id=user_123", nil)
	rec := httptest.NewRecorder()

	APIMEvents(mock)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if mock.LastCriteria.UserID != "user_123" {
		t.Errorf("expected UserID = user_123, got %q", mock.LastCriteria.UserID)
	}
}

func TestAPIMEvents_MoesifError(t *testing.T) {
	mock := &mockAPIMClient{Err: errFake}
	req := httptest.NewRequest(http.MethodGet, "/apim/events?company_id=company_789", nil)
	rec := httptest.NewRecorder()

	APIMEvents(mock)(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", rec.Code)
	}
}

// TestAPIMEvents_MeaningfulActivityThreshold: REDEFINED (2026-09-24) —
// HasMeaningfulActivity now counts API-Invoked events specifically, not
// raw eventsFound. A huge Total with zero real API-Invoked events should
// NOT trigger it.
func TestAPIMEvents_MeaningfulActivityThreshold(t *testing.T) {
	var hits []apim.RawHit
	for i := 0; i < 400; i++ {
		hits = append(hits, apim.RawHit{Source: apim.RawSource{ActionName: apim.ActionNameAPIInvoked, Request: apim.RawRequest{Time: "2026-09-14T08:30:00.000"}}})
	}
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{Result: apim.HitsResult{Hits: hits, Total: 400}},
	}
	req := httptest.NewRequest(http.MethodGet, "/apim/events?company_id=company_789", nil)
	rec := httptest.NewRecorder()

	APIMEvents(mock)(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	productActivity, ok := body["productActivity"].(map[string]interface{})
	if !ok {
		t.Fatal("expected productActivity in response")
	}
	if productActivity["hasMeaningfulActivity"] != true {
		t.Errorf("expected hasMeaningfulActivity = true at exactly 400 API-Invoked events, got %v", productActivity["hasMeaningfulActivity"])
	}
}

func TestAPIMEvents_ZeroEventsFound(t *testing.T) {
	mock := &mockAPIMClient{
		Response: apim.SearchResponse{Result: apim.HitsResult{Hits: []apim.RawHit{}, Total: 0}},
	}
	req := httptest.NewRequest(http.MethodGet, "/apim/events?company_id=nonexistent", nil)
	rec := httptest.NewRecorder()

	APIMEvents(mock)(rec, req)

	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)

	if body["eventsFound"] != float64(0) {
		t.Errorf("expected eventsFound = 0, got %v", body["eventsFound"])
	}
}
