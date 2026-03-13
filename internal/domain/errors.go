// Package domain defines sentinel errors used across the service layer.
//
// Wrapping convention: infrastructure errors (os, encoding, crypto/rand, etc.) must be
// wrapped with ErrIO at the point they first cross a boundary into service code, using
// fmt.Errorf("%w: %w", ErrIO, err). Raw stdlib errors must never propagate to callers.
package domain

import "errors"

var (
	ErrNotFound             = errors.New("not found")
	ErrConflict             = errors.New("conflict")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrForbidden            = errors.New("forbidden")
	ErrRegistrationDisabled = errors.New("registration disabled")
	ErrIO                   = errors.New("io error")
	ErrSessionExpired       = errors.New("session expired")
)
