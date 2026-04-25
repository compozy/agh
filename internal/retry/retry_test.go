package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoRetriesUntilSuccessWithDeterministicJitter(t *testing.T) {
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
			RandFloat64: sequenceRand(1, 0),
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
	if got, want := delays, []time.Duration{15 * time.Millisecond, 10 * time.Millisecond}; !equalDurations(got, want) {
		t.Fatalf("delays = %#v, want %#v", got, want)
	}
}

func TestDoStopsAtMaxAttemptsAndDoesNotSleepAfterFinalFailure(t *testing.T) {
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
}

func TestDoHonorsContextCancellation(t *testing.T) {
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
}

func TestDoValueStopsOnNonRetryableAndContextErrors(t *testing.T) {
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

	_, err = DoValue(
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
}

func TestDoRejectsNilInputs(t *testing.T) {
	t.Parallel()

	if err := Do(nilRetryContext(), Policy{}, nil, func(context.Context) error { return nil }); err == nil {
		t.Fatal("Do(nil context) error = nil, want non-nil")
	}
	if err := Do(context.Background(), Policy{}, nil, nil); err == nil {
		t.Fatal("Do(nil operation) error = nil, want non-nil")
	}
}

func TestWaitHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Wait(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait(canceled) error = %v, want context.Canceled", err)
	}
}

func TestWaitHandlesNilContextAndNonPositiveDelay(t *testing.T) {
	t.Parallel()

	if err := Wait(nilRetryContext(), 0); err == nil {
		t.Fatal("Wait(nil context) error = nil, want non-nil")
	}
	if err := Wait(context.Background(), 0); err != nil {
		t.Fatalf("Wait(zero delay) error = %v", err)
	}
}

func TestDelayAppliesJitterBoundsAndCap(t *testing.T) {
	t.Parallel()

	low := Delay(Policy{
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    time.Second,
		JitterRatio: 0.2,
		RandFloat64: func() float64 { return 0 },
	}, 1)
	if low != 80*time.Millisecond {
		t.Fatalf("Delay(low jitter) = %s, want 80ms", low)
	}

	high := Delay(Policy{
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    time.Second,
		JitterRatio: 0.2,
		RandFloat64: func() float64 { return 1 },
	}, 2)
	if high != 240*time.Millisecond {
		t.Fatalf("Delay(high jitter) = %s, want 240ms", high)
	}

	capped := Delay(Policy{
		BaseDelay:   750 * time.Millisecond,
		MaxDelay:    time.Second,
		JitterRatio: 0.5,
		RandFloat64: func() float64 { return 1 },
	}, 3)
	if capped != time.Second {
		t.Fatalf("Delay(capped) = %s, want 1s", capped)
	}
}

func sequenceRand(values ...float64) func() float64 {
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

func equalDurations(left, right []time.Duration) bool {
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

func nilRetryContext() context.Context {
	return nil
}
