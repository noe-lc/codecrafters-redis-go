package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	port := flag.Int("port", DEFAULT_PORT, "port number on which the server will run")
	replicaOf := flag.String("replicaof", DEFAULT_MASTER, "address of the master server from which to create the replica")
	rdbFileDir := flag.String("dir", RDB_DEFAULT_DIR, "directory where the RDB file is located")
	rdbFileName := flag.String("dbfilename", RDB_DEFAULT_FILENAME, "name of the RDB file")

	flag.Parse()

	server, err := CreateRedisServer(*port, *replicaOf, *rdbFileDir, *rdbFileName)
	if err != nil {
		fmt.Println("Failed to create server: ", err)
		os.Exit(1)
	}

	err = server.Start()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
