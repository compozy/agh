package hooks

import "testing"

func TestAllHookEvents(t *testing.T) {
	t.Parallel()

	events := AllHookEvents()
	if len(events) != 27 {
		t.Fatalf("len(AllHookEvents()) = %d, want 27", len(events))
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

	if !SyncEligible(HookSessionPreCreate) {
		t.Fatal("SyncEligible(session.pre_create) = false, want true")
	}
	if SyncEligible(HookMessageDelta) {
		t.Fatal("SyncEligible(message.delta) = true, want false")
	}

	for _, event := range AllHookEvents() {
		_, wantAsyncOnly := asyncOnly[event]
		got := SyncEligible(event)
		if wantAsyncOnly && got {
			t.Fatalf("SyncEligible(%q) = true, want false", event)
		}
		if !wantAsyncOnly && !got {
			t.Fatalf("SyncEligible(%q) = false, want true", event)
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
