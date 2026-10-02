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

func (s *Service) GetUser(strUuid string) (userHandler.UserDto, error) {
	user, err := s.getUserFromStrUuid(strUuid)
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

func (s *Service) ModifyUser(currentUserUuid string, userUpdate userHandler.UserUpdate) (userHandler.UserDto, error) {
	user, err := s.getUserFromStrUuid(currentUserUuid)
	if err != nil {
		log.Printf("ModifyUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	user.FirstName = userUpdate.FirstName
	user.LastName = userUpdate.LastName
	user.Username = userUpdate.Username

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

func (s *Service) AdminModifyUser(userUuid string, userUpdate userHandler.AdminUserUpdate) (userHandler.UserDto, error) {
	user, err := s.getUserFromStrUuid(userUuid)
	if err != nil {
		log.Printf("AdminModifyUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	role, err := s.roleRepo.GetRoleByName(userUpdate.Role)
	if err != nil {
		log.Printf("AdminModifyUser error: %s\n", fmt.Errorf("%w", err))
		return userHandler.UserDto{}, getAppErrorType(err)
	}

	user.FirstName = userUpdate.FirstName
	user.LastName = userUpdate.LastName
	user.Username = userUpdate.Username
	user.RoleId = role.Id

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

func (s *Service) ChangePassword(currentUserUuid string, newPassword string) error {
	if !isPasswordGoodLength(newPassword) {
		return handler_errors.ErrInvalidPassword
	}

	user, err := s.getUserFromStrUuid(currentUserUuid)
	if err != nil {
		log.Printf("ChangePassword error: %s\n", fmt.Errorf("%w", err))
		return getAppErrorType(err)
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

func (s *Service) GetAllUsers() ([]userHandler.UserDto, error) {
	users, err := s.repo.GetAllUsers()
	if err != nil {
		log.Printf("GetAllUsers error: %s\n", fmt.Errorf("%w", err))
		return nil, getAppErrorType(err)
	}

	var usersDto []userHandler.UserDto
	for _, u := range users {
		dto, err := s.userToDto(u)
		if err != nil {
			log.Printf("GetAllUsers error: %s\n", fmt.Errorf("%w", err))
			return nil, getAppErrorType(err)
		}

		usersDto = append(usersDto, dto)
	}

	return usersDto, nil
}

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

func (s *Service) DeleteUser(userUuid string) error {
	realUuid, err := uuid.Parse(userUuid)
	if err != nil {
		return getAppErrorType(err)
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
		return userRepo.User{}, err
	}

	user, err := s.repo.GetUser(realUuid)
	if err != nil {
		return userRepo.User{}, err
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

	return handler_errors.ErrUnexpectedError
}

func isPasswordGoodLength(password string) bool {
	return len(password) >= 12 && len(password) <= 72
}
