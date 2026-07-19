package web

import (
	"errors"
	"fmt"
	"net/http"
)

// Stable machine-readable error codes. Frontends localize system messages by
// switching on these codes; the English message is a fallback only.
const (
	CodeValidationFailed   = "validation_failed"
	CodeInvalidJSON        = "invalid_json"
	CodeUnauthorized       = "unauthorized"
	CodeInvalidCredentials = "invalid_credentials"
	CodeForbidden          = "forbidden"
	CodeNotFound           = "not_found"
	CodeConflict           = "conflict"
	CodeGone               = "gone"
	CodeRateLimited        = "rate_limited"
	CodePayloadTooLarge    = "payload_too_large"
	CodeInternal           = "internal_error"
	CodeAccountSuspended   = "account_suspended"
	CodeCooldownActive     = "cooldown_active"
	CodeUploadInvalid      = "upload_invalid"
	CodePreconditionFailed = "precondition_failed"
)

// Error is the application error type carried across module boundaries. It
// maps to a consistent JSON error body; Internal is never serialized.
type Error struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	// Internal carries the underlying cause for logs only.
	Internal error `json:"-"`
}

func (e *Error) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Internal }

// WithDetail returns a copy of the error with one field detail attached.
func (e *Error) WithDetail(field, msg string) *Error {
	c := *e
	c.Details = make(map[string]string, len(e.Details)+1)
	for k, v := range e.Details {
		c.Details[k] = v
	}
	c.Details[field] = msg
	return &c
}

func newErr(status int, code, msg string) *Error {
	return &Error{Status: status, Code: code, Message: msg}
}

// Constructors for the common cases.

func ErrValidation(msg string) *Error {
	return newErr(http.StatusUnprocessableEntity, CodeValidationFailed, msg)
}

func ErrInvalidJSON(err error) *Error {
	e := newErr(http.StatusBadRequest, CodeInvalidJSON, "request body is not valid JSON for this endpoint")
	e.Internal = err
	return e
}

func ErrUnauthorized(msg string) *Error {
	if msg == "" {
		msg = "authentication required"
	}
	return newErr(http.StatusUnauthorized, CodeUnauthorized, msg)
}

// ErrInvalidCredentials is deliberately generic to prevent user enumeration.
func ErrInvalidCredentials() *Error {
	return newErr(http.StatusUnauthorized, CodeInvalidCredentials, "invalid credentials")
}

func ErrForbidden(msg string) *Error {
	if msg == "" {
		msg = "you do not have permission to perform this action"
	}
	return newErr(http.StatusForbidden, CodeForbidden, msg)
}

func ErrNotFound(what string) *Error {
	return newErr(http.StatusNotFound, CodeNotFound, what+" not found")
}

func ErrConflict(msg string) *Error {
	return newErr(http.StatusConflict, CodeConflict, msg)
}

func ErrRateLimited() *Error {
	return newErr(http.StatusTooManyRequests, CodeRateLimited, "too many requests, slow down")
}

func ErrInternal(err error) *Error {
	e := newErr(http.StatusInternalServerError, CodeInternal, "an internal error occurred")
	e.Internal = err
	return e
}

func ErrPreconditionFailed(msg string) *Error {
	return newErr(http.StatusPreconditionFailed, CodePreconditionFailed, msg)
}

// AsError normalizes any error into an *Error, wrapping unknown errors as
// internal so no raw error text ever reaches a client.
func AsError(err error) *Error {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return ErrInternal(err)
}
