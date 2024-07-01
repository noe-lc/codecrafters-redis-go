package main

import (
	"fmt"
	"strings"
)

const (
	NULL_BULK_STRING = "$-1\r\n"
)

func encodeSimpleString(s string) string {
	return SIMPLE_STRING + s + PROTOCOL_TERMINATOR
}

func encodeBulkString(s string) string {
	return strings.Join([]string{
		BULK_STRING,
		fmt.Sprintf("%d", len(s)),
		PROTOCOL_TERMINATOR,
		s,
		PROTOCOL_TERMINATOR,
	}, "")
}
