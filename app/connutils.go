package main

import (
	"bufio"
	"net"
)

func ReadStringFromConn(readStr string, c net.Conn) (string, error) {
	responseBytes := make([]byte, len(readStr))
	readBytes, err := bufio.NewReader(c).Read(responseBytes)

	if err != nil {
		return "", err
	}

	return string(responseBytes[:readBytes]), nil
}

// 	The PSYNC command is used to synchronize the state of the replica with the master. The replica will send this command to the master with two arguments:
//
// The first argument is the replication ID of the master
// Since this is the first time the replica is connecting to the master, the replication ID will be ? (a question mark)
// The second argument is the offset of the master
// Since this is the first time the replica is connecting to the master, the offset will be -1
// So the final command sent will be PSYNC ? -1.
// The master will respond with a Simple string that looks like this:
//
// +FULLRESYNC <REPL_ID> 0\r\n
