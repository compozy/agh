// Package retry provides small context-aware retry and backoff primitives.
package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// Policy configures retry attempts and jittered exponential backoff.
type Policy struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	JitterRatio float64
	RandFloat64 func() float64
	Sleep       func(context.Context, time.Duration) error
	OnRetry     func(Attempt)
}

// Attempt describes one failed attempt that will be retried.
type Attempt struct {
	Number      int
	MaxAttempts int
	Err         error
	Delay       time.Duration
}

// ShouldRetry reports whether an operation error should be retried.
type ShouldRetry func(error) bool

const (
	defaultMaxAttempts = 1
	defaultBaseDelay   = 100 * time.Millisecond
	defaultMaxDelay    = 30 * time.Second
)

// Do retries fn until it succeeds, shouldRetry rejects the error, attempts are
// exhausted, or ctx is canceled.
func Do(ctx context.Context, policy Policy, shouldRetry ShouldRetry, fn func(context.Context) error) error {
	if fn == nil {
		return errors.New("retry: operation is required")
	}
	_, err := DoValue(ctx, policy, shouldRetry, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, fn(ctx)
	})
	return err
}

// DoValue is Do for operations that return a value.
func DoValue[T any](
	ctx context.Context,
	policy Policy,
	shouldRetry ShouldRetry,
	fn func(context.Context) (T, error),
) (T, error) {
	var zero T
	if ctx == nil {
		return zero, errors.New("retry: context is required")
	}
	if fn == nil {
		return zero, errors.New("retry: operation is required")
	}

	policy = normalizePolicy(policy)
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return zero, err
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}
		if isContextError(err) || attempt == policy.MaxAttempts || !retryable(shouldRetry, err) {
			return zero, err
		}

		delay := Delay(policy, attempt)
		if policy.OnRetry != nil {
			policy.OnRetry(Attempt{
				Number:      attempt,
				MaxAttempts: policy.MaxAttempts,
				Err:         err,
				Delay:       delay,
			})
		}
		if err := policy.Sleep(ctx, delay); err != nil {
			return zero, err
		}
	}

	return zero, nil
}

// Delay returns the jittered exponential backoff delay for the attempt number.
func Delay(policy Policy, attempt int) time.Duration {
	policy = normalizePolicy(policy)
	if attempt < 1 {
		attempt = 1
	}

	multiplier := math.Pow(2, float64(attempt-1))
	delay := float64(policy.BaseDelay) * multiplier
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}
	if policy.JitterRatio > 0 {
		jitterRange := delay * policy.JitterRatio
		delay += (policy.RandFloat64()*2 - 1) * jitterRange
	}
	if delay < 0 {
		return 0
	}
	if delay > float64(policy.MaxDelay) {
		return policy.MaxDelay
	}
	return time.Duration(delay)
}

// Wait blocks for delay or returns early when ctx is canceled.
func Wait(ctx context.Context, delay time.Duration) error {
	if ctx == nil {
		return errors.New("retry: context is required")
	}
	if delay <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func normalizePolicy(policy Policy) Policy {
	if policy.MaxAttempts <= 0 {
		policy.MaxAttempts = defaultMaxAttempts
	}
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = defaultBaseDelay
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = defaultMaxDelay
	}
	if policy.MaxDelay < policy.BaseDelay {
		policy.MaxDelay = policy.BaseDelay
	}
	if policy.JitterRatio < 0 {
		policy.JitterRatio = 0
	}
	if policy.RandFloat64 == nil {
		policy.RandFloat64 = rand.Float64
	}
	if policy.Sleep == nil {
		policy.Sleep = Wait
	}
	return policy
}

func retryable(shouldRetry ShouldRetry, err error) bool {
	if shouldRetry == nil {
		return true
	}
	return shouldRetry(err)
}

func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
