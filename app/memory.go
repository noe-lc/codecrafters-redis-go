package main

import (
	"errors"
	"fmt"
	"time"
)

type ServerMemory map[string]MemoryItem

var Memory ServerMemory = ServerMemory{}

// Memory errors
var (
	ErrExpiredKey = errors.New("expired key")
)

// memory item types
const (
	STRING = "string"
	STREAM = "stream"
)

// memory item utility values
const (
	XRANGE_MINUS = "-"
	XRANGE_PLUS  = "+"
)

func (m *ServerMemory) AddStreamItem(key string, item StreamItem) error {
	memItem, exists := Memory[key]

	if !exists {
		Memory[key] = MemoryItem{&StreamValue{item}, 0}
		return nil
	}

	value, valueType := memItem.GetValueDirectly()
	if valueType != STREAM {
		return fmt.Errorf("cannot insert stream item to non-stream key `%s`", key)
	}
	stream := value.(*StreamValue)
	newStream := StreamValue(append(*stream, item))
	Memory[key] = MemoryItem{&newStream, 0}
	return nil
}

type MemoryItem struct {
	value   MemoryItemValue
	expires int64
}

type MemoryItemValue interface {
	getValue() (interface{}, string)
}

func NewMemoryItem(value MemoryItemValue, expires int64) MemoryItem {
	return MemoryItem{
		value,
		expires,
	}
}

func (c *MemoryItem) GetValue() (interface{}, error) {
	if c.expires != 0 && time.Now().UnixMilli() > c.expires {
		return "", ErrExpiredKey
	}

	return c.value, nil
}

func (c *MemoryItem) GetValueDirectly() (interface{}, string) {
	return c.value.getValue()
}

// ToRespString transforms the value into the required response RESP string.
// It receives the same list of arguments as the command does on each RespCommand Execute call.
func (c *MemoryItem) ToRespString() (string, error) {
	value, valueType := c.GetValueDirectly()
	switch valueType {
	case STRING:
		stringValue := value.(*StringValue)
		return ToRespBulkString(string(*stringValue)), nil
	case STREAM:
		stream := value.(*StreamValue)
		streamRespArray := StreamItemsToRespArray(*stream)
		fmt.Println("stream array: ", streamRespArray)
		return streamRespArray, nil
	default:
		return "", nil
	}
}

type StringValue string

func (s *StringValue) getValue() (interface{}, string) {
	return s, STRING
}

// TODO: maybe make this a struct to require id
type StreamItem map[string]interface{}

func NewStreamItem(id string, entries []string) StreamItem {
	prevKey := ""
	streamItem := StreamItem{"id": id}
	for _, entry := range entries {
		if prevKey != "" {
			streamItem[prevKey] = entry
			prevKey = ""
		} else {
			prevKey = entry
		}
	}

	return streamItem
}

type StreamValue []StreamItem

func (s *StreamValue) getValue() (interface{}, string) {
	return s, STREAM
}
