package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
)

const eventEnvelopeSchema = "agh.session.event.v1"

// TranscriptRole is the renderable chat role emitted by the canonical transcript API.
type TranscriptRole string

const (
	TranscriptRoleUser       TranscriptRole = "user"
	TranscriptRoleAssistant  TranscriptRole = "assistant"
	TranscriptRoleToolCall   TranscriptRole = "tool_call"
	TranscriptRoleToolResult TranscriptRole = "tool_result"
)

// TranscriptToolResult is the canonical renderable tool output shape for replay.
type TranscriptToolResult struct {
	Stdout          string          `json:"stdout,omitempty"`
	Stderr          string          `json:"stderr,omitempty"`
	FilePath        string          `json:"file_path,omitempty"`
	Content         string          `json:"content,omitempty"`
	StructuredPatch json.RawMessage `json:"structured_patch,omitempty"`
	Error           string          `json:"error,omitempty"`
	RawOutput       json.RawMessage `json:"raw_output,omitempty"`
}

// TranscriptMessage is the canonical replay message returned to the frontend.
type TranscriptMessage struct {
	ID               string                `json:"id"`
	Role             TranscriptRole        `json:"role"`
	Content          string                `json:"content"`
	Thinking         string                `json:"thinking,omitempty"`
	ThinkingComplete bool                  `json:"thinking_complete"`
	ToolName         string                `json:"tool_name,omitempty"`
	ToolInput        json.RawMessage       `json:"tool_input,omitempty"`
	ToolResult       *TranscriptToolResult `json:"tool_result,omitempty"`
	ToolError        bool                  `json:"tool_error"`
	Timestamp        time.Time             `json:"timestamp"`
}

type transcriptEvent struct {
	ID         string
	TurnID     string
	Type       string
	Text       string
	ToolCallID string
	ToolName   string
	ToolInput  json.RawMessage
	ToolResult *TranscriptToolResult
	ToolError  bool
	Timestamp  time.Time
}

type assistantBuffer struct {
	id        string
	turnID    string
	timestamp time.Time
	content   strings.Builder
	thinking  strings.Builder
}

type toolLifecycle struct {
	callIndex   int
	resultIndex int
}

type canonicalEventPayload struct {
	Schema     string                `json:"schema,omitempty"`
	Type       string                `json:"type,omitempty"`
	SessionID  string                `json:"session_id,omitempty"`
	TurnID     string                `json:"turn_id,omitempty"`
	RequestID  string                `json:"request_id,omitempty"`
	Timestamp  time.Time             `json:"timestamp,omitempty"`
	Text       string                `json:"text,omitempty"`
	Title      string                `json:"title,omitempty"`
	ToolName   string                `json:"tool_name,omitempty"`
	ToolCallID string                `json:"tool_call_id,omitempty"`
	ToolInput  json.RawMessage       `json:"tool_input,omitempty"`
	ToolResult *TranscriptToolResult `json:"tool_result,omitempty"`
	ToolError  bool                  `json:"tool_error,omitempty"`
	StopReason string                `json:"stop_reason,omitempty"`
	Action     string                `json:"action,omitempty"`
	Resource   string                `json:"resource,omitempty"`
	Decision   string                `json:"decision,omitempty"`
	Error      string                `json:"error,omitempty"`
	Usage      *acp.TokenUsage       `json:"usage,omitempty"`
	Raw        json.RawMessage       `json:"raw,omitempty"`
}

// Transcript returns a canonical replay transcript for the requested session.
func (m *Manager) Transcript(ctx context.Context, id string) ([]TranscriptMessage, error) {
	recorder, cleanup, err := m.openQueryRecorder(ctx, id)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			logger := m.logger
			if logger == nil {
				logger = slog.Default()
			}
			logger.Warn("session: transcript cleanup failed", "session_id", strings.TrimSpace(id), "error", cleanupErr)
		}
	}()

	events, err := recorder.Query(ctx, store.EventQuery{})
	if err != nil {
		return nil, fmt.Errorf("session: query transcript events for %q: %w", strings.TrimSpace(id), err)
	}

	return assembleTranscript(events)
}

func assembleTranscript(events []store.SessionEvent) ([]TranscriptMessage, error) {
	if len(events) == 0 {
		return []TranscriptMessage{}, nil
	}

	sorted := append([]store.SessionEvent(nil), events...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Sequence == sorted[j].Sequence {
			if sorted[i].Timestamp.Equal(sorted[j].Timestamp) {
				return sorted[i].ID < sorted[j].ID
			}
			return sorted[i].Timestamp.Before(sorted[j].Timestamp)
		}
		return sorted[i].Sequence < sorted[j].Sequence
	})

	messages := make([]TranscriptMessage, 0, len(sorted))
	var assistant assistantBuffer
	toolStates := make(map[string]*toolLifecycle)

	flushAssistant := func() {
		if assistant.id == "" {
			return
		}
		content := assistant.content.String()
		thinking := assistant.thinking.String()
		if strings.TrimSpace(content) == "" && strings.TrimSpace(thinking) == "" {
			assistant = assistantBuffer{}
			return
		}

		messages = append(messages, TranscriptMessage{
			ID:               assistant.id,
			Role:             TranscriptRoleAssistant,
			Content:          content,
			Thinking:         thinking,
			ThinkingComplete: strings.TrimSpace(thinking) != "",
			Timestamp:        assistant.timestamp,
		})
		assistant = assistantBuffer{}
	}

	for _, event := range sorted {
		parsed, err := parseTranscriptEvent(event)
		if err != nil {
			return nil, err
		}

		if assistant.id != "" && assistant.turnID != "" && parsed.TurnID != "" && assistant.turnID != parsed.TurnID {
			flushAssistant()
		}

		switch parsed.Type {
		case acp.EventTypeUserMessage:
			flushAssistant()
			if strings.TrimSpace(parsed.Text) == "" {
				continue
			}
			messages = append(messages, TranscriptMessage{
				ID:        parsed.ID,
				Role:      TranscriptRoleUser,
				Content:   parsed.Text,
				Timestamp: parsed.Timestamp,
			})
		case acp.EventTypeAgentMessage:
			if strings.TrimSpace(parsed.Text) == "" && assistant.id == "" {
				continue
			}
			if assistant.id == "" {
				assistant.id = parsed.ID
				assistant.turnID = parsed.TurnID
				assistant.timestamp = parsed.Timestamp
			}
			assistant.content.WriteString(parsed.Text)
		case acp.EventTypeThought:
			if strings.TrimSpace(parsed.Text) == "" && assistant.id == "" {
				continue
			}
			if assistant.id == "" {
				assistant.id = parsed.ID
				assistant.turnID = parsed.TurnID
				assistant.timestamp = parsed.Timestamp
			}
			assistant.thinking.WriteString(parsed.Text)
		case acp.EventTypeToolCall:
			flushAssistant()
			applyToolCall(&messages, toolStates, parsed)
		case acp.EventTypeToolResult:
			flushAssistant()
			applyToolResult(&messages, toolStates, parsed)
		default:
			flushAssistant()
		}
	}

	flushAssistant()
	return messages, nil
}

func applyToolCall(messages *[]TranscriptMessage, toolStates map[string]*toolLifecycle, event transcriptEvent) {
	toolID := strings.TrimSpace(event.ToolCallID)
	if toolID == "" {
		toolID = event.ID
	}
	if toolID == "" {
		return
	}

	lifecycle, ok := toolStates[toolID]
	if !ok {
		lifecycle = &toolLifecycle{callIndex: -1, resultIndex: -1}
		toolStates[toolID] = lifecycle
	}

	if lifecycle.callIndex >= 0 {
		msg := &(*messages)[lifecycle.callIndex]
		mergeToolCallMessage(msg, event)
		return
	}

	*messages = append(*messages, TranscriptMessage{
		ID:        toolID,
		Role:      TranscriptRoleToolCall,
		Content:   "",
		ToolName:  event.ToolName,
		ToolInput: acp.CloneRawMessage(event.ToolInput),
		Timestamp: event.Timestamp,
	})
	lifecycle.callIndex = len(*messages) - 1
}

func applyToolResult(messages *[]TranscriptMessage, toolStates map[string]*toolLifecycle, event transcriptEvent) {
	toolID := strings.TrimSpace(event.ToolCallID)
	if toolID == "" {
		toolID = event.ID
	}
	if toolID == "" {
		return
	}

	lifecycle, ok := toolStates[toolID]
	if !ok {
		lifecycle = &toolLifecycle{callIndex: -1, resultIndex: -1}
		toolStates[toolID] = lifecycle
	}

	if lifecycle.callIndex < 0 {
		*messages = append(*messages, TranscriptMessage{
			ID:        toolID,
			Role:      TranscriptRoleToolCall,
			Content:   "",
			ToolName:  event.ToolName,
			ToolInput: acp.CloneRawMessage(event.ToolInput),
			Timestamp: event.Timestamp,
		})
		lifecycle.callIndex = len(*messages) - 1
	} else {
		mergeToolCallMessage(&(*messages)[lifecycle.callIndex], event)
	}

	result := cloneTranscriptToolResult(event.ToolResult)
	if result == nil {
		result = &TranscriptToolResult{}
	}
	if lifecycle.resultIndex >= 0 {
		msg := &(*messages)[lifecycle.resultIndex]
		msg.ToolName = firstNonEmpty(msg.ToolName, event.ToolName)
		msg.ToolResult = result
		msg.ToolError = msg.ToolError || event.ToolError
		return
	}

	*messages = append(*messages, TranscriptMessage{
		ID:         toolID,
		Role:       TranscriptRoleToolResult,
		Content:    "",
		ToolName:   event.ToolName,
		ToolResult: result,
		ToolError:  event.ToolError,
		Timestamp:  event.Timestamp,
	})
	lifecycle.resultIndex = len(*messages) - 1
}

func mergeToolCallMessage(msg *TranscriptMessage, event transcriptEvent) {
	if msg == nil {
		return
	}
	msg.ToolName = firstNonEmpty(msg.ToolName, event.ToolName)
	if (len(msg.ToolInput) == 0 || rawMessageIsEmptyObject(msg.ToolInput)) && len(event.ToolInput) > 0 && !rawMessageIsEmptyObject(event.ToolInput) {
		msg.ToolInput = acp.CloneRawMessage(event.ToolInput)
	}
}

func parseTranscriptEvent(event store.SessionEvent) (transcriptEvent, error) {
	parsed := transcriptEvent{
		ID:        strings.TrimSpace(event.ID),
		TurnID:    strings.TrimSpace(event.TurnID),
		Type:      strings.TrimSpace(event.Type),
		Timestamp: event.Timestamp.UTC(),
	}

	content := strings.TrimSpace(event.Content)
	if content == "" {
		return parsed, nil
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		if parsed.Type == acp.EventTypeUserMessage || parsed.Type == acp.EventTypeAgentMessage || parsed.Type == acp.EventTypeThought {
			parsed.Text = content
			return parsed, nil
		}
		return parsed, nil
	}

	if schema := nestedString(payload, "schema"); schema == eventEnvelopeSchema {
		return parseCanonicalTranscriptEvent(parsed, payload), nil
	}
	if _, ok := payload["sessionUpdate"]; ok {
		return parseLegacyTranscriptEvent(parsed, payload), nil
	}
	return parseLooseTranscriptEvent(parsed, payload), nil
}

func parseCanonicalTranscriptEvent(event transcriptEvent, payload map[string]any) transcriptEvent {
	event.Type = firstNonEmpty(nestedString(payload, "type"), event.Type)
	event.Text = nestedString(payload, "text")
	event.ToolCallID = firstNonEmpty(nestedString(payload, "tool_call_id"), nestedString(payload, "toolCallId"))
	event.ToolName = firstNonEmpty(nestedString(payload, "tool_name"), nestedString(payload, "title"))
	event.ToolInput = acp.CloneRawMessage(rawMessageFromValue(payload["tool_input"]))
	if toolResult := decodeTranscriptToolResult(rawMessageFromValue(payload["tool_result"])); toolResult != nil {
		event.ToolResult = toolResult
	}
	event.ToolError = nestedBool(payload, "tool_error") || strings.TrimSpace(nestedString(payload, "error")) != ""
	if event.ToolResult != nil && strings.TrimSpace(event.ToolResult.Error) != "" {
		event.ToolError = true
	}
	return event
}

func parseLegacyTranscriptEvent(event transcriptEvent, payload map[string]any) transcriptEvent {
	updateType := nestedString(payload, "sessionUpdate")
	status := strings.ToLower(strings.TrimSpace(nestedString(payload, "status")))
	event.Text = extractLegacyContentText(payload["content"])
	event.ToolCallID = firstNonEmpty(nestedString(payload, "toolCallId"), nestedString(payload, "tool_call_id"))
	event.ToolName = legacyToolName(payload)
	event.ToolInput = acp.CloneRawMessage(rawMessageFromValue(payload["rawInput"]))

	switch updateType {
	case "user_message_chunk":
		event.Type = acp.EventTypeUserMessage
	case "agent_message_chunk":
		event.Type = acp.EventTypeAgentMessage
	case "agent_thought_chunk":
		event.Type = acp.EventTypeThought
	case "tool_call":
		event.Type = acp.EventTypeToolCall
	case "tool_call_update":
		if event.Type != acp.EventTypeToolResult {
			if status == "completed" || status == "failed" {
				event.Type = acp.EventTypeToolResult
			} else {
				event.Type = acp.EventTypeToolCall
			}
		}
	}

	if event.Type == acp.EventTypeToolResult {
		event.ToolResult = buildToolResult(
			event.ToolName,
			strings.EqualFold(status, "failed"),
			extractLegacyContentText(payload["content"]),
			payload["rawOutput"],
		)
		event.ToolError = strings.EqualFold(status, "failed")
	}

	return event
}

func parseLooseTranscriptEvent(event transcriptEvent, payload map[string]any) transcriptEvent {
	event.Type = firstNonEmpty(nestedString(payload, "type"), event.Type)
	event.Text = nestedString(payload, "text")
	event.ToolCallID = firstNonEmpty(nestedString(payload, "tool_call_id"), nestedString(payload, "toolCallId"))
	event.ToolName = firstNonEmpty(nestedString(payload, "tool_name"), nestedString(payload, "title"), legacyToolName(payload))
	event.ToolInput = acp.CloneRawMessage(firstNonEmptyRaw(
		rawMessageFromValue(payload["tool_input"]),
		rawMessageFromValue(payload["rawInput"]),
		rawMessageFromValue(payload["raw"]),
	))

	if toolResult := decodeTranscriptToolResult(firstNonEmptyRaw(
		rawMessageFromValue(payload["tool_result"]),
		rawMessageFromValue(payload["toolResult"]),
	)); toolResult != nil {
		event.ToolResult = toolResult
	} else if event.Type == acp.EventTypeToolResult {
		event.ToolResult = buildToolResult(
			event.ToolName,
			strings.TrimSpace(nestedString(payload, "error")) != "",
			extractLegacyContentText(payload["content"]),
			firstNonNil(payload["raw_output"], payload["rawOutput"], payload["raw"]),
		)
	}

	event.ToolError = nestedBool(payload, "tool_error") || strings.TrimSpace(nestedString(payload, "error")) != ""
	if event.ToolResult != nil && strings.TrimSpace(event.ToolResult.Error) != "" {
		event.ToolError = true
	}
	return event
}

func buildToolResult(toolName string, failed bool, contentText string, rawOutput any) *TranscriptToolResult {
	result := &TranscriptToolResult{}

	displayText := strings.TrimSpace(firstNonEmpty(contentText, stringifyValue(rawOutput)))
	raw := rawMessageFromValue(rawOutput)
	if len(raw) > 0 {
		result.RawOutput = acp.CloneRawMessage(raw)
		if mapped := map[string]any(nil); json.Unmarshal(raw, &mapped) == nil {
			result.Stdout = firstNonEmpty(result.Stdout, nestedString(mapped, "stdout"))
			result.Stderr = firstNonEmpty(result.Stderr, nestedString(mapped, "stderr"))
			result.FilePath = firstNonEmpty(result.FilePath, nestedString(mapped, "file_path"), nestedString(mapped, "filePath"))
			result.Content = firstNonEmpty(result.Content, nestedString(mapped, "content"))
			result.Error = firstNonEmpty(result.Error, nestedString(mapped, "error"))
			if patch := rawMessageFromValue(mapped["structuredPatch"]); len(patch) > 0 {
				result.StructuredPatch = acp.CloneRawMessage(patch)
			}
		}
	}

	switch strings.ToLower(strings.TrimSpace(toolName)) {
	case "bash":
		if failed {
			result.Stderr = firstNonEmpty(result.Stderr, displayText)
		} else {
			result.Stdout = firstNonEmpty(result.Stdout, displayText)
		}
	case "glob", "grep", "search":
		result.Stdout = firstNonEmpty(result.Stdout, displayText)
	case "read":
		result.Content = firstNonEmpty(result.Content, displayText)
	default:
		result.Content = firstNonEmpty(result.Content, displayText)
	}

	if failed {
		result.Error = firstNonEmpty(result.Error, displayText)
	}

	if result.Stdout == "" &&
		result.Stderr == "" &&
		result.FilePath == "" &&
		result.Content == "" &&
		len(result.StructuredPatch) == 0 &&
		result.Error == "" &&
		len(result.RawOutput) == 0 {
		return &TranscriptToolResult{}
	}

	return result
}

func decodeTranscriptToolResult(raw json.RawMessage) *TranscriptToolResult {
	if len(raw) == 0 {
		return nil
	}
	var result TranscriptToolResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return &result
}

func extractLegacyContentText(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case map[string]any:
		if text := nestedString(typed, "text"); strings.TrimSpace(text) != "" {
			return text
		}
		if inner, ok := typed["content"].(map[string]any); ok {
			return extractLegacyContentText(inner)
		}
		return ""
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(extractLegacyContentText(item))
			if text == "" {
				continue
			}
			parts = append(parts, text)
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

func legacyToolName(payload map[string]any) string {
	if meta, ok := payload["_meta"].(map[string]any); ok {
		for _, value := range meta {
			nested, ok := value.(map[string]any)
			if !ok {
				continue
			}
			if toolName := strings.TrimSpace(nestedString(nested, "toolName")); toolName != "" {
				return toolName
			}
		}
	}
	return firstNonEmpty(nestedString(payload, "title"), nestedString(payload, "kind"))
}

func nestedString(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	value, ok := payload[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func nestedBool(payload map[string]any, key string) bool {
	if payload == nil {
		return false
	}
	value, ok := payload[key]
	if !ok {
		return false
	}
	typed, ok := value.(bool)
	return ok && typed
}

func stringifyValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return extractLegacyContentText(value)
	}
}

func rawMessageFromValue(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return json.RawMessage(encoded)
}

func rawMessageIsEmptyObject(value json.RawMessage) bool {
	return strings.TrimSpace(string(value)) == "{}"
}

func cloneTranscriptToolResult(value *TranscriptToolResult) *TranscriptToolResult {
	if value == nil {
		return nil
	}
	cloned := *value
	cloned.StructuredPatch = acp.CloneRawMessage(value.StructuredPatch)
	cloned.RawOutput = acp.CloneRawMessage(value.RawOutput)
	return &cloned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstNonEmptyRaw(values ...json.RawMessage) json.RawMessage {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
