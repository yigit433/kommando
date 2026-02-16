## Kommando [![Go Report Card](https://goreportcard.com/badge/github.com/yigit433/kommando)](https://goreportcard.com/report/github.com/yigit433/kommando)

A minimalist CLI framework for Go.

### Features

- Typed flags with short aliases (`-p`, `--port`)
- Nested subcommands (`app server start`)
- Global flags and environment variable binding
- Automatic `--help` / `-h` on every command
- Shell completion (Bash, Zsh, Fish, PowerShell)
- Error returns instead of panics

### Installation

```
go get github.com/yigit433/kommando/v3
```

### Quick Start

```go
package main

import (
    "fmt"
    "os"

    "github.com/yigit433/kommando/v3"
)

func main() {
    app := kommando.New("myapp",
        kommando.WithDescription("My CLI tool"),
        kommando.WithOutput(os.Stdout),
        kommando.WithGlobalFlags(
            kommando.Flag{Name: "verbose", Short: 'v', Type: kommando.FlagBool, Description: "verbose output"},
        ),
    )

    app.AddCommand(&kommando.Command{
        Name:        "greet",
        Description: "Greet someone",
        Aliases:     []string{"g"},
        Flags: []kommando.Flag{
            {Name: "name", Short: 'n', Description: "who to greet", Type: kommando.FlagString, Default: "World"},
            {Name: "times", Short: 't', Description: "repeat N times", Type: kommando.FlagInt, Default: "1"},
        },
        Execute: func(ctx *kommando.Context) error {
            name, _ := ctx.String("name")
            times, _ := ctx.Int("times")
            for i := 0; i < int(times); i++ {
                fmt.Fprintf(ctx.Output(), "Hello, %s!\n", name)
            }
            return nil
        },
    })

    if err := app.Run(os.Args[1:]); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}
```

```
$ myapp greet -n Alice -t 3
Hello, Alice!
Hello, Alice!
Hello, Alice!

$ myapp greet --help
greet - Greet someone
Aliases: g
Flags:
  -n, --name <string>   who to greet
  -t, --times <int>     repeat N times
Global Flags:
  -v, --verbose <bool>  verbose output
```

### Subcommands

```go
app.AddCommand(&kommando.Command{
    Name:        "server",
    Description: "Server management",
    SubCommands: []*kommando.Command{
        {
            Name:        "start",
            Description: "Start the server",
            Flags: []kommando.Flag{
                {Name: "port", Short: 'p', Type: kommando.FlagInt, Default: "8080", Env: "APP_PORT"},
            },
            Execute: func(ctx *kommando.Context) error {
                port, _ := ctx.Int("port")
                fmt.Fprintf(ctx.Output(), "Listening on :%d\n", port)
                return nil
            },
        },
        {
            Name:        "stop",
            Description: "Stop the server",
            Execute: func(ctx *kommando.Context) error {
                fmt.Fprintln(ctx.Output(), "Server stopped.")
                return nil
            },
        },
    },
})
```

```
$ myapp server start --port 3000
Listening on :3000

$ APP_PORT=9090 myapp server start
Listening on :9090

$ myapp server --help
server - Server management
Commands:
  start            Start the server
  stop             Stop the server
```

### Environment Variable Binding

Flags can read values from environment variables. Priority order: **CLI flag > env var > default value**.

```go
kommando.Flag{
    Name:     "token",
    Type:     kommando.FlagString,
    Required: true,
    Env:      "API_TOKEN",
}
```

```
$ API_TOKEN=secret myapp deploy    # works without --token flag
```

### Shell Completion

Built-in `completion` command generates scripts for your shell:

```
$ myapp completion bash >> ~/.bashrc
$ myapp completion zsh >> ~/.zshrc
$ myapp completion fish > ~/.config/fish/completions/myapp.fish
$ myapp completion powershell >> $PROFILE
```

Or programmatically:

```go
app.GenerateCompletion(os.Stdout, kommando.Bash)
```

### Error Handling

Kommando returns errors instead of panicking. Use `errors.Is` to check for specific conditions:

```go
err := app.Run(os.Args[1:])
if errors.Is(err, kommando.ErrCommandNotFound) {
    // unknown command
}
if errors.Is(err, kommando.ErrRequiredFlag) {
    // missing required flag
}
```

### Unknown Flag Handling

By default, unknown flags return an error. Use `WithAllowUnknownFlags` to accept them:

```go
// Default: unknown flags cause ErrUnknownFlag
app := kommando.New("myapp")

// Allow unknown flags (values accessible via ctx.String)
app := kommando.New("myapp", kommando.WithAllowUnknownFlags())
```

### Bare `--` Separator

Everything after a bare `--` is treated as positional arguments, even if it looks like a flag:

```
$ myapp greet --name Alice -- --not-a-flag
# Args: ["--not-a-flag"], Flags: {name: "Alice"}
```

### Available Errors

| Error | Description |
|-------|-------------|
| `ErrDuplicateCommand` | A command with that name already exists |
| `ErrRequiredFlag` | A required flag was not provided |
| `ErrInvalidFlagValue` | Flag value could not be parsed as the expected type |
| `ErrCommandNotFound` | The specified command does not exist |
| `ErrUnknownFlag` | A flag not defined for the command was provided |
| `ErrUnsupportedShell` | Completion requested for an unsupported shell |
| `ErrInvalidName` | A command or flag has an empty name |

### Flag Types

| Type | Go accessor | Zero value |
|------|-------------|------------|
| `FlagString` (default) | `ctx.String("name")` | `""` |
| `FlagBool` | `ctx.Bool("verbose")` | `false` |
| `FlagInt` | `ctx.Int("port")` | `0` |
| `FlagFloat` | `ctx.Float("rate")` | `0.0` |
| `FlagStringSlice` | `ctx.StringSlice("tag")` | `nil` |
| `FlagCount` | `ctx.Count("verbose")` | `0` |

### Slice Flags

`FlagStringSlice` collects multiple string values via repetition or commas:

```go
kommando.Flag{Name: "tag", Short: 't', Type: kommando.FlagStringSlice}
```

```
$ myapp test --tag a --tag b        # ["a", "b"]
$ myapp test --tag a,b,c            # ["a", "b", "c"]
$ myapp test --tag a,b --tag c      # ["a", "b", "c"]
$ myapp test --tag=x,y              # ["x", "y"]
```

Environment variables and defaults use comma-separated values: `TAG=a,b,c` or `Default: "x,y"`.

### Count Flags

`FlagCount` increments a counter each time the flag appears:

```go
kommando.Flag{Name: "verbose", Short: 'v', Type: kommando.FlagCount}
```

```
$ myapp test --verbose              # 1
$ myapp test --verbose --verbose    # 2
$ myapp test -vvv                   # 3
$ myapp test -vv --verbose          # 3
```
