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
	ErrPasswordChangeNeed = New(2009, http.StatusForbidden, "password change required")
	ErrSessionExpired     = New(2010, http.StatusUnauthorized, "session expired")
	ErrUserDisabled       = New(2011, http.StatusForbidden, "user is disabled")
	ErrLoginRateLimited   = New(2012, http.StatusTooManyRequests, "too many login attempts, try later")
	ErrHostNotFound       = New(2013, http.StatusNotFound, "host not found")
	ErrHostDisabled       = New(2014, http.StatusForbidden, "host is disabled")
	ErrAgentUnavailable   = New(3001, http.StatusServiceUnavailable, "core agent unavailable")
	ErrHostUnavailable    = New(3010, http.StatusServiceUnavailable, "host agent unavailable")
	ErrSettingNotFound    = New(2015, http.StatusNotFound, "setting not found")
	ErrSettingKeyExists   = New(2016, http.StatusBadRequest, "setting key already exists")
	ErrSettingKeyInvalid  = New(2017, http.StatusBadRequest, "invalid setting key")
)
