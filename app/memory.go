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
	INT    = "int"
	STRING = "string"
	STREAM = "stream"
)

// memory item utility values
const (
	XRANGE_MINUS = "-"
	XRANGE_PLUS  = "+"
)

func (m *ServerMemory) AddStreamItem(key string, s Stream) error {
	memItem, exists := Memory[key]

	if !exists {
		Memory[key] = MemoryItem{&StreamValue{s}, 0}
		return nil
	}

	value, valueType := memItem.GetValueDirectly()
	if valueType != STREAM {
		return fmt.Errorf("cannot insert stream s to non-stream key `%s`", key)
	}
	stream := *(value.(*StreamValue))
	stream = append(stream, s)
	Memory[key] = MemoryItem{&stream, 0}
	return nil
}

func (m *ServerMemory) LookupStream(key string) (StreamValue, error) {
	memItem, ok := (*m)[key]
	if !ok {
		return nil, fmt.Errorf("stream with key %s does not exist", key)
	}
	value, valueType := memItem.GetValueDirectly()
	if valueType != STREAM {
		return nil, fmt.Errorf("value at key %s is not a stream", key)
	}
	return *(value.(*StreamValue)), nil
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
		return streamRespArray, nil
	default:
		return "", nil
	}
}

type IntegerValue int

func (s *IntegerValue) getValue() (interface{}, string) {
	return s, INT
}

type StringValue string

func (s *StringValue) getValue() (interface{}, string) {
	return s, STRING
}

type Stream struct {
	id      string
	values  map[string]interface{}
	created int64
}

func NewStreamItem(id string, entries []string) Stream {
	prevKey := ""
	stream := Stream{id: id, created: time.Now().UnixMilli(), values: map[string]interface{}{}}
	for _, entry := range entries {
		if prevKey != "" {
			stream.values[prevKey] = entry
			prevKey = ""
		} else {
			prevKey = entry
		}
	}

	return stream
}

type StreamValue []Stream

func (s *StreamValue) getValue() (interface{}, string) {
	return s, STREAM
}

func (s *StreamValue) LookupItem(id string) (Stream, int, error) {
	for i, stream := range *s {
		itemId := stream.id
		if itemId == id {
			return stream, i, nil
		}
	}
	return Stream{}, 0, errors.New("stream item not found")
}
