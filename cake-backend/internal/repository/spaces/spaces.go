package spaces

import (
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	conn *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{
		conn: db,
	}
}

func (r *Repository) GetAllUserSpaces(ownerUuid uuid.UUID) ([]Space, error) {
	var spaces []Space
	err := r.conn.Select(&spaces, `
		SELECT 
		    * 
		FROM spaces
		LEFT JOIN users ON spaces.owner_id = users.id
		WHERE users.uuid = ?
	`, ownerUuid.String())
	if err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return spaces, nil
}

func (r *Repository) GetSpace(spaceUuid uuid.UUID) (Space, error) {
	var space Space
	err := r.conn.Get(&space, `
		SELECT 
		    * 
		FROM spaces
		WHERE spaces.uuid = ?
	`, spaceUuid.String())
	if err != nil {
		return Space{}, &repo_errors.InternalDbError{Err: err}
	}

	return space, nil
}

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

func (r *Repository) UpdateSpace(space Space) (Space, error) {
	space.UpdatedAt = time.Now()

	_, err := r.conn.NamedExec(`UPDATE spaces 
			SET name = :name,
			    system_prompt = :system_prompt
			WHERE uuid = :uuid`, space)
	if err != nil {
		return Space{}, &repo_errors.InternalDbError{Err: err}
	}

	return r.GetSpace(uuid.MustParse(space.Uuid))
}

func (r *Repository) DeleteSpace(spaceUuid uuid.UUID) error {
	_, err := r.conn.NamedExec(`DELETE 
       FROM spaces
       WHERE spaces.uuid = ?`, spaceUuid.String())
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	return nil
}
