package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
)

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

	respProcessor := NewRESPMessageReader()
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by client")
				break
			}

			fmt.Println("Error reading: " + err.Error())
			return
		}

		ready, err := respProcessor.Read(message)

		if err != nil {
			_, err = conn.Write([]byte(err.Error()))
			if err != nil {
				fmt.Println("Error writing:", err)
				return
			}
		}

		if ready {
			// TODO: implement a command runner
			command, args := respProcessor.GetCommandAndArgs()
			result := ExecuteCommand(command, args)

			fmt.Println("rmp command: ", command)
			fmt.Println("rmp args: ", args)
			fmt.Printf("processor: \n %+v\n", respProcessor)
			fmt.Println("result: ", result)
			fmt.Println([]byte(result))
			fmt.Println("------------------")

			_, err = conn.Write([]byte(result))
			if err != nil {
				fmt.Println("Error writing:", err)
			}
			respProcessor.Reset()
		}

		/* trimmedMessage := strings.TrimSuffix(message, "\r\n")
		fmt.Printf("Received: %s\n", trimmedMessage)

		// Echo the message back to the client
		response := trimmedMessage + "\r\n"
		_, err = conn.Write([]byte(response))
		if err != nil {
			fmt.Println("Error writing:", err)
			return
		} */

	}
}
