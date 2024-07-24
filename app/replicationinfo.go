package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	DEFAULT_HOST         = "localhost"
	DEFAULT_PORT         = 6379
	DEFAULT_HOST_ADDRESS = "0.0.0.0"
)

type ReplicaInfo struct {
	role string
	// connectedSlaves            int
	masterReplid     string
	masterReplOffset int
	// secondReplOffset           int
	// replBacklogActive          int
	// replBacklogSize            int
	// replBacklogFirstByteOffset int
	// replBacklogHistlen         any
}

type RedisServer struct {
	host        string
	port        int
	masterPort  int
	replicaInfo ReplicaInfo
}

func NewServer(port int, replicaOf string) RedisServer {
	replicaOfParts := strings.Split(replicaOf, " ")
	server := RedisServer{
		host:       DEFAULT_HOST,
		port:       port,
		masterPort: DEFAULT_PORT,
	}
	replicationInfo := ReplicaInfo{
		role: "master",
	}

	if len(replicaOfParts) >= 2 {
		masterPort, _ := strconv.Atoi(replicaOfParts[1])
		server.masterPort = masterPort
		replicationInfo.role = "slave"
	}

	server.replicaInfo = replicationInfo
	return server
}

func (s *RedisServer) Start() (net.Listener, error) {
	if s.replicaInfo.role == "master" {
		s.replicaInfo.masterReplid = string(RandByteSliceFromRanges(40, [][]int{{48, 57}, {97, 122}}))
		s.replicaInfo.masterReplOffset = 0
		return net.Listen("tcp", s.host+":"+strconv.Itoa(s.port))
	}

	conn, err := net.Dial("tcp", DEFAULT_HOST_ADDRESS+":"+strconv.Itoa(s.masterPort))

	if err != nil {
		fmt.Println("Error connecting to master server: ", err.Error())
		return nil, errors.New("error connecting to master server")
	}

	conn.Write([]byte(ToRespArrayString(PING)))
	okResponse := ToRespSimpleString(PONG)
	responseBytes := make([]byte, len(okResponse))
	readBytes, err := bufio.NewReader(conn).Read(responseBytes)
	responseString := string(responseBytes[:readBytes])

	if err != nil {
		conn.Close()
		fmt.Println("Failed to read response from master server: ", err.Error())
		return nil, errors.New("failed to read response from master server")
	}

	if responseString != okResponse {
		conn.Close()
		return nil, errors.New("unexpected response to " + PING + " from master server: " + responseString)
	}

	// TODO: find a way to make the replica server maintain the connection
	conn.Close()
	return net.Listen("tcp", s.host+":"+strconv.Itoa(s.port))
}

/*
conn, err := net.Dial("tcp", "localhost:"+ServerPort)

	if err != nil {
		fmt.Println("Error connecting to server")
	}

	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("Enter a message:")
		message, _ := reader.ReadString('\n')
		_, err := conn.Write([]byte(message))

		if err != nil {
			fmt.Println("Error writing to server:", err)
			return
		}

		responseBytes := make([]byte, 1024)
		readBytes, err := bufio.NewReader(conn).Read(responseBytes)

		if err != nil {
			fmt.Println("Error reading from server", err)
			return
		}

		os.Stdout.Write(responseBytes[:readBytes])
		os.Stdout.Write([]byte{'\n'})
	}

*/

/* func getReplicationInfo() {
	repInfo := ReplicaInfo{
		role: "master",
	}

	return
}
*/
