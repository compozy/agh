package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
)

const (
	UIRoleSystem    = "system"
	UIRoleUser      = "user"
	UIRoleAssistant = "assistant"

	uiPartText           = "text"
	uiPartReasoning      = "reasoning"
	uiPartDynamicTool    = "dynamic-tool"
	uiPartDataEvent      = "data-agh-event"
	uiPartDataPermission = "data-agh-permission"
	uiPartStateStreaming = "streaming"
	uiPartStateDone      = "done"
	uiToolStateStreaming = "input-streaming"
	uiToolStateAvailable = "input-available"
	uiToolStateOutput    = "output-available"
)

var emptyJSONObject = json.RawMessage(`{}`)

// UIMessage mirrors the AI SDK UIMessage wire shape used by the web client.
type UIMessage struct {
	ID       string          `json:"id"`
	Role     string          `json:"role"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Parts    []UIMessagePart `json:"parts"`
}

// UIMessagePart mirrors the AI SDK UIMessage part wire shape used by the web client.
type UIMessagePart struct {
	Type        string          `json:"type"`
	ID          string          `json:"id,omitempty"`
	Text        string          `json:"text,omitempty"`
	State       string          `json:"state,omitempty"`
	ToolName    string          `json:"toolName,omitempty"`
	ToolCallID  string          `json:"toolCallId,omitempty"`
	Title       string          `json:"title,omitempty"`
	Input       json.RawMessage `json:"input,omitempty"`
	RawInput    json.RawMessage `json:"rawInput,omitempty"`
	Output      json.RawMessage `json:"output,omitempty"`
	ErrorText   string          `json:"errorText,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
	Preliminary bool            `json:"preliminary,omitempty"`
}

// UIAgentEventPayload mirrors the prompt-stream data payload shape.
type UIAgentEventPayload struct {
	Type       string               `json:"type"`
	SessionID  string               `json:"session_id,omitempty"`
	TurnID     string               `json:"turn_id,omitempty"`
	RequestID  string               `json:"request_id,omitempty"`
	Timestamp  string               `json:"timestamp,omitempty"`
	Text       string               `json:"text,omitempty"`
	Title      string               `json:"title,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
	StopReason string               `json:"stop_reason,omitempty"`
	Action     string               `json:"action,omitempty"`
	Resource   string               `json:"resource,omitempty"`
	Decision   string               `json:"decision,omitempty"`
	Error      string               `json:"error,omitempty"`
	Usage      *UITokenUsagePayload `json:"usage,omitempty"`
	Runtime    *acp.RuntimeActivity `json:"runtime,omitempty"`
	Raw        json.RawMessage      `json:"raw,omitempty"`
}

// UITokenUsagePayload mirrors the prompt-stream token usage payload.
type UITokenUsagePayload struct {
	TurnID           string   `json:"turn_id,omitempty"`
	InputTokens      *int64   `json:"input_tokens,omitempty"`
	OutputTokens     *int64   `json:"output_tokens,omitempty"`
	TotalTokens      *int64   `json:"total_tokens,omitempty"`
	ThoughtTokens    *int64   `json:"thought_tokens,omitempty"`
	CacheReadTokens  *int64   `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens *int64   `json:"cache_write_tokens,omitempty"`
	ContextUsed      *int64   `json:"context_used,omitempty"`
	ContextSize      *int64   `json:"context_size,omitempty"`
	CostAmount       *float64 `json:"cost_amount,omitempty"`
	CostCurrency     *string  `json:"cost_currency,omitempty"`
	Timestamp        string   `json:"timestamp,omitempty"`
}

type decodedStoredEvent struct {
	stored store.SessionEvent
	parsed event
	agent  acp.AgentEvent
}

type uiMessageBuilder struct {
	id             string
	role           string
	finished       bool
	parts          []UIMessagePart
	textIndex      int
	reasoningIndex int
	toolIndices    map[string]int
}

// UIAgentEventPayloadFromEvent converts an ACP event into the prompt-stream data payload.
func UIAgentEventPayloadFromEvent(event acp.AgentEvent) UIAgentEventPayload {
	payload := UIAgentEventPayload{
		Type:       event.Type,
		SessionID:  event.SessionID,
		TurnID:     event.TurnID,
		RequestID:  event.RequestID,
		Text:       event.Text,
		Title:      event.Title,
		ToolCallID: event.ToolCallID,
		StopReason: event.StopReason,
		Action:     event.Action,
		Resource:   event.Resource,
		Decision:   event.Decision,
		Error:      event.Error,
		Usage:      uiTokenUsagePayloadFromUsage(event.Usage),
		Runtime:    cloneRuntimeActivity(event.Runtime),
		Raw:        payloadJSONBytes(event.Raw),
	}
	if !event.Timestamp.IsZero() {
		payload.Timestamp = event.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return payload
}

func uiTokenUsagePayloadFromUsage(usage *acp.TokenUsage) *UITokenUsagePayload {
	if usage == nil {
		return nil
	}

	payload := &UITokenUsagePayload{
		TurnID:           usage.TurnID,
		InputTokens:      usage.InputTokens,
		OutputTokens:     usage.OutputTokens,
		TotalTokens:      usage.TotalTokens,
		ThoughtTokens:    usage.ThoughtTokens,
		CacheReadTokens:  usage.CacheReadTokens,
		CacheWriteTokens: usage.CacheWriteTokens,
		ContextUsed:      usage.ContextUsed,
		ContextSize:      usage.ContextSize,
		CostAmount:       usage.CostAmount,
		CostCurrency:     usage.CostCurrency,
	}
	if !usage.Timestamp.IsZero() {
		payload.Timestamp = usage.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return payload
}

func payloadJSONBytes(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	if json.Valid(raw) {
		return acp.CloneRawMessage(raw)
	}
	return rawMessageFromValue(string(raw))
}

// ToUIMessages projects persisted session events into AI SDK UIMessage objects.
func ToUIMessages(events []store.SessionEvent) ([]UIMessage, error) {
	if len(events) == 0 {
		return []UIMessage{}, nil
	}

	sorted := sortedTranscriptEvents(events)
	messages := make([]UIMessage, 0, len(sorted))
	var assistant *uiMessageBuilder

	flushAssistant := func(forceComplete bool) {
		if assistant == nil {
			return
		}
		if message := assistant.build(forceComplete || assistant.finished); message != nil {
			messages = append(messages, *message)
		}
		assistant = nil
	}

	for _, storedEvent := range sorted {
		decoded := decodeStoredEvent(storedEvent)
		switch decoded.parsed.Type {
		case acp.EventTypeUserMessage:
			flushAssistant(true)
			if message := inputUIMessage(decoded, UIRoleUser); message != nil {
				messages = append(messages, *message)
			}
		case acp.EventTypeSyntheticReentry:
			flushAssistant(true)
			if message := inputUIMessage(decoded, UIRoleSystem); message != nil {
				messages = append(messages, *message)
			}
		default:
			assistantID := assistantMessageID(decoded)
			if assistant == nil || assistant.id != assistantID {
				flushAssistant(true)
				assistant = newUIMessageBuilder(assistantID, UIRoleAssistant)
			}
			applyDecodedEvent(assistant, decoded)
		}
	}

	flushAssistant(false)
	return messages, nil
}

func newUIMessageBuilder(id string, role string) *uiMessageBuilder {
	return &uiMessageBuilder{
		id:             id,
		role:           role,
		textIndex:      -1,
		reasoningIndex: -1,
		toolIndices:    make(map[string]int),
	}
}

func (b *uiMessageBuilder) build(complete bool) *UIMessage {
	if b == nil || len(b.parts) == 0 {
		return nil
	}

	parts := cloneUIMessageParts(b.parts)
	for index := range parts {
		switch parts[index].Type {
		case uiPartText, uiPartReasoning:
			if complete {
				parts[index].State = uiPartStateDone
			} else if parts[index].State == "" {
				parts[index].State = uiPartStateStreaming
			}
		}
	}

	return &UIMessage{
		ID:    b.id,
		Role:  b.role,
		Parts: parts,
	}
}

func cloneUIMessageParts(parts []UIMessagePart) []UIMessagePart {
	cloned := make([]UIMessagePart, 0, len(parts))
	for _, part := range parts {
		next := part
		next.Input = acp.CloneRawMessage(part.Input)
		next.RawInput = acp.CloneRawMessage(part.RawInput)
		next.Output = acp.CloneRawMessage(part.Output)
		next.Data = acp.CloneRawMessage(part.Data)
		cloned = append(cloned, next)
	}
	return cloned
}

func applyDecodedEvent(builder *uiMessageBuilder, decoded *decodedStoredEvent) {
	switch decoded.parsed.Type {
	case acp.EventTypeAgentMessage:
		builder.appendText(decoded.parsed.Text)
	case acp.EventTypeThought:
		builder.appendReasoning(decoded.parsed.Text)
	case acp.EventTypeToolCall:
		builder.applyToolCall(decoded)
		builder.appendDataPart(uiPartDataEvent, decoded.dataPayload())
	case acp.EventTypeToolResult:
		builder.applyToolResult(decoded)
	case acp.EventTypePermission:
		builder.appendDataPart(uiPartDataPermission, decoded.dataPayload())
	case acp.EventTypeDone, acp.EventTypeError:
		builder.finished = true
	default:
		builder.appendDataPart(uiPartDataEvent, decoded.dataPayload())
	}
}

func (b *uiMessageBuilder) appendText(text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" && b.textIndex < 0 {
		return
	}
	index := b.ensureTextPart()
	b.parts[index].Text += text
	if b.parts[index].State == "" {
		b.parts[index].State = uiPartStateStreaming
	}
}

func (b *uiMessageBuilder) appendReasoning(text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" && b.reasoningIndex < 0 {
		return
	}
	index := b.ensureReasoningPart()
	b.parts[index].Text += text
	if b.parts[index].State == "" {
		b.parts[index].State = uiPartStateStreaming
	}
}

func (b *uiMessageBuilder) ensureTextPart() int {
	if b.textIndex >= 0 {
		return b.textIndex
	}
	b.parts = append(b.parts, UIMessagePart{Type: uiPartText})
	b.textIndex = len(b.parts) - 1
	return b.textIndex
}

func (b *uiMessageBuilder) ensureReasoningPart() int {
	if b.reasoningIndex >= 0 {
		return b.reasoningIndex
	}
	b.parts = append(b.parts, UIMessagePart{Type: uiPartReasoning})
	b.reasoningIndex = len(b.parts) - 1
	return b.reasoningIndex
}

func (b *uiMessageBuilder) appendDataPart(partType string, payload json.RawMessage) {
	if len(payload) == 0 {
		return
	}
	b.parts = append(b.parts, UIMessagePart{
		Type: partType,
		Data: acp.CloneRawMessage(payload),
	})
}

func (b *uiMessageBuilder) applyToolCall(decoded *decodedStoredEvent) {
	part, _ := b.ensureToolPart(decoded)
	if input, ready := toolInputFromDecoded(decoded); ready {
		part.State = uiToolStateAvailable
		part.Input = input
		return
	}
	if part.State == "" {
		part.State = uiToolStateStreaming
	}
}

func (b *uiMessageBuilder) applyToolResult(decoded *decodedStoredEvent) {
	part, existed := b.ensureToolPart(decoded)
	input := acp.CloneRawMessage(part.Input)
	if len(input) == 0 {
		if next, ready := toolInputFromDecoded(decoded); ready {
			input = next
		} else {
			input = acp.CloneRawMessage(emptyJSONObject)
		}
	}

	part.State = uiToolStateOutput
	part.Input = input
	part.Output = decoded.dataPayload()
	part.ErrorText = ""

	if !existed {
		part.RawInput = nil
	}
}

func (b *uiMessageBuilder) ensureToolPart(decoded *decodedStoredEvent) (*UIMessagePart, bool) {
	key := toolLifecycleKey(decoded.parsed)
	if key == "" {
		key = fallbackMessageID(decoded.stored.ID, assistantMessageID(decoded), "tool")
	}
	if index, ok := b.toolIndices[key]; ok {
		part := &b.parts[index]
		if part.Type == uiPartDynamicTool && strings.TrimSpace(decoded.parsed.ToolName) != "" {
			part.Type = toolPartType(decoded.parsed.ToolName)
			part.ToolName = ""
		}
		if part.Title == "" {
			part.Title = strings.TrimSpace(decoded.agent.Title)
		}
		return part, true
	}

	part := UIMessagePart{
		Type:       toolPartKind(decoded.parsed.ToolName),
		ToolCallID: key,
		Title:      strings.TrimSpace(decoded.agent.Title),
	}
	if part.Type == uiPartDynamicTool {
		part.ToolName = strings.TrimSpace(decoded.parsed.ToolName)
	}
	b.parts = append(b.parts, part)
	index := len(b.parts) - 1
	b.toolIndices[key] = index
	return &b.parts[index], false
}

func toolPartKind(toolName string) string {
	if strings.TrimSpace(toolName) == "" {
		return uiPartDynamicTool
	}
	return toolPartType(toolName)
}

func toolPartType(toolName string) string {
	return "tool-" + strings.TrimSpace(toolName)
}

func inputUIMessage(decoded *decodedStoredEvent, role string) *UIMessage {
	text := strings.TrimSpace(decoded.parsed.Text)
	if text == "" {
		return nil
	}
	return &UIMessage{
		ID:   inputMessageID(decoded, role),
		Role: role,
		Parts: []UIMessagePart{{
			Type:  uiPartText,
			Text:  decoded.parsed.Text,
			State: uiPartStateDone,
		}},
	}
}

// UIMessageText returns the concatenated visible text parts for one UI message.
func UIMessageText(message UIMessage) string {
	parts := make([]string, 0, len(message.Parts))
	for _, part := range message.Parts {
		if part.Type != uiPartText {
			continue
		}
		if strings.TrimSpace(part.Text) == "" {
			continue
		}
		parts = append(parts, part.Text)
	}
	return strings.Join(parts, "\n")
}

// JoinUIMessageText joins visible text parts across all messages with newlines.
func JoinUIMessageText(messages []UIMessage) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		if text := strings.TrimSpace(UIMessageText(message)); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func decodeStoredEvent(storedEvent store.SessionEvent) *decodedStoredEvent {
	decoded := &decodedStoredEvent{
		stored: storedEvent,
		parsed: parseEvent(storedEvent),
	}
	if event, err := UnmarshalAgentEvent(strings.TrimSpace(storedEvent.Content)); err == nil {
		decoded.agent = event
	} else {
		decoded.agent = fallbackAgentEvent(decoded.parsed, storedEvent)
	}

	if strings.TrimSpace(decoded.agent.Type) == "" {
		decoded.agent.Type = decoded.parsed.Type
	}
	if strings.TrimSpace(decoded.agent.SessionID) == "" {
		decoded.agent.SessionID = strings.TrimSpace(storedEvent.SessionID)
	}
	if strings.TrimSpace(decoded.agent.TurnID) == "" {
		decoded.agent.TurnID = firstNonEmpty(
			strings.TrimSpace(decoded.parsed.TurnID),
			strings.TrimSpace(storedEvent.TurnID),
		)
	}
	if decoded.agent.Timestamp.IsZero() && !storedEvent.Timestamp.IsZero() {
		decoded.agent.Timestamp = storedEvent.Timestamp.UTC()
	}
	if strings.TrimSpace(decoded.agent.Text) == "" {
		decoded.agent.Text = decoded.parsed.Text
	}
	if strings.TrimSpace(decoded.agent.ToolCallID) == "" {
		decoded.agent.ToolCallID = decoded.parsed.ToolCallID
	}
	if strings.TrimSpace(decoded.agent.Title) == "" {
		decoded.agent.Title = decoded.parsed.ToolName
	}
	if strings.TrimSpace(decoded.agent.Error) == "" && decoded.parsed.ToolResult != nil {
		decoded.agent.Error = firstNonEmpty(decoded.parsed.ToolResult.Error, decoded.parsed.Text)
	}
	return decoded
}

func fallbackAgentEvent(parsed event, storedEvent store.SessionEvent) acp.AgentEvent {
	event := acp.AgentEvent{
		Type:       parsed.Type,
		SessionID:  strings.TrimSpace(storedEvent.SessionID),
		TurnID:     firstNonEmpty(strings.TrimSpace(parsed.TurnID), strings.TrimSpace(storedEvent.TurnID)),
		Timestamp:  storedEvent.Timestamp.UTC(),
		Text:       parsed.Text,
		Title:      parsed.ToolName,
		ToolCallID: parsed.ToolCallID,
	}

	content := strings.TrimSpace(storedEvent.Content)
	if content == "" {
		return event
	}
	if json.Valid([]byte(content)) {
		event.Raw = acp.CloneRawMessage(json.RawMessage(content))

		var payload map[string]any
		if err := json.Unmarshal([]byte(content), &payload); err == nil {
			event.RequestID = firstNonEmpty(
				nestedString(payload, "request_id"),
				nestedString(payload, "requestId"),
			)
			event.Action = nestedString(payload, "action")
			event.Resource = nestedString(payload, "resource")
			event.Decision = nestedString(payload, "decision")
			event.Error = firstNonEmpty(nestedString(payload, "error"), event.Error)
			event.StopReason = nestedString(payload, "stop_reason")
		}
	}

	return event
}

func (d *decodedStoredEvent) dataPayload() json.RawMessage {
	if d == nil {
		return nil
	}
	payload := UIAgentEventPayloadFromEvent(d.agent)
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return json.RawMessage(encoded)
}

func toolInputFromDecoded(decoded *decodedStoredEvent) (json.RawMessage, bool) {
	if decoded == nil {
		return nil, false
	}
	if len(decoded.parsed.ToolInput) == 0 {
		return nil, false
	}
	var value any
	if err := json.Unmarshal(decoded.parsed.ToolInput, &value); err != nil {
		return acp.CloneRawMessage(decoded.parsed.ToolInput), true
	}
	if !toolInputReadyValue(value) {
		return nil, false
	}
	return acp.CloneRawMessage(decoded.parsed.ToolInput), true
}

func toolInputReadyValue(input any) bool {
	switch value := input.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(value) != ""
	case []any:
		return len(value) > 0
	case map[string]any:
		return len(value) > 0
	default:
		return true
	}
}

func inputMessageID(decoded *decodedStoredEvent, role string) string {
	suffix := role
	if role == UIRoleSystem {
		suffix = "system"
	}
	return fallbackMessageID(strings.TrimSpace(decoded.stored.ID), strings.TrimSpace(decoded.parsed.TurnID), suffix)
}

func assistantMessageID(decoded *decodedStoredEvent) string {
	return fallbackMessageID(
		strings.TrimSpace(decoded.parsed.TurnID),
		strings.TrimSpace(decoded.stored.ID),
		"assistant",
	)
}

func fallbackMessageID(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return fmt.Sprintf("msg-%s", CanonicalSchema)
}
