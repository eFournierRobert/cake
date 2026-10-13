package spaces

import (
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"efournierrobert/cake-backend/internal/repository/spaces"
	"efournierrobert/cake-backend/internal/repository/users"
	"errors"
	"log"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	repo     *spaces.Repository
	userRepo *users.Repository
}

func New(db *sqlx.DB) *Service {
	return &Service{
		repo:     spaces.New(db),
		userRepo: users.New(db),
	}
}

func (s *Service) GetAllSpaces(ownerUuidStr string) ([]models.SpaceDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return nil, handler_errors.ErrInvalidRequest
	}

	allUserSpaces, err := s.repo.GetAllUserSpaces(ownerUuid)
	if err != nil {
		return nil, getAppErrorType(err)
	}

	var dto []models.SpaceDto
	for _, s := range allUserSpaces {
		dto = append(dto, spaceToDto(s))
	}

	return dto, nil
}

func (s *Service) GetSpace(spaceUuidStr, ownerUuidStr string) (models.SpaceDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return models.SpaceDto{}, handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return models.SpaceDto{}, handler_errors.ErrInvalidRequest
	}

	space, err := s.repo.GetUserSpace(ownerUuid, spaceUuid)
	if err != nil {
		return models.SpaceDto{}, getAppErrorType(err)
	}

	return spaceToDto(space), nil
}

func (s *Service) CreateSpace(ownerUuidStr string, spaceToCreate models.SpaceCreate) (models.SpaceDto, error) {
	user, err := s.getUserFromUuid(ownerUuidStr)
	if err != nil {
		return models.SpaceDto{}, err
	}

	space := spaces.Space{
		Uuid:         uuid.NewV4().String(),
		Name:         spaceToCreate.Name,
		SystemPrompt: spaceToCreate.SystemPrompt,
		OwnerId:      user.Id,
	}

	space, err = s.repo.CreateSpace(space)
	if err != nil {
		return models.SpaceDto{}, getAppErrorType(err)
	}

	return spaceToDto(space), nil
}

func (s *Service) UpdateSpace(ownerUuidStr, spaceUuidStr string, spaceUpdate models.SpaceUpdate) (models.SpaceDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return models.SpaceDto{}, handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return models.SpaceDto{}, handler_errors.ErrInvalidRequest
	}

	space, err := s.repo.GetUserSpace(ownerUuid, spaceUuid)
	if err != nil {
		return models.SpaceDto{}, getAppErrorType(err)
	}

	if spaceUpdate.Name != nil {
		space.Name = *spaceUpdate.Name
	}
	if spaceUpdate.SystemPrompt != nil {
		space.SystemPrompt = *spaceUpdate.SystemPrompt
	}

	space, err = s.repo.UpdateSpace(space)
	if err != nil {
		return models.SpaceDto{}, getAppErrorType(err)
	}

	return spaceToDto(space), nil
}

func (s *Service) DeleteSpace(ownerUuidStr, spaceUuidStr string) error {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return handler_errors.ErrInvalidRequest
	}

	err = s.repo.DeleteSpace(ownerUuid, spaceUuid)
	if err != nil {
		return getAppErrorType(err)
	}

	return nil
}

func (s *Service) getUserFromUuid(uuidStr string) (users.User, error) {
	userUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return users.User{}, getAppErrorType(err)
	}

	user, err := s.userRepo.GetUser(userUuid)
	if err != nil {
		return users.User{}, getAppErrorType(err)
	}

	return user, nil
}

func spaceToDto(s spaces.Space) models.SpaceDto {
	return models.SpaceDto{
		Uuid:         s.Uuid,
		Name:         s.Name,
		SystemPrompt: s.SystemPrompt,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// getAppErrorType maps repository errors to handler errors.
func getAppErrorType(err error) error {
	log.Println("error happened: " + err.Error())

	if errors.Is(err, &repo_errors.SpaceNotFound{}) {
		return handler_errors.ErrProviderNotFound
	}
	if errors.Is(err, &repo_errors.UserNotFound{}) {
		return handler_errors.ErrProviderNotFound
	}
	return handler_errors.ErrUnexpectedError
}
