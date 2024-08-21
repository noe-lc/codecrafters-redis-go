package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
)

type RedisMasterServer struct {
	Role               string
	Host               string
	Port               int
	listener           net.Listener
	replicaConnections []net.Conn
	replicaInfo        ReplicaInfo
	replicaAcks        int
	history            CommandHistory
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
	executor := CommandExecutors[command]

	r.history.Append(CommandHistoryItem{command, args, false, 0})
	modHistoryEntry := r.history.GetModifiableEntry(len(r.history))

	// 1. command executors produce the output to write
	result, err := executor.Execute(args, r)
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

		r.replicaConnections = append(r.replicaConnections, conn)
	case REPLCONF:
		break
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
		if executor.Type == WRITE {
			go r.propagateCommand(commandInput, modHistoryEntry)
		}
	}

	modHistoryEntry.success = true
	return nil
}

func (r *RedisMasterServer) propagateCommand(rawInput string, historyItem *CommandHistoryItem) []error {
	errors := []error{}
	for _, conn := range r.replicaConnections {
		fmt.Println("Propagating command to: ", conn.RemoteAddr().String())
		_, err := conn.Write([]byte(rawInput))
		if err != nil {
			fmt.Println("error propagating command: ", err)
			errors = append(errors, err)
		}
		historyItem.acks += 1
	}
	return errors
}
