package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type RedisSlaveServer struct {
	Role        string
	Host        string
	Port        int
	MasterPort  int
	listener    net.Listener
	connection  net.Conn
	replicaInfo ReplicaInfo
}

func NewSlaveServer(port int, replicaOf string) (RedisSlaveServer, error) {
	MasterPort := DEFAULT_PORT
	replicaOfParts := strings.Split(replicaOf, " ")

	if len(replicaOfParts) >= 2 {
		port, err := strconv.Atoi(replicaOfParts[1])
		if err != nil {
			return RedisSlaveServer{}, errors.New("could not create slave. Invalid replicaof argument")
		}
		MasterPort = port
	}

	server := RedisSlaveServer{
		Role:       SLAVE,
		Host:       DEFAULT_HOST,
		Port:       port,
		MasterPort: MasterPort,
		replicaInfo: ReplicaInfo{
			role: SLAVE,
		},
	}

	return server, nil
}

func (r *RedisSlaveServer) Start() (net.Listener, error) {
	listener, err := net.Listen("tcp", r.Host+":"+strconv.Itoa(r.Port))
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial("tcp", DEFAULT_HOST_ADDRESS+":"+strconv.Itoa(r.MasterPort))
	if err != nil {
		fmt.Println("Error connecting to master server")
		return nil, err
	}

	conn.Write([]byte(ToRespArrayString(PING)))

	r.listener = listener
	r.connection = conn

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
	replConf1 := REPLCONF + " " + "listening-port" + " " + strconv.Itoa(r.Port)
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
	// We pass a string of equal length to be able to read the whole response. Slaves have no visibility of the master id in start.
	pSyncResponse := ToRespSimpleString(buildPsyncResponse(strings.Repeat("*", REPLICA_ID_LENGTH)))
	responseString, err = ReadStringFromConn(pSyncResponse, conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if !strings.HasPrefix(responseString, SIMPLE_STRING+FULLRESYNC) {
		conn.Close()
		return nil, errors.New("unexpected response to " + PSYNC + " from master server: `" + responseString + "`")
	}

	// conn.Close()
	masterReplId := strings.Split(responseString, " ")[1]
	fmt.Println("Successfully replicated master " + masterReplId)
	return listener, nil
}

func (r *RedisSlaveServer) Stop() error {
	return r.listener.Close()
}

func (r *RedisSlaveServer) ReplicaInfo() ReplicaInfo {
	return r.replicaInfo
}

func (r RedisSlaveServer) RunCommand(cmp CommandComponents, conn net.Conn) error {
	result, err := CommandExecutors[cmp.Command].Execute(cmp.Args, &r, conn)
	if err != nil {
		return err
	}
	_, err = conn.Write([]byte(result))
	if err != nil {
		return err
	}

	return nil
}
