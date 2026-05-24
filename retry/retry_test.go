package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWaitingTime(t *testing.T) {
	tcs := []struct {
		name           string
		attempt        int
		intialDuration time.Duration
		minDuration    time.Duration
		maxDuration    time.Duration
	}{
		{
			name:           "retry 1",
			attempt:        1,
			intialDuration: 1 * time.Millisecond,
			minDuration:    1 * time.Millisecond,
			maxDuration:    2 * time.Millisecond,
		},
		{
			name:           "retry 2",
			attempt:        2,
			intialDuration: 1 * time.Millisecond,
			minDuration:    2 * time.Millisecond,
			maxDuration:    3 * time.Millisecond,
		},
		{
			name:           "failed case",
			attempt:        3,
			intialDuration: 1 * time.Millisecond,
			minDuration:    3 * time.Millisecond,
			maxDuration:    6 * time.Millisecond,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result := waitingTime(tc.attempt, tc.intialDuration)
			t.Logf("%s--%s", tc.name, result)
			if result < tc.minDuration || result > tc.maxDuration {
				t.Fail()
			}
		})
	}
}

func TestRetry(t *testing.T) {
	tcs := []struct {
		name            string
		attempts        int
		initialDuration time.Duration
		fn              func() (int, error) // Changed signature to just the function
		expectedErr     error
		expectedTime    time.Duration
	}{
		{
			name:            "always succeed",
			attempts:        3,
			initialDuration: 1 * time.Millisecond,
			fn: func() (int, error) {
				return 42, nil
			},
			expectedErr:  nil,
			expectedTime: 1 * time.Millisecond,
		},
		{
			name:            "always fail",
			attempts:        3,
			initialDuration: 1 * time.Millisecond,
			fn: func() (int, error) {
				return 0, errors.New("permanent error")
			},
			expectedErr:  errors.New("permanent error"), // Need to assert the error string
			expectedTime: 8 * time.Millisecond,
		},
	}

	// Dynamic closure for "succeeds after retries"
	calls := 0
	tcs = append(tcs, struct {
		name            string
		attempts        int
		initialDuration time.Duration
		fn              func() (int, error)
		expectedErr     error
		expectedTime    time.Duration
	}{
		name:            "succeeds after retries",
		attempts:        3,
		initialDuration: 1 * time.Millisecond,
		fn: func() (int, error) {
			calls++
			if calls < 3 {
				return 0, errors.New("temporary error")
			}
			return 42, nil
		},
		expectedErr:  nil,
		expectedTime: 3 * time.Millisecond,
	})

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Pass tc.initialDuration instead of hardcoded time.Millisecond(1)
			startedTime := time.Now()
			result, err := Retry(context.Background(), tc.fn, func(error) bool { return true }, tc.attempts, tc.initialDuration)

			// Assert errors
			if tc.expectedErr != nil {
				if err == nil || err.Error() != tc.expectedErr.Error() {
					t.Errorf("expected error %v, got %v", tc.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				// Assert result only when we expect success
				if result != 42 && result != 0 {
					t.Errorf("got unexpected result: %v", result)
				}
			}
			elapsed := time.Since(startedTime)
			if tc.expectedTime < elapsed {
				t.Errorf("problem with time elapsed: %v, expected time : %v", elapsed, tc.expectedTime)
			}
		})
	}
}
