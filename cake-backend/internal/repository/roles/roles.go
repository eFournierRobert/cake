package roles

import (
	"database/sql"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"efournierrobert/cake-backend/internal/repository/users"
	"errors"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	conn *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

func (r *Repository) GetUserRole(user users.User) (Role, error) {
	var role Role
	err := r.conn.Get(&role, "SELECT * FROM user_roles WHERE id = ?", user.Id)
	if errors.Is(err, sql.ErrNoRows) {
		return Role{}, &repo_errors.RoleNotFound{}
	}
	if err != nil {
		return Role{}, &repo_errors.InternalDbError{Err: err}
	}

	return role, nil
}
