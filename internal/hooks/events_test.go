package hooks

import "testing"

const expectedHookEventCount = 27

func TestAllHookEvents(t *testing.T) {
	t.Parallel()

	events := AllHookEvents()
	// Assert the exact count so accidental taxonomy additions/removals are caught explicitly.
	if len(events) != expectedHookEventCount {
		t.Fatalf("len(AllHookEvents()) = %d, want %d", len(events), expectedHookEventCount)
	}

	seen := make(map[HookEvent]struct{}, len(events))
	for _, event := range events {
		if event == "" {
			t.Fatal("AllHookEvents() contains an empty event")
		}
		if err := event.Validate(); err != nil {
			t.Fatalf("event.Validate() error = %v", err)
		}
		if _, ok := seen[event]; ok {
			t.Fatalf("AllHookEvents() contains duplicate event %q", event)
		}
		seen[event] = struct{}{}
	}
}

func TestSyncEligibleClassification(t *testing.T) {
	t.Parallel()

	asyncOnly := map[HookEvent]struct{}{
		HookMessageDelta:       {},
		HookEventPreRecord:     {},
		HookEventPostRecord:    {},
		HookPermissionResolved: {},
		HookPermissionDenied:   {},
	}

	if !HookSessionPreCreate.SyncEligible() {
		t.Fatal("HookSessionPreCreate.SyncEligible() = false, want true")
	}
	if HookMessageDelta.SyncEligible() {
		t.Fatal("HookMessageDelta.SyncEligible() = true, want false")
	}

	for _, event := range AllHookEvents() {
		_, wantAsyncOnly := asyncOnly[event]
		got := event.SyncEligible()
		if wantAsyncOnly && got {
			t.Fatalf("%s.SyncEligible() = true, want false", event)
		}
		if !wantAsyncOnly && !got {
			t.Fatalf("%s.SyncEligible() = false, want true", event)
		}
	}
}

func TestHookEventFamilyAndInvalidValidation(t *testing.T) {
	t.Parallel()

	if got := HookToolPostCall.Family(); got != HookEventFamilyTool {
		t.Fatalf("HookToolPostCall.Family() = %q, want %q", got, HookEventFamilyTool)
	}

	var invalid HookEvent = "nope.invalid"
	if got := invalid.Family(); got != "" {
		t.Fatalf("invalid.Family() = %q, want empty string", got)
	}
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid.Validate() error = nil, want non-nil")
	}
}
