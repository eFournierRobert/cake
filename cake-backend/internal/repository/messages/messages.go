package messages

import (
	"efournierrobert/cake-backend/internal/repository/repo_errors"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	conn *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

func (r *Repository) CreateMessage(message Message) error {
	_, err := r.conn.NamedExec(`INSERT INTO messages (
                      uuid,
                      content, 
                      created_at,
                      conversation_id, 
                      role_id) 
					VALUES (
	                    	:uuid,
					        :content,
					        :created_at,
					        :conversation_id,
					        :role_id
                      )`, message)
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	return nil
}
