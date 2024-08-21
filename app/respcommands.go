package main

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Supported commands
const (
	PING     = "PING"
	ECHO     = "ECHO"
	INFO     = "INFO"
	SET      = "SET"
	GET      = "GET"
	REPLCONF = "REPLCONF"
	PSYNC    = "PSYNC"
	WAIT     = "WAIT"
)

// Command types
const (
	READ  = "READ"
	WRITE = "WRITE"
)

type RespCommand struct {
	argLen int
	// signature string
	Type    string
	Execute func([]string, RedisServer) (string, error)
}

type CommandComponents struct {
	// the full raw string from which command and args is derived
	Input string
	// the RESP command
	Command string
	// the RESP command arguments
	Args []string
}

func (c *RespCommand) GetArgLen() int {
	return c.argLen
}

var (
	Ping = RespCommand{
		argLen: 1,
		Execute: func(args []string, server RedisServer) (string, error) {
			return ToRespSimpleString("PONG"), nil
		},
	}
	Echo = RespCommand{
		argLen: 2,
		Execute: func(args []string, server RedisServer) (string, error) {
			if len(args) == 0 {
				return ToRespBulkString(""), nil
			}
			return ToRespBulkString(args[0]), nil
		},
	}
	Set = RespCommand{
		argLen: 3,
		Type:   WRITE,
		Execute: func(args []string, server RedisServer) (string, error) {
			if len(args) < 2 {
				fmt.Println("args", args)
				return "", errors.New("insufficient arguments")
			}

			command := SET
			argMap := map[string][]string{}

			for i := range strings.Split("key value "+CommandFlags["PX"]+" milliseconds", " ") {
				if i >= len(args) {
					break
				}

				arg := args[i]

				if IsRespFlag(arg) {
					command = strings.ToUpper(arg)
					continue
				}

				argMap[command] = append(argMap[command], arg)
			}

			key, value := argMap[SET][0], argMap[SET][1]
			expArgs, exists := argMap["PX"]
			expiry := "0"

			if exists {
				expiry = expArgs[0]
			}

			Memory[key] = *NewMemoryItem(value, expiry)
			return ToRespSimpleString("OK"), nil
		},
	}
	Get = RespCommand{
		argLen: 2,
		Execute: func(args []string, server RedisServer) (string, error) {
			memItem, exists := Memory[args[0]]

			if !exists {
				fmt.Printf("key %s does not exist\n", args[0])
				return NULL_BULK_STRING, nil
			}

			value, err := memItem.GetValue()

			if err != nil {
				fmt.Printf("%s\n", err.Error())
				return NULL_BULK_STRING, nil
			}

			return ToRespBulkString(value), nil
		},
	}
	Info = RespCommand{
		argLen: 2,
		Execute: func(args []string, server RedisServer) (string, error) {
			infoType := args[0]

			switch infoType {
			case REPLICATION:
				response := []string{"#Replication"}
				replicaInfo := server.ReplicaInfo()
				valueOfReplInfo := reflect.ValueOf(replicaInfo)
				typeOfReplInfo := reflect.TypeOf(replicaInfo)

				// TODO: implement a struct serializer?
				for i := 0; i < valueOfReplInfo.NumField(); i++ {
					field := valueOfReplInfo.Field(i)
					fieldName := typeOfReplInfo.Field(i).Name
					response = append(response, fmt.Sprintf("%s:%v", CamelCaseToSnakeCase(fieldName), field))
				}

				return ToRespBulkString(strings.Join(response, "\r\n")), nil
			default:
				return ToRespSimpleString("unsupported INFO type"), nil

			}
		},
	}
	ReplConf = RespCommand{
		argLen: 1,
		Execute: func(args []string, server RedisServer) (string, error) {
			return ToRespSimpleString(OK), nil
		},
	}
	Psync = RespCommand{
		argLen: 1,
		Execute: func(args []string, server RedisServer) (string, error) {
			return BuildPsyncResponse(server.ReplicaInfo().masterReplid), nil
		},
	}
	Wait = RespCommand{
		Execute: func(args []string, server RedisServer) (string, error) {
			masterServer, ok := server.(*RedisMasterServer)
			if !ok {
				return ToRespInteger("0"), nil
			}

			prevHistoryItem := masterServer.history.GetEntry(len(masterServer.history) - 2)
			//prevHistoryType := GetRespCommand(prevHistoryItem.command).Type

			if "" != WRITE {
				replicaConnections := strconv.Itoa(len(masterServer.replicaConnections))
				return ToRespInteger(replicaConnections), nil
			}

			numberOfReplicas, err := strconv.Atoi(args[0])
			timeoutMillis, err := strconv.Atoi(args[0])
			if err != nil {
				return "", errors.New("invalid arguments for " + WAIT)
			}

			fmt.Println("previous item before wait: ", prevHistoryItem)
			acksChan := make(chan int)
			timeoutChan := make(chan bool)
			readAcks := func() int {
				return prevHistoryItem.Acks
			}

			go func() {
				time.Sleep(time.Duration(timeoutMillis) * time.Millisecond)
				timeoutChan <- true
			}()
			go func() {
				acksChan <- readAcks()
			}()

			for {
				select {
				case <-timeoutChan:
					break
				case acks := <-acksChan:
					if acks == numberOfReplicas {

					}
					go func() {
						acksChan <- readAcks()
					}()
				}
			}

		},
	}
)

var RespCommands = map[string]RespCommand{
	PING:     Ping,
	ECHO:     Echo,
	GET:      Get,
	SET:      Set,
	INFO:     Info,
	REPLCONF: ReplConf,
	PSYNC:    Psync,
	WAIT:     Wait,
}

var CommandFlags = map[string]string{

	"PX": "PX",
}

func IsRESPCommandSupported(command string) bool {

	_, exists := RespCommands[strings.ToUpper(command)]
	return exists
}

func IsRespFlag(flag string) bool {
	_, exists := CommandFlags[strings.ToUpper(flag)]
	return exists
}
