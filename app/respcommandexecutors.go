package main

func Ping() string {
	return encodeSimpleString("PONG")
}

func Echo(args []string) string {
	if len(args) == 0 {
		return encodeBulkString("")
	}

	return encodeBulkString(args[0])
}

func ExecuteCommand(command string, args []string) string {
	switch command {
	case "PING":
		return Ping()
	case "ECHO":
		return Echo(args)
	default:
		return "command execution not supported"
	}
}
