package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

type messageOutput[T any, U any] func(T) (U, error)
type message[T any, U any] struct {
	input  string
	output messageOutput[T, U]
}

// type messageProcessors[T any, U any] map[string]message[T, U]

var pingMessage message[string, string] = message[string, string]{
	input: "PING",
	output: func(i string) (string, error) {
		return "+PONG\r\n", nil
	},
}

func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")

	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()

		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Client connected")

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Println("Client disconnected")
			return
		}

		message = strings.TrimRight(message, "\n")
		returnMessage := ""

		switch message {
		case "PING":
			returnMessage, _ = pingMessage.output("")
		default:
			returnMessage, _ = pingMessage.output("")
			// returnMessage = "Message received: " + message
		}

		conn.Write([]byte(returnMessage))
	}
}
