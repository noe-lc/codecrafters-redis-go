package main

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// splitStreamId returns the transformed id parts, the raw id parts, and an error
func splitStreamId(id string) ([2]int, [2]string, error) {
	var ms, seq int
	if id == "*" {
		seq = 0
		ms = int(time.Now().UnixMilli())
		return [2]int{ms, seq}, [2]string{id}, nil
	}

	splitId := strings.Split(id, "-")
	msStr, seqStr := splitId[0], splitId[1]
	ms, err := strconv.Atoi(msStr)
	if err != nil {
		return [2]int{}, [2]string{}, err
	}
	if seqStr == "*" {
		return [2]int{ms, -1}, [2]string{splitId[0], splitId[1]}, nil
	}
	seq, _ = strconv.Atoi(seqStr)
	return [2]int{ms, seq}, [2]string{splitId[0], splitId[1]}, nil
}

func GenerateStreamId(memoryKey, id string) (string, error) {
	tSplitId, rSplitId, err := splitStreamId(id)
	if err != nil {
		return "", err
	}

	ms, seq := tSplitId[0], tSplitId[1]
	if ms == 0 && seq == 0 {
		return "", errors.New("the ID specified in XADD must be greater than 0-0")
	}

	memItem, ok := Memory[memoryKey]
	rawMs, _ := rSplitId[0], rSplitId[1]
	if !ok {
		if seq != -1 {
			return strconv.Itoa(ms) + "-" + strconv.Itoa(seq), nil
		}
		if ms == 0 {
			return rawMs + "-" + "1", nil
		}
		return rawMs + "-" + "0", nil
	}

	value, valueType := memItem.GetValueDirectly()
	if valueType != "stream" {
		return "", errors.New("cannot validate against a non-stream key " + memoryKey)
	}

	streamPtr := value.(*StreamValue)
	stream := *streamPtr
	lastStream := stream[len(stream)-1]
	tLastSplitId, _, _ := splitStreamId(lastStream["id"].(string))
	lastMs, lastSeq := tLastSplitId[0], tLastSplitId[1]

	if seq != -1 {
		if ms > lastMs {
			return strconv.Itoa(ms) + "-" + "0", nil
		}
		if ms < lastMs {
			return "", errors.New("the ID specified in XADD is equal or smaller than the target stream top item")
		}
		if seq <= lastSeq {
			return "", errors.New("the ID specified in XADD is equal or smaller than the target stream top item")
		}
		return strconv.Itoa(ms) + "-" + strconv.Itoa(seq), nil
	}

	if ms < lastMs {
		return "", errors.New("the ID specified in XADD is equal or smaller than the target stream top item")
	}
	if ms > lastMs {
		return rawMs + "-" + "0", nil
	}

	newMs := strconv.Itoa(ms)
	newSeq := strconv.Itoa(lastSeq + 1)
	return newMs + "-" + newSeq, nil
}
