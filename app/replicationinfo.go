package main

import (
	"strings"
)

type ReplicationInfo struct {
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

func NewReplicationInfo(masterAddr string) ReplicationInfo {
	replicationInfo := ReplicationInfo{
		role: "master",
	}

	if masterAddr := strings.Split(masterAddr, " "); len(masterAddr) >= 2 {
		// masterPort, _ := strconv.Atoi(masterAddr[1])
		replicationInfo.role = "slave"
	}

	if replicationInfo.role == "master" {
		intSlice := RandByteSliceFromRanges(40, [][]int{{48, 57} /* {65, 90}, */, {97, 122}})
		replicationInfo.masterReplid = string(intSlice)
		replicationInfo.masterReplOffset = 0
	}

	return replicationInfo
}

/* func getReplicationInfo() {
	repInfo := ReplicationInfo{
		role: "master",
	}

	return
}
*/
