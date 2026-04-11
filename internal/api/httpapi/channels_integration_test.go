//go:build integration

package httpapi

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/api/contract"
	channelspkg "github.com/pedronauck/agh/internal/channels"
	"github.com/pedronauck/agh/internal/testutil"
)

type blockingHTTPDeliveryTransport struct {
	releaseCh chan struct{}
}

func (t *blockingHTTPDeliveryTransport) DeliverChannel(ctx context.Context, _ string, req channelspkg.DeliveryRequest) (channelspkg.DeliveryAck, error) {
	if t != nil && req.Event.EventType == channelspkg.DeliveryEventTypeStart {
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

func TestHTTPChannelCreateReturnsPersistedPayload(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	resp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/channels"), []byte(`{"scope":"global","platform":"telegram","extension_name":"ext-telegram","display_name":"Support","enabled":true,"status":"starting","routing_policy":{"include_peer":true}}`), nil)
	if resp.StatusCode != http.StatusCreated {
		body := mustReadAll(t, resp.Body)
		t.Fatalf("create channel status = %d, want %d; body=%s", resp.StatusCode, http.StatusCreated, body)
	}

	var payload contract.ChannelResponse
	decodeHTTPJSON(t, resp, &payload)
	if payload.Channel.ID == "" || payload.Channel.Platform != "telegram" || payload.Channel.ExtensionName != "ext-telegram" {
		t.Fatalf("payload.Channel = %#v", payload.Channel)
	}

	stored, err := runtime.registry.GetChannelInstance(context.Background(), payload.Channel.ID)
	if err != nil {
		t.Fatalf("runtime.registry.GetChannelInstance() error = %v", err)
	}
	if stored.DisplayName != "Support" || stored.Status != channelspkg.ChannelStatusStarting {
		t.Fatalf("stored instance = %#v", stored)
	}
}

func TestHTTPChannelRoutesEndpointReturnsOnlyRequestedInstanceRoutes(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	first := createIntegrationChannel(t, runtime, channelspkg.CreateInstanceRequest{
		ID:            "chan-http-a",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "A",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})
	second := createIntegrationChannel(t, runtime, channelspkg.CreateInstanceRequest{
		ID:            "chan-http-b",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "B",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})

	upsertIntegrationChannelRoute(t, runtime, channelspkg.ChannelRoute{
		ChannelInstanceID: first.ID,
		Scope:             first.Scope,
		PeerID:            "peer-a",
		SessionID:         "sess-a",
		AgentName:         "coder",
		LastActivityAt:    time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
	})
	upsertIntegrationChannelRoute(t, runtime, channelspkg.ChannelRoute{
		ChannelInstanceID: second.ID,
		Scope:             second.Scope,
		PeerID:            "peer-b",
		SessionID:         "sess-b",
		AgentName:         "coder",
		LastActivityAt:    time.Date(2026, 4, 11, 12, 1, 0, 0, time.UTC),
	})

	resp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/channels/"+first.ID+"/routes"), nil, nil)
	if resp.StatusCode != http.StatusOK {
		body := mustReadAll(t, resp.Body)
		t.Fatalf("routes status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, body)
	}

	var payload contract.ChannelRoutesResponse
	decodeHTTPJSON(t, resp, &payload)
	if got, want := len(payload.Routes), 1; got != want {
		t.Fatalf("len(routes) = %d, want %d", got, want)
	}
	if payload.Routes[0].ChannelInstanceID != first.ID || payload.Routes[0].PeerID != "peer-a" {
		t.Fatalf("routes = %#v", payload.Routes)
	}
}

func TestHTTPChannelTestDeliveryResolvesTargetWithoutLiveAdapter(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	instance := createIntegrationChannel(t, runtime, channelspkg.CreateInstanceRequest{
		ID:               "chan-http-test-delivery",
		Scope:            channelspkg.ScopeGlobal,
		Platform:         "telegram",
		ExtensionName:    "ext-telegram",
		DisplayName:      "Test Delivery",
		Enabled:          true,
		Status:           channelspkg.ChannelStatusReady,
		RoutingPolicy:    channelspkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
		DeliveryDefaults: []byte(`{"peer_id":"peer-default","mode":"reply"}`),
	})

	resp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/channels/"+instance.ID+"/test-delivery"), []byte(`{"message":"hello","target":{"thread_id":"thread-1"}}`), nil)
	if resp.StatusCode != http.StatusOK {
		body := mustReadAll(t, resp.Body)
		t.Fatalf("test delivery status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, body)
	}

	var payload contract.ChannelTestDeliveryResponse
	decodeHTTPJSON(t, resp, &payload)
	if payload.Status != "resolved" || payload.DeliveryTarget.ChannelInstanceID != instance.ID {
		t.Fatalf("payload = %#v", payload)
	}
	if payload.DeliveryTarget.PeerID != "peer-default" || payload.DeliveryTarget.ThreadID != "thread-1" || payload.DeliveryTarget.Mode != channelspkg.DeliveryModeReply {
		t.Fatalf("delivery target = %#v", payload.DeliveryTarget)
	}
}

func TestHTTPObserveHealthIncludesChannelMetricsAndPreservesSessionFields(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	createSessionResp := mustHTTPRequest(t, runtime.client, http.MethodPost, mustURL(runtime.host, runtime.port, "/api/sessions"), []byte(`{"agent_name":"coder","workspace_path":"`+runtime.workspace+`"}`), nil)
	if createSessionResp.StatusCode != http.StatusCreated {
		body := mustReadAll(t, createSessionResp.Body)
		t.Fatalf("create session status = %d, want %d; body=%s", createSessionResp.StatusCode, http.StatusCreated, body)
	}

	instance := createIntegrationChannel(t, runtime, channelspkg.CreateInstanceRequest{
		ID:            "chan-http-health",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Health",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})
	upsertIntegrationChannelRoute(t, runtime, channelspkg.ChannelRoute{
		ChannelInstanceID: instance.ID,
		Scope:             instance.Scope,
		PeerID:            "peer-health",
		SessionID:         "sess-health",
		AgentName:         "coder",
		LastActivityAt:    time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC),
	})
	runtime.observer.RecordChannelAuthFailure(instance.ID)

	resp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/observe/health"), nil, nil)
	if resp.StatusCode != http.StatusOK {
		body := mustReadAll(t, resp.Body)
		t.Fatalf("health status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, body)
	}

	var payload contract.HealthResponse
	decodeHTTPJSON(t, resp, &payload)
	if got, want := payload.Health.Status, "ok"; got != want {
		t.Fatalf("health.status = %q, want %q", got, want)
	}
	if got, want := payload.Health.ActiveSessions, 1; got != want {
		t.Fatalf("health.active_sessions = %d, want %d", got, want)
	}
	if got, want := payload.Health.Channels.TotalInstances, 1; got != want {
		t.Fatalf("health.channels.total_instances = %d, want %d", got, want)
	}
	if got, want := payload.Health.Channels.RouteCount, 1; got != want {
		t.Fatalf("health.channels.route_count = %d, want %d", got, want)
	}
	if got, want := payload.Health.Channels.StatusCounts.Ready, 1; got != want {
		t.Fatalf("health.channels.status_counts.ready = %d, want %d", got, want)
	}
	if got, want := payload.Health.Channels.AuthFailuresTotal, 1; got != want {
		t.Fatalf("health.channels.auth_failures_total = %d, want %d", got, want)
	}
}

func TestHTTPChannelDetailShowsAuthRequiredStatusAndHealth(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	instance := createIntegrationChannel(t, runtime, channelspkg.CreateInstanceRequest{
		ID:            "chan-http-auth",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Auth",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})
	if _, err := runtime.channels.UpdateInstanceState(testutil.Context(t), channelspkg.UpdateInstanceStateRequest{
		ID:      instance.ID,
		Enabled: true,
		Status:  channelspkg.ChannelStatusAuthRequired,
	}); err != nil {
		t.Fatalf("runtime.channels.UpdateInstanceState() error = %v", err)
	}
	runtime.observer.RecordChannelAuthFailure(instance.ID)

	resp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/channels/"+instance.ID), nil, nil)
	if resp.StatusCode != http.StatusOK {
		body := mustReadAll(t, resp.Body)
		t.Fatalf("get channel status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, body)
	}

	var payload contract.ChannelResponse
	decodeHTTPJSON(t, resp, &payload)
	if got, want := payload.Channel.Status, channelspkg.ChannelStatusAuthRequired; got != want {
		t.Fatalf("channel.status = %q, want %q", got, want)
	}
	if got, want := payload.Health.Status, channelspkg.ChannelStatusAuthRequired; got != want {
		t.Fatalf("health.status = %q, want %q", got, want)
	}
	if got, want := payload.Health.AuthFailuresTotal, 1; got != want {
		t.Fatalf("health.auth_failures_total = %d, want %d", got, want)
	}
}

func TestHTTPChannelDetailReportsBacklogAndClearsAfterDeliveryCompletes(t *testing.T) {
	runtime := newIntegrationRuntime(t)

	instance := createIntegrationChannel(t, runtime, channelspkg.CreateInstanceRequest{
		ID:            "chan-http-backlog",
		Scope:         channelspkg.ScopeGlobal,
		Platform:      "telegram",
		ExtensionName: "ext-telegram",
		DisplayName:   "Backlog",
		Enabled:       true,
		Status:        channelspkg.ChannelStatusReady,
		RoutingPolicy: channelspkg.RoutingPolicy{IncludePeer: true},
	})

	transport := &blockingHTTPDeliveryTransport{releaseCh: make(chan struct{})}
	runtime.channels.Broker().SetTransport(transport)
	registration := registerIntegrationDelivery(t, runtime, instance, "sess-http-backlog", "turn-http-backlog", "peer-http-backlog")
	if err := runtime.channels.Broker().Deliver(testutil.Context(t), integrationDeliveryEvent(registration, 1, channelspkg.DeliveryEventTypeStart, "hello", false)); err != nil {
		t.Fatalf("Broker().Deliver(start) error = %v", err)
	}
	if err := runtime.channels.Broker().Deliver(testutil.Context(t), integrationDeliveryEvent(registration, 2, channelspkg.DeliveryEventTypeDelta, "hello again", false)); err != nil {
		t.Fatalf("Broker().Deliver(delta) error = %v", err)
	}

	waitForHTTPCondition(t, func() bool {
		channel := getHTTPChannel(t, runtime, instance.ID)
		return channel.Health.DeliveryBacklog == 1
	})

	channel := getHTTPChannel(t, runtime, instance.ID)
	if got, want := channel.Health.DeliveryBacklog, 1; got != want {
		t.Fatalf("channel.health.delivery_backlog = %d, want %d", got, want)
	}
	health := getHTTPHealth(t, runtime)
	if got, want := health.Health.Channels.DeliveryBacklog, 1; got != want {
		t.Fatalf("health.channels.delivery_backlog = %d, want %d", got, want)
	}

	close(transport.releaseCh)
	waitForHTTPCondition(t, func() bool {
		return getHTTPChannel(t, runtime, instance.ID).Health.DeliveryBacklog == 0
	})

	channel = getHTTPChannel(t, runtime, instance.ID)
	if channel.Health.DeliveryBacklog != 0 {
		t.Fatalf("channel.health.delivery_backlog = %d, want 0", channel.Health.DeliveryBacklog)
	}
}

func createIntegrationChannel(t *testing.T, runtime integrationRuntime, req channelspkg.CreateInstanceRequest) *channelspkg.ChannelInstance {
	t.Helper()

	instance, err := runtime.channels.CreateInstance(testutil.Context(t), req)
	if err != nil {
		t.Fatalf("runtime.channels.CreateInstance() error = %v", err)
	}
	return instance
}

func upsertIntegrationChannelRoute(t *testing.T, runtime integrationRuntime, route channelspkg.ChannelRoute) {
	t.Helper()

	if _, err := runtime.channels.UpsertRoute(testutil.Context(t), route); err != nil {
		t.Fatalf("runtime.channels.UpsertRoute() error = %v", err)
	}
}

func registerIntegrationDelivery(t *testing.T, runtime integrationRuntime, instance *channelspkg.ChannelInstance, sessionID string, turnID string, peerID string) channelspkg.DeliverySnapshot {
	t.Helper()

	snapshot, err := runtime.channels.Broker().RegisterPromptDelivery(testutil.Context(t), channelspkg.PromptDeliveryRegistration{
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

func integrationDeliveryEvent(snapshot channelspkg.DeliverySnapshot, seq int64, eventType string, text string, final bool) channelspkg.DeliveryEvent {
	return channelspkg.DeliveryEvent{
		DeliveryID:        snapshot.DeliveryID,
		ChannelInstanceID: snapshot.ChannelInstanceID,
		RoutingKey:        snapshot.RoutingKey,
		DeliveryTarget:    snapshot.DeliveryTarget,
		Seq:               seq,
		EventType:         eventType,
		Content:           channelspkg.MessageContent{Text: text},
		Final:             final,
	}
}

func getHTTPChannel(t *testing.T, runtime integrationRuntime, channelID string) contract.ChannelResponse {
	t.Helper()

	resp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/channels/"+channelID), nil, nil)
	if resp.StatusCode != http.StatusOK {
		body := mustReadAll(t, resp.Body)
		t.Fatalf("get channel status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, body)
	}
	var payload contract.ChannelResponse
	decodeHTTPJSON(t, resp, &payload)
	return payload
}

func getHTTPHealth(t *testing.T, runtime integrationRuntime) contract.HealthResponse {
	t.Helper()

	resp := mustHTTPRequest(t, runtime.client, http.MethodGet, mustURL(runtime.host, runtime.port, "/api/observe/health"), nil, nil)
	if resp.StatusCode != http.StatusOK {
		body := mustReadAll(t, resp.Body)
		t.Fatalf("get health status = %d, want %d; body=%s", resp.StatusCode, http.StatusOK, body)
	}
	var payload contract.HealthResponse
	decodeHTTPJSON(t, resp, &payload)
	return payload
}

func waitForHTTPCondition(t *testing.T, fn func() bool) {
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

func mustReadAll(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	defer func() {
		_ = body.Close()
	}()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	return string(data)
}
