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

func (r *Repository) GetUserRole() (MessageRole, error) {
	var role MessageRole
	err := r.conn.Get(&role, `SELECT * FROM message_roles WHERE name = 'user'`)
	if err != nil {
		return MessageRole{}, &repo_errors.InternalDbError{Err: err}
	}

	return role, nil
}

func (r *Repository) GetWholeConversation(conversationId int) ([]MessageWithRoleName, error) {
	var messages []MessageWithRoleName
	err := r.conn.Select(&messages, `SELECT
			messages.id,
			messages.uuid,
			messages.content,
			messages.created_at,
			messages.conversation_id,
			messages.role_id,
			message_roles.name AS 'role_name'
		FROM messages
		JOIN message_roles ON messages.role_id = message_roles.id
		WHERE messages.conversation_id = ?
		ORDER BY created_at DESC`, conversationId)
	if err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return messages, nil
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
