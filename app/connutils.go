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
		//fmt.Printf("processor %#v \n", respProcessor)

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

// 	The PSYNC command is used to synchronize the state of the replica with the master. The replica will send this command to the master with two arguments:
//
// The first argument is the replication ID of the master
// Since this is the first time the replica is connecting to the master, the replication ID will be ? (a question mark)
// The second argument is the offset of the master
// Since this is the first time the replica is connecting to the master, the offset will be -1
// So the final command sent will be PSYNC ? -1.
// The master will respond with a Simple string that looks like this:
//
// +FULLRESYNC <REPL_ID> 0\r\n
