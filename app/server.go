package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

func main() {
	defaultMaster := ""
	port := flag.Int("port", DEFAULT_PORT, "port number on which the server will run")
	replicaOf := flag.String("replicaof", defaultMaster, "address of the master server from which to create the replica")

	flag.Parse()

	server, err := CreateRedisServer(*port, *replicaOf)
	if err != nil {
		fmt.Println("Faile to create server: ", err)
		os.Exit(1)
	}
	l, err := server.Start()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Sucessfully started server, will listen on port ", *port)

	for {
		conn, err := l.Accept()

		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go HandleConnection(conn, server)
	}
}

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
