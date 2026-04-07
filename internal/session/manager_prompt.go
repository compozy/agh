package session

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/transcript"
)

// Prompt sends one prompt turn to an active session and mirrors the runtime stream into storage and observers.
func (m *Manager) Prompt(ctx context.Context, id string, msg string) (<-chan acp.AgentEvent, error) {
	if ctx == nil {
		return nil, errors.New("session: prompt context is required")
	}

	message := strings.TrimSpace(msg)
	if message == "" {
		return nil, errors.New("session: prompt message is required")
	}

	session, err := m.lookup(id)
	if err != nil {
		return nil, err
	}

	turnID := strings.TrimSpace(m.newTurnID())
	if turnID == "" {
		turnID = newID("turn")
	}

	proc, err := session.beginPromptSetup()
	if err != nil {
		return nil, err
	}
	defer session.finishPromptSetup()

	userEvent := m.normalizeEvent(session, turnID, acp.AgentEvent{
		Type:      acp.EventTypeUserMessage,
		TurnID:    turnID,
		Timestamp: m.now(),
		Text:      message,
	})
	if err := m.recordEvent(ctx, session, userEvent); err != nil {
		return nil, fmt.Errorf("session: persist prompt message for %q: %w", id, err)
	}
	if m.notifier != nil {
		m.notifier.OnAgentEvent(ctx, session.ID, userEvent)
	}

	source, err := m.driver.Prompt(ctx, proc, acp.PromptRequest{TurnID: turnID, Message: message})
	if err != nil {
		return nil, fmt.Errorf("session: prompt session %q: %w", id, err)
	}

	out := make(chan acp.AgentEvent, m.promptBufSize)
	// pumpPrompt terminates when the driver closes the source channel or the request context ends.
	go m.pumpPrompt(ctx, session, turnID, source, out)
	return out, nil
}

// ApprovePermission resolves one pending interactive permission request for an active session.
func (m *Manager) ApprovePermission(ctx context.Context, id string, req acp.ApproveRequest) error {
	if ctx == nil {
		return errors.New("session: approval context is required")
	}
	if err := req.Validate(); err != nil {
		return err
	}

	target := strings.TrimSpace(id)
	if target == "" {
		return errors.New("session: session id is required")
	}

	session, ok := m.Get(target)
	if !ok {
		meta, err := m.readMeta(target)
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: %s (%s)", ErrSessionNotActive, target, meta.State)
	}

	if err := session.ApprovePermission(ctx, req); err != nil {
		switch {
		case errors.Is(err, ErrSessionNotActive):
			return err
		case errors.Is(err, acp.ErrPendingPermissionNotFound):
			return fmt.Errorf("%w: %s", ErrPendingPermissionNotFound, target)
		case errors.Is(err, acp.ErrPendingPermissionConflict):
			return fmt.Errorf("%w: %s", ErrPendingPermissionConflict, target)
		default:
			return err
		}
	}
	return nil
}

func (m *Manager) pumpPrompt(ctx context.Context, session *Session, turnID string, source <-chan acp.AgentEvent, out chan<- acp.AgentEvent) {
	defer close(out)

	for event := range source {
		normalized := m.normalizeEvent(session, turnID, event)
		if err := m.recordEvent(ctx, session, normalized); err != nil {
			m.sessionLogger(session).Warn("session: record prompt event failed", "turn_id", turnID, "error", err)
		}
		if m.notifier != nil {
			m.notifier.OnAgentEvent(ctx, session.ID, normalized)
		}

		select {
		case out <- normalized:
		case <-ctx.Done():
		}
	}
}

func (m *Manager) normalizeEvent(session *Session, turnID string, event acp.AgentEvent) acp.AgentEvent {
	normalized := event
	if strings.TrimSpace(normalized.TurnID) == "" {
		normalized.TurnID = turnID
	}
	if normalized.Timestamp.IsZero() {
		normalized.Timestamp = m.now()
	}
	if session != nil {
		info := session.Info()
		if strings.TrimSpace(normalized.SessionID) == "" {
			normalized.SessionID = info.ACPSessionID
		}
	}
	return normalized
}

func (m *Manager) recordEvent(ctx context.Context, session *Session, event acp.AgentEvent) error {
	recorder := session.recorderHandle()
	if recorder == nil {
		return errors.New("session: event recorder is not available")
	}

	payload, err := marshalAgentEvent(event)
	if err != nil {
		return err
	}

	if err := recorder.Record(ctx, store.SessionEvent{
		TurnID:    event.TurnID,
		Type:      event.Type,
		AgentName: session.Info().AgentName,
		Content:   payload,
		Timestamp: event.Timestamp,
	}); err != nil {
		return err
	}

	if event.Usage != nil {
		if err := recorder.RecordTokenUsage(ctx, store.TokenUsage{
			TurnID:           event.Usage.TurnID,
			InputTokens:      event.Usage.InputTokens,
			OutputTokens:     event.Usage.OutputTokens,
			TotalTokens:      event.Usage.TotalTokens,
			ThoughtTokens:    event.Usage.ThoughtTokens,
			CacheReadTokens:  event.Usage.CacheReadTokens,
			CacheWriteTokens: event.Usage.CacheWriteTokens,
			ContextUsed:      event.Usage.ContextUsed,
			ContextSize:      event.Usage.ContextSize,
			CostAmount:       event.Usage.CostAmount,
			CostCurrency:     event.Usage.CostCurrency,
			Timestamp:        event.Usage.Timestamp,
		}); err != nil {
			return err
		}
	}

	return nil
}

func marshalAgentEvent(event acp.AgentEvent) (string, error) {
	data, err := transcript.MarshalAgentEvent(event)
	if err != nil {
		return "", fmt.Errorf("session: marshal agent event: %w", err)
	}
	return data, nil
}
