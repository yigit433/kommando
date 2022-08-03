package types

import (
	"fmt"
	"os"
	"strings"
)

const (
	MAIN_TEMPLATE string = "Welcome to {AppName}! That's a command list. Type 'help <command name>' to get help with any command.\n{CmdList}"
	CMD_LIST      string = "{CmdName} |> {CmdDescription}"
	CMD_HELP      string = "{CmdName} | Info\nDescription |> {CmdDescription}\nFlags |> {CmdFlags}\nAliases |> {CmdAliases}"
)

type Config struct {
	AppName  string
	commands []Command
}

func (c *Config) AddCommand(cmd *Command) {
	if len(c.commands) == 0 {
		c.commands = append(c.commands, *cmd)
	} else {
		for i, command := range c.commands {
			if command.Name == cmd.Name {
				panic("There is a command with the name you are trying to add.")
				break
			} else if i == len(c.commands)-1 {
				c.commands = append(c.commands, *cmd)
			}
		}
	}
}

func (c *Config) Run() {
	args := os.Args[1:]

	c.commands = append(c.commands, Command{
		Name:        "help",
		Description: "Basic helper command where you can get information about commands.",
		Execute: func(res *CmdResponse) {
			args := res.Args["args"].([]string)

			if len(args) > 0 {
				cname := args[0]

				for i, cmd := range c.commands {
					if cmd.Name == cname {
						message := strings.Replace(CMD_HELP, "{CmdName}", cname, -1)
						message = strings.Replace(message, "{CmdDescription}", cmd.Description, -1)

						flags := []string{}

						for _, flag := range cmd.Flags {
							flags = append(flags, fmt.Sprintf("--%s", flag.Name))
						}

						message = strings.Replace(message, "{CmdFlags}", strings.Join(flags[:], ", "), -1)
						message = strings.Replace(message, "{CmdAliases}", strings.Join(cmd.Aliases[:], ", "), -1)

						fmt.Println(message)
						break
					} else if i == len(c.commands)-1 {
						c.createCommandList()
					}
				}
			} else {
				c.createCommandList()
			}
		},
	})

	if len(args) == 0 {
		c.createCommandList()

		return
	}

	for i, cmd := range c.commands {
		if cmd.Name == args[0] || *cmd.isValidAliase(args[0]) {
			cmd.Execute(&CmdResponse{
				Command: cmd,
				Args:    cmd.argParser(args[1:]),
			})
			break
		} else if i == len(c.commands)-1 {
			c.createCommandList()
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
