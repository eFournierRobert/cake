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

// New creates a new User repository with the given database connection.
func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

// GetAllUsers returns all users, newest first.
func (r *Repository) GetAllUsers() ([]UserWithRole, error) {
	var users []UserWithRole
	if err := r.conn.Select(&users,
		`SELECT 
			 	users.id,
				users.uuid,
				users.username,
				users.password_hash,
				users.first_name,
				users.last_name,
				users.role_id,
				users.created_at,
				users.updated_at,
				user_roles.name AS role_name 
			FROM users 
			JOIN user_roles ON users.role_id = user_roles.id 
			ORDER BY users.created_at DESC`); err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return users, nil
}

// GetUser returns the user with the given uuid, or *repo_errors.UserNotFound.
func (r *Repository) GetUser(uuid uuid.UUID) (User, error) {
	var user User
	err := r.conn.Get(&user,
		`SELECT * FROM users WHERE uuid = ?`,
		uuid.String())
	if errors.Is(err, sql.ErrNoRows) {
		return user, &repo_errors.UserNotFound{}
	}
	if err != nil {
		return user, &repo_errors.InternalDbError{Err: err}
	}

	return user, nil
}

// UpdateUser updates a user by uuid and returns the updated user as
// stored in the database, or *repo_errors.UserNotFound if no user matches.
func (r *Repository) UpdateUser(user User) (User, error) {
	user.UpdatedAt = time.Now()
	_, err := r.conn.NamedExec(
		`UPDATE users 
			SET first_name = :first_name,
			    last_name = :last_name,
			    username = :username,
			    password_hash = :password_hash,
			    role_id = :role_id,
			    updated_at = :updated_at 
			WHERE uuid = :uuid`,
		user)
	if err != nil {
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) && mySQLError.Number == 1062 {
			return User{}, &repo_errors.UserAlreadyExists{}
		}
		return User{}, &repo_errors.InternalDbError{Err: err}
	}

	return r.GetUser(uuid.MustParse(user.Uuid))
}

// CreateUser inserts a user, filling in the timestamps, and returns
// the created user as stored in the database, including its
// auto-generated id. It returns *repo_errors.UserAlreadyExists if
// the username is already taken.
func (r *Repository) CreateUser(user User) (User, error) {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.conn.NamedExec(
		`INSERT INTO users (
                   uuid,
                   username,
                   password_hash,
                   first_name,
                   last_name,
                   created_at,
                   updated_at,
                   role_id) 
		VALUES (
		        :uuid,
		        :username,
		        :password_hash,
		        :first_name,
		        :last_name,
		        :created_at,
		        :updated_at,
		        :role_id)`,
		user)
	if err != nil {
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) && mySQLError.Number == 1062 {
			return User{}, &repo_errors.UserAlreadyExists{}
		}
		return User{}, &repo_errors.InternalDbError{Err: err}
	}

	return r.GetUser(uuid.MustParse(user.Uuid))
}

// DeleteUser removes the user with the given uuid and, by
// cascading, the conversations and spaces they own. It returns
// *repo_errors.UserNotFound if no user matches.
func (r *Repository) DeleteUser(uuid uuid.UUID) error {
	results, err := r.conn.Exec(
		`DELETE FROM users WHERE uuid = ?`,
		uuid.String())
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

// GetUserCredentials returns a user's credentials (uuid, username, password hash, role)
// for authentication purposes. It is used by the Login service.
func (r *Repository) GetUserCredentials(username string) (UserWithRole, error) {
	var user UserWithRole
	err := r.conn.Get(&user, `SELECT 
		 	users.uuid,
		 	users.username,
		 	users.password_hash,
		 	user_roles.name AS role_name
		 	FROM users
		 	JOIN user_roles ON users.role_id = user_roles.id
		 	WHERE users.username = ?`, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return UserWithRole{}, &repo_errors.UserNotFound{}
		}
		return UserWithRole{}, &repo_errors.InternalDbError{Err: err}
	}

	return user, nil
}