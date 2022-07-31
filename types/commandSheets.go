package types

type Command struct {
	Name        string
	Description string
	Execute     func(res *CmdResponse)
}

type CmdResponse struct {
	Command Command
	Args    []string
}
