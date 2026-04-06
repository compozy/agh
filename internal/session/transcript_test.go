package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
)

func TestManagerTranscriptAssemblesLegacyACPEvents(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		if err := h.manager.Stop(testContext(t), session.ID); err != nil {
			t.Logf("h.manager.Stop failed for session %s: %v", session.ID, err)
		}
	})

	recorder := session.recorderHandle()
	events := []store.SessionEvent{
		{
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeThought,
			AgentName: session.Info().AgentName,
			Content:   `{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":"Thinking "}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
		},
		{
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeThought,
			AgentName: session.Info().AgentName,
			Content:   `{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":"hard"}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 1, 0, time.UTC),
		},
		{
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeAgentMessage,
			AgentName: session.Info().AgentName,
			Content:   `{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"Let me read "}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 2, 0, time.UTC),
		},
		{
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeAgentMessage,
			AgentName: session.Info().AgentName,
			Content:   `{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"the file"}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 3, 0, time.UTC),
		},
		{
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeToolCall,
			AgentName: session.Info().AgentName,
			Content:   `{"_meta":{"claudeCode":{"toolName":"Read"}},"toolCallId":"call-1","sessionUpdate":"tool_call","rawInput":{},"status":"pending","title":"Read File","kind":"read","content":[]}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 4, 0, time.UTC),
		},
		{
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeToolCall,
			AgentName: session.Info().AgentName,
			Content:   `{"_meta":{"claudeCode":{"toolName":"Read"}},"toolCallId":"call-1","sessionUpdate":"tool_call_update","rawInput":{"file_path":"/tmp/demo.txt"},"status":"in_progress","title":"Read /tmp/demo.txt","kind":"read","content":[]}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 5, 0, time.UTC),
		},
		{
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeToolResult,
			AgentName: session.Info().AgentName,
			Content:   `{"_meta":{"claudeCode":{"toolName":"Read"}},"toolCallId":"call-1","sessionUpdate":"tool_call_update","status":"completed","rawOutput":"line1\nline2","content":[{"type":"content","content":{"type":"text","text":"line1\nline2"}}]}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 6, 0, time.UTC),
		},
		{
			TurnID:    "turn-legacy",
			Type:      acp.EventTypeAgentMessage,
			AgentName: session.Info().AgentName,
			Content:   `{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"Done."}}`,
			Timestamp: time.Date(2026, 4, 3, 12, 0, 7, 0, time.UTC),
		},
	}

	for _, event := range events {
		if err := recorder.Record(testContext(t), event); err != nil {
			t.Fatalf("Record(%s) error = %v", event.Type, err)
		}
	}

	messages, err := h.manager.Transcript(testContext(t), session.ID)
	if err != nil {
		t.Fatalf("Transcript() error = %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("Transcript() len = %d, want 4", len(messages))
	}

	if got := messages[0].Role; got != TranscriptRoleAssistant {
		t.Fatalf("messages[0].Role = %q, want %q", got, TranscriptRoleAssistant)
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
	if !messages[0].Timestamp.Equal(events[0].Timestamp) {
		t.Fatalf("messages[0].Timestamp = %s, want %s", messages[0].Timestamp, events[0].Timestamp)
	}

	if got := messages[1].Role; got != TranscriptRoleToolCall {
		t.Fatalf("messages[1].Role = %q, want %q", got, TranscriptRoleToolCall)
	}
	if got := messages[1].ToolName; got != "Read" {
		t.Fatalf("messages[1].ToolName = %q, want %q", got, "Read")
	}
	if got := string(messages[1].ToolInput); got != `{"file_path":"/tmp/demo.txt"}` {
		t.Fatalf("messages[1].ToolInput = %s", got)
	}

	if got := messages[2].Role; got != TranscriptRoleToolResult {
		t.Fatalf("messages[2].Role = %q, want %q", got, TranscriptRoleToolResult)
	}
	if messages[2].ToolResult == nil || messages[2].ToolResult.Content != "line1\nline2" {
		t.Fatalf("messages[2].ToolResult = %#v, want content", messages[2].ToolResult)
	}
	if messages[2].ToolError {
		t.Fatal("messages[2].ToolError = true, want false")
	}

	if got := messages[3].Role; got != TranscriptRoleAssistant {
		t.Fatalf("messages[3].Role = %q, want %q", got, TranscriptRoleAssistant)
	}
	if got := messages[3].Content; got != "Done." {
		t.Fatalf("messages[3].Content = %q, want %q", got, "Done.")
	}
	if !messages[3].Timestamp.Equal(events[7].Timestamp) {
		t.Fatalf("messages[3].Timestamp = %s, want %s", messages[3].Timestamp, events[7].Timestamp)
	}
}

func TestManagerTranscriptReadsCanonicalEnvelope(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		if err := h.manager.Stop(testContext(t), session.ID); err != nil {
			t.Logf("h.manager.Stop failed for session %s: %v", session.ID, err)
		}
	})

	events := []acp.AgentEvent{
		{
			Type:      acp.EventTypeUserMessage,
			TurnID:    "turn-canonical",
			Timestamp: time.Date(2026, 4, 3, 13, 0, 0, 0, time.UTC),
			Text:      "list files",
		},
		{
			Type:      acp.EventTypeAgentMessage,
			TurnID:    "turn-canonical",
			Timestamp: time.Date(2026, 4, 3, 13, 0, 1, 0, time.UTC),
			Text:      "Listing files",
		},
		{
			Type:       acp.EventTypeToolCall,
			TurnID:     "turn-canonical",
			Timestamp:  time.Date(2026, 4, 3, 13, 0, 2, 0, time.UTC),
			ToolCallID: "call-2",
			Title:      "Bash",
			Raw: json.RawMessage(
				`{"_meta":{"claudeCode":{"toolName":"Bash"}},"toolCallId":"call-2","sessionUpdate":"tool_call_update","rawInput":{"command":"ls -la"}}`,
			),
		},
		{
			Type:       acp.EventTypeToolResult,
			TurnID:     "turn-canonical",
			Timestamp:  time.Date(2026, 4, 3, 13, 0, 3, 0, time.UTC),
			ToolCallID: "call-2",
			Raw: json.RawMessage(
				`{"_meta":{"claudeCode":{"toolName":"Bash"}},"toolCallId":"call-2","sessionUpdate":"tool_call_update","status":"completed","rawOutput":"ok"}`,
			),
		},
	}

	for _, event := range events {
		normalized := h.manager.normalizeEvent(session, event.TurnID, event)
		if err := h.manager.recordEvent(testContext(t), session, normalized); err != nil {
			t.Fatalf("recordEvent(%s) error = %v", event.Type, err)
		}
	}

	messages, err := h.manager.Transcript(testContext(t), session.ID)
	if err != nil {
		t.Fatalf("Transcript() error = %v", err)
	}
	if len(messages) != 4 {
		t.Fatalf("Transcript() len = %d, want 4", len(messages))
	}

	if got := messages[0].Role; got != TranscriptRoleUser {
		t.Fatalf("messages[0].Role = %q, want %q", got, TranscriptRoleUser)
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

func TestParseLooseTranscriptEventBuildsToolResultFromLoosePayload(t *testing.T) {
	t.Parallel()

	event := parseLooseTranscriptEvent(transcriptEvent{Type: acp.EventTypeToolResult}, map[string]any{
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

	if got := event.ToolCallID; got != "call-loose" {
		t.Fatalf("ToolCallID = %q, want %q", got, "call-loose")
	}
	if got := event.ToolName; got != "Bash" {
		t.Fatalf("ToolName = %q, want %q", got, "Bash")
	}
	if got := string(event.ToolInput); got != `{"command":"pwd"}` {
		t.Fatalf("ToolInput = %s, want JSON command payload", got)
	}
	if event.ToolResult == nil {
		t.Fatal("ToolResult = nil, want populated result")
	}
	if got := event.ToolResult.Stdout; got != "workspace\n" {
		t.Fatalf("ToolResult.Stdout = %q, want %q", got, "workspace\n")
	}
	if event.ToolError {
		t.Fatal("ToolError = true, want false")
	}

	if got := string(firstNonEmptyRaw(nil, json.RawMessage(`{"ok":true}`))); got != `{"ok":true}` {
		t.Fatalf("firstNonEmptyRaw() = %s, want non-empty raw payload", got)
	}
	if got := firstNonNil(nil, "", "value"); got != "" {
		t.Fatalf("firstNonNil(nil, \"\", \"value\") = %#v, want empty string first", got)
	}
}
