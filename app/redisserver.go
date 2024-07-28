package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Default hosts and addresses
const (
	DEFAULT_HOST         = "localhost"
	DEFAULT_PORT         = 6379
	DEFAULT_HOST_ADDRESS = "0.0.0.0"
)

// Constants for server struct fields
const (
	MASTER = "master"
	SLAVE  = "slave"
)

// Constants for server helpers and utils
const (
	REPLICA_ID_LENGTH = 40
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
		role: MASTER,
	}

	if len(replicaOfParts) >= 2 {
		masterPort, _ := strconv.Atoi(replicaOfParts[1])
		server.masterPort = masterPort
		replicationInfo.role = SLAVE
	}

	server.replicaInfo = replicationInfo
	return server
}

func (s *RedisServer) Start() (net.Listener, error) {
	if s.replicaInfo.role == MASTER {
		return s.startMaster()
	}

	if s.replicaInfo.role == SLAVE {
		return s.startSlave()
	}

	return nil, errors.New("replica role not set")
}

func (s *RedisServer) startMaster() (net.Listener, error) {
	s.replicaInfo.masterReplid = string(RandByteSliceFromRanges(40, [][]int{{48, 57}, {97, 122}}))
	s.replicaInfo.masterReplOffset = 0
	return net.Listen("tcp", s.host+":"+strconv.Itoa(s.port))
}

func (s *RedisServer) startSlave() (net.Listener, error) {
	conn, err := net.Dial("tcp", DEFAULT_HOST_ADDRESS+":"+strconv.Itoa(s.masterPort))

	if err != nil {
		fmt.Println("Error connecting to master server: ", err.Error())
		return nil, errors.New("error connecting to master server")
	}

	conn.Write([]byte(ToRespArrayString(PING)))
	pingResponse := ToRespSimpleString(PONG)
	responseString, err := ReadStringFromConn(pingResponse, conn)

	if err != nil {
		conn.Close()
		fmt.Println("Failed to read response from master server: ", err.Error())
		return nil, errors.New("failed to read response from master server")
	}

	if responseString != pingResponse {
		conn.Close()
		return nil, errors.New("unexpected response to " + PING + " from master server: " + responseString)
	}

	okResponse := ToRespSimpleString(OK)
	replConf1 := REPLCONF + " " + "listening-port" + " " + strconv.Itoa(s.port)
	replConf2 := REPLCONF + " " + "capa" + " " + "psync2"
	replConfList := []string{
		replConf1, replConf2,
	}
	for _, replConfMessage := range replConfList {
		messageItems := strings.Split(replConfMessage, " ")
		conn.Write([]byte(ToRespArrayString(messageItems...)))
		responseString, err = ReadStringFromConn(okResponse, conn)
		// TODO: abstract out these checks as they will be run several times
		if err != nil {
			conn.Close()
			return nil, err
		}
		if responseString != okResponse {
			conn.Close()
			return nil, errors.New("unexpected response to " + REPLCONF + " from master server: `" + responseString + "`")
		}

		fmt.Println(responseString)
	}

	conn.Write([]byte(ToRespArrayString(PSYNC, "?", "-1")))
	// Slaves have no visibility of the master id in start.
	// We pass a string of equal length to be able to read the whole response
	pSyncResponse := ToRespSimpleString(buildPsyncResponse(strings.Repeat("*", REPLICA_ID_LENGTH)))
	responseString, err = ReadStringFromConn(pSyncResponse, conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if len(responseString) != len(pSyncResponse) {
		conn.Close()
		return nil, errors.New("unexpected response to " + PSYNC + " from master server: `" + responseString + "`")
	}

	masterReplId := strings.Split(responseString, " ")[1]
	fmt.Println("Successfully replicated master " + masterReplId)

	// TODO: find a way to make the replica server maintain the connection and close it when needed
	conn.Close()
	return net.Listen("tcp", s.host+":"+strconv.Itoa(s.port))
}

/* func createIdForReplica() string {
	return string(RandByteSliceFromRanges(40, [][]int{{48, 57}, {97, 122}}))
} */

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
