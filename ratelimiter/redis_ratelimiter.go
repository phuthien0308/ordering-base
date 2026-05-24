package ratelimiter

import (
	"context"
	"errors"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/phuthien0308/ordering-base/simplelog/tags"
)

// RedisTokenBucket implements the Token Bucket algorithm using Redis and Lua.
var KeyNotExists = errors.New("Key not existed")

type RedisTokenBucket struct {
	logger        *simplelog.SimpleLogger
	client        *redis.Client
	Keys          map[string]KeyCapacity
	ClientID      string
	clockInSecond func() float64
}
type KeyCapacity struct {
	Key          string
	RateInSecond float64
	Burts        float64
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
local burts = tonumber(ARGV[3])

-- 1. Load data from Redis Hash
local data = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(data[1])
local last_refill = tonumber(data[2])

-- 2. Initial state if key doesn't exist
if tokens == nil then
    tokens = burts
    last_refill = now
end

-- 3. Calculate refill
local elapsed = math.max(0, now - last_refill)
local refill = elapsed * rate
tokens = math.min(burts, tokens + refill)

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

// Allow checks if a request is permitted via Context injection.
func (rtb *RedisTokenBucket) Allow(ctx context.Context, key string) (bool, error) {

	keyCap, err := rtb.getKey(key)
	if err != nil {
		// No rate limit rule configuration found, allow the request
		return true, nil
	}

	// Suffix the global rule name with the client ID to isolate buckets per user/IP
	redisKey := key
	if rtb.ClientID != "" {
		redisKey += ":" + rtb.ClientID
	}

	now := rtb.clockInSecond()

	result, err := tokenBucketScript.Run(ctx, rtb.client, []string{redisKey}, now, keyCap.RateInSecond, keyCap.Burts).Result()
	if err != nil {
		return false, err
	}
	rtb.logger.Info(ctx, "Rate limit check result", tags.String("key", redisKey), tags.Int64("result", result.(int64)))
	return result.(int64) == 1, nil
}

// AddRule registers a new rate limit configuration for a specific key.
func (rtb *RedisTokenBucket) AddRule(key string, rate float64, burts float64) {
	if rtb.Keys == nil {
		rtb.Keys = make(map[string]KeyCapacity)
	}
	rtb.Keys[key] = KeyCapacity{
		Key:          key,
		RateInSecond: rate,
		Burts:        burts,
	}
}

func (rtb *RedisTokenBucket) getKey(key string) (*KeyCapacity, error) {
	for _, existingKey := range rtb.Keys {
		if strings.EqualFold(key, existingKey.Key) {
			return &existingKey, nil
		}
	}
	return nil, KeyNotExists
}
