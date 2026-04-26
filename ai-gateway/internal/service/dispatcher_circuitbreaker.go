package service

import (
	"sync"
	"time"
)

type CircuitBreaker struct {
	mu             sync.RWMutex
	failures       map[int64]int
	lastFailure    map[int64]time.Time
	threshold      int
	timeout        time.Duration
}

func NewCircuitBreaker(threshold int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failures:    make(map[int64]int),
		lastFailure: make(map[int64]time.Time),
		threshold:   threshold,
		timeout:     timeout,
	}
}

func (cb *CircuitBreaker) RecordSuccess(channelID int64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures[channelID] = 0
	cb.lastFailure[channelID] = time.Time{}
}

func (cb *CircuitBreaker) RecordFailure(channelID int64) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures[channelID]++
	cb.lastFailure[channelID] = time.Now()
}

func (cb *CircuitBreaker) IsOpen(channelID int64) bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	failures := cb.failures[channelID]
	lastFailure := cb.lastFailure[channelID]
	
	if failures >= cb.threshold {
		if time.Since(lastFailure) < cb.timeout {
			return true
		}
	}
	return false
}

func (cb *CircuitBreaker) GetFailureCount(channelID int64) int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures[channelID]
}
