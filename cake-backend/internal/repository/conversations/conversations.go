package conversations

import (
	"database/sql"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"errors"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	conn *sqlx.DB
}

func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

func (r *Repository) GetConversation(convoUuid uuid.UUID) (Conversation, error) {
	var conversation Conversation
	err := r.conn.Get(&conversation, `SELECT 
    	* 
		FROM conversations
		WHERE conversations.uuid = ?`, convoUuid)
	if errors.Is(err, sql.ErrNoRows) {
		return Conversation{}, &repo_errors.ConversationNotFound{}
	}
	if err != nil {
		return Conversation{}, &repo_errors.InternalDbError{Err: err}
	}

	return conversation, nil
}

func (r *Repository) CreateConversation(conversation Conversation) (Conversation, error) {
	conversation.CreatedAt = time.Now()

	_, err := r.conn.NamedExec(`INSERT INTO conversations (
        				uuid,
            			owner_id,
                        title,
                        created_at,
                        space_id
		) VALUES (
		          :uuid,
		          :owner_id,
		          :title,
		          :created_at,
		          :space_id
		)`, conversation)
	if err != nil {
		return Conversation{}, &repo_errors.InternalDbError{Err: err}
	}

	return r.GetConversation(uuid.MustParse(conversation.Uuid))
}
