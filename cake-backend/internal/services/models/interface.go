package models

import modelsHandler "efournierrobert/cake-backend/internal/handlers/models"

// ModelService is the business-logic contract consumed by model HTTP handlers.
type ModelService interface {
	GetAllModels() ([]modelsHandler.ModelDto, error)
	GetAllActivatedModels() ([]modelsHandler.ModelDto, error)
	UpdateModel(uuid string, modelUpdate modelsHandler.ModelUpdate) (modelsHandler.ModelDto, error)
	DeleteModel(uuid string) error
}
