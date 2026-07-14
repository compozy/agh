package network

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/store"
)

func TestFormatNetworkWakePromptPreservesDurableCorrelation(t *testing.T) {
	t.Parallel()

	prompt, meta, err := FormatNetworkWakePrompt([]store.NetworkMessageEntry{{
		MessageID: "message-1", WorkspaceID: "workspace-1", Channel: "builders",
		Surface: store.NetworkSurfaceDirect, DirectID: "direct-1",
		PeerFrom: "session-sender", PeerTo: "session-target",
		Kind: store.NetworkKindSay, WorkID: "work-1", ReplyTo: "message-root",
		TraceID: "trace-1", CausationID: "message-root",
		Mentions: []string{"reviewer.sess-target"},
		Body:     json.RawMessage(`{"text":"Review the patch"}`),
	}}, "session-target")
	if err != nil {
		t.Fatalf("FormatNetworkWakePrompt() error = %v", err)
	}
	if !strings.Contains(prompt, "message-1 from session-sender: Review the patch") {
		t.Fatalf("wake prompt = %q, want compact durable evidence", prompt)
	}
	if meta.MessageID != "message-1" || meta.Kind != store.NetworkKindSay ||
		meta.Channel != "builders" || meta.Surface != store.NetworkSurfaceDirect ||
		meta.DirectID != "direct-1" || meta.From != "session-sender" ||
		meta.To != "session-target" || meta.WorkID != "work-1" ||
		meta.ReplyTo != "message-root" || meta.TraceID != "trace-1" ||
		meta.CausationID != "message-root" {
		t.Fatalf("wake metadata = %#v, want first-message durable correlation", meta)
	}
	if meta.Trust != networkWakeTrustUntrusted || meta.DeliveryMode != store.NetworkWakeTriggerDirect {
		t.Fatalf("wake delivery metadata = %#v, want untrusted direct delivery", meta)
	}
}
