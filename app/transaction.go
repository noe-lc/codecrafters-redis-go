package main

import "net"

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

func (t *Transaction) Reset() {
	t.Conn = nil
	t.Queue = []CommandComponents{}
}

func ExecTransaction() {

}
