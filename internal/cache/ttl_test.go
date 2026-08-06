package cache

import (
	"testing"
	"time"
)

func TestTTLCache_GetPut(t *testing.T) {
	c := NewTTL[string](10, time.Hour)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss on empty cache")
	}
	c.Put("k", "v")
	v, ok := c.Get("k")
	if !ok || v != "v" {
		t.Fatalf("expected hit v, got %q ok=%v", v, ok)
	}
}

func TestTTLCache_Expired(t *testing.T) {
	c := NewTTL[string](10, 10*time.Millisecond)
	c.Put("k", "v")
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected miss after TTL")
	}
}

func TestTTLCache_EvictsOldestWhenFull(t *testing.T) {
	c := NewTTL[int](2, time.Hour)
	c.Put("a", 1)
	c.Put("b", 2)
	c.Put("c", 3) // evicts "a" (sớm nhất)
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected a evicted")
	}
	if _, ok := c.Get("b"); !ok {
		t.Fatal("expected b present")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatal("expected c present")
	}
	if c.Len() != 2 {
		t.Fatalf("expected len 2, got %d", c.Len())
	}
}
