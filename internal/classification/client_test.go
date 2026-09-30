package classification

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClassify_Success(t *testing.T) {
	var capturedAuth string
	var capturedBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Test-Key")
		json.NewDecoder(r.Body).Decode(&capturedBody)

		resp := Response{
			Email:      "jane@acme.com",
			Domain:     "acme.com",
			Category:   CategoryCorporate,
			Rating:     97,
			Confidence: 0.92,
			Source:     "llm",
			Signals: Signals{
				SyntaxValid: true, MXValid: true, Disposable: false, FreeProvider: false, RoleBased: false,
			},
			Reasoning: "ACME is a generic company domain name with valid MX records.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, AuthHeader: "Test-Key", AuthValue: "test-token-123"})

	result, err := client.Classify("jane@acme.com")
	if err != nil {
		t.Fatalf("Classify returned an error: %v", err)
	}

	if capturedAuth != "test-token-123" {
		t.Errorf("expected Test-Key header = test-token-123, got %q", capturedAuth)
	}
	if capturedBody["email"] != "jane@acme.com" {
		t.Errorf("expected request body email = jane@acme.com, got %q", capturedBody["email"])
	}
	if result.Category != CategoryCorporate {
		t.Errorf("expected Category = corporate, got %q", result.Category)
	}
	if result.Domain != "acme.com" {
		t.Errorf("expected Domain = acme.com, got %q", result.Domain)
	}
	if !result.Signals.MXValid {
		t.Error("expected Signals.MXValid = true")
	}
}

func TestClassify_UpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"invalid or expired key"}`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL, AuthHeader: "Test-Key", AuthValue: "expired-token"})

	_, err := client.Classify("jane@acme.com")
	if err == nil {
		t.Error("expected an error when upstream returns non-200, got nil")
	}
}

func TestClassify_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	client := NewClient(Config{BaseURL: server.URL})

	_, err := client.Classify("jane@acme.com")
	if err == nil {
		t.Error("expected an error when response body isn't valid JSON, got nil")
	}
}
