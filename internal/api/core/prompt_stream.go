package core

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/api/contract"
	"github.com/pedronauck/agh/internal/diagnostics"
	ssepkg "github.com/pedronauck/agh/internal/sse"
	taskpkg "github.com/pedronauck/agh/internal/task"
)

type promptAgentEventPayload struct {
	Type       string                           `json:"type"`
	SessionID  string                           `json:"session_id,omitempty"`
	TurnID     string                           `json:"turn_id,omitempty"`
	RequestID  string                           `json:"request_id,omitempty"`
	Timestamp  string                           `json:"timestamp,omitempty"`
	Text       string                           `json:"text,omitempty"`
	Title      string                           `json:"title,omitempty"`
	ToolCallID string                           `json:"tool_call_id,omitempty"`
	StopReason string                           `json:"stop_reason,omitempty"`
	Action     string                           `json:"action,omitempty"`
	Resource   string                           `json:"resource,omitempty"`
	Decision   string                           `json:"decision,omitempty"`
	Error      string                           `json:"error,omitempty"`
	Usage      *promptTokenUsagePayload         `json:"usage,omitempty"`
	Runtime    *contract.RuntimeActivityPayload `json:"runtime,omitempty"`
	Raw        json.RawMessage                  `json:"raw,omitempty"`
}

type promptTokenUsagePayload struct {
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

type promptFinishPayload struct {
	Type         string `json:"type"`
	FinishReason string `json:"finishReason,omitempty"`
}

// PromptStreamEncoder converts raw ACP agent events into the typed public
// prompt stream envelope used by HTTP, UDS, and CLI streaming surfaces.
type PromptStreamEncoder struct {
	now              func() string
	messageID        string
	textBlockID      string
	reasoningBlockID string
	messageStarted   bool
	textStarted      bool
	reasoningStarted bool
	toolStarted      map[string]struct{}
	toolInputsReady  map[string]struct{}
	toolInputPending map[string]struct{}
	toolNames        map[string]string
	finished         bool
}

// NewPromptStreamEncoder constructs a prompt-stream encoder with deterministic
// timestamps for generated IDs when provided.
func NewPromptStreamEncoder(now func() time.Time) *PromptStreamEncoder {
	clock := func() string {
		return time.Now().UTC().Format(time.RFC3339Nano)
	}
	if now != nil {
		clock = func() string {
			return now().UTC().Format(time.RFC3339Nano)
		}
	}

	return &PromptStreamEncoder{
		now:              clock,
		toolStarted:      make(map[string]struct{}),
		toolInputsReady:  make(map[string]struct{}),
		toolInputPending: make(map[string]struct{}),
		toolNames:        make(map[string]string),
	}
}

// Emit writes one public prompt-stream frame for the supplied raw agent event.
func (e *PromptStreamEncoder) Emit(writer FlushWriter, event acp.AgentEvent) error {
	e.ensureInitialized()
	if err := e.ensureMessageStarted(writer, event); err != nil {
		return err
	}

	switch event.Type {
	case acp.EventTypeAgentMessage:
		return e.emitAgentMessage(writer, event)
	case acp.EventTypeThought:
		return e.emitThought(writer, event)
	case acp.EventTypeToolCall:
		return e.emitToolCall(writer, event)
	case acp.EventTypeToolResult:
		return e.emitToolResult(writer, event)
	case acp.EventTypePermission:
		return e.emitPermission(writer, event)
	case acp.EventTypeError:
		return e.emitError(writer, event)
	case acp.EventTypeDone:
		return e.finish(writer, event)
	default:
		return e.emitGenericEvent(writer, event)
	}
}

// Finish closes any open blocks and emits the terminal finish frame followed by
// the `[DONE]` sentinel exactly once.
func (e *PromptStreamEncoder) Finish(writer FlushWriter, event acp.AgentEvent) error {
	e.ensureInitialized()
	return e.finish(writer, event)
}

func (e *PromptStreamEncoder) ensureInitialized() {
	if e.now == nil {
		e.now = func() string {
			return time.Now().UTC().Format(time.RFC3339Nano)
		}
	}
	if e.toolStarted == nil {
		e.toolStarted = make(map[string]struct{})
	}
	if e.toolInputsReady == nil {
		e.toolInputsReady = make(map[string]struct{})
	}
	if e.toolInputPending == nil {
		e.toolInputPending = make(map[string]struct{})
	}
	if e.toolNames == nil {
		e.toolNames = make(map[string]string)
	}
}

func (e *PromptStreamEncoder) emitAgentMessage(writer FlushWriter, event acp.AgentEvent) error {
	if err := e.ensureTextStarted(writer); err != nil {
		return err
	}
	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type":  "text-delta",
			"id":    e.textBlockID,
			"delta": event.Text,
		},
	})
}

func (e *PromptStreamEncoder) emitThought(writer FlushWriter, event acp.AgentEvent) error {
	if err := e.ensureReasoningStarted(writer); err != nil {
		return err
	}
	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type":  "reasoning-delta",
			"id":    e.reasoningBlockID,
			"delta": event.Text,
		},
	})
}

func (e *PromptStreamEncoder) emitToolCall(writer FlushWriter, event acp.AgentEvent) error {
	toolCallID := e.toolCallID(event)
	if err := e.ensureToolCallStarted(writer, toolCallID, event); err != nil {
		return err
	}
	if err := e.ensureToolInputAvailable(writer, toolCallID, event, false); err != nil {
		return err
	}
	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type": "data-agh-event",
			"data": promptAgentEventPayloadFromEvent(event),
		},
	})
}

func (e *PromptStreamEncoder) emitToolResult(writer FlushWriter, event acp.AgentEvent) error {
	toolCallID := e.toolCallID(event)
	if err := e.ensureToolCallStarted(writer, toolCallID, event); err != nil {
		return err
	}
	if err := e.ensureToolInputAvailable(writer, toolCallID, event, true); err != nil {
		return err
	}
	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type":       "tool-output-available",
			"toolCallId": toolCallID,
			"output":     promptAgentEventPayloadFromEvent(event),
		},
	})
}

func (e *PromptStreamEncoder) emitPermission(writer FlushWriter, event acp.AgentEvent) error {
	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type": "data-agh-permission",
			"data": promptAgentEventPayloadFromEvent(event),
		},
	})
}

func (e *PromptStreamEncoder) emitError(writer FlushWriter, event acp.AgentEvent) error {
	if err := e.closeOpenBlocks(writer); err != nil {
		return err
	}
	if err := WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type":      "error",
			"errorText": e.errorText(event),
		},
	}); err != nil {
		return err
	}
	return e.finish(writer, event)
}

func (e *PromptStreamEncoder) emitGenericEvent(writer FlushWriter, event acp.AgentEvent) error {
	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type": "data-agh-event",
			"data": promptAgentEventPayloadFromEvent(event),
		},
	})
}

func (e *PromptStreamEncoder) toolCallID(event acp.AgentEvent) string {
	toolCallID := strings.TrimSpace(event.ToolCallID)
	if toolCallID == "" {
		return e.messageID + "-tool"
	}
	return toolCallID
}

func (e *PromptStreamEncoder) ensureToolCallStarted(
	writer FlushWriter,
	toolCallID string,
	event acp.AgentEvent,
) error {
	if _, ok := e.toolStarted[toolCallID]; ok {
		if toolName := e.toolName(event); toolName != "" {
			e.toolNames[toolCallID] = toolName
		}
		return nil
	}

	e.toolStarted[toolCallID] = struct{}{}
	if toolName := e.toolName(event); toolName != "" {
		e.toolNames[toolCallID] = toolName
	}

	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type":       "tool-input-start",
			"toolCallId": toolCallID,
			"toolName":   e.toolNameByID(toolCallID),
		},
	})
}

func (e *PromptStreamEncoder) ensureToolInputAvailable(
	writer FlushWriter,
	toolCallID string,
	event acp.AgentEvent,
	force bool,
) error {
	if _, ok := e.toolInputsReady[toolCallID]; ok {
		return nil
	}

	input, ok := promptNormalizedToolInput(event)
	if !ok || !promptToolInputReady(input) {
		if !force {
			return nil
		}
		if _, ok := e.toolInputPending[toolCallID]; ok {
			return nil
		}
		e.toolInputPending[toolCallID] = struct{}{}
		input = map[string]any{}
	} else {
		delete(e.toolInputPending, toolCallID)
		e.toolInputsReady[toolCallID] = struct{}{}
	}

	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type":       "tool-input-available",
			"toolCallId": toolCallID,
			"toolName":   e.toolNameByID(toolCallID),
			"input":      input,
		},
	})
}

func (e *PromptStreamEncoder) toolNameByID(toolCallID string) string {
	toolName := strings.TrimSpace(e.toolNames[toolCallID])
	if toolName == "" {
		return "tool"
	}
	return toolName
}

func (e *PromptStreamEncoder) toolName(event acp.AgentEvent) string {
	toolName := strings.TrimSpace(event.Title)
	if toolName != "" {
		return toolName
	}

	rawPayload := promptRawEventMap(event.Raw)
	if toolName = strings.TrimSpace(promptStringValue(rawPayload["tool_name"])); toolName != "" {
		return toolName
	}
	if toolName = strings.TrimSpace(promptStringValue(rawPayload["title"])); toolName != "" {
		return toolName
	}

	meta := promptMapValue(rawPayload["_meta"])
	claudeCode := promptMapValue(meta["claudeCode"])
	return strings.TrimSpace(promptStringValue(claudeCode["toolName"]))
}

func (e *PromptStreamEncoder) errorText(event acp.AgentEvent) string {
	errorText := strings.TrimSpace(event.Error)
	if errorText == "" {
		errorText = strings.TrimSpace(event.Text)
	}
	return errorText
}

func (e *PromptStreamEncoder) ensureMessageStarted(writer FlushWriter, event acp.AgentEvent) error {
	if e.messageStarted {
		return nil
	}

	messageID := strings.TrimSpace(event.TurnID)
	if messageID == "" {
		messageID = e.messageID
	}
	if messageID == "" {
		messageID = "msg-" + strings.ReplaceAll(e.now(), ":", "-")
	}
	e.messageID = messageID
	e.textBlockID = messageID + "-text"
	e.reasoningBlockID = messageID + "-reasoning"
	e.messageStarted = true

	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type":      "start",
			"messageId": e.messageID,
		},
	})
}

func (e *PromptStreamEncoder) ensureTextStarted(writer FlushWriter) error {
	if e.textStarted {
		return nil
	}
	e.textStarted = true
	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type": "text-start",
			"id":   e.textBlockID,
		},
	})
}

func (e *PromptStreamEncoder) ensureReasoningStarted(writer FlushWriter) error {
	if e.reasoningStarted {
		return nil
	}
	e.reasoningStarted = true
	return WriteSSE(writer, SSEMessage{
		Data: map[string]any{
			"type": "reasoning-start",
			"id":   e.reasoningBlockID,
		},
	})
}

func (e *PromptStreamEncoder) closeOpenBlocks(writer FlushWriter) error {
	if e.textStarted {
		if err := WriteSSE(writer, SSEMessage{
			Data: map[string]any{
				"type": "text-end",
				"id":   e.textBlockID,
			},
		}); err != nil {
			return err
		}
		e.textStarted = false
	}
	if e.reasoningStarted {
		if err := WriteSSE(writer, SSEMessage{
			Data: map[string]any{
				"type": "reasoning-end",
				"id":   e.reasoningBlockID,
			},
		}); err != nil {
			return err
		}
		e.reasoningStarted = false
	}
	return nil
}

func (e *PromptStreamEncoder) finish(writer FlushWriter, event acp.AgentEvent) error {
	if e.finished {
		return nil
	}
	if err := e.ensureMessageStarted(writer, event); err != nil {
		return err
	}
	if err := e.closeOpenBlocks(writer); err != nil {
		return err
	}
	e.finished = true

	finishPayload := promptFinishPayload{Type: "finish"}
	if finishReason := promptAISDKFinishReason(event.StopReason); finishReason != "" {
		finishPayload.FinishReason = finishReason
	}

	if err := WriteSSE(writer, SSEMessage{
		Data: finishPayload,
	}); err != nil {
		return err
	}
	return WriteSSERaw(writer, "", "[DONE]")
}

func promptAISDKFinishReason(stopReason string) string {
	switch strings.TrimSpace(stopReason) {
	case "", "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "canceled", "max_turn_requests", "refusal":
		return "other"
	default:
		return "other"
	}
}

func promptAgentEventPayloadFromEvent(event acp.AgentEvent) promptAgentEventPayload {
	base := AgentEventPayloadFromEvent(event)
	payload := promptAgentEventPayload{
		Type:       base.Type,
		SessionID:  base.SessionID,
		TurnID:     base.TurnID,
		RequestID:  base.RequestID,
		Text:       promptRedactString(base.Text),
		Title:      promptRedactString(base.Title),
		ToolCallID: base.ToolCallID,
		StopReason: promptRedactString(base.StopReason),
		Action:     promptRedactString(base.Action),
		Resource:   promptRedactString(base.Resource),
		Decision:   promptRedactString(base.Decision),
		Error:      promptRedactString(base.Error),
		Usage:      promptTokenUsagePayloadFromUsage(event.Usage),
		Runtime:    base.Runtime,
		Raw:        promptRedactRaw(base.Raw),
	}
	if !event.Timestamp.IsZero() {
		payload.Timestamp = event.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return payload
}

func promptTokenUsagePayloadFromUsage(usage *acp.TokenUsage) *promptTokenUsagePayload {
	base := TokenUsagePayloadFromUsage(usage)
	if base == nil {
		return nil
	}

	payload := &promptTokenUsagePayload{
		TurnID:           base.TurnID,
		InputTokens:      base.InputTokens,
		OutputTokens:     base.OutputTokens,
		TotalTokens:      base.TotalTokens,
		ThoughtTokens:    base.ThoughtTokens,
		CacheReadTokens:  base.CacheReadTokens,
		CacheWriteTokens: base.CacheWriteTokens,
		ContextUsed:      base.ContextUsed,
		ContextSize:      base.ContextSize,
		CostAmount:       base.CostAmount,
		CostCurrency:     base.CostCurrency,
	}
	if !base.Timestamp.IsZero() {
		payload.Timestamp = base.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return payload
}

func promptNormalizedToolInput(event acp.AgentEvent) (any, bool) {
	rawPayload := promptRawEventMap(event.Raw)
	if len(rawPayload) == 0 {
		return nil, false
	}

	input, ok := promptFirstNonNil(
		rawPayload["tool_input"],
		rawPayload["rawInput"],
	)
	if !ok || input == nil {
		return nil, false
	}

	return promptRedactValue(input), true
}

func promptRawEventMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	return payload
}

func promptToolInputReady(input any) bool {
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

func promptFirstNonNil(values ...any) (any, bool) {
	for _, value := range values {
		if value != nil {
			return value, true
		}
	}
	return nil, false
}

func promptRedactRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err == nil {
		redacted, marshalErr := json.Marshal(promptRedactValue(payload))
		if marshalErr == nil {
			return redacted
		}
	}
	return PayloadJSON(promptRedactString(string(raw)))
}

func promptRedactValue(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return promptRedactString(typed)
	case map[string]any:
		redacted := make(map[string]any, len(typed))
		for key, nested := range typed {
			if promptKeyCarriesSecret(key) {
				redacted[key] = "[REDACTED]"
				continue
			}
			redacted[key] = promptRedactValue(nested)
		}
		return redacted
	case []any:
		redacted := make([]any, 0, len(typed))
		for _, nested := range typed {
			redacted = append(redacted, promptRedactValue(nested))
		}
		return redacted
	default:
		return typed
	}
}

func promptRedactString(value string) string {
	return ssepkg.ScrubMemoryContextString(diagnostics.Redact(taskpkg.RedactClaimTokens(value)))
}

func promptKeyCarriesSecret(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	if normalized == "" {
		return false
	}
	for _, marker := range []string{
		"apikey",
		"accesstoken",
		"refreshtoken",
		"mcpauthtoken",
		"oauthcode",
		"authorizationcode",
		"codeverifier",
		"pkceverifier",
		"secretbinding",
		"authorization",
		"password",
		"secret",
		"token",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func promptStringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func promptMapValue(value any) map[string]any {
	if payload, ok := value.(map[string]any); ok {
		return payload
	}
	return nil
}
