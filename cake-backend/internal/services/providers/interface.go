package providers

import (
	"context"
	"efournierrobert/cake-backend/internal/handlers/models"
)

// ProviderService is the contract behind the API Providers operations.
// HTTP handlers depend on this interface so tests can substitute a mock.
type ProviderService interface {
	GetAllProviders() ([]models.ProviderDto, error)
	GetProvider(uuid string) (models.ProviderDto, error)
	CreateProvider(provider models.ProviderCreate) (models.ProviderDto, error)
	ModifyProvider(uuid string, provider models.ProviderUpdate) (models.ProviderDto, error)
	DeleteProvider(uuid string) error
	// TestProvider probes the provider's model endpoint and reports connection
	// failures in the returned response.
	TestProvider(ctx context.Context, uuid string) (models.ProviderTestResponse, error)
	// RefreshModelsForProvider synchronizes one provider's model catalog.
	RefreshModelsForProvider(ctx context.Context, uuid string) (models.ProviderRefreshResult, error)
	// RefreshModelsForAllProviders synchronizes every provider's model catalog.
	RefreshModelsForAllProviders(ctx context.Context) (models.ProviderRefreshResponse, error)
}

var _ ProviderService = (*Service)(nil)
