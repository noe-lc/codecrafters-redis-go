package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type MemoryItem struct {
	value   string
	created time.Time
	expires time.Duration
}

func NewMemoryItem(value string, expires string) *MemoryItem {
	exp, _ := time.ParseDuration(expires + "ms")

	fmt.Println(exp.Milliseconds())
	return &MemoryItem{
		value:   value,
		created: time.Now(),
		expires: exp,
	}
}

func (c *MemoryItem) getValue() (string, error) {
	expires := c.created.Add(c.expires)

	if c.expires.Milliseconds() != 0 && time.Since(expires).Milliseconds() > 0 {
		return "", errors.New("expired key")
	}

	return c.value, nil
}

type CommandExecutor struct {
	argLen    int
	signature string
	Execute   func([]string) (string, error)
}

func (c *CommandExecutor) GetArgLen() int {
	return c.argLen
}

var Memory = map[string]MemoryItem{}

var (
	Ping = CommandExecutor{
		argLen: 1,
		Execute: func(args []string) (string, error) {
			return encodeSimpleString("PONG"), nil
		},
	}
	Echo = CommandExecutor{
		argLen: 2,
		Execute: func(args []string) (string, error) {
			if len(args) == 0 {
				return encodeBulkString(""), nil
			}
			return encodeBulkString(args[0]), nil
		},
	}
	Set = CommandExecutor{
		argLen: 3,
		Execute: func(args []string) (string, error) {
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
			fmt.Printf("%#v", Memory[key])
			return encodeSimpleString("OK"), nil
		},
	}
	Get = CommandExecutor{
		argLen: 2,
		Execute: func(args []string) (string, error) {
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

			return encodeBulkString(value), nil
		},
	}
)

var CommandExecutors = map[string]CommandExecutor{
	"PING": Ping,
	"ECHO": Echo,
	"GET":  Get,
	"SET":  Set,
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
