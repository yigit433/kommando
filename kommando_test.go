package kommando

import (
	"bytes"
	"errors"
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
