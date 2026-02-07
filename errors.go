package kommando

import "errors"

// Sentinel errors returned by the kommando package.
// Use errors.Is to check for specific error conditions.
var (
	// ErrDuplicateCommand is returned when adding a command whose name
	// already exists in the application.
	ErrDuplicateCommand = errors.New("duplicate command")

	// ErrRequiredFlag is returned when a required flag is not provided.
	ErrRequiredFlag = errors.New("required flag not provided")

	// ErrInvalidFlagValue is returned when a flag value cannot be parsed
	// as the expected type.
	ErrInvalidFlagValue = errors.New("invalid flag value")

	// ErrCommandNotFound is returned when the specified command does not
	// exist in the application.
	ErrCommandNotFound = errors.New("command not found")
)
