package extension

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/testutil"
)

type recordingDeliveryTransport struct {
	mu     sync.Mutex
	calls  []bridgepkg.DeliveryRequest
	notify chan struct{}
}

func (t *recordingDeliveryTransport) DeliverBridge(
	_ context.Context,
	_ string,
	req bridgepkg.DeliveryRequest,
) (bridgepkg.DeliveryAck, error) {
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

	return bridgepkg.DeliveryAck{
		DeliveryID: req.Event.DeliveryID,
		Seq:        req.Event.Seq,
	}, nil
}

func (t *recordingDeliveryTransport) snapshotCalls() []bridgepkg.DeliveryRequest {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := make([]bridgepkg.DeliveryRequest, 0, len(t.calls))
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

func TestBridgeDeliveryNotifierProjectsEventsAndForwardsLifecycle(t *testing.T) {
	t.Parallel()

	transport := &recordingDeliveryTransport{}
	broker := bridgepkg.NewBroker(transport)
	t.Cleanup(broker.Close)

	registration, err := broker.RegisterPromptDelivery(testutil.Context(t), bridgepkg.PromptDeliveryRegistration{
		SessionID:     "sess-notify",
		TurnID:        "turn-notify",
		ExtensionName: "ext-telegram",
		RoutingKey: bridgepkg.RoutingKey{
			Scope:            bridgepkg.ScopeWorkspace,
			WorkspaceID:      "ws-1",
			BridgeInstanceID: "brg-notify",
			PeerID:           "peer-notify",
		},
		DeliveryTarget: bridgepkg.DeliveryTarget{
			BridgeInstanceID: "brg-notify",
			PeerID:           "peer-notify",
			Mode:             bridgepkg.DeliveryModeReply,
		},
	})
	if err != nil {
		t.Fatalf("RegisterPromptDelivery() error = %v", err)
	}

	downstream := &recordingNotifier{}
	notifier := NewBridgeDeliveryNotifier(broker, downstream)
	if notifier == nil {
		t.Fatal("NewBridgeDeliveryNotifier() = nil, want notifier")
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
	if got, want := calls[0].Event.EventType, bridgepkg.DeliveryEventTypeStart; got != want {
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
	if got, want := last.EventType, bridgepkg.DeliveryEventTypeError; got != want {
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

func TestBridgeDeliveryNotifierNilPathsAreNoOps(t *testing.T) {
	t.Parallel()

	var notifier *BridgeDeliveryNotifier
	notifier.OnSessionCreated(testutil.Context(t), nil)
	notifier.OnSessionStopped(testutil.Context(t), nil)
	notifier.OnAgentEvent(testutil.Context(t), "sess-nil", nil)

	standalone := NewBridgeDeliveryNotifier(nil, nil)
	standalone.OnSessionCreated(testutil.Context(t), &session.Session{ID: "sess-nil"})
	standalone.OnSessionStopped(testutil.Context(t), &session.Session{ID: "sess-nil"})
	standalone.OnAgentEvent(testutil.Context(t), "sess-nil", "ignored")
}

func TestManagerDeliverBridge(t *testing.T) {
	t.Parallel()

	req := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "del-manager",
			BridgeInstanceID: "brg-manager",
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-1",
				BridgeInstanceID: "brg-manager",
				PeerID:           "peer-manager",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: "brg-manager",
				PeerID:           "peer-manager",
				Mode:             bridgepkg.DeliveryModeReply,
			},
			Seq:       1,
			EventType: bridgepkg.DeliveryEventTypeStart,
			Content:   bridgepkg.MessageContent{Text: "hello"},
		},
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		process := newFakeProcess(9001)
		process.callFn = func(_ context.Context, method string, params, result any) error {
			if got, want := method, string(extensionprotocol.ExtensionServiceMethodBridgesDeliver); got != want {
				t.Fatalf("Call() method = %q, want %q", got, want)
			}
			typedReq, ok := params.(bridgepkg.DeliveryRequest)
			if !ok {
				t.Fatalf("Call() params type = %T, want bridge delivery request", params)
			}
			if got, want := typedReq.Event.DeliveryID, req.Event.DeliveryID; got != want {
				t.Fatalf("Call() delivery id = %q, want %q", got, want)
			}
			ack, ok := result.(*bridgepkg.DeliveryAck)
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
						ImplementedMethods: []string{string(extensionprotocol.ExtensionServiceMethodBridgesDeliver)},
					},
				},
			},
		}

		ack, err := manager.DeliverBridge(testutil.Context(t), "ext-telegram", req)
		if err != nil {
			t.Fatalf("DeliverBridge() error = %v", err)
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
		_, err := manager.DeliverBridge(ctx, "ext-telegram", req)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("DeliverBridge() error = %v, want context.Canceled", err)
		}
	})

	t.Run("nil manager", func(t *testing.T) {
		t.Parallel()

		var manager *Manager
		_, err := manager.DeliverBridge(testutil.Context(t), "ext-telegram", req)
		if err == nil || !strings.Contains(err.Error(), "manager is required") {
			t.Fatalf("DeliverBridge() error = %v, want nil manager error", err)
		}
	})

	t.Run("inactive extension", func(t *testing.T) {
		t.Parallel()

		manager := &Manager{
			extensions: map[string]*managedExtension{
				"ext-telegram": {},
			},
		}

		_, err := manager.DeliverBridge(testutil.Context(t), "ext-telegram", req)
		if !errors.Is(err, bridgepkg.ErrDeliveryTransportUnavailable) {
			t.Fatalf("DeliverBridge() error = %v, want ErrDeliveryTransportUnavailable", err)
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

		_, err := manager.DeliverBridge(testutil.Context(t), "ext-telegram", req)
		if !errors.Is(err, bridgepkg.ErrDeliveryTransportUnavailable) {
			t.Fatalf("DeliverBridge() error = %v, want wrapped ErrDeliveryTransportUnavailable", err)
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
						ImplementedMethods: []string{string(extensionprotocol.ExtensionServiceMethodBridgesDeliver)},
					},
				},
			},
		}

		_, err := manager.DeliverBridge(testutil.Context(t), "ext-telegram", req)
		if err == nil || !strings.Contains(err.Error(), "rpc failed") {
			t.Fatalf("DeliverBridge() error = %v, want wrapped process failure", err)
		}
	})

	t.Run("invalid request", func(t *testing.T) {
		t.Parallel()

		manager := &Manager{}
		_, err := manager.DeliverBridge(testutil.Context(t), "ext-telegram", bridgepkg.DeliveryRequest{})
		if err == nil {
			t.Fatal("DeliverBridge() error = nil, want validation error")
		}
	})

	t.Run("missing extension name", func(t *testing.T) {
		t.Parallel()

		manager := &Manager{}
		_, err := manager.DeliverBridge(testutil.Context(t), "  ", req)
		if err == nil || !strings.Contains(err.Error(), "extension name is required") {
			t.Fatalf("DeliverBridge() error = %v, want missing extension name error", err)
		}
	})
}

func cloneExtensionDeliveryRequest(req bridgepkg.DeliveryRequest) bridgepkg.DeliveryRequest {
	cloned := req
	cloned.Event.ProviderMetadata = append([]byte(nil), req.Event.ProviderMetadata...)
	if req.Event.Reference != nil {
		reference := *req.Event.Reference
		cloned.Event.Reference = &reference
	}
	if req.Event.Error != nil {
		errorDetail := *req.Event.Error
		cloned.Event.Error = &errorDetail
	}
	if req.Event.Resume != nil {
		resume := *req.Event.Resume
		cloned.Event.Resume = &resume
	}
	if req.Snapshot != nil {
		snapshot := *req.Snapshot
		snapshot.ProviderMetadata = append([]byte(nil), req.Snapshot.ProviderMetadata...)
		if req.Snapshot.Reference != nil {
			reference := *req.Snapshot.Reference
			snapshot.Reference = &reference
		}
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
