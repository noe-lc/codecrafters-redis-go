package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func extractCommandAndArgs(message string) (string, []string) {
	splitMessage := regexp.MustCompile(`\s\s*`).Split(message, -1)

	if len(splitMessage) > 1 {
		return splitMessage[0], splitMessage[1:]
	}

	return splitMessage[0], []string{}
}

func getRespMessageLen(s string, firstChar string) int {
	lenRegexp := regexp.MustCompile("\\" + firstChar + "\\d" + "+")
	arrayLenString := strings.Replace(lenRegexp.FindString(s), firstChar, "", -1)
	arrayLen, err := strconv.Atoi(arrayLenString)

	if err != nil {
		return -1
	}

	return arrayLen
}

func splitFullRespArrayMessage(s string, protocolTerminator string) ([]string, error) {
	r := regexp.MustCompile(protocolTerminator)
	splitString := r.Split(s, -1)
	lengthStr, parts := splitString[0], splitString[1:]
	length, err := strconv.Atoi(strings.Replace(lengthStr, ARRAY, "", 1))

	if err != nil {
		return nil, err
	}

	if len(parts)/2 < length {
		return nil, fmt.Errorf("specified an array length of %s, got %d", lengthStr, len(parts)/2)
	}

	finalMessageParts := []string{}

	for i := 0; i < length*2; i += 2 {
		stringLen := parts[i]
		stringValue := parts[i+1]
		length := getRespMessageLen(stringLen, BULK_STRING)

		if length == -1 {
			return []string{}, fmt.Errorf("invalid bulk string length %s", stringLen)
		}

		if len(stringValue) < length {
			return []string{}, fmt.Errorf("string does not have enough length, attempted to read %d, actual length is %d", length, len(stringValue))
		}

		finalMessageParts = append(finalMessageParts, stringValue[:length])

	}

	return finalMessageParts, nil
}
