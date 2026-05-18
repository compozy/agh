package httpapi

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	core "github.com/pedronauck/agh/internal/api/core"
)

func TestDrainPromptEventsAsyncClawpatch(t *testing.T) {
	t.Parallel()

	t.Run("Should not cancel prompt when HTTP drain context is canceled before events close", func(t *testing.T) {
		t.Parallel()

		streamDone := make(chan struct{})
		handlers := &Handlers{
			BaseHandlers: core.NewBaseHandlers(&core.BaseHandlerConfig{StreamDone: streamDone}),
		}
		events := make(chan acp.AgentEvent)
		promptCanceled := make(chan struct{})
		var cancelOnce sync.Once
		drainCtx, cancelDrain := context.WithCancel(context.Background())
		handlers.drainPromptEventsAsync(drainCtx, events, func() {
			cancelOnce.Do(func() {
				close(promptCanceled)
			})
		})

		cancelDrain()
		assertPromptNotCanceled(t, promptCanceled, 50*time.Millisecond)
		close(events)

		waitCtx, cancelWait := context.WithTimeout(context.Background(), time.Second)
		defer cancelWait()
		if err := handlers.waitForPromptDrains(waitCtx); err != nil {
			t.Fatalf("waitForPromptDrains() error = %v", err)
		}
		assertPromptCanceled(t, promptCanceled, time.Second)
	})
}

func assertPromptNotCanceled(t *testing.T, promptCanceled <-chan struct{}, timeout time.Duration) {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-promptCanceled:
		t.Fatal("prompt canceled before drained events channel closed")
	case <-timer.C:
	}
}

func assertPromptCanceled(t *testing.T, promptCanceled <-chan struct{}, timeout time.Duration) {
	t.Helper()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-promptCanceled:
	case <-timer.C:
		t.Fatal("prompt cancel was not called after drained events channel closed")
	}
}
