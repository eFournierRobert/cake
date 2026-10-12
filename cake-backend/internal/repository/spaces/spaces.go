package spaces

import (
	"database/sql"
	"errors"
	"time"
	"uuid"

	"efournierrobert/cake-backend/internal/repository/repo_errors"

	"github.com/jmoiron/sqlx"
)

// Repository implements CRUD operations on the spaces table.
type Repository struct {
	conn *sqlx.DB
}

// New creates a spaces repository using db for database operations.
func New(db *sqlx.DB) *Repository {
	return &Repository{
		conn: db,
	}
}

// GetAllUserSpaces returns the spaces owned by the user identified by ownerUuid.
func (r *Repository) GetAllUserSpaces(ownerUuid uuid.UUID) ([]Space, error) {
	var spaces []Space
	err := r.conn.Select(&spaces, `
		SELECT
		    spaces.id,
		    spaces.uuid,
		    spaces.name,
		    spaces.system_prompt,
		    spaces.created_at,
		    spaces.updated_at,
		    spaces.owner_id
		FROM spaces
		LEFT JOIN users ON spaces.owner_id = users.id
		WHERE users.uuid = ?
	`, ownerUuid.String())
	if err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return spaces, nil
}

// GetSpace returns the space identified by spaceUuid, or SpaceNotFound if absent.
func (r *Repository) GetSpace(spaceUuid uuid.UUID) (Space, error) {
	var space Space
	err := r.conn.Get(&space, `
		SELECT 
		    * 
		FROM spaces
		WHERE spaces.uuid = ?
	`, spaceUuid.String())
	if errors.Is(err, sql.ErrNoRows) {
		return Space{}, &repo_errors.SpaceNotFound{}
	}
	if err != nil {
		return Space{}, &repo_errors.InternalDbError{Err: err}
	}

	return space, nil
}

// CreateSpace inserts a space, sets its timestamps, and returns the stored row.
func (r *Repository) CreateSpace(space Space) (Space, error) {
	space.CreatedAt = time.Now()
	space.UpdatedAt = time.Now()

	_, err := r.conn.NamedExec(`INSERT INTO spaces (
                    uuid, 
                    name, 
                    system_prompt, 
                    created_at, 
                    updated_at, 
                    owner_id) 
                    VALUES (
                            :uuid,
                            :name,
                            :system_prompt,
                            :created_at,
                            :updated_at,
                            :owner_id
                    )`, space)
	if err != nil {
		return Space{}, &repo_errors.InternalDbError{Err: err}
	}

	return r.GetSpace(uuid.MustParse(space.Uuid))
}

// UpdateSpace updates a space's name and system prompt and returns the stored row.
func (r *Repository) UpdateSpace(space Space) (Space, error) {
	space.UpdatedAt = time.Now()

	_, err := r.conn.NamedExec(`UPDATE spaces 
			SET name = :name,
			    system_prompt = :system_prompt,
			    updated_at = :updated_at
			WHERE uuid = :uuid`, space)
	if err != nil {
		return Space{}, &repo_errors.InternalDbError{Err: err}
	}

	return r.GetSpace(uuid.MustParse(space.Uuid))
}

// DeleteSpace removes the space identified by spaceUuid, or returns SpaceNotFound.
func (r *Repository) DeleteSpace(spaceUuid uuid.UUID) error {
	result, err := r.conn.Exec(`DELETE
       FROM spaces
       WHERE spaces.uuid = ?`, spaceUuid.String())
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}
	if deleted == 0 {
		return &repo_errors.SpaceNotFound{}
	}

	return nil
}
