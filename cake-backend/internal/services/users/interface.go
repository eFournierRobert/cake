package users

import (
	"efournierrobert/cake-backend/internal/handlers/models"
)

// UserService is the contract behind the API Users operations.
// The HTTP handlers depend on this interface rather than the
// concrete Service, so tests can substitute a mock (see mock.go)
// for the real service.
type UserService interface {
	// GetUser returns the user with the given uuid as a UserDto.
	// It returns handler_errors.ErrUserDoesNotExist when no user
	// matches.
	GetUser(strUuid string) (models.UserDto, error)

	// ModifyUser lets the identified user update their own first
	// name, last name and username.
	ModifyUser(currentUserUuid string, userUpdate models.UserUpdate) (models.UserDto, error)

	// AdminModifyUser is the admin variant of ModifyUser for any
	// user identified by uuid: it also changes the user's role.
	AdminModifyUser(userUuid string, userUpdate models.AdminUserUpdate) (models.UserDto, error)

	// ChangePassword updates the password of the identified user.
	// The new password must be 12 to 72 characters long.
	ChangePassword(currentUserUuid string, newPassword string) error

	// GetAllUsers returns every user as a UserDto, newest first.
	GetAllUsers() ([]models.UserDto, error)

	// CreateUser creates a user from a UserCreate request.
	CreateUser(userCreateDto models.UserCreate) (models.UserDto, error)

	// DeleteUser removes the user with the given uuid.
	DeleteUser(userUuid string) error

	// Login exchanges valid credentials for a signed JWT.
	Login(username, password string) (string, error)
}

// Compile-time proof that Service keeps fulfilling the interface.
var _ UserService = (*Service)(nil)
