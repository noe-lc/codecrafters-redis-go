package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
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
	CONFIG   = "CONFIG"
	KEYS     = "KEYS"
	TYPE     = "TYPE"
	XADD     = "XADD"
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
			expiresInMs, err := strconv.Atoi(expiry)
			if err != nil {
				return "", err
			}
			itemExpires := int64(0)
			if expiresInMs != 0 {
				itemExpires = time.Now().UnixMilli() + int64(expiresInMs)
			}
			Memory[key] = *NewMemoryItem(value, itemExpires)
			return ToRespSimpleString("OK"), nil
		},
	}
	Get = RespCommand{
		argLen: 2,
		Execute: func(args []string, server RedisServer) (string, error) {
			key := args[0]
			memItem, exists := Memory[key]

			if exists {
				value, err := memItem.GetValue()
				if err != nil {
					if err == ErrExpiredKey {
						return NULL_BULK_STRING, nil
					}
					fmt.Printf("Failed to get key %s: %v\n", key, err)
					return "", err
				}

				return ToRespBulkString(value), nil
			}

			filePath := GetRDBFilePath(server)
			if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
				return NULL_BULK_STRING, nil
			} else if err != nil {
				return "", err
			}

			dbEntries, err := GetRDBEntries(filePath)
			if err != nil {
				if err == io.EOF {
					return NULL_BULK_STRING, nil
				}
				return "", err
			}

			for _, entry := range dbEntries {
				if key != entry.key {
					continue
				}

				expires := int64(0)
				if entry.expiry != 0 {
					expires = entry.expiry
				}
				memItem := NewMemoryItem(entry.value, expires)
				value, err := memItem.GetValue()
				if err != nil {
					if err == ErrExpiredKey {
						return NULL_BULK_STRING, nil
					}
					return "", err
				}

				return ToRespBulkString(value), nil
			}

			return NULL_BULK_STRING, nil

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
	Config = RespCommand{
		Execute: func(args []string, server RedisServer) (string, error) {
			concatArgs := strings.Join(args, " ")
			configRdb, _ := regexp.MatchString(`^`+GET+` `+`(`+RDB_DIR_ARG+`|`+RDB_FILENAME_ARG+`)$`, concatArgs)

			switch {
			case configRdb:
				rdbArg := args[1]
				return ToRespArrayString(rdbArg, server.GetRDBConfig()[rdbArg]), nil
			default:
				return ToRespArrayString(""), nil
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
			if len(masterServer.replicas) < 1 {
				return ToRespInteger("0"), nil
			}
			numberOfReplicas, err := strconv.Atoi(args[0])
			if err != nil {
				return "", errors.New("invalid num replicas arguments for " + WAIT)
			}
			timeoutMillis, err := strconv.Atoi(args[1])
			if err != nil {
				return "", errors.New("invalid timeout value for " + WAIT)
			}

			prevHistoryItem := masterServer.history.GetModifiableEntry(len(masterServer.history) - 2)

			if prevHistoryItem.RespCommand.Type != WRITE {
				numberOfReplicas := strconv.Itoa(len(masterServer.replicas))
				return ToRespInteger(numberOfReplicas), nil
			}

			ackChan := make(chan bool)
			// handle the acknowledge update during command execution
			masterServer.SetAcknowledgeItem(prevHistoryItem, ackChan)

			for _, replica := range masterServer.replicas {
				_, err := replica.conn.Write([]byte(ToRespArrayString(REPLCONF, GETACK, GETACK_FROM_REPLICA_ARG)))
				if err != nil {
					fmt.Println("Failed write GETACK to " + replica.conn.RemoteAddr().String())
					continue
				}
			}

			timer := time.After(time.Duration(timeoutMillis) * time.Millisecond)

			for {
				select {
				case <-ackChan:
					if masterServer.waitAckFor.Acks == numberOfReplicas {
						masterServer.SetAcknowledgeItem(nil, nil)
						return ToRespInteger(strconv.Itoa(numberOfReplicas)), nil
					}
				case <-timer:
					lastAcksRead := masterServer.waitAckFor.Acks
					masterServer.SetAcknowledgeItem(nil, nil)
					return ToRespInteger(strconv.Itoa(lastAcksRead)), nil
				}
			}

		},
	}
	Save = RespCommand{
		Execute: func(s []string, rs RedisServer) (string, error) {
			return "", nil
		},
	}
	Keys = RespCommand{
		Execute: func(args []string, rs RedisServer) (string, error) {
			pattern := args[0]
			filePath := GetRDBFilePath(rs)

			switch pattern {
			case "*":
				/* 				f, _ := os.OpenFile(filePath, os.O_RDONLY, 0644)
				   				d, _ := io.ReadAll(f)
				   				f, _ = os.OpenFile(filePath, os.O_RDONLY, 0644)
				   				b := []byte{}
				   				for {
				   					buf := make([]byte, 1)
				   					r, err := f.Read(buf)
				   					if err != nil {
				   						fmt.Println("decode err", err, r)
				   						break
				   					}
				   					b = append(b, buf[0])
				   				}
				   				fmt.Println("data:", b)
				   				fmt.Println("string data", hex.EncodeToString(d)) */
				entries, err := GetRDBEntries(filePath)
				if err != nil {
					return "", err
				}
				keys := []string{}
				for _, entry := range entries {
					keys = append(keys, entry.key)
				}

				fmt.Println("keys:", keys)
				return ToRespArrayString(keys...), nil
			default:
				return ToRespArrayString(""), nil
			}

		},
	}
	Type = RespCommand{
		Execute: func(args []string, rs RedisServer) (string, error) {
			key := args[0]
			value, err := Get.Execute([]string{key}, rs)
			if err != nil {
				return "", err
			}
			if value == NULL_BULK_STRING {
				return ToRespSimpleString(EMPTY_KEY_TYPE), nil
			}
			return ToRespSimpleString("string"), nil
		},
	}
	XAdd = RespCommand{
		Execute: func(args []string, rs RedisServer) (string, error) {
			concatArgs := strings.Join(args, " ")
			simpleStreamRegExp := `(\w+){1} [-0-9]+ (\w+ )+\w+$`
			isSimpleStream, _ := regexp.MatchString(simpleStreamRegExp, concatArgs)

			switch {
			case isSimpleStream:
				key, id := args[0], args[1]
				return ToRespArrayString(""), nil
			default:
				return ToRespArrayString(""), nil
			}

			return "", nil
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
	CONFIG:   Config,
	KEYS:     Keys,
	TYPE:     Type,
	XADD:     XAdd,
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
