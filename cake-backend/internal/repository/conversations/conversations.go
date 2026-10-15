package conversations

import (
	"database/sql"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"errors"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
)

// Repository implements conversation persistence using SQL.
type Repository struct {
	conn *sqlx.DB
}

// New creates a conversation repository backed by db.
func New(db *sqlx.DB) *Repository {
	return &Repository{conn: db}
}

// GetConversation retrieves a conversation scoped to the supplied owner and
// space. Missing or out-of-scope rows return ConversationNotFound.
func (r *Repository) GetConversation(ownerUuid, spaceUuid, convoUuid uuid.UUID) (Conversation, error) {
	var conversation Conversation
	err := r.conn.Get(&conversation, `SELECT 
			conversations.id,
			conversations.uuid,
			conversations.owner_id,
			conversations.title,
			conversations.created_at,
			conversations.space_id
		FROM conversations
		JOIN users ON conversations.owner_id = users.id
		JOIN spaces ON conversations.space_id = spaces.id
		WHERE conversations.uuid = ? AND users.uuid = ? AND spaces.uuid = ?`,
		convoUuid.String(), ownerUuid.String(), spaceUuid.String())
	if errors.Is(err, sql.ErrNoRows) {
		return Conversation{}, &repo_errors.ConversationNotFound{}
	}
	if err != nil {
		return Conversation{}, &repo_errors.InternalDbError{Err: err}
	}

	return conversation, nil
}

// CreateConversation inserts a conversation, stamps its creation time, and
// returns the stored row after verifying it is visible in the supplied scope.
func (r *Repository) CreateConversation(ownerUuid, spaceUuid uuid.UUID, conversation Conversation) (Conversation, error) {
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

	return r.GetConversation(ownerUuid, spaceUuid, uuid.MustParse(conversation.Uuid))
}
