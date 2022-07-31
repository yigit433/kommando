package kommando

import (
	"fmt"
	"github.com/yigit433/kommando/types"
	"testing"
)

func TestKommandoApp(t *testing.T) {
	app := types.Config{
		AppName: "Kommando Test App",
	}

	app.AddCommand(
		&types.Command{
			Name:        "test",
			Description: "This is a test command!",
			Execute: func(res *types.CmdResponse) {
				fmt.Println("Hello world!")
			},
		},
	)

	app.Run()
}
