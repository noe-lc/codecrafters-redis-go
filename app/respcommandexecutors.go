package main

type CommandExecutor struct {
	argLen  int
	Execute func([]string) (string, error)
}

func (c *CommandExecutor) GetArgLen() int {
	return c.argLen
}

var Memory = map[string]string{}

var (
	Ping = CommandExecutor{
		argLen: 1,
		Execute: func(args []string) (string, error) {
			return encodeSimpleString("PONG"), nil
		},
	}
	Echo = CommandExecutor{
		argLen: 2,
		Execute: func(args []string) (string, error) {
			if len(args) == 0 {
				return encodeBulkString(""), nil
			}
			return encodeBulkString(args[0]), nil
		},
	}
	Set = CommandExecutor{
		argLen: 3,
		Execute: func(args []string) (string, error) {
			key, value := args[0], args[1]
			Memory[key] = value
			return encodeSimpleString("OK"), nil
		},
	}
	Get = CommandExecutor{
		argLen: 2,
		Execute: func(args []string) (string, error) {
			value, exists := Memory[args[0]]

			if !exists {
				return NULL_BULK_STRING, nil
			}

			return encodeBulkString(value), nil
		},
	}
)

var CommandExecutors = map[string]CommandExecutor{
	"PING": Ping,
	"ECHO": Echo,
	"GET":  Get,
	"SET":  Set,
}

func IsRESPCommandSupported(command string) bool {
	_, exists := CommandExecutors[command]
	return exists
}

/* func ExecuteCommand(command string, args []string) string {
	commandExecutor, exists := CommandExecutors[command]

	if()

} */
