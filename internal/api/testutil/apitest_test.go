package testutil

import (
	"context"
	"errors"
	"testing"

	"github.com/pedronauck/agh/internal/session"
)

func TestStubSessionManagerListReturnsEmptySliceOnFallbackError(t *testing.T) {
	t.Parallel()

	manager := StubSessionManager{
		ListAllFn: func(context.Context) ([]*session.SessionInfo, error) {
			return nil, errors.New("boom")
		},
	}

	got := manager.List()
	if got == nil {
		t.Fatal("List() = nil, want empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("len(List()) = %d, want 0", len(got))
	}
}
