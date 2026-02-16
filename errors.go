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

	// ErrUnknownFlag is returned when a flag is not defined for the command.
	// Use WithAllowUnknownFlags to disable this check.
	ErrUnknownFlag = errors.New("unknown flag")

	// ErrUnsupportedShell is returned when requesting completion for
	// an unsupported shell type.
	ErrUnsupportedShell = errors.New("unsupported shell")

	// ErrInvalidName is returned when a command or flag has an empty name.
	ErrInvalidName = errors.New("invalid name")

	// ErrInvalidArgs is returned when the number of positional arguments
	// does not satisfy the command's ArgsMin/ArgsMax constraints or when
	// a custom ArgsValidator returns an error.
	ErrInvalidArgs = errors.New("invalid arguments")
)
