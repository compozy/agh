package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	core "github.com/pedronauck/agh/internal/api/core"
)

type promptRequest struct {
	Message  string              `json:"message"`
	Messages []uiMessageEnvelope `json:"messages"`
}

type uiMessageEnvelope struct {
	Role    string              `json:"role"`
	Content string              `json:"content"`
	Parts   []uiMessageTextPart `json:"parts"`
}

type uiMessageTextPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type agentEventPayload struct {
	Type       string             `json:"type"`
	SessionID  string             `json:"session_id,omitempty"`
	TurnID     string             `json:"turn_id,omitempty"`
	RequestID  string             `json:"request_id,omitempty"`
	Timestamp  string             `json:"timestamp,omitempty"`
	Text       string             `json:"text,omitempty"`
	Title      string             `json:"title,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
	StopReason string             `json:"stop_reason,omitempty"`
	Action     string             `json:"action,omitempty"`
	Resource   string             `json:"resource,omitempty"`
	Decision   string             `json:"decision,omitempty"`
	Error      string             `json:"error,omitempty"`
	Usage      *tokenUsagePayload `json:"usage,omitempty"`
	Raw        json.RawMessage    `json:"raw,omitempty"`
}

type tokenUsagePayload struct {
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

type promptStreamState struct {
	now              func() string
	messageID        string
	textBlockID      string
	reasoningBlockID string
	messageStarted   bool
	textStarted      bool
	reasoningStarted bool
	toolStarted      map[string]struct{}
	finished         bool
}

func (h *Handlers) promptSession(c *gin.Context) {
	var req promptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.Logger.Debug("httpapi: decode prompt request failed", "error", err)
		core.RespondError(c, http.StatusBadRequest, errors.New("invalid request payload"), true)
		return
	}

	message, err := extractPromptMessage(req)
	if err != nil {
		core.RespondError(c, http.StatusBadRequest, err, true)
		return
	}

	events, err := h.Sessions.Prompt(c.Request.Context(), c.Param("id"), message)
	if err != nil {
		core.RespondError(c, core.StatusForSessionError(err), err, true)
		return
	}

	c.Header("x-vercel-ai-ui-message-stream", "v1")
	writer, err := core.PrepareSSE(c)
	if err != nil {
		core.RespondError(c, http.StatusInternalServerError, err, true)
		return
	}

	state := &promptStreamState{
		now: func() string {
			return h.Now().UTC().Format(time.RFC3339Nano)
		},
		toolStarted: make(map[string]struct{}),
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.StreamDoneChannel():
			return
		case event, ok := <-events:
			if !ok {
				if err := state.finish(writer, acp.AgentEvent{}); err != nil {
					return
				}
				return
			}
			if err := state.emit(writer, event); err != nil {
				return
			}
		}
	}
}

func extractPromptMessage(req promptRequest) (string, error) {
	if message := strings.TrimSpace(req.Message); message != "" {
		return message, nil
	}

	for i := len(req.Messages) - 1; i >= 0; i-- {
		msg := req.Messages[i]
		if strings.TrimSpace(msg.Role) != "user" {
			continue
		}

		if content := strings.TrimSpace(msg.Content); content != "" {
			return content, nil
		}

		parts := make([]string, 0, len(msg.Parts))
		for _, part := range msg.Parts {
			partType := strings.TrimSpace(part.Type)
			if partType != "" && !strings.EqualFold(partType, "text") {
				continue
			}
			if text := strings.TrimSpace(part.Text); text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n"), nil
		}
	}

	return "", errors.New("message is required")
}

func (s *promptStreamState) emit(writer core.FlushWriter, event acp.AgentEvent) error {
	if err := s.ensureMessageStarted(writer, event); err != nil {
		return err
	}

	switch event.Type {
	case acp.EventTypeAgentMessage:
		return s.emitAgentMessage(writer, event)
	case acp.EventTypeThought:
		return s.emitThought(writer, event)
	case acp.EventTypeToolCall:
		return s.emitToolCall(writer, event)
	case acp.EventTypeToolResult:
		return s.emitToolResult(writer, event)
	case acp.EventTypePermission:
		return s.emitPermission(writer, event)
	case acp.EventTypeError:
		return s.emitError(writer, event)
	case acp.EventTypeDone:
		return s.finish(writer, event)
	default:
		return s.emitGenericEvent(writer, event)
	}
}

func (s *promptStreamState) emitAgentMessage(writer core.FlushWriter, event acp.AgentEvent) error {
	if err := s.ensureTextStarted(writer); err != nil {
		return err
	}
	return core.WriteSSE(writer, core.SSEMessage{
		Name: "agent_message",
		Data: map[string]any{
			"type":  "text-delta",
			"id":    s.textBlockID,
			"delta": event.Text,
		},
	})
}

func (s *promptStreamState) emitThought(writer core.FlushWriter, event acp.AgentEvent) error {
	if err := s.ensureReasoningStarted(writer); err != nil {
		return err
	}
	return core.WriteSSE(writer, core.SSEMessage{
		Name: "thought",
		Data: map[string]any{
			"type":  "reasoning-delta",
			"id":    s.reasoningBlockID,
			"delta": event.Text,
		},
	})
}

func (s *promptStreamState) emitToolCall(writer core.FlushWriter, event acp.AgentEvent) error {
	toolCallID := s.toolCallID(event)
	if err := s.ensureToolCallStarted(writer, toolCallID, event); err != nil {
		return err
	}
	return core.WriteSSE(writer, core.SSEMessage{
		Name: "tool_call",
		Data: map[string]any{
			"type":       "data-agh-event",
			"data":       agentEventPayloadFromEvent(event),
			"toolCallId": toolCallID,
		},
	})
}

func (s *promptStreamState) emitToolResult(writer core.FlushWriter, event acp.AgentEvent) error {
	return core.WriteSSE(writer, core.SSEMessage{
		Name: "tool_result",
		Data: map[string]any{
			"type":       "tool-output-available",
			"toolCallId": s.toolCallID(event),
			"output":     agentEventPayloadFromEvent(event),
		},
	})
}

func (s *promptStreamState) emitPermission(writer core.FlushWriter, event acp.AgentEvent) error {
	return core.WriteSSE(writer, core.SSEMessage{
		Name: "permission",
		Data: map[string]any{
			"type": "data-agh-permission",
			"data": agentEventPayloadFromEvent(event),
		},
	})
}

func (s *promptStreamState) emitError(writer core.FlushWriter, event acp.AgentEvent) error {
	if err := s.closeOpenBlocks(writer); err != nil {
		return err
	}
	if err := core.WriteSSE(writer, core.SSEMessage{
		Name: "error",
		Data: map[string]any{
			"type":      "error",
			"errorText": s.errorText(event),
		},
	}); err != nil {
		return err
	}
	return s.finish(writer, event)
}

func (s *promptStreamState) emitGenericEvent(writer core.FlushWriter, event acp.AgentEvent) error {
	return core.WriteSSE(writer, core.SSEMessage{
		Name: event.Type,
		Data: map[string]any{
			"type": "data-agh-event",
			"data": agentEventPayloadFromEvent(event),
		},
	})
}

func (s *promptStreamState) toolCallID(event acp.AgentEvent) string {
	toolCallID := strings.TrimSpace(event.ToolCallID)
	if toolCallID == "" {
		return s.messageID + "-tool"
	}
	return toolCallID
}

func (s *promptStreamState) ensureToolCallStarted(
	writer core.FlushWriter,
	toolCallID string,
	event acp.AgentEvent,
) error {
	if _, ok := s.toolStarted[toolCallID]; ok {
		return nil
	}

	s.toolStarted[toolCallID] = struct{}{}
	toolName := strings.TrimSpace(event.Title)
	if toolName == "" {
		toolName = "tool"
	}

	return core.WriteSSE(writer, core.SSEMessage{
		Name: "tool_call",
		Data: map[string]any{
			"type":       "tool-input-start",
			"toolCallId": toolCallID,
			"toolName":   toolName,
		},
	})
}

func (s *promptStreamState) errorText(event acp.AgentEvent) string {
	errorText := strings.TrimSpace(event.Error)
	if errorText == "" {
		errorText = strings.TrimSpace(event.Text)
	}
	return errorText
}

func (s *promptStreamState) ensureMessageStarted(writer core.FlushWriter, event acp.AgentEvent) error {
	if s.messageStarted {
		return nil
	}

	messageID := strings.TrimSpace(event.TurnID)
	if messageID == "" {
		messageID = s.messageID
	}
	if messageID == "" {
		messageID = "msg-" + strings.ReplaceAll(s.now(), ":", "-")
	}
	s.messageID = messageID
	s.textBlockID = messageID + "-text"
	s.reasoningBlockID = messageID + "-reasoning"
	s.messageStarted = true

	return core.WriteSSE(writer, core.SSEMessage{
		Data: map[string]any{
			"type":      "start",
			"messageId": s.messageID,
		},
	})
}

func (s *promptStreamState) ensureTextStarted(writer core.FlushWriter) error {
	if s.textStarted {
		return nil
	}
	s.textStarted = true
	return core.WriteSSE(writer, core.SSEMessage{
		Data: map[string]any{
			"type": "text-start",
			"id":   s.textBlockID,
		},
	})
}

func (s *promptStreamState) ensureReasoningStarted(writer core.FlushWriter) error {
	if s.reasoningStarted {
		return nil
	}
	s.reasoningStarted = true
	return core.WriteSSE(writer, core.SSEMessage{
		Data: map[string]any{
			"type": "reasoning-start",
			"id":   s.reasoningBlockID,
		},
	})
}

func (s *promptStreamState) closeOpenBlocks(writer core.FlushWriter) error {
	if s.textStarted {
		if err := core.WriteSSE(writer, core.SSEMessage{
			Data: map[string]any{
				"type": "text-end",
				"id":   s.textBlockID,
			},
		}); err != nil {
			return err
		}
		s.textStarted = false
	}
	if s.reasoningStarted {
		if err := core.WriteSSE(writer, core.SSEMessage{
			Data: map[string]any{
				"type": "reasoning-end",
				"id":   s.reasoningBlockID,
			},
		}); err != nil {
			return err
		}
		s.reasoningStarted = false
	}
	return nil
}

func (s *promptStreamState) finish(writer core.FlushWriter, event acp.AgentEvent) error {
	if s.finished {
		return nil
	}
	if err := s.ensureMessageStarted(writer, event); err != nil {
		return err
	}
	if err := s.closeOpenBlocks(writer); err != nil {
		return err
	}
	s.finished = true

	finishPayload := map[string]any{
		"type": "finish",
	}
	if finishReason := aiSDKFinishReason(event.StopReason); finishReason != "" {
		finishPayload["finishReason"] = finishReason
	}

	if err := core.WriteSSE(writer, core.SSEMessage{
		Name: "done",
		Data: finishPayload,
	}); err != nil {
		return err
	}
	return core.WriteSSERaw(writer, "", "[DONE]")
}

func aiSDKFinishReason(stopReason string) string {
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

func agentEventPayloadFromEvent(event acp.AgentEvent) agentEventPayload {
	base := core.AgentEventPayloadFromEvent(event)
	payload := agentEventPayload{
		Type:       base.Type,
		SessionID:  base.SessionID,
		TurnID:     base.TurnID,
		RequestID:  base.RequestID,
		Text:       base.Text,
		Title:      base.Title,
		ToolCallID: base.ToolCallID,
		StopReason: base.StopReason,
		Action:     base.Action,
		Resource:   base.Resource,
		Decision:   base.Decision,
		Error:      base.Error,
		Usage:      tokenUsagePayloadFromUsage(event.Usage),
		Raw:        base.Raw,
	}
	if !event.Timestamp.IsZero() {
		payload.Timestamp = event.Timestamp.UTC().Format(time.RFC3339Nano)
	}
	return payload
}

func tokenUsagePayloadFromUsage(usage *acp.TokenUsage) *tokenUsagePayload {
	base := core.TokenUsagePayloadFromUsage(usage)
	if base == nil {
		return nil
	}

	payload := &tokenUsagePayload{
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
