package conversations

import "time"

type Conversation struct {
	ID        int       `db:"id"`
	Uuid      string    `db:"uuid"`
	OwnerId   int       `db:"owner_id"`
	Title     string    `db:"title"`
	CreatedAt time.Time `db:"created_at"`
	SpaceId   int       `db:"space_id"`
}
