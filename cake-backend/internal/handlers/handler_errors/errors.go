// Package handler_errors defines error types and utilities for HTTP
// handlers. Errors are designed to be serialized as JSON responses
// matching the OpenAPI specification.
package handler_errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// AppError is a structured error type for API responses.
// It contains a machine-readable code, human-readable message, and HTTP status.
type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HttpStatus int    `json:"-"`
}

// Error implements the error interface for AppError.
func (a AppError) Error() string {
	return fmt.Sprintf("error %s happened: %s", a.Code, a.Message)
}

// WriteError writes an error response to the HTTP response writer.
// It accepts any error type and converts KnownAppError types to their
// pre-defined response codes. Unknown error types are treated as
// ErrUnexpectedError.
func WriteError(w http.ResponseWriter, err error) {
	var appErr AppError
	ok := errors.As(err, &appErr)
	if !ok {
		appErr = ErrUnexpectedError
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.HttpStatus)
	_ = json.NewEncoder(w).Encode(appErr)
}

var (
	// ErrUnexpectedError is returned for unexpected server errors.
	ErrUnexpectedError = AppError{
		Code:       "unexpected_error",
		Message:    "The request encountered an unexpected error",
		HttpStatus: http.StatusInternalServerError,
	}

	// ErrUserDoesNotExist is returned when a requested user cannot be found.
	ErrUserDoesNotExist = AppError{
		Code:       "user_does_not_exist",
		Message:    "The requested user does not exist",
		HttpStatus: http.StatusNotFound,
	}

	// ErrRoleNotFound is returned when a requested role cannot be found.
	ErrRoleNotFound = AppError{
		Code:       "role_not_found",
		Message:    "The requested role does not exist",
		HttpStatus: http.StatusNotFound,
	}

	ErrProviderNotFound = AppError{
		Code:       "provider_not_found",
		Message:    "The requested provider does not exist",
		HttpStatus: http.StatusNotFound,
	}

	ErrModelNotFound = AppError{
		Code:       "model_not_found",
		Message:    "The requested model does not exist",
		HttpStatus: http.StatusNotFound,
	}

	// ErrInvalidPassword is returned when authentication credentials are invalid.
	ErrInvalidPassword = AppError{
		Code:       "invalid_credentials",
		Message:    "Invalid username or password",
		HttpStatus: http.StatusUnauthorized,
	}

	// ErrInvalidRequest is returned when the request is malformed or invalid.
	ErrInvalidRequest = AppError{
		Code:       "invalid_request",
		Message:    "The request is invalid",
		HttpStatus: http.StatusBadRequest,
	}

	// ErrResourceConflict is returned when a resource conflict occurs
	// (e.g., duplicate username).
	ErrResourceConflict = AppError{
		Code:       "resource_conflict",
		Message:    "The provided resource conflicts with an existing one",
		HttpStatus: http.StatusConflict,
	}
)
