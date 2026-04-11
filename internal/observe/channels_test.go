package observe

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	channelspkg "github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/testutil"
)

type blockingObserveTransport struct {
	releaseCh  chan struct{}
	blockStart bool
}

func (t *blockingObserveTransport) DeliverChannel(ctx context.Context, _ string, req channelspkg.DeliveryRequest) (channelspkg.DeliveryAck, error) {
	if t != nil && t.blockStart && req.Event.EventType == channelspkg.DeliveryEventTypeStart {
		select {
		case <-t.releaseCh:
		case <-ctx.Done():
			return channelspkg.DeliveryAck{}, ctx.Err()
		}
	}
	return channelspkg.DeliveryAck{
		DeliveryID: req.Event.DeliveryID,
		Seq:        req.Event.Seq,
	}, nil
}

func TestHealthIncludesChannelStatusCountsAndRouteSummary(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	starting := createObserveChannelInstance(t, h, "chan-starting", channelspkg.ChannelStatusStarting)
	ready := createObserveChannelInstance(t, h, "chan-ready", channelspkg.ChannelStatusReady)
	degraded := createObserveChannelInstance(t, h, "chan-degraded", channelspkg.ChannelStatusDegraded)
	authRequired := createObserveChannelInstance(t, h, "chan-auth", channelspkg.ChannelStatusAuthRequired)
	h.observer.RecordChannelAuthFailure(authRequired.ID)

	upsertObserveRoute(t, h, starting, "peer-starting", "sess-starting")
	upsertObserveRoute(t, h, ready, "peer-ready-a", "sess-ready-a")
	upsertObserveRoute(t, h, ready, "peer-ready-b", "sess-ready-b")
	upsertObserveRoute(t, h, degraded, "peer-degraded", "sess-degraded")

	health, err := h.observer.Health(testutil.Context(t))
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if got, want := health.Channels.TotalInstances, 4; got != want {
		t.Fatalf("Health().Channels.TotalInstances = %d, want %d", got, want)
	}
	if got, want := health.Channels.RouteCount, 4; got != want {
		t.Fatalf("Health().Channels.RouteCount = %d, want %d", got, want)
	}
	if got, want := health.Channels.StatusCounts.Starting, 1; got != want {
		t.Fatalf("Health().Channels.StatusCounts.Starting = %d, want %d", got, want)
	}
	if got, want := health.Channels.StatusCounts.Ready, 1; got != want {
		t.Fatalf("Health().Channels.StatusCounts.Ready = %d, want %d", got, want)
	}
	if got, want := health.Channels.StatusCounts.Degraded, 1; got != want {
		t.Fatalf("Health().Channels.StatusCounts.Degraded = %d, want %d", got, want)
	}
	if got, want := health.Channels.StatusCounts.AuthRequired, 1; got != want {
		t.Fatalf("Health().Channels.StatusCounts.AuthRequired = %d, want %d", got, want)
	}
	if got, want := health.Channels.AuthFailuresTotal, 1; got != want {
		t.Fatalf("Health().Channels.AuthFailuresTotal = %d, want %d", got, want)
	}

	observed := observeChannelHealthMap(t, h)
	if got, want := observed[ready.ID].RouteCount, 2; got != want {
		t.Fatalf("QueryChannelHealth(%s).RouteCount = %d, want %d", ready.ID, got, want)
	}
	if got, want := observed[authRequired.ID].Status, channelspkg.ChannelStatusAuthRequired; got != want {
		t.Fatalf("QueryChannelHealth(%s).Status = %q, want %q", authRequired.ID, got, want)
	}
}

func TestHealthTracksDeliveryBacklogWithoutChangingActiveSessions(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.source.sessions = []*session.SessionInfo{{
		ID:        "sess-live",
		AgentName: "coder",
		State:     session.StateActive,
	}}

	instance := createObserveChannelInstance(t, h, "chan-backlog", channelspkg.ChannelStatusReady)
	transport := &blockingObserveTransport{
		releaseCh:  make(chan struct{}),
		blockStart: true,
	}
	h.channels.broker.SetTransport(transport)

	registration := registerObserveDelivery(t, h, instance, "sess-live", "turn-live", "peer-backlog")
	if err := h.channels.broker.Deliver(testutil.Context(t), observeDeliveryEvent(registration, 1, channelspkg.DeliveryEventTypeStart, "hello", false)); err != nil {
		t.Fatalf("Deliver(start) error = %v", err)
	}
	if err := h.channels.broker.Deliver(testutil.Context(t), observeDeliveryEvent(registration, 2, channelspkg.DeliveryEventTypeDelta, "hello again", false)); err != nil {
		t.Fatalf("Deliver(delta) error = %v", err)
	}

	waitForObserveCondition(t, func() bool {
		health, err := h.observer.Health(testutil.Context(t))
		return err == nil && health.Channels.DeliveryBacklog == 1
	})

	health, err := h.observer.Health(testutil.Context(t))
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if got, want := health.ActiveSessions, 1; got != want {
		t.Fatalf("Health().ActiveSessions = %d, want %d", got, want)
	}
	if got, want := health.Channels.DeliveryBacklog, 1; got != want {
		t.Fatalf("Health().Channels.DeliveryBacklog = %d, want %d", got, want)
	}

	close(transport.releaseCh)
	waitForObserveCondition(t, func() bool {
		health, err := h.observer.Health(testutil.Context(t))
		return err == nil && health.Channels.DeliveryBacklog == 0
	})
}

func TestQueryChannelHealthSurfacesAuthAndTerminalDeliveryFailuresPerInstance(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	authInstance := createObserveChannelInstance(t, h, "chan-auth-failure", channelspkg.ChannelStatusAuthRequired)
	deliveryInstance := createObserveChannelInstance(t, h, "chan-delivery-failure", channelspkg.ChannelStatusReady)
	otherInstance := createObserveChannelInstance(t, h, "chan-other", channelspkg.ChannelStatusReady)
	h.observer.RecordChannelAuthFailure(authInstance.ID)

	registration := registerObserveDelivery(t, h, deliveryInstance, "sess-delivery", "turn-delivery", "peer-delivery")
	if err := h.channels.broker.Deliver(testutil.Context(t), observeDeliveryEvent(registration, 1, channelspkg.DeliveryEventTypeError, "boom", true)); err != nil {
		t.Fatalf("Deliver(error) error = %v", err)
	}

	observed := observeChannelHealthMap(t, h)
	if got, want := observed[authInstance.ID].AuthFailuresTotal, 1; got != want {
		t.Fatalf("auth instance AuthFailuresTotal = %d, want %d", got, want)
	}
	if got, want := observed[deliveryInstance.ID].DeliveryFailuresTotal, 1; got != want {
		t.Fatalf("delivery instance DeliveryFailuresTotal = %d, want %d", got, want)
	}
	if got, want := observed[deliveryInstance.ID].LastError, "boom"; got != want {
		t.Fatalf("delivery instance LastError = %q, want %q", got, want)
	}
	if observed[otherInstance.ID].AuthFailuresTotal != 0 || observed[otherInstance.ID].DeliveryFailuresTotal != 0 {
		t.Fatalf("other instance health = %#v, want zero auth/delivery failures", observed[otherInstance.ID])
	}
}

func TestChannelRuntimeOverridesAffectStatusCountsAndCanBeCleared(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	disabled := createObserveChannelInstance(t, h, "chan-disabled", channelspkg.ChannelStatusDisabled)
	ready := createObserveChannelInstance(t, h, "chan-runtime", channelspkg.ChannelStatusReady)

	h.observer.RecordChannelRuntimeIssue(disabled.ID, channelspkg.ChannelStatusError, "ignored for disabled")
	h.observer.RecordChannelRuntimeIssue(ready.ID, channelspkg.ChannelStatusError, "adapter crashed")

	health, err := h.observer.Health(testutil.Context(t))
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if got, want := health.Channels.StatusCounts.Disabled, 1; got != want {
		t.Fatalf("Health().Channels.StatusCounts.Disabled = %d, want %d", got, want)
	}
	if got, want := health.Channels.StatusCounts.Error, 1; got != want {
		t.Fatalf("Health().Channels.StatusCounts.Error = %d, want %d", got, want)
	}

	observed := observeChannelHealthMap(t, h)
	if got, want := observed[disabled.ID].Status, channelspkg.ChannelStatusDisabled; got != want {
		t.Fatalf("disabled instance Status = %q, want %q", got, want)
	}
	if got, want := observed[ready.ID].Status, channelspkg.ChannelStatusError; got != want {
		t.Fatalf("runtime instance Status = %q, want %q", got, want)
	}
	if got, want := observed[ready.ID].LastError, "adapter crashed"; got != want {
		t.Fatalf("runtime instance LastError = %q, want %q", got, want)
	}

	h.observer.ClearChannelRuntimeIssue(ready.ID)

	health, err = h.observer.Health(testutil.Context(t))
	if err != nil {
		t.Fatalf("Health() after clear error = %v", err)
	}
	if got, want := health.Channels.StatusCounts.Ready, 1; got != want {
		t.Fatalf("Health().Channels.StatusCounts.Ready = %d, want %d", got, want)
	}
	if got := health.Channels.StatusCounts.Error; got != 0 {
		t.Fatalf("Health().Channels.StatusCounts.Error = %d, want 0", got)
	}

	observed = observeChannelHealthMap(t, h)
	if got, want := observed[ready.ID].Status, channelspkg.ChannelStatusReady; got != want {
		t.Fatalf("runtime instance Status after clear = %q, want %q", got, want)
	}
	if got := observed[ready.ID].LastError; got != "" {
		t.Fatalf("runtime instance LastError after clear = %q, want empty", got)
	}
}

func createObserveChannelInstance(t *testing.T, h *harness, id string, status channelspkg.ChannelStatus) *channelspkg.ChannelInstance {
	t.Helper()

	instance, err := h.channels.CreateInstance(testutil.Context(t), channelspkg.CreateInstanceRequest{
		ID:            id,
		Scope:         channelspkg.ScopeWorkspace,
		WorkspaceID:   h.workspaceID,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   id,
		Enabled:       status != channelspkg.ChannelStatusDisabled,
		Status:        status,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})
	if err != nil {
		t.Fatalf("CreateInstance(%s) error = %v", id, err)
	}
	return instance
}

func upsertObserveRoute(t *testing.T, h *harness, instance *channelspkg.ChannelInstance, peerID string, sessionID string) {
	t.Helper()

	if _, err := h.channels.UpsertRoute(testutil.Context(t), channelspkg.ChannelRoute{
		ChannelInstanceID: instance.ID,
		Scope:             instance.Scope,
		WorkspaceID:       instance.WorkspaceID,
		PeerID:            peerID,
		SessionID:         sessionID,
		AgentName:         "coder",
		LastActivityAt:    h.now,
	}); err != nil {
		t.Fatalf("UpsertRoute(%s) error = %v", peerID, err)
	}
}

func registerObserveDelivery(t *testing.T, h *harness, instance *channelspkg.ChannelInstance, sessionID string, turnID string, peerID string) channelspkg.DeliverySnapshot {
	t.Helper()

	snapshot, err := h.channels.broker.RegisterPromptDelivery(testutil.Context(t), channelspkg.PromptDeliveryRegistration{
		SessionID:     sessionID,
		TurnID:        turnID,
		ExtensionName: instance.ExtensionName,
		RoutingKey: channelspkg.RoutingKey{
			Scope:             instance.Scope,
			WorkspaceID:       instance.WorkspaceID,
			ChannelInstanceID: instance.ID,
			PeerID:            peerID,
		},
		DeliveryTarget: channelspkg.DeliveryTarget{
			ChannelInstanceID: instance.ID,
			PeerID:            peerID,
			Mode:              channelspkg.DeliveryModeReply,
		},
	})
	if err != nil {
		t.Fatalf("RegisterPromptDelivery(%s) error = %v", instance.ID, err)
	}
	return *snapshot
}

func observeDeliveryEvent(snapshot channelspkg.DeliverySnapshot, seq int64, eventType string, text string, final bool) channelspkg.DeliveryEvent {
	var metadata json.RawMessage
	if eventType == channelspkg.DeliveryEventTypeError {
		data, err := json.Marshal(map[string]string{"error": text})
		if err == nil {
			metadata = json.RawMessage(data)
		}
	}
	return channelspkg.DeliveryEvent{
		DeliveryID:        snapshot.DeliveryID,
		ChannelInstanceID: snapshot.ChannelInstanceID,
		RoutingKey:        snapshot.RoutingKey,
		DeliveryTarget:    snapshot.DeliveryTarget,
		Seq:               seq,
		EventType:         eventType,
		Content:           channelspkg.MessageContent{Text: text},
		Final:             final,
		Metadata:          metadata,
	}
}

func observeChannelHealthMap(t *testing.T, h *harness) map[string]ChannelInstanceHealth {
	t.Helper()

	rows, err := h.observer.QueryChannelHealth(testutil.Context(t))
	if err != nil {
		t.Fatalf("QueryChannelHealth() error = %v", err)
	}

	health := make(map[string]ChannelInstanceHealth, len(rows))
	for _, row := range rows {
		health[row.ChannelInstanceID] = row
	}
	return health
}

func waitForObserveCondition(t *testing.T, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition did not become true before timeout")
}
