package ratelimiter

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v8"
	"github.com/phuthien0308/ordering-base/simplelog"
	"go.uber.org/zap"
)

func TestRedisTokenBucket(t *testing.T) {
	db, mock := redismock.NewClientMock()
	ctx := context.Background()
	key := "test-limiter"
	rate := 10.0
	capacity := 5.0

	rtb := NewRedisTokenBucket(&simplelog.SimpleLogger{Logger: zap.NewNop()}, "ClientID", db)
	rtb.AddRule(key, rate, capacity)

	now := 1705000000.0
	rtb.clockInSecond = func() float64 { return now }

	// Mock for the first call (initial state)
	mock.ExpectEvalSha(tokenBucketScript.Hash(), []string{key + ":" + "ClientID"}, now, rate, capacity).SetVal(int64(1))

	allowed, err := rtb.Allow(ctx, key)
	if err != nil {
		t.Fatalf("Allow failed: %v", err)
	}
	if !allowed {
		t.Error("Expected request to be allowed")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
