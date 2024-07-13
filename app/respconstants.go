package main

// Supported commands
const (
	PING = "PING"
	ECHO = "ECHO"
	SET  = "SET"
	GET  = "GET"
)

// RESP protocol constants. Use for interpreted strings, and regex only if characters are not escaped
const (
	PROTOCOL_TERMINATOR           = "\r\n"
	PROTOCOL_TERMINATOR_UNESCAPED = "\\r\\n"
	SIMPLE_STRING                 = "+"
	BULK_STRING                   = "$"
	NULL_BULK_STRING              = "$-1\r\n"
	ARRAY                         = "*"
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
