package models

import "time"

type ModelDto struct {
	Uuid            string    `json:"uuid"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	ContextLength   uint32    `json:"context_length"`
	ProviderModelId string    `json:"provider_model_id"`
	ProviderId      int       `json:"provider_id"`
	Activated       bool      `json:"activated"`
	CreatedAt       time.Time `json:"created_at"`
}

type ModelUpdate struct {
	Activated bool `json:"activated"`
}
