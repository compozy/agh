package bridgesdk

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

// ErrorClass is the shared bridge-sdk provider failure classification.
type ErrorClass string

const (
	ErrorClassAuth      ErrorClass = "auth"
	ErrorClassRateLimit ErrorClass = "rate_limit"
	ErrorClassTimeout   ErrorClass = "timeout"
	ErrorClassTransient ErrorClass = "transient"
	ErrorClassPermanent ErrorClass = "permanent"
)

// HTTPError captures provider HTTP failures with optional Retry-After guidance.
type HTTPError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Message) == "" {
		return fmt.Sprintf("http %d", e.StatusCode)
	}
	return e.Message
}

// AuthError marks an authentication failure explicitly.
type AuthError struct {
	Err error
}

func (e *AuthError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *AuthError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// RateLimitError marks a rate-limit failure explicitly.
type RateLimitError struct {
	Err        error
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *RateLimitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// TransientError marks a retryable provider failure explicitly.
type TransientError struct {
	Err error
}

func (e *TransientError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *TransientError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// PermanentError marks a non-retryable provider failure explicitly.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// ClassifiedError is one actionable provider failure classification.
type ClassifiedError struct {
	Class      ErrorClass
	Err        error
	RetryAfter time.Duration
	Message    string
}

// RecoveryDecision is the runtime action derived from one classified error.
type RecoveryDecision struct {
	Retry       bool
	RetryAfter  time.Duration
	Status      bridgepkg.BridgeStatus
	Degradation *bridgepkg.BridgeDegradation
}

// RetryConfig configures the jittered backoff retry helper.
type RetryConfig struct {
	Attempts  int
	MinDelay  time.Duration
	MaxDelay  time.Duration
	Jitter    float64
	RandFloat func() float64
	OnRetry   func(attempt int, maxAttempts int, classified ClassifiedError)
}

// DefaultRetryConfig returns the bridge-sdk default retry policy.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Attempts:  3,
		MinDelay:  300 * time.Millisecond,
		MaxDelay:  30 * time.Second,
		Jitter:    0.1,
		RandFloat: rand.Float64,
	}
}

// ClassifyError maps one provider failure into the shared recovery classes.
func ClassifyError(err error) ClassifiedError {
	if err == nil {
		return ClassifiedError{}
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return ClassifiedError{
			Class:   ErrorClassAuth,
			Err:     err,
			Message: errorMessage(err),
		}
	}

	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return ClassifiedError{
			Class:      ErrorClassRateLimit,
			Err:        err,
			RetryAfter: rateLimitErr.RetryAfter,
			Message:    errorMessage(err),
		}
	}

	var permanentErr *PermanentError
	if errors.As(err, &permanentErr) {
		return ClassifiedError{
			Class:   ErrorClassPermanent,
			Err:     err,
			Message: errorMessage(err),
		}
	}

	var transientErr *TransientError
	if errors.As(err, &transientErr) {
		return ClassifiedError{
			Class:   ErrorClassTransient,
			Err:     err,
			Message: errorMessage(err),
		}
	}

	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return ClassifiedError{Class: ErrorClassAuth, Err: err, Message: errorMessage(err)}
		case http.StatusTooManyRequests:
			return ClassifiedError{
				Class:      ErrorClassRateLimit,
				Err:        err,
				RetryAfter: httpErr.RetryAfter,
				Message:    errorMessage(err),
			}
		case http.StatusRequestTimeout, http.StatusGatewayTimeout:
			return ClassifiedError{Class: ErrorClassTimeout, Err: err, Message: errorMessage(err)}
		case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusInternalServerError:
			return ClassifiedError{Class: ErrorClassTransient, Err: err, Message: errorMessage(err)}
		default:
			return ClassifiedError{Class: ErrorClassPermanent, Err: err, Message: errorMessage(err)}
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ClassifiedError{Class: ErrorClassTimeout, Err: err, Message: errorMessage(err)}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ClassifiedError{Class: ErrorClassTimeout, Err: err, Message: errorMessage(err)}
		}
		return ClassifiedError{Class: ErrorClassTransient, Err: err, Message: errorMessage(err)}
	}

	text := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(text, "auth"), strings.Contains(text, "forbidden"), strings.Contains(text, "unauthorized"), strings.Contains(text, "token"):
		return ClassifiedError{Class: ErrorClassAuth, Err: err, Message: errorMessage(err)}
	case strings.Contains(text, "rate limit"), strings.Contains(text, "too many requests"):
		return ClassifiedError{Class: ErrorClassRateLimit, Err: err, Message: errorMessage(err)}
	case strings.Contains(text, "timeout"), strings.Contains(text, "deadline exceeded"):
		return ClassifiedError{Class: ErrorClassTimeout, Err: err, Message: errorMessage(err)}
	case strings.Contains(text, "temporary"), strings.Contains(text, "unavailable"), strings.Contains(text, "connection reset"), strings.Contains(text, "broken pipe"), strings.Contains(text, "eof"):
		return ClassifiedError{Class: ErrorClassTransient, Err: err, Message: errorMessage(err)}
	default:
		return ClassifiedError{Class: ErrorClassPermanent, Err: err, Message: errorMessage(err)}
	}
}

// Recovery maps the classified provider failure into runtime actions.
func (c ClassifiedError) Recovery() RecoveryDecision {
	switch c.Class {
	case ErrorClassAuth:
		return RecoveryDecision{
			Status: bridgepkg.BridgeStatusAuthRequired,
			Degradation: &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
				Message: c.Message,
			},
		}
	case ErrorClassRateLimit:
		return RecoveryDecision{
			Retry:      true,
			RetryAfter: c.RetryAfter,
			Status:     bridgepkg.BridgeStatusDegraded,
			Degradation: &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonRateLimited,
				Message: c.Message,
			},
		}
	case ErrorClassTimeout:
		return RecoveryDecision{
			Retry:  true,
			Status: bridgepkg.BridgeStatusDegraded,
			Degradation: &bridgepkg.BridgeDegradation{
				Reason:  bridgepkg.BridgeDegradationReasonProviderTimeout,
				Message: c.Message,
			},
		}
	case ErrorClassTransient:
		return RecoveryDecision{
			Retry:  true,
			Status: bridgepkg.BridgeStatusDegraded,
		}
	case ErrorClassPermanent:
		return RecoveryDecision{
			Status: bridgepkg.BridgeStatusError,
		}
	default:
		return RecoveryDecision{}
	}
}

// RetryDo retries the operation according to the shared classification policy.
func RetryDo[T any](ctx context.Context, config RetryConfig, fn func(context.Context) (T, error)) (T, error) {
	if config.Attempts <= 0 {
		config.Attempts = 1
	}
	if config.MinDelay <= 0 {
		config.MinDelay = 300 * time.Millisecond
	}
	if config.MaxDelay <= 0 {
		config.MaxDelay = 30 * time.Second
	}
	if config.RandFloat == nil {
		config.RandFloat = rand.Float64
	}

	var zero T
	for attempt := 1; attempt <= config.Attempts; attempt++ {
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		classified := ClassifyError(err)
		recovery := classified.Recovery()
		if !recovery.Retry || attempt == config.Attempts {
			return zero, err
		}

		delay := retryDelay(config, attempt, recovery)
		if config.OnRetry != nil {
			config.OnRetry(attempt, config.Attempts, classified)
		}

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
		}
	}

	return zero, nil
}

func retryDelay(config RetryConfig, attempt int, recovery RecoveryDecision) time.Duration {
	if recovery.RetryAfter > 0 {
		return recovery.RetryAfter
	}

	delay := float64(config.MinDelay) * math.Pow(2, float64(attempt-1))
	if time.Duration(delay) > config.MaxDelay {
		delay = float64(config.MaxDelay)
	}
	if config.Jitter > 0 {
		jitterRange := delay * config.Jitter
		delay += (config.RandFloat()*2 - 1) * jitterRange
	}
	if delay < float64(config.MinDelay) {
		delay = float64(config.MinDelay)
	}
	return time.Duration(delay)
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}
