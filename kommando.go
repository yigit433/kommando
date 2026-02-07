// Package kommando is a minimalist CLI framework for Go.
//
// It provides a simple API for building command-line applications with
// typed flags, command aliases, and automatic help generation.
package kommando

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// App is the top-level CLI application.
type App struct {
	name        string
	description string
	commands    []*Command
	output      io.Writer
	helpAdded   bool
}

// Option configures an App.
type Option func(*App)

// WithDescription sets the application description shown in help output.
func WithDescription(desc string) Option {
	return func(a *App) {
		a.description = desc
	}
}

// WithOutput sets the writer for all application output.
// Defaults to os.Stdout.
func WithOutput(w io.Writer) Option {
	return func(a *App) {
		a.output = w
	}
}

// New creates a new CLI application with the given name and options.
func New(name string, opts ...Option) *App {
	a := &App{
		name:   name,
		output: os.Stdout,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// AddCommand registers a command with the application.
// It returns ErrDuplicateCommand if a command with the same name already exists.
func (a *App) AddCommand(cmd *Command) error {
	for _, existing := range a.commands {
		if existing.Name == cmd.Name {
			return fmt.Errorf("%w: %s", ErrDuplicateCommand, cmd.Name)
		}
	}
	a.commands = append(a.commands, cmd)
	return nil
}

// Run parses the given arguments and executes the matching command.
// Pass os.Args[1:] for normal CLI usage.
func (a *App) Run(args []string) error {
	a.ensureHelp()

	if len(args) == 0 {
		a.printCommandList()
		return nil
	}

	name := args[0]
	cmd := a.findCommand(name)
	if cmd == nil {
		return fmt.Errorf("%w: %s", ErrCommandNotFound, name)
	}

	positional, flags, err := parseArgs(cmd, args[1:])
	if err != nil {
		return err
	}

	ctx := &Context{
		command: cmd,
		args:    positional,
		flags:   flags,
		output:  a.output,
	}

	return cmd.Execute(ctx)
}

// ensureHelp adds the built-in help command exactly once.
func (a *App) ensureHelp() {
	if a.helpAdded {
		return
	}
	a.helpAdded = true

	a.commands = append(a.commands, &Command{
		Name:        "help",
		Description: "Show help for a command.",
		Execute: func(ctx *Context) error {
			if len(ctx.Args()) > 0 {
				name := ctx.Args()[0]
				cmd := a.findCommand(name)
				if cmd == nil {
					return fmt.Errorf("%w: %s", ErrCommandNotFound, name)
				}
				a.printCommandHelp(cmd)
				return nil
			}
			a.printCommandList()
			return nil
		},
	})
}

// findCommand looks up a command by name or alias.
func (a *App) findCommand(name string) *Command {
	for _, cmd := range a.commands {
		if cmd.Name == name || cmd.hasAlias(name) {
			return cmd
		}
	}
	return nil
}

// printCommandList writes the list of all commands to the output.
func (a *App) printCommandList() {
	fmt.Fprintf(a.output, "Welcome to %s!", a.name)
	if a.description != "" {
		fmt.Fprintf(a.output, " %s", a.description)
	}
	fmt.Fprintln(a.output)
	fmt.Fprintln(a.output, "Type 'help <command>' to get help with any command.")
	fmt.Fprintln(a.output)
	for _, cmd := range a.commands {
		fmt.Fprintf(a.output, "  %-16s %s\n", cmd.Name, cmd.Description)
	}
}

// printCommandHelp writes detailed help for a single command.
func (a *App) printCommandHelp(cmd *Command) {
	fmt.Fprintf(a.output, "%s - %s\n", cmd.Name, cmd.Description)

	if len(cmd.Aliases) > 0 {
		fmt.Fprintf(a.output, "Aliases: %s\n", strings.Join(cmd.Aliases, ", "))
	}

	if len(cmd.Flags) > 0 {
		fmt.Fprintln(a.output, "Flags:")
		for _, f := range cmd.Flags {
			req := ""
			if f.Required {
				req = " (required)"
			}
			fmt.Fprintf(a.output, "  --%s <%s>\t%s%s\n", f.Name, f.Type, f.Description, req)
		}
	}
}
