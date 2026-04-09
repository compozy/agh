package hooks

import (
	"context"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/session"
)

var _ session.Notifier = (*Hooks)(nil)

// OnSessionCreated dispatches the session.post_create hook family.
func (h *Hooks) OnSessionCreated(ctx context.Context, sess *session.Session) {
	if h == nil || sess == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := h.DispatchSessionPostCreate(ctx, SessionPostCreatePayload{
		PayloadBase:    h.payloadBase(HookSessionPostCreate),
		SessionContext: sessionContextFromSession(sess),
	}); err != nil {
		h.logger.WarnContext(ctx, "hook.dispatch.notifier_failed", "event", HookSessionPostCreate.String(), "error", err)
	}
}

// OnSessionStopped dispatches the session.post_stop hook family.
func (h *Hooks) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if h == nil || sess == nil {
		return
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if _, err := h.DispatchSessionPostStop(ctx, SessionPostStopPayload{
		PayloadBase:    h.payloadBase(HookSessionPostStop),
		SessionContext: sessionContextFromSession(sess),
	}); err != nil {
		h.logger.WarnContext(ctx, "hook.dispatch.notifier_failed", "event", HookSessionPostStop.String(), "error", err)
	}
}

// OnAgentEvent is intentionally conservative in task_06; richer runtime event
// mapping lands with the direct session integrations in task_10.
func (h *Hooks) OnAgentEvent(_ context.Context, _ string, _ acp.AgentEvent) {}

func (h *Hooks) payloadBase(event HookEvent) PayloadBase {
	now := time.Now
	if h != nil && h.now != nil {
		now = h.now
	}

	return PayloadBase{
		Event:     event,
		Timestamp: now(),
	}
}

func sessionContextFromSession(sess *session.Session) SessionContext {
	if sess == nil {
		return SessionContext{}
	}

	info := sess.Info()
	if info == nil {
		return SessionContext{}
	}

	return SessionContext{
		SessionID:    info.ID,
		SessionName:  info.Name,
		SessionType:  string(info.Type),
		AgentName:    info.AgentName,
		WorkspaceID:  info.WorkspaceID,
		Workspace:    info.Workspace,
		ACPSessionID: info.ACPSessionID,
		State:        string(info.State),
	}
}
