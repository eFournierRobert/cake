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

// GetSpaceConversations returns conversations visible to the owner in the
// supplied space, ordered by most recent message. Conversations without messages
// are included after conversations that have messages.
func (r *Repository) GetSpaceConversations(ownerUuid, spaceUuid uuid.UUID) ([]Conversation, error) {
	var conversations []Conversation
	err := r.conn.Select(&conversations, `SELECT 
			conversations.id,
			conversations.uuid,
			conversations.owner_id,
			conversations.title,
			conversations.created_at,
			conversations.space_id
		FROM conversations
		JOIN users ON conversations.owner_id = users.id
		JOIN spaces ON conversations.space_id = spaces.id
		LEFT JOIN (
		    SELECT messages.conversation_id, MAX(messages.created_at) AS latest_message_at
		    FROM messages
		    GROUP BY conversation_id
		) latest_messages ON conversations.id = latest_messages.conversation_id
		WHERE users.uuid = ? AND spaces.uuid = ?
		ORDER BY latest_messages.latest_message_at DESC,
		         conversations.created_at DESC,
		         conversations.id DESC`, ownerUuid.String(), spaceUuid.String())
	if err != nil {
		return nil, &repo_errors.InternalDbError{Err: err}
	}

	return conversations, nil
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

// UpdateConversation changes a conversation's title when it belongs to the
// supplied owner and space. Missing or out-of-scope conversations return
// ConversationNotFound.
func (r *Repository) UpdateConversation(ownerUuid, spaceUuid uuid.UUID, conversation Conversation) (Conversation, error) {
	_, err := r.conn.Exec(`UPDATE conversations 
			JOIN users ON conversations.owner_id = users.id
			JOIN spaces ON conversations.space_id = spaces.id
			SET conversations.title = ? 
			WHERE users.uuid = ? AND spaces.uuid = ? AND conversations.uuid = ?`,
		conversation.Title,
		ownerUuid.String(),
		spaceUuid.String(),
		conversation.Uuid)
	if err != nil {
		return Conversation{}, &repo_errors.InternalDbError{Err: err}
	}

	return r.GetConversation(ownerUuid, spaceUuid, uuid.MustParse(conversation.Uuid))
}

// DeleteConversation removes a conversation when it belongs to the supplied
// owner and space. Missing or out-of-scope conversations return
// ConversationNotFound.
func (r *Repository) DeleteConversation(ownerUuid, spaceUuid uuid.UUID, conversation Conversation) error {
	res, err := r.conn.Exec(`DELETE conversations FROM conversations 
			JOIN users ON conversations.owner_id = users.id
			JOIN spaces ON conversations.space_id = spaces.id
			WHERE users.uuid = ? AND spaces.uuid = ? AND conversations.uuid = ?`,
		ownerUuid.String(),
		spaceUuid.String(),
		conversation.Uuid)
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}

	deleted, err := res.RowsAffected()
	if err != nil {
		return &repo_errors.InternalDbError{Err: err}
	}
	if deleted == 0 {
		return &repo_errors.ConversationNotFound{}
	}

	return nil
}
