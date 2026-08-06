package cache

import (
	"sync"
	"time"
)

// TTLCache là cache in-memory đơn giản có TTL và giới hạn số entry.
// Khi đầy: evict các entry hết hạn trước, nếu vẫn đầy thì đá entry có
// expiresAt sớm nhất. An toàn cho dùng đồng thời (mutex).
type TTLCache[T any] struct {
	mu      sync.Mutex
	entries map[string]entry[T]
	max     int
	ttl     time.Duration
}

type entry[T any] struct {
	value     T
	expiresAt time.Time
}

func NewTTL[T any](max int, ttl time.Duration) *TTLCache[T] {
	return &TTLCache[T]{
		entries: make(map[string]entry[T]),
		max:     max,
		ttl:     ttl,
	}
}

func (c *TTLCache[T]) Get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok {
		var zero T
		return zero, false
	}
	if time.Now().After(e.expiresAt) {
		delete(c.entries, key)
		var zero T
		return zero, false
	}
	return e.value, true
}

func (c *TTLCache[T]) Put(key string, v T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()

	if len(c.entries) >= c.max {
		for k, e := range c.entries {
			if now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
	}
	if len(c.entries) >= c.max {
		var oldestKey string
		var oldest time.Time
		first := true
		for k, e := range c.entries {
			if first || e.expiresAt.Before(oldest) {
				oldest = e.expiresAt
				oldestKey = k
				first = false
			}
		}
		if oldestKey != "" {
			delete(c.entries, oldestKey)
		}
	}

	c.entries[key] = entry[T]{value: v, expiresAt: now.Add(c.ttl)}
}

func (c *TTLCache[T]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
