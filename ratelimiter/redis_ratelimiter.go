package ratelimiter

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/phuthien0308/ordering-base/simplelog/tags"
)

// RedisTokenBucket implements the Token Bucket algorithm using Redis and Lua.
type RedisTokenBucket struct {
	logger   *simplelog.SimpleZapLogger
	client   *redis.Client
	key      string
	rate     float64 // tokens per second
	capacity float64 // max tokens
	clock    func() float64
}

// NewRedisTokenBucket creates a new Redis-based rate limiter.
func NewRedisTokenBucket(logger *simplelog.SimpleZapLogger, client *redis.Client, key string, rate, capacity float64) *RedisTokenBucket {
	return &RedisTokenBucket{
		logger:   logger,
		client:   client,
		key:      key,
		rate:     rate,
		capacity: capacity,
		clock: func() float64 {
			return float64(time.Now().UnixNano()) / 1e9
		},
	}
}

// Lua script for atomic Token Bucket logic in Redis.
// KEYS[1]: The rate limit key (e.g., "ratelimit:user:123")
// ARGV[1]: Current Unix timestamp (fractional)
// ARGV[2]: Refill rate (tokens per second)
// ARGV[3]: Bucket capacity
var tokenBucketScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])

-- 1. Load data from Redis Hash
local data = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

-- 2. Initial state if key doesn't exist
if tokens == nil then
    tokens = capacity
    last_refill = now
end

-- 3. Calculate refill
local elapsed = math.max(0, now - last_refill)
local refill = elapsed * rate
tokens = math.min(capacity, tokens + refill)

-- 4. Check and consume
local allowed = 0
if tokens >= 1 then
    tokens = tokens - 1
    allowed = 1
end

-- 5. Save state
redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
-- Expire after 1 hour of inactivity to save memory
redis.call('EXPIRE', key, 3600)

return allowed
`)

// Allow checks if a request is permitted.
func (rtb *RedisTokenBucket) Allow(ctx context.Context) (bool, error) {
	now := rtb.clock()

	result, err := tokenBucketScript.Run(ctx, rtb.client, []string{rtb.key}, now, rtb.rate, rtb.capacity).Result()
	if err != nil {
		return false, err
	}
	rtb.logger.Info(ctx, "Rate limit check result", tags.String("key", rtb.key), tags.Int64("result", result.(int64)))
	return result.(int64) == 1, nil
}
