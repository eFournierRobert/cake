// Package spaces provides persistence operations for user-owned spaces.
package spaces

import "time"

// Space is a row from the spaces table.
type Space struct {
	ID           int       `db:"id"`
	Uuid         string    `db:"uuid"`
	Name         string    `db:"name"`
	SystemPrompt string    `db:"system_prompt"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
	OwnerId      int       `db:"owner_id"`
}
