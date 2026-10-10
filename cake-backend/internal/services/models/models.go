package models

import (
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

type Service struct {
	repo *models.Repository
}

func New(db *sqlx.DB) *Service {
	return &Service{
		repo: models.New(db),
	}
}

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

func (s *Service) UpdateModel(strUuid string, activated bool) (modelsHandler.ModelDto, error) {
	model, err := s.getModelFromStrUuid(strUuid)
	if err != nil {
		return modelsHandler.ModelDto{}, err
	}

	if activated != model.Activated {
		model.Activated = activated
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
	return modelsHandler.ModelDto{
		Uuid:            model.Uuid,
		Name:            model.Name,
		Description:     &model.Description.String,
		ContextLength:   model.ContextLength,
		ProviderModelId: model.ProviderModelId,
		ProviderUuid:    model.ProviderUuid,
		Activated:       model.Activated,
		CreatedAt:       model.CreatedAt,
	}
}

// getAppErrorType maps repository errors to handler errors.
func getAppErrorType(err error) error {
	log.Println("error happened: " + err.Error())

	if errors.Is(err, &repo_errors.ModelNotFound{}) {
		return handler_errors.ErrProviderNotFound
	}
	return handler_errors.ErrUnexpectedError
}
