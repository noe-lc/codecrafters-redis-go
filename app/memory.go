package main

import (
	"errors"
	"time"
)

type StreamEntry map[string]interface{}

type Stream map[string]StreamEntry

type MemoryValue interface {
	string | Stream
}

// TODOs:
// Opt 1: Create a memvalue interface with a method and make string and Stream types satisfy it
//
//	then pass the new interface as return and param types
//
// Opt 1: Make the memItem value an interface altogether
type MemoryItem[T MemoryValue] struct {
	value   T
	expires int64
}

var (
	ErrExpiredKey = errors.New("expired key")
)

var Memory = map[string]MemoryItem[MemoryValue]{}

func NewMemoryItem[T MemoryValue](value T, expires int64) *MemoryItem[T] {
	return &MemoryItem[T]{
		value:   value,
		expires: expires,
	}
}

func (c *MemoryItem[T]) GetValue() (T, error) {
	if c.expires != 0 && time.Now().UnixMilli() > c.expires {
		var zero T
		return zero, ErrExpiredKey
	}

	return c.value, nil
}
