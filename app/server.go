package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
)

const (
	DEFAULT_PORT = 6379
)

func main() {
	defaultMaster := ""
	port := flag.Int("port", DEFAULT_PORT, "port number on which the server will run")
	masterAddr := flag.String("replicaof", defaultMaster, "address of the master server from which to create the replica")

	flag.Parse()

	serverCtx := NewReplicationInfo(*masterAddr)
	l, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(*port))

	if err != nil {
		fmt.Println("Failed to bind to port ", *port)
		os.Exit(1)
	}

	fmt.Println("Logs from your program will appear here!")

	for {
		conn, err := l.Accept()

		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn, serverCtx)
	}
}

func handleConnection(conn net.Conn, replicationInfo ReplicationInfo) {
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
		//fmt.Printf("processor %#v \n", respProcessor)

		if err != nil {
			_, err = conn.Write([]byte(err.Error()))
			if err != nil {
				fmt.Println("Error writing:", err)
				return
			}

			continue
		}

		if ready {
			command, args := respProcessor.GetCommandAndArgs()
			result, err := CommandExecutors[command].Execute(args, replicationInfo)
			if err != nil {
				fmt.Println("Error executing command:", err)
			}

			_, err = conn.Write([]byte(result))
			if err != nil {
				fmt.Println("Error writing:", err)
			}

			respProcessor.Reset()
		}
	}
}
