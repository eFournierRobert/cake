package roles

import (
	"database/sql"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"errors"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	conn *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

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
