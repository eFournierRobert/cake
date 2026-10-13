package conversations

import "uuid"

type ConvosRepo interface {
	GetConversation(convoUuid uuid.UUID) (Conversation, error)
	CreateConversation(conversation Conversation) (Conversation, error)
}

var _ ConvosRepo = (*Repository)(nil)
