package kommando

import (
	"fmt"
	"strconv"
	"strings"
)

// parseArgs parses raw command-line arguments into positional args and flag values.
// It supports --flag=value, --flag value, -flag=value, -flag value syntax,
// and the -- bare separator to stop flag parsing.
func parseArgs(cmd *Command, raw []string) ([]string, map[string]string, error) {
	var positional []string
	flags := make(map[string]string)
	stopFlags := false

	i := 0
	for i < len(raw) {
		arg := raw[i]

		// After --, everything is positional.
		if arg == "--" {
			stopFlags = true
			i++
			continue
		}

		if stopFlags || !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			i++
			continue
		}

		name, value, consumed, err := parseFlag(cmd, raw, i)
		if err != nil {
			return nil, nil, err
		}
		flags[name] = value
		i += consumed
	}

	// Apply defaults for flags not provided.
	for _, f := range cmd.Flags {
		if _, ok := flags[f.Name]; !ok && f.Default != "" {
			flags[f.Name] = f.Default
		}
	}

	// Check required flags.
	for _, f := range cmd.Flags {
		if f.Required {
			if _, ok := flags[f.Name]; !ok {
				return nil, nil, fmt.Errorf("%w: --%s", ErrRequiredFlag, f.Name)
			}
		}
	}

	return positional, flags, nil
}

// parseFlag parses a single flag starting at raw[i].
// It returns the canonical flag name, value, number of consumed arguments, and any error.
// Short flags (e.g. -v) are resolved to their long name (e.g. "verbose").
func parseFlag(cmd *Command, raw []string, i int) (string, string, int, error) {
	arg := raw[i]

	// Strip leading dashes.
	name := strings.TrimLeft(arg, "-")

	// Handle --flag=value or -flag=value syntax.
	if eqIdx := strings.IndexByte(name, '='); eqIdx >= 0 {
		flagName := name[:eqIdx]
		flagValue := name[eqIdx+1:]

		f := findFlag(cmd, flagName)
		if f == nil {
			return flagName, flagValue, 1, nil
		}
		if err := validateFlagValue(f, flagValue); err != nil {
			return "", "", 0, err
		}
		return f.Name, flagValue, 1, nil
	}

	// Resolve short/long name via findFlag.
	f := findFlag(cmd, name)

	// Handle boolean flags that don't require a value.
	if f != nil && f.Type == FlagBool {
		// If next arg looks like a bool value, consume it.
		if i+1 < len(raw) {
			next := strings.ToLower(raw[i+1])
			if next == "true" || next == "false" || next == "1" || next == "0" {
				return f.Name, raw[i+1], 2, nil
			}
		}
		return f.Name, "true", 1, nil
	}

	// Handle --flag value syntax: next arg is the value.
	if i+1 >= len(raw) {
		return "", "", 0, fmt.Errorf("%w: flag --%s requires a value", ErrInvalidFlagValue, name)
	}
	value := raw[i+1]

	if f != nil {
		if err := validateFlagValue(f, value); err != nil {
			return "", "", 0, err
		}
		return f.Name, value, 2, nil
	}
	return name, value, 2, nil
}

// findFlag looks up a flag definition by name or short alias in the command.
func findFlag(cmd *Command, name string) *Flag {
	for idx := range cmd.Flags {
		if cmd.Flags[idx].Name == name {
			return &cmd.Flags[idx]
		}
		if cmd.Flags[idx].Short != 0 && len(name) == 1 && rune(name[0]) == cmd.Flags[idx].Short {
			return &cmd.Flags[idx]
		}
	}
	return nil
}

// validateFlagValue checks that value is valid for the given flag type.
func validateFlagValue(f *Flag, value string) error {
	switch f.Type {
	case FlagBool:
		if _, err := strconv.ParseBool(value); err != nil {
			return fmt.Errorf("%w: flag --%s: expected bool, got %q", ErrInvalidFlagValue, f.Name, value)
		}
	case FlagInt:
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("%w: flag --%s: expected int, got %q", ErrInvalidFlagValue, f.Name, value)
		}
	case FlagFloat:
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("%w: flag --%s: expected float, got %q", ErrInvalidFlagValue, f.Name, value)
		}
	case FlagString:
		// All values are valid strings.
	}
	return nil
}
