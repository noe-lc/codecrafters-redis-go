package main

import (
	"errors"
	"time"
)

var (
	ErrExpiredKey = errors.New("expired key")
)

var Memory = map[string]MemoryItem{}

type MemoryItem struct {
	value   string
	created time.Time
	expires time.Duration
}

func NewMemoryItem(value string, expires string) *MemoryItem {
	exp, _ := time.ParseDuration(expires + "ms")
	return &MemoryItem{
		value:   value,
		created: time.Now(),
		expires: exp,
	}
}

func (c *MemoryItem) GetValue() (string, error) {
	expires := c.created.Add(c.expires)

	if c.expires.Milliseconds() != 0 && time.Since(expires).Milliseconds() > 0 {
		return "", ErrExpiredKey
	}

	return c.value, nil
}
