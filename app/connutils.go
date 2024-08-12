package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

func HandleConnection(conn net.Conn, server RedisServer) {
	defer conn.Close()

	fmt.Println("Client connected: ", conn.RemoteAddr())

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
			fmt.Println(err)
			_, err = conn.Write([]byte(err.Error()))
			if err != nil {
				fmt.Println("Error writing:", err)
				return
			}

			continue
		}
		if ready {
			commandComponents := respProcessor.GetCommandComponents()
			err := server.RunCommand(commandComponents, conn)
			if err != nil {
				fmt.Printf("Error executing command %s in %s. Error: %s", commandComponents.Command, server.ReplicaInfo().role, err.Error())
			}
			respProcessor.Reset()
		}
	}
}

func ReadStringFromConn(readStr string, c net.Conn) (string, error) {
	responseBytes := make([]byte, len(readStr))
	readBytes, err := bufio.NewReader(c).Read(responseBytes)

	if err != nil {
		return "", err
	}

	return string(responseBytes[:readBytes]), nil
}
