package main

import (
	"errors"
	"regexp"
	"strings"
)

type RESPMessageReader struct {
	rawInput string
	command  string
	args     []string
	len      int
	lenRead  int
	// lenLimit  int
	nextBytes int
}

type CommandComponents struct {
	// the full raw string from which command and args is derived
	Input string
	// the RESP command
	Command string
	// the RESP command arguments
	Args []string
}

func NewRESPMessageReader() RESPMessageReader {
	return RESPMessageReader{nextBytes: -1}
}

func (r *RESPMessageReader) Read(message string) (bool, error) {
	trimmedMessage := strings.TrimRight(message, "\r\n")

	// handling partial resp array
	if r.len != 0 {
		r.rawInput += message

		// reads bulk string length
		if r.nextBytes == -1 {
			stringLen := getRespMessageLen(trimmedMessage, BULK_STRING)
			if stringLen == -1 {
				return true, errors.New("invalid bulk string length in message" + trimmedMessage)
			}
			r.nextBytes = stringLen
			return false, nil
		}

		// set command or read bulk string of length = nextBytes
		if r.command == "" {
			if !IsRESPCommandSupported(strings.ToUpper(trimmedMessage)) {
				return true, errors.New("invalid or unsupported resp command: " + trimmedMessage)
			}
			r.setCommand(trimmedMessage)
			// r.setLengthLimit(CommandExecutors[r.command].argLen)
		} else {
			nextArg := trimmedMessage[:r.nextBytes] // TODO: validate this length
			r.setArgs(append(r.args, nextArg))
		}

		ready := r.advancePartialRespRead()
		if ready {
			return true, nil
		}

		return false, nil
	}

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

		r.setCommand(messageParts[0])
		r.setArgs(messageParts[1:])
		r.rawInput = message
		return true, nil
	}

	if isPartialRespArray {
		arrayLen := getRespMessageLen(trimmedMessage, ARRAY)
		r.len = arrayLen
		r.rawInput = message
		return false, nil
	}

	// plain message, with or without command
	command, args := extractCommandAndArgs(trimmedMessage)

	if IsRESPCommandSupported(command) {
		r.setCommand(command)
		r.setArgs(args)
		r.rawInput = message
		return true, nil
	}

	return true, errors.New("invalid or unsupported command in plain string: " + command)
}

// Sets the underlying RESP command, making it uppercase. Command must be validated beforehand.
func (r *RESPMessageReader) setCommand(command string) {
	r.command = strings.ToUpper(command)
}

func (r *RESPMessageReader) setArgs(args []string) {
	r.args = args
}

/* func (r *RESPMessageReader) setLengthLimit(limit int) {
	r.lenLimit = limit
} */

// Use when reading an actual string value. Adds 1 to the read length and
// sets the nextBytes to -1 so a bulk string length is read on the next Read call.
// Returns true when either a number of arguments larger or equal to the command arg length limit,
// or a number of arguments equal to the bulk string arg length have been read.
func (r *RESPMessageReader) advancePartialRespRead() bool {
	r.lenRead += 1
	r.nextBytes = -1
	return r.len == r.lenRead // || r.lenRead >= r.lenLimit
}

func (r *RESPMessageReader) GetCommandComponents() CommandComponents {
	return CommandComponents{
		Input:   r.rawInput,
		Command: r.command,
		Args:    r.args,
	}
}

// resets the struct to its initial state. Must be called explicitly
func (r *RESPMessageReader) Reset() {
	*r = RESPMessageReader{nextBytes: -1}
}
