package extension

import (
	"context"
	"log/slog"

	"github.com/pedronauck/agh/internal/acp"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/transcript"
)

// ChannelDeliveryNotifier projects prompt-time ACP events into the channel
// delivery broker while preserving an optional downstream notifier chain.
type ChannelDeliveryNotifier struct {
	broker     *channelspkg.Broker
	downstream session.Notifier
}

var _ session.Notifier = (*ChannelDeliveryNotifier)(nil)

// NewChannelDeliveryNotifier wraps the provided downstream notifier with
// session-to-channel delivery projection.
func NewChannelDeliveryNotifier(broker *channelspkg.Broker, downstream session.Notifier) *ChannelDeliveryNotifier {
	return &ChannelDeliveryNotifier{
		broker:     broker,
		downstream: downstream,
	}
}

// OnSessionCreated forwards the lifecycle callback unchanged.
func (n *ChannelDeliveryNotifier) OnSessionCreated(ctx context.Context, sess *session.Session) {
	if n == nil || n.downstream == nil {
		return
	}
	n.downstream.OnSessionCreated(ctx, sess)
}

// OnSessionStopped fails unfinished channel deliveries before forwarding the lifecycle callback.
func (n *ChannelDeliveryNotifier) OnSessionStopped(ctx context.Context, sess *session.Session) {
	if n == nil {
		return
	}
	if n.broker != nil && sess != nil {
		if err := n.broker.FailSession(ctx, sess.ID, ""); err != nil {
			slog.ErrorContext(ctx, "extension: fail session delivery projection",
				"session_id", sess.ID,
				"error", err,
			)
		}
	}
	if n.downstream != nil {
		n.downstream.OnSessionStopped(ctx, sess)
	}
}

// OnAgentEvent projects ACP runtime output into the delivery broker before forwarding.
func (n *ChannelDeliveryNotifier) OnAgentEvent(ctx context.Context, sessionID string, payload any) {
	if n == nil {
		return
	}
	if n.broker != nil {
		if event, ok := payload.(acp.AgentEvent); ok {
			if err := n.broker.ProjectEvent(ctx, sessionID, projectionEventFromAgentEvent(event)); err != nil {
				slog.ErrorContext(ctx, "extension: project channel delivery event",
					"session_id", sessionID,
					"event_type", event.Type,
					"turn_id", event.TurnID,
					"error", err,
				)
			}
		}
	}
	if n.downstream != nil {
		n.downstream.OnAgentEvent(ctx, sessionID, payload)
	}
}

func projectionEventFromAgentEvent(event acp.AgentEvent) channelspkg.DeliveryProjectionEvent {
	projected := channelspkg.DeliveryProjectionEvent{
		Type:      event.Type,
		TurnID:    event.TurnID,
		Timestamp: event.Timestamp,
		Text:      event.Text,
		Error:     event.Error,
	}
	if fingerprint, err := transcript.MarshalAgentEvent(event); err == nil {
		projected.Fingerprint = fingerprint
	}
	return projected
}
