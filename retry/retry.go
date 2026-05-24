package retry

import (
	"context"
	"math/rand"

	"time"
)

func Retry[T any](ctx context.Context, fn func() (T, error), retriable func(error) bool, attempt int,
	initalDurationInMs time.Duration) (T, error) {
	var result T
	var err error
	timer := time.NewTimer(0)
	for i := 0; i < attempt; i++ {
		result, err = fn()
		if err == nil {
			break
		}
		if !retriable(err) {
			break
		}
		waiting := waitingTime(i, initalDurationInMs)
		timer.Reset(waiting)
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-timer.C:
			continue
		}
	}
	return result, err
}

func waitingTime(attempt int, initialDuration time.Duration) time.Duration {
	backoff := initialDuration
	jitter := rand.Int63n(int64(backoff))
	return backoff*time.Duration(attempt) + time.Duration(jitter)
}
