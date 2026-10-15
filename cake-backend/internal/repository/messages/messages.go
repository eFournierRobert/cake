package messages

import (
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// Repository implements message and role persistence using SQL.
type Repository struct {
	conn *sqlx.DB
}

// New creates a message repository backed by db.
func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

// GetRole returns the persisted role with the given name.
func (r *Repository) GetRole(name string) (MessageRole, error) {
	var role MessageRole
	err := r.conn.Get(&role, `SELECT * FROM message_roles WHERE name = ?`, name)
	if err != nil {
		return MessageRole{}, &repo_errors.InternalDbError{Err: err}
	}

	return role, nil
}

// GetWholeConversation returns the conversation's messages in chronological
// order, using message ID to order entries with equal timestamps.
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
		ORDER BY messages.created_at, messages.id`, conversationId)
	if err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return messages, nil
}

// CreateMessage assigns the current time to message.CreatedAt and inserts the
// message row.
func (r *Repository) CreateMessage(message Message) error {
	message.CreatedAt = time.Now()

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
