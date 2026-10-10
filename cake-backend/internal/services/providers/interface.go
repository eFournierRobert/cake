package providers

import "efournierrobert/cake-backend/internal/handlers/models"

// ProviderService is the contract behind the API Providers operations.
// HTTP handlers depend on this interface so tests can substitute a mock.
type ProviderService interface {
	GetAllProviders() ([]models.ProviderDto, error)
	GetProvider(uuid string) (models.ProviderDto, error)
	CreateProvider(provider models.ProviderCreate) (models.ProviderDto, error)
	ModifyProvider(uuid string, provider models.ProviderUpdate) (models.ProviderDto, error)
	DeleteProvider(uuid string) error
	TestProvider(uuid string) (models.ProviderTestResponse, error)
}

var _ ProviderService = (*Service)(nil)
