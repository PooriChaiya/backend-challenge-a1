// Package domain holds the core entities and domain errors. No project-internal
// imports allowed here — this is the center of the hexagon.
package domain

import (
	"errors"
	"time"
)

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// Domain errors. Adapters map these to transport-specific responses (HTTP codes).
var (
	ErrNotFound           = errors.New("user not found")
	ErrDuplicateEmail     = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidInput       = errors.New("invalid input")
)
