## Kommando [![Go Report Card](https://goreportcard.com/badge/github.com/yigit433/kommando)](https://goreportcard.com/report/github.com/yigit433/kommando)

A minimalist CLI framework for Go.

### Installation

```
go get github.com/yigit433/kommando
```

### Example

```go
package main

import (
    "fmt"
    "os"

    "github.com/yigit433/kommando"
)

func main() {
    app := kommando.New("myapp", kommando.WithOutput(os.Stdout))

    app.AddCommand(&kommando.Command{
        Name:        "greet",
        Description: "Greet someone",
        Aliases:     []string{"g"},
        Flags: []kommando.Flag{
            {Name: "loud", Description: "shout", Type: kommando.FlagBool},
            {Name: "times", Description: "repeat N times", Type: kommando.FlagInt, Required: true},
        },
        Execute: func(ctx *kommando.Context) error {
            times, _ := ctx.Int("times")
            for i := 0; i < int(times); i++ {
                fmt.Fprintln(ctx.Output(), "Hello!")
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

### Error Handling

Kommando returns errors instead of panicking. Use `errors.Is` to check for specific conditions:

```go
import "errors"

err := app.Run(os.Args[1:])
if errors.Is(err, kommando.ErrCommandNotFound) {
    // handle unknown command
}
if errors.Is(err, kommando.ErrRequiredFlag) {
    // handle missing required flag
}
```

### Available Errors

- `ErrDuplicateCommand` - a command with that name already exists
- `ErrRequiredFlag` - a required flag was not provided
- `ErrInvalidFlagValue` - a flag value could not be parsed as the expected type
- `ErrCommandNotFound` - the specified command does not exist

### Flag Types

- `kommando.FlagString` (default)
- `kommando.FlagBool`
- `kommando.FlagInt`
- `kommando.FlagFloat`
