package models

import "time"

// ProviderDto is the client-facing representation of a provider. It exposes
// only whether an API key is stored, never the key itself.
type ProviderDto struct {
	Uuid      string    `json:"uuid"`
	Name      string    `json:"name"`
	BaseUrl   string    `json:"base_url"`
	HasApiKey bool      `json:"has_api_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProviderCreate is the request body used to create a provider.
type ProviderCreate struct {
	Name    string `json:"name"`
	BaseUrl string `json:"base_url"`
	ApiKey  string `json:"api_key,omitempty"`
}

// ProviderUpdate contains provider fields to modify. Nil ApiKey leaves the
// stored key unchanged; a non-nil empty ApiKey clears it.
type ProviderUpdate struct {
	Name    string `json:"name"`
	BaseUrl string `json:"base_url"`
	// Nil means keep the existing key; a non-nil value replaces it,
	// including an empty string to clear it.
	ApiKey *string `json:"api_key,omitempty"`
}

// ProviderTestResponse describes the result of a connectivity check.
type ProviderTestResponse struct {
	Success   bool   `json:"success"`
	LatencyMs *int64 `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ProviderRefreshResult describes one provider's model-catalog refresh.
type ProviderRefreshResult struct {
	Uuid       string `json:"provider_uuid"`
	Success    bool   `json:"success"`
	Discovered int    `json:"discovered"`
	Added      int    `json:"added"`
	Error      string `json:"error"`
}

// ProviderRefreshResponse contains the individual results from refreshing
// provider model catalogs.
type ProviderRefreshResponse struct {
	Results []ProviderRefreshResult `json:"results"`
}
