package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

type BytesReadable interface {
	int | string
}

func HandleConnection(conn net.Conn, server RedisServer) {
	defer conn.Close()

	fmt.Printf("Client connected to %s. Remote addr: %s\n", server.ReplicaInfo().role, conn.RemoteAddr())
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
			fmt.Println("RESP Processor read error: ", err)
			respProcessor.Reset()

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
				fmt.Printf("Error executing command %s in %s. Error: %s\n", commandComponents.Command, server.ReplicaInfo().role, err.Error())
			}
			respProcessor.Reset()
		}
	}
}

func HandleHandshakeConnection(conn net.Conn, server RedisServer) error {
	fmt.Printf("Master connected. Remote addr: %s\n", conn.RemoteAddr())

	defer conn.Close()
	respProcessor := NewRESPMessageReader()
	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err == io.EOF {
			fmt.Println("Connection terminated by master")
			return err
		}
		if err != nil {
			fmt.Println("Error reading handshake message: ", err)
			return err
		}
		ready, err := respProcessor.Read(message)
		if err != nil {
			LogServerError(server, "RESP Processor read error", err)
			respProcessor.Reset()
			// TODO: do not ignore actual errors.
			// this has been commented out so that master replies during handshake
			// do not generate unexpected writes to slave replicas
			/*
				_, err = conn.Write([]byte(err.Error()))
				if err != nil {
					fmt.Println("Error writing:", err)
					return
				}
			*/

			continue
		}

		if ready {
			commandComponents := respProcessor.GetCommandComponents()
			err := server.RunCommand(commandComponents, conn)
			if err != nil {
				fmt.Printf("Error executing command %s in %s. Error: %s\n", commandComponents.Command, server.ReplicaInfo().role, err.Error())
			}
			respProcessor.Reset()
		}
	}
}

func BufioRead[T BytesReadable](reader *bufio.Reader, readable T) (string, error) {
	var lenBytes int
	input := any(readable)
	if str, ok := input.(string); ok {
		lenBytes = len(str)
	} else {
		lenBytes = input.(int)
	}

	buf := make([]byte, lenBytes)
	readBytes, err := reader.Read(buf)

	if err != nil {
		return "", err
	}

	return string(buf[:readBytes]), nil
}
