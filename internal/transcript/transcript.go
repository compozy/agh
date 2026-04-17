// Package transcript assembles canonical replay messages from persisted session events.
package transcript

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
)

// CanonicalSchema is the stored envelope schema for transcript-aware session events.
const CanonicalSchema = "agh.session.event.v1"

// Assembler assembles persisted session events into the canonical transcript shape.
type Assembler interface {
	Assemble(events []store.SessionEvent) ([]Message, error)
}

// Role is the renderable chat role emitted by the canonical transcript API.
type Role string

const (
	RoleUser       Role = "user"
	RoleAssistant  Role = "assistant"
	RoleToolCall   Role = "tool_call"
	RoleToolResult Role = "tool_result"
)

// ToolResult is the canonical renderable tool output shape for replay.
type ToolResult struct {
	Stdout          string          `json:"stdout,omitempty"`
	Stderr          string          `json:"stderr,omitempty"`
	FilePath        string          `json:"file_path,omitempty"`
	Content         string          `json:"content,omitempty"`
	StructuredPatch json.RawMessage `json:"structured_patch,omitempty"`
	Error           string          `json:"error,omitempty"`
	RawOutput       json.RawMessage `json:"raw_output,omitempty"`
}

// Message is the canonical replay message returned to transport callers.
type Message struct {
	ID               string          `json:"id"`
	Role             Role            `json:"role"`
	Content          string          `json:"content"`
	Thinking         string          `json:"thinking,omitempty"`
	ThinkingComplete bool            `json:"thinking_complete"`
	ToolName         string          `json:"tool_name,omitempty"`
	ToolInput        json.RawMessage `json:"tool_input,omitempty"`
	ToolResult       *ToolResult     `json:"tool_result,omitempty"`
	ToolError        bool            `json:"tool_error"`
	Timestamp        time.Time       `json:"timestamp"`
}

type event struct {
	ID         string
	TurnID     string
	Type       string
	Text       string
	ToolCallID string
	ToolName   string
	ToolInput  json.RawMessage
	ToolResult *ToolResult
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
	Schema     string          `json:"schema,omitempty"`
	Type       string          `json:"type,omitempty"`
	SessionID  string          `json:"session_id,omitempty"`
	TurnID     string          `json:"turn_id,omitempty"`
	RequestID  string          `json:"request_id,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
	Text       string          `json:"text,omitempty"`
	Title      string          `json:"title,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolInput  json.RawMessage `json:"tool_input,omitempty"`
	ToolResult *ToolResult     `json:"tool_result,omitempty"`
	ToolError  bool            `json:"tool_error,omitempty"`
	StopReason string          `json:"stop_reason,omitempty"`
	Action     string          `json:"action,omitempty"`
	Resource   string          `json:"resource,omitempty"`
	Decision   string          `json:"decision,omitempty"`
	Error      string          `json:"error,omitempty"`
	Usage      *acp.TokenUsage `json:"usage,omitempty"`
	Raw        json.RawMessage `json:"raw,omitempty"`
}

// Assemble returns the canonical replay transcript for the provided persisted events.
func Assemble(events []store.SessionEvent) ([]Message, error) {
	if len(events) == 0 {
		return []Message{}, nil
	}

	sorted := sortedTranscriptEvents(events)

	messages := make([]Message, 0, len(sorted))
	var assistant assistantBuffer
	toolStates := make(map[string]*toolLifecycle)

	for _, sessionEvent := range sorted {
		processTranscriptEvent(&messages, &assistant, toolStates, parseEvent(sessionEvent))
	}

	flushAssistantBuffer(&messages, &assistant)
	return messages, nil
}

func sortedTranscriptEvents(events []store.SessionEvent) []store.SessionEvent {
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
	return sorted
}

func processTranscriptEvent(
	messages *[]Message,
	assistant *assistantBuffer,
	toolStates map[string]*toolLifecycle,
	parsed event,
) {
	flushAssistantOnTurnChange(messages, assistant, parsed)

	switch parsed.Type {
	case acp.EventTypeUserMessage:
		appendUserTranscriptMessage(messages, assistant, parsed)
	case acp.EventTypeAgentMessage:
		appendAssistantTranscriptContent(assistant, parsed, false)
	case acp.EventTypeThought:
		appendAssistantTranscriptContent(assistant, parsed, true)
	case acp.EventTypeToolCall:
		flushAssistantBuffer(messages, assistant)
		applyToolCall(messages, toolStates, parsed)
	case acp.EventTypeToolResult:
		flushAssistantBuffer(messages, assistant)
		applyToolResult(messages, toolStates, parsed)
	default:
		flushAssistantBuffer(messages, assistant)
	}
}

func flushAssistantOnTurnChange(messages *[]Message, assistant *assistantBuffer, parsed event) {
	if assistant.id == "" || assistant.turnID == "" || parsed.TurnID == "" {
		return
	}
	if assistant.turnID != parsed.TurnID {
		flushAssistantBuffer(messages, assistant)
	}
}

func appendUserTranscriptMessage(messages *[]Message, assistant *assistantBuffer, parsed event) {
	flushAssistantBuffer(messages, assistant)
	if strings.TrimSpace(parsed.Text) == "" {
		return
	}
	*messages = append(*messages, Message{
		ID:        parsed.ID,
		Role:      RoleUser,
		Content:   parsed.Text,
		Timestamp: parsed.Timestamp,
	})
}

func appendAssistantTranscriptContent(assistant *assistantBuffer, parsed event, thinking bool) {
	if strings.TrimSpace(parsed.Text) == "" && assistant.id == "" {
		return
	}
	ensureAssistantBuffer(assistant, parsed)
	if thinking {
		assistant.thinking.WriteString(parsed.Text)
		return
	}
	assistant.content.WriteString(parsed.Text)
}

func ensureAssistantBuffer(assistant *assistantBuffer, parsed event) {
	if assistant.id != "" {
		return
	}
	assistant.id = parsed.ID
	assistant.turnID = parsed.TurnID
	assistant.timestamp = parsed.Timestamp
}

func flushAssistantBuffer(messages *[]Message, assistant *assistantBuffer) {
	if assistant.id == "" {
		return
	}
	content := assistant.content.String()
	thinking := assistant.thinking.String()
	if strings.TrimSpace(content) == "" && strings.TrimSpace(thinking) == "" {
		*assistant = assistantBuffer{}
		return
	}

	*messages = append(*messages, Message{
		ID:               assistant.id,
		Role:             RoleAssistant,
		Content:          content,
		Thinking:         thinking,
		ThinkingComplete: strings.TrimSpace(thinking) != "",
		Timestamp:        assistant.timestamp,
	})
	*assistant = assistantBuffer{}
}

func applyToolCall(messages *[]Message, toolStates map[string]*toolLifecycle, parsed event) {
	toolID := strings.TrimSpace(parsed.ToolCallID)
	if toolID == "" {
		toolID = parsed.ID
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
		mergeToolCallMessage(msg, parsed)
		return
	}

	*messages = append(*messages, Message{
		ID:        toolID,
		Role:      RoleToolCall,
		Content:   "",
		ToolName:  parsed.ToolName,
		ToolInput: acp.CloneRawMessage(parsed.ToolInput),
		Timestamp: parsed.Timestamp,
	})
	lifecycle.callIndex = len(*messages) - 1
}

func applyToolResult(messages *[]Message, toolStates map[string]*toolLifecycle, parsed event) {
	toolID := strings.TrimSpace(parsed.ToolCallID)
	if toolID == "" {
		toolID = parsed.ID
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
		*messages = append(*messages, Message{
			ID:        toolID,
			Role:      RoleToolCall,
			Content:   "",
			ToolName:  parsed.ToolName,
			ToolInput: acp.CloneRawMessage(parsed.ToolInput),
			Timestamp: parsed.Timestamp,
		})
		lifecycle.callIndex = len(*messages) - 1
	} else {
		mergeToolCallMessage(&(*messages)[lifecycle.callIndex], parsed)
	}

	result := cloneToolResult(parsed.ToolResult)
	if result == nil {
		result = &ToolResult{}
	}
	if lifecycle.resultIndex >= 0 {
		msg := &(*messages)[lifecycle.resultIndex]
		msg.ToolName = firstNonEmpty(msg.ToolName, parsed.ToolName)
		msg.ToolResult = result
		msg.ToolError = msg.ToolError || parsed.ToolError
		return
	}

	*messages = append(*messages, Message{
		ID:         toolID,
		Role:       RoleToolResult,
		Content:    "",
		ToolName:   parsed.ToolName,
		ToolResult: result,
		ToolError:  parsed.ToolError,
		Timestamp:  parsed.Timestamp,
	})
	lifecycle.resultIndex = len(*messages) - 1
}

func mergeToolCallMessage(msg *Message, parsed event) {
	if msg == nil {
		return
	}
	msg.ToolName = firstNonEmpty(msg.ToolName, parsed.ToolName)
	if (len(msg.ToolInput) == 0 || rawMessageIsEmptyObject(msg.ToolInput)) && len(parsed.ToolInput) > 0 &&
		!rawMessageIsEmptyObject(parsed.ToolInput) {
		msg.ToolInput = acp.CloneRawMessage(parsed.ToolInput)
	}
}

func parseEvent(sessionEvent store.SessionEvent) event {
	parsed := event{
		ID:        strings.TrimSpace(sessionEvent.ID),
		TurnID:    strings.TrimSpace(sessionEvent.TurnID),
		Type:      strings.TrimSpace(sessionEvent.Type),
		Timestamp: sessionEvent.Timestamp.UTC(),
	}

	content := strings.TrimSpace(sessionEvent.Content)
	if content == "" {
		return parsed
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		if parsed.Type == acp.EventTypeUserMessage || parsed.Type == acp.EventTypeAgentMessage ||
			parsed.Type == acp.EventTypeThought {
			parsed.Text = content
			return parsed
		}
		return parsed
	}

	if schema := nestedString(payload, "schema"); schema == CanonicalSchema {
		return parseCanonicalEvent(parsed, payload)
	}
	if _, ok := payload["sessionUpdate"]; ok {
		return parseLegacyEvent(parsed, payload)
	}
	return parseLooseEvent(parsed, payload)
}

func parseCanonicalEvent(parsed event, payload map[string]any) event {
	parsed.Type = firstNonEmpty(nestedString(payload, "type"), parsed.Type)
	parsed.Text = nestedString(payload, "text")
	parsed.ToolCallID = firstNonEmpty(nestedString(payload, "tool_call_id"), nestedString(payload, "toolCallId"))
	parsed.ToolName = firstNonEmpty(nestedString(payload, "tool_name"), nestedString(payload, "title"))
	parsed.ToolInput = acp.CloneRawMessage(rawMessageFromValue(payload["tool_input"]))
	if toolResult := decodeToolResult(rawMessageFromValue(payload["tool_result"])); toolResult != nil {
		parsed.ToolResult = toolResult
	}
	parsed.ToolError = nestedBool(payload, "tool_error") || strings.TrimSpace(nestedString(payload, "error")) != ""
	if parsed.ToolResult != nil && strings.TrimSpace(parsed.ToolResult.Error) != "" {
		parsed.ToolError = true
	}
	return parsed
}

func parseLegacyEvent(parsed event, payload map[string]any) event {
	updateType := nestedString(payload, "sessionUpdate")
	status := strings.ToLower(strings.TrimSpace(nestedString(payload, "status")))
	parsed.Text = extractLegacyContentText(payload["content"])
	parsed.ToolCallID = firstNonEmpty(nestedString(payload, "toolCallId"), nestedString(payload, "tool_call_id"))
	parsed.ToolName = legacyToolName(payload)
	parsed.ToolInput = acp.CloneRawMessage(rawMessageFromValue(payload["rawInput"]))

	switch updateType {
	case "user_message_chunk":
		parsed.Type = acp.EventTypeUserMessage
	case "agent_message_chunk":
		parsed.Type = acp.EventTypeAgentMessage
	case "agent_thought_chunk":
		parsed.Type = acp.EventTypeThought
	case "tool_call":
		parsed.Type = acp.EventTypeToolCall
	case "tool_call_update":
		if parsed.Type != acp.EventTypeToolResult {
			if status == "completed" || status == "failed" {
				parsed.Type = acp.EventTypeToolResult
			} else {
				parsed.Type = acp.EventTypeToolCall
			}
		}
	}

	if parsed.Type == acp.EventTypeToolResult {
		parsed.ToolResult = buildToolResult(
			parsed.ToolName,
			strings.EqualFold(status, "failed"),
			extractLegacyContentText(payload["content"]),
			payload["rawOutput"],
		)
		parsed.ToolError = strings.EqualFold(status, "failed")
	}

	return parsed
}

func parseLooseEvent(parsed event, payload map[string]any) event {
	parsed.Type = firstNonEmpty(nestedString(payload, "type"), parsed.Type)
	parsed.Text = nestedString(payload, "text")
	parsed.ToolCallID = firstNonEmpty(nestedString(payload, "tool_call_id"), nestedString(payload, "toolCallId"))
	parsed.ToolName = firstNonEmpty(
		nestedString(payload, "tool_name"),
		nestedString(payload, "title"),
		legacyToolName(payload),
	)
	parsed.ToolInput = acp.CloneRawMessage(firstNonEmptyRaw(
		rawMessageFromValue(payload["tool_input"]),
		rawMessageFromValue(payload["rawInput"]),
		rawMessageFromValue(payload["raw"]),
	))

	if toolResult := decodeToolResult(firstNonEmptyRaw(
		rawMessageFromValue(payload["tool_result"]),
		rawMessageFromValue(payload["toolResult"]),
	)); toolResult != nil {
		parsed.ToolResult = toolResult
	} else if parsed.Type == acp.EventTypeToolResult {
		parsed.ToolResult = buildToolResult(
			parsed.ToolName,
			strings.TrimSpace(nestedString(payload, "error")) != "",
			extractLegacyContentText(payload["content"]),
			firstNonNil(payload["raw_output"], payload["rawOutput"], payload["raw"]),
		)
	}

	parsed.ToolError = nestedBool(payload, "tool_error") || strings.TrimSpace(nestedString(payload, "error")) != ""
	if parsed.ToolResult != nil && strings.TrimSpace(parsed.ToolResult.Error) != "" {
		parsed.ToolError = true
	}
	return parsed
}

func buildToolResult(toolName string, failed bool, contentText string, rawOutput any) *ToolResult {
	result := &ToolResult{}

	displayText := strings.TrimSpace(firstNonEmpty(contentText, stringifyValue(rawOutput)))
	raw, mapped := rawToolResultOutput(rawOutput)
	if len(raw) > 0 {
		result.RawOutput = acp.CloneRawMessage(raw)
		if mapped == nil {
			mapped = rawToolResultObject(raw)
		}
		if mapped != nil {
			result.Stdout = firstNonEmpty(result.Stdout, nestedString(mapped, "stdout"))
			result.Stderr = firstNonEmpty(result.Stderr, nestedString(mapped, "stderr"))
			result.FilePath = firstNonEmpty(
				result.FilePath,
				nestedString(mapped, "file_path"),
				nestedString(mapped, "filePath"),
			)
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
		return &ToolResult{}
	}

	return result
}

func decodeToolResult(raw json.RawMessage) *ToolResult {
	if len(raw) == 0 {
		return nil
	}
	var result ToolResult
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
	typed, ok := value.(string)
	if !ok {
		return ""
	}
	return typed
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
	return bytes.Equal(bytes.TrimSpace(value), []byte("{}"))
}

func cloneToolResult(value *ToolResult) *ToolResult {
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

func rawToolResultOutput(value any) (json.RawMessage, map[string]any) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case json.RawMessage:
		return acp.CloneRawMessage(typed), nil
	case map[string]any:
		return rawMessageFromValue(typed), typed
	default:
		return rawMessageFromValue(value), nil
	}
}

func rawToolResultObject(raw json.RawMessage) map[string]any {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil
	}

	var mapped map[string]any
	if err := json.Unmarshal(trimmed, &mapped); err != nil {
		return nil
	}
	return mapped
}

func canonicalPayload(
	eventType string,
	turnID string,
	timestamp time.Time,
	text string,
	toolName string,
	toolCallID string,
	toolInput json.RawMessage,
	toolResult *ToolResult,
	toolError bool,
) ([]byte, error) {
	payload := canonicalEventPayload{
		Schema:     CanonicalSchema,
		Type:       strings.TrimSpace(eventType),
		TurnID:     strings.TrimSpace(turnID),
		Timestamp:  timestamp.UTC(),
		Text:       text,
		ToolName:   toolName,
		ToolCallID: strings.TrimSpace(toolCallID),
		ToolInput:  acp.CloneRawMessage(toolInput),
		ToolResult: cloneToolResult(toolResult),
		ToolError:  toolError,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("transcript: marshal canonical payload: %w", err)
	}
	return data, nil
}

// MarshalAgentEvent converts a runtime ACP event into the canonical stored payload.
func MarshalAgentEvent(event acp.AgentEvent) (string, error) {
	payload := canonicalEventPayload{
		Schema:     CanonicalSchema,
		Type:       event.Type,
		SessionID:  event.SessionID,
		TurnID:     event.TurnID,
		RequestID:  event.RequestID,
		Timestamp:  event.Timestamp,
		Text:       event.Text,
		Title:      event.Title,
		ToolCallID: event.ToolCallID,
		StopReason: event.StopReason,
		Action:     event.Action,
		Resource:   event.Resource,
		Decision:   event.Decision,
		Error:      event.Error,
		Usage:      event.Usage,
	}

	if len(event.Raw) > 0 {
		if json.Valid(event.Raw) {
			payload.Raw = acp.CloneRawMessage(event.Raw)
		} else {
			payload.Raw = rawMessageFromValue(string(event.Raw))
		}

		var rawPayload map[string]any
		if err := json.Unmarshal(event.Raw, &rawPayload); err == nil {
			payload.ToolName = legacyToolName(rawPayload)
			payload.ToolInput = acp.CloneRawMessage(rawMessageFromValue(rawPayload["rawInput"]))
			if event.Type == acp.EventTypeToolResult {
				toolResult := buildToolResult(
					payload.ToolName,
					strings.EqualFold(nestedString(rawPayload, "status"), "failed"),
					extractLegacyContentText(rawPayload["content"]),
					rawPayload["rawOutput"],
				)
				payload.ToolResult = toolResult
				payload.ToolError = strings.EqualFold(nestedString(rawPayload, "status"), "failed")
			}
		}
	}

	if payload.ToolName == "" {
		payload.ToolName = event.Title
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("transcript: marshal agent event: %w", err)
	}
	return string(data), nil
}

// UnmarshalAgentEvent converts a canonical stored payload back into an ACP event.
func UnmarshalAgentEvent(payload string) (acp.AgentEvent, error) {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return acp.AgentEvent{}, nil
	}

	var decoded canonicalEventPayload
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return acp.AgentEvent{}, fmt.Errorf("transcript: unmarshal agent event: %w", err)
	}

	event := acp.AgentEvent{
		Type:       strings.TrimSpace(decoded.Type),
		SessionID:  strings.TrimSpace(decoded.SessionID),
		TurnID:     strings.TrimSpace(decoded.TurnID),
		RequestID:  strings.TrimSpace(decoded.RequestID),
		Timestamp:  decoded.Timestamp,
		Text:       decoded.Text,
		Title:      firstNonEmpty(decoded.Title, decoded.ToolName),
		ToolCallID: strings.TrimSpace(decoded.ToolCallID),
		StopReason: strings.TrimSpace(decoded.StopReason),
		Action:     strings.TrimSpace(decoded.Action),
		Resource:   strings.TrimSpace(decoded.Resource),
		Decision:   strings.TrimSpace(decoded.Decision),
		Error:      strings.TrimSpace(decoded.Error),
		Usage:      decoded.Usage,
		Raw:        acp.CloneRawMessage(decoded.Raw),
	}
	return event, nil
}
