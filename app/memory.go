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
	expires int64
}

func NewMemoryItem(value string, expires int64) *MemoryItem {
	return &MemoryItem{
		value:   value,
		expires: expires,
	}
}

func (c *MemoryItem) GetValue() (string, error) {
	if c.expires != 0 && time.Now().UnixMilli() > c.expires {
		return "", ErrExpiredKey
	}

	return c.value, nil
}
