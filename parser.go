package kommando

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// sliceSep is the internal separator used to join multiple values for
// FlagStringSlice within the map[string]string storage. Null byte is chosen
// because OS-level argv and environment variables cannot contain it, so it
// will never collide with user-supplied data.
const sliceSep = "\x00"

// allSameRune reports whether every character in s is the same single-byte rune.
func allSameRune(s string) bool {
	if len(s) == 0 {
		return false
	}
	r := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != r {
			return false
		}
	}
	return true
}

// commaToSliceSep replaces commas with sliceSep for FlagStringSlice values.
func commaToSliceSep(s string) string {
	return strings.ReplaceAll(s, ",", sliceSep)
}

// parseArgs parses raw command-line arguments into positional args and flag values.
// It supports --flag=value, --flag value, -flag=value, -flag value syntax,
// and the -- bare separator to stop flag parsing.
// When allowUnknown is false, any flag not defined on the command returns ErrUnknownFlag.
func parseArgs(cmd *Command, raw []string, allowUnknown bool) ([]string, map[string]string, error) {
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

		name, value, consumed, f, err := parseFlag(cmd, raw, i, allowUnknown)
		if err != nil {
			return nil, nil, err
		}

		switch {
		case f != nil && f.Type == FlagStringSlice:
			// Accumulate: append to existing value with sliceSep.
			if prev, ok := flags[name]; ok {
				flags[name] = prev + sliceSep + value
			} else {
				flags[name] = value
			}
		case f != nil && f.Type == FlagCount:
			// Increment: parse existing count and add new increment.
			prev := 0
			if existing, ok := flags[name]; ok {
				prev, _ = strconv.Atoi(existing)
			}
			add, _ := strconv.Atoi(value)
			flags[name] = strconv.Itoa(prev + add)
		default:
			flags[name] = value
		}
		i += consumed
	}

	// Apply environment variables for flags not provided on the command line.
	for _, f := range cmd.Flags {
		if _, ok := flags[f.Name]; !ok && f.Env != "" {
			if envVal, exists := os.LookupEnv(f.Env); exists {
				if err := validateFlagValue(&f, envVal); err != nil {
					return nil, nil, err
				}
				if f.Type == FlagStringSlice {
					flags[f.Name] = commaToSliceSep(envVal)
				} else {
					flags[f.Name] = envVal
				}
			}
		}
	}

	// Apply defaults for flags not provided.
	for _, f := range cmd.Flags {
		if _, ok := flags[f.Name]; !ok && f.Default != "" {
			if f.Type == FlagStringSlice {
				flags[f.Name] = commaToSliceSep(f.Default)
			} else {
				flags[f.Name] = f.Default
			}
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
// It returns the canonical flag name, value, number of consumed arguments,
// the matched *Flag (nil for unknown flags), and any error.
// Short flags (e.g. -v) are resolved to their long name (e.g. "verbose").
// When allowUnknown is false, unrecognized flags return ErrUnknownFlag.
func parseFlag(cmd *Command, raw []string, i int, allowUnknown bool) (string, string, int, *Flag, error) {
	arg := raw[i]

	// Reject args with 3+ leading dashes (e.g. ---flag).
	if strings.HasPrefix(arg, "---") {
		return "", "", 0, nil, fmt.Errorf("%w: %s", ErrInvalidFlagValue, arg)
	}

	// Strip leading dashes.
	name := strings.TrimLeft(arg, "-")

	// Handle --flag=value or -flag=value syntax.
	if eqIdx := strings.IndexByte(name, '='); eqIdx >= 0 {
		flagName := name[:eqIdx]
		flagValue := name[eqIdx+1:]

		f := findFlag(cmd, flagName)
		if f == nil {
			if !allowUnknown {
				return "", "", 0, nil, fmt.Errorf("%w: --%s", ErrUnknownFlag, flagName)
			}
			return flagName, flagValue, 1, nil, nil
		}
		if f.Type == FlagStringSlice {
			return f.Name, commaToSliceSep(flagValue), 1, f, nil
		}
		if err := validateFlagValue(f, flagValue); err != nil {
			return "", "", 0, nil, err
		}
		return f.Name, flagValue, 1, f, nil
	}

	// Resolve short/long name via findFlag.
	f := findFlag(cmd, name)

	// Handle bundled short count flags: -vvv where all chars are the same
	// and the single-char flag is FlagCount.
	if f == nil && len(name) > 1 && allSameRune(name) {
		singleChar := string(name[0])
		cf := findFlag(cmd, singleChar)
		if cf != nil && cf.Type == FlagCount {
			return cf.Name, strconv.Itoa(len(name)), 1, cf, nil
		}
	}

	if f == nil && !allowUnknown {
		return "", "", 0, nil, fmt.Errorf("%w: --%s", ErrUnknownFlag, name)
	}

	// Handle boolean flags that don't require a value.
	if f != nil && f.Type == FlagBool {
		// If next arg looks like a bool value, consume it.
		if i+1 < len(raw) {
			next := strings.ToLower(raw[i+1])
			if next == "true" || next == "false" || next == "1" || next == "0" {
				return f.Name, raw[i+1], 2, f, nil
			}
		}
		return f.Name, "true", 1, f, nil
	}

	// Handle FlagCount: does not consume a value argument.
	if f != nil && f.Type == FlagCount {
		return f.Name, "1", 1, f, nil
	}

	// Handle --flag value syntax: next arg is the value.
	if i+1 >= len(raw) {
		return "", "", 0, nil, fmt.Errorf("%w: flag --%s requires a value", ErrInvalidFlagValue, name)
	}
	value := raw[i+1]

	if f != nil {
		if f.Type == FlagStringSlice {
			return f.Name, commaToSliceSep(value), 2, f, nil
		}
		if err := validateFlagValue(f, value); err != nil {
			return "", "", 0, nil, err
		}
		return f.Name, value, 2, f, nil
	}
	return name, value, 2, nil, nil
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
	case FlagString, FlagStringSlice:
		// All values are valid strings.
	case FlagCount:
		// Validation not needed; parser produces values.
	}
	return nil
}
