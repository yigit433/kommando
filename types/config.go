package types

import (
	"fmt"
	"os"
	"strings"
)

const (
	MAIN_TEMPLATE string = "Welcome to {AppName}! That's a command list. Type 'help <command name>' to get help with any command.\n{CmdList}"
	CMD_LIST      string = "{CmdName} |> {CmdDescription}"
	CMD_HELP      string = "{CmdName} | Info\n{CmdDescription}\n{CmdAliases}"
)

type Config struct {
	AppName  string
	commands []Command
}

func (c *Config) AddCommand(cmd *Command) {
	c.commands = append(c.commands, *cmd)
}

func (c *Config) Run() {
	args := os.Args[1:]

	c.commands = append(c.commands, Command{
		Name:        "help",
		Description: "Basic helper command where you can get information about commands.",
		Execute: func(res *CmdResponse) {
			c.createCommandList()
		},
	})

	if len(args) == 0 {
		c.createCommandList()

		return
	}

	for _, cmd := range c.commands {
		if cmd.Name == args[0] {
			cmd.Execute(&CmdResponse{
				Command: cmd,
				Args:    args,
			})
			break
		}
	}
}

func (c *Config) createCommandList() {
	var cmds []string

	for _, cmd := range c.commands {
		var command string = strings.Replace(CMD_LIST, "{CmdName}", cmd.Name, -1)
		command = strings.Replace(command, "{CmdDescription}", cmd.Description, -1)

		cmds = append(cmds, command)
	}

	var logmsg string = strings.Replace(MAIN_TEMPLATE, "{AppName}", c.AppName, -1)
	logmsg = strings.Replace(logmsg, "{CmdList}", strings.Join(cmds, "\n"), -1)

	fmt.Println(logmsg)
}
