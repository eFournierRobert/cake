// Package conversations persists conversations scoped to users and spaces.
package conversations

import "uuid"

// ConvosRepo provides persistence operations for conversations. Conversation
// lookups are constrained by owner and space as well as conversation UUID.
type ConvosRepo interface {
	// GetConversation returns a conversation only when it belongs to the owner
	// and space. A missing or out-of-scope conversation returns not-found.
	GetConversation(ownerUuid, spaceUuid, convoUuid uuid.UUID) (Conversation, error)
	// CreateConversation inserts a conversation in the supplied owner's space
	// and returns the persisted row.
	CreateConversation(ownerUuid, spaceUuid uuid.UUID, conversation Conversation) (Conversation, error)
	// GetSpaceConversations returns conversations visible to the owner in the
	// supplied space, ordered by latest message; conversations without messages
	// are included last.
	GetSpaceConversations(ownerUuid, spaceUuid uuid.UUID) ([]Conversation, error)
	// UpdateConversation updates a conversation's title within the supplied scope.
	// A missing or out-of-scope conversation returns ConversationNotFound.
	UpdateConversation(ownerUuid, spaceUuid uuid.UUID, conversation Conversation) (Conversation, error)
	// DeleteConversation deletes a conversation within the supplied scope.
	// A missing or out-of-scope conversation returns ConversationNotFound.
	DeleteConversation(ownerUuid, spaceUuid uuid.UUID, conversation Conversation) error
}

var _ ConvosRepo = (*Repository)(nil)
