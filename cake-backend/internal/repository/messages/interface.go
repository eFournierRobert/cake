// Package messages persists conversation messages and their roles.
package messages

// MessagesRepo provides persistence operations for conversation messages and
// their role names.
type MessagesRepo interface {
	// CreateMessage inserts a message and assigns its creation timestamp.
	CreateMessage(message Message) error
	// GetRole returns the persisted role identified by name.
	GetRole(name string) (MessageRole, error)
	// GetWholeConversation returns all messages in chronological order with
	// their role names joined in.
	GetWholeConversation(conversationId int) ([]MessageWithRoleName, error)
}

var _ MessagesRepo = (*Repository)(nil)
