package bridgesdk

import (
	"context"
	"testing"
)

func TestRetryDoRefacs(t *testing.T) {
	t.Parallel()

	t.Run("Should reject nil context before invoking operation", func(t *testing.T) {
		t.Parallel()

		called := false
		_, err := RetryDo(nilRetryContext(), RetryConfig{}, func(context.Context) (string, error) {
			called = true
			return "unexpected", nil
		})
		if err == nil {
			t.Fatal("RetryDo(nil context) error = nil, want non-nil")
		}
		if called {
			t.Fatal("operation called = true, want false")
		}
	})

	t.Run("Should reject nil operation", func(t *testing.T) {
		t.Parallel()

		_, err := RetryDo[string](context.Background(), RetryConfig{}, nil)
		if err == nil {
			t.Fatal("RetryDo(nil operation) error = nil, want non-nil")
		}
	})
}

func nilRetryContext() context.Context {
	return nil
}
