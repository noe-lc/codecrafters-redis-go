package main

import "net"

// Default hosts and addresses
const (
	DEFAULT_HOST         = "localhost"
	DEFAULT_PORT         = 6379
	DEFAULT_HOST_ADDRESS = "127.0.0.1"
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

type RedisServer interface {
	Start() (net.Listener, error)
	ReplicaInfo() ReplicaInfo
	RunCommand(cmp CommandComponents, conn net.Conn) error
}

func CreateRedisServer(port int, replicaOf string) (RedisServer, error) {
	if replicaOf != "" {
		server, err := NewSlaveServer(port, replicaOf)
		if err != nil {
			return nil, err
		}
		return &server, nil
	}

	server := NewMasterServer(port)

	return &server, nil
}
