package main

import (
	"errors"
	"fmt"
	"time"
)

type StreamEntry map[string]interface{}

type Stream []StreamEntry
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
	t, err := item.Type()
	if err != nil {
		fmt.Println(err)
	}
	switch t {
	case "stream":
		currentStream, ok := Memory[key]
		if ok {
			stream := currentStream.getValueDirectly().(Stream)
			streamItem, _ := item.getValueDirectly().(StreamEntry)
			s := StreamValue{append(stream, streamItem)}
			Memory[key] = MemoryItem{value: s}

		}

	default:
		Memory[key] = item
	}

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

func (c *MemoryItem) getValueDirectly() interface{} {
	return c.value.getValue()
}

func (c *MemoryItem) Update() (interface{}, error) {

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
