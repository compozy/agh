package contract_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/apicore"
	"github.com/pedronauck/agh/internal/session"
)

func TestSessionPayloadJSONShape(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC)
	payload := apicore.SessionPayloadFromInfo(&session.SessionInfo{
		ID:           "sess-1",
		Name:         "demo",
		AgentName:    "coder",
		WorkspaceID:  "ws_alpha",
		Workspace:    "/workspace",
		State:        session.StateActive,
		ACPSessionID: "acp-123",
		CreatedAt:    now,
		UpdatedAt:    now,
		ACPCaps: acp.ACPCaps{
			SupportsLoadSession: true,
			SupportedModes:      []string{"chat"},
			SupportedModels:     []string{"gpt-test"},
		},
	})

	var got map[string]any
	marshalJSON(t, payload, &got)

	if got["agent_name"] != "coder" || got["workspace_id"] != "ws_alpha" || got["workspace_path"] != "/workspace" {
		t.Fatalf("session JSON = %#v", got)
	}
	if _, exists := got["acp_session_id"]; !exists {
		t.Fatalf("session JSON missing acp_session_id: %#v", got)
	}
	acpCaps, ok := got["acp_caps"].(map[string]any)
	if !ok {
		t.Fatalf("acp_caps type = %T, want object", got["acp_caps"])
	}
	if acpCaps["supports_load_session"] != true {
		t.Fatalf("acp_caps JSON = %#v", acpCaps)
	}
}

func TestWorkspacePayloadPreservesOmitEmptyBehavior(t *testing.T) {
	t.Parallel()

	payload := contract.WorkspacePayload{
		ID:        "ws_alpha",
		RootDir:   "/workspace",
		AddDirs:   []string{},
		Name:      "alpha",
		CreatedAt: time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 7, 11, 30, 0, 0, time.UTC),
	}

	var got map[string]any
	marshalJSON(t, payload, &got)

	if _, exists := got["default_agent"]; exists {
		t.Fatalf("default_agent should be omitted: %#v", got)
	}
	addDirs, ok := got["add_dirs"].([]any)
	if !ok {
		t.Fatalf("add_dirs type = %T, want array", got["add_dirs"])
	}
	if len(addDirs) != 0 {
		t.Fatalf("add_dirs length = %d, want 0", len(addDirs))
	}
}

func TestAgentEventPayloadRoundTripsThroughJSON(t *testing.T) {
	t.Parallel()

	inputTokens := int64(12)
	event := acp.AgentEvent{
		Type:      acp.EventTypePermission,
		SessionID: "sess-1",
		TurnID:    "turn-1",
		RequestID: "req-1",
		Timestamp: time.Date(2026, 4, 7, 10, 30, 0, 0, time.UTC),
		Action:    "fs/read_text_file",
		Resource:  "/tmp/file.txt",
		Decision:  "pending",
		Error:     "",
		Usage: &acp.TokenUsage{
			TurnID:      "turn-1",
			InputTokens: &inputTokens,
			Timestamp:   time.Date(2026, 4, 7, 10, 30, 1, 0, time.UTC),
		},
		Raw: []byte(`{"ok":true}`),
	}

	payload := apicore.AgentEventPayloadFromEvent(event)
	var roundTrip contract.AgentEventPayload
	marshalJSON(t, payload, &roundTrip)

	if roundTrip.Type != event.Type || roundTrip.RequestID != event.RequestID || roundTrip.Action != event.Action {
		t.Fatalf("roundTrip payload = %#v", roundTrip)
	}
	if roundTrip.Usage == nil || roundTrip.Usage.InputTokens == nil || *roundTrip.Usage.InputTokens != inputTokens {
		t.Fatalf("usage payload = %#v", roundTrip.Usage)
	}
	if string(roundTrip.Raw) != `{"ok":true}` {
		t.Fatalf("raw payload = %s", string(roundTrip.Raw))
	}
}

func marshalJSON[T any](t *testing.T, value any, target *T) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}
