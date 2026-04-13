package extension

import (
	"context"
	"log/slog"

	"github.com/pedronauck/agh/internal/acp"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/transcript"
)

// BridgeDeliveryNotifier projects prompt-time ACP events into the bridge
// delivery broker while preserving an optional downstream notifier chain.
type BridgeDeliveryNotifier struct {
	broker     *bridgepkg.Broker
	downstream session.Notifier
}

var _ session.Notifier = (*BridgeDeliveryNotifier)(nil)

// NewBridgeDeliveryNotifier wraps the provided downstream notifier with
// session-to-bridge delivery projection.
func NewBridgeDeliveryNotifier(broker *bridgepkg.Broker, downstream session.Notifier) *BridgeDeliveryNotifier {
	return &BridgeDeliveryNotifier{
		broker:     broker,
		downstream: downstream,
	}
}

// OnSessionCreated forwards the lifecycle callback unchanged.
func (n *BridgeDeliveryNotifier) OnSessionCreated(ctx context.Context, sess *session.Session) {
	if n == nil || n.downstream == nil {
		return
	}
	n.downstream.OnSessionCreated(ctx, sess)
}

// OnSessionStopped fails unfinished bridge deliveries before forwarding the lifecycle callback.
func (n *BridgeDeliveryNotifier) OnSessionStopped(ctx context.Context, sess *session.Session) {
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
func (n *BridgeDeliveryNotifier) OnAgentEvent(ctx context.Context, sessionID string, payload any) {
	if n == nil {
		return
	}
	if n.broker != nil {
		if event, ok := payload.(acp.AgentEvent); ok {
			if err := n.broker.ProjectEvent(ctx, sessionID, projectionEventFromAgentEvent(event)); err != nil {
				slog.ErrorContext(ctx, "extension: project bridge delivery event",
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

func projectionEventFromAgentEvent(event acp.AgentEvent) bridgepkg.DeliveryProjectionEvent {
	projected := bridgepkg.DeliveryProjectionEvent{
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
