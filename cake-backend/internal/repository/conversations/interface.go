package conversations

import "uuid"

type ConvosRepo interface {
	GetConversation(ownerUuid, spaceUuid, convoUuid uuid.UUID) (Conversation, error)
	CreateConversation(ownerUuid, spaceUuid uuid.UUID, conversation Conversation) (Conversation, error)
}

var _ ConvosRepo = (*Repository)(nil)
