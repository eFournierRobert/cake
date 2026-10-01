package repo_errors

import "fmt"

type UserNotFound struct{}

func (e *UserNotFound) Error() string {
	return "user not found"
}

type UsernameAlreadyExists struct{}

func (e *UsernameAlreadyExists) Error() string {
	return "username already exists"
}

type InternalDbError struct {
	Err error
}

func (e *InternalDbError) Error() string {
	return fmt.Errorf("internal DB error: %w", e.Err).Error()
}
