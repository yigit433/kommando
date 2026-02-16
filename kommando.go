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
	name              string
	description       string
	commands          []*Command
	globalFlags       []Flag
	output            io.Writer
	helpAdded         bool
	allowUnknownFlags bool
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

// WithGlobalFlags sets flags that are available to all commands.
// Global flags are merged with command-specific flags during parsing.
// If a command defines a flag with the same name, the command flag takes precedence.
func WithGlobalFlags(flags ...Flag) Option {
	return func(a *App) {
		a.globalFlags = flags
	}
}

// WithAllowUnknownFlags disables the unknown flag error.
// By default, unknown flags cause an ErrUnknownFlag error.
// When this option is set, unknown flags are silently accepted
// and their values are accessible through the Context.
func WithAllowUnknownFlags() Option {
	return func(a *App) {
		a.allowUnknownFlags = true
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
// It returns ErrInvalidName if the command name is empty,
// ErrDuplicateCommand if a command with the same name already exists,
// or ErrInvalidName if any flag has an empty name.
func (a *App) AddCommand(cmd *Command) error {
	if cmd.Name == "" {
		return fmt.Errorf("%w: command name cannot be empty", ErrInvalidName)
	}
	for _, f := range cmd.Flags {
		if f.Name == "" {
			return fmt.Errorf("%w: flag name cannot be empty in command %q", ErrInvalidName, cmd.Name)
		}
	}
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
// If --help or -h appears in the arguments, help is printed instead of
// executing the command.
func (a *App) Run(args []string) error {
	a.ensureHelp()

	if len(args) == 0 {
		a.printCommandList()
		return nil
	}

	// Top-level --help / -h shows the command list.
	if args[0] == "--help" || args[0] == "-h" {
		a.printCommandList()
		return nil
	}

	name := args[0]
	cmd := a.findCommand(name)
	if cmd == nil {
		return fmt.Errorf("%w: %s", ErrCommandNotFound, name)
	}

	// Resolve subcommands: walk down the command tree as long as the
	// next positional argument matches a subcommand.
	cmdArgs := args[1:]
	for len(cmd.SubCommands) > 0 && len(cmdArgs) > 0 {
		// Skip if next arg looks like a flag.
		if strings.HasPrefix(cmdArgs[0], "-") {
			break
		}
		sub := cmd.findSubCommand(cmdArgs[0])
		if sub == nil {
			break
		}
		cmd = sub
		cmdArgs = cmdArgs[1:]
	}

	// If any remaining arg is --help / -h, show help for the resolved command.
	for _, arg := range cmdArgs {
		if arg == "--help" || arg == "-h" {
			a.printCommandHelp(cmd)
			return nil
		}
		// Stop scanning after bare -- separator.
		if arg == "--" {
			break
		}
	}

	if cmd.Execute == nil {
		a.printCommandHelp(cmd)
		return nil
	}

	// Merge global flags with command flags. Command flags take precedence.
	mergedCmd := a.mergeGlobalFlags(cmd)

	positional, flags, err := parseArgs(mergedCmd, cmdArgs, a.allowUnknownFlags)
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

// ensureHelp adds the built-in help and completion commands exactly once.
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

	a.commands = append(a.commands, &Command{
		Name:        "completion",
		Description: "Generate shell completion script.",
		Execute: func(ctx *Context) error {
			args := ctx.Args()
			if len(args) == 0 {
				fmt.Fprintln(ctx.Output(), "Usage: completion <bash|zsh|fish|powershell>")
				return nil
			}
			return a.GenerateCompletion(ctx.Output(), Shell(args[0]))
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

	if len(a.globalFlags) > 0 {
		fmt.Fprintln(a.output)
		fmt.Fprintln(a.output, "Global Flags:")
		a.printFlagList(a.globalFlags)
	}
}

// mergeGlobalFlags returns a shallow copy of cmd with global flags appended,
// skipping any global flag whose name collides with a command-level flag.
func (a *App) mergeGlobalFlags(cmd *Command) *Command {
	if len(a.globalFlags) == 0 {
		return cmd
	}
	merged := *cmd
	merged.Flags = make([]Flag, len(cmd.Flags))
	copy(merged.Flags, cmd.Flags)
	for _, gf := range a.globalFlags {
		if findFlag(cmd, gf.Name) == nil {
			merged.Flags = append(merged.Flags, gf)
		}
	}
	return &merged
}

// printCommandHelp writes detailed help for a single command.
func (a *App) printCommandHelp(cmd *Command) {
	fmt.Fprintf(a.output, "%s - %s\n", cmd.Name, cmd.Description)

	if len(cmd.Aliases) > 0 {
		fmt.Fprintf(a.output, "Aliases: %s\n", strings.Join(cmd.Aliases, ", "))
	}

	if len(cmd.SubCommands) > 0 {
		fmt.Fprintln(a.output, "Commands:")
		for _, sub := range cmd.SubCommands {
			fmt.Fprintf(a.output, "  %-16s %s\n", sub.Name, sub.Description)
		}
	}

	if len(cmd.Flags) > 0 {
		fmt.Fprintln(a.output, "Flags:")
		a.printFlagList(cmd.Flags)
	}

	if len(a.globalFlags) > 0 {
		fmt.Fprintln(a.output, "Global Flags:")
		a.printFlagList(a.globalFlags)
	}
}

// printFlagList writes a formatted list of flags to the output.
func (a *App) printFlagList(flags []Flag) {
	for _, f := range flags {
		req := ""
		if f.Required {
			req = " (required)"
		}
		env := ""
		if f.Env != "" {
			env = fmt.Sprintf(" [env: %s]", f.Env)
		}
		flagLabel := fmt.Sprintf("--%s", f.Name)
		if f.Short != 0 {
			flagLabel = fmt.Sprintf("-%c, --%s", f.Short, f.Name)
		}
		fmt.Fprintf(a.output, "  %s <%s>\t%s%s%s\n", flagLabel, f.Type, f.Description, req, env)
	}
}
