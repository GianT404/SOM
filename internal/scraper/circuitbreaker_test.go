package scraper

import (
	"errors"
	"testing"
	"time"
)

var errCBTest = errors.New("error test")

func TestCircuitBreaker_OpensAfterMaxFailures(t *testing.T) {
	cb := newCircuitBreaker(3, time.Hour)

	for i := 0; i < 3; i++ {
		if !cb.allow() {
			t.Fatalf("lần thử  %d trước khi đủ maxFailures phải đc  allow", i+1)
		}
		cb.recordResult(errCBTest)
	}

	if cb.allow() {
		t.Fatal("sau đủ  maxFailures, circuit phải đóng và không cho phép allow")
	}
}

func TestCircuitBreaker_ResetsOnSuccess(t *testing.T) {
	cb := newCircuitBreaker(3, time.Hour)

	cb.recordResult(errCBTest)
	cb.recordResult(errCBTest)
	cb.recordResult(nil)
	for i := 0; i < 3; i++ {
		if !cb.allow() {
			t.Fatal("sau khi reset,phải cho phép allow")
		}
		cb.recordResult(errCBTest)
	}
	if cb.allow() {
		t.Fatal("sau khi đủ maxFailures sau reset, circuit phải đóng và không cho phép allow")
	}
}

func TestCircuitBreaker_HalfOpenAfterCooldown(t *testing.T) {
	cb := newCircuitBreaker(2, 50*time.Millisecond)

	cb.recordResult(errCBTest)
	cb.recordResult(errCBTest)

	if cb.allow() {
		t.Fatal("sau khi đủ maxFailures, circuit phải đóng và không cho phép allow")
	}

	time.Sleep(80 * time.Millisecond)

	if !cb.allow() {
		t.Fatal("sau khi cooldown, circuit phải half-open và cho phép allow")
	}
}
