package models

import "uuid"

type ModelRepository interface {
	GetAllModels() ([]ModelWithProviderUuid, error)
	GetAllActivatedModels() ([]ModelWithProviderUuid, error)
	GetModel(uuid.UUID) (Model, error)
	GetModelWithProviderUuid(uuid.UUID) (ModelWithProviderUuid, error)
	UpdateModel(Model) error
	DeleteModel(uuid.UUID) error
	// GetActiveModel returns the model for uuid only when it is activated.
	GetActiveModel(uuid uuid.UUID) (Model, error)
}

var _ ModelRepository = (*Repository)(nil)
