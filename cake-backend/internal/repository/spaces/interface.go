package spaces

import "uuid"

type SpaceRepository interface {
	GetAllUserSpaces(ownerUuid uuid.UUID) ([]Space, error)
	GetUserSpace(ownerUuid, spaceUuid uuid.UUID) (Space, error)
	CreateSpace(space Space) (Space, error)
	UpdateSpace(space Space) (Space, error)
	DeleteSpace(ownerUuid, spaceUuid uuid.UUID) error
}

var _ SpaceRepository = (*Repository)(nil)
