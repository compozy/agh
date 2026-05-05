package network

import (
	"context"
	"strings"
	"time"

	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/store"
)

// HookDispatcher observes committed network conversation state changes.
type HookDispatcher interface {
	DispatchNetworkThreadOpened(
		context.Context,
		hookspkg.NetworkThreadOpenedPayload,
	) (hookspkg.NetworkThreadOpenedPayload, error)
	DispatchNetworkDirectRoomOpened(
		context.Context,
		hookspkg.NetworkDirectRoomOpenedPayload,
	) (hookspkg.NetworkDirectRoomOpenedPayload, error)
	DispatchNetworkMessagePersisted(
		context.Context,
		hookspkg.NetworkMessagePersistedPayload,
	) (hookspkg.NetworkMessagePersistedPayload, error)
	DispatchNetworkWorkOpened(
		context.Context,
		hookspkg.NetworkWorkOpenedPayload,
	) (hookspkg.NetworkWorkOpenedPayload, error)
	DispatchNetworkWorkTransitioned(
		context.Context,
		hookspkg.NetworkWorkTransitionedPayload,
	) (hookspkg.NetworkWorkTransitionedPayload, error)
	DispatchNetworkWorkClosed(
		context.Context,
		hookspkg.NetworkWorkClosedPayload,
	) (hookspkg.NetworkWorkClosedPayload, error)
}

// WithManagerHookDispatcher injects the network hook dispatcher.
func WithManagerHookDispatcher(dispatcher HookDispatcher) ManagerOption {
	return func(opts *managerOptions) {
		opts.hooks = dispatcher
	}
}

func (m *Manager) observeConversationWrite(
	ctx context.Context,
	entry store.NetworkConversationMessage,
	result store.NetworkConversationWriteResult,
) {
	if m == nil || result.Duplicate {
		return
	}
	if m.stats != nil {
		m.stats.recordConversationWrite(entry, result)
	}
	m.dispatchNetworkHooks(ctx, entry, result)
}

func (m *Manager) dispatchNetworkHooks(
	ctx context.Context,
	entry store.NetworkConversationMessage,
	result store.NetworkConversationWriteResult,
) {
	if m == nil || m.hooks == nil {
		return
	}
	basePayload := networkPayloadForWrite(entry, result, networkPayloadTimestamp(entry, result, m.now))
	for _, event := range networkHookEventsForWrite(entry, result) {
		payload := basePayload
		payload.Event = event
		if err := m.dispatchNetworkHook(ctx, payload); err != nil {
			m.logNetworkHookFailure(event, payload, err)
		}
	}
}

func (m *Manager) dispatchNetworkHook(ctx context.Context, payload hookspkg.NetworkPayload) error {
	switch payload.Event {
	case hookspkg.HookNetworkThreadOpened:
		_, err := m.hooks.DispatchNetworkThreadOpened(ctx, payload)
		return err
	case hookspkg.HookNetworkDirectRoomOpened:
		_, err := m.hooks.DispatchNetworkDirectRoomOpened(ctx, payload)
		return err
	case hookspkg.HookNetworkMessagePersisted:
		_, err := m.hooks.DispatchNetworkMessagePersisted(ctx, payload)
		return err
	case hookspkg.HookNetworkWorkOpened:
		_, err := m.hooks.DispatchNetworkWorkOpened(ctx, payload)
		return err
	case hookspkg.HookNetworkWorkTransitioned:
		_, err := m.hooks.DispatchNetworkWorkTransitioned(ctx, payload)
		return err
	case hookspkg.HookNetworkWorkClosed:
		_, err := m.hooks.DispatchNetworkWorkClosed(ctx, payload)
		return err
	default:
		return nil
	}
}

func (m *Manager) logNetworkHookFailure(
	event hookspkg.HookEvent,
	payload hookspkg.NetworkPayload,
	err error,
) {
	if m == nil || m.logger == nil || err == nil {
		return
	}
	m.logger.Warn(
		"network.hook.dispatch_failed",
		"event", event.String(),
		"message_id", strings.TrimSpace(payload.MessageID),
		"work_id", strings.TrimSpace(payload.WorkID),
		"trace_id", strings.TrimSpace(payload.TraceID),
		"causation_id", strings.TrimSpace(payload.CausationID),
		"channel", strings.TrimSpace(payload.Channel),
		"surface", strings.TrimSpace(payload.Surface),
		"thread_id", strings.TrimSpace(payload.ThreadID),
		"direct_id", strings.TrimSpace(payload.DirectID),
		"error", err,
	)
}

func networkHookEventsForWrite(
	entry store.NetworkConversationMessage,
	result store.NetworkConversationWriteResult,
) []hookspkg.HookEvent {
	events := make([]hookspkg.HookEvent, 0, 5)
	if result.ConversationOpened {
		switch strings.TrimSpace(entry.Surface) {
		case store.NetworkSurfaceThread:
			events = append(events, hookspkg.HookNetworkThreadOpened)
		case store.NetworkSurfaceDirect:
			events = append(events, hookspkg.HookNetworkDirectRoomOpened)
		}
	}
	events = append(events, hookspkg.HookNetworkMessagePersisted)
	if result.WorkOpened {
		events = append(events, hookspkg.HookNetworkWorkOpened)
	}
	if result.WorkTransitioned {
		events = append(events, hookspkg.HookNetworkWorkTransitioned)
	}
	if (result.WorkOpened || result.WorkTransitioned) && networkWorkStateIsTerminal(result.WorkState) {
		events = append(events, hookspkg.HookNetworkWorkClosed)
	}
	return events
}

func networkPayloadForWrite(
	entry store.NetworkConversationMessage,
	result store.NetworkConversationWriteResult,
	timestamp time.Time,
) hookspkg.NetworkPayload {
	messageID := strings.TrimSpace(result.MessageID)
	if messageID == "" {
		messageID = strings.TrimSpace(entry.MessageID)
	}
	return hookspkg.NetworkPayload{
		PayloadBase: hookspkg.PayloadBase{
			Event:     hookspkg.HookNetworkMessagePersisted,
			Timestamp: timestamp,
		},
		SessionID:   strings.TrimSpace(entry.SessionID),
		Channel:     strings.TrimSpace(entry.Channel),
		Surface:     strings.TrimSpace(entry.Surface),
		ThreadID:    strings.TrimSpace(entry.ThreadID),
		DirectID:    strings.TrimSpace(entry.DirectID),
		MessageID:   messageID,
		Kind:        strings.TrimSpace(entry.Kind),
		Direction:   strings.TrimSpace(entry.Direction),
		WorkID:      strings.TrimSpace(entry.WorkID),
		WorkState:   strings.TrimSpace(result.WorkState),
		PeerFrom:    strings.TrimSpace(entry.PeerFrom),
		PeerTo:      strings.TrimSpace(entry.PeerTo),
		TraceID:     strings.TrimSpace(entry.TraceID),
		CausationID: strings.TrimSpace(entry.CausationID),
	}
}

func networkPayloadTimestamp(
	entry store.NetworkConversationMessage,
	result store.NetworkConversationWriteResult,
	now func() time.Time,
) time.Time {
	if !result.LastActivityAt.IsZero() {
		return result.LastActivityAt.UTC()
	}
	if !entry.Timestamp.IsZero() {
		return entry.Timestamp.UTC()
	}
	if now != nil {
		return now().UTC()
	}
	return time.Now().UTC()
}
