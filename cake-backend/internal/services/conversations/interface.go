package conversations

import "efournierrobert/cake-backend/internal/handlers/models"

// ConversationService is the business-logic contract for conversation
// operations consumed by HTTP handlers.
type ConversationService interface {
	// GetConversations returns conversations belonging to an owner in a space.
	GetConversations(ownerUuidStr, spaceUuidStr string) ([]models.ConversationDto, error)
	// GetConversation returns one conversation when it belongs to the owner and
	// space.
	GetConversation(ownerUuidStr, spaceUuidStr, convoUuidStr string) (models.ConversationDto, error)
	// UpdateConversation changes a conversation's title and returns the updated
	// conversation.
	UpdateConversation(ownerUuidStr, spaceUuidStr, convoUuidStr, newTitle string) (models.ConversationDto, error)
	// DeleteConversation removes a conversation belonging to the owner and
	// space.
	DeleteConversation(ownerUuidStr, spaceUuidStr, convoUuidStr string) error
	// GetFullConversation returns the messages in a conversation belonging to
	// the owner and space.
	GetFullConversation(ownerUuidStr, spaceUuidStr, convoUuidStr string) ([]models.ConversationMessagesDto, error)
}

var _ ConversationService = (*Service)(nil)
