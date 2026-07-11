package session

import (
	"context"
	"testing"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

func TestSessionEventBroadcaster(t *testing.T) {
	t.Parallel()

	t.Run("Should deliver persisted events after cursor and ignore older sequences", func(t *testing.T) {
		t.Parallel()

		manager, err := NewManager(WithHomePaths(testHomePaths(t)))
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		events, cancel, err := manager.SubscribeSessionEvents(testutil.Context(t), "sess-stream", 1)
		if err != nil {
			t.Fatalf("SubscribeSessionEvents() error = %v", err)
		}
		defer cancel()

		manager.publishSessionEvent(testutil.Context(t), &Session{ID: "sess-stream"}, store.SessionEvent{
			SessionID: "sess-stream",
			Sequence:  1,
			TurnID:    "turn-1",
			Type:      "agent_message",
		})
		manager.publishSessionEvent(testutil.Context(t), &Session{ID: "sess-stream"}, store.SessionEvent{
			SessionID: "sess-stream",
			Sequence:  2,
			TurnID:    "turn-1",
			Type:      "agent_message",
		})

		select {
		case event := <-events:
			if event.Sequence != 2 {
				t.Fatalf("event.Sequence = %d, want 2", event.Sequence)
			}
		case <-testutil.Context(t).Done():
			t.Fatal("timed out waiting for stream event")
		}
	})

	t.Run("Should deliver wake events across a sequence reset", func(t *testing.T) {
		t.Parallel()

		manager, err := NewManager(WithHomePaths(testHomePaths(t)))
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		events, cancel, err := manager.SubscribeSessionEventWakes(testutil.Context(t), "sess-wake")
		if err != nil {
			t.Fatalf("SubscribeSessionEventWakes() error = %v", err)
		}
		defer cancel()

		for _, sequence := range []int64{100, 1} {
			manager.publishSessionEvent(testutil.Context(t), &Session{ID: "sess-wake"}, store.SessionEvent{
				SessionID: "sess-wake",
				Sequence:  sequence,
				TurnID:    "turn-wake",
				Type:      "agent_message",
			})
		}

		for _, wantSequence := range []int64{100, 1} {
			select {
			case event := <-events:
				if event.Sequence != wantSequence {
					t.Fatalf("event.Sequence = %d, want %d", event.Sequence, wantSequence)
				}
			case <-testutil.Context(t).Done():
				t.Fatalf("timed out waiting for wake sequence %d", wantSequence)
			}
		}
	})

	t.Run("Should close slow subscriber on overflow", func(t *testing.T) {
		t.Parallel()

		manager, err := NewManager(WithHomePaths(testHomePaths(t)))
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		events, cancel, err := manager.SubscribeSessionEvents(testutil.Context(t), "sess-overflow", 0)
		if err != nil {
			t.Fatalf("SubscribeSessionEvents() error = %v", err)
		}
		defer cancel()

		for sequence := int64(1); sequence <= sessionEventSubscriberBuffer+1; sequence++ {
			manager.publishSessionEvent(context.Background(), &Session{ID: "sess-overflow"}, store.SessionEvent{
				SessionID: "sess-overflow",
				Sequence:  sequence,
				TurnID:    "turn-overflow",
				Type:      "agent_message",
			})
		}

		for range sessionEventSubscriberBuffer {
			<-events
		}
		if _, ok := <-events; ok {
			t.Fatal("events channel remains open after overflow")
		}
	})
}

func testHomePaths(t *testing.T) aghconfig.HomePaths {
	t.Helper()
	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	return homePaths
}
