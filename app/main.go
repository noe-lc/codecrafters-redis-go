package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	defaultMaster := ""
	port := flag.Int("port", DEFAULT_PORT, "port number on which the server will run")
	replicaOf := flag.String("replicaof", defaultMaster, "address of the master server from which to create the replica")

	flag.Parse()

	server, err := CreateRedisServer(*port, *replicaOf)
	if err != nil {
		fmt.Println("Failed to create server: ", err)
		os.Exit(1)
	}
	err = server.Start()
	// TODO: maybe use a channel, but there should be cleaner ways to achieve this
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
