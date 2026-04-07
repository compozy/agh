package daemon

import (
	"context"

	"github.com/pedronauck/agh/internal/acp"
	"github.com/pedronauck/agh/internal/session"
)

type notifierFanout struct {
	notifiers        []session.Notifier
	onSessionStopped func(context.Context, *session.Session)
}

var _ session.Notifier = (*notifierFanout)(nil)

func (f *notifierFanout) OnSessionCreated(ctx context.Context, sess *session.Session) {
	for _, notifier := range f.notifiers {
		if notifier == nil {
			continue
		}
		notifier.OnSessionCreated(ctx, sess)
	}
}

func (f *notifierFanout) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if f.onSessionStopped != nil {
		f.onSessionStopped(ctx, sess)
	}
	for _, notifier := range f.notifiers {
		if notifier == nil {
			continue
		}
		notifier.OnSessionStopped(ctx, sess)
	}
}

func (f *notifierFanout) OnAgentEvent(ctx context.Context, sessionID string, event acp.AgentEvent) {
	for _, notifier := range f.notifiers {
		if notifier == nil {
			continue
		}
		notifier.OnAgentEvent(ctx, sessionID, event)
	}
}
