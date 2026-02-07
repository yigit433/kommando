package kommando

// FlagType represents the data type of a command flag.
type FlagType int

const (
	// FlagString is the default flag type for string values.
	FlagString FlagType = iota
	// FlagBool represents a boolean flag.
	FlagBool
	// FlagInt represents an integer flag.
	FlagInt
	// FlagFloat represents a floating-point flag.
	FlagFloat
)

// String returns the string representation of a FlagType.
func (ft FlagType) String() string {
	switch ft {
	case FlagBool:
		return "bool"
	case FlagInt:
		return "int"
	case FlagFloat:
		return "float"
	default:
		return "string"
	}
}

// Flag defines a command-line flag for a Command.
type Flag struct {
	// Name is the flag identifier used on the command line (e.g. --name).
	Name string
	// Short is an optional single-character shorthand (e.g. 'v' for -v).
	// Use 0 to indicate no short flag.
	Short rune
	// Description is a short explanation of the flag's purpose.
	Description string
	// Type determines how the flag value is parsed. Defaults to FlagString.
	Type FlagType
	// Required causes an error if the flag is not provided.
	Required bool
	// Default is the default value used when the flag is not provided.
	Default string
}

// Command represents a CLI command with its metadata and execution logic.
type Command struct {
	// Name is the primary identifier for the command.
	Name string
	// Description is a short explanation of the command's purpose.
	Description string
	// Flags defines the flags accepted by this command.
	Flags []Flag
	// Aliases are alternative names for the command.
	Aliases []string
	// Execute is the function called when the command is invoked.
	// It receives a Context containing parsed flags and arguments.
	Execute func(ctx *Context) error
}

// hasAlias reports whether the command has the given alias.
func (c *Command) hasAlias(name string) bool {
	for _, alias := range c.Aliases {
		if alias == name {
			return true
		}
	}
	return false
}
