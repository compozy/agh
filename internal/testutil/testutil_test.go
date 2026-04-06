package testutil

import (
	"context"
	"errors"
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := EqualStringSlices(tt.left, tt.right); got != tt.want {
				t.Fatalf("EqualStringSlices(%v, %v) = %v, want %v", tt.left, tt.right, got, tt.want)
			}
		})
	}
}
