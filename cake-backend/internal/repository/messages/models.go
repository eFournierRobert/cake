package messages

import "time"

// Message is a persisted conversation message.
type Message struct {
	ID             int       `db:"id"`
	Uuid           string    `db:"uuid"`
	Content        string    `db:"content"`
	CreatedAt      time.Time `db:"created_at"`
	ConversationId int       `db:"conversation_id"`
	RoleId         int       `db:"role_id"`
}

// MessageRole identifies a message's role, such as user or assistant.
type MessageRole struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

// MessageWithRoleName combines a persisted message with its role name.
type MessageWithRoleName struct {
	Message
	RoleName string `db:"role_name"`
}
