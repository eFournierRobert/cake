// Package models implements business logic for retrieving, updating, and
// deleting provider models.
package models

import (
	"database/sql"
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	modelsHandler "efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/repository/models"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"errors"
	"fmt"
	"log"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type modelRepository interface {
	GetAllModels() ([]models.ModelWithProviderUuid, error)
	GetAllActivatedModels() ([]models.ModelWithProviderUuid, error)
	GetModel(uuid.UUID) (models.Model, error)
	GetModelWithProviderUuid(uuid.UUID) (models.ModelWithProviderUuid, error)
	UpdateModel(models.Model) error
	DeleteModel(uuid.UUID) error
}

// ModelService is the business-logic contract consumed by model HTTP handlers.
type ModelService interface {
	GetAllModels() ([]modelsHandler.ModelDto, error)
	GetAllActivatedModels() ([]modelsHandler.ModelDto, error)
	UpdateModel(uuid string, modelUpdate modelsHandler.ModelUpdate) (modelsHandler.ModelDto, error)
	DeleteModel(uuid string) error
}

// Service applies model business rules and maps repository models to API DTOs.
type Service struct {
	repo modelRepository
}

var _ ModelService = (*Service)(nil)

// New creates a model service backed by db.
func New(db *sqlx.DB) *Service {
	return &Service{
		repo: models.New(db),
	}
}

// GetAllModels returns all models, including inactive ones.
func (s *Service) GetAllModels() ([]modelsHandler.ModelDto, error) {
	allModels, err := s.repo.GetAllModels()
	if err != nil {
		log.Printf("GetAllModels error: %s", fmt.Errorf("%w", err).Error())
		return nil, handler_errors.ErrUnexpectedError
	}

	var dto []modelsHandler.ModelDto
	for _, m := range allModels {
		dto = append(dto, s.modelToDto(m))
	}

	if dto == nil {
		dto = []modelsHandler.ModelDto{}
	}

	return dto, nil
}

// GetAllActivatedModels returns only models currently enabled for use.
func (s *Service) GetAllActivatedModels() ([]modelsHandler.ModelDto, error) {
	allModels, err := s.repo.GetAllActivatedModels()
	if err != nil {
		log.Printf("GetAllActivatedModels error: %s", fmt.Errorf("%w", err).Error())
		return nil, handler_errors.ErrUnexpectedError
	}

	var dto []modelsHandler.ModelDto
	for _, m := range allModels {
		dto = append(dto, s.modelToDto(m))
	}

	if dto == nil {
		dto = []modelsHandler.ModelDto{}
	}

	return dto, nil
}

// UpdateModel changes the supplied editable fields on the model identified by
// strUuid and returns the updated DTO. Invalid UUIDs and missing models are
// mapped to client-facing errors.
func (s *Service) UpdateModel(strUuid string, modelUpdate modelsHandler.ModelUpdate) (modelsHandler.ModelDto, error) {
	model, err := s.getModelFromStrUuid(strUuid)
	if err != nil {
		return modelsHandler.ModelDto{}, err
	}

	if modelUpdate.Activated != model.Activated {
		model.Activated = modelUpdate.Activated
	}
	if len(modelUpdate.Name) != 0 {
		model.Name = modelUpdate.Name
	}
	if modelUpdate.Description != nil && len(*modelUpdate.Description) != 0 {
		model.Description = sql.NullString{
			String: *modelUpdate.Description,
			Valid:  true,
		}
	}

	err = s.repo.UpdateModel(model)
	if err != nil {
		log.Printf("UpdateModel error: %s", fmt.Errorf("%w", err).Error())
		return modelsHandler.ModelDto{}, handler_errors.ErrUnexpectedError
	}

	updatedModel, err := s.repo.GetModelWithProviderUuid(uuid.MustParse(model.Uuid))
	if err != nil {
		log.Printf("UpdateModel error: %s", fmt.Errorf("%w", err).Error())
		return modelsHandler.ModelDto{}, getAppErrorType(err)
	}

	return s.modelToDto(updatedModel), nil
}

// DeleteModel removes the model identified by strUuid. Invalid UUIDs, missing
// models, and repository failures are returned as client-facing errors.
func (s *Service) DeleteModel(strUuid string) error {
	realUuid, err := uuid.Parse(strUuid)
	if err != nil {
		log.Printf("DeleteModel error: %v\n", err)
		return handler_errors.ErrInvalidRequest
	}

	if err := s.repo.DeleteModel(realUuid); err != nil {
		return getAppErrorType(err)
	}

	return nil
}

func (s *Service) getModelFromStrUuid(strUuid string) (models.Model, error) {
	realUuid, err := uuid.Parse(strUuid)
	if err != nil {
		log.Printf("getModelFromStrUuid error: %s\n", fmt.Errorf("%w", err))
		return models.Model{}, handler_errors.ErrInvalidRequest
	}

	model, err := s.repo.GetModel(realUuid)
	if err != nil {
		log.Printf("getModelFromStrUuid error: %s\n", fmt.Errorf("%w", err))
		return models.Model{}, getAppErrorType(err)
	}

	return model, nil
}

func (s *Service) modelToDto(model models.ModelWithProviderUuid) modelsHandler.ModelDto {
	var description *string
	if model.Description.Valid {
		value := model.Description.String
		description = &value
	}

	return modelsHandler.ModelDto{
		Uuid:            model.Uuid,
		Name:            model.Name,
		Description:     description,
		ProviderModelId: model.ProviderModelId,
		ProviderUuid:    model.ProviderUuid,
		Activated:       model.Activated,
		CreatedAt:       model.CreatedAt,
	}
}

// getAppErrorType maps repository errors to handler errors.
func getAppErrorType(err error) error {
	log.Println("error happened: " + err.Error())

	var notFound *repo_errors.ModelNotFound
	if errors.As(err, &notFound) {
		return handler_errors.ErrModelNotFound
	}
	return handler_errors.ErrUnexpectedError
}
