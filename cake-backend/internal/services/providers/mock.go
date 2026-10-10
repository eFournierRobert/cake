package providers

import "efournierrobert/cake-backend/internal/handlers/models"

// MockService is a hand-written test double for ProviderService, used by the
// HTTP handler tests. Each method records the arguments it receives and
// delegates to its corresponding Func field when one is configured.
type MockService struct {
	GetAllProvidersFunc func() ([]models.ProviderDto, error)

	GetProviderFunc func(uuid string) (models.ProviderDto, error)
	GetProviderArg  string

	CreateProviderFunc func(provider models.ProviderCreate) (models.ProviderDto, error)
	CreateProviderArg  models.ProviderCreate

	ModifyProviderFunc    func(uuid string, provider models.ProviderUpdate) (models.ProviderDto, error)
	ModifyProviderUuidArg string
	ModifyProviderArg     models.ProviderUpdate

	DeleteProviderFunc func(uuid string) error
	DeleteProviderArg  string

	TestProviderFunc func(uuid string) (models.ProviderTestResponse, error)
	TestProviderArg  string
}

var _ ProviderService = (*MockService)(nil)

func (m *MockService) GetAllProviders() ([]models.ProviderDto, error) {
	if m.GetAllProvidersFunc == nil {
		return nil, nil
	}
	return m.GetAllProvidersFunc()
}

func (m *MockService) GetProvider(uuid string) (models.ProviderDto, error) {
	m.GetProviderArg = uuid
	if m.GetProviderFunc == nil {
		return models.ProviderDto{}, nil
	}
	return m.GetProviderFunc(uuid)
}

func (m *MockService) CreateProvider(provider models.ProviderCreate) (models.ProviderDto, error) {
	m.CreateProviderArg = provider
	if m.CreateProviderFunc == nil {
		return models.ProviderDto{}, nil
	}
	return m.CreateProviderFunc(provider)
}

func (m *MockService) ModifyProvider(uuid string, provider models.ProviderUpdate) (models.ProviderDto, error) {
	m.ModifyProviderUuidArg = uuid
	m.ModifyProviderArg = provider
	if m.ModifyProviderFunc == nil {
		return models.ProviderDto{}, nil
	}
	return m.ModifyProviderFunc(uuid, provider)
}

func (m *MockService) DeleteProvider(uuid string) error {
	m.DeleteProviderArg = uuid
	if m.DeleteProviderFunc == nil {
		return nil
	}
	return m.DeleteProviderFunc(uuid)
}

func (m *MockService) TestProvider(uuid string) (models.ProviderTestResponse, error) {
	m.TestProviderArg = uuid
	if m.TestProviderFunc == nil {
		return models.ProviderTestResponse{}, nil
	}
	return m.TestProviderFunc(uuid)
}
