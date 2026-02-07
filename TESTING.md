# Testing Guide

## Running Tests

```bash
# Run all tests
go test ./...

# Verbose output (see each test name and result)
go test ./... -v

# Run with coverage report
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run with race detector (requires cgo)
CGO_ENABLED=1 go test -race ./...

# Run a specific test by name
go test -v -run TestParseArgs ./...

# Run a specific sub-test
go test -v -run TestParseArgs/bool_flag_without_value ./...
```

## Test Structure

All tests are in the `kommando` package (white-box testing) so they can access unexported functions like `parseArgs`, `findFlag`, and `validateFlagValue`.

### Files

| File | Purpose |
|------|---------|
| `kommando_test.go` | Main test suite with unit and integration tests |
| `example_test.go` | Godoc examples (package `kommando_test`, black-box) |

### Test Helper

```go
func newTestApp(name string) (*App, *bytes.Buffer)
```

Creates an `App` with a `bytes.Buffer` as its output writer. This lets tests capture and assert on output without printing to stdout.

### Test Categories

#### Constructor & Options
- `TestNew` - `New()` with default values and functional options (`WithDescription`, `WithOutput`)

#### Command Registration
- `TestAddCommand` - successful add and `ErrDuplicateCommand` on duplicates

#### Command Execution
- `TestRunNoArgs` - prints command list when no arguments given
- `TestRunCommandExecution` - correct command runs with parsed flags
- `TestRunAlias` - commands can be invoked by alias
- `TestRunUnknownCommand` - returns `ErrCommandNotFound`
- `TestCommandExecuteError` - errors from `Execute` are propagated

#### Flag Parsing (table-driven)
- `TestParseArgs` - 13 sub-cases covering:
  - `--flag=value` and `--flag value` syntax
  - `-flag=value` and `-flag value` syntax
  - Positional arguments
  - Mixed flags and positional args
  - `--` bare separator (POSIX)
  - Bool flags with and without explicit value
  - Invalid values, missing values, empty args

#### Flag Validation
- `TestParseFlagValidation` - validates each `FlagType` (bool, int, float, string) with valid and invalid inputs

#### Required Flags
- `TestRequiredFlags` - returns `ErrRequiredFlag` when missing, succeeds when provided

#### Context Accessors
- `TestContextAccessors` - tests `Bool()`, `Int()`, `Float()`, `String()`, `Args()`, `Command()`, and behavior for unset flags

#### Help System
- `TestHelpCommand` - `help <cmd>` prints flags, aliases, description
- `TestHelpUnknownCommand` - `help nonexistent` returns `ErrCommandNotFound`
- `TestHelpNotDuplicated` - multiple `Run()` calls don't duplicate the help command

#### Defaults & Types
- `TestFlagDefault` - default values are applied when flag is not provided
- `TestFlagTypeString` - `FlagType.String()` returns correct strings
- `TestHasAlias` - `hasAlias()` helper works correctly

### Godoc Examples

In `example_test.go` (package `kommando_test`):

- `ExampleNew` - creating an app and displaying help
- `ExampleApp_AddCommand` - adding a command with flags and defaults
- `ExampleApp_Run` - running a command with parsed int flags

These are runnable examples that also serve as documentation. They are verified by `go test` via output matching.

## Writing New Tests

When adding a new feature, follow this pattern:

```go
func TestMyFeature(t *testing.T) {
    app, buf := newTestApp("myapp")
    _ = app.AddCommand(&Command{
        Name: "test",
        Flags: []Flag{
            {Name: "myflag", Type: FlagString},
        },
        Execute: func(ctx *Context) error {
            val, _ := ctx.String("myflag")
            ctx.Output().Write([]byte(val))
            return nil
        },
    })

    err := app.Run([]string{"test", "--myflag", "hello"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if buf.String() != "hello" {
        t.Fatalf("expected %q, got %q", "hello", buf.String())
    }
}
```

For table-driven tests:

```go
func TestMyFeatureCases(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid", "hello", "hello", false},
        {"empty", "", "", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ... test logic
        })
    }
}
```

## Coverage Target

The project targets **80%+ statement coverage**. Current coverage is ~95%.
