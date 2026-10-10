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
	ContextLength   uint32         `db:"context_length"`
	ProviderModelId string         `db:"provider_model_id"`
	ProviderId      int            `db:"provider_id"`
	CreatedAt       time.Time      `db:"created_at"`
	Activated       bool           `db:"activated"`
}

type ModelWithProviderUuid struct {
	Model
	ProviderUuid string `db:"provider_uuid"`
}
