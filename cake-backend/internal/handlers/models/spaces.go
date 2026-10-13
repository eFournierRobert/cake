package models

import "time"

// SpaceDto is the client-facing representation of a space.
type SpaceDto struct {
	Uuid         string    `json:"uuid"`
	Name         string    `json:"name"`
	SystemPrompt string    `json:"system_prompt"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SpaceCreate is the request body used to create a space.
type SpaceCreate struct {
	Name         string `json:"name"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

// SpaceUpdate contains the optional space fields accepted by an update
// request. SystemPrompt tracks presence separately so null can clear the
// stored prompt while an omitted property leaves it unchanged.
type SpaceUpdate struct {
	Name         *string `json:"name,omitempty"`
	SystemPrompt *string `json:"system_prompt,omitempty"`
}
