package models

import (
	"database/sql"
	"testing"
	"uuid"

	handlerErrors "efournierrobert/cake-backend/internal/handlers/handler_errors"
	modelsHandler "efournierrobert/cake-backend/internal/handlers/models"
	modelsRepo "efournierrobert/cake-backend/internal/repository/models"
	"efournierrobert/cake-backend/internal/repository/repo_errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockModelRepository struct {
	allModels         []modelsRepo.ModelWithProviderUuid
	allModelsErr      error
	activatedModels   []modelsRepo.ModelWithProviderUuid
	activatedErr      error
	model             modelsRepo.Model
	modelErr          error
	modelWithProvider modelsRepo.ModelWithProviderUuid
	modelWithErr      error
	updatedModel      modelsRepo.Model
	updateErr         error
	deletedUuid       uuid.UUID
	deleteErr         error
}

func (m *mockModelRepository) GetAllModels() ([]modelsRepo.ModelWithProviderUuid, error) {
	return m.allModels, m.allModelsErr
}

func (m *mockModelRepository) GetAllActivatedModels() ([]modelsRepo.ModelWithProviderUuid, error) {
	return m.activatedModels, m.activatedErr
}

func (m *mockModelRepository) GetModel(uuid.UUID) (modelsRepo.Model, error) {
	return m.model, m.modelErr
}

func (m *mockModelRepository) GetModelWithProviderUuid(uuid.UUID) (modelsRepo.ModelWithProviderUuid, error) {
	return m.modelWithProvider, m.modelWithErr
}

func (m *mockModelRepository) UpdateModel(model modelsRepo.Model) error {
	m.updatedModel = model
	return m.updateErr
}

func (m *mockModelRepository) DeleteModel(id uuid.UUID) error {
	m.deletedUuid = id
	return m.deleteErr
}

func serviceModel() modelsRepo.ModelWithProviderUuid {
	return modelsRepo.ModelWithProviderUuid{
		Model: modelsRepo.Model{
			Uuid:            uuid.NewV4().String(),
			Name:            "model-name",
			Description:     sql.NullString{String: "description", Valid: true},
			ProviderModelId: "provider-model-id",
			ProviderId:      17,
			Activated:       true,
		},
		ProviderUuid: uuid.NewV4().String(),
	}
}

func TestGetAllModels(t *testing.T) {
	row := serviceModel()
	repo := &mockModelRepository{allModels: []modelsRepo.ModelWithProviderUuid{row}}
	svc := &Service{repo: repo}

	got, err := svc.GetAllModels()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, row.Uuid, got[0].Uuid)
	assert.Equal(t, row.ProviderUuid, got[0].ProviderUuid)
	require.NotNil(t, got[0].Description)
	assert.Equal(t, "description", *got[0].Description)

	repo.allModels = nil
	got, err = svc.GetAllModels()
	require.NoError(t, err)
	assert.NotNil(t, got, "empty results should encode as an empty array, not null")
	assert.Empty(t, got)

	repo.allModelsErr = &repo_errors.InternalDbError{Err: assert.AnError}
	_, err = svc.GetAllModels()
	assert.ErrorIs(t, err, handlerErrors.ErrUnexpectedError)
}

func TestGetAllActivatedModels(t *testing.T) {
	row := serviceModel()
	repo := &mockModelRepository{activatedModels: []modelsRepo.ModelWithProviderUuid{row}}
	svc := &Service{repo: repo}

	got, err := svc.GetAllActivatedModels()
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, row.Uuid, got[0].Uuid)
	assert.True(t, got[0].Activated)

	repo.activatedErr = assert.AnError
	_, err = svc.GetAllActivatedModels()
	assert.ErrorIs(t, err, handlerErrors.ErrUnexpectedError)

	repo.activatedErr = nil
	repo.activatedModels = nil
	got, err = svc.GetAllActivatedModels()
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestUpdateModel(t *testing.T) {
	row := serviceModel()
	row.Activated = false
	repo := &mockModelRepository{
		model:             row.Model,
		modelWithProvider: row,
	}
	svc := &Service{repo: repo}
	description := "updated description"

	got, err := svc.UpdateModel(row.Uuid, modelsHandler.ModelUpdate{
		Name:        "updated name",
		Description: &description,
		Activated:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, "updated name", repo.updatedModel.Name)
	assert.Equal(t, "updated description", repo.updatedModel.Description.String)
	assert.True(t, repo.updatedModel.Description.Valid)
	assert.True(t, repo.updatedModel.Activated)
	assert.Equal(t, row.ProviderUuid, got.ProviderUuid)

	_, err = svc.UpdateModel("not-a-uuid", modelsHandler.ModelUpdate{})
	assert.ErrorIs(t, err, handlerErrors.ErrInvalidRequest)

	repo.modelErr = &repo_errors.ModelNotFound{}
	_, err = svc.UpdateModel(uuid.NewV4().String(), modelsHandler.ModelUpdate{})
	assert.ErrorIs(t, err, handlerErrors.ErrModelNotFound)

	repo.modelErr = &repo_errors.InternalDbError{Err: assert.AnError}
	_, err = svc.UpdateModel(row.Uuid, modelsHandler.ModelUpdate{})
	assert.ErrorIs(t, err, handlerErrors.ErrUnexpectedError)
	repo.modelErr = nil
	repo.updateErr = assert.AnError
	_, err = svc.UpdateModel(row.Uuid, modelsHandler.ModelUpdate{})
	assert.ErrorIs(t, err, handlerErrors.ErrUnexpectedError)

	repo.updateErr = nil
	repo.modelWithErr = &repo_errors.ModelNotFound{}
	_, err = svc.UpdateModel(row.Uuid, modelsHandler.ModelUpdate{})
	assert.ErrorIs(t, err, handlerErrors.ErrModelNotFound)
}

func TestUpdateModelKeepsNilDescription(t *testing.T) {
	row := serviceModel()
	repo := &mockModelRepository{
		model:             row.Model,
		modelWithProvider: row,
	}
	svc := &Service{repo: repo}

	_, err := svc.UpdateModel(row.Uuid, modelsHandler.ModelUpdate{Activated: row.Activated})
	require.NoError(t, err)
	assert.Equal(t, row.Description, repo.updatedModel.Description)
}

func TestDeleteModel(t *testing.T) {
	repo := &mockModelRepository{}
	svc := &Service{repo: repo}
	id := uuid.NewV4()

	require.NoError(t, svc.DeleteModel(id.String()))
	assert.Equal(t, id, repo.deletedUuid)

	assert.ErrorIs(t, svc.DeleteModel("bad-uuid"), handlerErrors.ErrInvalidRequest)
	repo.deleteErr = &repo_errors.InternalDbError{Err: assert.AnError}
	assert.ErrorIs(t, svc.DeleteModel(id.String()), handlerErrors.ErrUnexpectedError)
	repo.deleteErr = &repo_errors.ModelNotFound{}
	assert.ErrorIs(t, svc.DeleteModel(id.String()), handlerErrors.ErrModelNotFound)
}

func TestModelToDtoPreservesNullDescription(t *testing.T) {
	row := serviceModel()
	row.Description = sql.NullString{}

	got := (&Service{}).modelToDto(row)
	assert.Nil(t, got.Description)
}

func TestGetAppErrorType(t *testing.T) {
	assert.ErrorIs(t, getAppErrorType(&repo_errors.ModelNotFound{}), handlerErrors.ErrModelNotFound)
	assert.ErrorIs(t, getAppErrorType(assert.AnError), handlerErrors.ErrUnexpectedError)
}
