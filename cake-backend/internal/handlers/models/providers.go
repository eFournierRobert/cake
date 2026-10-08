package models

import "time"

type ProviderDto struct {
	Uuid      string    `json:"uuid"`
	Name      string    `json:"name"`
	BaseUrl   string    `json:"base_url"`
	HasApiKey bool      `json:"has_api_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProviderCreate struct {
	Name    string `json:"name"`
	BaseUrl string `json:"base_url"`
	ApiKey  string `json:"api_key,omitempty"`
}

type ProviderUpdate struct {
	Name    string `json:"name"`
	BaseUrl string `json:"base_url"`
	// Nil means keep the existing key; a non-nil value replaces it,
	// including an empty string to clear it.
	ApiKey *string `json:"api_key,omitempty"`
}

type ProviderTestResponse struct {
	Success   bool   `json:"success"`
	LatencyMs *int64 `json:"latency_ms,omitempty"`
	Error     string `json:"error,omitempty"`
}
