// Package spaces implements the business logic for user-owned conversation spaces.
package spaces

import (
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	spaceRepo "efournierrobert/cake-backend/internal/repository/spaces"
	userRepo "efournierrobert/cake-backend/internal/repository/users"
	"errors"
	"log"
	"uuid"

	"github.com/jmoiron/sqlx"
)

// Service applies ownership rules and maps persisted spaces to API DTOs.
type Service struct {
	repo     spaceRepo.SpaceRepository
	userRepo userRepo.UserRepository
}

// New creates a spaces service backed by db.
func New(db *sqlx.DB) *Service {
	return &Service{
		repo:     spaceRepo.New(db),
		userRepo: userRepo.New(db),
	}
}

// GetAllSpaces returns every space owned by ownerUuidStr. An empty result is
// returned as a non-nil empty slice. Invalid owner UUIDs return
// handler_errors.ErrInvalidRequest.
func (s *Service) GetAllSpaces(ownerUuidStr string) ([]models.SpaceDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return nil, handler_errors.ErrInvalidRequest
	}

	allUserSpaces, err := s.repo.GetAllUserSpaces(ownerUuid)
	if err != nil {
		return nil, getAppErrorType(err)
	}

	dto := []models.SpaceDto{}
	for _, s := range allUserSpaces {
		dto = append(dto, spaceToDto(s))
	}

	return dto, nil
}

// GetSpace returns the space identified by spaceUuidStr, provided it belongs
// to ownerUuidStr. Invalid UUIDs return handler_errors.ErrInvalidRequest and
// missing or non-owned spaces return handler_errors.ErrSpaceNotFound.
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

// CreateSpace creates a space owned by ownerUuidStr and returns its DTO. The
// owner must identify an existing user. The system prompt is stored as a
// string, including an empty string when omitted.
func (s *Service) CreateSpace(ownerUuidStr string, spaceToCreate models.SpaceCreate) (models.SpaceDto, error) {
	user, err := s.getUserFromUuid(ownerUuidStr)
	if err != nil {
		return models.SpaceDto{}, err
	}

	space := spaceRepo.Space{
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

// UpdateSpace applies the non-nil fields in spaceUpdate to the owner's space
// and returns the updated DTO. A nil field is left unchanged; an explicitly
// supplied empty string is stored as an empty string.
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

// DeleteSpace removes the space identified by spaceUuidStr when it belongs to
// ownerUuidStr. Invalid UUIDs return handler_errors.ErrInvalidRequest and a
// missing or non-owned space returns handler_errors.ErrSpaceNotFound.
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

func (s *Service) getUserFromUuid(uuidStr string) (userRepo.User, error) {
	userUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return userRepo.User{}, handler_errors.ErrInvalidRequest
	}

	user, err := s.userRepo.GetUser(userUuid)
	if err != nil {
		return userRepo.User{}, getAppErrorType(err)
	}

	return user, nil
}

func spaceToDto(s spaceRepo.Space) models.SpaceDto {
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

	if _, ok := errors.AsType[*repo_errors.SpaceNotFound](err); ok {
		return handler_errors.ErrSpaceNotFound
	}

	if _, ok := errors.AsType[*repo_errors.UserNotFound](err); ok {
		return handler_errors.ErrUserDoesNotExist
	}
	return handler_errors.ErrUnexpectedError
}
