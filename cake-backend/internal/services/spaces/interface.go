package spaces

import "efournierrobert/cake-backend/internal/handlers/models"

// SpaceService is the business-logic contract consumed by space HTTP handlers.
type SpaceService interface {
	GetAllSpaces(ownerUuidStr string) ([]models.SpaceDto, error)
	GetSpace(spaceUuidStr, ownerUuidStr string) (models.SpaceDto, error)
	CreateSpace(ownerUuidStr string, spaceToCreate models.SpaceCreate) (models.SpaceDto, error)
	UpdateSpace(ownerUuidStr, spaceUuidStr string, spaceUpdate models.SpaceUpdate) (models.SpaceDto, error)
	DeleteSpace(ownerUuidStr, spaceUuidStr string) error
}

var _ SpaceService = (*Service)(nil)
