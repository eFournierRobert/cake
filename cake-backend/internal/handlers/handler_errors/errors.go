package handler_errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string
	Message    string
	HttpStatus int
}

func (a AppError) Error() string {
	return fmt.Sprintf("error %s happened: %s", a.Code, a.Message)
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
)
