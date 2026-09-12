// Package service implements transactional use cases over the domain rules and sqlc queries.
package service

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
}

func (e *Error) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

func NotFound(what string) *Error {
	return &Error{Status: http.StatusNotFound, Code: "not_found", Message: what + " not found"}
}
func Conflict(code, msg string) *Error {
	return &Error{Status: http.StatusConflict, Code: code, Message: msg}
}
func Invalid(fields map[string]string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: "invalid", Message: "validation failed", Fields: fields}
}
func Unauthorized(msg string) *Error {
	return &Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: msg}
}

func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// notFoundIf converts sql.ErrNoRows into a 404 for the named thing.
func notFoundIf(err error, what string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return NotFound(what)
	}
	return err
}
