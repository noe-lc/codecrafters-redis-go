package main

import (
	"net"
)

type Transaction struct {
	Conn  net.Conn
	Queue []CommandComponents
}

// New Transaction returns a new transaction with an empty queue and a nil channel
func NewTransaction(conn net.Conn) Transaction {
	return Transaction{conn, []CommandComponents{}}
}

// EnqueueCommand appens a new set of command components into the Transaction
func (t *Transaction) EnqueueCommand(cmp CommandComponents) {
	t.Queue = append(t.Queue, cmp)
}

func (t *Transaction) ExecTransaction(s RedisServer) string {
	results := []string{}
	for _, cmp := range t.Queue {
		command, args, _ := cmp.Command, cmp.Args, cmp.Input
		respCommand := RespCommands[command]
		result, err := respCommand.Execute(args, s)

		if err != nil {
			results = append(results, err.Error())
		} else {
			results = append(results, result)
		}

	}

	t.Reset()
	return ConcatIntoRespArray(results)
}

func (t *Transaction) Reset() {
	t.Conn = nil
	t.Queue = []CommandComponents{}
}
