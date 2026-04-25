package bridgesdk

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

func TestClassifyErrorMapsRepresentativeProviderFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantClass  ErrorClass
		wantRetry  bool
		wantStatus bridgepkg.BridgeStatus
		wantReason bridgepkg.BridgeDegradationReason
	}{
		{
			name:       "auth",
			err:        &AuthError{Err: errors.New("invalid token")},
			wantClass:  ErrorClassAuth,
			wantRetry:  false,
			wantStatus: bridgepkg.BridgeStatusAuthRequired,
			wantReason: bridgepkg.BridgeDegradationReasonAuthFailed,
		},
		{
			name: "rate_limit",
			err: &HTTPError{
				StatusCode: http.StatusTooManyRequests,
				Message:    "too many requests",
				RetryAfter: time.Second,
			},
			wantClass:  ErrorClassRateLimit,
			wantRetry:  true,
			wantStatus: bridgepkg.BridgeStatusDegraded,
			wantReason: bridgepkg.BridgeDegradationReasonRateLimited,
		},
		{
			name:       "timeout",
			err:        context.DeadlineExceeded,
			wantClass:  ErrorClassTimeout,
			wantRetry:  true,
			wantStatus: bridgepkg.BridgeStatusDegraded,
			wantReason: bridgepkg.BridgeDegradationReasonProviderTimeout,
		},
		{
			name:       "transient",
			err:        &TransientError{Err: io.EOF},
			wantClass:  ErrorClassTransient,
			wantRetry:  true,
			wantStatus: bridgepkg.BridgeStatusDegraded,
		},
		{
			name:       "permanent",
			err:        &PermanentError{Err: errors.New("bad request")},
			wantClass:  ErrorClassPermanent,
			wantRetry:  false,
			wantStatus: bridgepkg.BridgeStatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			classified := ClassifyError(tt.err)
			if got, want := classified.Class, tt.wantClass; got != want {
				t.Fatalf("ClassifyError().Class = %q, want %q", got, want)
			}

			recovery := classified.Recovery()
			if got, want := recovery.Retry, tt.wantRetry; got != want {
				t.Fatalf("Recovery().Retry = %v, want %v", got, want)
			}
			if got, want := recovery.Status, tt.wantStatus; got != want {
				t.Fatalf("Recovery().Status = %q, want %q", got, want)
			}
			if tt.wantReason != "" {
				if recovery.Degradation == nil {
					t.Fatal("Recovery().Degradation = nil, want non-nil")
				}
				if got, want := recovery.Degradation.Reason, tt.wantReason; got != want {
					t.Fatalf("Recovery().Degradation.Reason = %q, want %q", got, want)
				}
			}
		})
	}
}

func TestRetryDoRetriesRateLimitedFailuresAndSucceeds(t *testing.T) {
	t.Parallel()

	attempts := 0
	result, err := RetryDo(context.Background(), RetryConfig{
		Attempts:  3,
		MinDelay:  time.Millisecond,
		MaxDelay:  2 * time.Millisecond,
		Jitter:    0,
		RandFloat: func() float64 { return 0.5 },
	}, func(context.Context) (string, error) {
		attempts++
		if attempts < 3 {
			return "", &RateLimitError{
				Err:        errors.New("slow down"),
				RetryAfter: time.Millisecond,
			}
		}
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("RetryDo() error = %v", err)
	}
	if got, want := result, "ok"; got != want {
		t.Fatalf("RetryDo() result = %q, want %q", got, want)
	}
	if got, want := attempts, 3; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
}

func TestDefaultRetryConfigAndErrorUnwrapHelpers(t *testing.T) {
	t.Parallel()

	config := DefaultRetryConfig()
	if config.Attempts <= 0 || config.MinDelay <= 0 || config.MaxDelay <= 0 {
		t.Fatalf("DefaultRetryConfig() = %#v, want positive retry settings", config)
	}

	root := errors.New("root")
	if !errors.Is((&AuthError{Err: root}).Unwrap(), root) {
		t.Fatal("AuthError.Unwrap() does not expose root error")
	}
	if !errors.Is((&RateLimitError{Err: root}).Unwrap(), root) {
		t.Fatal("RateLimitError.Unwrap() does not expose root error")
	}
	if !errors.Is((&TransientError{Err: root}).Unwrap(), root) {
		t.Fatal("TransientError.Unwrap() does not expose root error")
	}
	if !errors.Is((&PermanentError{Err: root}).Unwrap(), root) {
		t.Fatal("PermanentError.Unwrap() does not expose root error")
	}
}

func TestClassifyErrorCoversHTTPNetAndStringFallbacks(t *testing.T) {
	t.Parallel()

	timeoutNetErr := &net.DNSError{IsTimeout: true}
	otherNetErr := &net.DNSError{Err: "temporary failure"}

	tests := []struct {
		name string
		err  error
		want ErrorClass
	}{
		{
			name: "http auth",
			err:  &HTTPError{StatusCode: http.StatusForbidden, Message: "forbidden"},
			want: ErrorClassAuth,
		},
		{
			name: "http timeout",
			err:  &HTTPError{StatusCode: http.StatusGatewayTimeout, Message: "timed out"},
			want: ErrorClassTimeout,
		},
		{
			name: "http transient",
			err:  &HTTPError{StatusCode: http.StatusServiceUnavailable, Message: "unavailable"},
			want: ErrorClassTransient,
		},
		{
			name: "http permanent",
			err:  &HTTPError{StatusCode: http.StatusBadRequest, Message: "bad request"},
			want: ErrorClassPermanent,
		},
		{name: "net timeout", err: timeoutNetErr, want: ErrorClassTimeout},
		{name: "net transient", err: otherNetErr, want: ErrorClassTransient},
		{name: "string auth", err: errors.New("authentication failed"), want: ErrorClassAuth},
		{name: "string rate limit", err: errors.New("rate limit exceeded"), want: ErrorClassRateLimit},
		{name: "string timeout", err: errors.New("request timeout"), want: ErrorClassTimeout},
		{name: "string transient", err: errors.New("temporary unavailable"), want: ErrorClassTransient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ClassifyError(tt.err).Class; got != tt.want {
				t.Fatalf("ClassifyError().Class = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRetryDoStopsOnPermanentErrorAndHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	t.Run("Should stop immediately on permanent errors", func(t *testing.T) {
		t.Parallel()

		_, err := RetryDo(context.Background(), RetryConfig{
			Attempts: 3,
			MinDelay: time.Millisecond,
			MaxDelay: time.Millisecond,
		}, func(context.Context) (string, error) {
			return "", &PermanentError{Err: errors.New("bad request")}
		})
		if err == nil {
			t.Fatal("RetryDo(permanent) error = nil, want non-nil")
		}
	})

	t.Run("Should preserve context cancellation while waiting to retry", func(t *testing.T) {
		t.Parallel()

		canceled, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := RetryDo(canceled, RetryConfig{
			Attempts: 3,
			MinDelay: time.Millisecond,
			MaxDelay: time.Millisecond,
		}, func(context.Context) (string, error) {
			return "", &RateLimitError{Err: errors.New("slow down"), RetryAfter: time.Millisecond}
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RetryDo(canceled) error = %v, want context.Canceled", err)
		}
		if !strings.Contains(err.Error(), "wait before retry") {
			t.Fatalf("RetryDo(canceled) error = %v, want retry wait context", err)
		}
	})
}

func TestRetryDoAppliesDefaultsAndStopsWhenDelayContextCancels(t *testing.T) {
	t.Parallel()

	t.Run("Should apply default single attempt", func(t *testing.T) {
		t.Parallel()

		_, err := RetryDo(context.Background(), RetryConfig{}, func(context.Context) (string, error) {
			return "", &TransientError{Err: errors.New("temporary failure")}
		})
		if err == nil {
			t.Fatal("RetryDo(default single attempt) error = nil, want transient failure")
		}
	})

	t.Run("Should stop when delay context cancels", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		attempts := 0
		onRetry := 0
		_, err := RetryDo(ctx, RetryConfig{
			Attempts: 2,
			OnRetry: func(int, int, ClassifiedError) {
				onRetry++
				cancel()
			},
		}, func(context.Context) (string, error) {
			attempts++
			return "", &TransientError{Err: errors.New("temporary failure")}
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RetryDo(canceled during delay) error = %v, want context.Canceled", err)
		}
		if attempts != 1 {
			t.Fatalf("attempts = %d, want 1", attempts)
		}
		if onRetry != 1 {
			t.Fatalf("onRetry calls = %d, want 1", onRetry)
		}
	})
}

func TestRetryDelayPrefersRetryAfterAndAppliesBackoff(t *testing.T) {
	t.Parallel()

	config := RetryConfig{
		Attempts: 3,
		MinDelay: 10 * time.Millisecond,
		MaxDelay: 100 * time.Millisecond,
		Jitter:   0,
		RandFloat: func() float64 {
			return 0.5
		},
	}

	for _, tc := range []struct {
		name     string
		attempt  int
		recovery RecoveryDecision
		want     time.Duration
	}{
		{
			name:     "Should prefer Retry-After",
			attempt:  1,
			recovery: RecoveryDecision{RetryAfter: 25 * time.Millisecond},
			want:     25 * time.Millisecond,
		},
		{
			name:    "Should apply exponential backoff",
			attempt: 3,
			want:    40 * time.Millisecond,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := retryDelay(config, tc.attempt, tc.recovery); got != tc.want {
				t.Fatalf("retryDelay() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestErrorHelpersHandleEmptyValues(t *testing.T) {
	t.Parallel()

	t.Run("Should handle empty Error methods", func(t *testing.T) {
		t.Parallel()

		var nilHTTP *HTTPError
		if got := nilHTTP.Error(); got != "" {
			t.Fatalf("nil HTTPError.Error() = %q, want empty string", got)
		}
		if got := (&HTTPError{StatusCode: http.StatusTooManyRequests}).Error(); got == "" {
			t.Fatal("HTTPError.Error() = empty string, want fallback text")
		}
		if got := (&AuthError{}).Error(); got != "" {
			t.Fatalf("AuthError{}.Error() = %q, want empty string", got)
		}
		if got := (&RateLimitError{}).Error(); got != "" {
			t.Fatalf("RateLimitError{}.Error() = %q, want empty string", got)
		}
		if got := (&TransientError{}).Error(); got != "" {
			t.Fatalf("TransientError{}.Error() = %q, want empty string", got)
		}
		if got := (&PermanentError{}).Error(); got != "" {
			t.Fatalf("PermanentError{}.Error() = %q, want empty string", got)
		}
		if got := errorMessage(nil); got != "" {
			t.Fatalf("errorMessage(nil) = %q, want empty string", got)
		}
	})

	t.Run("Should handle nil Unwrap receivers", func(t *testing.T) {
		t.Parallel()

		var nilAuth *AuthError
		if err := nilAuth.Unwrap(); err != nil {
			t.Fatalf("nil AuthError.Unwrap() = %v, want nil", err)
		}
		var nilRateLimit *RateLimitError
		if err := nilRateLimit.Unwrap(); err != nil {
			t.Fatalf("nil RateLimitError.Unwrap() = %v, want nil", err)
		}
		var nilTransient *TransientError
		if err := nilTransient.Unwrap(); err != nil {
			t.Fatalf("nil TransientError.Unwrap() = %v, want nil", err)
		}
		var nilPermanent *PermanentError
		if err := nilPermanent.Unwrap(); err != nil {
			t.Fatalf("nil PermanentError.Unwrap() = %v, want nil", err)
		}
	})

	t.Run("Should classify nil and default errors safely", func(t *testing.T) {
		t.Parallel()

		if got := ClassifyError(nil); got.Class != "" || got.Err != nil {
			t.Fatalf("ClassifyError(nil) = %#v, want zero classification", got)
		}
		if got := ClassifyError(errors.New("domain-specific provider failure")); got.Class != ErrorClassPermanent {
			t.Fatalf("ClassifyError(default) = %q, want permanent", got.Class)
		}
	})
}
