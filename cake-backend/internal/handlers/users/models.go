package users

import (
	"time"
)

// UserDto is the user representation returned to API clients,
// matching the User schema in doc/openapi.yaml. It is used as the
// response body for /user, PATCH /user, /users, POST /users and
// /users/{uuid}. All fields are required by the spec, so they are
// plain values.
type UserDto struct {
	Uuid        string    `json:"uuid"`
	Username    string    `json:"username"`
	Role        string    `json:"role"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	LastUpdated time.Time `json:"last_updated"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserCreate is the request body for POST /users, matching the
// UserCreate schema in doc/openapi.yaml. username, password and
// role are required; first_name and last_name may be omitted.
type UserCreate struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	Role      string `json:"role"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// UserUpdate is the request body for PATCH /user, matching the
// UserUpdate schema in doc/openapi.yaml. All fields are optional;
// the spec requires at least one to be provided, which the handler
// must verify before treating the request as valid.
type UserUpdate struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// AdminUserUpdate is the request body for PATCH /users/{uuid},
// matching the AdminUserUpdate schema in doc/openapi.yaml. All
// fields are optional; the spec requires at least one to be
// provided, which the handler must verify before treating the
// request as valid.
type AdminUserUpdate struct {
	Username  string `json:"username,omitempty"`
	Role      string `json:"role,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// PasswordChangeRequest is the request body for POST /user/password
// and POST /users/{uuid}/password, matching the
// PasswordChangeRequest schema in doc/openapi.yaml.
type PasswordChangeRequest struct {
	Password string `json:"password"`
}
