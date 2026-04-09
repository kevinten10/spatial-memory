package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for domain-level error matching.
var (
	ErrNotFound           = &DomainError{Status: http.StatusNotFound, Code: 40400, Msg: "resource not found"}
	ErrUnauthorized       = &DomainError{Status: http.StatusUnauthorized, Code: 40100, Msg: "unauthorized"}
	ErrForbidden          = &DomainError{Status: http.StatusForbidden, Code: 40300, Msg: "forbidden"}
	ErrValidation         = &DomainError{Status: http.StatusBadRequest, Code: 40000, Msg: "validation error"}
	ErrConflict           = &DomainError{Status: http.StatusConflict, Code: 40900, Msg: "conflict"}
	ErrRateLimit          = &DomainError{Status: http.StatusTooManyRequests, Code: 42900, Msg: "rate limit exceeded"}
	ErrInternal           = &DomainError{Status: http.StatusInternalServerError, Code: 50000, Msg: "internal server error"}
	ErrInvalidMediaType   = &DomainError{Status: http.StatusBadRequest, Code: 40001, Msg: "invalid media type"}
	ErrInvalidContentType = &DomainError{Status: http.StatusBadRequest, Code: 40002, Msg: "invalid content type"}
	ErrFileTooLarge       = &DomainError{Status: http.StatusBadRequest, Code: 40003, Msg: "file too large"}
)

// DomainError carries an HTTP status, a machine-readable code, and a message.
type DomainError struct {
	Status int    `json:"-"`
	Code   int    `json:"code"`
	Msg    string `json:"message"`
}

func (e *DomainError) Error() string {
	return e.Msg
}

// Wrap returns a new DomainError that wraps the sentinel with additional context.
func Wrap(sentinel *DomainError, detail string) *DomainError {
	return &DomainError{
		Status: sentinel.Status,
		Code:   sentinel.Code,
		Msg:    fmt.Sprintf("%s: %s", sentinel.Msg, detail),
	}
}

// Is enables errors.Is matching against sentinel DomainErrors by code.
func (e *DomainError) Is(target error) bool {
	var t *DomainError
	if errors.As(target, &t) {
		return e.Code == t.Code
	}
	return false
}

// AsDomainError extracts a *DomainError from err if present.
func AsDomainError(err error) (*DomainError, bool) {
	var de *DomainError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}
