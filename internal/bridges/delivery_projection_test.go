package bridges

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/testutil"
)

func TestBrokerSetTransportFlushesQueuedResume(t *testing.T) {
	t.Parallel()

	parentCtx, cancelParent := context.WithCancel(context.Background())
	defer cancelParent()

	now := time.Date(2026, time.April, 11, 12, 0, 0, 0, time.UTC)
	broker := NewBroker(
		nil,
		WithDeliveryBrokerNow(func() time.Time { return now }),
		WithDeliveryBrokerRetryDelay(5*time.Millisecond),
		WithDeliveryBrokerRequestTimeout(75*time.Millisecond),
		WithDeliveryBrokerLifecycleContext(parentCtx),
	)
	t.Cleanup(broker.Close)

	if got, want := broker.requestTimeout, 75*time.Millisecond; got != want {
		t.Fatalf("broker.requestTimeout = %s, want %s", got, want)
	}
	if got := broker.now(); !got.Equal(now) {
		t.Fatalf("broker.now() = %s, want %s", got, now)
	}

	reg := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:     "sess-resume-route",
		TurnID:        "turn-resume-route",
		ExtensionName: "ext-telegram",
		RoutingKey:    testRoutingKey("brg-resume-route", "peer-resume-route"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-resume-route",
			PeerID:           "peer-resume-route",
			Mode:             DeliveryModeReply,
		},
	})

	ctx := testutil.Context(t)
	if err := broker.Deliver(ctx, testDeliveryEvent(
		reg.DeliveryID,
		reg.BridgeInstanceID,
		reg.RoutingKey,
		reg.DeliveryTarget,
		1,
		DeliveryEventTypeStart,
		"queued",
		false,
	)); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}

	waitForSnapshot(t, broker, reg.DeliveryID, func(snapshot *DeliverySnapshot) bool {
		if snapshot.LastSentSeq != 1 || snapshot.LastAckedSeq != 0 {
			return false
		}
		metrics := broker.DeliveryMetrics()
		entry, ok := metrics[reg.BridgeInstanceID]
		return ok && strings.TrimSpace(entry.LastError) != ""
	})

	transport := &fakeDeliveryTransport{}
	broker.SetTransport(transport)
	waitForCalls(t, transport, 1)

	calls := transport.snapshotCalls()
	if len(calls) != 1 {
		t.Fatalf("len(delivery calls) = %d, want 1", len(calls))
	}
	if got, want := calls[0].request.Event.EventType, DeliveryEventTypeResume; got != want {
		t.Fatalf("resume event type = %q, want %q", got, want)
	}
	if calls[0].request.Snapshot == nil {
		t.Fatal("resume request snapshot = nil, want non-nil")
	}
	if got, want := calls[0].request.Snapshot.DeliveryID, reg.DeliveryID; got != want {
		t.Fatalf("resume snapshot delivery id = %q, want %q", got, want)
	}

	cancelParent()
	select {
	case <-broker.lifecycleCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("broker lifecycle context was not canceled")
	}
}

func TestBrokerProjectEventDeduplicatesAndFailsSession(t *testing.T) {
	t.Parallel()

	transport := &fakeDeliveryTransport{}
	broker := NewBroker(transport)
	t.Cleanup(broker.Close)

	seedTime := time.Date(2026, time.April, 11, 12, 1, 0, 0, time.UTC)
	reg := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:     "sess-project",
		TurnID:        "turn-project",
		ExtensionName: "ext-telegram",
		RoutingKey:    testRoutingKey("brg-project", "peer-project"),
		DeliveryTarget: DeliveryTarget{
			BridgeInstanceID: "brg-project",
			PeerID:           "peer-project",
			Mode:             DeliveryModeReply,
		},
		SeedEvents: []DeliveryProjectionEvent{
			{
				Type:        "agent_message",
				TurnID:      "turn-project",
				Timestamp:   seedTime,
				Text:        "hello",
				Fingerprint: "seed-1",
			},
		},
	})

	waitForCalls(t, transport, 1)
	calls := transport.snapshotCalls()
	assertDeliveryOrder(t, calls, reg.DeliveryID, []string{DeliveryEventTypeStart}, []int64{1})
	if got, want := calls[0].request.Event.Content.Text, "hello"; got != want {
		t.Fatalf("seed content = %q, want %q", got, want)
	}

	ctx := testutil.Context(t)
	if err := broker.ProjectEvent(ctx, reg.SessionID, DeliveryProjectionEvent{
		Type:        "agent_message",
		TurnID:      reg.TurnID,
		Timestamp:   seedTime,
		Text:        "hello",
		Fingerprint: "seed-1",
	}); err != nil {
		t.Fatalf("ProjectEvent(duplicate) error = %v", err)
	}
	assertCallCountStable(t, transport, 1, 50*time.Millisecond)

	nextEvent := DeliveryProjectionEvent{
		Type:      "agent_message",
		TurnID:    reg.TurnID,
		Timestamp: seedTime.Add(time.Second),
		Text:      " world",
	}
	if err := broker.ProjectEvent(ctx, reg.SessionID, nextEvent); err != nil {
		t.Fatalf("ProjectEvent(delta) error = %v", err)
	}

	waitForCalls(t, transport, 2)
	calls = transport.snapshotCalls()
	assertDeliveryOrder(t, calls, reg.DeliveryID, []string{DeliveryEventTypeStart, DeliveryEventTypeDelta}, []int64{1, 2})
	if got, want := calls[1].request.Event.Content.Text, "hello world"; got != want {
		t.Fatalf("delta content = %q, want %q", got, want)
	}

	if got := agentEventFingerprint(nextEvent); !strings.Contains(got, "agent_message|turn-project|") {
		t.Fatalf("agentEventFingerprint() = %q, want composed fingerprint", got)
	}

	if err := broker.FailSession(ctx, reg.SessionID, "adapter stopped"); err != nil {
		t.Fatalf("FailSession() error = %v", err)
	}

	waitForCalls(t, transport, 3)
	calls = transport.snapshotCalls()
	assertDeliveryOrder(
		t,
		calls,
		reg.DeliveryID,
		[]string{DeliveryEventTypeStart, DeliveryEventTypeDelta, DeliveryEventTypeError},
		[]int64{1, 2, 3},
	)
	last := calls[len(calls)-1].request.Event
	if !last.Final {
		t.Fatal("failed-session event Final = false, want true")
	}
	if got, want := deliveryErrorText(last.Metadata), "adapter stopped"; got != want {
		t.Fatalf("deliveryErrorText(metadata) = %q, want %q", got, want)
	}
}

func TestBrokerRejectedProjectedEventDoesNotAdvanceStateOrConsumeFingerprint(t *testing.T) {
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

	routingKey := testRoutingKey("brg-project-saturated", "peer-project-saturated")
	target := DeliveryTarget{
		BridgeInstanceID: "brg-project-saturated",
		PeerID:           "peer-project-saturated",
		Mode:             DeliveryModeReply,
	}
	regA := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:      "sess-project-saturated-a",
		TurnID:         "turn-project-saturated-a",
		ExtensionName:  "ext-telegram",
		RoutingKey:     routingKey,
		DeliveryTarget: target,
	})
	blockedDeliveryID = regA.DeliveryID
	regB := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:      "sess-project-saturated-b",
		TurnID:         "turn-project-saturated-b",
		ExtensionName:  "ext-telegram",
		RoutingKey:     routingKey,
		DeliveryTarget: target,
	})

	ctx := testutil.Context(t)
	if err := broker.Deliver(ctx, testDeliveryEvent(regA.DeliveryID, regA.BridgeInstanceID, regA.RoutingKey, regA.DeliveryTarget, 1, DeliveryEventTypeStart, "hello", false)); err != nil {
		t.Fatalf("Deliver(regA start) error = %v", err)
	}
	waitForCalls(t, transport, 1)
	if err := broker.Deliver(ctx, testDeliveryEvent(regB.DeliveryID, regB.BridgeInstanceID, regB.RoutingKey, regB.DeliveryTarget, 1, DeliveryEventTypeStart, "other", false)); err != nil {
		t.Fatalf("Deliver(regB start) error = %v", err)
	}
	if err := broker.Deliver(ctx, testDeliveryEvent(regB.DeliveryID, regB.BridgeInstanceID, regB.RoutingKey, regB.DeliveryTarget, 2, DeliveryEventTypeFinal, "other done", true)); err != nil {
		t.Fatalf("Deliver(regB final) error = %v", err)
	}

	projected := DeliveryProjectionEvent{
		Type:        "agent_message",
		TurnID:      regA.TurnID,
		Timestamp:   time.Date(2026, time.April, 11, 12, 5, 0, 0, time.UTC),
		Text:        " world",
		Fingerprint: "fp-project-saturated",
	}
	if err := broker.ProjectEvent(ctx, regA.SessionID, projected); !errors.Is(err, ErrDeliveryQueueSaturated) {
		t.Fatalf("ProjectEvent() error = %v, want %v", err, ErrDeliveryQueueSaturated)
	}

	snapshot, err := broker.Snapshot(ctx, regA.DeliveryID)
	if err != nil {
		t.Fatalf("Snapshot(after rejected project) error = %v", err)
	}
	if got, want := snapshot.LatestSeq, int64(1); got != want {
		t.Fatalf("LatestSeq after rejected project = %d, want %d", got, want)
	}
	if got, want := snapshot.CurrentContent.Text, "hello"; got != want {
		t.Fatalf("CurrentContent after rejected project = %q, want %q", got, want)
	}
	if snapshot.Final {
		t.Fatal("Final after rejected project = true, want false")
	}

	close(releaseStart)
	waitForCalls(t, transport, 3)

	if err := broker.ProjectEvent(ctx, regA.SessionID, projected); err != nil {
		t.Fatalf("ProjectEvent(retry) error = %v", err)
	}
	waitForCalls(t, transport, 4)

	snapshot, err = broker.Snapshot(ctx, regA.DeliveryID)
	if err != nil {
		t.Fatalf("Snapshot(after retry) error = %v", err)
	}
	if got, want := snapshot.LatestSeq, int64(2); got != want {
		t.Fatalf("LatestSeq after retry = %d, want %d", got, want)
	}
	if got, want := snapshot.CurrentContent.Text, "hello world"; got != want {
		t.Fatalf("CurrentContent after retry = %q, want %q", got, want)
	}
}

func TestBrokerRejectedFailSessionDoesNotFinalizeDelivery(t *testing.T) {
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

	routingKey := testRoutingKey("brg-fail-saturated", "peer-fail-saturated")
	target := DeliveryTarget{
		BridgeInstanceID: "brg-fail-saturated",
		PeerID:           "peer-fail-saturated",
		Mode:             DeliveryModeReply,
	}
	regA := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:      "sess-fail-saturated-a",
		TurnID:         "turn-fail-saturated-a",
		ExtensionName:  "ext-telegram",
		RoutingKey:     routingKey,
		DeliveryTarget: target,
	})
	blockedDeliveryID = regA.DeliveryID
	regB := mustRegisterTestDelivery(t, broker, PromptDeliveryRegistration{
		SessionID:      "sess-fail-saturated-b",
		TurnID:         "turn-fail-saturated-b",
		ExtensionName:  "ext-telegram",
		RoutingKey:     routingKey,
		DeliveryTarget: target,
	})

	ctx := testutil.Context(t)
	if err := broker.Deliver(ctx, testDeliveryEvent(regA.DeliveryID, regA.BridgeInstanceID, regA.RoutingKey, regA.DeliveryTarget, 1, DeliveryEventTypeStart, "hello", false)); err != nil {
		t.Fatalf("Deliver(regA start) error = %v", err)
	}
	waitForCalls(t, transport, 1)
	if err := broker.Deliver(ctx, testDeliveryEvent(regB.DeliveryID, regB.BridgeInstanceID, regB.RoutingKey, regB.DeliveryTarget, 1, DeliveryEventTypeStart, "other", false)); err != nil {
		t.Fatalf("Deliver(regB start) error = %v", err)
	}
	if err := broker.Deliver(ctx, testDeliveryEvent(regB.DeliveryID, regB.BridgeInstanceID, regB.RoutingKey, regB.DeliveryTarget, 2, DeliveryEventTypeFinal, "other done", true)); err != nil {
		t.Fatalf("Deliver(regB final) error = %v", err)
	}

	if err := broker.FailSession(ctx, regA.SessionID, "adapter stopped"); !errors.Is(err, ErrDeliveryQueueSaturated) {
		t.Fatalf("FailSession() error = %v, want %v", err, ErrDeliveryQueueSaturated)
	}

	snapshot, err := broker.Snapshot(ctx, regA.DeliveryID)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if got, want := snapshot.LatestSeq, int64(1); got != want {
		t.Fatalf("LatestSeq after rejected fail-session = %d, want %d", got, want)
	}
	if got, want := snapshot.LatestEventType, DeliveryEventTypeStart; got != want {
		t.Fatalf("LatestEventType after rejected fail-session = %q, want %q", got, want)
	}
	if snapshot.Final {
		t.Fatal("Final after rejected fail-session = true, want false")
	}
	if snapshot.Error != "" {
		t.Fatalf("Error after rejected fail-session = %q, want empty", snapshot.Error)
	}

	close(releaseStart)
}

func TestDeliveryValidationAndMetadataHelpers(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, time.April, 11, 12, 2, 0, 0, time.UTC)
	routingKey := testRoutingKey("brg-validate", "peer-validate")
	target := DeliveryTarget{
		BridgeInstanceID: "brg-validate",
		PeerID:           "peer-validate",
		Mode:             DeliveryModeReply,
	}
	newSnapshot := func() DeliverySnapshot {
		return DeliverySnapshot{
			DeliveryID:       "del-validate",
			SessionID:        "sess-validate",
			TurnID:           "turn-validate",
			BridgeInstanceID: "brg-validate",
			RoutingKey:       routingKey,
			DeliveryTarget:   target,
			LatestSeq:        2,
			LatestEventType:  DeliveryEventTypeDelta,
			CurrentContent:   MessageContent{Text: "hello"},
			LastSentSeq:      1,
			LastAckedSeq:     1,
			UpdatedAt:        updatedAt,
		}
	}
	newResumeRequest := func(snapshot DeliverySnapshot) DeliveryRequest {
		return DeliveryRequest{
			Event: DeliveryEvent{
				DeliveryID:       snapshot.DeliveryID,
				BridgeInstanceID: snapshot.BridgeInstanceID,
				RoutingKey:       routingKey,
				DeliveryTarget:   target,
				Seq:              snapshot.LatestSeq,
				EventType:        DeliveryEventTypeResume,
				Content:          snapshot.CurrentContent,
				Metadata:         deliveryMetadataJSON(map[string]string{"latest_event_type": DeliveryEventTypeDelta}),
			},
			Snapshot: &snapshot,
		}
	}

	t.Run("ShouldValidateDeliverySnapshots", func(t *testing.T) {
		t.Run("ShouldAcceptValidSnapshot", func(t *testing.T) {
			snapshot := newSnapshot()
			if err := snapshot.Validate(); err != nil {
				t.Fatalf("valid DeliverySnapshot.Validate() error = %v", err)
			}
		})

		t.Run("ShouldRejectSnapshotWhenLastAckedExceedsLastSent", func(t *testing.T) {
			snapshot := newSnapshot()
			snapshot.LastAckedSeq = 3
			err := snapshot.Validate()
			if err == nil || !strings.Contains(err.Error(), "last acked sequence cannot exceed last sent sequence") {
				t.Fatalf("snapshot.Validate() error = %v, want lastAckedSeq invariant", err)
			}
		})
	})

	t.Run("ShouldValidateResumeRequests", func(t *testing.T) {
		t.Run("ShouldAcceptMatchingSnapshot", func(t *testing.T) {
			snapshot := newSnapshot()
			if err := newResumeRequest(snapshot).Validate(); err != nil {
				t.Fatalf("valid DeliveryRequest.Validate() error = %v", err)
			}
		})

		t.Run("ShouldRejectMissingSnapshot", func(t *testing.T) {
			snapshot := newSnapshot()
			req := newResumeRequest(snapshot)
			req.Snapshot = nil
			err := req.Validate()
			if err == nil || !strings.Contains(err.Error(), "resume delivery request requires a snapshot") {
				t.Fatalf("req.Validate() error = %v, want missing snapshot validation", err)
			}
		})

		t.Run("ShouldRejectSnapshotsForNonResumeEvents", func(t *testing.T) {
			snapshot := newSnapshot()
			req := newResumeRequest(snapshot)
			req.Event.EventType = DeliveryEventTypeStart
			err := req.Validate()
			if err == nil || !strings.Contains(err.Error(), "only resume delivery requests may include a snapshot") {
				t.Fatalf("req.Validate() error = %v, want non-resume snapshot validation", err)
			}
		})

		t.Run("ShouldRejectMismatchedSnapshotDeliveryID", func(t *testing.T) {
			snapshot := newSnapshot()
			req := newResumeRequest(snapshot)
			mismatched := snapshot
			mismatched.DeliveryID = "del-other"
			req.Snapshot = &mismatched
			err := req.Validate()
			if err == nil || !strings.Contains(err.Error(), "snapshot must match event delivery id") {
				t.Fatalf("req.Validate() error = %v, want delivery id mismatch validation", err)
			}
		})
	})

	t.Run("ShouldNormalizeProjectionEventFields", func(t *testing.T) {
		normalized := (DeliveryProjectionEvent{
			Type:        " agent_message ",
			TurnID:      " turn-validate ",
			Error:       " boom ",
			Fingerprint: " fp-1 ",
		}).normalize()
		if got, want := normalized.Type, "agent_message"; got != want {
			t.Fatalf("normalized.Type = %q, want %q", got, want)
		}
		if got, want := normalized.TurnID, "turn-validate"; got != want {
			t.Fatalf("normalized.TurnID = %q, want %q", got, want)
		}
		if got, want := normalized.Error, "boom"; got != want {
			t.Fatalf("normalized.Error = %q, want %q", got, want)
		}
		if got, want := normalized.Fingerprint, "fp-1"; got != want {
			t.Fatalf("normalized.Fingerprint = %q, want %q", got, want)
		}
	})

	t.Run("ShouldEncodeAndDecodeDeliveryMetadata", func(t *testing.T) {
		metadata := deliveryMetadataJSON(map[string]string{"error": "broken"})
		if len(metadata) == 0 {
			t.Fatal("deliveryMetadataJSON(map) = empty, want JSON payload")
		}
		var decoded map[string]string
		if err := json.Unmarshal(metadata, &decoded); err != nil {
			t.Fatalf("json.Unmarshal(metadata) error = %v", err)
		}
		if got, want := decoded["error"], "broken"; got != want {
			t.Fatalf("decoded error = %q, want %q", got, want)
		}
		if got := deliveryMetadataJSON(func() {}); got != nil {
			t.Fatalf("deliveryMetadataJSON(func) = %q, want nil", string(got))
		}
		if got := deliveryErrorText([]byte("{")); got != "" {
			t.Fatalf("deliveryErrorText(invalid json) = %q, want empty string", got)
		}
	})
}

func TestBrokerProjectEventLockedCoversTerminalAndIgnoredPaths(t *testing.T) {
	t.Parallel()

	broker := NewBroker(nil, WithDeliveryBrokerNow(func() time.Time {
		return time.Date(2026, time.April, 11, 12, 4, 0, 0, time.UTC)
	}))
	t.Cleanup(broker.Close)

	delivery := &activeDelivery{
		deliveryID:       "del-locked",
		bridgeInstanceID: "brg-locked",
		routingKey:       testRoutingKey("brg-locked", "peer-locked"),
		target: DeliveryTarget{
			BridgeInstanceID: "brg-locked",
			PeerID:           "peer-locked",
			Mode:             DeliveryModeReply,
		},
	}

	if _, ok, err := broker.projectEventLocked(nil, DeliveryProjectionEvent{Type: "agent_message"}); !errors.Is(err, ErrDeliveryNotFound) || ok {
		t.Fatalf("projectEventLocked(nil) = (%v, %v), want ErrDeliveryNotFound and ok=false", err, ok)
	}

	if _, ok, err := broker.projectEventLocked(delivery, DeliveryProjectionEvent{Type: "done"}); err != nil || ok {
		t.Fatalf("projectEventLocked(done without content) = (%v, %v), want nil and ok=false", err, ok)
	}

	start, ok, err := broker.projectEventLocked(delivery, DeliveryProjectionEvent{Type: "agent_message", Text: "hello"})
	if err != nil || !ok {
		t.Fatalf("projectEventLocked(agent_message) = (%v, %v), want nil and ok=true", err, ok)
	}
	if got, want := start.EventType, DeliveryEventTypeStart; got != want {
		t.Fatalf("start event type = %q, want %q", got, want)
	}
	broker.applyQueuedEventLocked(delivery, start)

	final, ok, err := broker.projectEventLocked(delivery, DeliveryProjectionEvent{Type: "done"})
	if err != nil || !ok {
		t.Fatalf("projectEventLocked(done) = (%v, %v), want nil and ok=true", err, ok)
	}
	if got, want := final.EventType, DeliveryEventTypeFinal; got != want {
		t.Fatalf("final event type = %q, want %q", got, want)
	}

	errorDelivery := &activeDelivery{
		deliveryID:       "del-error",
		bridgeInstanceID: "brg-locked",
		routingKey:       testRoutingKey("brg-locked", "peer-locked"),
		target: DeliveryTarget{
			BridgeInstanceID: "brg-locked",
			PeerID:           "peer-locked",
			Mode:             DeliveryModeReply,
		},
		currentContent: MessageContent{Text: "partial"},
	}
	errorEvent, ok, err := broker.projectEventLocked(errorDelivery, DeliveryProjectionEvent{Type: "error", Error: "boom"})
	if err != nil || !ok {
		t.Fatalf("projectEventLocked(error) = (%v, %v), want nil and ok=true", err, ok)
	}
	if got, want := errorEvent.EventType, DeliveryEventTypeError; got != want {
		t.Fatalf("error event type = %q, want %q", got, want)
	}
	if got, want := deliveryErrorText(errorEvent.Metadata), "boom"; got != want {
		t.Fatalf("deliveryErrorText(errorEvent.Metadata) = %q, want %q", got, want)
	}

	if _, ok, err := broker.projectEventLocked(delivery, DeliveryProjectionEvent{Type: "unknown"}); err != nil || ok {
		t.Fatalf("projectEventLocked(unknown) = (%v, %v), want nil and ok=false", err, ok)
	}
}

func TestBrokerEnqueueEventLockedCoversReplacementAndSaturation(t *testing.T) {
	t.Parallel()

	broker := NewBroker(nil, WithDeliveryBrokerQueueCapacity(2))
	t.Cleanup(broker.Close)

	route := &routeWorker{bridgeInstanceID: "brg-queue"}
	delivery := &activeDelivery{deliveryID: "del-queue"}

	start := testDeliveryEvent(
		delivery.deliveryID,
		"brg-queue",
		testRoutingKey("brg-queue", "peer-queue"),
		DeliveryTarget{
			BridgeInstanceID: "brg-queue",
			PeerID:           "peer-queue",
			Mode:             DeliveryModeReply,
		},
		1,
		DeliveryEventTypeStart,
		"hello",
		false,
	)
	if err := broker.enqueueEventLocked(route, delivery, start); err != nil {
		t.Fatalf("enqueueEventLocked(start) error = %v", err)
	}

	startUpdate := start
	startUpdate.Seq = 2
	startUpdate.Content.Text = "hello again"
	if err := broker.enqueueEventLocked(route, delivery, startUpdate); err != nil {
		t.Fatalf("enqueueEventLocked(start update) error = %v", err)
	}
	if got, want := delivery.pendingStart.Seq, int64(2); got != want {
		t.Fatalf("pendingStart.Seq = %d, want %d", got, want)
	}

	route.queue = append(route.queue, deliveryQueueItem{deliveryID: "other", kind: deliveryQueueKindStart})
	deltaPromoted := start
	deltaPromoted.Seq = 3
	deltaPromoted.EventType = DeliveryEventTypeDelta
	deltaPromoted.Content.Text = "hello promoted"
	if err := broker.enqueueEventLocked(route, delivery, deltaPromoted); err != nil {
		t.Fatalf("enqueueEventLocked(delta promoted into start) error = %v", err)
	}
	if got, want := delivery.pendingStart.EventType, DeliveryEventTypeStart; got != want {
		t.Fatalf("pendingStart.EventType = %q, want %q", got, want)
	}
	if got, want := delivery.pendingStart.Content.Text, "hello promoted"; got != want {
		t.Fatalf("pendingStart.Content.Text = %q, want %q", got, want)
	}

	route.queue = route.queue[:1]
	delivery.startDelivered = true
	delta := deltaPromoted
	if err := broker.enqueueEventLocked(route, delivery, delta); err != nil {
		t.Fatalf("enqueueEventLocked(delta) error = %v", err)
	}
	deltaUpdate := delta
	deltaUpdate.Seq = 4
	deltaUpdate.Content.Text = "hello final delta"
	if err := broker.enqueueEventLocked(route, delivery, deltaUpdate); err != nil {
		t.Fatalf("enqueueEventLocked(delta update) error = %v", err)
	}
	if got, want := delivery.pendingDelta.Seq, int64(4); got != want {
		t.Fatalf("pendingDelta.Seq = %d, want %d", got, want)
	}

	final := deltaUpdate
	final.Seq = 5
	final.EventType = DeliveryEventTypeFinal
	final.Final = true
	final.Content.Text = "done"
	if err := broker.enqueueEventLocked(route, delivery, final); err != nil {
		t.Fatalf("enqueueEventLocked(final) error = %v", err)
	}
	if delivery.pendingDelta != nil {
		t.Fatal("pendingDelta = non-nil after final enqueue, want cleared delta")
	}
	if delivery.pendingTerminal == nil {
		t.Fatal("pendingTerminal = nil, want queued terminal event")
	}

	fullRoute := &routeWorker{
		bridgeInstanceID: "brg-queue",
		queue: []deliveryQueueItem{
			{deliveryID: "del-a", kind: deliveryQueueKindStart},
			{deliveryID: "del-b", kind: deliveryQueueKindStart},
		},
	}
	if err := broker.enqueueEventLocked(fullRoute, &activeDelivery{deliveryID: "del-c"}, start); !errors.Is(err, ErrDeliveryQueueSaturated) {
		t.Fatalf("enqueueEventLocked(full route) error = %v, want ErrDeliveryQueueSaturated", err)
	}

	metrics := broker.DeliveryMetrics()["brg-queue"]
	if got, want := metrics.DeliveryDroppedByReason["queue_saturated"], 1; got != want {
		t.Fatalf("DeliveryMetrics().DeliveryDroppedByReason[queue_saturated] = %d, want %d", got, want)
	}
}

func assertCallCountStable(t *testing.T, transport *fakeDeliveryTransport, want int, duration time.Duration) {
	t.Helper()

	timer := time.NewTimer(duration)
	defer timer.Stop()
	updates := transport.updateCh()
	for {
		if got := len(transport.snapshotCalls()); got != want {
			t.Fatalf("delivery call count = %d, want stable count %d", got, want)
		}
		select {
		case <-updates:
		case <-timer.C:
			return
		}
	}
}

func waitForSnapshot(
	t *testing.T,
	broker *Broker,
	deliveryID string,
	match func(*DeliverySnapshot) bool,
) *DeliverySnapshot {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()

	for {
		snapshot, err := broker.Snapshot(testutil.Context(t), deliveryID)
		if err == nil && match(snapshot) {
			return snapshot
		}
		select {
		case <-ticker.C:
		case <-timeout.C:
			t.Fatalf("snapshot %q did not reach expected state before timeout", deliveryID)
		}
	}
}
