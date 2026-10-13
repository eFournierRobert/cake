package messages

type MessagesRepo interface {
	CreateMessage(message Message) error
}

var _ MessagesRepo = (*Repository)(nil)
