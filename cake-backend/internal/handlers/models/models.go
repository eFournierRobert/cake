// Package models contains API request and response DTOs for models and
// providers.
package models

import "time"

// ModelDto is the client-facing representation of a model and its provider.
type ModelDto struct {
	Uuid            string    `json:"uuid"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	ProviderModelId string    `json:"provider_model_id"`
	ProviderUuid    string    `json:"provider_uuid"`
	Activated       bool      `json:"activated"`
	CreatedAt       time.Time `json:"created_at"`
}

// ModelUpdate contains the editable model fields accepted by an update
// request. An empty name or nil/empty description leaves the corresponding
// stored value unchanged; Activated is always set to the supplied value.
type ModelUpdate struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Activated   bool    `json:"activated"`
}
