package users

import (
	"time"
)

// User is a row of the users table.
type User struct {
	Id           int       `db:"id"`
	Uuid         string    `db:"uuid"`
	Username     string    `db:"username"`
	PasswordHash []byte    `db:"password_hash"`
	FirstName    string    `db:"first_name"`
	LastName     string    `db:"last_name"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	RoleId       int       `db:"role_id"`
}

type UserWithRole struct {
	User
	RoleName string `db:"role_name"`
}
