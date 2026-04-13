package bridges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/testutil"
)

type recordedDeliveryCall struct {
	extensionName string
	request       DeliveryRequest
}

type fakeDeliveryTransport struct {
	mu      sync.Mutex
	calls   []recordedDeliveryCall
	acks    int
	updates chan struct{}
	handler func(context.Context, string, DeliveryRequest) (DeliveryAck, error)
}

func (f *fakeDeliveryTransport) DeliverBridge(
	ctx context.Context,
	extensionName string,
	req DeliveryRequest,
) (DeliveryAck, error) {
	if f == nil {
		return DeliveryAck{}, nil
	}

	f.mu.Lock()
	f.calls = append(f.calls, recordedDeliveryCall{
		extensionName: extensionName,
		request:       cloneDeliveryRequest(req),
	})
	handler := f.handler
	f.mu.Unlock()
	f.signalUpdate()

	var ack DeliveryAck
	var err error
	if handler != nil {
		ack, err = handler(ctx, extensionName, req)
	} else {
		ack = DeliveryAck{
			DeliveryID: req.Event.DeliveryID,
			Seq:        req.Event.Seq,
		}
	}
	if err == nil {
		f.mu.Lock()
		f.acks++
		f.mu.Unlock()
		f.signalUpdate()
	}
	return ack, err
}

func (f *fakeDeliveryTransport) snapshotCalls() []recordedDeliveryCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]recordedDeliveryCall, 0, len(f.calls))
	for _, call := range f.calls {
		out = append(out, recordedDeliveryCall{
			extensionName: call.extensionName,
			request:       cloneDeliveryRequest(call.request),
		})
	}
	return out
}

func (f *fakeDeliveryTransport) snapshotState() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls), f.acks
}

func (f *fakeDeliveryTransport) updateCh() chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updates == nil {
		f.updates = make(chan struct{}, 1)
	}
	return f.updates
}

func (f *fakeDeliveryTransport) signalUpdate() {
	if f == nil {
		return
	}
	ch := f.updateCh()
	select {
	case ch <- struct{}{}:
	default:
	}
}

func TestBrokerDeliversInOrderPerRoutingKeyWhileOtherRoutesStayActive(t *testing.T) {
	t.Parallel()

	releaseA := make(chan struct{})
	transport := &fakeDeliveryTransport{
		handler: func(ctx context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
			if req.Event.DeliveryID == "del-a" && req.Event.EventType == DeliveryEventTypeStart {
				select {
				case <-releaseA:
				case <-ctx.Done():
					return DeliveryAck{}, ctx.Err()
				}
			}
			return DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
		},
	}
	broker := NewBroker(transport)
	t.Cleanup(broker.Close)

	regA := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:     "sess-a",
		TurnID:        "turn-a",
		ExtensionName: "ext-telegram",
		RoutingKey:    testRoutingKey("brg-a", "peer-a"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-a",
			PeerID:           "peer-a",
			Mode:             DeliveryModeReply,
		},
	})
	regB := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:     "sess-b",
		TurnID:        "turn-b",
		ExtensionName: "ext-telegram",
		RoutingKey:    testRoutingKey("brg-b", "peer-b"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-b",
			PeerID:           "peer-b",
			Mode:             DeliveryModeReply,
		},
	})

	ctx := testutil.Context(t)
	deliveries := []DeliveryEvent{
		testDeliveryEvent(regA.DeliveryID, regA.BridgeInstanceID, regA.RoutingKey, regA.DeliveryTarget, 1, DeliveryEventTypeStart, "hello", false),
		testDeliveryEvent(regA.DeliveryID, regA.BridgeInstanceID, regA.RoutingKey, regA.DeliveryTarget, 2, DeliveryEventTypeDelta, "hello again", false),
		testDeliveryEvent(regA.DeliveryID, regA.BridgeInstanceID, regA.RoutingKey, regA.DeliveryTarget, 3, DeliveryEventTypeFinal, "hello again", true),
		testDeliveryEvent(regB.DeliveryID, regB.BridgeInstanceID, regB.RoutingKey, regB.DeliveryTarget, 1, DeliveryEventTypeStart, "route b", false),
		testDeliveryEvent(regB.DeliveryID, regB.BridgeInstanceID, regB.RoutingKey, regB.DeliveryTarget, 2, DeliveryEventTypeFinal, "route b", true),
	}
	for _, event := range deliveries {
		if err := broker.Deliver(ctx, event); err != nil {
			t.Fatalf("Deliver(%s:%d) error = %v", event.DeliveryID, event.Seq, err)
		}
	}

	waitForCalls(t, transport, 2)
	close(releaseA)
	waitForCalls(t, transport, 4)

	calls := transport.snapshotCalls()
	assertDeliveryOrder(t, calls, regB.DeliveryID, []string{DeliveryEventTypeStart, DeliveryEventTypeFinal}, []int64{1, 2})
	assertDeliveryStartsAndFinishesInOrder(t, calls, regA.DeliveryID)
}

func TestBrokerCoalescesIntermediateDeltaUnderBackpressure(t *testing.T) {
	t.Parallel()

	releaseStart := make(chan struct{})
	transport := &fakeDeliveryTransport{
		handler: func(ctx context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
			if req.Event.EventType == DeliveryEventTypeStart {
				select {
				case <-releaseStart:
				case <-ctx.Done():
					return DeliveryAck{}, ctx.Err()
				}
			}
			return DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
		},
	}
	broker := NewBroker(transport, WithDeliveryBrokerQueueCapacity(2))
	t.Cleanup(broker.Close)

	reg := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:     "sess-1",
		TurnID:        "turn-1",
		ExtensionName: "ext-telegram",
		RoutingKey:    testRoutingKey("brg-1", "peer-1"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-1",
			PeerID:           "peer-1",
			Mode:             DeliveryModeReply,
		},
	})

	ctx := testutil.Context(t)
	events := []DeliveryEvent{
		testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 1, DeliveryEventTypeStart, "h", false),
		testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 2, DeliveryEventTypeDelta, "he", false),
		testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 3, DeliveryEventTypeDelta, "hello", false),
		testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 4, DeliveryEventTypeFinal, "hello!", true),
	}
	for _, event := range events {
		if err := broker.Deliver(ctx, event); err != nil {
			t.Fatalf("Deliver(%d) error = %v", event.Seq, err)
		}
	}

	waitForCalls(t, transport, 1)
	close(releaseStart)
	waitForCalls(t, transport, 2)

	calls := transport.snapshotCalls()
	if len(calls) != 2 {
		t.Fatalf("len(delivery calls) = %d, want 2 after coalescing", len(calls))
	}
	assertDeliveryOrder(t, calls, reg.DeliveryID, []string{DeliveryEventTypeStart, DeliveryEventTypeFinal}, []int64{1, 4})
	if got, want := calls[1].request.Event.Content.Text, "hello!"; got != want {
		t.Fatalf("terminal content = %q, want %q", got, want)
	}
}

func TestBrokerAckTracksRemoteAndReplacementIDs(t *testing.T) {
	t.Parallel()

	transport := &fakeDeliveryTransport{
		handler: func(_ context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
			switch req.Event.Seq {
			case 1:
				return DeliveryAck{
					DeliveryID:      req.Event.DeliveryID,
					Seq:             req.Event.Seq,
					RemoteMessageID: "remote-1",
				}, nil
			case 2:
				return DeliveryAck{
					DeliveryID:             req.Event.DeliveryID,
					Seq:                    req.Event.Seq,
					RemoteMessageID:        "remote-2",
					ReplaceRemoteMessageID: "remote-1",
				}, nil
			default:
				return DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
			}
		},
	}
	broker := NewBroker(transport)
	t.Cleanup(broker.Close)

	reg := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:     "sess-ack",
		TurnID:        "turn-ack",
		ExtensionName: "ext-telegram",
		RoutingKey:    testRoutingKey("brg-ack", "peer-ack"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-ack",
			PeerID:           "peer-ack",
			Mode:             DeliveryModeReply,
		},
	})

	ctx := testutil.Context(t)
	if err := broker.Deliver(ctx, testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 1, DeliveryEventTypeStart, "hello", false)); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	if err := broker.Deliver(ctx, testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 2, DeliveryEventTypeDelta, "hello world", false)); err != nil {
		t.Fatalf("Deliver(delta) error = %v", err)
	}

	waitForAcks(t, transport, 2)

	snapshot, err := broker.Snapshot(ctx, reg.DeliveryID)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if got, want := snapshot.DeliveryID, reg.DeliveryID; got != want {
		t.Fatalf("snapshot.DeliveryID = %q, want %q", got, want)
	}
	if got, want := snapshot.LastAckedSeq, int64(2); got != want {
		t.Fatalf("snapshot.LastAckedSeq = %d, want %d", got, want)
	}
	if got, want := snapshot.RemoteMessageID, "remote-2"; got != want {
		t.Fatalf("snapshot.RemoteMessageID = %q, want %q", got, want)
	}
	if got, want := snapshot.ReplaceRemoteMessageID, "remote-1"; got != want {
		t.Fatalf("snapshot.ReplaceRemoteMessageID = %q, want %q", got, want)
	}
}

func TestBrokerSnapshotCapturesActiveDeliveryAfterFailure(t *testing.T) {
	t.Parallel()

	deltaFailed := make(chan struct{}, 1)
	transport := &fakeDeliveryTransport{
		handler: func(_ context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
			switch req.Event.EventType {
			case DeliveryEventTypeStart:
				return DeliveryAck{
					DeliveryID:      req.Event.DeliveryID,
					Seq:             req.Event.Seq,
					RemoteMessageID: "remote-1",
				}, nil
			case DeliveryEventTypeDelta:
				select {
				case deltaFailed <- struct{}{}:
				default:
				}
				return DeliveryAck{}, errors.New("adapter down")
			default:
				return DeliveryAck{}, errors.New("adapter still down")
			}
		},
	}
	broker := NewBroker(transport, WithDeliveryBrokerRetryDelay(100*time.Millisecond))
	t.Cleanup(broker.Close)

	reg := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:     "sess-resume",
		TurnID:        "turn-resume",
		ExtensionName: "ext-telegram",
		RoutingKey:    testRoutingKey("brg-resume", "peer-resume"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-resume",
			PeerID:           "peer-resume",
			Mode:             DeliveryModeReply,
		},
	})

	ctx := testutil.Context(t)
	if err := broker.Deliver(ctx, testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 1, DeliveryEventTypeStart, "hello", false)); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	waitForCalls(t, transport, 1)
	if err := broker.Deliver(ctx, testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 2, DeliveryEventTypeDelta, "hello world", false)); err != nil {
		t.Fatalf("Deliver(delta) error = %v", err)
	}

	select {
	case <-deltaFailed:
	case <-time.After(time.Second):
		t.Fatal("delta delivery failure was not observed")
	}

	snapshot, err := broker.Snapshot(ctx, reg.DeliveryID)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if got, want := snapshot.LatestSeq, int64(2); got != want {
		t.Fatalf("snapshot.LatestSeq = %d, want %d", got, want)
	}
	if got, want := snapshot.LastSentSeq, int64(2); got != want {
		t.Fatalf("snapshot.LastSentSeq = %d, want %d", got, want)
	}
	if got, want := snapshot.LastAckedSeq, int64(1); got != want {
		t.Fatalf("snapshot.LastAckedSeq = %d, want %d", got, want)
	}
	if got, want := snapshot.CurrentContent.Text, "hello world"; got != want {
		t.Fatalf("snapshot.CurrentContent.Text = %q, want %q", got, want)
	}
	if got, want := snapshot.RemoteMessageID, "remote-1"; got != want {
		t.Fatalf("snapshot.RemoteMessageID = %q, want %q", got, want)
	}
}

func TestBrokerDeliveryMetricsReflectBacklogAndClearAfterAck(t *testing.T) {
	t.Parallel()

	releaseStart := make(chan struct{})
	transport := &fakeDeliveryTransport{
		handler: func(ctx context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
			if req.Event.EventType == DeliveryEventTypeStart {
				select {
				case <-releaseStart:
				case <-ctx.Done():
					return DeliveryAck{}, ctx.Err()
				}
			}
			return DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
		},
	}
	broker := NewBroker(transport)
	t.Cleanup(broker.Close)

	reg := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:     "sess-metrics",
		TurnID:        "turn-metrics",
		ExtensionName: "ext-telegram",
		RoutingKey:    testRoutingKey("brg-metrics", "peer-metrics"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-metrics",
			PeerID:           "peer-metrics",
			Mode:             DeliveryModeReply,
		},
	})

	ctx := testutil.Context(t)
	if err := broker.Deliver(ctx, testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 1, DeliveryEventTypeStart, "hello", false)); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	if err := broker.Deliver(ctx, testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 2, DeliveryEventTypeDelta, "hello again", false)); err != nil {
		t.Fatalf("Deliver(delta) error = %v", err)
	}
	waitForCalls(t, transport, 1)

	metrics := broker.DeliveryMetrics()["brg-metrics"]
	if got, want := metrics.DeliveryBacklog, 1; got != want {
		t.Fatalf("DeliveryMetrics().DeliveryBacklog = %d, want %d", got, want)
	}

	close(releaseStart)
	waitForAcks(t, transport, 2)

	metrics = broker.DeliveryMetrics()["brg-metrics"]
	if got, want := metrics.DeliveryBacklog, 0; got != want {
		t.Fatalf("DeliveryMetrics().DeliveryBacklog after ack = %d, want %d", got, want)
	}
}

func TestBrokerDeliveryMetricsCaptureTerminalFailures(t *testing.T) {
	t.Parallel()

	transport := &fakeDeliveryTransport{
		handler: func(_ context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
			return DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
		},
	}
	broker := NewBroker(transport)
	t.Cleanup(broker.Close)

	reg := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:     "sess-failure",
		TurnID:        "turn-failure",
		ExtensionName: "ext-telegram",
		RoutingKey:    testRoutingKey("brg-failure", "peer-failure"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-failure",
			PeerID:           "peer-failure",
			Mode:             DeliveryModeReply,
		},
	})

	ctx := testutil.Context(t)
	if err := broker.Deliver(ctx, testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 1, DeliveryEventTypeStart, "hello", false)); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	errorEvent := testDeliveryEvent(reg.DeliveryID, reg.BridgeInstanceID, reg.RoutingKey, reg.DeliveryTarget, 2, DeliveryEventTypeError, "boom", true)
	errorEvent.Metadata = json.RawMessage(`{"error":"boom"}`)
	if err := broker.Deliver(ctx, errorEvent); err != nil {
		t.Fatalf("Deliver(error) error = %v", err)
	}

	metrics := broker.DeliveryMetrics()["brg-failure"]
	if got, want := metrics.DeliveryFailuresTotal, 1; got != want {
		t.Fatalf("DeliveryMetrics().DeliveryFailuresTotal = %d, want %d", got, want)
	}
	if got, want := metrics.LastError, "boom"; got != want {
		t.Fatalf("DeliveryMetrics().LastError = %q, want %q", got, want)
	}
}

func TestBrokerRejectedDeliverDoesNotAdvanceSnapshot(t *testing.T) {
	t.Parallel()

	var blockedDeliveryID string
	releaseStart := make(chan struct{})
	transport := &fakeDeliveryTransport{
		handler: func(ctx context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
			if req.Event.DeliveryID == blockedDeliveryID && req.Event.EventType == DeliveryEventTypeStart {
				select {
				case <-releaseStart:
				case <-ctx.Done():
					return DeliveryAck{}, ctx.Err()
				}
			}
			return DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
		},
	}
	broker := NewBroker(transport, WithDeliveryBrokerQueueCapacity(2))
	t.Cleanup(broker.Close)

	routingKey := testRoutingKey("brg-saturated", "peer-saturated")
	target := DeliveryTarget{
		BridgeInstanceID: "brg-saturated",
		PeerID:           "peer-saturated",
		Mode:             DeliveryModeReply,
	}
	regA := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:      "sess-saturated-a",
		TurnID:         "turn-saturated-a",
		ExtensionName:  "ext-telegram",
		RoutingKey:     routingKey,
		DeliveryTarget: target,
	})
	blockedDeliveryID = regA.DeliveryID
	regB := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:      "sess-saturated-b",
		TurnID:         "turn-saturated-b",
		ExtensionName:  "ext-telegram",
		RoutingKey:     routingKey,
		DeliveryTarget: target,
	})

	ctx := testutil.Context(t)
	if err := broker.Deliver(ctx, testDeliveryEvent(regA.DeliveryID, regA.BridgeInstanceID, regA.RoutingKey, regA.DeliveryTarget, 1, DeliveryEventTypeStart, "alpha", false)); err != nil {
		t.Fatalf("Deliver(regA start) error = %v", err)
	}
	waitForCalls(t, transport, 1)
	if err := broker.Deliver(ctx, testDeliveryEvent(regB.DeliveryID, regB.BridgeInstanceID, regB.RoutingKey, regB.DeliveryTarget, 1, DeliveryEventTypeStart, "bravo", false)); err != nil {
		t.Fatalf("Deliver(regB start) error = %v", err)
	}
	if err := broker.Deliver(ctx, testDeliveryEvent(regB.DeliveryID, regB.BridgeInstanceID, regB.RoutingKey, regB.DeliveryTarget, 2, DeliveryEventTypeFinal, "bravo done", true)); err != nil {
		t.Fatalf("Deliver(regB final) error = %v", err)
	}

	err := broker.Deliver(ctx, testDeliveryEvent(regA.DeliveryID, regA.BridgeInstanceID, regA.RoutingKey, regA.DeliveryTarget, 2, DeliveryEventTypeFinal, "alpha done", true))
	if !errors.Is(err, ErrDeliveryQueueSaturated) {
		t.Fatalf("Deliver(regA final) error = %v, want %v", err, ErrDeliveryQueueSaturated)
	}

	snapshot, err := broker.Snapshot(ctx, regA.DeliveryID)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if got, want := snapshot.LatestSeq, int64(1); got != want {
		t.Fatalf("LatestSeq after rejected deliver = %d, want %d", got, want)
	}
	if got, want := snapshot.LatestEventType, DeliveryEventTypeStart; got != want {
		t.Fatalf("LatestEventType after rejected deliver = %q, want %q", got, want)
	}
	if snapshot.Final {
		t.Fatal("Final after rejected deliver = true, want false")
	}
	if got, want := snapshot.CurrentContent.Text, "alpha"; got != want {
		t.Fatalf("CurrentContent after rejected deliver = %q, want %q", got, want)
	}
	if snapshot.Error != "" {
		t.Fatalf("Error after rejected deliver = %q, want empty", snapshot.Error)
	}

	close(releaseStart)
}

func mustRegisterTestDelivery(t *testing.T, broker *Broker, reg PromptDeliveryRegistration) DeliverySnapshot {
	t.Helper()

	snapshot, err := broker.RegisterPromptDelivery(testutil.Context(t), reg)
	if err != nil {
		t.Fatalf("RegisterPromptDelivery() error = %v", err)
	}
	if snapshot == nil {
		t.Fatal("RegisterPromptDelivery() snapshot = nil, want non-nil")
	}
	return *snapshot
}

func testRoutingKey(bridgeInstanceID string, peerID string) RoutingKey {
	return RoutingKey{
		Scope:            ScopeWorkspace,
		WorkspaceID:      "ws-1",
		BridgeInstanceID: bridgeInstanceID,
		PeerID:           peerID,
	}
}

func testDeliveryEvent(
	deliveryID string,
	bridgeInstanceID string,
	routingKey RoutingKey,
	target DeliveryTarget,
	seq int64,
	eventType string,
	text string,
	final bool,
) DeliveryEvent {
	return DeliveryEvent{
		DeliveryID:       deliveryID,
		BridgeInstanceID: bridgeInstanceID,
		RoutingKey:       routingKey,
		DeliveryTarget:   target,
		Seq:              seq,
		EventType:        eventType,
		Content:          MessageContent{Text: text},
		Final:            final,
	}
}

func waitForCalls(t *testing.T, transport *fakeDeliveryTransport, want int) {
	t.Helper()

	waitForTransportState(
		t,
		transport,
		func(calls int, _ int) bool {
			return calls >= want
		},
		func(calls int, _ int) string {
			return fmt.Sprintf("delivery call count did not reach %d before timeout; got %d", want, calls)
		},
	)
}

func waitForAcks(t *testing.T, transport *fakeDeliveryTransport, want int) {
	t.Helper()

	waitForTransportState(
		t,
		transport,
		func(_ int, acks int) bool {
			return acks >= want
		},
		func(_ int, acks int) string {
			return fmt.Sprintf("delivery ack count did not reach %d before timeout; got %d", want, acks)
		},
	)
}

func waitForTransportState(
	t *testing.T,
	transport *fakeDeliveryTransport,
	match func(calls int, acks int) bool,
	timeoutMessage func(calls int, acks int) string,
) {
	t.Helper()

	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	updates := transport.updateCh()
	for {
		calls, acks := transport.snapshotState()
		if match(calls, acks) {
			return
		}
		select {
		case <-updates:
		case <-timer.C:
			t.Fatal(timeoutMessage(calls, acks))
		}
	}
}

func assertDeliveryOrder(
	t *testing.T,
	calls []recordedDeliveryCall,
	deliveryID string,
	wantTypes []string,
	wantSeqs []int64,
) {
	t.Helper()

	gotTypes := make([]string, 0, len(calls))
	gotSeqs := make([]int64, 0, len(calls))
	for _, call := range calls {
		if call.request.Event.DeliveryID != deliveryID {
			continue
		}
		gotTypes = append(gotTypes, call.request.Event.EventType)
		gotSeqs = append(gotSeqs, call.request.Event.Seq)
	}
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("delivery %q type count = %v, want %v", deliveryID, gotTypes, wantTypes)
	}
	for idx := range wantTypes {
		if gotTypes[idx] != wantTypes[idx] {
			t.Fatalf("delivery %q type[%d] = %q, want %q", deliveryID, idx, gotTypes[idx], wantTypes[idx])
		}
		if gotSeqs[idx] != wantSeqs[idx] {
			t.Fatalf("delivery %q seq[%d] = %d, want %d", deliveryID, idx, gotSeqs[idx], wantSeqs[idx])
		}
	}
}

func assertDeliveryStartsAndFinishesInOrder(t *testing.T, calls []recordedDeliveryCall, deliveryID string) {
	t.Helper()

	filtered := make([]recordedDeliveryCall, 0, len(calls))
	for _, call := range calls {
		if call.request.Event.DeliveryID == deliveryID {
			filtered = append(filtered, call)
		}
	}
	if len(filtered) < 2 {
		t.Fatalf("delivery %q call count = %d, want at least start and final", deliveryID, len(filtered))
	}
	if got := filtered[0].request.Event.EventType; got != DeliveryEventTypeStart {
		t.Fatalf("delivery %q first event = %q, want start", deliveryID, got)
	}
	if got := filtered[len(filtered)-1].request.Event.EventType; got != DeliveryEventTypeFinal {
		t.Fatalf("delivery %q last event = %q, want final", deliveryID, got)
	}
	lastSeq := int64(0)
	for idx, call := range filtered {
		if call.request.Event.Seq <= lastSeq {
			t.Fatalf("delivery %q seq[%d] = %d, want increasing order after %d", deliveryID, idx, call.request.Event.Seq, lastSeq)
		}
		lastSeq = call.request.Event.Seq
	}
}
