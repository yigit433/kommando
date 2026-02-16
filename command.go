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
	// FlagStringSlice represents a repeatable string flag that collects multiple values.
	// Values can be provided via repetition (--tag a --tag b) or commas (--tag a,b,c).
	FlagStringSlice
	// FlagCount represents a counter flag incremented by repetition (e.g. -vvv → 3).
	FlagCount
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
	case FlagStringSlice:
		return "[]string"
	case FlagCount:
		return "count"
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
	// Env is an optional environment variable name. When set and the flag
	// is not provided on the command line, the value is read from this
	// environment variable (checked after Default).
	Env string
}

// Command represents a CLI command with its metadata and execution logic.
type Command struct {
	// Name is the primary identifier for the command.
	Name string
	// Description is a short explanation of the command's purpose.
	Description string
	// Usage is a custom usage line shown in help (e.g. "greet [flags] <name>").
	Usage string
	// Example is an optional example block shown at the end of help output.
	// Can be multiline.
	Example string
	// Flags defines the flags accepted by this command.
	Flags []Flag
	// Aliases are alternative names for the command.
	Aliases []string
	// SubCommands defines nested commands (e.g. "server start", "server stop").
	// When SubCommands is set and the first positional argument matches a
	// subcommand, that subcommand is executed instead of Execute.
	SubCommands []*Command
	// ArgsMin is the minimum number of positional arguments required.
	// Zero means no minimum constraint. Ignored when ArgsValidator is set.
	ArgsMin int
	// ArgsMax is the maximum number of positional arguments allowed.
	// Zero means no maximum constraint. Ignored when ArgsValidator is set.
	ArgsMax int
	// ArgsValidator is a custom validation function for positional arguments.
	// When set, ArgsMin and ArgsMax are ignored.
	ArgsValidator func([]string) error
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

// findSubCommand looks up a subcommand by name or alias.
func (c *Command) findSubCommand(name string) *Command {
	for _, sub := range c.SubCommands {
		if sub.Name == name || sub.hasAlias(name) {
			return sub
		}
	}
	return nil
}
