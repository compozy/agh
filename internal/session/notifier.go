package session

import "context"

func (m *Manager) notifyAgentEvent(ctx context.Context, session *Session, event any) {
	if m == nil || m.notifier == nil || session == nil {
		return
	}

	if aware, ok := m.notifier.(AgentEventNotifier); ok {
		aware.OnAgentEventForSession(ctx, session, event)
		return
	}

	m.notifier.OnAgentEvent(ctx, session.ID, event)
}
