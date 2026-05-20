package transcript

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/diagnostics"
	"github.com/pedronauck/agh/internal/store"
)

func TestAssembleLegacyACPEvents(t *testing.T) {
	t.Parallel()

	events := []store.SessionEvent{
		{
			ID:        "ev-1",
			Sequence:  1,
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeThought,
			Content:   `{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":"Thinking "}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:        "ev-2",
			Sequence:  2,
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeThought,
			Content:   `{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":"hard"}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
		},
		{
			ID:        "ev-3",
			Sequence:  3,
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeAgentMessage,
			Content:   `{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"Let me read "}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 2, 0, time.UTC),
		},
		{
			ID:        "ev-4",
			Sequence:  4,
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeAgentMessage,
			Content:   `{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"the file"}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 3, 0, time.UTC),
		},
		{
			ID:        "ev-5",
			Sequence:  5,
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeToolCall,
			Content:   `{"_meta":{"claudeCode":{"toolName":"Read"}},"toolCallId":"call-1","sessionUpdate":"tool_call","rawInput":{},"status":"pending","title":"Read File","kind":"read","content":[]}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 4, 0, time.UTC),
		},
		{
			ID:        "ev-6",
			Sequence:  6,
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeToolCall,
			Content:   `{"_meta":{"claudeCode":{"toolName":"Read"}},"toolCallId":"call-1","sessionUpdate":"tool_call_update","rawInput":{"file_path":"/tmp/demo.txt"},"status":"in_progress","title":"Read /tmp/demo.txt","kind":"read","content":[]}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 5, 0, time.UTC),
		},
		{
			ID:        "ev-7",
			Sequence:  7,
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeToolResult,
			Content:   `{"_meta":{"claudeCode":{"toolName":"Read"}},"toolCallId":"call-1","sessionUpdate":"tool_call_update","status":"completed","rawOutput":"line1\nline2","content":[{"type":"content","content":{"type":"text","text":"line1\nline2"}}]}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 6, 0, time.UTC),
		},
		{
			ID:        "ev-8",
			Sequence:  8,
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeAgentMessage,
			Content:   `{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"Done."}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 7, 0, time.UTC),
		},
	}

	messages, err := Assemble(events)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("Assemble() len = %d, want 4", len(messages))
	}

	if got := messages[0].Role; got != RoleAssistant {
		t.Fatalf("messages[0].Role = %q, want %q", got, RoleAssistant)
	}
	if got := messages[0].Thinking; got != "Thinking hard" {
		t.Fatalf("messages[0].Thinking = %q, want %q", got, "Thinking hard")
	}
	if got := messages[0].Content; got != "Let me read the file" {
		t.Fatalf("messages[0].Content = %q, want %q", got, "Let me read the file")
	}
	if !messages[0].ThinkingComplete {
		t.Fatal("messages[0].ThinkingComplete = false, want true")
	}

	if got := messages[1].Role; got != RoleToolCall {
		t.Fatalf("messages[1].Role = %q, want %q", got, RoleToolCall)
	}
	if got := messages[1].ToolName; got != "Read" {
		t.Fatalf("messages[1].ToolName = %q, want %q", got, "Read")
	}
	if got := string(messages[1].ToolInput); got != `{"file_path":"/tmp/demo.txt"}` {
		t.Fatalf("messages[1].ToolInput = %s", got)
	}

	if got := messages[2].Role; got != RoleToolResult {
		t.Fatalf("messages[2].Role = %q, want %q", got, RoleToolResult)
	}
	if messages[2].ToolResult == nil || messages[2].ToolResult.Content != "line1\nline2" {
		t.Fatalf("messages[2].ToolResult = %#v, want content", messages[2].ToolResult)
	}
	if messages[2].ToolError {
		t.Fatal("messages[2].ToolError = true, want false")
	}

	if got := messages[3].Role; got != RoleAssistant {
		t.Fatalf("messages[3].Role = %q, want %q", got, RoleAssistant)
	}
	if got := messages[3].Content; got != "Done." {
		t.Fatalf("messages[3].Content = %q, want %q", got, "Done.")
	}
}

func TestAssembleReadsCanonicalEnvelopeAndStableOrdering(t *testing.T) {
	t.Parallel()

	events := []store.SessionEvent{
		{
			ID:       "b",
			Sequence: 3,
			TurnID:   "turn-canonical",
			Type:     acp.EventTypeToolCall,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeToolCall,
				"turn-canonical",
				time.Date(2026, 4, 3, 13, 0, 2, 0, time.UTC),
				"",
				"Bash",
				"call-2",
				json.RawMessage(`{"command":"ls -la"}`),
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 3, 13, 0, 2, 0, time.UTC),
		},
		{
			ID:       "a",
			Sequence: 1,
			TurnID:   "turn-canonical",
			Type:     acp.EventTypeUserMessage,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeUserMessage,
				"turn-canonical",
				time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
				"list files",
				"",
				"",
				nil,
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
		},
		{
			ID:       "c",
			Sequence: 4,
			TurnID:   "turn-canonical",
			Type:     acp.EventTypeToolResult,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeToolResult,
				"turn-canonical",
				time.Date(2026, 4, 3, 13, 0, 3, 0, time.UTC),
				"",
				"Bash",
				"call-2",
				nil,
				&ToolResult{Stdout: "ok"},
				false,
			),
			Timestamp: time.Date(2026, 4, 3, 13, 0, 3, 0, time.UTC),
		},
		{
			ID:       "d",
			Sequence: 2,
			TurnID:   "turn-canonical",
			Type:     acp.EventTypeAgentMessage,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeAgentMessage,
				"turn-canonical",
				time.Date(2026, 4, 3, 13, 0, 1, 0, time.UTC),
				"Listing files",
				"",
				"",
				nil,
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 3, 13, 0, 1, 0, time.UTC),
		},
	}

	messages, err := Assemble(events)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("Assemble() len = %d, want 4", len(messages))
	}

	if got := messages[0].Role; got != RoleUser {
		t.Fatalf("messages[0].Role = %q, want %q", got, RoleUser)
	}
	if got := messages[0].Content; got != "list files" {
		t.Fatalf("messages[0].Content = %q, want %q", got, "list files")
	}
	if got := messages[2].ToolName; got != "Bash" {
		t.Fatalf("messages[2].ToolName = %q, want %q", got, "Bash")
	}
	if got := string(messages[2].ToolInput); got != `{"command":"ls -la"}` {
		t.Fatalf("messages[2].ToolInput = %s", got)
	}
	if messages[3].ToolResult == nil || messages[3].ToolResult.Stdout != "ok" {
		t.Fatalf("messages[3].ToolResult = %#v, want stdout ok", messages[3].ToolResult)
	}
}

func TestAssembleRendersSyntheticReentryAsSystemMessage(t *testing.T) {
	t.Parallel()

	events := []store.SessionEvent{
		{
			ID:       "user-1",
			Sequence: 1,
			TurnID:   "turn-user",
			Type:     acp.EventTypeUserMessage,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeUserMessage,
				"turn-user",
				time.Date(2026, 4, 18, 11, 0, 0, 0, time.UTC),
				"human prompt",
				"",
				"",
				nil,
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 18, 11, 0, 0, 0, time.UTC),
		},
		{
			ID:       "synth-1",
			Sequence: 2,
			TurnID:   "turn-synth",
			Type:     acp.EventTypeSyntheticReentry,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeSyntheticReentry,
				"turn-synth",
				time.Date(2026, 4, 18, 11, 0, 1, 0, time.UTC),
				"daemon wake-up",
				"",
				"",
				nil,
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 18, 11, 0, 1, 0, time.UTC),
		},
	}

	messages, err := Assemble(events)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("Assemble() len = %d, want 2", len(messages))
	}
	if got := messages[0].Role; got != RoleUser {
		t.Fatalf("messages[0].Role = %q, want %q", got, RoleUser)
	}
	if got := messages[1].Role; got != RoleSystem {
		t.Fatalf("messages[1].Role = %q, want %q", got, RoleSystem)
	}
	if got := messages[1].Content; got != "daemon wake-up" {
		t.Fatalf("messages[1].Content = %q, want %q", got, "daemon wake-up")
	}
}

func TestAssemblePreservesMixedTurnOrderingAndToolPairingAcrossTurns(t *testing.T) {
	t.Parallel()

	events := []store.SessionEvent{
		{
			ID:       "user-1",
			Sequence: 1,
			TurnID:   "turn-user",
			Type:     acp.EventTypeUserMessage,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeUserMessage,
				"turn-user",
				time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC),
				"user prompt",
				"",
				"",
				nil,
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:       "call-user",
			Sequence: 2,
			TurnID:   "turn-user",
			Type:     acp.EventTypeToolCall,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeToolCall,
				"turn-user",
				time.Date(2026, 4, 18, 12, 0, 1, 0, time.UTC),
				"",
				"Bash",
				"call-user",
				json.RawMessage(`{"command":"echo user"}`),
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 18, 12, 0, 1, 0, time.UTC),
		},
		{
			ID:       "synth-1",
			Sequence: 3,
			TurnID:   "turn-synth",
			Type:     acp.EventTypeSyntheticReentry,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeSyntheticReentry,
				"turn-synth",
				time.Date(2026, 4, 18, 12, 0, 2, 0, time.UTC),
				"daemon wake-up",
				"",
				"",
				nil,
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 18, 12, 0, 2, 0, time.UTC),
		},
		{
			ID:       "result-user",
			Sequence: 4,
			TurnID:   "turn-user",
			Type:     acp.EventTypeToolResult,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeToolResult,
				"turn-user",
				time.Date(2026, 4, 18, 12, 0, 3, 0, time.UTC),
				"",
				"Bash",
				"call-user",
				nil,
				&ToolResult{Stdout: "user"},
				false,
			),
			Timestamp: time.Date(2026, 4, 18, 12, 0, 3, 0, time.UTC),
		},
		{
			ID:       "network-1",
			Sequence: 5,
			TurnID:   "turn-network",
			Type:     acp.EventTypeUserMessage,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeUserMessage,
				"turn-network",
				time.Date(2026, 4, 18, 12, 0, 4, 0, time.UTC),
				"network prompt",
				"",
				"",
				nil,
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 18, 12, 0, 4, 0, time.UTC),
		},
		{
			ID:       "call-network",
			Sequence: 6,
			TurnID:   "turn-network",
			Type:     acp.EventTypeToolCall,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeToolCall,
				"turn-network",
				time.Date(2026, 4, 18, 12, 0, 5, 0, time.UTC),
				"",
				"Bash",
				"call-network",
				json.RawMessage(`{"command":"echo network"}`),
				nil,
				false,
			),
			Timestamp: time.Date(2026, 4, 18, 12, 0, 5, 0, time.UTC),
		},
		{
			ID:       "result-network",
			Sequence: 7,
			TurnID:   "turn-network",
			Type:     acp.EventTypeToolResult,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeToolResult,
				"turn-network",
				time.Date(2026, 4, 18, 12, 0, 6, 0, time.UTC),
				"",
				"Bash",
				"call-network",
				nil,
				&ToolResult{Stdout: "network"},
				false,
			),
			Timestamp: time.Date(2026, 4, 18, 12, 0, 6, 0, time.UTC),
		},
	}

	messages, err := Assemble(events)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if len(messages) != 7 {
		t.Fatalf("Assemble() len = %d, want 7", len(messages))
	}

	if got := messages[0].Role; got != RoleUser {
		t.Fatalf("messages[0].Role = %q, want %q", got, RoleUser)
	}
	if got := messages[0].Content; got != "user prompt" {
		t.Fatalf("messages[0].Content = %q, want %q", got, "user prompt")
	}
	if got := messages[1].Role; got != RoleToolCall {
		t.Fatalf("messages[1].Role = %q, want %q", got, RoleToolCall)
	}
	if got := messages[2].Role; got != RoleSystem {
		t.Fatalf("messages[2].Role = %q, want %q", got, RoleSystem)
	}
	if got := messages[2].Content; got != "daemon wake-up" {
		t.Fatalf("messages[2].Content = %q, want %q", got, "daemon wake-up")
	}
	if got := messages[3].Role; got != RoleToolResult {
		t.Fatalf("messages[3].Role = %q, want %q", got, RoleToolResult)
	}
	if got := messages[4].Role; got != RoleUser {
		t.Fatalf("messages[4].Role = %q, want %q", got, RoleUser)
	}
	if got := messages[4].Content; got != "network prompt" {
		t.Fatalf("messages[4].Content = %q, want %q", got, "network prompt")
	}
	if got := messages[5].Role; got != RoleToolCall {
		t.Fatalf("messages[5].Role = %q, want %q", got, RoleToolCall)
	}
	if got := messages[6].Role; got != RoleToolResult {
		t.Fatalf("messages[6].Role = %q, want %q", got, RoleToolResult)
	}
	if got, want := messages[1].ID, "call-user"; got != want {
		t.Fatalf("messages[1].ID = %q, want %q", got, want)
	}
	if got, want := messages[3].ID, "call-user"; got != want {
		t.Fatalf("messages[3].ID = %q, want %q", got, want)
	}
	if got, want := messages[5].ID, "call-network"; got != want {
		t.Fatalf("messages[5].ID = %q, want %q", got, want)
	}
	if got, want := messages[6].ID, "call-network"; got != want {
		t.Fatalf("messages[6].ID = %q, want %q", got, want)
	}
	if messages[3].ToolResult == nil || messages[3].ToolResult.Stdout != "user" {
		t.Fatalf("messages[3].ToolResult = %#v, want stdout user", messages[3].ToolResult)
	}
	if messages[6].ToolResult == nil || messages[6].ToolResult.Stdout != "network" {
		t.Fatalf("messages[6].ToolResult = %#v, want stdout network", messages[6].ToolResult)
	}
}

func TestAssembleSkipsIgnorableEvents(t *testing.T) {
	t.Parallel()

	events := []store.SessionEvent{
		{
			ID:        "ev-empty-1",
			Sequence:  1,
			Type:      acp.EventTypeAgentMessage,
			Content:   `{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"   "}}`,
			Timestamp: time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC),
		},
		{
			ID:        "ev-empty-2",
			Sequence:  2,
			Type:      acp.EventTypeThought,
			Content:   `{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":" "}}`,
			Timestamp: time.Date(2026, 4, 3, 14, 0, 1, 0, time.UTC),
		},
		{
			ID:        "ev-empty-3",
			Sequence:  3,
			Type:      acp.EventTypeUserMessage,
			Content:   "",
			Timestamp: time.Date(2026, 4, 3, 14, 0, 2, 0, time.UTC),
		},
	}

	messages, err := Assemble(events)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("Assemble() len = %d, want 0", len(messages))
	}
}

func TestAssemblePairsToolLifecycleWhenResultOmitsTurnID(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2026, 4, 18, 16, 0, 0, 0, time.UTC)
	events := []store.SessionEvent{
		{
			ID:       "call-1",
			Sequence: 1,
			TurnID:   "turn-tool",
			Type:     acp.EventTypeToolCall,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeToolCall,
				"turn-tool",
				timestamp,
				"",
				"Bash",
				"shared-call",
				json.RawMessage(`{"command":"pwd"}`),
				nil,
				false,
			),
			Timestamp: timestamp,
		},
		{
			ID:       "result-1",
			Sequence: 2,
			Type:     acp.EventTypeToolResult,
			Content: mustMarshalCanonical(
				t,
				acp.EventTypeToolResult,
				"",
				timestamp.Add(time.Second),
				"",
				"Bash",
				"shared-call",
				nil,
				&ToolResult{Stdout: "workspace"},
				false,
			),
			Timestamp: timestamp.Add(time.Second),
		},
	}

	messages, err := Assemble(events)
	if err != nil {
		t.Fatalf("Assemble() error = %v", err)
	}
	if got, want := len(messages), 2; got != want {
		t.Fatalf("Assemble() len = %d, want %d", got, want)
	}
	if got, want := messages[0].Role, RoleToolCall; got != want {
		t.Fatalf("messages[0].Role = %q, want %q", got, want)
	}
	if got, want := messages[1].Role, RoleToolResult; got != want {
		t.Fatalf("messages[1].Role = %q, want %q", got, want)
	}
	if got, want := messages[0].ID, "shared-call"; got != want {
		t.Fatalf("messages[0].ID = %q, want %q", got, want)
	}
	if got, want := messages[1].ID, "shared-call"; got != want {
		t.Fatalf("messages[1].ID = %q, want %q", got, want)
	}
	if messages[1].ToolResult == nil || messages[1].ToolResult.Stdout != "workspace" {
		t.Fatalf("messages[1].ToolResult = %#v, want stdout workspace", messages[1].ToolResult)
	}
}

func TestParseLooseEventBuildsToolResultFromLoosePayload(t *testing.T) {
	t.Parallel()

	parsed := parseLooseEvent(event{Type: acp.EventTypeToolResult}, map[string]any{
		"type":         acp.EventTypeToolResult,
		"tool_call_id": "call-loose",
		"title":        "Bash",
		"rawInput": map[string]any{
			"command": "pwd",
		},
		"rawOutput": map[string]any{
			"stdout": "workspace\n",
		},
	})

	if got := parsed.ToolCallID; got != "call-loose" {
		t.Fatalf("ToolCallID = %q, want %q", got, "call-loose")
	}
	if got := parsed.ToolName; got != "Bash" {
		t.Fatalf("ToolName = %q, want %q", got, "Bash")
	}
	if got := string(parsed.ToolInput); got != `{"command":"pwd"}` {
		t.Fatalf("ToolInput = %s, want JSON command payload", got)
	}
	if parsed.ToolResult == nil {
		t.Fatal("ToolResult = nil, want populated result")
	}
	if got := parsed.ToolResult.Stdout; got != "workspace\n" {
		t.Fatalf("ToolResult.Stdout = %q, want %q", got, "workspace\n")
	}
	if parsed.ToolError {
		t.Fatal("ToolError = true, want false")
	}

	if got := string(firstNonEmptyRaw(nil, json.RawMessage(`{"ok":true}`))); got != `{"ok":true}` {
		t.Fatalf("firstNonEmptyRaw() = %s, want non-empty raw payload", got)
	}
	if got := firstNonNil(nil, "", "value"); got != "" {
		t.Fatalf("firstNonNil(nil, \"\", \"value\") = %#v, want empty string first", got)
	}
}

func TestMarshalAgentEventBuildsCanonicalPayload(t *testing.T) {
	t.Parallel()

	totalTokens := int64(4)
	payload, err := MarshalAgentEvent(acp.AgentEvent{
		Type:      acp.EventTypeDone,
		SessionID: "acp-1",
		TurnID:    "turn-1",
		Timestamp: time.Date(2026, 4, 3, 15, 0, 0, 0, time.UTC),
		Text:      "done",
		Error:     "none",
		Usage: &acp.TokenUsage{
			TurnID:      "turn-1",
			TotalTokens: &totalTokens,
			Timestamp:   time.Date(2026, 4, 3, 15, 0, 1, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("MarshalAgentEvent(structured) error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if decoded["schema"] != CanonicalSchema {
		t.Fatalf("decoded[schema] = %v, want %q", decoded["schema"], CanonicalSchema)
	}
	if decoded["type"] != acp.EventTypeDone {
		t.Fatalf("decoded[type] = %v, want %q", decoded["type"], acp.EventTypeDone)
	}
	if decoded["text"] != "done" {
		t.Fatalf("decoded[text] = %v, want %q", decoded["text"], "done")
	}
}

func TestMarshalAgentEventExtractsToolResultShapeWithoutPersistingRaw(t *testing.T) {
	t.Run("Should extract structured tool result fields without persisting raw payloads", func(t *testing.T) {
		t.Parallel()

		payload, err := MarshalAgentEvent(acp.AgentEvent{
			Type: acp.EventTypeToolResult,
			Raw: json.RawMessage(`{
				"sessionUpdate":"tool_call_update",
				"status":"failed",
				"rawOutput":{"stderr":"boom"},
				"content":[{"type":"content","content":{"type":"text","text":"boom"}}],
				"_meta":{"claudeCode":{"toolName":"Bash"}},
				"rawInput":{"command":"pwd"}
			}`),
			Title: "tool result",
		})
		if err != nil {
			t.Fatalf("MarshalAgentEvent(raw) error = %v", err)
		}

		var decoded struct {
			Schema     string          `json:"schema"`
			ToolName   string          `json:"tool_name"`
			ToolInput  json.RawMessage `json:"tool_input"`
			ToolError  bool            `json:"tool_error"`
			ToolResult ToolResult      `json:"tool_result"`
			Raw        json.RawMessage `json:"raw"`
		}
		if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
			t.Fatalf("json.Unmarshal(raw payload) error = %v", err)
		}
		if decoded.Schema != CanonicalSchema {
			t.Fatalf("Schema = %q, want %q", decoded.Schema, CanonicalSchema)
		}
		if decoded.ToolName != "Bash" {
			t.Fatalf("ToolName = %q, want %q", decoded.ToolName, "Bash")
		}
		if got := string(decoded.ToolInput); got != `{"command":"pwd"}` {
			t.Fatalf("ToolInput = %s, want command payload", got)
		}
		if !decoded.ToolError {
			t.Fatal("ToolError = false, want true")
		}
		if decoded.ToolResult.Stderr != "boom" || decoded.ToolResult.Error != "boom" {
			t.Fatalf("ToolResult = %#v, want stderr/error boom", decoded.ToolResult)
		}
		if len(decoded.Raw) != 0 {
			t.Fatalf("Raw = %s, want empty persisted raw payload", string(decoded.Raw))
		}
	})
}

func TestTranscriptRedactsSecretsAcrossDisplaySurfaces(t *testing.T) {
	t.Parallel()

	runtimeSecret := "sk-transcript-display-secret-123456"
	cleanup := diagnostics.RegisterDynamicSecret(runtimeSecret)
	t.Cleanup(cleanup)

	t.Run("Should redact live event payloads before UI projection", func(t *testing.T) {
		t.Parallel()

		leaks := []string{
			runtimeSecret,
			"text-secret",
			"agh_claim_live_123",
			"bearer-secret",
			"failure-secret",
			"runtime-secret",
			"raw-secret",
		}
		event := acp.AgentEvent{
			Type:      acp.EventTypeAgentMessage,
			SessionID: "sess-redact",
			TurnID:    "turn-redact",
			Timestamp: time.Date(2026, 5, 19, 10, 0, 0, 0, time.UTC),
			Text:      "assistant stdout " + runtimeSecret + " token=text-secret agh_claim_live_123",
			Error:     "Bearer bearer-secret",
			Failure: &store.SessionFailure{
				Kind:    store.FailurePrompt,
				Summary: "secret_binding=failure-secret",
			},
			Runtime: &acp.RuntimeActivity{
				LastActivityDetail: "token=runtime-secret",
			},
			Raw: json.RawMessage(`{"access_token":"raw-secret","note":"` + runtimeSecret + `"}`),
		}

		assertNoDisplayLeaks(t, RedactAgentEvent(event), leaks)
		assertNoDisplayLeaks(t, UIAgentEventPayloadFromEvent(event), leaks)
	})

	t.Run("Should redact stored tool output before transcript and chat replay", func(t *testing.T) {
		t.Parallel()

		leaks := []string{
			runtimeSecret,
			"stdout-secret",
			"stderr-secret",
			"content-secret",
			"raw-binding",
			"raw-secret",
			"input-secret",
			"agh_claim_tool_123",
		}
		payload, err := MarshalAgentEvent(acp.AgentEvent{
			Type:      acp.EventTypeToolResult,
			SessionID: "sess-redact",
			TurnID:    "turn-redact",
			Timestamp: time.Date(2026, 5, 19, 10, 0, 1, 0, time.UTC),
			Title:     "Bash",
			Raw: json.RawMessage(`{
				"sessionUpdate":"tool_call_update",
				"status":"completed",
				"rawOutput":{
					"stdout":"runtime ` + runtimeSecret + ` token=stdout-secret agh_claim_tool_123",
					"stderr":"Bearer stderr-secret",
					"content":"secret_binding=raw-binding",
					"api_key":"raw-secret"
				},
				"content":[{"type":"content","content":{"type":"text","text":"token=content-secret ` + runtimeSecret + `"}}],
				"_meta":{"claudeCode":{"toolName":"Bash"}},
				"rawInput":{"api_key":"input-secret","command":"echo ok"}
			}`),
		})
		if err != nil {
			t.Fatalf("MarshalAgentEvent() error = %v", err)
		}
		assertNoDisplayLeaks(t, payload, leaks)

		events := []store.SessionEvent{{
			ID:        "ev-redact-tool",
			SessionID: "sess-redact",
			TurnID:    "turn-redact",
			Sequence:  1,
			Type:      acp.EventTypeToolResult,
			Content:   payload,
			Timestamp: time.Date(2026, 5, 19, 10, 0, 1, 0, time.UTC),
		}}
		transcriptMessages, err := Assemble(events)
		if err != nil {
			t.Fatalf("Assemble() error = %v", err)
		}
		assertNoDisplayLeaks(t, transcriptMessages, leaks)

		uiMessages, err := ToUIMessages(events)
		if err != nil {
			t.Fatalf("ToUIMessages() error = %v", err)
		}
		assertNoDisplayLeaks(t, uiMessages, leaks)
	})
}

func TestTranscriptRuntimeMarkers(t *testing.T) {
	t.Parallel()

	timestamp := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		event acp.AgentEvent
		want  string
	}{
		{
			name: "Should render timeout marker from runtime warning",
			event: acp.AgentEvent{
				Type:      acp.EventTypeRuntimeWarning,
				SessionID: "sess-marker",
				TurnID:    "turn-timeout",
				Timestamp: timestamp,
				Text:      "Runtime activity timed out (30 seconds idle).",
				Runtime: &acp.RuntimeActivity{
					LastActivityKind:   "timeout",
					LastActivityDetail: "Runtime activity timed out (30 seconds idle).",
				},
			},
			want: "*[timeout]* Runtime activity timed out (30 seconds idle).",
		},
		{
			name: "Should render unhealthy marker from runtime warning",
			event: acp.AgentEvent{
				Type:      acp.EventTypeRuntimeWarning,
				SessionID: "sess-marker",
				TurnID:    "turn-unhealthy",
				Timestamp: timestamp.Add(time.Second),
				Text:      "Runtime health check failed; prompt may be stalled.",
				Runtime: &acp.RuntimeActivity{
					LastActivityKind:   "warning",
					LastActivityDetail: string(store.SessionStallReasonProcessUnhealthy),
				},
			},
			want: "*[unhealthy]* Runtime health check failed; prompt may be stalled.",
		},
		{
			name: "Should render interrupted marker from session stopped",
			event: acp.AgentEvent{
				Type:       sessionStoppedEventType,
				SessionID:  "sess-marker",
				TurnID:     "turn-interrupt",
				Timestamp:  timestamp.Add(2 * time.Second),
				StopReason: string(store.StopUserCanceled),
				Failure: &store.SessionFailure{
					Kind:    store.FailureCanceled,
					Summary: "operator interrupted the turn",
				},
			},
			want: "*[interrupted]* operator interrupted the turn",
		},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			events := []store.SessionEvent{
				mustUIAgentSessionEvent(
					t,
					"ev-marker-"+test.event.TurnID,
					int64(index+1),
					test.event.Timestamp,
					test.event,
				),
			}
			transcriptMessages, err := Assemble(events)
			if err != nil {
				t.Fatalf("Assemble() error = %v", err)
			}
			if got, want := len(transcriptMessages), 1; got != want {
				t.Fatalf("len(transcriptMessages) = %d, want %d; messages=%#v", got, want, transcriptMessages)
			}
			if got, want := transcriptMessages[0].Role, RoleSystem; got != want {
				t.Fatalf("transcript role = %q, want %q", got, want)
			}
			if got := transcriptMessages[0].Content; got != test.want {
				t.Fatalf("transcript content = %q, want %q", got, test.want)
			}

			uiMessages, err := ToUIMessages(events)
			if err != nil {
				t.Fatalf("ToUIMessages() error = %v", err)
			}
			if got, want := len(uiMessages), 1; got != want {
				t.Fatalf("len(uiMessages) = %d, want %d; messages=%#v", got, want, uiMessages)
			}
			if got, want := uiMessages[0].Role, UIRoleSystem; got != want {
				t.Fatalf("UI role = %q, want %q", got, want)
			}
			if got := UIMessageText(uiMessages[0]); got != test.want {
				t.Fatalf("UI text = %q, want %q", got, test.want)
			}
		})
	}
}

func TestToUIMessagesPermissionDataParts(t *testing.T) {
	t.Run("ShouldReplacePendingPermissionWithFinalDecision", func(t *testing.T) {
		t.Parallel()

		timestamp := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
		events := []store.SessionEvent{
			mustPermissionSessionEvent(t, "ev-pending", 1, timestamp, ""),
			mustPermissionSessionEvent(t, "ev-final", 2, timestamp.Add(time.Second), "allow-once"),
		}

		messages, err := ToUIMessages(events)
		if err != nil {
			t.Fatalf("ToUIMessages() error = %v", err)
		}
		if got, want := len(messages), 1; got != want {
			t.Fatalf("len(messages) = %d, want %d", got, want)
		}
		if got, want := len(messages[0].Parts), 1; got != want {
			t.Fatalf("len(parts) = %d, want %d; parts=%#v", got, want, messages[0].Parts)
		}

		part := messages[0].Parts[0]
		if got, want := part.Type, uiPartDataPermission; got != want {
			t.Fatalf("part.Type = %q, want %q", got, want)
		}
		if got, want := part.ID, "req-permission"; got != want {
			t.Fatalf("part.ID = %q, want %q", got, want)
		}

		var payload UIAgentEventPayload
		if err := json.Unmarshal(part.Data, &payload); err != nil {
			t.Fatalf("json.Unmarshal(part.Data) error = %v", err)
		}
		if got, want := payload.Decision, "allow-once"; got != want {
			t.Fatalf("payload.Decision = %q, want %q", got, want)
		}
	})

	t.Run("ShouldPreservePendingPermissionWithoutDecision", func(t *testing.T) {
		t.Parallel()

		timestamp := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
		events := []store.SessionEvent{
			mustPermissionSessionEvent(t, "ev-pending", 1, timestamp, ""),
		}

		messages, err := ToUIMessages(events)
		if err != nil {
			t.Fatalf("ToUIMessages() error = %v", err)
		}
		if got, want := len(messages), 1; got != want {
			t.Fatalf("len(messages) = %d, want %d", got, want)
		}
		if got, want := len(messages[0].Parts), 1; got != want {
			t.Fatalf("len(parts) = %d, want %d; parts=%#v", got, want, messages[0].Parts)
		}

		part := messages[0].Parts[0]
		if got, want := part.ID, "req-permission"; got != want {
			t.Fatalf("part.ID = %q, want %q", got, want)
		}

		var payload UIAgentEventPayload
		if err := json.Unmarshal(part.Data, &payload); err != nil {
			t.Fatalf("json.Unmarshal(part.Data) error = %v", err)
		}
		if payload.Decision != "" {
			t.Fatalf("payload.Decision = %q, want empty", payload.Decision)
		}
	})

	t.Run("ShouldPreservePermissionOptionsForReplay", func(t *testing.T) {
		t.Parallel()

		timestamp := time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
		content, err := MarshalAgentEvent(acp.AgentEvent{
			Type:      acp.EventTypePermission,
			SessionID: "sess-permission",
			TurnID:    "turn-permission",
			RequestID: "req-permission-options",
			Timestamp: timestamp,
			Title:     "Bash",
			Action:    "session/request_permission",
			Resource:  "Bash",
			Raw: json.RawMessage(`{
				"request_id":"req-permission-options",
				"options":[
					{"decision":"allow-once","option_id":"allow-once","kind":"allow_once"},
					{"decision":"reject-once","option_id":"reject-once","kind":"reject_once"}
				],
				"tool_input":{"command":"touch blocked.txt"}
			}`),
		})
		if err != nil {
			t.Fatalf("MarshalAgentEvent() error = %v", err)
		}

		event := store.SessionEvent{
			ID:        "ev-permission-options",
			SessionID: "sess-permission",
			TurnID:    "turn-permission",
			Sequence:  1,
			Type:      acp.EventTypePermission,
			Content:   content,
			Timestamp: timestamp,
		}
		messages, err := ToUIMessages([]store.SessionEvent{event})
		if err != nil {
			t.Fatalf("ToUIMessages() error = %v", err)
		}
		if got, want := len(messages), 1; got != want {
			t.Fatalf("len(messages) = %d, want %d", got, want)
		}
		if got, want := len(messages[0].Parts), 1; got != want {
			t.Fatalf("len(parts) = %d, want %d; parts=%#v", got, want, messages[0].Parts)
		}

		var payload UIAgentEventPayload
		if err := json.Unmarshal(messages[0].Parts[0].Data, &payload); err != nil {
			t.Fatalf("json.Unmarshal(part.Data) error = %v", err)
		}
		var raw struct {
			Options []struct {
				Decision string `json:"decision"`
			} `json:"options"`
			ToolInput struct {
				Command string `json:"command"`
			} `json:"tool_input"`
		}
		if err := json.Unmarshal(payload.Raw, &raw); err != nil {
			t.Fatalf("json.Unmarshal(payload.Raw) error = %v", err)
		}
		if got, want := len(raw.Options), 2; got != want {
			t.Fatalf("len(raw.Options) = %d, want %d", got, want)
		}
		if got, want := raw.Options[0].Decision, "allow-once"; got != want {
			t.Fatalf("raw.Options[0].Decision = %q, want %q", got, want)
		}
		if got, want := raw.Options[1].Decision, "reject-once"; got != want {
			t.Fatalf("raw.Options[1].Decision = %q, want %q", got, want)
		}
		if got, want := raw.ToolInput.Command, "touch blocked.txt"; got != want {
			t.Fatalf("raw.ToolInput.Command = %q, want %q", got, want)
		}
	})
}

func TestToUIMessagesOrderedAssistantParts(t *testing.T) {
	t.Run("ShouldPreserveFatalPromptErrorAsDataPart", func(t *testing.T) {
		t.Parallel()

		timestamp := time.Date(2026, 5, 14, 15, 32, 0, 0, time.UTC)
		errorText := `{"code":-32603,"message":"Internal error","data":{"error":"peer disconnected before response"}}`
		events := []store.SessionEvent{
			mustUIAgentSessionEvent(t, "ev-text", 1, timestamp, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "sess-failed",
				TurnID:    "turn-failed",
				Timestamp: timestamp,
				Text:      "partial response",
			}),
			mustUIAgentSessionEvent(t, "ev-error", 2, timestamp.Add(time.Second), acp.AgentEvent{
				Type:      acp.EventTypeError,
				SessionID: "sess-failed",
				TurnID:    "turn-failed",
				Timestamp: timestamp.Add(time.Second),
				Error:     errorText,
				Failure: &store.SessionFailure{
					Kind:    store.FailureProcess,
					Summary: "peer disconnected before response",
				},
			}),
		}

		messages, err := ToUIMessages(events)
		if err != nil {
			t.Fatalf("ToUIMessages() error = %v", err)
		}
		if got, want := len(messages), 2; got != want {
			t.Fatalf("len(messages) = %d, want %d; messages=%#v", got, want, messages)
		}
		if got, want := len(messages[0].Parts), 1; got != want {
			t.Fatalf("len(messages[0].Parts) = %d, want %d; parts=%#v", got, want, messages[0].Parts)
		}
		if got, want := messages[0].Parts[0].Type, uiPartText; got != want {
			t.Fatalf("parts[0].Type = %q, want %q", got, want)
		}
		if got, want := messages[0].Parts[0].State, uiPartStateDone; got != want {
			t.Fatalf("parts[0].State = %q, want %q", got, want)
		}

		if got, want := messages[1].ID, "ev-error"; got != want {
			t.Fatalf("messages[1].ID = %q, want %q", got, want)
		}
		if got, want := len(messages[1].Parts), 1; got != want {
			t.Fatalf("len(messages[1].Parts) = %d, want %d; parts=%#v", got, want, messages[1].Parts)
		}

		errorPart := messages[1].Parts[0]
		if got, want := errorPart.Type, uiPartDataEvent; got != want {
			t.Fatalf("parts[1].Type = %q, want %q", got, want)
		}

		var payload UIAgentEventPayload
		if err := json.Unmarshal(errorPart.Data, &payload); err != nil {
			t.Fatalf("json.Unmarshal(errorPart.Data) error = %v", err)
		}
		if got, want := payload.Type, acp.EventTypeError; got != want {
			t.Fatalf("payload.Type = %q, want %q", got, want)
		}
		if got, want := payload.Error, errorText; got != want {
			t.Fatalf("payload.Error = %q, want %q", got, want)
		}
		if payload.Failure == nil {
			t.Fatal("payload.Failure = nil, want process failure")
		}
		if got, want := payload.Failure.Kind, store.FailureProcess; got != want {
			t.Fatalf("payload.Failure.Kind = %q, want %q", got, want)
		}
	})

	t.Run("ShouldPreserveTextToolTextOrderInsideOneAssistantMessage", func(t *testing.T) {
		t.Parallel()

		timestamp := time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)
		events := []store.SessionEvent{
			mustUIAgentSessionEvent(t, "ev-text-1", 1, timestamp, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "sess-mixed",
				TurnID:    "turn-mixed",
				Timestamp: timestamp,
				Text:      "text 1",
			}),
			mustUIAgentSessionEvent(t, "ev-tool-2", 2, timestamp.Add(time.Second), acp.AgentEvent{
				Type:       acp.EventTypeToolCall,
				SessionID:  "sess-mixed",
				TurnID:     "turn-mixed",
				Timestamp:  timestamp.Add(time.Second),
				Title:      "Bash",
				ToolCallID: "tool-2",
				Raw:        json.RawMessage("{\"rawInput\":{\"command\":\"pwd\"}}"),
			}),
			mustUIAgentSessionEvent(t, "ev-tool-3", 3, timestamp.Add(2*time.Second), acp.AgentEvent{
				Type:       acp.EventTypeToolCall,
				SessionID:  "sess-mixed",
				TurnID:     "turn-mixed",
				Timestamp:  timestamp.Add(2 * time.Second),
				Title:      "Read",
				ToolCallID: "tool-3",
				Raw:        json.RawMessage("{\"rawInput\":{\"file_path\":\"README.md\"}}"),
			}),
			mustUIAgentSessionEvent(t, "ev-text-4", 4, timestamp.Add(3*time.Second), acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "sess-mixed",
				TurnID:    "turn-mixed",
				Timestamp: timestamp.Add(3 * time.Second),
				Text:      "text 4",
			}),
			mustUIAgentSessionEvent(t, "ev-text-5", 5, timestamp.Add(4*time.Second), acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "sess-mixed",
				TurnID:    "turn-mixed",
				Timestamp: timestamp.Add(4 * time.Second),
				Text:      "text 5",
			}),
			mustUIAgentSessionEvent(t, "ev-done", 6, timestamp.Add(5*time.Second), acp.AgentEvent{
				Type:       acp.EventTypeDone,
				SessionID:  "sess-mixed",
				TurnID:     "turn-mixed",
				Timestamp:  timestamp.Add(5 * time.Second),
				StopReason: "end_turn",
			}),
		}

		messages, err := ToUIMessages(events)
		if err != nil {
			t.Fatalf("ToUIMessages() error = %v", err)
		}
		if got, want := len(messages), 1; got != want {
			t.Fatalf("len(messages) = %d, want %d; messages=%#v", got, want, messages)
		}

		got := uiVisiblePartSignatures(messages[0].Parts)
		want := []string{
			"text:turn-mixed-text-1:text 1:done",
			"tool-Bash:tool-2:input-available",
			"tool-Read:tool-3:input-available",
			"text:turn-mixed-text-2:text 4text 5:done",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("visible part signatures = %#v, want %#v; parts=%#v", got, want, messages[0].Parts)
		}
	})

	t.Run("ShouldPreserveReasoningAsASeparateOrderedPart", func(t *testing.T) {
		t.Parallel()

		timestamp := time.Date(2026, 5, 13, 11, 0, 0, 0, time.UTC)
		events := []store.SessionEvent{
			mustUIAgentSessionEvent(t, "ev-text-1", 1, timestamp, acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "sess-reasoning",
				TurnID:    "turn-reasoning",
				Timestamp: timestamp,
				Text:      "visible",
			}),
			mustUIAgentSessionEvent(t, "ev-thought-1", 2, timestamp.Add(time.Second), acp.AgentEvent{
				Type:      acp.EventTypeThought,
				SessionID: "sess-reasoning",
				TurnID:    "turn-reasoning",
				Timestamp: timestamp.Add(time.Second),
				Text:      "checking",
			}),
			mustUIAgentSessionEvent(t, "ev-text-2", 3, timestamp.Add(2*time.Second), acp.AgentEvent{
				Type:      acp.EventTypeAgentMessage,
				SessionID: "sess-reasoning",
				TurnID:    "turn-reasoning",
				Timestamp: timestamp.Add(2 * time.Second),
				Text:      "answer",
			}),
			mustUIAgentSessionEvent(t, "ev-done", 4, timestamp.Add(3*time.Second), acp.AgentEvent{
				Type:       acp.EventTypeDone,
				SessionID:  "sess-reasoning",
				TurnID:     "turn-reasoning",
				Timestamp:  timestamp.Add(3 * time.Second),
				StopReason: "end_turn",
			}),
		}

		messages, err := ToUIMessages(events)
		if err != nil {
			t.Fatalf("ToUIMessages() error = %v", err)
		}
		if got, want := len(messages), 1; got != want {
			t.Fatalf("len(messages) = %d, want %d; messages=%#v", got, want, messages)
		}

		got := uiVisiblePartSignatures(messages[0].Parts)
		want := []string{
			"text:turn-reasoning-text-1:visible:done",
			"reasoning:turn-reasoning-reasoning-1:checking:done",
			"text:turn-reasoning-text-2:answer:done",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("visible part signatures = %#v, want %#v; parts=%#v", got, want, messages[0].Parts)
		}
	})
}

func TestBuildToolResultDecodesRawJSONObjectPayload(t *testing.T) {
	t.Parallel()

	t.Run("ShouldDecodeRawJSONObjectPayload", func(t *testing.T) {
		result := buildToolResult("Read", false, "", json.RawMessage(`{
			"stdout":"workspace\n",
			"content":"workspace\n",
			"structuredPatch":{"ops":[{"op":"replace","path":"/tmp/demo.txt"}]}
		}`))
		if result == nil {
			t.Fatal("buildToolResult() = nil, want populated result")
			return
		}
		if got := result.Stdout; got != "workspace\n" {
			t.Fatalf("Stdout = %q, want %q", got, "workspace\n")
		}
		if got := result.Content; got != "workspace\n" {
			t.Fatalf("Content = %q, want %q", got, "workspace\n")
		}
		if len(result.StructuredPatch) == 0 {
			t.Fatal("StructuredPatch = empty, want preserved patch payload")
		}
	})
}

func mustUIAgentSessionEvent(
	t *testing.T,
	id string,
	sequence int64,
	timestamp time.Time,
	event acp.AgentEvent,
) store.SessionEvent {
	t.Helper()

	content, err := MarshalAgentEvent(event)
	if err != nil {
		t.Fatalf("MarshalAgentEvent(%s) error = %v", event.Type, err)
	}

	return store.SessionEvent{
		ID:        id,
		SessionID: event.SessionID,
		TurnID:    event.TurnID,
		Sequence:  sequence,
		Type:      event.Type,
		Content:   content,
		Timestamp: timestamp,
	}
}

func uiVisiblePartSignatures(parts []UIMessagePart) []string {
	signatures := make([]string, 0, len(parts))
	for _, part := range parts {
		switch {
		case part.Type == uiPartText:
			signatures = append(signatures, part.Type+":"+part.ID+":"+part.Text+":"+part.State)
		case part.Type == uiPartReasoning:
			signatures = append(signatures, part.Type+":"+part.ID+":"+part.Text+":"+part.State)
		case part.Type == uiPartDynamicTool || strings.HasPrefix(part.Type, "tool-"):
			signatures = append(signatures, part.Type+":"+part.ToolCallID+":"+part.State)
		}
	}
	return signatures
}

func mustPermissionSessionEvent(
	t *testing.T,
	id string,
	sequence int64,
	timestamp time.Time,
	decision string,
) store.SessionEvent {
	t.Helper()

	content, err := MarshalAgentEvent(acp.AgentEvent{
		Type:      acp.EventTypePermission,
		SessionID: "sess-permission",
		TurnID:    "turn-permission",
		RequestID: "req-permission",
		Timestamp: timestamp,
		Title:     "Bash",
		Action:    "session/request_permission",
		Resource:  "Bash",
		Decision:  decision,
		Raw:       json.RawMessage(`{"command":"pwd"}`),
	})
	if err != nil {
		t.Fatalf("MarshalAgentEvent() error = %v", err)
	}

	return store.SessionEvent{
		ID:        id,
		SessionID: "sess-permission",
		TurnID:    "turn-permission",
		Sequence:  sequence,
		Type:      acp.EventTypePermission,
		Content:   content,
		Timestamp: timestamp,
	}
}

func TestUnmarshalAgentEventRoundTripPreservesStructuredFieldsWithoutRaw(t *testing.T) {
	t.Run("Should round-trip structured fields without restoring canonical raw payloads", func(t *testing.T) {
		t.Parallel()

		payload, err := MarshalAgentEvent(acp.AgentEvent{
			Type:      acp.EventTypeAgentMessage,
			SessionID: "acp-1",
			TurnID:    "turn-1",
			RequestID: "req-1",
			Timestamp: time.Date(2026, 4, 11, 2, 0, 0, 0, time.UTC),
			Text:      "hello",
			Title:     "assistant",
			Error:     "",
			Raw:       json.RawMessage(`{"chunk":1}`),
		})
		if err != nil {
			t.Fatalf("MarshalAgentEvent() error = %v", err)
		}

		event, err := UnmarshalAgentEvent(payload)
		if err != nil {
			t.Fatalf("UnmarshalAgentEvent() error = %v", err)
		}
		if got, want := event.Type, acp.EventTypeAgentMessage; got != want {
			t.Fatalf("Type = %q, want %q", got, want)
		}
		if got, want := event.SessionID, "acp-1"; got != want {
			t.Fatalf("SessionID = %q, want %q", got, want)
		}
		if got, want := event.TurnID, "turn-1"; got != want {
			t.Fatalf("TurnID = %q, want %q", got, want)
		}
		if got, want := event.RequestID, "req-1"; got != want {
			t.Fatalf("RequestID = %q, want %q", got, want)
		}
		if got, want := event.Text, "hello"; got != want {
			t.Fatalf("Text = %q, want %q", got, want)
		}
		if len(event.Raw) != 0 {
			t.Fatalf("Raw = %s, want empty canonical raw payload", string(event.Raw))
		}
	})
}

func assertNoDisplayLeaks(t *testing.T, value any, leaks []string) {
	t.Helper()

	var data []byte
	switch typed := value.(type) {
	case string:
		data = []byte(typed)
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			t.Fatalf("json.Marshal(%T) error = %v", value, err)
		}
		data = encoded
	}
	for _, leak := range leaks {
		if strings.Contains(string(data), leak) {
			t.Fatalf("display payload leaked %q: %s", leak, data)
		}
	}
}

func mustMarshalCanonical(
	t *testing.T,
	eventType string,
	turnID string,
	timestamp time.Time,
	text string,
	toolName string,
	toolCallID string,
	toolInput json.RawMessage,
	toolResult *ToolResult,
	toolError bool,
) string {
	t.Helper()

	payload, err := canonicalPayload(
		eventType,
		turnID,
		timestamp,
		text,
		toolName,
		toolCallID,
		toolInput,
		toolResult,
		toolError,
	)
	if err != nil {
		t.Fatalf("canonicalPayload() error = %v", err)
	}
	return string(payload)
}
