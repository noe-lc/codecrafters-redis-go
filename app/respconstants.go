package main

// Standard reponses
const (
	OK         = "OK"
	PONG       = "PONG"
	FULLRESYNC = "FULLRESYNC"
)

// Argument constants
const (
	REPLICATION             = "replication"
	ACK                     = "ACK"
	GETACK                  = "GETACK"
	GETACK_FROM_REPLICA_ARG = "*"
)

// RESP protocol constants. Use for interpreted strings, and regex only if characters are not escaped
const (
	PROTOCOL_TERMINATOR           = "\r\n"
	PROTOCOL_TERMINATOR_UNESCAPED = "\\r\\n"
	SIMPLE_STRING                 = "+"
	BULK_STRING                   = "$"
	NULL_BULK_STRING              = "$-1\r\n"
	ARRAY                         = "*"
	INTEGER                       = ":"
	INTEGER_POSITIVE              = "+"
	INTEGER_NEGATIVE              = "-"
)

// RESP protocol raw constants. Use for regex.
const (
	PROTOCOL_TERMINATOR_RAW           = `\r\n`
	PROTOCOL_TERMINATOR_UNESCAPED_RAW = `\\r\\n`
)

// regexp strings used for command type identification
const (
	// ^\*\d+\r\n(\$\d+\r\n)+
	FULL_RESP_ARRAY = `^\` + ARRAY + `\d+` + PROTOCOL_TERMINATOR_RAW + `(\` + BULK_STRING + `\d+` + PROTOCOL_TERMINATOR_RAW + `)+`
	// ^\*\d+\\r\\n(\$\d+\\r\\n)+
	FULL_RESP_ARRAY_UNESCAPED = `^\` + ARRAY + `\d+` + PROTOCOL_TERMINATOR_UNESCAPED_RAW + `(\` + BULK_STRING + `\d+` + PROTOCOL_TERMINATOR_UNESCAPED_RAW + `)+`
	// ^\*\d+\r\n$
	PARTIAL_RESP_ARRAY = `^\` + ARRAY + `\d+` + PROTOCOL_TERMINATOR_RAW + `$`
)

// Other constants
const (
	RDB_EMPTY_FILE_HEX = "524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2"
)

// Handshake constants
const (
	LISTENING_PORT_ARG = "listening-port"
)
