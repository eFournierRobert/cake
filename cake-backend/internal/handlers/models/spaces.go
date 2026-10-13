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
// request. Non-nil fields are applied; nil fields are left unchanged. For
// pointer fields, omitted and JSON null values both decode to nil.
type SpaceUpdate struct {
	Name         *string `json:"name,omitempty"`
	SystemPrompt *string `json:"system_prompt,omitempty"`
}
