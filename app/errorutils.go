package main

import "fmt"

func LogServerError(s RedisServer, prefix string, err error) error {
	fmt.Printf("%s: %v\n", s.ReplicaInfo().role, err)
	return fmt.Errorf("%s - %s: %v", s.ReplicaInfo().role, prefix, err)
}
