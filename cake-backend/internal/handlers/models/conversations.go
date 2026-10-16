package models

import "time"

type ConversationDto struct {
	Uuid      string    `json:"uuid"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type ConversationUpdate struct {
	Title string `json:"title"`
}

type ConversationMessagesDto struct {
	Uuid      string    `json:"uuid"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	RoleName  string    `json:"role_name"`
}
