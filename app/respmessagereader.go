package main

import (
	"errors"
	"io"
	"regexp"
	"strings"
)

type RESPMessageReader struct {
	command   string
	args      []string
	len       int
	lenRead   int
	nextBytes int
}

func NewRESPMessageReader() RESPMessageReader {
	return RESPMessageReader{nextBytes: -1}
}

func (r *RESPMessageReader) Read(message string) (bool, error) {
	trimmedMessage := strings.TrimRight(message, "\r\n")

	// handling partial resp array
	if r.len != 0 {
		// reads bulk string length
		if r.nextBytes == -1 {
			stringLen := getRespMessageLen(trimmedMessage, BULK_STRING)
			if stringLen == -1 {
				return true, errors.New("invalid bulk string length in message" + trimmedMessage)
			}
			r.nextBytes = stringLen
			return false, nil
		}

		// sets command
		if r.command == "" {
			if !isRespCommand(trimmedMessage) {
				return true, errors.New("invalid or unsupported resp command: " + trimmedMessage)
			}
			r.setCommandAndArgs(trimmedMessage, r.args)
		} else { // reads bulk string of length = nextBytes
			nextArg := trimmedMessage[:r.nextBytes]
			r.setCommandAndArgs(r.command, append(r.args, nextArg))
		}

		r.advancePartialRespRead()
		if r.len == r.lenRead {
			return true, nil
		}

		return false, nil
	}

	// full resp array
	isFullRespArray, _ := regexp.MatchString(FULL_RESP_ARRAY, message)
	isFullRespArrayUnescaped, _ := regexp.MatchString(FULL_RESP_ARRAY_UNESCAPED, message)
	isPartialRespArray, _ := regexp.MatchString(PARTIAL_RESP_ARRAY, message)

	if isFullRespArray || isFullRespArrayUnescaped {
		protocolTerminator := PROTOCOL_TERMINATOR_RAW
		if isFullRespArrayUnescaped {
			protocolTerminator = PROTOCOL_TERMINATOR_UNESCAPED_RAW
		}
		messageParts, err := splitFullRespArrayMessage(trimmedMessage, protocolTerminator)
		if err != nil {
			return true, errors.New("failed to read full RESP array: " + err.Error())
		}

		r.setCommandAndArgs(messageParts[0], messageParts[1:])
		return true, nil
	}

	if isPartialRespArray {
		arrayLen := getRespMessageLen(trimmedMessage, ARRAY)
		r.len = arrayLen
		return false, nil
	}

	// plain message, with or without command
	command, args := extractCommandAndArgs(trimmedMessage)

	if !isRespCommand(command) {
		return true, errors.New("invalid or unsupported command in plain string: " + command)
	}

	r.setCommandAndArgs(command, args)
	return true, nil
}

func (r *RESPMessageReader) setCommandAndArgs(command string, args []string) {
	r.command = strings.ToUpper(command)
	r.args = args
}

// Use when reading an actual string value. Adds 1 to the read length and
// sets the nextBytes to -1 so a bulk string length is read on the next Read call.
func (r *RESPMessageReader) advancePartialRespRead() {
	r.lenRead += 1
	r.nextBytes = -1
}

func (r *RESPMessageReader) GetCommandAndArgs() (string, []string) {
	return r.command, r.args
}

func (r *RESPMessageReader) Write(w io.Writer) (int, error) {
	return w.Write([]byte{0})
}

// resets the struct to its initial state. Must be called explicitly
func (r *RESPMessageReader) Reset() {
	*r = RESPMessageReader{nextBytes: -1}
}
