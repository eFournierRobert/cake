package messages

import "time"

type Message struct {
	ID             int       `db:"id"`
	Uuid           string    `db:"uuid"`
	Content        string    `db:"content"`
	CreatedAt      time.Time `db:"created_at"`
	ConversationId int       `db:"conversation_id"`
	RoleId         int       `db:"role_id"`
}
