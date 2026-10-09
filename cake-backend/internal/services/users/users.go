// Package users implements the business logic behind the API Users operations.
package users

import (
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"efournierrobert/cake-backend/internal/repository/roles"
	userRepo "efournierrobert/cake-backend/internal/repository/users"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo     *userRepo.Repository
	roleRepo *roles.Repository
}

// New creates a new User service with the given database connection.
func New(db *sqlx.DB) *Service {
	return &Service{
		repo:     userRepo.New(db),
		roleRepo: roles.New(db),
	}
}

// GetUser returns the user with the given uuid as a UserDto. It returns
// handler_errors.ErrUserDoesNotExist when no user matches.
func (s *Service) GetUser(strUuid string) (models.UserDto, error) {
	user, err := s.getUserFromStrUuid(strUuid)
	if err != nil {
		return models.UserDto{}, err
	}

	userDto, err := s.userToDto(user)
	if err != nil {
		log.Printf("GetUser error: %s\n", fmt.Errorf("%w", err))
		return models.UserDto{}, getAppErrorType(err)
	}

	return userDto, nil
}

// ModifyUser lets the identified user update their own first name, last
// name and username, and returns the updated user as a UserDto.
func (s *Service) ModifyUser(currentUserUuid string, userUpdate models.UserUpdate) (models.UserDto, error) {
	user, err := s.getUserFromStrUuid(currentUserUuid)
	if err != nil {
		return models.UserDto{}, err
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
		return models.UserDto{}, getAppErrorType(err)
	}

	userDto, err := s.userToDto(user)
	if err != nil {
		log.Printf("ModifyUser error: %s\n", fmt.Errorf("%w", err))
		return models.UserDto{}, getAppErrorType(err)
	}

	return userDto, nil
}

// AdminModifyUser is the admin variant of ModifyUser for any user
// identified by uuid: it also changes the user's role.
func (s *Service) AdminModifyUser(userUuid string, userUpdate models.AdminUserUpdate) (models.UserDto, error) {
	user, err := s.getUserFromStrUuid(userUuid)
	if err != nil {
		return models.UserDto{}, err
	}

	if len(userUpdate.Role) > 0 {
		role, err := s.roleRepo.GetRoleByName(userUpdate.Role)
		if err != nil {
			log.Printf("AdminModifyUser error: %s\n", fmt.Errorf("%w", err))
			return models.UserDto{}, getAppErrorType(err)
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
		return models.UserDto{}, getAppErrorType(err)
	}

	userDto, err := s.userToDto(user)
	if err != nil {
		log.Printf("AdminModifyUser error: %s\n", fmt.Errorf("%w", err))
		return models.UserDto{}, getAppErrorType(err)
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
func (s *Service) GetAllUsers() ([]models.UserDto, error) {
	users, err := s.repo.GetAllUsers()
	if err != nil {
		log.Printf("GetAllUsers error: %s\n", fmt.Errorf("%w", err))
		return nil, getAppErrorType(err)
	}

	var usersDto []models.UserDto
	for _, u := range users {
		usersDto = append(usersDto, s.userWithRoleToDto(u))
	}

	return usersDto, nil
}

// CreateUser creates a user from a UserCreate request.
func (s *Service) CreateUser(userCreateDto models.UserCreate) (models.UserDto, error) {
	if !isPasswordGoodLength(userCreateDto.Password) {
		return models.UserDto{}, handler_errors.ErrInvalidPassword
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(userCreateDto.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("CreateUser error: %s\n", fmt.Errorf("%w", err))
		return models.UserDto{}, getAppErrorType(err)
	}

	user, err := s.userCreateDtoToUser(userCreateDto, newHash)
	if err != nil {
		log.Printf("CreateUser error: %s\n", fmt.Errorf("%w", err))
		return models.UserDto{}, getAppErrorType(err)
	}

	user, err = s.repo.CreateUser(user)
	if err != nil {
		log.Printf("CreateUser error: %s\n", fmt.Errorf("%w", err))
		return models.UserDto{}, getAppErrorType(err)
	}

	userDto, err := s.userToDto(user)
	if err != nil {
		log.Printf("CreateUser error: %s\n", fmt.Errorf("%w", err))
		return models.UserDto{}, getAppErrorType(err)
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

// Login exchanges valid credentials for a signed JWT.
func (s *Service) Login(username, password string) (string, error) {
	u, err := s.repo.GetUserCredentials(username)
	if err != nil {
		return "", getAppErrorType(err)
	}

	err = bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password))
	if err != nil {
		return "", getAppErrorType(err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  u.Uuid,
		"role": u.RoleName,
		"exp":  time.Now().Add(1 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", getAppErrorType(err)
	}

	return tokenString, nil
}

// getUserFromStrUuid parses a string UUID and retrieves the user from the repository.
// It returns handler_errors.ErrInvalidRequest if the UUID is invalid, or
// handler_errors.ErrUserDoesNotExist if no user is found.
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

// userCreateDtoToUser converts a UserCreate DTO to a repository User entity.
// It looks up the role by name and generates a new UUID for the user.
func (s *Service) userCreateDtoToUser(user models.UserCreate, hashedPassword []byte) (userRepo.User, error) {
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

// userWithRoleToDto converts a UserWithRole entity to a UserDto.
func (s *Service) userWithRoleToDto(user userRepo.UserWithRole) models.UserDto {
	return models.UserDto{
		Uuid:        user.User.Uuid,
		Username:    user.User.Username,
		Role:        user.RoleName,
		FirstName:   user.User.FirstName,
		LastName:    user.User.LastName,
		LastUpdated: user.User.UpdatedAt,
		CreatedAt:   user.User.CreatedAt,
	}
}

// userToDto converts a User entity to a UserDto, looking up the role name.
func (s *Service) userToDto(user userRepo.User) (models.UserDto, error) {
	role, err := s.roleRepo.GetRoleById(user.RoleId)
	if err != nil {
		return models.UserDto{}, err
	}

	return models.UserDto{
		Uuid:        user.Uuid,
		Username:    user.Username,
		Role:        role.Name,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		LastUpdated: user.UpdatedAt,
		CreatedAt:   user.CreatedAt,
	}, nil
}

// getAppErrorType maps repository errors to handler errors.
func getAppErrorType(err error) error {
	log.Println("error happened: " + err.Error())

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

// isPasswordGoodLength validates that a password is between 12 and 72 characters.
func isPasswordGoodLength(password string) bool {
	return len(password) >= 12 && len(password) <= 72
}
