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
	XREAD_ONLY_NEW          = "$"
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
	EMPTY_KEY_TYPE                = "none"
	ERROR                         = "-ERR"
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

// Handshake constants
const (
	LISTENING_PORT_ARG = "listening-port"
)

// RDB constants
const (
	RDB_EMPTY_FILE_HEX   = "524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2"
	RDB_DEFAULT_DIR      = "rdb"
	RDB_DIR_ARG          = "dir"
	RDB_DEFAULT_FILENAME = "rdbfile"
	RDB_FILENAME_ARG     = "dbfilename"
)

var RDB_CONFIG = map[string]string{
	RDB_DIR_ARG:      RDB_DEFAULT_DIR,
	RDB_FILENAME_ARG: RDB_DEFAULT_FILENAME,
}

const (
	RDB_MAGIC_STRING        = "REDIS0007"
	RDB_METADATA_START      = "FA"
	RDB_DB_SUBSECTION_START = "FE"
	RDB_HASH_TABLE_START    = "FB"
	RDB_END_OF_FILE         = "FF"
	RDB_STRING_KEY          = "00"
	RDB_TIMESTAMP_MILLIS    = "FC" // 8 bytes
	RDB_TIMESTAMP_SECONDS   = "FD" // 4 bytes
)

const (
	RDB_TIMESTAMP_MILLIS_BYTE_LENGTH  = 8
	RDB_TIMESTAMP_SECONDS_BYTE_LENGTH = 4
)

var (
	RDB_MAGIC_STRING_BYTE, _        = RDBHexStringToByte(RDB_MAGIC_STRING)
	RDB_METADATA_START_BYTE, _      = RDBHexStringToByte(RDB_METADATA_START)
	RDB_DB_SUBSECTION_START_BYTE, _ = RDBHexStringToByte(RDB_DB_SUBSECTION_START)
	RDB_HASH_TABLE_START_BYTE, _    = RDBHexStringToByte(RDB_HASH_TABLE_START)
	RDB_END_OF_FILE_BYTE, _         = RDBHexStringToByte(RDB_END_OF_FILE)
	RDB_STRING_KEY_BYTE, _          = RDBHexStringToByte(RDB_STRING_KEY)
	RDB_TIMESTAMP_MILLIS_BYTE, _    = RDBHexStringToByte(RDB_TIMESTAMP_MILLIS)
	RDB_TIMESTAMP_SECONDS_BYTE, _   = RDBHexStringToByte(RDB_TIMESTAMP_SECONDS)
)
