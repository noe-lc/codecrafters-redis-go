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
	Role        string
	Host        string
	Port        int
	Status      ServerStatus
	listener    net.Listener
	waitAckFor  *CommandHistoryItem
	ackChannel  chan bool
	replicas    []Replica
	replicaInfo ReplicaInfo
	history     CommandHistory
	rdbConfig   map[string]string
}

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

func (r *RedisMasterServer) RunCommand(cmp CommandComponents, conn net.Conn, t *Transaction) error {
	command, args, commandInput := cmp.Command, cmp.Args, cmp.Input
	respCommand := RespCommands[command]

	r.history.Append(CommandHistoryItem{&respCommand, args, false, 0})

	// 1. command executors produce the output to write
	writeCommandOutput := func() error {
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
		if err != nil {
			return err
		}

		r.replicas = append(r.replicas, Replica{conn})
	case REPLCONF:
		concatArgs := strings.Join(args, " ")
		if matches, _ := regexp.MatchString(ACK+` `+`\d+`, concatArgs); matches {
			r.waitAckFor.Acks += 1
			r.ackChannel <- true
			return nil
		}

		err := writeCommandOutput()
		if err != nil {
			return err
		}
	case MULTI:
		t.Conn = conn
		err := writeCommandOutput()
		if err != nil {
			t.Reset()
			return err
		}
	case EXEC:
		result := ""

		if t.Conn == nil {
			result = ToRespError(fmt.Errorf("%s without %s", EXEC, MULTI))
		} else {
			result = t.ExecTransaction(r)
		}

		_, err := conn.Write([]byte(result))
		if err != nil {
			return err
		}
		return nil
	case DISCARD:
		result := ""

		if t.Conn == nil {
			result = ToRespError(fmt.Errorf("%s without %s", DISCARD, MULTI))
		} else {
			t.Reset()
			result, _ = respCommand.Execute(args, r)
		}

		_, err := conn.Write([]byte(result))
		if err != nil {
			return err
		}
		return nil

	default:
		if t.Conn != nil {
			t.EnqueueCommand(cmp)
			_, err := conn.Write([]byte(ToRespSimpleString(QUEUED)))
			if err != nil {
				return err
			}

			return nil
		}

		err := writeCommandOutput()
		if err != nil {
			return err
		}

		if respCommand.Type == WRITE {
			r.propagateCommand(commandInput)
		}
	}

	return nil
}

func (r *RedisMasterServer) SetAcknowledgeItem(historyItem *CommandHistoryItem, ackChan chan bool) {
	r.waitAckFor = historyItem
	r.ackChannel = ackChan
}

func (r *RedisMasterServer) GetRDBConfig() map[string]string {
	return r.rdbConfig
}

func (r *RedisMasterServer) GetStatus() *ServerStatus {
	return &r.Status
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
