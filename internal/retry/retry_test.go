package retry

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDoValue(t *testing.T) {
	t.Parallel()

	t.Run("ShouldRetryUntilSuccessWithDeterministicJitter", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		delays := make([]time.Duration, 0, 2)
		errTransient := errors.New("transient")
		result, err := DoValue(
			context.Background(),
			Policy{
				MaxAttempts: 3,
				BaseDelay:   10 * time.Millisecond,
				MaxDelay:    50 * time.Millisecond,
				JitterRatio: 0.5,
				RandFloat64: sequenceRand(t, 1, 0),
				Sleep: func(_ context.Context, delay time.Duration) error {
					delays = append(delays, delay)
					return nil
				},
			},
			func(err error) bool {
				return errors.Is(err, errTransient)
			},
			func(context.Context) (string, error) {
				attempts++
				if attempts < 3 {
					return "", errTransient
				}
				return "ok", nil
			},
		)
		if err != nil {
			t.Fatalf("DoValue() error = %v", err)
		}
		if result != "ok" {
			t.Fatalf("DoValue() result = %q, want ok", result)
		}
		if attempts != 3 {
			t.Fatalf("attempts = %d, want 3", attempts)
		}
		if got, want := delays, []time.Duration{
			15 * time.Millisecond,
			10 * time.Millisecond,
		}; !equalDurations(
			t,
			got,
			want,
		) {
			t.Fatalf("delays = %#v, want %#v", got, want)
		}
	})

	t.Run("ShouldStopOnNonRetryableError", func(t *testing.T) {
		t.Parallel()

		errPermanent := errors.New("permanent")
		attempts := 0
		_, err := DoValue(
			context.Background(),
			Policy{MaxAttempts: 3},
			func(error) bool { return false },
			func(context.Context) (string, error) {
				attempts++
				return "", errPermanent
			},
		)
		if !errors.Is(err, errPermanent) {
			t.Fatalf("DoValue(nonretryable) error = %v, want permanent", err)
		}
		if attempts != 1 {
			t.Fatalf("attempts = %d, want 1", attempts)
		}
	})

	t.Run("ShouldStopOnContextErrorReturnedByOperation", func(t *testing.T) {
		t.Parallel()

		_, err := DoValue(
			context.Background(),
			Policy{MaxAttempts: 3},
			nil,
			func(context.Context) (string, error) {
				return "", context.DeadlineExceeded
			},
		)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("DoValue(context error) error = %v, want context.DeadlineExceeded", err)
		}
	})
}

func TestDo(t *testing.T) {
	t.Parallel()

	t.Run("ShouldStopAtMaxAttemptsAndAvoidSleepAfterFinalFailure", func(t *testing.T) {
		t.Parallel()

		attempts := 0
		sleepCalls := 0
		errBoom := errors.New("boom")
		err := Do(
			context.Background(),
			Policy{
				MaxAttempts: 2,
				BaseDelay:   time.Millisecond,
				Sleep: func(context.Context, time.Duration) error {
					sleepCalls++
					return nil
				},
			},
			nil,
			func(context.Context) error {
				attempts++
				return errBoom
			},
		)
		if !errors.Is(err, errBoom) {
			t.Fatalf("Do() error = %v, want errBoom", err)
		}
		if attempts != 2 {
			t.Fatalf("attempts = %d, want 2", attempts)
		}
		if sleepCalls != 1 {
			t.Fatalf("sleepCalls = %d, want 1", sleepCalls)
		}
	})

	t.Run("ShouldHonorContextCancellationBeforeFirstAttempt", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		called := false
		err := Do(ctx, Policy{MaxAttempts: 3}, nil, func(context.Context) error {
			called = true
			return errors.New("should not run")
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Do(canceled) error = %v, want context.Canceled", err)
		}
		if called {
			t.Fatal("operation called for canceled context")
		}
	})

	t.Run("ShouldRejectNilInputs", func(t *testing.T) {
		t.Parallel()

		err := Do(nilRetryContext(t), Policy{}, nil, func(context.Context) error { return nil })
		if err == nil || !strings.Contains(err.Error(), "context is required") {
			t.Fatalf("Do(nil context) error = %v, want context required", err)
		}
		err = Do(context.Background(), Policy{}, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "operation is required") {
			t.Fatalf("Do(nil operation) error = %v, want operation required", err)
		}
	})
}

func TestWait(t *testing.T) {
	t.Parallel()

	t.Run("ShouldHonorContextCancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := Wait(ctx, time.Hour); !errors.Is(err, context.Canceled) {
			t.Fatalf("Wait(canceled) error = %v, want context.Canceled", err)
		}
	})

	t.Run("ShouldHandleNilContextAndNonPositiveDelay", func(t *testing.T) {
		t.Parallel()

		if err := Wait(nilRetryContext(t), 0); err == nil || !strings.Contains(err.Error(), "context is required") {
			t.Fatalf("Wait(nil context) error = %v, want context required", err)
		}
		if err := Wait(context.Background(), 0); err != nil {
			t.Fatalf("Wait(zero delay) error = %v", err)
		}
	})
}

func TestDelay(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		policy  Policy
		attempt int
		want    time.Duration
	}{
		{
			name: "ShouldApplyLowJitterBound",
			policy: Policy{
				BaseDelay:   100 * time.Millisecond,
				MaxDelay:    time.Second,
				JitterRatio: 0.2,
				RandFloat64: func() float64 { return 0 },
			},
			attempt: 1,
			want:    80 * time.Millisecond,
		},
		{
			name: "ShouldApplyHighJitterBound",
			policy: Policy{
				BaseDelay:   100 * time.Millisecond,
				MaxDelay:    time.Second,
				JitterRatio: 0.2,
				RandFloat64: func() float64 { return 1 },
			},
			attempt: 2,
			want:    240 * time.Millisecond,
		},
		{
			name: "ShouldCapDelayAtMaxDelay",
			policy: Policy{
				BaseDelay:   750 * time.Millisecond,
				MaxDelay:    time.Second,
				JitterRatio: 0.5,
				RandFloat64: func() float64 { return 1 },
			},
			attempt: 3,
			want:    time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := Delay(tc.policy, tc.attempt); got != tc.want {
				t.Fatalf("Delay() = %s, want %s", got, tc.want)
			}
		})
	}
}

func sequenceRand(t *testing.T, values ...float64) func() float64 {
	t.Helper()

	index := 0
	return func() float64 {
		if index >= len(values) {
			return values[len(values)-1]
		}
		value := values[index]
		index++
		return value
	}
}

func equalDurations(t *testing.T, left, right []time.Duration) bool {
	t.Helper()

	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func nilRetryContext(t *testing.T) context.Context {
	t.Helper()

	return nil
}
