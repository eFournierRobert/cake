package conversations

import "uuid"

type ConvosRepo interface {
	GetConversation(ownerUuid, convoUuid uuid.UUID) (Conversation, error)
	CreateConversation(ownerUuid uuid.UUID, conversation Conversation) (Conversation, error)
}

var _ ConvosRepo = (*Repository)(nil)
