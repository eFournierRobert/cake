package handler_errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HttpStatus int
}

func (a AppError) Error() string {
	return fmt.Sprintf("error %s happened: %s", a.Code, a.Message)
}

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
	ErrUnexpectedError = AppError{
		Code:       "unexpected_error",
		Message:    "The request encountered an unexpected error",
		HttpStatus: http.StatusInternalServerError,
	}

	ErrUserDoesNotExist = AppError{
		Code:       "user_does_not_exist",
		Message:    "The requested user does not exist",
		HttpStatus: http.StatusNotFound,
	}

	ErrRoleNotFound = AppError{
		Code:       "role_not_found",
		Message:    "The requested role does not exist",
		HttpStatus: http.StatusNotFound,
	}

	ErrInvalidPassword = AppError{
		Code:       "invalid_password",
		Message:    "The provided password is invalid",
		HttpStatus: http.StatusBadRequest,
	}

	ErrInvalidRequest = AppError{
		Code:       "invalid_request",
		Message:    "The request is invalid",
		HttpStatus: http.StatusBadRequest,
	}

	ErrResourceConflict = AppError{
		Code:       "resource_conflict",
		Message:    "The provided resource conflicts with an existing one",
		HttpStatus: http.StatusConflict,
	}
)
