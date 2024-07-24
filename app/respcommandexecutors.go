package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type CommandExecutor struct {
	argLen int
	// signature string
	Execute func([]string, RedisServer) (string, error)
}

func (c *CommandExecutor) GetArgLen() int {
	return c.argLen
}

var Memory = map[string]MemoryItem{}

var (
	Ping = CommandExecutor{
		argLen: 1,
		Execute: func(args []string, server RedisServer) (string, error) {
			return ToRespSimpleString("PONG"), nil
		},
	}
	Echo = CommandExecutor{
		argLen: 2,
		Execute: func(args []string, server RedisServer) (string, error) {
			if len(args) == 0 {
				return ToRespBulkString(""), nil
			}
			return ToRespBulkString(args[0]), nil
		},
	}
	Set = CommandExecutor{
		argLen: 3,
		Execute: func(args []string, server RedisServer) (string, error) {
			if len(args) < 2 {
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
	Get = CommandExecutor{
		argLen: 2,
		Execute: func(args []string, server RedisServer) (string, error) {
			memItem, exists := Memory[args[0]]

			if !exists {
				fmt.Printf("key %s does not exist\n", args[0])
				return NULL_BULK_STRING, nil
			}

			value, err := memItem.getValue()

			if err != nil {
				fmt.Printf("%s\n", err.Error())
				return NULL_BULK_STRING, nil
			}

			return ToRespBulkString(value), nil
		},
	}
	Info = CommandExecutor{
		argLen: 2,
		Execute: func(args []string, server RedisServer) (string, error) {
			infoType := args[0]

			switch infoType {
			case "replication":
				response := []string{"#Replication"}
				valueOfReplInfo := reflect.ValueOf(server.replicaInfo)
				typeOfReplInfo := reflect.TypeOf(server.replicaInfo)

				// TODO: implement a struct serializer?
				for i := 0; i < valueOfReplInfo.NumField(); i++ {
					field := valueOfReplInfo.Field(i)
					fieldName := typeOfReplInfo.Field(i).Name
					response = append(response, fmt.Sprintf("%s:%v", CamelCaseToSnakeCase(fieldName), field))
				}

				return ToRespBulkString(strings.Join(response, "\r\n")), nil
			default:
				return ToRespSimpleString("unsupported info type"), nil

			}
		},
	}
)

var CommandExecutors = map[string]CommandExecutor{
	"PING": Ping,
	"ECHO": Echo,
	"GET":  Get,
	"SET":  Set,
	"INFO": Info,
}

var CommandFlags = map[string]string{
	"PX": "PX",
}

func IsRESPCommandSupported(command string) bool {
	_, exists := CommandExecutors[strings.ToUpper(command)]
	return exists
}

func IsRespFlag(flag string) bool {
	_, exists := CommandFlags[strings.ToUpper(flag)]
	return exists
}

/* func ExecuteCommand(command string, args []string) string {
	commandExecutor, exists := CommandExecutors[command]

	if()

} */
