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
	XRANGE   = "XRANGE"
	XREAD    = "XREAD"
	INCR     = "INCR"
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

			key, stringValueArg := argMap[SET][0], argMap[SET][1]
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

			var memoryItemValue MemoryItemValue
			numericValue, err := strconv.Atoi(stringValueArg)
			if err == nil {
				value := IntegerValue(numericValue)
				memoryItemValue = &value
			} else {
				value := StringValue(stringValueArg)
				memoryItemValue = &value

			}

			Memory[key] = MemoryItem{memoryItemValue, itemExpires}
			return ToRespSimpleString("OK"), nil
		},
	}
	Get = RespCommand{
		argLen: 2,
		Execute: func(args []string, server RedisServer) (string, error) {
			key := args[0]
			memItem, exists := Memory[key]

			if exists {
				_, err := memItem.GetValue()
				if err != nil {
					if err == ErrExpiredKey {
						return NULL_BULK_STRING, nil
					}
					fmt.Printf("Failed to get key %s: %v\n", key, err)
					return "", err
				}

				respString, err := memItem.ToRespString()
				if err != nil {
					return "", err
				}
				return respString, nil
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
				memValue := StringValue(entry.value)
				memItem := MemoryItem{&memValue, expires}
				_, err := memItem.GetValue()
				if err != nil {
					if err == ErrExpiredKey {
						return NULL_BULK_STRING, nil
					}
					return "", err
				}

				return ToRespBulkString(entry.value), nil
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
				return ToRespInteger(0), nil
			}
			if len(masterServer.replicas) < 1 {
				return ToRespInteger(0), nil
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
				return ToRespInteger(len(masterServer.replicas)), nil
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
						return ToRespInteger(numberOfReplicas), nil
					}
				case <-timer:
					lastAcksRead := masterServer.waitAckFor.Acks
					masterServer.SetAcknowledgeItem(nil, nil)
					return ToRespInteger(lastAcksRead), nil
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
			memItem, exists := Memory[key]
			if !exists {
				return ToRespSimpleString(EMPTY_KEY_TYPE), nil
			}
			_, err := memItem.GetValue()
			if err != nil {
				if err == ErrExpiredKey {
					return NULL_BULK_STRING, nil
				} else {
					fmt.Printf("Failed to get key %s: %v\n", key, err)
					return "", err
				}
			}
			_, valueType := memItem.GetValueDirectly()
			return ToRespSimpleString(valueType), nil
		},
	}
	XAdd = RespCommand{
		Execute: func(args []string, rs RedisServer) (string, error) {
			concatArgs := strings.Join(args, " ")
			simpleStreamRegExp := `(\w+){1} (([0-9]+-([0-9]|\*))+|\*{1}) (\w+ )+\w+$`
			isSimpleStream, _ := regexp.MatchString(simpleStreamRegExp, concatArgs)

			switch {
			case isSimpleStream:
				key, idArg := args[0], args[1]
				newId, err := GenerateStreamId(key, idArg)
				if err != nil {
					return ToRespError(err), nil
				}

				streamItem := NewStreamItem(newId, args[2:])

				Memory.AddStreamItem(key, streamItem)

				status := rs.GetStatus()
				if status.XReadBlock != nil {
					status.XReadBlock <- true
				}

				return ToRespBulkString(newId), nil
			default:
				fmt.Println("unrecognized XADD args")
				return ToRespBulkString(""), nil
			}
		},
	}
	XRange = RespCommand{
		Execute: func(args []string, rs RedisServer) (string, error) {
			key, startId, endId := args[0], args[1], args[2]
			memItem, ok := Memory[key]

			if !ok {
				return "", fmt.Errorf("stream with key %s does not exist", key)
			}

			value, valueType := memItem.GetValueDirectly()
			if valueType != STREAM {
				return "", fmt.Errorf("value at key %s is not a stream", key)
			}

			stream := *(value.(*StreamValue))
			streamItemsMatched := []Stream{}
			if endId == XRANGE_PLUS {
				for i, item := range stream {
					if item.id >= startId {
						streamItemsMatched = stream[i:]
						break
					}
				}
			} else {
				endId = endId + "-9" // append -9 as suffix to ensure every id is within range
				for _, item := range stream {
					itemId := item.id
					if startId <= itemId && endId >= itemId {
						streamItemsMatched = append(streamItemsMatched, item)
					}
				}
			}

			newStream := StreamValue(streamItemsMatched)
			newMemItem := MemoryItem{&newStream, 0}
			respStringResponse, _ := newMemItem.ToRespString()
			return respStringResponse, nil
		},
	}
	XRead = RespCommand{
		Execute: func(args []string, rs RedisServer) (string, error) {
			concatArgs := strings.Join(args, " ")
			blockRegex := `^block \d+ streams \w+ (([0-9]+-([0-9]|\*))+|\*{1}|\${1})$`
			streamReadRegex := `^streams (\w+ )+((([0-9]+-([0-9]|\*))+|\*{1}) )*(([0-9]+-([0-9]|\*))+|\*{1})$`
			numKeys := len(args[1:]) / 2
			streamItemsMatched := []Stream{}
			idStreamItemsStr := ARRAY + strconv.Itoa(numKeys) + PROTOCOL_TERMINATOR
			isBlockRead, _ := regexp.MatchString(blockRegex, concatArgs)
			isSimpleRead, _ := regexp.MatchString(streamReadRegex, concatArgs)

			if isSimpleRead {
				for keyIndex := 1; keyIndex <= numKeys; keyIndex++ {
					idIndex := numKeys + keyIndex
					key, id := args[keyIndex], args[idIndex]
					stream, err := Memory.LookupStream(key)
					if err != nil {
						return NULL_BULK_STRING, err
					}

					for i, item := range stream {
						if item.id > id {
							streamItemsMatched = stream[i:]
							break
						}
					}

					idStreamItemsStr += ARRAY + "2" + PROTOCOL_TERMINATOR + ToRespBulkString(key) + StreamItemsToRespArray(streamItemsMatched)
				}

				idStreamItemsStr += StreamItemsToRespArray(streamItemsMatched)
				return idStreamItemsStr, nil
			}

			if isBlockRead {
				blockMsStr, key, id := args[1], args[3], args[4]
				blockTime, err := time.ParseDuration(blockMsStr + "ms")
				if err != nil {
					return "", err
				}

				stream, err := Memory.LookupStream(key)
				if err != nil {
					return "", err
				}
				lastKnownIndex := len(stream)
				if lastKnownIndex > 0 {
					lastKnownIndex -= 1
				}

				status := rs.GetStatus()
				// onlyNewReads := strings.HasSuffix(concatArgs, XREAD_ONLY_NEW)

				if blockTime == 0 {
					status.XReadBlock = make(chan bool)
					<-status.XReadBlock
				} else {
					duration := time.Duration(blockTime.Milliseconds()) * time.Millisecond
					time.Sleep(duration)
				}

				var index int

				if id == XREAD_ONLY_NEW {
					index = lastKnownIndex
				} else {
					_, index, err = stream.LookupItem(id)
					if err != nil {
						return "", err
					}
				}

				stream, _ = Memory.LookupStream(key)
				if len(stream) <= index+1 {
					return NULL_BULK_STRING, nil
				}

				status.XReadBlock = nil

				streamItem := stream[index+1]
				return ARRAY + "1" + PROTOCOL_TERMINATOR + ARRAY + "2" + PROTOCOL_TERMINATOR + BULK_STRING + strconv.Itoa(len(key)) + PROTOCOL_TERMINATOR + key + PROTOCOL_TERMINATOR + StreamItemsToRespArray([]Stream{streamItem}), nil
			}

			return "", nil
		},
	}
	Incr = RespCommand{
		Execute: func(args []string, rs RedisServer) (string, error) {
			memItem, exists := Memory[args[0]]

			if !exists {
				return "", errors.New("key does not exist")
			}

			value, valueType := memItem.GetValueDirectly()

			if valueType != INT {
				return "", errors.New("cannot increment non-integer value")
			}

			integerValue := value.(*IntegerValue)
			updatedInt := int(*integerValue) + 1
			*integerValue = IntegerValue(updatedInt)
			Memory[args[0]] = MemoryItem{integerValue, memItem.expires}
			return ToRespInteger(updatedInt), nil
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
	XRANGE:   XRange,
	XREAD:    XRead,
	INCR:     Incr,
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
