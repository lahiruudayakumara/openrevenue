// Package domain contains the small, framework-free shared kernel used at
// bounded-context and application boundaries.
package domain

import "fmt"

// ValidationError identifies an invalid field without echoing sensitive input.
type ValidationError struct {
	Field  string
	Reason string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s %s", e.Field, e.Reason)
}

func invalid(field, reason string) error {
	return ValidationError{Field: field, Reason: reason}
}
