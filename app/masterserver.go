package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Replica struct {
	conn net.Conn
}

type RedisMasterServer struct {
	Role string
	Host string
	Port int
	// ReadNext    bool
	waitAckFor  *CommandHistoryItem
	listener    net.Listener
	replicas    []Replica
	replicaInfo ReplicaInfo
	history     CommandHistory
}

func NewMasterServer(port int) RedisMasterServer {
	server := RedisMasterServer{
		Role: MASTER,
		Host: DEFAULT_HOST,
		Port: port,
		replicaInfo: ReplicaInfo{
			role: MASTER,
		},
	}

	return server
}

func (r *RedisMasterServer) Start() error {
	port := strconv.Itoa(r.Port)
	r.replicaInfo.masterReplOffset = 0
	r.replicaInfo.masterReplid = string(RandByteSliceFromRanges(40, [][]int{{48, 57}, {97, 122}}))
	listener, err := net.Listen("tcp", r.Host+":"+port)
	if err != nil {
		return err
	}
	r.listener = listener

	fmt.Println("Master server listening on port", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go HandleConnection(conn, r)
	}
}

func (r RedisMasterServer) ReplicaInfo() ReplicaInfo {
	return r.replicaInfo
}

func (r *RedisMasterServer) RunCommand(cmp CommandComponents, conn net.Conn) error {
	command, args, commandInput := cmp.Command, cmp.Args, cmp.Input
	respCommand := RespCommands[command]

	// TODO: find a different way of avoiding circular references instead of using a pointer here
	r.history.Append(CommandHistoryItem{&respCommand, args, false, 0})

	// 1. command executors produce the output to write
	result, err := respCommand.Execute(args, r)
	if err != nil {
		return err
	}
	_, err = conn.Write([]byte(result))
	if err != nil {
		return err
	}

	// 2. handle side effects internally
	switch command {
	case PSYNC:
		rdbFileBytes, err := hex.DecodeString(RDB_EMPTY_FILE_HEX)
		if err != nil {
			return err
		}
		rdbFileLength := BULK_STRING + strconv.Itoa(len(rdbFileBytes)) + PROTOCOL_TERMINATOR
		_, err = conn.Write([]byte(rdbFileLength))
		if err != nil {
			return err
		}
		_, err = conn.Write(rdbFileBytes)
		//fmt.Println("wrote", wrote)
		if err != nil {
			return err
		}

		r.replicas = append(r.replicas, Replica{conn})
	case REPLCONF:
		concatArgs := strings.Join(args, " ")

		// REPLCONF ACK <BYTES>
		if matches, _ := regexp.MatchString(ACK+` `+`\d+`, concatArgs); matches {
			r.waitAckFor.Acks += 1
			fmt.Println("cmd wait for", r.waitAckFor)
		}

		/* indexOfPortArg := slices.Index(args, LISTENING_PORT_ARG)
		indexOfPort := indexOfPortArg + 1
		if indexOfPortArg == -1 || len(args) <= indexOfPort {
			break
		}
		replicaPort := args[indexOfPort]
		conn, err := net.Dial("tcp", DEFAULT_HOST_ADDRESS+":"+replicaPort)
		if err != nil {
			fmt.Printf("Could not connect to replica on port %s\n", replicaPort)
			return err
		}
		r.replicaConnections = append(r.replicaConnections, conn) */
	default:
		if respCommand.Type == WRITE {
			go r.propagateCommand(commandInput /* modHistoryEntry */)
		}
	}

	// modHistoryEntry.Success = true
	return nil
}

func (r *RedisMasterServer) isReplicaConnection(addr string) bool {
	for _, replica := range r.replicas {
		if addr == replica.conn.RemoteAddr().String() {
			return true
		}
	}
	return false
}

func (r *RedisMasterServer) propagateCommand(rawInput string /* historyItem *CommandHistoryItem */) []error {
	errors := []error{}
	for _, replica := range r.replicas {
		fmt.Println("Propagating command to: ", replica.conn.RemoteAddr().String())
		_, err := replica.conn.Write([]byte(rawInput))
		if err != nil {
			fmt.Println("error propagating command: ", err)
			errors = append(errors, err)
			continue
		}
		// historyItem.Acks += 1
	}
	return errors
}
