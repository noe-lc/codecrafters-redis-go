package main

import (
	"errors"
	"fmt"
	"time"
)

// TODOs:
// Opt 1: Create a memvalue interface with a method and make string and Stream types satisfy it
//
//	then pass the new interface as return and param types
//
// Opt 1: Make the memItem value an interface altogether

type StreamEntry map[string]interface{}

type Stream map[string]StreamEntry
type MemoryValue interface {
	getValue() interface{}
}

type StringValue struct {
	value string
}

func NewStringValue(value string) *StringValue {
	return &StringValue{value}
}

func (s *StringValue) getValue() interface{} {
	return s.value
}

type StreamValue struct {
	value Stream
}

func NewStreamValue(value Stream) *StreamValue {
	return &StreamValue{value}
}

func (s *StreamValue) getValue() interface{} {
	return s.value
}

var Memory = map[string]MemoryItem{}
var (
	ErrExpiredKey = errors.New("expired key")
)

func InsertIntoMemory(key string, item MemoryItem) {
	Memory[key] = item
}

type MemoryItem struct {
	value   MemoryValue
	expires int64
}

func NewMemoryItem(value MemoryValue, expires int64) *MemoryItem {
	return &MemoryItem{
		value,
		expires,
	}
}

func (c *MemoryItem) GetValue() (interface{}, error) {
	if c.expires != 0 && time.Now().UnixMilli() > c.expires {
		return "", ErrExpiredKey
	}

	return c.value.getValue(), nil
}

func (c *MemoryItem) Type() (string, error) {
	switch t := c.value.getValue().(type) {
	case string:
		return "string", nil
	case Stream:
		return "stream", nil
	default:
		return "", fmt.Errorf("illegal type %s for value in memory", t)
	}
}

func (c *MemoryItem) ToRespString() (string, error) {
	value := c.value.getValue()
	valueType, err := c.Type()
	if err != nil {
		return "", err
	}

	switch valueType {
	case "string":
		return ToRespBulkString(value.(string)), nil
	case "stream":
		return "stream", nil
	default:
		return "", nil
	}
}
