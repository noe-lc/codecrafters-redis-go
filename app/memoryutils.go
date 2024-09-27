package main

import (
	"errors"
	"fmt"
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
		return nil
	}
	if ms == 0 && seq == 0 {
		return errors.New("the ID specified in XADD must be greater than 0-0")
	}

	value, valueType := memItem.GetValueDirectly()
	if valueType != "stream" {
		return errors.New("cannot validate agains a non-stream key " + memoryKey)
	}

	streamPtr := value.(*StreamValue)
	stream := *streamPtr
	lastStream := stream[len(stream)-1]
	fmt.Println("last stream:", lastStream)
	lastMs, lastSeq := splitStreamId(lastStream["id"].(string))

	if ms > lastMs {
		return nil
	}
	if ms < lastMs {
		return errors.New("the ID specified in XADD is equal or smaller than the target stream top item")
	}
	if seq <= lastSeq {
		return errors.New("the ID specified in XADD is equal or smaller than the target stream top item")
	}

	return nil
}
