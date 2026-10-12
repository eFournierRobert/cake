package oai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OpenAi is an HTTP client for the OpenAI-compatible provider API.
type OpenAi struct {
	httpClient *http.Client
}

// New creates a provider API client with a ten-second request timeout.
func New() *OpenAi {
	return &OpenAi{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetModels fetches and decodes the model catalog at baseUrl + "/models".
// When apiKey is non-empty it is sent as a bearer token. Non-2xx responses,
// request failures, and invalid response JSON are returned as errors.
func (oai *OpenAi) GetModels(ctx context.Context, apiKey, baseUrl string) (GetModelsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseUrl+"/models", nil)
	if err != nil {
		return GetModelsResponse{}, err
	}

	if len(apiKey) > 0 {
		req.Header.Set("Authorization", "Bearer "+string(apiKey))
	}

	resp, err := oai.httpClient.Do(req)

	if err != nil {
		return GetModelsResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GetModelsResponse{}, fmt.Errorf("error: received %d from API %s", resp.StatusCode, baseUrl)
	}

	var getModelsResponse GetModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&getModelsResponse); err != nil {
		return GetModelsResponse{}, err
	}

	return getModelsResponse, nil
}
