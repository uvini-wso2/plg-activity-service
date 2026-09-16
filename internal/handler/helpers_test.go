package handler

import (
	"errors"

	"github.com/uvini-wso2/plg-activity-service/internal/apim"
	"github.com/uvini-wso2/plg-activity-service/internal/email"
	"github.com/uvini-wso2/plg-activity-service/internal/moesif"
)

// mockMoesifClient is a test double for eventsClient. Configure Response
// and Err before use in a test; LastCriteria captures the FilterCriteria
// passed to the most recent Search call, so tests can assert the handler
// built the right query.
type mockMoesifClient struct {
	Response     moesif.SearchResponse
	Err          error
	LastCriteria moesif.FilterCriteria
}

func (m *mockMoesifClient) Search(criteria moesif.FilterCriteria) (moesif.SearchResponse, error) {
	m.LastCriteria = criteria
	if m.Err != nil {
		return moesif.SearchResponse{}, m.Err
	}
	return m.Response, nil
}

var errFake = errors.New("simulated moesif failure")

// mockGenerator is a test double for email.Generator.
type mockGenerator struct {
	Response string
	Err      error
	LastCall email.Prompt
}

func (m *mockGenerator) Generate(p email.Prompt) (string, error) {
	m.LastCall = p
	if m.Err != nil {
		return "", m.Err
	}
	return m.Response, nil
}

// mockAPIMClient is a test double for apimClient.
type mockAPIMClient struct {
	Response     apim.SearchResponse
	Err          error
	LastCriteria apim.FilterCriteria
}

func (m *mockAPIMClient) Search(criteria apim.FilterCriteria) (apim.SearchResponse, error) {
	m.LastCriteria = criteria
	if m.Err != nil {
		return apim.SearchResponse{}, m.Err
	}
	return m.Response, nil
}
