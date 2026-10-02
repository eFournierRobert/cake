package users

import (
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	userHandler "efournierrobert/cake-backend/internal/handlers/users"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"efournierrobert/cake-backend/internal/repository/roles"
	userRepo "efournierrobert/cake-backend/internal/repository/users"
	"errors"
	"fmt"
	"log"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type Service struct {
	repo     *userRepo.Repository
	roleRepo *roles.Repository
}

func New(db *sqlx.DB) *Service {
	return &Service{
		repo: userRepo.New(db),
	}
}

func (s *Service) GetUser(strUuid string) (userHandler.UserDto, error) {
	realUuid, err := uuid.Parse(strUuid)
	if err != nil {
		log.Printf("GetUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	user, err := s.repo.GetUser(realUuid)
	if err != nil {
		log.Printf("GetUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	userDto, err := s.userToDto(user)
	if err != nil {
		log.Printf("GetUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	return userDto, nil
}

func (s *Service) userToDto(user userRepo.User) (userHandler.UserDto, error) {
	role, err := s.roleRepo.GetUserRole(user)
	if err != nil {
		return userHandler.UserDto{}, err
	}

	return userHandler.UserDto{
		Uuid:        user.Uuid,
		Username:    user.Username,
		Role:        role.Name,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		LastUpdated: user.UpdatedAt,
		CreatedAt:   user.CreatedAt,
	}, nil
}

func getAppErrorType(err error) error {
	if errors.Is(err, &repo_errors.UserNotFound{}) {
		return handler_errors.ErrUserDoesNotExist
	}

	return handler_errors.ErrUnexpectedError
}
