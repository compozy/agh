package extension

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/testutil"
)

type recordingDeliveryTransport struct {
	mu     sync.Mutex
	calls  []channelspkg.DeliveryRequest
	notify chan struct{}
}

func (t *recordingDeliveryTransport) DeliverChannel(
	_ context.Context,
	_ string,
	req channelspkg.DeliveryRequest,
) (channelspkg.DeliveryAck, error) {
	t.mu.Lock()
	if t.notify == nil {
		t.notify = make(chan struct{}, 1)
	}
	t.calls = append(t.calls, cloneExtensionDeliveryRequest(req))
	notify := t.notify
	t.mu.Unlock()
	select {
	case notify <- struct{}{}:
	default:
	}

	return channelspkg.DeliveryAck{
		DeliveryID: req.Event.DeliveryID,
		Seq:        req.Event.Seq,
	}, nil
}

func (t *recordingDeliveryTransport) snapshotCalls() []channelspkg.DeliveryRequest {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]channelspkg.DeliveryRequest, 0, len(t.calls))
	for _, call := range t.calls {
		out = append(out, cloneExtensionDeliveryRequest(call))
	}
	return out
}

func (t *recordingDeliveryTransport) notifyChan() <-chan struct{} {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.notify == nil {
		t.notify = make(chan struct{}, 1)
	}
	return t.notify
}

type recordingNotifier struct {
	mu      sync.Mutex
	created []*session.Session
	stopped []*session.Session
	events  []any
}

func (n *recordingNotifier) OnSessionCreated(_ context.Context, sess *session.Session) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.created = append(n.created, sess)
}

func (n *recordingNotifier) OnSessionStopped(_ context.Context, sess *session.Session) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.stopped = append(n.stopped, sess)
}

func (n *recordingNotifier) OnAgentEvent(_ context.Context, _ string, event any) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, event)
}

func (n *recordingNotifier) snapshot() (created int, stopped int, events []any) {
	n.mu.Lock()
	defer n.mu.Unlock()

	return len(n.created), len(n.stopped), append([]any(nil), n.events...)
}

func TestChannelDeliveryNotifierProjectsEventsAndForwardsLifecycle(t *testing.T) {
	t.Parallel()

	transport := &recordingDeliveryTransport{}
	broker := channelspkg.NewBroker(transport)
	t.Cleanup(broker.Close)

	registration, err := broker.RegisterPromptDelivery(testutil.Context(t), channelspkg.PromptDeliveryRegistration{
		SessionID:     "sess-notify",
		TurnID:        "turn-notify",
		ExtensionName: "ext-telegram",
		RoutingKey: channelspkg.RoutingKey{
			Scope:             channelspkg.ScopeWorkspace,
			WorkspaceID:       "ws-1",
			ChannelInstanceID: "chan-notify",
			PeerID:            "peer-notify",
		},
		DeliveryTarget: channelspkg.DeliveryTarget{
			ChannelInstanceID: "chan-notify",
			PeerID:            "peer-notify",
			Mode:              channelspkg.DeliveryModeReply,
		},
	})
	if err != nil {
		t.Fatalf("RegisterPromptDelivery() error = %v", err)
	}

	downstream := &recordingNotifier{}
	notifier := NewChannelDeliveryNotifier(broker, downstream)
	if notifier == nil {
		t.Fatal("NewChannelDeliveryNotifier() = nil, want notifier")
	}

	sess := &session.Session{ID: registration.SessionID}
	notifier.OnSessionCreated(testutil.Context(t), sess)

	messageEvent := acp.AgentEvent{
		Type:      "agent_message",
		TurnID:    registration.TurnID,
		Timestamp: time.Date(2026, time.April, 11, 12, 3, 0, 0, time.UTC),
		Text:      "hello",
	}
	projected := projectionEventFromAgentEvent(messageEvent)
	if projected.Fingerprint == "" {
		t.Fatal("projectionEventFromAgentEvent() fingerprint = empty, want stable fingerprint")
	}

	notifier.OnAgentEvent(testutil.Context(t), registration.SessionID, messageEvent)
	waitForExtensionDeliveryCalls(t, transport, 1)

	calls := transport.snapshotCalls()
	if got, want := calls[0].Event.EventType, channelspkg.DeliveryEventTypeStart; got != want {
		t.Fatalf("projected event type = %q, want %q", got, want)
	}
	if got, want := calls[0].Event.Content.Text, "hello"; got != want {
		t.Fatalf("projected content = %q, want %q", got, want)
	}

	notifier.OnAgentEvent(testutil.Context(t), registration.SessionID, "ignored")
	createdCount, stoppedCount, forwarded := downstream.snapshot()
	if createdCount != 1 || stoppedCount != 0 || len(forwarded) != 2 {
		t.Fatalf("downstream lifecycle/events = created:%d stopped:%d events:%d, want 1/0/2", createdCount, stoppedCount, len(forwarded))
	}

	notifier.OnSessionStopped(testutil.Context(t), sess)
	waitForExtensionDeliveryCalls(t, transport, 2)

	calls = transport.snapshotCalls()
	last := calls[len(calls)-1].Event
	if got, want := last.EventType, channelspkg.DeliveryEventTypeError; got != want {
		t.Fatalf("session stop event type = %q, want %q", got, want)
	}
	if !last.Final {
		t.Fatal("session stop event Final = false, want true")
	}

	createdCount, stoppedCount, forwarded = downstream.snapshot()
	if createdCount != 1 || stoppedCount != 1 || len(forwarded) != 2 {
		t.Fatalf("downstream lifecycle/events after stop = created:%d stopped:%d events:%d, want 1/1/2", createdCount, stoppedCount, len(forwarded))
	}
}

func TestChannelDeliveryNotifierNilPathsAreNoOps(t *testing.T) {
	t.Parallel()

	var notifier *ChannelDeliveryNotifier
	notifier.OnSessionCreated(testutil.Context(t), nil)
	notifier.OnSessionStopped(testutil.Context(t), nil)
	notifier.OnAgentEvent(testutil.Context(t), "sess-nil", nil)

	standalone := NewChannelDeliveryNotifier(nil, nil)
	standalone.OnSessionCreated(testutil.Context(t), &session.Session{ID: "sess-nil"})
	standalone.OnSessionStopped(testutil.Context(t), &session.Session{ID: "sess-nil"})
	standalone.OnAgentEvent(testutil.Context(t), "sess-nil", "ignored")
}

func TestManagerDeliverChannel(t *testing.T) {
	t.Parallel()

	req := channelspkg.DeliveryRequest{
		Event: channelspkg.DeliveryEvent{
			DeliveryID:        "del-manager",
			ChannelInstanceID: "chan-manager",
			RoutingKey: channelspkg.RoutingKey{
				Scope:             channelspkg.ScopeWorkspace,
				WorkspaceID:       "ws-1",
				ChannelInstanceID: "chan-manager",
				PeerID:            "peer-manager",
			},
			DeliveryTarget: channelspkg.DeliveryTarget{
				ChannelInstanceID: "chan-manager",
				PeerID:            "peer-manager",
				Mode:              channelspkg.DeliveryModeReply,
			},
			Seq:       1,
			EventType: channelspkg.DeliveryEventTypeStart,
			Content:   channelspkg.MessageContent{Text: "hello"},
		},
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		process := newFakeProcess(9001)
		process.callFn = func(_ context.Context, method string, params, result any) error {
			if got, want := method, string(extensionprotocol.ExtensionServiceMethodChannelsDeliver); got != want {
				t.Fatalf("Call() method = %q, want %q", got, want)
			}
			typedReq, ok := params.(channelspkg.DeliveryRequest)
			if !ok {
				t.Fatalf("Call() params type = %T, want channel delivery request", params)
			}
			if got, want := typedReq.Event.DeliveryID, req.Event.DeliveryID; got != want {
				t.Fatalf("Call() delivery id = %q, want %q", got, want)
			}
			ack, ok := result.(*channelspkg.DeliveryAck)
			if !ok {
				t.Fatalf("Call() result type = %T, want *DeliveryAck", result)
			}
			ack.DeliveryID = typedReq.Event.DeliveryID
			ack.Seq = typedReq.Event.Seq
			ack.RemoteMessageID = "remote-manager"
			return nil
		}

		manager := &Manager{
			extensions: map[string]*managedExtension{
				"ext-telegram": {
					active:  true,
					process: process,
					initialize: &subprocess.InitializeResponse{
						ImplementedMethods: []string{string(extensionprotocol.ExtensionServiceMethodChannelsDeliver)},
					},
				},
			},
		}

		ack, err := manager.DeliverChannel(testutil.Context(t), "ext-telegram", req)
		if err != nil {
			t.Fatalf("DeliverChannel() error = %v", err)
		}
		if got, want := ack.RemoteMessageID, "remote-manager"; got != want {
			t.Fatalf("ack.RemoteMessageID = %q, want %q", got, want)
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		manager := &Manager{}
		_, err := manager.DeliverChannel(ctx, "ext-telegram", req)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("DeliverChannel() error = %v, want context.Canceled", err)
		}
	})

	t.Run("nil manager", func(t *testing.T) {
		t.Parallel()

		var manager *Manager
		_, err := manager.DeliverChannel(testutil.Context(t), "ext-telegram", req)
		if err == nil || !strings.Contains(err.Error(), "manager is required") {
			t.Fatalf("DeliverChannel() error = %v, want nil manager error", err)
		}
	})

	t.Run("inactive extension", func(t *testing.T) {
		t.Parallel()

		manager := &Manager{
			extensions: map[string]*managedExtension{
				"ext-telegram": {},
			},
		}

		_, err := manager.DeliverChannel(testutil.Context(t), "ext-telegram", req)
		if !errors.Is(err, channelspkg.ErrDeliveryTransportUnavailable) {
			t.Fatalf("DeliverChannel() error = %v, want ErrDeliveryTransportUnavailable", err)
		}
	})

	t.Run("missing negotiated method", func(t *testing.T) {
		t.Parallel()

		manager := &Manager{
			extensions: map[string]*managedExtension{
				"ext-telegram": {
					active:     true,
					process:    newFakeProcess(9002),
					initialize: &subprocess.InitializeResponse{},
				},
			},
		}

		_, err := manager.DeliverChannel(testutil.Context(t), "ext-telegram", req)
		if !errors.Is(err, channelspkg.ErrDeliveryTransportUnavailable) {
			t.Fatalf("DeliverChannel() error = %v, want wrapped ErrDeliveryTransportUnavailable", err)
		}
	})

	t.Run("process call failure", func(t *testing.T) {
		t.Parallel()

		process := newFakeProcess(9003)
		process.callFn = func(context.Context, string, any, any) error {
			return errors.New("rpc failed")
		}

		manager := &Manager{
			extensions: map[string]*managedExtension{
				"ext-telegram": {
					active:  true,
					process: process,
					initialize: &subprocess.InitializeResponse{
						ImplementedMethods: []string{string(extensionprotocol.ExtensionServiceMethodChannelsDeliver)},
					},
				},
			},
		}

		_, err := manager.DeliverChannel(testutil.Context(t), "ext-telegram", req)
		if err == nil || !strings.Contains(err.Error(), "rpc failed") {
			t.Fatalf("DeliverChannel() error = %v, want wrapped process failure", err)
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		t.Parallel()

		manager := &Manager{}
		_, err := manager.DeliverChannel(testutil.Context(t), "ext-telegram", channelspkg.DeliveryRequest{})
		if err == nil {
			t.Fatal("DeliverChannel() error = nil, want validation error")
		}
	})

	t.Run("missing extension name", func(t *testing.T) {
		t.Parallel()

		manager := &Manager{}
		_, err := manager.DeliverChannel(testutil.Context(t), "  ", req)
		if err == nil || !strings.Contains(err.Error(), "extension name is required") {
			t.Fatalf("DeliverChannel() error = %v, want missing extension name error", err)
		}
	})
}

func cloneExtensionDeliveryRequest(req channelspkg.DeliveryRequest) channelspkg.DeliveryRequest {
	cloned := req
	cloned.Event.Metadata = append([]byte(nil), req.Event.Metadata...)
	if req.Snapshot != nil {
		snapshot := *req.Snapshot
		cloned.Snapshot = &snapshot
	}
	return cloned
}

func waitForExtensionDeliveryCalls(t *testing.T, transport *recordingDeliveryTransport, want int) {
	t.Helper()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()

	notify := transport.notifyChan()
	for {
		if len(transport.snapshotCalls()) >= want {
			return
		}
		select {
		case <-notify:
		case <-timer.C:
			t.Fatalf("delivery call count did not reach %d before timeout; got %d", want, len(transport.snapshotCalls()))
		}
	}
}
