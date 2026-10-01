// Package users provides a repository for the users table.
package users

import (
	"database/sql"
	"errors"
	"time"
	"uuid"

	"efournierrobert/cake-backend/internal/repository/repo_errors"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// Repository implements CRUD operations on the users table.
type Repository struct {
	conn *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

// GetAllUsers returns all users, newest first.
func (r *Repository) GetAllUsers() ([]User, error) {
	var users []User
	if err := r.conn.Select(&users, "SELECT * FROM users ORDER BY created_at DESC"); err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return users, nil
}

// GetUser returns the user with the given uuid, or
// *repo_errors.UserNotFound.
func (r *Repository) GetUser(uuid uuid.UUID) (User, error) {
	var user User
	err := r.conn.Get(&user, "SELECT * FROM users WHERE uuid = ?", uuid.String())
	if errors.Is(err, sql.ErrNoRows) {
		return user, &repo_errors.UserNotFound{}
	}
	if err != nil {
		return user, &repo_errors.InternalDbError{Err: err}
	}

	return user, nil
}

// UpdateUser updates a user by uuid and returns
// *repo_errors.UserNotFound if no user matches.
func (r *Repository) UpdateUser(user User) error {
	user.UpdatedAt = time.Now()
	results, err := r.conn.NamedExec("UPDATE users SET first_name = :FirstName, last_name = :LastName, username = :Username, password_hash = :PasswordHash, role_id = :RoleId, updated_at = :UpdatedAt WHERE uuid = :Uuid", user)
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	if rowsAffected == 0 {
		return &repo_errors.UserNotFound{}
	}

	return nil
}

// CreateUser inserts a user, filling in the timestamps. It returns
// *repo_errors.UserAlreadyExists if the username is already taken.
func (r *Repository) CreateUser(user User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.conn.NamedExec("INSERT INTO users (uuid, username, password_hash, first_name, last_name, created_at, updated_at, role_id) VALUES (:Uuid, :Username, :PasswordHash, :FirstName, :LastName, :CreatedAt, :UpdatedAt, :RoleId)", user)
	if err != nil {
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) && mySQLError.Number == 1062 {
			return &repo_errors.UserAlreadyExists{}
		}
		return &repo_errors.InternalDbError{Err: err}
	}

	return nil
}

// DeleteUser removes the user with the given uuid and, by
// cascading, the conversations and spaces they own. It returns
// *repo_errors.UserNotFound if no user matches.
func (r *Repository) DeleteUser(uuid uuid.UUID) error {
	results, err := r.conn.Exec("DELETE FROM users WHERE uuid = ?", uuid.String())
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	if rowsAffected == 0 {
		return &repo_errors.UserNotFound{}
	}

	return nil
}
