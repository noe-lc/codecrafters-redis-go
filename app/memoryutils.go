package main

import (
	"errors"
	"strconv"
	"strings"
)

func splitStreamId(id string) (int, int) {
	splitId := strings.Split(id, "-")
	ms, _ := strconv.Atoi(splitId[0])
	seq, _ := strconv.Atoi(splitId[1])
	return ms, seq
}

func ValidateStreamId(memoryKey, id string) error {
	ms, seq := splitStreamId(id)
	memItem, ok := Memory[memoryKey]

	if !ok {
		if ms == 0 && seq == 0 {
			return errors.New("the ID specified in XADD must be greater than 0-0")
		}
	}

	stream := memItem.value.getValue().(Stream)
	lastStream := stream[len(stream)-1]
	lastMs, lastSeq := splitStreamId(lastStream["id"].(string))

	if ms < lastMs {
		return errors.New("the ID specified in XADD is equal or smaller than the target stream top item")
	}

	if ms == lastMs && seq < lastSeq {
		return errors.New("the ID specified in XADD is equal or smaller than the target stream top item")
	}

	return nil
}
