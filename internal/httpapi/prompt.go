package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/session"
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
		respondError(c, http.StatusBadRequest, fmt.Errorf("httpapi: decode prompt request: %w", err))
		return
	}

	message, err := extractPromptMessage(req)
	if err != nil {
		respondError(c, http.StatusBadRequest, err)
		return
	}

	events, err := h.sessions.Prompt(c.Request.Context(), c.Param("id"), message)
	if err != nil {
		respondError(c, statusForSessionError(err), err)
		return
	}

	c.Header("x-vercel-ai-ui-message-stream", "v1")
	writer, err := prepareSSE(c)
	if err != nil {
		respondError(c, http.StatusInternalServerError, err)
		return
	}

	state := &promptStreamState{
		now: func() string {
			return h.now().UTC().Format(timeRFC3339Nano)
		},
		toolStarted: make(map[string]struct{}),
	}

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-h.streamDone:
			return
		case event, ok := <-events:
			if !ok {
				if err := state.finish(writer, session.AgentEvent{}); err != nil {
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
			if strings.TrimSpace(part.Type) != "" && part.Type != "text" {
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

func (s *promptStreamState) emit(writer flushWriter, event session.AgentEvent) error {
	if err := s.ensureMessageStarted(writer, event); err != nil {
		return err
	}

	switch event.Type {
	case acp.EventTypeAgentMessage:
		if err := s.ensureTextStarted(writer); err != nil {
			return err
		}
		return writeSSE(writer, sseMessage{
			Name: "agent_message",
			Data: map[string]any{
				"type":  "text-delta",
				"id":    s.textBlockID,
				"delta": event.Text,
			},
		})
	case acp.EventTypeThought:
		if err := s.ensureReasoningStarted(writer); err != nil {
			return err
		}
		return writeSSE(writer, sseMessage{
			Name: "thought",
			Data: map[string]any{
				"type":  "reasoning-delta",
				"id":    s.reasoningBlockID,
				"delta": event.Text,
			},
		})
	case acp.EventTypeToolCall:
		toolCallID := strings.TrimSpace(event.ToolCallID)
		if toolCallID == "" {
			toolCallID = s.messageID + "-tool"
		}
		if _, ok := s.toolStarted[toolCallID]; !ok {
			s.toolStarted[toolCallID] = struct{}{}
			if err := writeSSE(writer, sseMessage{
				Name: "tool_call",
				Data: map[string]any{
					"type":       "tool-input-start",
					"toolCallId": toolCallID,
					"toolName":   firstNonBlank(strings.TrimSpace(event.Title), "tool"),
				},
			}); err != nil {
				return err
			}
		}
		return writeSSE(writer, sseMessage{
			Name: "tool_call",
			Data: map[string]any{
				"type":       "data-agh-event",
				"data":       agentEventPayloadFromEvent(event),
				"toolCallId": toolCallID,
			},
		})
	case acp.EventTypeToolResult:
		toolCallID := strings.TrimSpace(event.ToolCallID)
		if toolCallID == "" {
			toolCallID = s.messageID + "-tool"
		}
		return writeSSE(writer, sseMessage{
			Name: "tool_result",
			Data: map[string]any{
				"type":       "tool-output-available",
				"toolCallId": toolCallID,
				"output":     agentEventPayloadFromEvent(event),
			},
		})
	case acp.EventTypePermission:
		return writeSSE(writer, sseMessage{
			Name: "permission",
			Data: map[string]any{
				"type": "data-agh-permission",
				"data": agentEventPayloadFromEvent(event),
			},
		})
	case acp.EventTypeError:
		if err := s.closeOpenBlocks(writer); err != nil {
			return err
		}
		if err := writeSSE(writer, sseMessage{
			Name: "error",
			Data: map[string]any{
				"type":      "error",
				"errorText": firstNonBlank(strings.TrimSpace(event.Error), strings.TrimSpace(event.Text)),
			},
		}); err != nil {
			return err
		}
		return s.finish(writer, event)
	case acp.EventTypeDone:
		return s.finish(writer, event)
	default:
		return writeSSE(writer, sseMessage{
			Name: event.Type,
			Data: map[string]any{
				"type": "data-agh-event",
				"data": agentEventPayloadFromEvent(event),
			},
		})
	}
}

func (s *promptStreamState) ensureMessageStarted(writer flushWriter, event session.AgentEvent) error {
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

	return writeSSE(writer, sseMessage{
		Data: map[string]any{
			"type":      "start",
			"messageId": s.messageID,
		},
	})
}

func (s *promptStreamState) ensureTextStarted(writer flushWriter) error {
	if s.textStarted {
		return nil
	}
	s.textStarted = true
	return writeSSE(writer, sseMessage{
		Data: map[string]any{
			"type": "text-start",
			"id":   s.textBlockID,
		},
	})
}

func (s *promptStreamState) ensureReasoningStarted(writer flushWriter) error {
	if s.reasoningStarted {
		return nil
	}
	s.reasoningStarted = true
	return writeSSE(writer, sseMessage{
		Data: map[string]any{
			"type": "reasoning-start",
			"id":   s.reasoningBlockID,
		},
	})
}

func (s *promptStreamState) closeOpenBlocks(writer flushWriter) error {
	if s.textStarted {
		if err := writeSSE(writer, sseMessage{
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
		if err := writeSSE(writer, sseMessage{
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

func (s *promptStreamState) finish(writer flushWriter, event session.AgentEvent) error {
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

	if err := writeSSE(writer, sseMessage{
		Name: "done",
		Data: map[string]any{
			"type":       "finish",
			"stopReason": strings.TrimSpace(event.StopReason),
		},
	}); err != nil {
		return err
	}
	return writeSSERaw(writer, "", "[DONE]")
}

func agentEventPayloadFromEvent(event session.AgentEvent) agentEventPayload {
	payload := agentEventPayload{
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
		Usage:      tokenUsagePayloadFromUsage(event.Usage),
		Raw:        payloadJSON(string(event.Raw)),
	}
	if !event.Timestamp.IsZero() {
		payload.Timestamp = event.Timestamp.UTC().Format(timeRFC3339Nano)
	}
	return payload
}

func tokenUsagePayloadFromUsage(usage *session.TokenUsage) *tokenUsagePayload {
	if usage == nil {
		return nil
	}

	payload := &tokenUsagePayload{
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
		payload.Timestamp = usage.Timestamp.UTC().Format(timeRFC3339Nano)
	}
	return payload
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
