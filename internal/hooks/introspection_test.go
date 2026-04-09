package hooks

import (
	"context"
	"testing"
)

func TestHooksCatalogFiltersByWorkspaceAndAgent(t *testing.T) {
	t.Parallel()

	readOnly := true
	hooks := newTestHooks(
		t,
		WithConfigDeclarations([]HookDecl{
			{
				Name:  "matching-session",
				Event: HookSessionPostCreate,
				Mode:  HookModeSync,
				Matcher: HookMatcher{
					AgentName:     "coder",
					WorkspaceID:   "ws-alpha",
					WorkspaceRoot: "/workspace/alpha",
				},
				Command:  "/bin/sh",
				Args:     []string{"-c", "printf '{}'"},
				Metadata: map[string]string{"origin": "test"},
			},
			{
				Name:  "tool-hook",
				Event: HookToolPreCall,
				Mode:  HookModeSync,
				Matcher: HookMatcher{
					ToolReadOnly: &readOnly,
				},
				Command: "/bin/sh",
				Args:    []string{"-c", "printf '{}'"},
			},
			{
				Name:  "other-workspace",
				Event: HookSessionPostCreate,
				Mode:  HookModeSync,
				Matcher: HookMatcher{
					WorkspaceID: "ws-beta",
				},
				Command: "/bin/sh",
				Args:    []string{"-c", "printf '{}'"},
			},
		}),
	)

	if err := hooks.Rebuild(t.Context()); err != nil {
		t.Fatalf("Rebuild() error = %v", err)
	}

	entries, err := hooks.Catalog(CatalogFilter{
		AgentName:     "coder",
		WorkspaceID:   "ws-alpha",
		WorkspaceRoot: "/workspace/alpha",
	})
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	if got, want := len(entries), 2; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}
	if entries[0].Name != "matching-session" {
		t.Fatalf("entries[0].Name = %q, want matching-session", entries[0].Name)
	}
	if entries[0].Metadata["origin"] != "test" {
		t.Fatalf("entries[0].Metadata = %#v", entries[0].Metadata)
	}
	if entries[1].Name != "tool-hook" {
		t.Fatalf("entries[1].Name = %q, want tool-hook", entries[1].Name)
	}
	if entries[1].Matcher.ToolReadOnly == nil || !*entries[1].Matcher.ToolReadOnly {
		t.Fatalf("entries[1].Matcher.ToolReadOnly = %#v, want true", entries[1].Matcher.ToolReadOnly)
	}
}

func TestAllEventDescriptorsReturnsFullTaxonomy(t *testing.T) {
	t.Parallel()

	descriptors := AllEventDescriptors()
	if got, want := len(descriptors), len(AllHookEvents()); got != want {
		t.Fatalf("len(descriptors) = %d, want %d", got, want)
	}

	byEvent := make(map[HookEvent]EventDescriptor, len(descriptors))
	for _, descriptor := range descriptors {
		byEvent[descriptor.Event] = descriptor
	}
	if descriptor := byEvent[HookMessageDelta]; descriptor.SyncEligible {
		t.Fatalf("message.delta SyncEligible = true, want false")
	}
	if descriptor := byEvent[HookPermissionRequest]; !descriptor.SyncEligible {
		t.Fatalf("permission.request SyncEligible = false, want true")
	}
}

func TestHookTelemetryHelpersExposeSessionIDAndSink(t *testing.T) {
	t.Parallel()

	sink := &captureTelemetrySink{}
	hooks := NewHooks(WithTelemetrySink(sink))
	if hooks.telemetrySink != sink {
		t.Fatalf("telemetrySink = %#v, want %#v", hooks.telemetrySink, sink)
	}

	writer := &captureHookRunWriter{}
	ctx := WithHookRunWriter(context.Background(), writer)
	if HookRunWriterFromContext(ctx) != writer {
		t.Fatal("HookRunWriterFromContext() did not return attached writer")
	}

	payload := SessionPostCreatePayload{
		SessionContext: SessionContext{SessionID: "sess-1"},
	}
	if got := sessionIDFromPayload(payload); got != "sess-1" {
		t.Fatalf("sessionIDFromPayload() = %q, want sess-1", got)
	}
}

type captureTelemetrySink struct{}

func (*captureTelemetrySink) WriteHookRecord(context.Context, string, HookRunRecord) error {
	return nil
}
