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

// ChatCreate is the request body for sending one user message to a space.
// Omitting ConversationUuid starts a new conversation; Reasoning is passed to
// the selected provider as its reasoning effort when non-empty.
type ChatCreate struct {
	ConversationUuid *string `json:"conversation_uuid,omitempty"`
	ModelUuid        string  `json:"model_uuid"`
	Message          string  `json:"message"`
	Reasoning        string  `json:"reasoning"`
}

// ChatDto is an internal event passed from the chat service to the SSE handler.
// When Error is non-empty, the handler uses it as the SSE event name instead
// of Event; only Data is serialized as the event payload.
type ChatDto struct {
	Event string      `json:"event"`
	Data  ChatDtoData `json:"data"`
	Error string      `json:"error,omitempty"`
}

// ChatDtoData is the JSON payload sent for each chat SSE event. Fields that do
// not apply to an event are encoded as null.
type ChatDtoData struct {
	MessageUuid      *string `json:"message_uuid"`
	Content          *string `json:"content"`
	ConversationUuid *string `json:"conversation_uuid"`
}
