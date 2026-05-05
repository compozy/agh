package hooks

import (
	"context"
	"testing"
	"time"
)

func TestDispatchNetworkHooksUseAsyncObservationPayloads(t *testing.T) {
	t.Parallel()

	events := []HookEvent{
		HookNetworkThreadOpened,
		HookNetworkDirectRoomOpened,
		HookNetworkMessagePersisted,
		HookNetworkWorkOpened,
		HookNetworkWorkTransitioned,
		HookNetworkWorkClosed,
	}
	seen := make(chan HookEvent, len(events))
	decls := make([]HookDecl, 0, len(events))
	executors := make(map[string]Executor, len(events))
	for _, event := range events {
		name := event.String()
		decls = append(decls, HookDecl{
			Name:         name,
			Event:        event,
			Mode:         HookModeAsync,
			Matcher:      HookMatcher{NetworkMatcher: &NetworkMatcher{Channel: "builders", Surface: "thread"}},
			ExecutorKind: HookExecutorNative,
		})
		executors[name] = NewTypedNativeExecutor(
			func(_ context.Context, _ RegisteredHook, payload NetworkPayload) (NetworkObservationPatch, error) {
				seen <- payload.Event
				return NetworkObservationPatch{}, nil
			},
		)
	}
	hooks := newTestHooks(
		t,
		WithNativeDeclarations(decls),
		WithExecutorResolver(testExecutorResolver(executors)),
		WithAsyncWorkerCount(1),
		WithAsyncQueueCapacity(len(events)),
	)
	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	for _, event := range events {
		payload := networkDispatchTestPayload(event)
		var err error
		switch event {
		case HookNetworkThreadOpened:
			_, err = hooks.DispatchNetworkThreadOpened(t.Context(), payload)
		case HookNetworkDirectRoomOpened:
			_, err = hooks.DispatchNetworkDirectRoomOpened(t.Context(), payload)
		case HookNetworkMessagePersisted:
			_, err = hooks.DispatchNetworkMessagePersisted(t.Context(), payload)
		case HookNetworkWorkOpened:
			_, err = hooks.DispatchNetworkWorkOpened(t.Context(), payload)
		case HookNetworkWorkTransitioned:
			_, err = hooks.DispatchNetworkWorkTransitioned(t.Context(), payload)
		case HookNetworkWorkClosed:
			_, err = hooks.DispatchNetworkWorkClosed(t.Context(), payload)
		}
		if err != nil {
			t.Fatalf("dispatch %s error = %v", event, err)
		}
	}

	for _, want := range events {
		select {
		case got := <-seen:
			if got != want {
				t.Fatalf("async network hook event = %q, want %q", got, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for async network hook %q", want)
		}
	}
}

func networkDispatchTestPayload(event HookEvent) NetworkPayload {
	return NetworkPayload{
		PayloadBase: PayloadBase{
			Event:     event,
			Timestamp: time.Date(2026, time.May, 5, 12, 0, 0, 0, time.UTC),
		},
		SessionID:   "sess-coder",
		Channel:     "builders",
		Surface:     "thread",
		ThreadID:    "thread_hooks_01",
		MessageID:   "msg-hook",
		Kind:        "trace",
		Direction:   "received",
		WorkID:      "work_hooks_01",
		WorkState:   "completed",
		PeerFrom:    "coder.sess-abc",
		PeerTo:      "reviewer.sess-xyz",
		TraceID:     "trace-hooks-01",
		CausationID: "cause-hooks-01",
	}
}
