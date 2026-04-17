package testutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestContextIsCanceledOnCleanup(t *testing.T) {
	t.Parallel()

	var ctx context.Context
	done := make(chan struct{})

	t.Run("subtest", func(t *testing.T) {
		ctx = Context(t)
		go func() {
			<-ctx.Done()
			close(done)
		}()
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Context() was not canceled after cleanup")
	}

	if !errors.Is(ctx.Err(), context.Canceled) && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("Context() err = %v, want canceled or deadline exceeded", ctx.Err())
	}
}

func TestContextCreatedDuringCleanupRemainsUsable(t *testing.T) {
	t.Parallel()

	errCh := make(chan error, 1)

	t.Run("subtest", func(t *testing.T) {
		t.Cleanup(func() {
			ctx := Context(t)
			if err := ctx.Err(); err != nil {
				errCh <- fmt.Errorf("Context() err during cleanup = %v, want nil", err)
				return
			}
			errCh <- nil
		})
	})

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cleanup callback did not report result")
	}
}

func TestEqualStringSlices(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		left  []string
		right []string
		want  bool
	}{
		{name: "both nil", want: true},
		{name: "equal", left: []string{"a", "b"}, right: []string{"a", "b"}, want: true},
		{name: "different length", left: []string{"a"}, right: []string{"a", "b"}, want: false},
		{name: "different value", left: []string{"a", "b"}, right: []string{"a", "c"}, want: false},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := EqualStringSlices(tt.left, tt.right); got != tt.want {
				t.Fatalf("EqualStringSlices(%v, %v) = %v, want %v", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestFreeTCPPort(t *testing.T) {
	t.Parallel()

	port := FreeTCPPort(t)
	if port <= 0 {
		t.Fatalf("FreeTCPPort() = %d, want positive port", port)
	}

	ln, err := (&net.ListenConfig{}).Listen(Context(t), "tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Listen(reused port %d) error = %v", port, err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("ln.Close() error = %v", err)
	}
}
