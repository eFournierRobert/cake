package providers

import (
	providersRepo "efournierrobert/cake-backend/internal/repository/providers"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	repo *providersRepo.Repository
}

func New(db *sqlx.DB) *Service {
	return &Service{
		repo: providersRepo.New(db),
	}
}
