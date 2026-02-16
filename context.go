package kommando

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Context provides access to parsed flags, positional arguments,
// and the output writer during command execution.
type Context struct {
	command *Command
	args    []string
	flags   map[string]string
	output  io.Writer
}

// Args returns the positional arguments that were not parsed as flags.
func (c *Context) Args() []string {
	return c.args
}

// Command returns the command being executed.
func (c *Context) Command() *Command {
	return c.command
}

// Output returns the io.Writer configured for the application.
func (c *Context) Output() io.Writer {
	return c.output
}

// String returns the string value of the named flag and true if it was set.
// If the flag was not provided, it returns ("", false).
func (c *Context) String(name string) (string, bool) {
	v, ok := c.flags[name]
	return v, ok
}

// Bool returns the boolean value of the named flag.
// It returns an error if the value cannot be parsed as a boolean.
// If the flag was not set, it returns (false, nil).
func (c *Context) Bool(name string) (bool, error) {
	v, ok := c.flags[name]
	if !ok {
		return false, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%w: flag %q: %v", ErrInvalidFlagValue, name, err)
	}
	return b, nil
}

// Int returns the int64 value of the named flag.
// It returns an error if the value cannot be parsed as an integer.
// If the flag was not set, it returns (0, nil).
func (c *Context) Int(name string) (int64, error) {
	v, ok := c.flags[name]
	if !ok {
		return 0, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: flag %q: %v", ErrInvalidFlagValue, name, err)
	}
	return n, nil
}

// StringSlice returns the string slice value of the named flag and true if it was set.
// If the flag was not provided, it returns (nil, false).
func (c *Context) StringSlice(name string) ([]string, bool) {
	v, ok := c.flags[name]
	if !ok {
		return nil, false
	}
	return strings.Split(v, sliceSep), true
}

// Count returns the count value of the named flag.
// If the flag was not set, it returns 0.
func (c *Context) Count(name string) int {
	v, ok := c.flags[name]
	if !ok {
		return 0
	}
	n, _ := strconv.Atoi(v)
	return n
}

// Float returns the float64 value of the named flag.
// It returns an error if the value cannot be parsed as a float.
// If the flag was not set, it returns (0, nil).
func (c *Context) Float(name string) (float64, error) {
	v, ok := c.flags[name]
	if !ok {
		return 0, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: flag %q: %v", ErrInvalidFlagValue, name, err)
	}
	return f, nil
}
