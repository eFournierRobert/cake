package users

import (
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	conn *sqlx.DB
}

func New(db *sqlx.DB) Repository {
	return Repository{conn: db}
}

func (r *Repository) GetAllUsers() ([]User, error) {
	var users []User
	if err := r.conn.Select(&users, "SELECT * FROM users ORDER BY created_at DESC"); err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return users, nil
}

func (r *Repository) GetUser(uuid uuid.UUID) (User, error) {
	var user User
	err := r.conn.Get(&user, "SELECT * FROM users WHERE uuid = $1", uuid.String())
	return user, &repo_errors.InternalDbError{Err: err}
}

func (r *Repository) UpdateUser(user User) error {
	user.UpdatedAt = time.Now()
	results, err := r.conn.NamedExec("UPDATE users SET first_name = :FirstName, last_name = :LastName, username = :Username, password_hash = :PasswordHash, updated_at = :UpdatedAt WHERE uuid = :Uuid", user)
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

func (r *Repository) CreateUser(user User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.conn.NamedExec("INSERT INTO users (uuid, username, password_hash, first_name, last_name, created_at, updated_at, role_id) VALUES (:Uuid, :Username, :PasswordHash, :FirstName, :LastName, :CreatedAt, :UpdatedAt, :RoleId)", user)
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	return nil
}

func (r *Repository) DeleteUser(id int) error {
	_, err := r.conn.NamedExec("DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	return nil
}
