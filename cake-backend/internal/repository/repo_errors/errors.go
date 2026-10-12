// Package repo_errors defines the error types returned by the
// repository packages.
package repo_errors

import "fmt"

// UserNotFound is returned when no user matches the targeted uuid.
type UserNotFound struct{}

func (e *UserNotFound) Error() string {
	return "user not found"
}

// UserAlreadyExists is returned when the username is already taken.
type UserAlreadyExists struct{}

func (e *UserAlreadyExists) Error() string {
	return "user already exists"
}

// InternalDbError wraps an unexpected database failure.
type InternalDbError struct {
	Err error
}

func (e *InternalDbError) Error() string {
	return fmt.Errorf("internal DB error: %w", e.Err).Error()
}

// RoleNotFound is returned when no role matches the targeted identifier.
type RoleNotFound struct{}

func (e *RoleNotFound) Error() string {
	return "role not found"
}

// ProviderNotFound is returned when no provider matches the targeted uuid.
type ProviderNotFound struct{}

func (e *ProviderNotFound) Error() string {
	return "provider not found"
}

// ModelNotFound is returned when no model matches the targeted uuid.
type ModelNotFound struct{}

func (e *ModelNotFound) Error() string {
	return "model not found"
}
