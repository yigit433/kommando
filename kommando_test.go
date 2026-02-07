package kommando

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// newTestApp creates an App with a bytes.Buffer as output for testing.
func newTestApp(name string) (*App, *bytes.Buffer) {
	var buf bytes.Buffer
	app := New(name, WithOutput(&buf))
	return app, &buf
}

func TestNew(t *testing.T) {
	t.Run("default output", func(t *testing.T) {
		app := New("myapp")
		if app.name != "myapp" {
			t.Fatalf("expected name %q, got %q", "myapp", app.name)
		}
		if app.output == nil {
			t.Fatal("expected non-nil output")
		}
	})

	t.Run("with description", func(t *testing.T) {
		app := New("myapp", WithDescription("a test app"))
		if app.description != "a test app" {
			t.Fatalf("expected description %q, got %q", "a test app", app.description)
		}
	})

	t.Run("with output", func(t *testing.T) {
		var buf bytes.Buffer
		app := New("myapp", WithOutput(&buf))
		if app.output != &buf {
			t.Fatal("expected custom output writer")
		}
	})
}

func TestAddCommand(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		app, _ := newTestApp("myapp")
		err := app.AddCommand(&Command{
			Name:    "test",
			Execute: func(ctx *Context) error { return nil },
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		app, _ := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name:    "test",
			Execute: func(ctx *Context) error { return nil },
		})
		err := app.AddCommand(&Command{
			Name:    "test",
			Execute: func(ctx *Context) error { return nil },
		})
		if !errors.Is(err, ErrDuplicateCommand) {
			t.Fatalf("expected ErrDuplicateCommand, got %v", err)
		}
	})
}

func TestRunNoArgs(t *testing.T) {
	app, buf := newTestApp("testapp")
	_ = app.AddCommand(&Command{
		Name:        "greet",
		Description: "Say hello",
		Execute:     func(ctx *Context) error { return nil },
	})

	err := app.Run([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "testapp") {
		t.Fatalf("expected output to contain app name, got:\n%s", output)
	}
	if !strings.Contains(output, "greet") {
		t.Fatalf("expected output to contain command name, got:\n%s", output)
	}
}

func TestRunCommandExecution(t *testing.T) {
	app, buf := newTestApp("myapp")
	_ = app.AddCommand(&Command{
		Name: "echo",
		Flags: []Flag{
			{Name: "msg", Type: FlagString},
		},
		Execute: func(ctx *Context) error {
			msg, _ := ctx.String("msg")
			ctx.Output().Write([]byte(msg))
			return nil
		},
	})

	err := app.Run([]string{"echo", "--msg", "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "hello" {
		t.Fatalf("expected %q, got %q", "hello", buf.String())
	}
}

func TestRunAlias(t *testing.T) {
	app, buf := newTestApp("myapp")
	_ = app.AddCommand(&Command{
		Name:    "greet",
		Aliases: []string{"g"},
		Execute: func(ctx *Context) error {
			ctx.Output().Write([]byte("hi"))
			return nil
		},
	})

	err := app.Run([]string{"g"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "hi" {
		t.Fatalf("expected %q, got %q", "hi", buf.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	app, _ := newTestApp("myapp")
	err := app.Run([]string{"nonexistent"})
	if !errors.Is(err, ErrCommandNotFound) {
		t.Fatalf("expected ErrCommandNotFound, got %v", err)
	}
}

func TestParseArgs(t *testing.T) {
	cmd := &Command{
		Name: "test",
		Flags: []Flag{
			{Name: "name", Type: FlagString},
			{Name: "count", Type: FlagInt},
			{Name: "verbose", Type: FlagBool},
			{Name: "rate", Type: FlagFloat},
		},
	}

	tests := []struct {
		name      string
		raw       []string
		wantArgs  []string
		wantFlags map[string]string
		wantErr   bool
	}{
		{
			name:      "double dash equals",
			raw:       []string{"--name=alice"},
			wantFlags: map[string]string{"name": "alice"},
		},
		{
			name:      "double dash space",
			raw:       []string{"--name", "bob"},
			wantFlags: map[string]string{"name": "bob"},
		},
		{
			name:      "single dash equals",
			raw:       []string{"-name=charlie"},
			wantFlags: map[string]string{"name": "charlie"},
		},
		{
			name:      "single dash space",
			raw:       []string{"-name", "dave"},
			wantFlags: map[string]string{"name": "dave"},
		},
		{
			name:     "positional args",
			raw:      []string{"foo", "bar"},
			wantArgs: []string{"foo", "bar"},
		},
		{
			name:      "mixed flags and args",
			raw:       []string{"--name", "eve", "pos1", "--count", "5", "pos2"},
			wantArgs:  []string{"pos1", "pos2"},
			wantFlags: map[string]string{"name": "eve", "count": "5"},
		},
		{
			name:      "bare double dash separator",
			raw:       []string{"--name", "frank", "--", "--not-a-flag"},
			wantArgs:  []string{"--not-a-flag"},
			wantFlags: map[string]string{"name": "frank"},
		},
		{
			name:      "bool flag without value",
			raw:       []string{"--verbose"},
			wantFlags: map[string]string{"verbose": "true"},
		},
		{
			name:      "bool flag with value",
			raw:       []string{"--verbose", "false"},
			wantFlags: map[string]string{"verbose": "false"},
		},
		{
			name:    "empty args",
			raw:     []string{},
			wantErr: false,
		},
		{
			name:    "int flag invalid value",
			raw:     []string{"--count", "abc"},
			wantErr: true,
		},
		{
			name:    "float flag invalid value",
			raw:     []string{"--rate", "notfloat"},
			wantErr: true,
		},
		{
			name:    "flag missing value",
			raw:     []string{"--name"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, flags, err := parseArgs(cmd, tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Check positional args.
			if tt.wantArgs == nil {
				tt.wantArgs = []string{}
			}
			if len(args) != len(tt.wantArgs) {
				t.Fatalf("args: expected %v, got %v", tt.wantArgs, args)
			}
			for i, want := range tt.wantArgs {
				if args[i] != want {
					t.Fatalf("args[%d]: expected %q, got %q", i, want, args[i])
				}
			}

			// Check flags.
			if tt.wantFlags == nil {
				tt.wantFlags = map[string]string{}
			}
			for k, want := range tt.wantFlags {
				got, ok := flags[k]
				if !ok {
					t.Fatalf("flag %q: not found in parsed flags", k)
				}
				if got != want {
					t.Fatalf("flag %q: expected %q, got %q", k, want, got)
				}
			}
		})
	}
}

func TestParseFlagValidation(t *testing.T) {
	tests := []struct {
		name    string
		flag    Flag
		value   string
		wantErr bool
	}{
		{"valid bool true", Flag{Name: "b", Type: FlagBool}, "true", false},
		{"valid bool false", Flag{Name: "b", Type: FlagBool}, "false", false},
		{"valid bool 1", Flag{Name: "b", Type: FlagBool}, "1", false},
		{"invalid bool", Flag{Name: "b", Type: FlagBool}, "yes", true},
		{"valid int", Flag{Name: "i", Type: FlagInt}, "42", false},
		{"valid int negative", Flag{Name: "i", Type: FlagInt}, "-10", false},
		{"invalid int", Flag{Name: "i", Type: FlagInt}, "3.14", true},
		{"valid float", Flag{Name: "f", Type: FlagFloat}, "3.14", false},
		{"invalid float", Flag{Name: "f", Type: FlagFloat}, "abc", true},
		{"valid string", Flag{Name: "s", Type: FlagString}, "anything", false},
		{"empty string", Flag{Name: "s", Type: FlagString}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFlagValue(&tt.flag, tt.value)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !errors.Is(err, ErrInvalidFlagValue) {
				t.Fatalf("expected ErrInvalidFlagValue, got %v", err)
			}
		})
	}
}

func TestRequiredFlags(t *testing.T) {
	app, _ := newTestApp("myapp")
	_ = app.AddCommand(&Command{
		Name: "test",
		Flags: []Flag{
			{Name: "name", Type: FlagString, Required: true},
		},
		Execute: func(ctx *Context) error { return nil },
	})

	err := app.Run([]string{"test"})
	if !errors.Is(err, ErrRequiredFlag) {
		t.Fatalf("expected ErrRequiredFlag, got %v", err)
	}

	// With required flag provided, should succeed.
	err = app.Run([]string{"test", "--name", "alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContextAccessors(t *testing.T) {
	app, _ := newTestApp("myapp")
	_ = app.AddCommand(&Command{
		Name: "test",
		Flags: []Flag{
			{Name: "b", Type: FlagBool},
			{Name: "i", Type: FlagInt},
			{Name: "f", Type: FlagFloat},
			{Name: "s", Type: FlagString},
		},
		Execute: func(ctx *Context) error {
			// Test Bool.
			bv, err := ctx.Bool("b")
			if err != nil {
				t.Fatalf("Bool error: %v", err)
			}
			if bv != true {
				t.Fatalf("Bool: expected true, got %v", bv)
			}

			// Test Int.
			iv, err := ctx.Int("i")
			if err != nil {
				t.Fatalf("Int error: %v", err)
			}
			if iv != 42 {
				t.Fatalf("Int: expected 42, got %d", iv)
			}

			// Test Float.
			fv, err := ctx.Float("f")
			if err != nil {
				t.Fatalf("Float error: %v", err)
			}
			if fv != 3.14 {
				t.Fatalf("Float: expected 3.14, got %f", fv)
			}

			// Test String.
			sv, ok := ctx.String("s")
			if !ok {
				t.Fatal("String: expected ok=true")
			}
			if sv != "hello" {
				t.Fatalf("String: expected %q, got %q", "hello", sv)
			}

			// Test Args.
			if len(ctx.Args()) != 1 || ctx.Args()[0] != "pos" {
				t.Fatalf("Args: expected [pos], got %v", ctx.Args())
			}

			// Test Command.
			if ctx.Command().Name != "test" {
				t.Fatalf("Command: expected %q, got %q", "test", ctx.Command().Name)
			}

			// Test unset flag.
			_, ok = ctx.String("nonexistent")
			if ok {
				t.Fatal("String: expected ok=false for unset flag")
			}

			bv2, err := ctx.Bool("nonexistent")
			if err != nil {
				t.Fatalf("Bool unset error: %v", err)
			}
			if bv2 != false {
				t.Fatalf("Bool unset: expected false, got %v", bv2)
			}

			iv2, err := ctx.Int("nonexistent")
			if err != nil {
				t.Fatalf("Int unset error: %v", err)
			}
			if iv2 != 0 {
				t.Fatalf("Int unset: expected 0, got %d", iv2)
			}

			fv2, err := ctx.Float("nonexistent")
			if err != nil {
				t.Fatalf("Float unset error: %v", err)
			}
			if fv2 != 0 {
				t.Fatalf("Float unset: expected 0, got %f", fv2)
			}

			return nil
		},
	})

	err := app.Run([]string{"test", "--b", "true", "--i", "42", "--f", "3.14", "--s", "hello", "pos"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHelpCommand(t *testing.T) {
	app, buf := newTestApp("myapp")
	_ = app.AddCommand(&Command{
		Name:        "greet",
		Description: "Say hello",
		Aliases:     []string{"g"},
		Flags: []Flag{
			{Name: "loud", Type: FlagBool, Description: "shout"},
			{Name: "times", Type: FlagInt, Description: "repeat N times", Required: true},
		},
		Execute: func(ctx *Context) error { return nil },
	})

	err := app.Run([]string{"help", "greet"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"greet", "Say hello", "g", "loud", "times", "required"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestHelpUnknownCommand(t *testing.T) {
	app, _ := newTestApp("myapp")
	err := app.Run([]string{"help", "nonexistent"})
	if !errors.Is(err, ErrCommandNotFound) {
		t.Fatalf("expected ErrCommandNotFound, got %v", err)
	}
}

func TestHelpNotDuplicated(t *testing.T) {
	app, buf := newTestApp("myapp")
	_ = app.AddCommand(&Command{
		Name:    "test",
		Execute: func(ctx *Context) error { return nil },
	})

	// Run multiple times - help command should only appear once in list.
	_ = app.Run([]string{})
	buf.Reset()
	_ = app.Run([]string{})

	output := buf.String()
	// Count lines that start with the help command entry (indented name).
	count := strings.Count(output, "  help")
	if count != 1 {
		t.Fatalf("expected help command entry to appear once, appeared %d times:\n%s", count, output)
	}
}

func TestFlagDefault(t *testing.T) {
	app, buf := newTestApp("myapp")
	_ = app.AddCommand(&Command{
		Name: "test",
		Flags: []Flag{
			{Name: "name", Type: FlagString, Default: "world"},
		},
		Execute: func(ctx *Context) error {
			v, _ := ctx.String("name")
			ctx.Output().Write([]byte(v))
			return nil
		},
	})

	err := app.Run([]string{"test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "world" {
		t.Fatalf("expected %q, got %q", "world", buf.String())
	}
}

func TestCommandExecuteError(t *testing.T) {
	app, _ := newTestApp("myapp")
	wantErr := errors.New("command failed")
	_ = app.AddCommand(&Command{
		Name: "fail",
		Execute: func(ctx *Context) error {
			return wantErr
		},
	})

	err := app.Run([]string{"fail"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func TestFlagTypeString(t *testing.T) {
	tests := []struct {
		ft   FlagType
		want string
	}{
		{FlagString, "string"},
		{FlagBool, "bool"},
		{FlagInt, "int"},
		{FlagFloat, "float"},
	}
	for _, tt := range tests {
		if got := tt.ft.String(); got != tt.want {
			t.Fatalf("FlagType(%d).String() = %q, want %q", tt.ft, got, tt.want)
		}
	}
}

func TestHasAlias(t *testing.T) {
	cmd := &Command{
		Name:    "greet",
		Aliases: []string{"g", "hello"},
	}
	if !cmd.hasAlias("g") {
		t.Fatal("expected hasAlias(g) = true")
	}
	if !cmd.hasAlias("hello") {
		t.Fatal("expected hasAlias(hello) = true")
	}
	if cmd.hasAlias("nope") {
		t.Fatal("expected hasAlias(nope) = false")
	}
}

func TestShortFlag(t *testing.T) {
	cmd := &Command{
		Name: "test",
		Flags: []Flag{
			{Name: "verbose", Short: 'v', Type: FlagBool},
			{Name: "output", Short: 'o', Type: FlagString},
			{Name: "count", Short: 'n', Type: FlagInt},
		},
	}

	tests := []struct {
		name      string
		raw       []string
		wantFlags map[string]string
		wantArgs  []string
		wantErr   bool
	}{
		{
			name:      "short bool flag",
			raw:       []string{"-v"},
			wantFlags: map[string]string{"verbose": "true"},
		},
		{
			name:      "short bool flag with value",
			raw:       []string{"-v", "false"},
			wantFlags: map[string]string{"verbose": "false"},
		},
		{
			name:      "short string flag space",
			raw:       []string{"-o", "file.txt"},
			wantFlags: map[string]string{"output": "file.txt"},
		},
		{
			name:      "short string flag equals",
			raw:       []string{"-o=file.txt"},
			wantFlags: map[string]string{"output": "file.txt"},
		},
		{
			name:      "short int flag",
			raw:       []string{"-n", "5"},
			wantFlags: map[string]string{"count": "5"},
		},
		{
			name:      "mixed short and long",
			raw:       []string{"-v", "--output", "file.txt", "-n", "3"},
			wantFlags: map[string]string{"verbose": "true", "output": "file.txt", "count": "3"},
		},
		{
			name:      "short flags with positional",
			raw:       []string{"-o", "file.txt", "arg1", "arg2"},
			wantFlags: map[string]string{"output": "file.txt"},
			wantArgs:  []string{"arg1", "arg2"},
		},
		{
			name:    "short flag missing value",
			raw:     []string{"-o"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, flags, err := parseArgs(cmd, tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for k, want := range tt.wantFlags {
				got, ok := flags[k]
				if !ok {
					t.Fatalf("flag %q: not found (flags: %v)", k, flags)
				}
				if got != want {
					t.Fatalf("flag %q: expected %q, got %q", k, want, got)
				}
			}

			if tt.wantArgs != nil {
				if len(args) != len(tt.wantArgs) {
					t.Fatalf("args: expected %v, got %v", tt.wantArgs, args)
				}
				for i, want := range tt.wantArgs {
					if args[i] != want {
						t.Fatalf("args[%d]: expected %q, got %q", i, want, args[i])
					}
				}
			}
		})
	}
}

func TestShortFlagEndToEnd(t *testing.T) {
	app, buf := newTestApp("myapp")
	_ = app.AddCommand(&Command{
		Name: "greet",
		Flags: []Flag{
			{Name: "name", Short: 'n', Type: FlagString},
			{Name: "loud", Short: 'l', Type: FlagBool},
		},
		Execute: func(ctx *Context) error {
			name, _ := ctx.String("name")
			loud, _ := ctx.Bool("loud")
			msg := "Hello, " + name
			if loud {
				msg = strings.ToUpper(msg)
			}
			ctx.Output().Write([]byte(msg))
			return nil
		},
	})

	err := app.Run([]string{"greet", "-n", "World", "-l"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.String() != "HELLO, WORLD" {
		t.Fatalf("expected %q, got %q", "HELLO, WORLD", buf.String())
	}
}

func TestHelpFlag(t *testing.T) {
	executed := false
	makeApp := func() (*App, *bytes.Buffer) {
		app, buf := newTestApp("myapp")
		executed = false
		_ = app.AddCommand(&Command{
			Name:        "greet",
			Description: "Say hello",
			Aliases:     []string{"g"},
			Flags: []Flag{
				{Name: "name", Type: FlagString},
			},
			Execute: func(ctx *Context) error {
				executed = true
				return nil
			},
		})
		return app, buf
	}

	t.Run("command --help", func(t *testing.T) {
		app, buf := makeApp()
		err := app.Run([]string{"greet", "--help"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if executed {
			t.Fatal("command should not have been executed")
		}
		if !strings.Contains(buf.String(), "greet") || !strings.Contains(buf.String(), "Say hello") {
			t.Fatalf("expected command help, got:\n%s", buf.String())
		}
	})

	t.Run("command -h", func(t *testing.T) {
		app, buf := makeApp()
		err := app.Run([]string{"greet", "-h"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if executed {
			t.Fatal("command should not have been executed")
		}
		if !strings.Contains(buf.String(), "greet") {
			t.Fatalf("expected command help, got:\n%s", buf.String())
		}
	})

	t.Run("--help with flags", func(t *testing.T) {
		app, buf := makeApp()
		err := app.Run([]string{"greet", "--name", "alice", "--help"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if executed {
			t.Fatal("command should not have been executed")
		}
		if !strings.Contains(buf.String(), "greet") {
			t.Fatalf("expected command help, got:\n%s", buf.String())
		}
	})

	t.Run("top-level --help", func(t *testing.T) {
		app, buf := makeApp()
		err := app.Run([]string{"--help"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "myapp") || !strings.Contains(buf.String(), "greet") {
			t.Fatalf("expected command list, got:\n%s", buf.String())
		}
	})

	t.Run("top-level -h", func(t *testing.T) {
		app, buf := makeApp()
		err := app.Run([]string{"-h"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(buf.String(), "myapp") {
			t.Fatalf("expected command list, got:\n%s", buf.String())
		}
	})

	t.Run("alias --help", func(t *testing.T) {
		app, buf := makeApp()
		err := app.Run([]string{"g", "--help"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if executed {
			t.Fatal("command should not have been executed")
		}
		if !strings.Contains(buf.String(), "greet") {
			t.Fatalf("expected command help, got:\n%s", buf.String())
		}
	})

	t.Run("--help after bare -- is ignored", func(t *testing.T) {
		app, _ := makeApp()
		err := app.Run([]string{"greet", "--", "--help"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !executed {
			t.Fatal("command should have been executed (--help after -- is positional)")
		}
	})
}

func TestShortFlagHelpOutput(t *testing.T) {
	app, buf := newTestApp("myapp")
	_ = app.AddCommand(&Command{
		Name: "test",
		Flags: []Flag{
			{Name: "verbose", Short: 'v', Type: FlagBool, Description: "verbose output"},
			{Name: "output", Type: FlagString, Description: "output file"},
		},
		Execute: func(ctx *Context) error { return nil },
	})

	_ = app.Run([]string{"help", "test"})
	output := buf.String()

	// Short flag should show "-v, --verbose" format.
	if !strings.Contains(output, "-v, --verbose") {
		t.Fatalf("expected '-v, --verbose' in help output, got:\n%s", output)
	}
	// Flag without short should show only "--output".
	if !strings.Contains(output, "--output") {
		t.Fatalf("expected '--output' in help output, got:\n%s", output)
	}
	if strings.Contains(output, "-, --output") {
		t.Fatalf("flag without short should not have '-, ' prefix, got:\n%s", output)
	}
}

func TestSubCommands(t *testing.T) {
	t.Run("basic subcommand execution", func(t *testing.T) {
		app, buf := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name:        "server",
			Description: "Server management",
			SubCommands: []*Command{
				{
					Name:        "start",
					Description: "Start the server",
					Flags: []Flag{
						{Name: "port", Short: 'p', Type: FlagInt, Default: "8080"},
					},
					Execute: func(ctx *Context) error {
						port, _ := ctx.Int("port")
						fmt.Fprintf(ctx.Output(), "started:%d", port)
						return nil
					},
				},
				{
					Name:        "stop",
					Description: "Stop the server",
					Execute: func(ctx *Context) error {
						ctx.Output().Write([]byte("stopped"))
						return nil
					},
				},
			},
		})

		err := app.Run([]string{"server", "start", "--port", "3000"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.String() != "started:3000" {
			t.Fatalf("expected %q, got %q", "started:3000", buf.String())
		}
	})

	t.Run("subcommand with default flag", func(t *testing.T) {
		app, buf := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name: "server",
			SubCommands: []*Command{
				{
					Name: "start",
					Flags: []Flag{
						{Name: "port", Type: FlagInt, Default: "8080"},
					},
					Execute: func(ctx *Context) error {
						port, _ := ctx.Int("port")
						fmt.Fprintf(ctx.Output(), "port:%d", port)
						return nil
					},
				},
			},
		})

		err := app.Run([]string{"server", "start"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.String() != "port:8080" {
			t.Fatalf("expected %q, got %q", "port:8080", buf.String())
		}
	})

	t.Run("subcommand with alias", func(t *testing.T) {
		app, buf := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name: "server",
			SubCommands: []*Command{
				{
					Name:    "start",
					Aliases: []string{"s"},
					Execute: func(ctx *Context) error {
						ctx.Output().Write([]byte("ok"))
						return nil
					},
				},
			},
		})

		err := app.Run([]string{"server", "s"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.String() != "ok" {
			t.Fatalf("expected %q, got %q", "ok", buf.String())
		}
	})

	t.Run("parent without execute shows help", func(t *testing.T) {
		app, buf := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name:        "server",
			Description: "Server management",
			SubCommands: []*Command{
				{Name: "start", Description: "Start the server", Execute: func(ctx *Context) error { return nil }},
				{Name: "stop", Description: "Stop the server", Execute: func(ctx *Context) error { return nil }},
			},
		})

		err := app.Run([]string{"server"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "start") || !strings.Contains(output, "stop") {
			t.Fatalf("expected subcommand list in help, got:\n%s", output)
		}
	})

	t.Run("unknown subcommand falls to parent", func(t *testing.T) {
		executed := false
		app, _ := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name: "server",
			SubCommands: []*Command{
				{Name: "start", Execute: func(ctx *Context) error { return nil }},
			},
			Execute: func(ctx *Context) error {
				executed = true
				return nil
			},
		})

		err := app.Run([]string{"server", "unknown"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !executed {
			t.Fatal("parent Execute should have been called")
		}
	})

	t.Run("subcommand --help", func(t *testing.T) {
		app, buf := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name: "server",
			SubCommands: []*Command{
				{
					Name:        "start",
					Description: "Start it",
					Flags:       []Flag{{Name: "port", Type: FlagInt}},
					Execute:     func(ctx *Context) error { return nil },
				},
			},
		})

		err := app.Run([]string{"server", "start", "--help"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "start") || !strings.Contains(output, "port") {
			t.Fatalf("expected subcommand help, got:\n%s", output)
		}
	})

	t.Run("nested subcommands", func(t *testing.T) {
		app, buf := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name: "db",
			SubCommands: []*Command{
				{
					Name: "migrate",
					SubCommands: []*Command{
						{
							Name: "up",
							Execute: func(ctx *Context) error {
								ctx.Output().Write([]byte("migrated"))
								return nil
							},
						},
					},
				},
			},
		})

		err := app.Run([]string{"db", "migrate", "up"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if buf.String() != "migrated" {
			t.Fatalf("expected %q, got %q", "migrated", buf.String())
		}
	})

	t.Run("help command shows subcommands", func(t *testing.T) {
		app, buf := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name:        "server",
			Description: "Server ops",
			SubCommands: []*Command{
				{Name: "start", Description: "Start the server"},
				{Name: "stop", Description: "Stop the server"},
			},
		})

		err := app.Run([]string{"help", "server"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		output := buf.String()
		if !strings.Contains(output, "Commands:") {
			t.Fatalf("expected 'Commands:' section, got:\n%s", output)
		}
		if !strings.Contains(output, "start") || !strings.Contains(output, "stop") {
			t.Fatalf("expected subcommands listed, got:\n%s", output)
		}
	})

	t.Run("subcommand context has correct command", func(t *testing.T) {
		app, _ := newTestApp("myapp")
		_ = app.AddCommand(&Command{
			Name: "server",
			SubCommands: []*Command{
				{
					Name: "start",
					Execute: func(ctx *Context) error {
						if ctx.Command().Name != "start" {
							t.Fatalf("expected command name %q, got %q", "start", ctx.Command().Name)
						}
						return nil
					},
				},
			},
		})

		err := app.Run([]string{"server", "start"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
