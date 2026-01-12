package ratelimiter

import (
	"math"
	"sync"
	"time"
)

// RateLimiter defines the interface for different rate limiting algorithms.
type RateLimiter interface {
	Allow() bool
}

// TokenBucket implements the Token Bucket rate limiting algorithm
// using a background goroutine for replenishing tokens.
type TokenBucket struct {
	rate     float64    // tokens per second
	capacity float64    // maximum tokens (burst size)
	tokens   float64    // current available tokens
	mu       sync.Mutex // ensures thread safety
	stop     chan struct{}
}

// NewTokenBucket creates a new TokenBucket and starts the refill goroutine.
// rate: tokens per second
// capacity: maximum tokens in the bucket
func NewTokenBucket(rate, capacity float64) *TokenBucket {
	tb := &TokenBucket{
		rate:     rate,
		capacity: capacity,
		tokens:   capacity, // start full
		stop:     make(chan struct{}),
	}

	go tb.refillLoop()

	return tb
}

// refillLoop runs in the background and replenishes tokens at a fixed interval.
func (tb *TokenBucket) refillLoop() {
	// Refill every 10ms for smooth replenishment
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	tokensPerTick := tb.rate * 0.01 // 10ms is 0.01 seconds

	for {
		select {
		case <-ticker.C:
			tb.mu.Lock()
			tb.tokens = math.Min(tb.capacity, tb.tokens+tokensPerTick)
			tb.mu.Unlock()
		case <-tb.stop:
			return
		}
	}
}

// Allow checks if a request is permitted.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}

	return false
}

// Stop shuts down the background refill goroutine.
func (tb *TokenBucket) Stop() {
	close(tb.stop)
}
