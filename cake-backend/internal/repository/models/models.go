// Package models provides persistence operations and row types for provider
// models.
package models

import (
	"database/sql"
	"time"
)

// Model is a row of the models table.
type Model struct {
	Id              int            `db:"id"`
	Uuid            string         `db:"uuid"`
	Name            string         `db:"name"`
	Description     sql.NullString `db:"description"`
	ProviderModelId string         `db:"provider_model_id"`
	ProviderId      int            `db:"provider_id"`
	CreatedAt       time.Time      `db:"created_at"`
	Activated       bool           `db:"activated"`
}

// ModelWithProviderUuid is a model row joined with its provider's public UUID.
type ModelWithProviderUuid struct {
	Model
	ProviderUuid string `db:"provider_uuid"`
}
