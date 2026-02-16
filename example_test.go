package kommando_test

import (
	"fmt"
	"os"

	"github.com/yigit433/kommando/v3"
)

func ExampleNew() {
	app := kommando.New("myapp",
		kommando.WithDescription("A sample CLI application"),
		kommando.WithOutput(os.Stdout),
	)
	_ = app.Run([]string{})
	// Output:
	// Welcome to myapp! A sample CLI application
	// Type 'help <command>' to get help with any command.
	//
	//   help             Show help for a command.
	//   completion       Generate shell completion script.
}

func ExampleApp_AddCommand() {
	app := kommando.New("myapp", kommando.WithOutput(os.Stdout))
	err := app.AddCommand(&kommando.Command{
		Name:        "greet",
		Description: "Greet someone",
		Aliases:     []string{"g"},
		Flags: []kommando.Flag{
			{Name: "name", Description: "who to greet", Type: kommando.FlagString, Default: "World"},
		},
		Execute: func(ctx *kommando.Context) error {
			name, _ := ctx.String("name")
			fmt.Fprintf(ctx.Output(), "Hello, %s!\n", name)
			return nil
		},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}

	_ = app.Run([]string{"greet"})
	// Output:
	// Hello, World!
}

func ExampleApp_Run() {
	app := kommando.New("calc", kommando.WithOutput(os.Stdout))
	_ = app.AddCommand(&kommando.Command{
		Name:        "add",
		Description: "Add two numbers",
		Flags: []kommando.Flag{
			{Name: "a", Type: kommando.FlagInt, Required: true},
			{Name: "b", Type: kommando.FlagInt, Required: true},
		},
		Execute: func(ctx *kommando.Context) error {
			a, _ := ctx.Int("a")
			b, _ := ctx.Int("b")
			fmt.Fprintf(ctx.Output(), "%d + %d = %d\n", a, b, a+b)
			return nil
		},
	})

	if err := app.Run([]string{"add", "--a", "3", "--b", "5"}); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
	// Output:
	// 3 + 5 = 8
}
