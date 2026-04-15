// Package testutil provides shared test helpers for internal packages.
package testutil

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

const defaultTimeout = 10 * time.Second

var tcpPortCounter atomic.Uint32

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

// FreeTCPPort returns an available localhost TCP port chosen from a
// per-process pseudo-random walk through a high, non-default port range. The
// listener is closed before returning, so callers still need to bind quickly,
// but this avoids repeated reuse of the same ephemeral port under parallel
// package execution.
func FreeTCPPort(t testing.TB) int {
	t.Helper()

	const (
		minPort     = 20000
		portSpan    = 40000
		maxAttempts = portSpan
	)

	start := int((uint64(os.Getpid())*131 + uint64(tcpPortCounter.Add(1))) % portSpan)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		port := minPort + ((start + attempt) % portSpan)
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			continue
		}
		if err := ln.Close(); err != nil {
			t.Fatalf("net.Listener.Close(%d) error = %v", port, err)
		}
		return port
	}

	t.Fatal("FreeTCPPort() exhausted candidate range")
	return 0
}
