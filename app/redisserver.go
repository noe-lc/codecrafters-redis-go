package main

import (
	"net"
)

// Default hosts and addresses
const (
	DEFAULT_HOST         = "localhost"
	DEFAULT_PORT         = 6379
	DEFAULT_MASTER       = ""
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

type ServerStatus struct {
	XReadBlock chan bool
	Multi      bool
	execQueue  [](func() error)
}

type RedisServer interface {
	Start() error
	ReplicaInfo() ReplicaInfo
	RunCommand(cmp CommandComponents, conn net.Conn, trx *Transaction) error
	GetRDBConfig() map[string]string
	GetStatus() *ServerStatus
}

func CreateRedisServer(port int, replicaOf string, rdbDir, rdbFileName string) (RedisServer, error) {
	if replicaOf != "" {
		server, err := NewSlaveServer(port, replicaOf)
		if err != nil {
			return nil, err
		}
		return &server, nil
	}

	server := NewMasterServer(port, rdbDir, rdbFileName)
	return &server, nil
}
