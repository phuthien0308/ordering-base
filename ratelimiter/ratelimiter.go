package ratelimiter

import "context"

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	// AddRule dynamically registers a rate limiting bucket for a given key.
	AddRule(key string, rate float64, capacity float64)
}
