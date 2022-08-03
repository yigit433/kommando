package types

import (
	"reflect"
	"strconv"
	"strings"
)

type CmdResponse struct {
	Command Command
	Args    map[string]interface{}
}

type Flag struct {
	Required    *bool
	Name        string
	Description string
	ValueType   string
}

type Command struct {
	Name        string
	Description string
	Flags       []Flag
	Aliases     []string
	Execute     func(res *CmdResponse)
}

func (c *Command) isValidAliase(aliase string) *bool {
	var output bool = false

	for _, alias := range c.Aliases {
		if alias == aliase {
			output = true

			break
		}
	}

	return &output
}

func (c *Command) isValidFlag(fname string, fvalue interface{}) *bool {
	var output bool = false

	for _, flag := range c.Flags {
		if flag.Name == fname {
			if flag.ValueType == "bool" {
				_, err := strconv.ParseBool(fvalue.(string))
				if err != nil {
					panic(err)
					break
				}

				output = true
			} else if flag.ValueType == "int" {
				_, err := strconv.ParseInt(fvalue.(string), 10, 64)
				if err != nil {
					panic(err)
					break
				}

				output = true
			} else if flag.ValueType == "float" {
				_, err := strconv.ParseFloat(fvalue.(string), 64)
				if err != nil {
					panic(err)
					break
				}

				output = true
			} else if reflect.TypeOf(fvalue).Name() == "string" {
				output = true
			}
		}
	}

	return &output
}

func (c *Command) argParser(args []string) map[string]interface{} {
	output := make(map[string]interface{})

	output["args"] = []string{}

	for ind, arg := range args {
		if strings.Contains(arg, "--") {
			vals := strings.Split(arg, "--")

			if strings.Contains(vals[1], "=") {
				parsed := strings.Split(vals[1], "=")

				if *c.isValidFlag(parsed[0], parsed[1]) {
					output[parsed[0]] = parsed[1]
				}
			} else if *c.isValidFlag(vals[1], args[ind+1]) {
				output[vals[1]] = args[ind+1]
			}
		} else if strings.Contains(arg, "-") {
			vals := strings.Split(arg, "-")

			if strings.Contains(vals[1], "=") {
				parsed := strings.Split(vals[1], "=")

				if *c.isValidFlag(parsed[0], parsed[1]) {
					output[parsed[0]] = parsed[1]
				}
			} else if *c.isValidFlag(vals[1], args[ind+1]) {
				output[vals[1]] = args[ind+1]
			}
		} else {
			if (ind - 1) >= 0 {
				cont1 := strings.Contains(args[ind-1], "--")
				cont2 := strings.Contains(args[ind-1], "-")

				if !cont1 || !cont2 || ((cont1 || cont2) && strings.Contains(args[ind-1], "=")) {
					args := output["args"].([]string)

					args = append(args, arg)

					output["args"] = args
				}
			} else {
				args := output["args"].([]string)

				args = append(args, arg)

				output["args"] = args
			}
		}
	}

	if len(output) >= 1 {
		for _, flags := range c.Flags {
			_, ok := output[flags.Name]

			if *flags.Required && !ok {
				panic("Required flag not specified!")
			}
		}
	}

	return output
}
