package models

import "time"

type ModelDto struct {
	Uuid            string    `json:"uuid"`
	Name            string    `json:"name"`
	Description     *string   `json:"description"`
	ProviderModelId string    `json:"provider_model_id"`
	ProviderUuid    string    `json:"provider_uuid"`
	Activated       bool      `json:"activated"`
	CreatedAt       time.Time `json:"created_at"`
}

type ModelUpdate struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Activated   bool    `json:"activated"`
}
