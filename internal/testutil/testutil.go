// Package testutil provides shared test helpers for internal packages.
package testutil

import (
	"context"
	"testing"
	"time"
)

const defaultTimeout = 10 * time.Second

// Context returns a context canceled during test cleanup.
func Context(t testing.TB) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	t.Cleanup(cancel)
	return ctx
}

// EqualStringSlices reports whether two string slices have equal contents.
func EqualStringSlices(left, right []string) bool {
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
