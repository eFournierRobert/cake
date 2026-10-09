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
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	repo       *providersRepo.Repository
	encryptor  *encryptor.Encryptor
	httpClient *http.Client
}

func New(db *sqlx.DB, encryptor *encryptor.Encryptor) *Service {
	return &Service{
		repo:      providersRepo.New(db),
		encryptor: encryptor,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Service) GetAllProviders() ([]models.ProviderDto, error) {
	providers, err := s.repo.GetAllProviders()
	if err != nil {
		log.Printf("GetAllProviders error: %s\n", fmt.Errorf("%w", err))
		return nil, getAppErrorType(err)
	}

	var dto []models.ProviderDto
	for _, p := range providers {
		dto = append(dto, providerToDto(p))
	}

	return dto, nil
}

func (s *Service) GetProvider(uuidStr string) (models.ProviderDto, error) {
	provider, err := s.getProviderFromStrUuid(uuidStr)
	if err != nil {
		return models.ProviderDto{}, err
	}

	return providerToDto(provider), nil
}

func (s *Service) CreateProvider(dto models.ProviderCreate) (models.ProviderDto, error) {
	if !isValidProviderURL(dto.BaseUrl) {
		return models.ProviderDto{}, handler_errors.ErrInvalidRequest
	}

	provider := providersRepo.Provider{
		Uuid:    uuid.New().String(),
		Name:    dto.Name,
		BaseUrl: strings.TrimSuffix(dto.BaseUrl, "/"),
		ApiKey:  nil,
	}

	if len(dto.ApiKey) > 0 {
		apiKeyBlob, err := s.encryptor.Encrypt(dto.ApiKey)
		if err != nil {
			log.Printf("CreateProvider error: %s\n", fmt.Errorf("%w", err))
			return models.ProviderDto{}, getAppErrorType(err)
		}

		provider.ApiKey = apiKeyBlob
	}

	newProvider, err := s.repo.CreateProvider(provider)
	if err != nil {
		log.Printf("CreateProvider error: %s\n", fmt.Errorf("%w", err))
		return models.ProviderDto{}, getAppErrorType(err)
	}

	return providerToDto(newProvider), nil
}

func (s *Service) ModifyProvider(uuidStr string, dto models.ProviderUpdate) (models.ProviderDto, error) {
	provider, err := s.getProviderFromStrUuid(uuidStr)
	if err != nil {
		return models.ProviderDto{}, err
	}

	if len(dto.BaseUrl) > 0 && !isValidProviderURL(dto.BaseUrl) {
		return models.ProviderDto{}, handler_errors.ErrInvalidRequest
	}

	if dto.ApiKey != nil {
		if len(*dto.ApiKey) == 0 {
			provider.ApiKey = nil
		} else {
			apiKeyBlob, err := s.encryptor.Encrypt(*dto.ApiKey)
			if err != nil {
				log.Printf("ModifyProvider error: %s\n", fmt.Errorf("%w", err))
				return models.ProviderDto{}, getAppErrorType(err)
			}

			provider.ApiKey = apiKeyBlob
		}
	}

	if len(dto.Name) > 0 {
		provider.Name = dto.Name
	}
	if len(dto.BaseUrl) > 0 {
		provider.BaseUrl = strings.TrimSuffix(dto.BaseUrl, "/")
	}

	modifiedProvider, err := s.repo.UpdateProvider(provider)
	if err != nil {
		log.Printf("ModifyProvider error: %s\n", fmt.Errorf("%w", err))
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
	provider, err := s.getProviderFromStrUuid(uuidStr)
	if err != nil {
		return models.ProviderTestResponse{}, err
	}

	req, err := http.NewRequest(http.MethodGet, provider.BaseUrl+"/models", nil)
	if err != nil {
		log.Printf("TestProvider error: %s\n", fmt.Errorf("%w", err))
		return models.ProviderTestResponse{}, getAppErrorType(err)
	}

	if len(provider.ApiKey) > 0 {
		apiKey, err := s.encryptor.Decrypt(provider.ApiKey)
		if err != nil {
			log.Printf("TestProvider error: %s\n", fmt.Errorf("%w", err))
			return models.ProviderTestResponse{}, getAppErrorType(err)
		}

		req.Header.Set("Authorization", "Bearer "+string(apiKey))
	}

	start := time.Now()

	resp, err := s.httpClient.Do(req)

	latencyMs := time.Since(start).Milliseconds()

	if err != nil {
		log.Printf("TestProvider error: %s\n", err)
		return models.ProviderTestResponse{
			Success:   false,
			LatencyMs: &latencyMs,
			Error:     "error happened when testing the provider",
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

// getProviderFromStrUuid parses a string UUID and retrieves the provider from
// the repository. It returns handler_errors.ErrInvalidRequest if the UUID is
// invalid, or handler_errors.ErrProviderNotFound if no provider is found.
func (s *Service) getProviderFromStrUuid(strUuid string) (providersRepo.Provider, error) {
	realUuid, err := uuid.Parse(strUuid)
	if err != nil {
		log.Printf("getProviderFromStrUuid error: %s\n", fmt.Errorf("%w", err))
		return providersRepo.Provider{}, handler_errors.ErrInvalidRequest
	}

	provider, err := s.repo.GetProvider(realUuid)
	if err != nil {
		log.Printf("getProviderFromStrUuid error: %s\n", fmt.Errorf("%w", err))
		return providersRepo.Provider{}, getAppErrorType(err)
	}

	return provider, nil
}

// isValidProviderURL reports whether s is an absolute http(s) URL with a
// host. Empty strings, relative references (which url.Parse happily
// accepts), and non-http(s) schemes are rejected: the base URL is the
// only provider field that is ever dialed, so it is the only one that
// needs this check.
func isValidProviderURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Host == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
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
