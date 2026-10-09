package models

import "time"

type UserDto struct {
	Uuid        string    `json:"uuid"`
	Username    string    `json:"username"`
	Role        string    `json:"role"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	LastUpdated time.Time `json:"last_updated"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserCreate struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type UserUpdate struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type AdminUserUpdate struct {
	Username  string `json:"username,omitempty"`
	Role      string `json:"role,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type PasswordChangeRequest struct {
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
