package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Code       int
	Message    string
	HTTPStatus int
	Err        error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func New(code int, status int, message string) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: status,
	}
}

func Wrap(code int, status int, message string, err error) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: status,
		Err:        err,
	}
}

func As(err error) (*Error, bool) {
	var appErr *Error
	ok := errors.As(err, &appErr)
	return appErr, ok
}

var (
	ErrBadRequest         = New(2000, http.StatusBadRequest, "bad request")
	ErrInvalidCredential  = New(2001, http.StatusUnauthorized, "invalid username or password")
	ErrUnauthorized       = New(2002, http.StatusUnauthorized, "unauthorized")
	ErrUserNotFound       = New(2003, http.StatusNotFound, "user not found")
	ErrInternal           = New(1000, http.StatusInternalServerError, "internal server error")
	ErrTokenGenerate      = New(2004, http.StatusInternalServerError, "failed to generate token")
	ErrTokenParse         = New(2005, http.StatusUnauthorized, "invalid token")
	ErrBootstrapAdminFail = New(2006, http.StatusInternalServerError, "failed to bootstrap default admin")
	ErrPermissionDenied   = New(2007, http.StatusForbidden, "permission denied")
	ErrTaskNotFound       = New(2008, http.StatusNotFound, "task not found")
	ErrAgentUnavailable   = New(3001, http.StatusServiceUnavailable, "core agent unavailable")
)
