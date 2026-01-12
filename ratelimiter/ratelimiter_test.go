package ratelimiter

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenBucket_RefillGoroutine(t *testing.T) {
	// Rate: 100 tokens/sec, Capacity: 5
	// Every 10ms it should refill 1 token.
	tb := NewTokenBucket(100, 5)
	defer tb.Stop()

	// Consume all 5 initial tokens
	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// 6th should be blocked
	if tb.Allow() {
		t.Error("6th request should be blocked")
	}

	// Wait for 15ms (should refill at least 1 token at the 10ms mark)
	time.Sleep(15 * time.Millisecond)

	if !tb.Allow() {
		t.Error("Should be allowed after background refill")
	}
}

func TestTokenBucket_Stop(t *testing.T) {
	tb := NewTokenBucket(100, 5)

	// Consume all
	for i := 0; i < 5; i++ {
		tb.Allow()
	}

	tb.Stop()

	// Wait for 50ms
	time.Sleep(50 * time.Millisecond)

	// Since it's stopped, no new tokens should be added
	if tb.Allow() {
		t.Error("Should NOT be allowed after stop")
	}
}

func TestTokenBucket_Concurrency(t *testing.T) {
	// Start with 100 tokens, but very slow refill (1/sec) to avoid test flakiness
	tb := NewTokenBucket(1, 100)
	defer tb.Stop()

	const workers = 10
	const reqsPerWorker = 10
	done := make(chan bool)

	var successfulAllows int64

	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < reqsPerWorker; j++ {
				if tb.Allow() {
					atomic.AddInt64(&successfulAllows, 1)
				}
			}
			done <- true
		}()
	}

	for i := 0; i < workers; i++ {
		<-done
	}

	// We have 10 workers * 10 requests/worker = 100 total requests.
	// The bucket capacity is 100.
	// Since the refill rate is 1 token/sec and the test finishes quickly,
	// we expect exactly 100 successful allows (the initial capacity).
	if atomic.LoadInt64(&successfulAllows) != 100 {
		t.Errorf("Expected 100 successful allows, got %d", atomic.LoadInt64(&successfulAllows))
	}

	if tb.Allow() {
		t.Error("Bucket should be empty after consuming all tokens in a fast burst")
	}
}
