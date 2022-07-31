## Kommando [![Go Report Card](https://goreportcard.com/badge/github.com/yigit433/kommando)](https://goreportcard.com/report/github.com/yigit433/kommando)
Simple and usable cli tool for go lang.
### Installation
`go get github.com/yigit433/kommando`
### Example
```go
package main

import (
    "fmt"
    "github.com/yigit433/kommando"
    "github.com/yigit433/kommando/types"
)

func main() {
    handler := kommando.NewKommando(types.Config{
        AppName: "Kommando Test App",
    })

    handler.AddCommand(
        &types.Command{
            Name: "test",
            Description: "Hello world test example!",
            Execute: func(res *types.CmdResponse) {
                fmt.Println("Hello world!")
            },
        },
    )
}
```
