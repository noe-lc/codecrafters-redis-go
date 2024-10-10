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

type XReadBlock struct {
	key    string
	id     string
	status string
}

type RedisMasterServer struct {
	Role        string
	Host        string
	Port        int
	listener    net.Listener
	waitAckFor  *CommandHistoryItem
	ackChannel  chan bool
	replicas    []Replica
	replicaInfo ReplicaInfo
	history     CommandHistory
	rdbConfig   map[string]string
	xReadBlock  XReadBlock
}

// additional statuses for XREAD blocks
const (
	XREAD_FREE    = "FREE"
	XREAD_BLOCKED = "BLOCKED"
)

func NewMasterServer(port int, rdbDir, rdbFileName string) RedisMasterServer {
	server := RedisMasterServer{
		Role: MASTER,
		Host: DEFAULT_HOST,
		Port: port,
		replicaInfo: ReplicaInfo{
			role: MASTER,
		},
		rdbConfig: map[string]string{
			RDB_DIR_ARG:      rdbDir,
			RDB_FILENAME_ARG: rdbFileName,
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

	fmt.Println("command input", commandInput)

	// TODO: find a different way of avoiding circular references instead of using a pointer here
	r.history.Append(CommandHistoryItem{&respCommand, args, false, 0})

	// 1. command executors produce the output to write
	writeCommandOutput := func() error {
		// TODO: maybe do not reply on XREAD block
		result, err := respCommand.Execute(args, r)
		if err != nil {
			return err
		}
		_, err = conn.Write([]byte(result))
		if err != nil {
			return err
		}
		return nil
	}

	// 2. handle side effects internally
	switch command {
	case PSYNC:
		err := writeCommandOutput()
		if err != nil {
			return err
		}

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
			r.ackChannel <- true
			return nil
		}

		err := writeCommandOutput()
		if err != nil {
			return err
		}
	default:
		err := writeCommandOutput()
		if err != nil {
			return err
		}

		if respCommand.Type == WRITE {
			r.propagateCommand(commandInput)
		}
	}

	// modHistoryEntry.Success = true
	return nil
}

func (r *RedisMasterServer) SetAcknowledgeItem(historyItem *CommandHistoryItem, ackChan chan bool) {
	r.waitAckFor = historyItem
	r.ackChannel = ackChan
}

func (r *RedisMasterServer) GetRDBConfig() map[string]string {
	return r.rdbConfig
}

func (r *RedisMasterServer) GetXReadBlock() XReadBlock {
	return r.xReadBlock
}

func (r *RedisMasterServer) SetXReadBlock(key, id, status string) {
	r.xReadBlock.key = key
	r.xReadBlock.id = id
	r.xReadBlock.status = status
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
