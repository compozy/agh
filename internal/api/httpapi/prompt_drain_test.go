package httpapi

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/acp"
	core "github.com/compozy/agh/internal/api/core"
)

func TestDrainPromptEventsAsyncContract(t *testing.T) {
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

	t.Run("Should preserve an existing request deadline when detaching prompt drains", func(t *testing.T) {
		t.Parallel()

		wantDeadline := time.Now().Add(2 * time.Second).Round(0)
		ctx, cancel := context.WithDeadline(context.Background(), wantDeadline)
		defer cancel()

		drainCtx, cancelDrain := detachPromptDrainContext(ctx)
		defer cancelDrain()

		gotDeadline, ok := drainCtx.Deadline()
		if !ok {
			t.Fatal("detachPromptDrainContext() removed the request deadline")
		}
		if !gotDeadline.Equal(wantDeadline) {
			t.Fatalf("drain deadline = %v, want %v", gotDeadline, wantDeadline)
		}
	})

	t.Run("Should attach a fallback timeout when no request deadline exists", func(t *testing.T) {
		t.Parallel()

		drainCtx, cancelDrain := detachPromptDrainContext(context.Background())
		defer cancelDrain()

		deadline, ok := drainCtx.Deadline()
		if !ok {
			t.Fatal("detachPromptDrainContext() did not attach a fallback timeout")
		}
		if remaining := time.Until(deadline); remaining <= 0 || remaining > detachedPromptDrainTimeout+time.Second {
			t.Fatalf("fallback timeout remaining = %v, want bounded positive duration", remaining)
		}
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
