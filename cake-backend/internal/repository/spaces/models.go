package spaces

import "time"

type Space struct {
	ID           int       `db:"id"`
	Uuid         string    `db:"uuid"`
	Name         string    `db:"name"`
	SystemPrompt string    `db:"system_prompt"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	OwnerId      int       `db:"owner_id"`
}
