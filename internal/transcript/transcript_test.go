package transcript

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
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

func TestUnmarshalAgentEventRoundTripPreservesStructuredFieldsWithoutRaw(t *testing.T) {
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
