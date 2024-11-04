package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
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

type RedisSlaveServer struct {
	Role             string
	Host             string
	Port             int
	MasterPort       int
	Status           ServerStatus
	listener         net.Listener
	masterConnection net.Conn
	replicaInfo      ReplicaInfo
	rdbFile          []byte
	offset           int
	rdbConfig        map[string]string
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

func (r *RedisSlaveServer) Start() error {
	port := strconv.Itoa(r.Port)
	listener, err := net.Listen("tcp", r.Host+":"+port)
	if err != nil {
		fmt.Println("Error connecting to master server")
		return err
	}
	r.listener = listener

	conn, err := net.Dial("tcp", DEFAULT_HOST_ADDRESS+":"+strconv.Itoa(r.MasterPort))
	if err != nil {
		fmt.Println("Error connecting to master server")
		return err
	}
	r.masterConnection = conn

	masterConnReader := bufio.NewReader(conn)
	handshakeErrChannel := make(chan error)
	serverConnErrChannel := make(chan error)
	go func() {
		err := r.handshakeWithMaster(masterConnReader)
		handshakeErrChannel <- err
		err = HandleHandshakeConnection(r.masterConnection, r, masterConnReader)
		handshakeErrChannel <- err
	}()
	go func() {
		err := r.acceptConnections(r.listener)
		serverConnErrChannel <- err
	}()

	// r.handshakeWithMaster()

	for {
		select {
		case err := <-handshakeErrChannel:
			if err == io.EOF {
				fmt.Println("EOFF")
				continue
			}
			if err != nil {
				fmt.Println("Failed to execute handshake with master: ", err)
				return err
			}
		case err := <-serverConnErrChannel:
			if err != nil {
				fmt.Println("Error accepting connection: ", err)
				return err
			}
		}
	}
}

func (r *RedisSlaveServer) Stop() error {
	return r.listener.Close()
}

func (r *RedisSlaveServer) ReplicaInfo() ReplicaInfo {
	return r.replicaInfo
}

func (r *RedisSlaveServer) runCommandInternally(cmp CommandComponents) (string, bool, error) {
	var err error
	var result string
	var writeToMaster bool
	command, args := cmp.Command, cmp.Args

	switch command {
	case REPLCONF:
		argLen := len(cmp.Args)
		if argLen < 2 {
			break
		}
		arg1, arg2 := cmp.Args[0], cmp.Args[1]
		if arg1 == GETACK && arg2 == GETACK_FROM_REPLICA_ARG {
			writeToMaster = true
			result = ToRespBulkStringArray(REPLCONF, ACK, strconv.Itoa(r.offset))
		}
	default:
		result, err = RespCommands[command].Execute(args, r)
	}

	if err != nil {
		return "", writeToMaster, nil
	}

	return result, writeToMaster, nil
}

// Use for commands sent by a client which is NOT master
func (r RedisSlaveServer) RunCommand(cmp CommandComponents, conn net.Conn, t *Transaction) error {
	result, _, err := r.runCommandInternally(cmp)
	if err != nil {
		return err
	}
	_, err = conn.Write([]byte(result))
	if err != nil {
		return err
	}

	return nil
}

// Use for running commands sent by the master (handshake connection)
func (r *RedisSlaveServer) RunCommandSilently(cmp CommandComponents) error {
	result, writeToMaster, err := r.runCommandInternally(cmp)
	if err != nil {
		return err
	}
	fmt.Println(result, writeToMaster, err)
	if writeToMaster {
		_, err = r.masterConnection.Write([]byte(result))
		if err != nil {
			return err
		}
	}

	// seems like calling methods with pointer receivers which modify internal state should be called
	// by methods that have a pointer receiver as well
	r.updateProcessedBytes(len(cmp.Input))

	return nil
}

func (r *RedisSlaveServer) GetRDBConfig() map[string]string {
	return r.rdbConfig
}

func (r *RedisSlaveServer) GetStatus() *ServerStatus {
	return &r.Status
}

func (r *RedisSlaveServer) acceptConnections(l net.Listener) error {
	fmt.Println("Slave server listening on port", r.Port)
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go HandleConnection(conn, r)
	}
}

func (r *RedisSlaveServer) updateProcessedBytes(bytes int) {
	r.offset += bytes
	fmt.Println("processed bytes increased by ", bytes, "final: ", r.offset)
}

func (r *RedisSlaveServer) handshakeWithMaster(reader *bufio.Reader) error {
	// * 1 - PING
	_, err := r.masterConnection.Write([]byte(ToRespBulkStringArray(PING)))
	if err != nil {
		return err
	}

	pingResponseExpected := ToRespSimpleString(PONG)
	pingResponse, err := BufioRead(reader, pingResponseExpected)
	if err != nil {
		r.masterConnection.Close()
		fmt.Println("Failed to read response from master server")
		return err
	}
	if pingResponse != pingResponseExpected {
		r.masterConnection.Close()
		return fmt.Errorf("unexpected response to %s from master. Expected: %s Received: %s", PING, pingResponseExpected, pingResponse)
	}

	// * 2 - REPLCONF
	replConfResponseExpected := ToRespSimpleString(OK)
	replConf1 := REPLCONF + " " + "listening-port" + " " + strconv.Itoa(r.Port)
	replConf2 := REPLCONF + " " + "capa" + " " + "psync2"
	replConfList := []string{
		replConf1, replConf2,
	}

	for _, replConfMessage := range replConfList {
		messageItems := strings.Split(replConfMessage, " ")
		message := ToRespBulkStringArray(messageItems...)
		_, err = r.masterConnection.Write([]byte(message))
		if err != nil {
			return err
		}

		replConfResponse, err := BufioRead(reader, replConfResponseExpected)
		// TODO: abstract out these checks as they will be run several times
		if err != nil {
			r.masterConnection.Close()
			return err
		}
		if replConfResponse != replConfResponseExpected {
			r.masterConnection.Close()
			return fmt.Errorf("unexpected response to %s from master. Expected: %s Received: %s", REPLCONF, replConfResponseExpected, replConfResponse)
		}
	}

	// * 3 - PSYNC
	r.masterConnection.Write([]byte(ToRespBulkStringArray(PSYNC, "?", "-1")))
	psyncResponseExpected := BuildPsyncResponse(strings.Repeat("*", REPLICA_ID_LENGTH)) // Slaves have no visibility of master IDs on startup.
	psyncResponse, err := BufioRead(reader, psyncResponseExpected)
	if err != nil {
		r.masterConnection.Close()
		return err
	}
	if !strings.HasPrefix(psyncResponse, SIMPLE_STRING+FULLRESYNC) {
		r.masterConnection.Close()
		return fmt.Errorf("unexpected response to %s from master. Expected: %s Received: %s", PSYNC, psyncResponseExpected, psyncResponse)
	}

	// * 4 - RDB File
	fileLengthPrefix, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	fileLengthPrefix = strings.TrimRight(fileLengthPrefix, PROTOCOL_TERMINATOR)
	fileLength, err := strconv.Atoi(strings.Replace(fileLengthPrefix, BULK_STRING, "", -1))
	if err != nil {
		return err
	}

	rdbFile, err := BufioRead(reader, fileLength)
	if err != nil {
		return err
	}
	r.rdbFile = []byte(rdbFile)
	fmt.Println("Successfully executed handshake. Master ID: " + strings.Split(psyncResponse, " ")[1])
	return nil
}
