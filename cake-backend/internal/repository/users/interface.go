package users

import "uuid"

type UserRepository interface {
	GetUser(userUuid uuid.UUID) (User, error)
	GetAllUsers() ([]UserWithRole, error)
	UpdateUser(user User) (User, error)
	CreateUser(user User) (User, error)
	DeleteUser(uuid uuid.UUID) error
	GetUserCredentials(username string) (UserWithRole, error)
}

var _ UserRepository = (*Repository)(nil)
