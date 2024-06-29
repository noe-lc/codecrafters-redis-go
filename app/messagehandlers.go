package main

type MessageHandler func(string) string

func noopMessageHandler(message string) string {
	return "resp message handler: no handler is specified"
}

func echoMessageHandler(message string) string {
	return ""
}

func pingMessageHandler(message string) string {
	return ""
}

// Raw handlers
func rawNoopMessageHandler(message string) string {
	return "resp message handler: no handler is specified"
}

func rawEchoMessageHandler(message string) string {
	return message
}

func rawPingMessageHandler(message string) string {
	return encodeSimpleString("PONG")
}

/* type MessageHandler[T any, U any] struct {
	Command string
	Output  func(T) (U, string)
}

var Ping = MessageHandler[string, string]{
	Command: "PING",
	Output: func(s string) (string, string) {
		return encodeSimpleString("PONG"), nil
	},
}

var Echo = MessageHandler[[]string, string]{
	Command: "ECHO",
	Output: func(args []string) (string, string) {
		return encodeBulkString(strings.Join(args, " ")), nil
	},
}
*/
