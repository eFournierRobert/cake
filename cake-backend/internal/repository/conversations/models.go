package conversations

import "time"

// Conversation is a persisted conversation associated with an owner and space.
type Conversation struct {
	ID        int       `db:"id"`
	Uuid      string    `db:"uuid"`
	OwnerId   int       `db:"owner_id"`
	Title     string    `db:"title"`
	CreatedAt time.Time `db:"created_at"`
	SpaceId   int       `db:"space_id"`
}
