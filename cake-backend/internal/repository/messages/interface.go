package messages

type MessagesRepo interface {
	CreateMessage(message Message) error
	GetRole(name string) (MessageRole, error)
	GetWholeConversation(conversationId int) ([]MessageWithRoleName, error)
}

var _ MessagesRepo = (*Repository)(nil)
