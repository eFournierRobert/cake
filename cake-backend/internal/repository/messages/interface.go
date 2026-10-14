package messages

type MessagesRepo interface {
	CreateMessage(message Message) error
	GetUserRole() (MessageRole, error)
	GetWholeConversation(conversationId int) ([]MessageWithRoleName, error)
}

var _ MessagesRepo = (*Repository)(nil)
