// Package users implements the business logic behind the API Users operations.
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
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo     *userRepo.Repository
	roleRepo *roles.Repository
}

func New(db *sqlx.DB) *Service {
	return &Service{
		repo:     userRepo.New(db),
		roleRepo: roles.New(db),
	}
}

// GetUser returns the user with the given uuid as a UserDto. It returns
// handler_errors.ErrUserDoesNotExist when no user matches.
func (s *Service) GetUser(strUuid string) (userHandler.UserDto, error) {
	user, err := s.getUserFromStrUuid(strUuid)
	if err != nil {
		return userHandler.UserDto{}, err
	}

	userDto, err := s.userToDto(user)
	if err != nil {
		log.Printf("GetUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	return userDto, nil
}

// ModifyUser lets the identified user update their own first name, last
// name and username, and returns the updated user as a UserDto.
func (s *Service) ModifyUser(currentUserUuid string, userUpdate userHandler.UserUpdate) (userHandler.UserDto, error) {
	user, err := s.getUserFromStrUuid(currentUserUuid)
	if err != nil {
		return userHandler.UserDto{}, err
	}

	if len(userUpdate.FirstName) > 0 {
		user.FirstName = userUpdate.FirstName
	}
	if len(userUpdate.LastName) > 0 {
		user.LastName = userUpdate.LastName
	}
	if len(userUpdate.Username) > 0 {
		user.Username = userUpdate.Username
	}

	user, err = s.repo.UpdateUser(user)
	if err != nil {
		log.Printf("ModifyUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	userDto, err := s.userToDto(user)
	if err != nil {
		log.Printf("ModifyUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	return userDto, nil
}

// AdminModifyUser is the admin variant of ModifyUser for any user
// identified by uuid: it also changes the user's role.
func (s *Service) AdminModifyUser(userUuid string, userUpdate userHandler.AdminUserUpdate) (userHandler.UserDto, error) {
	user, err := s.getUserFromStrUuid(userUuid)
	if err != nil {
		return userHandler.UserDto{}, err
	}

	if len(userUpdate.Role) > 0 {
		role, err := s.roleRepo.GetRoleByName(userUpdate.Role)
		if err != nil {
			log.Printf("AdminModifyUser error: %s\n", fmt.Errorf("%w", err))
			return userHandler.UserDto{}, getAppErrorType(err)
		}

		user.RoleId = role.Id
	}

	if len(userUpdate.FirstName) > 0 {
		user.FirstName = userUpdate.FirstName
	}
	if len(userUpdate.LastName) > 0 {
		user.LastName = userUpdate.LastName
	}
	if len(userUpdate.Username) > 0 {
		user.Username = userUpdate.Username
	}

	user, err = s.repo.UpdateUser(user)
	if err != nil {
		log.Printf("AdminModifyUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	userDto, err := s.userToDto(user)
	if err != nil {
		log.Printf("AdminModifyUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	return userDto, nil
}

// ChangePassword updates the password of the identified user. The new
// password must be 12 to 72 characters long.
func (s *Service) ChangePassword(currentUserUuid string, newPassword string) error {
	if !isPasswordGoodLength(newPassword) {
		return handler_errors.ErrInvalidPassword
	}

	user, err := s.getUserFromStrUuid(currentUserUuid)
	if err != nil {
		return err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ChangePassword error: %s\n", fmt.Errorf("%w", err))
		return getAppErrorType(err)
	}

	user.PasswordHash = newHash

	user, err = s.repo.UpdateUser(user)
	if err != nil {
		log.Printf("ChangePassword error: %s\n", fmt.Errorf("%w", err))
		return getAppErrorType(err)
	}

	return nil
}

// GetAllUsers returns every user as a UserDto, newest first.
func (s *Service) GetAllUsers() ([]userHandler.UserDto, error) {
	users, err := s.repo.GetAllUsers()
	if err != nil {
		log.Printf("GetAllUsers error: %s\n", fmt.Errorf("%w", err))
		return nil, getAppErrorType(err)
	}

	var usersDto []userHandler.UserDto
	for _, u := range users {
		usersDto = append(usersDto, s.userWithRoleToDto(u))
	}

	return usersDto, nil
}

// CreateUser creates a user from a UserCreate request.
func (s *Service) CreateUser(userCreateDto userHandler.UserCreate) (userHandler.UserDto, error) {
	if !isPasswordGoodLength(userCreateDto.Password) {
		return userHandler.UserDto{}, handler_errors.ErrInvalidPassword
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(userCreateDto.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("CreateUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	user, err := s.userCreateDtoToUser(userCreateDto, newHash)
	if err != nil {
		log.Printf("CreateUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	user, err = s.repo.CreateUser(user)
	if err != nil {
		log.Printf("CreateUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	userDto, err := s.userToDto(user)
	if err != nil {
		log.Printf("CreateUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	return userDto, nil
}

// DeleteUser removes the user with the given uuid.
func (s *Service) DeleteUser(userUuid string) error {
	realUuid, err := uuid.Parse(userUuid)
	if err != nil {
		return handler_errors.ErrInvalidRequest
	}

	err = s.repo.DeleteUser(realUuid)
	if err != nil {
		return getAppErrorType(err)
	}
	return nil
}

func (s *Service) getUserFromStrUuid(strUuid string) (userRepo.User, error) {
	realUuid, err := uuid.Parse(strUuid)
	if err != nil {
		log.Printf("getUserFromStrUuid error: %s\n", fmt.Errorf("%w", err))
		return userRepo.User{}, handler_errors.ErrInvalidRequest
	}

	user, err := s.repo.GetUser(realUuid)
	if err != nil {
		log.Printf("getUserFromStrUuid error: %s\n", fmt.Errorf("%w", err))
		return userRepo.User{}, getAppErrorType(err)
	}

	return user, nil
}

func (s *Service) userCreateDtoToUser(user userHandler.UserCreate, hashedPassword []byte) (userRepo.User, error) {
	role, err := s.roleRepo.GetRoleByName(user.Role)
	if err != nil {
		return userRepo.User{}, err
	}

	return userRepo.User{
		Id:           0,
		Uuid:         uuid.NewV4().String(),
		Username:     user.Username,
		PasswordHash: hashedPassword,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		CreatedAt:    time.Time{},
		UpdatedAt:    time.Time{},
		RoleId:       role.Id,
	}, nil
}

func (s *Service) userWithRoleToDto(user userRepo.UserWithRole) userHandler.UserDto {
	return userHandler.UserDto{
		Uuid:        user.User.Uuid,
		Username:    user.User.Username,
		Role:        user.RoleName,
		FirstName:   user.User.FirstName,
		LastName:    user.User.LastName,
		LastUpdated: user.User.UpdatedAt,
		CreatedAt:   user.User.CreatedAt,
	}
}

func (s *Service) userToDto(user userRepo.User) (userHandler.UserDto, error) {
	role, err := s.roleRepo.GetRoleById(user.RoleId)
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
	if errors.Is(err, &repo_errors.UserAlreadyExists{}) {
		return handler_errors.ErrResourceConflict
	}
	if errors.Is(err, &repo_errors.RoleNotFound{}) {
		return handler_errors.ErrRoleNotFound
	}

	return handler_errors.ErrUnexpectedError
}

func isPasswordGoodLength(password string) bool {
	return len(password) >= 12 && len(password) <= 72
}
