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

func (m *ServerMemory) AddStreamItem(key string, item StreamItem) error {
	var currentValue interface{}
	var currentValueType string
	memItem, exists := Memory[key]

	if !exists {
		Memory[key] = MemoryItem{&StreamValue{item}, 0}
		return nil
	}

	currentValue, currentValueType = memItem.GetValueDirectly()
	switch currentValueType {
	case "stream":
		stream := currentValue.(*StreamValue)
		newStream := StreamValue(append(*stream, item))
		Memory[key] = MemoryItem{&newStream, 0}
	default:
		return fmt.Errorf("cannot insert stream item to non-stream key `%s`", key)
	}

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

func (c *MemoryItem) ToRespString() (string, error) {
	value, valueType := c.GetValueDirectly()
	switch valueType {
	case STRING:
		stringValue := value.(*StringValue)
		return ToRespBulkString(string(*stringValue)), nil
	case STREAM:
		// TODO: transform the stream into resp array of arrays here
		return STREAM, nil
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

type StreamValue []StreamItem

func (s *StreamValue) getValue() (interface{}, string) {
	return s, STREAM
}
