package providers

import (
	"efournierrobert/cake-backend/internal/encryptor"
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	providersRepo "efournierrobert/cake-backend/internal/repository/providers"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	repo      *providersRepo.Repository
	encryptor *encryptor.Encryptor
}

func New(db *sqlx.DB, encryptor *encryptor.Encryptor) *Service {
	return &Service{
		repo:      providersRepo.New(db),
		encryptor: encryptor,
	}
}

func (s *Service) GetAllProviders() ([]models.ProviderDto, error) {
	providers, err := s.repo.GetAllProviders()
	if err != nil {
		return nil, getAppErrorType(err)
	}

	var dto []models.ProviderDto
	for _, p := range providers {
		dto = append(dto, providerToDto(p))
	}

	return dto, nil
}

func (s *Service) GetProvider(uuidStr string) (models.ProviderDto, error) {
	realUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return models.ProviderDto{}, handler_errors.ErrInvalidRequest
	}

	provider, err := s.repo.GetProvider(realUuid)
	if err != nil {
		return models.ProviderDto{}, getAppErrorType(err)
	}

	return providerToDto(provider), nil
}

func (s *Service) CreateProvider(dto models.ProviderCreate) (models.ProviderDto, error) {
	apiKeyBlob, err := s.encryptor.Encrypt(dto.ApiKey)
	if err != nil {
		return models.ProviderDto{}, getAppErrorType(err)
	}

	provider := providersRepo.Provider{
		Uuid:    uuid.New().String(),
		Name:    dto.Name,
		BaseUrl: dto.BaseUrl,
		ApiKey:  apiKeyBlob,
	}

	newProvider, err := s.repo.CreateProvider(provider)
	if err != nil {
		return models.ProviderDto{}, getAppErrorType(err)
	}

	return providerToDto(newProvider), nil
}

func (s *Service) ModifyProvider(uuidStr string, dto models.ProviderUpdate) (models.ProviderDto, error) {
	realUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return models.ProviderDto{}, handler_errors.ErrInvalidRequest
	}

	provider, err := s.repo.GetProvider(realUuid)
	if err != nil {
		return models.ProviderDto{}, getAppErrorType(err)
	}

	if dto.ApiKey != nil {
		apiKeyBlob, err := s.encryptor.Encrypt(*dto.ApiKey)
		if err != nil {
			return models.ProviderDto{}, getAppErrorType(err)
		}

		provider.ApiKey = apiKeyBlob
	}

	if len(dto.Name) > 0 {
		provider.Name = dto.Name
	}
	if len(dto.BaseUrl) > 0 {
		provider.BaseUrl = dto.BaseUrl
	}

	modifiedProvider, err := s.repo.UpdateProvider(provider)
	if err != nil {
		return models.ProviderDto{}, getAppErrorType(err)
	}

	return providerToDto(modifiedProvider), nil
}

func (s *Service) DeleteProvider(uuidStr string) error {
	realUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return handler_errors.ErrInvalidRequest
	}

	if err := s.repo.DeleteProvider(realUuid); err != nil {
		return getAppErrorType(err)
	}

	return nil
}

func (s *Service) TestProvider(uuidStr string) (models.ProviderTestResponse, error) {
	realUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return models.ProviderTestResponse{}, getAppErrorType(err)
	}

	provider, err := s.repo.GetProvider(realUuid)
	if err != nil {
		return models.ProviderTestResponse{}, getAppErrorType(err)
	}

	req, err := http.NewRequest(http.MethodGet, provider.BaseUrl+"/models", nil)
	if err != nil {
		return models.ProviderTestResponse{}, getAppErrorType(err)
	}

	apiKey, err := s.encryptor.Decrypt(provider.ApiKey)
	if err != nil {
		return models.ProviderTestResponse{}, getAppErrorType(err)
	}

	req.Header.Set("Authorization", "Bearer "+string(apiKey))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	start := time.Now()

	resp, err := client.Do(req)

	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		return models.ProviderTestResponse{
			Success:   false,
			LatencyMs: &latencyMs,
			Error:     err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.ProviderTestResponse{
			Success:   false,
			LatencyMs: &latencyMs,
			Error:     fmt.Sprintf("provider returned HTTP %d", resp.StatusCode),
		}, nil
	}

	return models.ProviderTestResponse{
		Success:   true,
		LatencyMs: &latencyMs,
		Error:     "",
	}, nil
}

func providerToDto(p providersRepo.Provider) models.ProviderDto {
	return models.ProviderDto{
		Uuid:      p.Uuid,
		Name:      p.Name,
		BaseUrl:   p.BaseUrl,
		HasApiKey: len(p.ApiKey) > 0,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// getAppErrorType maps repository errors to handler errors.
func getAppErrorType(err error) error {
	log.Println("error happened: " + err.Error())

	if errors.Is(err, &repo_errors.ProviderNotFound{}) {
		return handler_errors.ErrProviderNotFound
	}
	return handler_errors.ErrUnexpectedError
}
