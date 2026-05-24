package ratelimiter

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/phuthien0308/ordering-base/simplelog"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	// AddRule dynamically registers a rate limiting bucket for a given key.
	AddRule(key string, rate float64, capacity float64)
}

// NewRedisTokenBucket creates a new Redis-based rate limiter.
func NewRedisTokenBucket(logger *simplelog.SimpleLogger, clientID string, client *redis.Client) *RedisTokenBucket {
	ratelimit := &RedisTokenBucket{
		logger:   logger,
		client:   client,
		ClientID: clientID,
		clockInSecond: func() float64 {
			return float64(time.Now().UnixNano()) / 1e9
		},
	}
	return ratelimit
}
