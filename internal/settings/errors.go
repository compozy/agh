package settings

import (
	"errors"
	"fmt"
)

// ErrValidation marks malformed settings requests or payloads.
var ErrValidation = errors.New("settings: validation")

// ErrNotFound marks missing settings sections, collections, or items.
var ErrNotFound = errors.New("settings: not found")

// ErrConflict marks conflicting settings scope or target combinations.
var ErrConflict = errors.New("settings: conflict")

// ErrForbidden marks settings operations rejected by policy.
var ErrForbidden = errors.New("settings: forbidden")

// ErrUnprocessable marks settings requests whose selected resource exists but
// cannot be processed because its backing state is invalid.
var ErrUnprocessable = errors.New("settings: unprocessable")

func validationError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrValidation, err)
}

func notFoundError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrNotFound, err)
}

func conflictError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrConflict, err)
}

func unprocessableError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrUnprocessable, err)
}
