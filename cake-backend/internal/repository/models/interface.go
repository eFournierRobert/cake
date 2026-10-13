package models

import "uuid"

type ModelRepository interface {
	GetAllModels() ([]ModelWithProviderUuid, error)
	GetAllActivatedModels() ([]ModelWithProviderUuid, error)
	GetModel(uuid.UUID) (Model, error)
	GetModelWithProviderUuid(uuid.UUID) (ModelWithProviderUuid, error)
	UpdateModel(Model) error
	DeleteModel(uuid.UUID) error
}

var _ ModelRepository = (*Repository)(nil)
