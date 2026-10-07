// Package roles provides a repository for the user_roles table.
package roles

import (
	"database/sql"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"errors"

	"github.com/jmoiron/sqlx"
)

// Repository implements read operations on the user_roles table.
type Repository struct {
	conn *sqlx.DB
}

// New creates a new Role repository with the given database connection.
func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

// GetRoleByName returns the role with the given name, or *repo_errors.RoleNotFound.
func (r *Repository) GetRoleByName(name string) (Role, error) {
	var role Role
	err := r.conn.Get(&role, "SELECT * FROM user_roles WHERE name = ?", name)
	if errors.Is(err, sql.ErrNoRows) {
		return Role{}, &repo_errors.RoleNotFound{}
	}
	if err != nil {
		return Role{}, &repo_errors.InternalDbError{Err: err}
	}

	return role, nil
}

// GetRoleById returns the role with the given id, or *repo_errors.RoleNotFound.
func (r *Repository) GetRoleById(id int) (Role, error) {
	var role Role
	err := r.conn.Get(&role, "SELECT * FROM user_roles WHERE id = ?", id)
	if errors.Is(err, sql.ErrNoRows) {
		return Role{}, &repo_errors.RoleNotFound{}
	}
	if err != nil {
		return Role{}, &repo_errors.InternalDbError{Err: err}
	}

	return role, nil
}