package scraper

import (
	"errors"
	"sync"
	"time"
)

var ErrCircuitOpen = errors.New("Service temporarily overloaded due to consecutive errors; please try again later.")

type circuitBreaker struct {
	mu          sync.Mutex
	failures    int
	openUntil   time.Time
	maxFailures int
	cooldown    time.Duration
}

func newCircuitBreaker(maxFailures int, cooldown time.Duration) *circuitBreaker {
	return &circuitBreaker{maxFailures: maxFailures, cooldown: cooldown}
}

func (cb *circuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.failures < cb.maxFailures {
		return true
	}
	return time.Now().After(cb.openUntil)
}

func (cb *circuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.failures++
		if cb.failures >= cb.maxFailures {
			cb.openUntil = time.Now().Add(cb.cooldown)
		}
		return
	}
	cb.failures = 0
}
