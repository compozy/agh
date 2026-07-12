package bridges

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/compozy/agh/internal/testutil"
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

var _ DeliveryTransport = (*fakeDeliveryTransport)(nil)

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
	var blockedDeliveryID string
	transport := &fakeDeliveryTransport{
		handler: func(ctx context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
			if req.Event.DeliveryID == blockedDeliveryID && req.Event.EventType == DeliveryEventTypeStart {
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
	blockedDeliveryID = regA.DeliveryID
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
		testDeliveryEvent(
			regA.DeliveryID,
			regA.BridgeInstanceID,
			regA.RoutingKey,
			regA.DeliveryTarget,
			1,
			DeliveryEventTypeStart,
			"hello",
			false,
		),
		testDeliveryEvent(
			regA.DeliveryID,
			regA.BridgeInstanceID,
			regA.RoutingKey,
			regA.DeliveryTarget,
			2,
			DeliveryEventTypeDelta,
			"hello again",
			false,
		),
		testDeliveryEvent(
			regA.DeliveryID,
			regA.BridgeInstanceID,
			regA.RoutingKey,
			regA.DeliveryTarget,
			3,
			DeliveryEventTypeFinal,
			"hello again",
			true,
		),
		testDeliveryEvent(
			regB.DeliveryID,
			regB.BridgeInstanceID,
			regB.RoutingKey,
			regB.DeliveryTarget,
			1,
			DeliveryEventTypeStart,
			"route b",
			false,
		),
		testDeliveryEvent(
			regB.DeliveryID,
			regB.BridgeInstanceID,
			regB.RoutingKey,
			regB.DeliveryTarget,
			2,
			DeliveryEventTypeFinal,
			"route b",
			true,
		),
	}
	for _, event := range deliveries {
		if err := broker.Deliver(ctx, event); err != nil {
			t.Fatalf("Deliver(%s:%d) error = %v", event.DeliveryID, event.Seq, err)
		}
	}

	waitForAcks(t, transport, 2)
	close(releaseA)
	waitForAcks(t, transport, 4)

	calls := transport.snapshotCalls()
	assertDeliveryOrder(
		t,
		calls,
		regB.DeliveryID,
		[]DeliveryEventType{DeliveryEventTypeStart, DeliveryEventTypeFinal},
		[]int64{1, 2},
	)
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
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			1,
			DeliveryEventTypeStart,
			"h",
			false,
		),
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			2,
			DeliveryEventTypeDelta,
			"he",
			false,
		),
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			3,
			DeliveryEventTypeDelta,
			"hello",
			false,
		),
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			4,
			DeliveryEventTypeFinal,
			"hello!",
			true,
		),
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
	assertDeliveryOrder(
		t,
		calls,
		reg.DeliveryID,
		[]DeliveryEventType{DeliveryEventTypeStart, DeliveryEventTypeFinal},
		[]int64{1, 4},
	)
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
	if err := broker.Deliver(
		ctx,
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			1,
			DeliveryEventTypeStart,
			"hello",
			false,
		),
	); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	if err := broker.Deliver(
		ctx,
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			2,
			DeliveryEventTypeDelta,
			"hello world",
			false,
		),
	); err != nil {
		t.Fatalf("Deliver(delta) error = %v", err)
	}

	waitForAcks(t, transport, 2)

	snapshot := waitForSnapshot(t, broker, reg.DeliveryID, func(snapshot *DeliverySnapshot) bool {
		if snapshot == nil {
			return false
		}
		return snapshot.LastAckedSeq == 2 &&
			snapshot.RemoteMessageID == "remote-2" &&
			snapshot.ReplaceRemoteMessageID == "remote-1"
	})
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

func TestBrokerProgressAckPreservesTextualRemoteIDs(t *testing.T) {
	t.Run("Should advance the acknowledgement sequence without replacing textual remote ids", func(t *testing.T) {
		t.Parallel()

		transport := &fakeDeliveryTransport{
			handler: func(_ context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
				switch req.Event.EventType {
				case DeliveryEventTypeStart:
					return DeliveryAck{
						DeliveryID:      req.Event.DeliveryID,
						Seq:             req.Event.Seq,
						RemoteMessageID: "answer-1",
					}, nil
				case DeliveryEventTypeDelta:
					return DeliveryAck{
						DeliveryID:             req.Event.DeliveryID,
						Seq:                    req.Event.Seq,
						RemoteMessageID:        "answer-2",
						ReplaceRemoteMessageID: "answer-1",
					}, nil
				case DeliveryEventTypeProgress:
					return DeliveryAck{
						DeliveryID:             req.Event.DeliveryID,
						Seq:                    req.Event.Seq,
						RemoteMessageID:        "progress-2",
						ReplaceRemoteMessageID: "progress-1",
					}, nil
				default:
					return DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
				}
			},
		}
		broker := NewBroker(transport)
		t.Cleanup(broker.Close)

		reg := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
			SessionID:     "sess-progress-ack-state",
			TurnID:        "turn-progress-ack-state",
			ExtensionName: "ext-telegram",
			RoutingKey:    testRoutingKey("brg-progress-ack-state", "peer-progress-queue"),
			DeliveryTarget: DeliveryTarget{
				BridgeInstanceID: "brg-progress-ack-state",
				PeerID:           "peer-progress-queue",
				Mode:             DeliveryModeReply,
			},
		})

		ctx := testutil.Context(t)
		for _, event := range []DeliveryEvent{
			testDeliveryEvent(
				reg.DeliveryID,
				reg.BridgeInstanceID,
				reg.RoutingKey,
				reg.DeliveryTarget,
				1,
				DeliveryEventTypeStart,
				"answer",
				false,
			),
			testDeliveryEvent(
				reg.DeliveryID,
				reg.BridgeInstanceID,
				reg.RoutingKey,
				reg.DeliveryTarget,
				2,
				DeliveryEventTypeDelta,
				"answer complete",
				false,
			),
		} {
			if err := broker.Deliver(ctx, event); err != nil {
				t.Fatalf("Deliver(%s) error = %v", event.EventType, err)
			}
			waitForAcks(t, transport, int(event.Seq))
		}

		progress := testProgressDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			"call-progress-ack-state",
			"agh__terminal",
			ToolProgressPhaseStarted,
			3,
			1,
			"inspect",
		)
		if err := broker.Deliver(ctx, progress); err != nil {
			t.Fatalf("Deliver(progress) error = %v", err)
		}
		waitForAcks(t, transport, 3)

		snapshot := waitForSnapshot(t, broker, reg.DeliveryID, func(snapshot *DeliverySnapshot) bool {
			return snapshot != nil && snapshot.LastAckedSeq == 3
		})
		if got, want := snapshot.RemoteMessageID, "answer-2"; got != want {
			t.Fatalf("snapshot.RemoteMessageID = %q, want textual %q", got, want)
		}
		if got, want := snapshot.ReplaceRemoteMessageID, "answer-1"; got != want {
			t.Fatalf("snapshot.ReplaceRemoteMessageID = %q, want textual %q", got, want)
		}
	})
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
	if err := broker.Deliver(
		ctx,
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			1,
			DeliveryEventTypeStart,
			"hello",
			false,
		),
	); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	waitForCalls(t, transport, 1)
	if err := broker.Deliver(
		ctx,
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			2,
			DeliveryEventTypeDelta,
			"hello world",
			false,
		),
	); err != nil {
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

	now := time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)
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
	broker := NewBroker(transport, WithDeliveryBrokerNow(func() time.Time { return now }))
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
	if err := broker.Deliver(
		ctx,
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			1,
			DeliveryEventTypeStart,
			"hello",
			false,
		),
	); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	if err := broker.Deliver(
		ctx,
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			2,
			DeliveryEventTypeDelta,
			"hello again",
			false,
		),
	); err != nil {
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
	if got, want := metrics.LastSuccessAt, now; !got.Equal(want) {
		t.Fatalf("DeliveryMetrics().LastSuccessAt = %s, want %s", got, want)
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
	if err := broker.Deliver(
		ctx,
		testDeliveryEvent(
			reg.DeliveryID,
			reg.BridgeInstanceID,
			reg.RoutingKey,
			reg.DeliveryTarget,
			1,
			DeliveryEventTypeStart,
			"hello",
			false,
		),
	); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	errorEvent := testDeliveryEvent(
		reg.DeliveryID,
		reg.BridgeInstanceID,
		reg.RoutingKey,
		reg.DeliveryTarget,
		2,
		DeliveryEventTypeError,
		"boom",
		true,
	)
	errorEvent.Error = &DeliveryErrorDetail{Message: "boom"}
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

func TestBrokerProgressDeliveryFailuresRemainBestEffort(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		progressReply func(DeliveryRequest) (DeliveryAck, error)
		wantLastError string
		wantAcks      int
	}{
		{
			name: "Should redact a progress transport failure and continue content delivery without retry",
			progressReply: func(DeliveryRequest) (DeliveryAck, error) {
				return DeliveryAck{}, errors.New("adapter unavailable api_key=sk-progress-secret")
			},
			wantLastError: "adapter unavailable api_key=[REDACTED]",
			wantAcks:      2,
		},
		{
			name: "Should record an invalid progress acknowledgement and continue content delivery without retry",
			progressReply: func(req DeliveryRequest) (DeliveryAck, error) {
				return DeliveryAck{DeliveryID: "wrong-delivery", Seq: req.Event.Seq}, nil
			},
			wantLastError: "does not match event",
			wantAcks:      3,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			now := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
			transport := &fakeDeliveryTransport{
				handler: func(_ context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
					if req.Event.EventType == DeliveryEventTypeProgress {
						return test.progressReply(req)
					}
					return DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
				},
			}
			broker := NewBroker(transport, WithDeliveryBrokerNow(func() time.Time { return now }))
			t.Cleanup(broker.Close)

			reg := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
				SessionID:     "sess-progress-failure",
				TurnID:        "turn-progress-failure",
				ExtensionName: "ext-telegram",
				RoutingKey:    testRoutingKey("brg-progress-failure", "peer-progress-queue"),
				DeliveryTarget: DeliveryTarget{
					BridgeInstanceID: "brg-progress-failure",
					PeerID:           "peer-progress-queue",
					Mode:             DeliveryModeReply,
				},
			})

			ctx := testutil.Context(t)
			progress := testProgressDeliveryEvent(
				reg.DeliveryID,
				reg.BridgeInstanceID,
				"call-progress-failure",
				"agh__terminal",
				ToolProgressPhaseStarted,
				0,
				1,
				"inspect",
			)
			if err := broker.Deliver(ctx, progress); err != nil {
				t.Fatalf("Deliver(progress) error = %v", err)
			}
			for _, event := range []DeliveryEvent{
				testDeliveryEvent(
					reg.DeliveryID,
					reg.BridgeInstanceID,
					reg.RoutingKey,
					reg.DeliveryTarget,
					1,
					DeliveryEventTypeStart,
					"answer",
					false,
				),
				testDeliveryEvent(
					reg.DeliveryID,
					reg.BridgeInstanceID,
					reg.RoutingKey,
					reg.DeliveryTarget,
					2,
					DeliveryEventTypeFinal,
					"answer complete",
					true,
				),
			} {
				if err := broker.Deliver(ctx, event); err != nil {
					t.Fatalf("Deliver(%s) error = %v", event.EventType, err)
				}
			}

			waitForCalls(t, transport, 3)
			waitForAcks(t, transport, test.wantAcks)
			calls := transport.snapshotCalls()
			assertDeliveryOrder(
				t,
				calls,
				reg.DeliveryID,
				[]DeliveryEventType{DeliveryEventTypeProgress, DeliveryEventTypeStart, DeliveryEventTypeFinal},
				[]int64{0, 1, 2},
			)

			metrics := broker.DeliveryMetrics()[reg.BridgeInstanceID]
			if got, want := metrics.DeliveryDroppedTotal, 1; got != want {
				t.Fatalf("DeliveryDroppedTotal = %d, want %d", got, want)
			}
			if got, want := metrics.DeliveryDroppedByReason["progress_delivery_failed"], 1; got != want {
				t.Fatalf("progress_delivery_failed drops = %d, want %d", got, want)
			}
			if got := metrics.LastError; !strings.Contains(got, test.wantLastError) {
				t.Fatalf("LastError = %q, want containing %q", got, test.wantLastError)
			}
			if got, want := metrics.LastErrorAt, now; !got.Equal(want) {
				t.Fatalf("LastErrorAt = %s, want %s", got, want)
			}
			if got := metrics.DeliveryFailuresTotal; got != 0 {
				t.Fatalf("DeliveryFailuresTotal = %d, want progress failure excluded", got)
			}
		})
	}
}

func TestBrokerDeliveryMetricsForReadsOnlyRequestedBridgeRoutes(t *testing.T) {
	t.Parallel()

	t.Run("Should exclude foreign metrics and route backlog from a bounded snapshot", func(t *testing.T) {
		t.Parallel()

		const (
			pageBridgeID               = "brg-page"
			foreignBridgeID            = "brg-foreign"
			pageBlockedDeliveryID      = "delivery-page-blocked"
			foreignBlockedDeliveryID   = "delivery-foreign-blocked"
			recreatedBlockedDeliveryID = "delivery-page-recreated-blocked"
		)

		pageRelease := make(chan struct{})
		foreignRelease := make(chan struct{})
		recreatedRelease := make(chan struct{})
		transport := &fakeDeliveryTransport{
			handler: func(ctx context.Context, _ string, req DeliveryRequest) (DeliveryAck, error) {
				var release <-chan struct{}
				switch req.Event.DeliveryID {
				case pageBlockedDeliveryID:
					release = pageRelease
				case foreignBlockedDeliveryID:
					release = foreignRelease
				case recreatedBlockedDeliveryID:
					release = recreatedRelease
				}
				if release != nil {
					select {
					case <-release:
					case <-ctx.Done():
						return DeliveryAck{}, ctx.Err()
					}
				}
				return DeliveryAck{DeliveryID: req.Event.DeliveryID, Seq: req.Event.Seq}, nil
			},
		}
		broker := NewBroker(transport)
		t.Cleanup(broker.Close)
		t.Cleanup(func() {
			closeTestSignal(t, pageRelease)
			closeTestSignal(t, foreignRelease)
			closeTestSignal(t, recreatedRelease)
		})

		pageRoutingKey := testRoutingKey(pageBridgeID, "peer-page")
		pageTarget := DeliveryTarget{
			BridgeInstanceID: pageBridgeID,
			PeerID:           "peer-page",
			Mode:             DeliveryModeReply,
		}
		foreignRoutingKey := testRoutingKey(foreignBridgeID, "peer-foreign")
		foreignTarget := DeliveryTarget{
			BridgeInstanceID: foreignBridgeID,
			PeerID:           "peer-foreign",
			Mode:             DeliveryModeReply,
		}
		registerStart := func(
			deliveryID string,
			sessionID string,
			turnID string,
			extensionName string,
			routingKey RoutingKey,
			target DeliveryTarget,
		) DeliverySnapshot {
			t.Helper()
			registered := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
				SessionID:      sessionID,
				TurnID:         turnID,
				ExtensionName:  extensionName,
				DeliveryID:     deliveryID,
				RoutingKey:     routingKey,
				DeliveryTarget: target,
			})
			if err := broker.Deliver(testutil.Context(t), testDeliveryEvent(
				registered.DeliveryID,
				registered.BridgeInstanceID,
				routingKey,
				target,
				1,
				DeliveryEventTypeStart,
				"answer",
				false,
			)); err != nil {
				t.Fatalf("Deliver(start %s) error = %v", deliveryID, err)
			}
			return registered
		}
		deliverFinal := func(registered DeliverySnapshot, routingKey RoutingKey, target DeliveryTarget) {
			t.Helper()
			if err := broker.Deliver(testutil.Context(t), testDeliveryEvent(
				registered.DeliveryID,
				registered.BridgeInstanceID,
				routingKey,
				target,
				2,
				DeliveryEventTypeFinal,
				"answer complete",
				true,
			)); err != nil {
				t.Fatalf("Deliver(final %s) error = %v", registered.DeliveryID, err)
			}
		}

		pageBlocked := registerStart(
			pageBlockedDeliveryID,
			"sess-page-blocked",
			"turn-page-blocked",
			"ext-page",
			pageRoutingKey,
			pageTarget,
		)
		registerStart(
			foreignBlockedDeliveryID,
			"sess-foreign-blocked",
			"turn-foreign-blocked",
			"ext-foreign",
			foreignRoutingKey,
			foreignTarget,
		)
		waitForCalls(t, transport, 2)
		pageQueued := registerStart(
			"delivery-page-queued",
			"sess-page-queued",
			"turn-page-queued",
			"ext-page",
			pageRoutingKey,
			pageTarget,
		)
		registerStart(
			"delivery-foreign-queued",
			"sess-foreign-queued",
			"turn-foreign-queued",
			"ext-foreign",
			foreignRoutingKey,
			foreignTarget,
		)

		broker.mu.Lock()
		broker.metrics[pageBridgeID] = &instanceDeliveryMetrics{deliveryFailuresTotal: 2}
		broker.metrics[foreignBridgeID] = &instanceDeliveryMetrics{deliveryFailuresTotal: 99}
		if got, want := len(broker.bridgeRoutes[pageBridgeID]), 1; got != want {
			broker.mu.Unlock()
			t.Fatalf("page route index size = %d, want %d", got, want)
		}
		if got, want := len(broker.bridgeRoutes[foreignBridgeID]), 1; got != want {
			broker.mu.Unlock()
			t.Fatalf("foreign route index size = %d, want %d", got, want)
		}
		broker.mu.Unlock()

		snapshot, err := broker.DeliveryMetricsFor([]string{pageBridgeID})
		if err != nil {
			t.Fatalf("DeliveryMetricsFor() error = %v", err)
		}
		if got, want := len(snapshot), 1; got != want {
			t.Fatalf("len(DeliveryMetricsFor()) = %d, want %d: %#v", got, want, snapshot)
		}
		if got, want := snapshot[pageBridgeID].DeliveryBacklog, 1; got != want {
			t.Fatalf("DeliveryMetricsFor().DeliveryBacklog = %d, want %d", got, want)
		}
		if got, want := snapshot[pageBridgeID].DeliveryFailuresTotal, 2; got != want {
			t.Fatalf("DeliveryMetricsFor().DeliveryFailuresTotal = %d, want %d", got, want)
		}
		if _, leaked := snapshot[foreignBridgeID]; leaked {
			t.Fatalf("DeliveryMetricsFor() leaked foreign bridge telemetry: %#v", snapshot)
		}

		deliverFinal(pageBlocked, pageRoutingKey, pageTarget)
		deliverFinal(pageQueued, pageRoutingKey, pageTarget)
		closeTestSignal(t, pageRelease)
		waitForBridgeRouteCount(t, broker, pageBridgeID, 0)

		registerStart(
			recreatedBlockedDeliveryID,
			"sess-page-recreated-blocked",
			"turn-page-recreated-blocked",
			"ext-page",
			pageRoutingKey,
			pageTarget,
		)
		waitForCalls(t, transport, 6)
		registerStart(
			"delivery-page-recreated-queued",
			"sess-page-recreated-queued",
			"turn-page-recreated-queued",
			"ext-page",
			pageRoutingKey,
			pageTarget,
		)

		broker.mu.Lock()
		if got, want := len(broker.bridgeRoutes[pageBridgeID]), 1; got != want {
			broker.mu.Unlock()
			t.Fatalf("recreated page route index size = %d, want %d", got, want)
		}
		broker.mu.Unlock()

		snapshot, err = broker.DeliveryMetricsFor([]string{pageBridgeID})
		if err != nil {
			t.Fatalf("DeliveryMetricsFor(recreated) error = %v", err)
		}
		if got, want := snapshot[pageBridgeID].DeliveryBacklog, 1; got != want {
			t.Fatalf("recreated DeliveryBacklog = %d, want %d without duplicate index entries", got, want)
		}
	})
}

func TestBrokerTerminalDeliverySurvivesSaturatedQueue(t *testing.T) {
	t.Run("Should preserve the terminal delivery when the route queue is saturated", func(t *testing.T) {
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
		if err := broker.Deliver(
			ctx,
			testDeliveryEvent(
				regA.DeliveryID,
				regA.BridgeInstanceID,
				regA.RoutingKey,
				regA.DeliveryTarget,
				1,
				DeliveryEventTypeStart,
				"alpha",
				false,
			),
		); err != nil {
			t.Fatalf("Deliver(regA start) error = %v", err)
		}
		waitForCalls(t, transport, 1)
		if err := broker.Deliver(
			ctx,
			testDeliveryEvent(
				regB.DeliveryID,
				regB.BridgeInstanceID,
				regB.RoutingKey,
				regB.DeliveryTarget,
				1,
				DeliveryEventTypeStart,
				"bravo",
				false,
			),
		); err != nil {
			t.Fatalf("Deliver(regB start) error = %v", err)
		}
		if err := broker.Deliver(
			ctx,
			testDeliveryEvent(
				regB.DeliveryID,
				regB.BridgeInstanceID,
				regB.RoutingKey,
				regB.DeliveryTarget,
				2,
				DeliveryEventTypeFinal,
				"bravo done",
				true,
			),
		); err != nil {
			t.Fatalf("Deliver(regB final) error = %v", err)
		}

		err := broker.Deliver(
			ctx,
			testDeliveryEvent(
				regA.DeliveryID,
				regA.BridgeInstanceID,
				regA.RoutingKey,
				regA.DeliveryTarget,
				2,
				DeliveryEventTypeFinal,
				"alpha done",
				true,
			),
		)
		if err != nil {
			t.Fatalf("Deliver(regA final) error = %v, want preserved terminal", err)
		}

		snapshot, err := broker.Snapshot(ctx, regA.DeliveryID)
		if err != nil {
			t.Fatalf("Snapshot() error = %v", err)
		}
		if got, want := snapshot.LatestSeq, int64(2); got != want {
			t.Fatalf("LatestSeq after terminal enqueue = %d, want %d", got, want)
		}
		if got, want := snapshot.LatestEventType, DeliveryEventTypeFinal; got != want {
			t.Fatalf("LatestEventType after terminal enqueue = %q, want %q", got, want)
		}
		if !snapshot.Final {
			t.Fatal("Final after terminal enqueue = false, want true")
		}
		if got, want := snapshot.CurrentContent.Text, "alpha done"; got != want {
			t.Fatalf("CurrentContent after terminal enqueue = %q, want %q", got, want)
		}
		if snapshot.Error != "" {
			t.Fatalf("Error after terminal enqueue = %q, want empty", snapshot.Error)
		}

		close(releaseStart)
		waitForCalls(t, transport, 4)
		terminalCount := 0
		for _, call := range transport.snapshotCalls() {
			if call.request.Event.DeliveryID == regA.DeliveryID &&
				call.request.Event.EventType == DeliveryEventTypeFinal {
				terminalCount++
			}
		}
		if got, want := terminalCount, 1; got != want {
			t.Fatalf("regA terminal call count = %d, want %d", got, want)
		}
	})
}

func TestBrokerProgressQueueBackpressure(t *testing.T) {
	t.Parallel()

	t.Run("Should coalesce only the same tool call and phase newest wins", func(t *testing.T) {
		t.Parallel()

		broker := NewBroker(nil, WithDeliveryBrokerQueueCapacity(2))
		t.Cleanup(broker.Close)
		route := &routeWorker{bridgeInstanceID: "brg-progress-coalesce"}
		delivery := &activeDelivery{deliveryID: "del-progress-coalesce"}
		broker.deliveries[delivery.deliveryID] = delivery

		first := testProgressDeliveryEvent(
			delivery.deliveryID,
			route.bridgeInstanceID,
			"call-same",
			"agh__terminal",
			ToolProgressPhaseStarted,
			1,
			1,
			"first",
		)
		newest := first
		newest.Seq = 2
		newest.Progress = cloneToolProgress(first.Progress)
		newest.Progress.Index = 2
		newest.Progress.Preview = "newest"
		if err := broker.enqueueEventLocked(route, delivery, first); err != nil {
			t.Fatalf("enqueueEventLocked(first progress) error = %v", err)
		}
		if err := broker.enqueueEventLocked(route, delivery, newest); err != nil {
			t.Fatalf("enqueueEventLocked(newest progress) error = %v", err)
		}

		if got, want := len(route.queue), 1; got != want {
			t.Fatalf("queue length = %d, want %d same-key slot", got, want)
		}
		key, err := progressKeyFor(first)
		if err != nil {
			t.Fatalf("progressKeyFor(first) error = %v", err)
		}
		pending := delivery.pendingProgress[key]
		if pending == nil || pending.Progress == nil {
			t.Fatalf("pending same-key progress = %#v, want newest event", pending)
		}
		if got, want := pending.Progress.Preview, "newest"; got != want {
			t.Fatalf("pending progress preview = %q, want %q", got, want)
		}
		metrics := broker.DeliveryMetrics()[route.bridgeInstanceID]
		if got, want := metrics.DeliveryDroppedByReason["progress_coalesced"], 1; got != want {
			t.Fatalf("progress_coalesced metric = %d, want %d", got, want)
		}
	})

	t.Run("Should keep distinct tool keys separate and count forced intermediate drops", func(t *testing.T) {
		t.Parallel()

		broker := NewBroker(nil, WithDeliveryBrokerQueueCapacity(2))
		t.Cleanup(broker.Close)
		route := &routeWorker{bridgeInstanceID: "brg-progress-distinct"}
		delivery := &activeDelivery{deliveryID: "del-progress-distinct"}
		broker.deliveries[delivery.deliveryID] = delivery

		for index, callID := range []string{"call-a", "call-b"} {
			event := testProgressDeliveryEvent(
				delivery.deliveryID,
				route.bridgeInstanceID,
				callID,
				"agh__task_list",
				ToolProgressPhaseStarted,
				int64(index+1),
				index+1,
				callID,
			)
			if err := broker.enqueueEventLocked(route, delivery, event); err != nil {
				t.Fatalf("enqueueEventLocked(%s) error = %v", callID, err)
			}
		}
		if got, want := len(route.queue), 2; got != want {
			t.Fatalf("queue length = %d, want %d distinct progress slots", got, want)
		}
		if route.queue[0].progress == route.queue[1].progress {
			t.Fatalf("distinct progress keys merged: %#v", route.queue)
		}

		incoming := testProgressDeliveryEvent(
			delivery.deliveryID,
			route.bridgeInstanceID,
			"call-c",
			"agh__task_list",
			ToolProgressPhaseStarted,
			3,
			3,
			"call-c",
		)
		if err := broker.enqueueEventLocked(route, delivery, incoming); err != nil {
			t.Fatalf("enqueueEventLocked(incoming distinct progress) error = %v", err)
		}
		if got, want := len(route.queue), 2; got != want {
			t.Fatalf("queue length after eviction = %d, want bounded %d", got, want)
		}
		metrics := broker.DeliveryMetrics()[route.bridgeInstanceID]
		if got, want := metrics.DeliveryDroppedByReason["progress_evicted"], 1; got != want {
			t.Fatalf("progress_evicted metric = %d, want %d", got, want)
		}
	})

	t.Run("Should preserve delivery terminals when the queue is full", func(t *testing.T) {
		t.Parallel()

		broker := NewBroker(nil, WithDeliveryBrokerQueueCapacity(2))
		t.Cleanup(broker.Close)
		route := &routeWorker{bridgeInstanceID: "brg-progress-terminal"}
		deliveryA := &activeDelivery{deliveryID: "del-progress-terminal-a"}
		deliveryB := &activeDelivery{deliveryID: "del-progress-terminal-b"}
		broker.deliveries[deliveryA.deliveryID] = deliveryA
		broker.deliveries[deliveryB.deliveryID] = deliveryB

		target := DeliveryTarget{
			BridgeInstanceID: route.bridgeInstanceID,
			PeerID:           "peer-terminal",
			Mode:             DeliveryModeReply,
		}
		for _, delivery := range []*activeDelivery{deliveryA, deliveryB} {
			start := testDeliveryEvent(
				delivery.deliveryID,
				route.bridgeInstanceID,
				testRoutingKey(route.bridgeInstanceID, "peer-terminal"),
				target,
				1,
				DeliveryEventTypeStart,
				"answer",
				false,
			)
			if err := broker.enqueueEventLocked(route, delivery, start); err != nil {
				t.Fatalf("enqueueEventLocked(start %s) error = %v", delivery.deliveryID, err)
			}
		}

		terminal := testDeliveryEvent(
			deliveryA.deliveryID,
			route.bridgeInstanceID,
			testRoutingKey(route.bridgeInstanceID, "peer-terminal"),
			target,
			2,
			DeliveryEventTypeFinal,
			"answer",
			true,
		)
		if err := broker.enqueueEventLocked(route, deliveryA, terminal); err != nil {
			t.Fatalf("enqueueEventLocked(terminal on full queue) error = %v", err)
		}
		if deliveryA.pendingTerminal == nil || !deliveryA.queuedTerminal {
			t.Fatalf(
				"terminal state = (%#v, %v), want queued terminal",
				deliveryA.pendingTerminal,
				deliveryA.queuedTerminal,
			)
		}
		if got, want := len(route.queue), 3; got != want {
			t.Fatalf("queue length = %d, want %d with preserved terminal overflow", got, want)
		}
	})
}

func testProgressDeliveryEvent(
	deliveryID string,
	bridgeInstanceID string,
	toolCallID string,
	toolID string,
	phase ToolProgressPhase,
	seq int64,
	index int,
	preview string,
) DeliveryEvent {
	return DeliveryEvent{
		DeliveryID:       deliveryID,
		BridgeInstanceID: bridgeInstanceID,
		RoutingKey:       testRoutingKey(bridgeInstanceID, "peer-progress-queue"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: bridgeInstanceID,
			PeerID:           "peer-progress-queue",
			Mode:             DeliveryModeReply,
		},
		Seq:       seq,
		EventType: DeliveryEventTypeProgress,
		Progress: &ToolProgress{
			ToolCallID: toolCallID,
			ToolID:     toolID,
			Phase:      phase,
			Label:      "Running",
			Preview:    preview,
			Emoji:      "⚙️",
			Index:      index,
		},
	}
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
	eventType DeliveryEventType,
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

func waitForBridgeRouteCount(t *testing.T, broker *Broker, bridgeInstanceID string, want int) {
	t.Helper()

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		broker.mu.Lock()
		got := len(broker.bridgeRoutes[bridgeInstanceID])
		broker.mu.Unlock()
		if got == want {
			return
		}
		select {
		case <-ticker.C:
		case <-timer.C:
			t.Fatalf("bridge %q route count did not reach %d before timeout; got %d", bridgeInstanceID, want, got)
		}
	}
}

func closeTestSignal(t *testing.T, signal chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	default:
		close(signal)
	}
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
	wantTypes []DeliveryEventType,
	wantSeqs []int64,
) {
	t.Helper()

	gotTypes := make([]DeliveryEventType, 0, len(calls))
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
			t.Fatalf(
				"delivery %q seq[%d] = %d, want increasing order after %d",
				deliveryID,
				idx,
				call.request.Event.Seq,
				lastSeq,
			)
		}
		lastSeq = call.request.Event.Seq
	}
}
