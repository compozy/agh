package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/subprocess"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestMapTelegramUpdateDirectAndForumRouting(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-telegram")

	direct, err := mapTelegramUpdate(telegramUpdate{
		UpdateID: 10,
		Message: &telegramMessage{
			MessageID:       111,
			MessageThreadID: 77,
			Date:            now.Unix(),
			Chat: telegramChat{
				ID:   42,
				Type: "private",
			},
			From: telegramUser{
				ID:        7,
				Username:  "alice",
				FirstName: "Alice",
				LastName:  "Example",
			},
			Text: " hello ",
		},
	}, managed, now)
	if err != nil {
		t.Fatalf("mapTelegramUpdate(direct) error = %v", err)
	}
	if got, want := direct.PeerID, "42"; got != want {
		t.Fatalf("direct.PeerID = %q, want %q", got, want)
	}
	if got := direct.GroupID; got != "" {
		t.Fatalf("direct.GroupID = %q, want empty", got)
	}
	if got, want := direct.ThreadID, "77"; got != want {
		t.Fatalf("direct.ThreadID = %q, want %q", got, want)
	}

	forum, err := mapTelegramUpdate(telegramUpdate{
		UpdateID: 11,
		Message: &telegramMessage{
			MessageID: 222,
			Date:      now.Unix(),
			Chat: telegramChat{
				ID:      -100123,
				Type:    "supergroup",
				IsForum: true,
				Title:   "ops",
			},
			From: telegramUser{
				ID:       8,
				Username: "bob",
			},
			Caption: "  from forum  ",
		},
	}, managed, now)
	if err != nil {
		t.Fatalf("mapTelegramUpdate(forum) error = %v", err)
	}
	if got := forum.PeerID; got != "" {
		t.Fatalf("forum.PeerID = %q, want empty", got)
	}
	if got, want := forum.GroupID, "-100123"; got != want {
		t.Fatalf("forum.GroupID = %q, want %q", got, want)
	}
	if got, want := forum.ThreadID, telegramGeneralTopicID; got != want {
		t.Fatalf("forum.ThreadID = %q, want %q", got, want)
	}
	if got, want := forum.Content.Text, "from forum"; got != want {
		t.Fatalf("forum.Content.Text = %q, want %q", got, want)
	}
}

func TestAllowDirectMessagePolicies(t *testing.T) {
	t.Parallel()

	message := telegramMessage{
		Chat: telegramChat{Type: "private"},
		From: telegramUser{ID: 42, Username: "alice"},
	}

	if !allowDirectMessage(
		resolvedInstanceConfig{dmPolicy: bridgepkg.BridgeDMPolicyOpen},
		message,
	) {
		t.Fatal("allowDirectMessage(open) = false, want true")
	}

	allowlist := resolvedInstanceConfig{
		dmPolicy:       bridgepkg.BridgeDMPolicyAllowlist,
		allowUserIDs:   map[string]struct{}{"42": {}},
		allowUsernames: map[string]struct{}{"bob": {}},
	}
	if !allowDirectMessage(allowlist, message) {
		t.Fatal("allowDirectMessage(allowlist by id) = false, want true")
	}

	pairing := resolvedInstanceConfig{
		dmPolicy:        bridgepkg.BridgeDMPolicyPairing,
		pairedUsernames: map[string]struct{}{"alice": {}},
	}
	if !allowDirectMessage(pairing, message) {
		t.Fatal("allowDirectMessage(pairing by username) = false, want true")
	}

	rejected := resolvedInstanceConfig{
		dmPolicy: bridgepkg.BridgeDMPolicyAllowlist,
	}
	if allowDirectMessage(rejected, message) {
		t.Fatal("allowDirectMessage(rejected) = true, want false")
	}
}

func TestExecuteDeliveryPostEditDeleteAndResume(t *testing.T) {
	t.Parallel()

	api := &fakeTelegramAPI{nextMessageID: 500}
	cfg := resolvedInstanceConfig{instanceID: "brg-1"}

	startReq := testDeliveryRequest(
		"brg-1",
		"delivery-1",
		1,
		bridgepkg.DeliveryEventTypeStart,
		false,
	)
	startAck, state, err := executeDelivery(
		context.Background(),
		api,
		cfg,
		startReq,
		deliveryState{},
	)
	if err != nil {
		t.Fatalf("executeDelivery(start) error = %v", err)
	}
	if got, want := startAck.RemoteMessageID, "peer-1:500"; got != want {
		t.Fatalf("startAck.RemoteMessageID = %q, want %q", got, want)
	}

	finalReq := testDeliveryRequest(
		"brg-1",
		"delivery-1",
		2,
		bridgepkg.DeliveryEventTypeFinal,
		true,
	)
	finalReq.Event.Content.Text = "hello world"
	finalAck, state, err := executeDelivery(context.Background(), api, cfg, finalReq, state)
	if err != nil {
		t.Fatalf("executeDelivery(final) error = %v", err)
	}
	if got, want := finalAck.ReplaceRemoteMessageID, startAck.RemoteMessageID; got != want {
		t.Fatalf("finalAck.ReplaceRemoteMessageID = %q, want %q", got, want)
	}

	finalNoOpReq := testDeliveryRequest(
		"brg-1",
		"delivery-1",
		3,
		bridgepkg.DeliveryEventTypeFinal,
		true,
	)
	finalNoOpReq.Event.Content.Text = "hello world"
	finalNoOpAck, state, err := executeDelivery(context.Background(), api, cfg, finalNoOpReq, state)
	if err != nil {
		t.Fatalf("executeDelivery(final no-op) error = %v", err)
	}
	if got, want := finalNoOpAck.RemoteMessageID, finalAck.RemoteMessageID; got != want {
		t.Fatalf("finalNoOpAck.RemoteMessageID = %q, want %q", got, want)
	}
	if got, want := finalNoOpAck.ReplaceRemoteMessageID, finalAck.RemoteMessageID; got != want {
		t.Fatalf("finalNoOpAck.ReplaceRemoteMessageID = %q, want %q", got, want)
	}

	deleteReq := testDeleteRequest("brg-1", "delivery-1", 4, finalNoOpAck.RemoteMessageID)
	deleteAck, _, err := executeDelivery(context.Background(), api, cfg, deleteReq, state)
	if err != nil {
		t.Fatalf("executeDelivery(delete) error = %v", err)
	}
	if got, want := deleteAck.RemoteMessageID, finalNoOpAck.RemoteMessageID; got != want {
		t.Fatalf("deleteAck.RemoteMessageID = %q, want %q", got, want)
	}
	if got, want := strings.Join(api.methods, ","), "sendMessage,editMessageText,deleteMessage"; got != want {
		t.Fatalf("api methods = %q, want %q", got, want)
	}

	resumeAPI := &fakeTelegramAPI{nextMessageID: 900}
	resumeReq := testDeliveryRequest(
		"brg-1",
		"delivery-2",
		1,
		bridgepkg.DeliveryEventTypeResume,
		false,
	)
	resumeReq.Event.Resume = &bridgepkg.DeliveryResumeState{
		LatestEventType: bridgepkg.DeliveryEventTypeFinal,
	}
	resumeReq.Snapshot = &bridgepkg.DeliverySnapshot{
		DeliveryID:       "delivery-2",
		SessionID:        "sess-1",
		TurnID:           "turn-1",
		BridgeInstanceID: "brg-1",
		RoutingKey:       resumeReq.Event.RoutingKey,
		DeliveryTarget:   resumeReq.Event.DeliveryTarget,
		LatestSeq:        1,
		LatestEventType:  bridgepkg.DeliveryEventTypeFinal,
		CurrentContent:   bridgepkg.MessageContent{Text: "hello"},
		Final:            true,
		UpdatedAt:        time.Date(2026, 4, 15, 12, 5, 0, 0, time.UTC),
	}
	resumeAck, _, err := executeDelivery(
		context.Background(),
		resumeAPI,
		cfg,
		resumeReq,
		deliveryState{},
	)
	if err != nil {
		t.Fatalf("executeDelivery(resume without remote) error = %v", err)
	}
	if got, want := resumeAck.RemoteMessageID, "peer-1:900"; got != want {
		t.Fatalf("resumeAck.RemoteMessageID = %q, want %q", got, want)
	}
}

func TestVerifyWebhookSecret(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://example.com/telegram/brg-1",
		strings.NewReader(`{}`),
	)
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "secret")

	if err := verifyWebhookSecret(context.Background(), req, nil, "secret"); err != nil {
		t.Fatalf("verifyWebhookSecret(valid) error = %v", err)
	}
	if err := verifyWebhookSecret(context.Background(), req, nil, "wrong"); err == nil {
		t.Fatal("verifyWebhookSecret(invalid) error = nil, want non-nil")
	}
	if err := verifyWebhookSecret(context.Background(), req, nil, ""); err == nil {
		t.Fatal("verifyWebhookSecret(missing configured secret) error = nil, want non-nil")
	}
}

func TestRuntimeInitializeStartsWebhookServerAndWritesMarkers(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newTelegramAPIServer(t)
	t.Setenv(telegramListenAddrEnv, listenAddr)
	t.Setenv(telegramAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 0, 0, 0, time.UTC)
	managed := []subprocess.InitializeBridgeManagedInstance{
		testBridgeRuntime(now, "brg-1"),
		testBridgeRuntime(now, "brg-2"),
	}
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesList),
		func(context.Context, json.RawMessage) (any, error) {
			return []bridgepkg.BridgeInstance{managed[0].Instance, managed[1].Instance}, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgeInstanceTargetParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			switch payload.BridgeInstanceID {
			case "brg-1":
				return managed[0].Instance, nil
			case "brg-2":
				return managed[1].Instance, nil
			default:
				return nil, errors.New("unexpected instance")
			}
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgesInstancesReportStateParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			instance := managed[0].Instance
			if payload.BridgeInstanceID == "brg-2" {
				instance = managed[1].Instance
			}
			instance.Status = payload.Status
			instance.Degradation = payload.Degradation
			return instance, nil
		},
	)

	if err := hostPeer.Call(
		context.Background(),
		"initialize",
		testInitializeRequest(now, managed...),
		nil,
	); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}

	handshake := waitForJSONFile[initializeMarker](t, env.handshakePath)
	if got, want := handshake.Request.Runtime.Bridge.Provider, "telegram"; got != want {
		t.Fatalf("handshake provider = %q, want %q", got, want)
	}
	ownership := waitForJSONFile[ownershipMarker](t, env.ownershipPath)
	if got, want := len(ownership.Fetched), 2; got != want {
		t.Fatalf("len(ownership.Fetched) = %d, want %d", got, want)
	}
	states := waitForJSONLinesFile[stateMarker](
		t,
		env.statePath,
		func(items []stateMarker) bool { return len(items) >= 2 },
	)
	if got, want := states[0].Status.Normalize(), bridgepkg.BridgeStatusReady; got != want {
		t.Fatalf("states[0].Status = %q, want %q", got, want)
	}
	waitForCondition(t, func() bool {
		runtime.mu.RLock()
		defer runtime.mu.RUnlock()
		return strings.TrimSpace(runtime.serverAddr) != ""
	})
}

func TestWebhookIngressRejectsInvalidSecretAndIngestsMessage(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newTelegramAPIServer(t)
	t.Setenv(telegramListenAddrEnv, listenAddr)
	t.Setenv(telegramAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 5, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	managed.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "bot_token", Kind: "token", Value: "telegram-token"},
		{BindingName: "webhook_secret", Kind: "token", Value: "top-secret"},
	}

	var ingested []bridgepkg.InboundMessageEnvelope
	var mu sync.Mutex

	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesList),
		func(context.Context, json.RawMessage) (any, error) {
			return []bridgepkg.BridgeInstance{managed.Instance}, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
		func(context.Context, json.RawMessage) (any, error) {
			return managed.Instance, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgesInstancesReportStateParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			instance := managed.Instance
			instance.Status = payload.Status
			return instance, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesMessagesIngest),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var envelope bridgepkg.InboundMessageEnvelope
			if err := json.Unmarshal(params, &envelope); err != nil {
				return nil, err
			}
			mu.Lock()
			ingested = append(ingested, envelope)
			mu.Unlock()
			return extensioncontract.BridgesMessagesIngestResult{
				SessionID:    "sess-1",
				RouteCreated: true,
				RoutingKey: bridgepkg.RoutingKey{
					Scope:            envelope.Scope,
					WorkspaceID:      envelope.WorkspaceID,
					BridgeInstanceID: envelope.BridgeInstanceID,
					PeerID:           envelope.PeerID,
					ThreadID:         envelope.ThreadID,
					GroupID:          envelope.GroupID,
				},
			}, nil
		},
	)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	waitForCondition(t, func() bool {
		runtime.mu.RLock()
		defer runtime.mu.RUnlock()
		return strings.TrimSpace(runtime.serverAddr) != ""
	})

	runtime.mu.RLock()
	serverAddr := runtime.serverAddr
	runtime.mu.RUnlock()
	webhookURL := "http://" + serverAddr + "/telegram/brg-1"

	invalidReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		webhookURL,
		strings.NewReader(telegramWebhookPayload()),
	)
	if err != nil {
		t.Fatalf("http.NewRequest(invalid) error = %v", err)
	}
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	invalidResp, err := http.DefaultClient.Do(invalidReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(invalid) error = %v", err)
	}
	defer func() {
		if closeErr := invalidResp.Body.Close(); closeErr != nil {
			t.Fatalf("invalidResp.Body.Close() error = %v", closeErr)
		}
	}()
	if got, want := invalidResp.StatusCode, http.StatusUnauthorized; got != want {
		t.Fatalf("invalid webhook status = %d, want %d", got, want)
	}

	validReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		webhookURL,
		strings.NewReader(telegramWebhookPayload()),
	)
	if err != nil {
		t.Fatalf("http.NewRequest(valid) error = %v", err)
	}
	validReq.Header.Set("Content-Type", "application/json")
	validReq.Header.Set("X-Telegram-Bot-Api-Secret-Token", "top-secret")
	validResp, err := http.DefaultClient.Do(validReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(valid) error = %v", err)
	}
	defer func() {
		if closeErr := validResp.Body.Close(); closeErr != nil {
			t.Fatalf("validResp.Body.Close() error = %v", closeErr)
		}
	}()
	if got, want := validResp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("valid webhook status = %d, want %d", got, want)
	}

	ingests := waitForJSONLinesFile[ingestMarker](
		t,
		env.ingestPath,
		func(items []ingestMarker) bool {
			return len(items) == 1 && strings.TrimSpace(items[0].Result.SessionID) != ""
		},
	)
	if got, want := ingests[0].Envelope.PeerID, "12345"; got != want {
		t.Fatalf("ingest envelope peer id = %q, want %q", got, want)
	}
	mu.Lock()
	if got, want := len(ingested), 1; got != want {
		t.Fatalf("len(ingested) = %d, want %d", got, want)
	}
	mu.Unlock()
}

func TestRuntimeDeliveriesCallTelegramBotAPI(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newTelegramAPIServer(t)
	t.Setenv(telegramListenAddrEnv, listenAddr)
	t.Setenv(telegramAPIBaseEnv, mockAPI.URL())

	_, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 10, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	managed.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "bot_token", Kind: "token", Value: "telegram-token"},
		{BindingName: "webhook_secret", Kind: "token", Value: "top-secret"},
	}

	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesList),
		func(context.Context, json.RawMessage) (any, error) {
			return []bridgepkg.BridgeInstance{managed.Instance}, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
		func(context.Context, json.RawMessage) (any, error) {
			return managed.Instance, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgesInstancesReportStateParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			instance := managed.Instance
			instance.Status = payload.Status
			return instance, nil
		},
	)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	states := waitForJSONLinesFile[stateMarker](
		t,
		env.statePath,
		func(items []stateMarker) bool { return len(items) >= 1 },
	)
	if got, want := states[len(states)-1].Status.Normalize(), bridgepkg.BridgeStatusReady; got != want {
		t.Fatalf("states[last].Status = %q, want %q", got, want)
	}

	var ack bridgepkg.DeliveryAck
	if err := hostPeer.Call(
		context.Background(),
		"bridges/deliver",
		testDeliveryRequest("brg-1", "delivery-1", 1, bridgepkg.DeliveryEventTypeStart, false),
		&ack,
	); err != nil {
		t.Fatalf("hostPeer.Call(start delivery) error = %v", err)
	}
	finalReq := testDeliveryRequest(
		"brg-1",
		"delivery-1",
		2,
		bridgepkg.DeliveryEventTypeFinal,
		true,
	)
	finalReq.Event.Content.Text = "hello world"
	if err := hostPeer.Call(context.Background(), "bridges/deliver", finalReq, &ack); err != nil {
		t.Fatalf("hostPeer.Call(final delivery) error = %v", err)
	}

	records := waitForJSONLinesFile[deliveryMarker](
		t,
		env.deliveryPath,
		func(items []deliveryMarker) bool { return len(items) >= 2 },
	)
	if records[0].Ack == nil || records[1].Ack == nil {
		t.Fatalf("delivery markers = %#v, want recorded acks", records)
	}

	calls := mockAPI.Calls()
	if got, want := len(calls), 3; got != want {
		t.Fatalf("len(mockAPI calls) = %d, want %d (getMe + send + edit)", got, want)
	}
	if got, want := calls[0].Method, "getMe"; got != want {
		t.Fatalf("calls[0].Method = %q, want %q", got, want)
	}
	if got, want := calls[1].Method, "sendMessage"; got != want {
		t.Fatalf("calls[1].Method = %q, want %q", got, want)
	}
	if got, want := calls[2].Method, "editMessageText"; got != want {
		t.Fatalf("calls[2].Method = %q, want %q", got, want)
	}
}

func TestHandleShutdownWritesMarker(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newTelegramAPIServer(t)
	t.Setenv(telegramListenAddrEnv, listenAddr)
	t.Setenv(telegramAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 15, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	managed.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "bot_token", Kind: "token", Value: "telegram-token"},
		{BindingName: "webhook_secret", Kind: "token", Value: "top-secret"},
	}

	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesList),
		func(context.Context, json.RawMessage) (any, error) {
			return []bridgepkg.BridgeInstance{managed.Instance}, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
		func(context.Context, json.RawMessage) (any, error) {
			return managed.Instance, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgesInstancesReportStateParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			instance := managed.Instance
			instance.Status = payload.Status
			return instance, nil
		},
	)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	if err := runtime.handleShutdown(
		context.Background(),
		nil,
		subprocess.ShutdownRequest{DeadlineMS: 50},
	); err != nil {
		t.Fatalf("handleShutdown() error = %v", err)
	}
	lines := waitForNonEmptyLines(t, env.shutdownPath)
	if len(lines) == 0 || !strings.Contains(lines[0], "pid=") {
		t.Fatalf("shutdown marker lines = %#v, want pid entry", lines)
	}
}

func TestResolveInstanceConfigAndHelperNormalization(t *testing.T) {
	env := setProviderTestEnv(t)
	_ = env
	listenAddr := reserveListenAddr(t)
	mockAPI := newTelegramAPIServer(t)
	apiBaseURL := mockAPI.URL() + "/"

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 14, 0, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	managed.Instance.DMPolicy = bridgepkg.BridgeDMPolicyPairing
	managed.Instance.ProviderConfig = fmt.Appendf(nil, `{
		"api_base_url":%q,
		"webhook":{"listen_addr":%q,"path":"telegram"},
		"batching":{"delay_ms":5,"split_delay_ms":7,"split_threshold":2},
		"dm":{"allow_user_ids":[" 42 "],"allow_usernames":["@Alice"],"paired_usernames":["Bob"]}
	}`, apiBaseURL, listenAddr)
	managed.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "bot_token", Kind: "token", Value: "telegram-token"},
		{BindingName: "webhook_secret", Kind: "token", Value: "top-secret"},
	}

	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesList),
		func(context.Context, json.RawMessage) (any, error) {
			return []bridgepkg.BridgeInstance{managed.Instance}, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
		func(context.Context, json.RawMessage) (any, error) {
			return managed.Instance, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgesInstancesReportStateParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			instance := managed.Instance
			instance.Status = payload.Status
			return instance, nil
		},
	)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}

	session := runtime.currentSession()
	if session == nil {
		t.Fatal("runtime.currentSession() = nil, want session")
	}

	cfg := runtime.resolveInstanceConfig(session, managed)
	if cfg.configError != nil {
		t.Fatalf("resolveInstanceConfig() configError = %v, want nil", cfg.configError)
	}
	defer cfg.batcher.Close()

	if got, want := cfg.apiBaseURL, strings.TrimSuffix(mockAPI.URL(), "/"); got != want {
		t.Fatalf("cfg.apiBaseURL = %q, want %q", got, want)
	}
	if got, want := cfg.listenAddr, listenAddr; got != want {
		t.Fatalf("cfg.listenAddr = %q, want %q", got, want)
	}
	if got, want := cfg.webhookPath, "/telegram"; got != want {
		t.Fatalf("cfg.webhookPath = %q, want %q", got, want)
	}
	if got, want := cfg.botToken, "telegram-token"; got != want {
		t.Fatalf("cfg.botToken = %q, want %q", got, want)
	}
	if got, want := cfg.webhookSecret, "top-secret"; got != want {
		t.Fatalf("cfg.webhookSecret = %q, want %q", got, want)
	}
	if cfg.batcher == nil {
		t.Fatal("cfg.batcher = nil, want batcher")
	}
	if _, ok := cfg.allowUserIDs["42"]; !ok {
		t.Fatalf("cfg.allowUserIDs = %#v, want normalized user id", cfg.allowUserIDs)
	}
	if _, ok := cfg.allowUsernames["alice"]; !ok {
		t.Fatalf("cfg.allowUsernames = %#v, want normalized username", cfg.allowUsernames)
	}
	if _, ok := cfg.pairedUsernames["bob"]; !ok {
		t.Fatalf("cfg.pairedUsernames = %#v, want normalized username", cfg.pairedUsernames)
	}
	if got, want := normalizeWebhookPath("telegram"), "/telegram"; got != want {
		t.Fatalf("normalizeWebhookPath() = %q, want %q", got, want)
	}
	if got, want := normalizeURL("http://example.com/path/"), "http://example.com/path"; got != want {
		t.Fatalf("normalizeURL() = %q, want %q", got, want)
	}

	bad := managed
	bad.Instance.ProviderConfig = []byte("{")
	if cfg := runtime.resolveInstanceConfig(session, bad); cfg.configError == nil {
		t.Fatal("resolveInstanceConfig(bad json) configError = nil, want non-nil")
	}
}

func TestDetermineInitialStateRetryAndHealthHelpers(t *testing.T) {
	t.Parallel()

	runtime, err := newTelegramProvider(io.Discard)
	if err != nil {
		t.Fatalf("newTelegramProvider() error = %v", err)
	}

	badConfig := errors.New("bad config")
	status, degradation, err := runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID:  "cfg-err",
			configError: badConfig,
		},
	)
	if !errors.Is(err, badConfig) {
		t.Fatalf("determineInitialState(configError) error = %v, want %v", err, badConfig)
	}
	if got, want := status, bridgepkg.BridgeStatusDegraded; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil ||
		degradation.Reason != bridgepkg.BridgeDegradationReasonTenantConfigInvalid {
		t.Fatalf("degradation = %#v, want tenant config invalid", degradation)
	}

	status, degradation, err = runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{instanceID: "missing-token"},
	)
	if err == nil {
		t.Fatal("determineInitialState(missing token) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusAuthRequired; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil || degradation.Reason != bridgepkg.BridgeDegradationReasonAuthFailed {
		t.Fatalf("degradation = %#v, want auth failed", degradation)
	}

	runtime.apiFactory = func(cfg resolvedInstanceConfig) telegramAPI {
		switch cfg.instanceID {
		case "auth":
			return fakeTelegramAPIError{err: &bridgesdk.AuthError{Err: errors.New("invalid token")}}
		case "transient":
			return fakeTelegramAPIError{
				err: &bridgesdk.HTTPError{
					StatusCode: http.StatusServiceUnavailable,
					Message:    "provider unavailable",
				},
			}
		default:
			return &fakeTelegramAPI{}
		}
	}

	status, degradation, err = runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID: "auth",
			botToken:   "telegram-token",
		},
	)
	if err == nil {
		t.Fatal("determineInitialState(auth) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusAuthRequired; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil || degradation.Reason != bridgepkg.BridgeDegradationReasonAuthFailed {
		t.Fatalf("degradation = %#v, want auth failed", degradation)
	}

	status, degradation, err = runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID: "transient",
			botToken:   "telegram-token",
		},
	)
	if err == nil {
		t.Fatal("determineInitialState(transient) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusDegraded; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil ||
		degradation.Reason != bridgepkg.BridgeDegradationReasonProviderTimeout {
		t.Fatalf("degradation = %#v, want provider timeout", degradation)
	}

	status, degradation, err = runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID: "ready",
			botToken:   "telegram-token",
		},
	)
	if err != nil {
		t.Fatalf("determineInitialState(ready) error = %v", err)
	}
	if got, want := status, bridgepkg.BridgeStatusReady; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation != nil {
		t.Fatalf("degradation = %#v, want nil", degradation)
	}

	runtime.setLastError(errors.New("boom"))
	if err := runtime.healthCheck(); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("healthCheck() error = %v, want boom", err)
	}
	runtime.clearLastError()
	if err := runtime.healthCheck(); err != nil {
		t.Fatalf("healthCheck() error = %v, want nil", err)
	}

	providerForRetry := &telegramProvider{stopCh: make(chan struct{})}
	attempts := 0
	err = providerForRetry.retryHostCall(context.Background(), func(context.Context) error {
		attempts++
		if attempts < 3 {
			return subprocess.NewRPCError(rpcCodeNotInitialized, "Not initialized", nil)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retryHostCall() error = %v", err)
	}
	if got, want := attempts, 3; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}

	providerStopped := &telegramProvider{stopCh: make(chan struct{})}
	close(providerStopped.stopCh)
	stopErr := subprocess.NewRPCError(rpcCodeNotInitialized, "Not initialized", nil)
	if err := providerStopped.retryHostCall(
		context.Background(),
		func(context.Context) error { return stopErr },
	); !errors.Is(
		err,
		stopErr,
	) {
		t.Fatalf("retryHostCall(stopped) error = %v, want %v", err, stopErr)
	}

	providerWait := &telegramProvider{
		stopCh: make(chan struct{}),
		routes: make(map[string]resolvedInstanceConfig),
	}
	go func() {
		time.Sleep(20 * time.Millisecond)
		providerWait.mu.Lock()
		providerWait.routes["brg-1"] = resolvedInstanceConfig{instanceID: "brg-1"}
		providerWait.mu.Unlock()
	}()
	cfg, err := providerWait.waitForInstanceConfig("brg-1", 200*time.Millisecond)
	if err != nil {
		t.Fatalf("waitForInstanceConfig() error = %v", err)
	}
	if got, want := cfg.instanceID, "brg-1"; got != want {
		t.Fatalf("cfg.instanceID = %q, want %q", got, want)
	}

	if !isNotInitializedRPCError(
		subprocess.NewRPCError(rpcCodeNotInitialized, "Not initialized", nil),
	) {
		t.Fatal("isNotInitializedRPCError() = false, want true")
	}
	if isNotInitializedRPCError(errors.New("boom")) {
		t.Fatal("isNotInitializedRPCError(non-rpc) = true, want false")
	}

	degradation = &bridgepkg.BridgeDegradation{
		Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
		Message: "bad token",
	}
	cloned := cloneDegradation(degradation)
	if cloned == degradation || cloned.Message != degradation.Message {
		t.Fatalf("cloneDegradation() = %#v, want distinct equal copy", cloned)
	}
	if got, want := maxInt(0, 2, 1), 2; got != want {
		t.Fatalf("maxInt() = %d, want %d", got, want)
	}
}

func TestTelegramBotClientAndClassificationHelpers(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch filepath.Base(r.URL.Path) {
		case "deleteMessage":
			writeTelegramAPIResponse(t, w, true)
		case "getMe":
			_, _ = w.Write([]byte(`not-json`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":          false,
				"error_code":  http.StatusInternalServerError,
				"description": "broken",
			})
		}
	}))
	defer server.Close()

	client := &telegramBotClient{
		baseURL:    server.URL,
		botToken:   "telegram-token",
		httpClient: server.Client(),
	}
	if err := client.DeleteMessage(
		context.Background(),
		telegramDeleteMessageRequest{ChatID: "42", MessageID: 99},
	); err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	if _, err := client.GetMe(context.Background()); err == nil {
		t.Fatal("GetMe(invalid json) error = nil, want non-nil")
	}

	rateErr := classifyTelegramHTTPError(429, telegramAPIEnvelope[json.RawMessage]{
		Description: "slow down",
		Parameters:  telegramAPIErrorDetails{RetryAfter: 2},
	})
	var typedRateErr *bridgesdk.RateLimitError
	if !errors.As(rateErr, &typedRateErr) {
		t.Fatalf("classifyTelegramHTTPError(429) = %T, want *RateLimitError", rateErr)
	}

	authErr := classifyTelegramHTTPError(
		401,
		telegramAPIEnvelope[json.RawMessage]{Description: "unauthorized"},
	)
	var typedAuthErr *bridgesdk.AuthError
	if !errors.As(authErr, &typedAuthErr) {
		t.Fatalf("classifyTelegramHTTPError(401) = %T, want *AuthError", authErr)
	}

	httpErr := classifyTelegramHTTPError(0, telegramAPIEnvelope[json.RawMessage]{ErrorCode: 502})
	var typedHTTPErr *bridgesdk.HTTPError
	if !errors.As(httpErr, &typedHTTPErr) {
		t.Fatalf("classifyTelegramHTTPError(default) = %T, want *HTTPError", httpErr)
	}
	if got, want := typedHTTPErr.Message, "telegram bot api error 502"; got != want {
		t.Fatalf("typedHTTPErr.Message = %q, want %q", got, want)
	}

	update := telegramUpdate{EditedMessage: &telegramMessage{MessageID: 1}}
	if message := selectTelegramMessage(update); message == nil || message.MessageID != 1 {
		t.Fatalf("selectTelegramMessage(edited) = %#v, want edited message", message)
	}
	if got, want := resolveTelegramThreadID("654", "-100777"), int64(654); got != want {
		t.Fatalf("resolveTelegramThreadID() = %d, want %d", got, want)
	}
	if got := resolveTelegramThreadID("1", "-100777"); got != 0 {
		t.Fatalf("resolveTelegramThreadID(general topic) = %d, want 0", got)
	}
	chatID, messageID, err := decodeRemoteMessageID("chat:321")
	if err != nil {
		t.Fatalf("decodeRemoteMessageID() error = %v", err)
	}
	if got, want := chatID, "chat"; got != want {
		t.Fatalf("chatID = %q, want %q", got, want)
	}
	if got, want := messageID, int64(321); got != want {
		t.Fatalf("messageID = %d, want %d", got, want)
	}
	if _, _, err := decodeRemoteMessageID("bad"); err == nil {
		t.Fatal("decodeRemoteMessageID(bad) error = nil, want non-nil")
	}
	if got, want := referenceRemoteMessageID(
		&bridgepkg.DeliveryMessageReference{RemoteMessageID: " remote "},
	), "remote"; got != want {
		t.Fatalf("referenceRemoteMessageID() = %q, want %q", got, want)
	}
	if got, want := firstNonEmpty("", " value ", "other"), "value"; got != want {
		t.Fatalf("firstNonEmpty() = %q, want %q", got, want)
	}
}

func TestWebhookShortCircuitsAndBatchDispatch(t *testing.T) {
	env := setProviderTestEnv(t)

	runtime, err := newTelegramProvider(io.Discard)
	if err != nil {
		t.Fatalf("newTelegramProvider() error = %v", err)
	}
	managed := testBridgeRuntime(time.Date(2026, 4, 15, 14, 9, 0, 0, time.UTC), "brg-1")

	dedupCfg := resolvedInstanceConfig{
		instanceID: "brg-1",
		managed:    managed,
		dedup:      bridgesdk.NewDedupCache(time.Minute, 10),
		dmPolicy:   bridgepkg.BridgeDMPolicyOpen,
	}
	dedupCfg.dedup.Mark("telegram:brg-1:1001")
	recorder := httptest.NewRecorder()
	if err := runtime.handleWebhookRequest(recorder, nil, dedupCfg, bridgesdk.WebhookRequest{
		Body:       []byte(telegramWebhookPayload()),
		ReceivedAt: time.Date(2026, 4, 15, 14, 10, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("handleWebhookRequest(dedup) error = %v", err)
	}
	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("recorder.Code = %d, want %d", got, want)
	}

	blockedRecorder := httptest.NewRecorder()
	if err := runtime.handleWebhookRequest(blockedRecorder, nil, resolvedInstanceConfig{
		instanceID: "brg-1",
		managed:    managed,
		dedup:      bridgesdk.NewDedupCache(time.Minute, 10),
		dmPolicy:   bridgepkg.BridgeDMPolicyAllowlist,
	}, bridgesdk.WebhookRequest{
		Body:       []byte(telegramWebhookPayload()),
		ReceivedAt: time.Date(2026, 4, 15, 14, 11, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("handleWebhookRequest(blocked dm) error = %v", err)
	}
	if got, want := blockedRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("blockedRecorder.Code = %d, want %d", got, want)
	}

	noopRecorder := httptest.NewRecorder()
	if err := runtime.handleWebhookRequest(noopRecorder, nil, resolvedInstanceConfig{
		instanceID: "brg-1",
		managed:    managed,
		dedup:      bridgesdk.NewDedupCache(time.Minute, 10),
		dmPolicy:   bridgepkg.BridgeDMPolicyOpen,
	}, bridgesdk.WebhookRequest{
		Body:       []byte(`{"update_id":2001}`),
		ReceivedAt: time.Date(2026, 4, 15, 14, 12, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("handleWebhookRequest(no message) error = %v", err)
	}
	if got, want := noopRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("noopRecorder.Code = %d, want %d", got, want)
	}

	listenAddr := reserveListenAddr(t)
	mockAPI := newTelegramAPIServer(t)
	t.Setenv(telegramListenAddrEnv, listenAddr)
	t.Setenv(telegramAPIBaseEnv, mockAPI.URL())

	runtimeWithSession, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 14, 15, 0, 0, time.UTC)
	managedRuntime := testBridgeRuntime(now, "brg-1")
	var ingested []bridgepkg.InboundMessageEnvelope
	var mu sync.Mutex

	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesList),
		func(context.Context, json.RawMessage) (any, error) {
			return []bridgepkg.BridgeInstance{managedRuntime.Instance}, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
		func(context.Context, json.RawMessage) (any, error) {
			return managedRuntime.Instance, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgesInstancesReportStateParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			instance := managedRuntime.Instance
			instance.Status = payload.Status
			return instance, nil
		},
	)
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesMessagesIngest),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var envelope bridgepkg.InboundMessageEnvelope
			if err := json.Unmarshal(params, &envelope); err != nil {
				return nil, err
			}
			mu.Lock()
			ingested = append(ingested, envelope)
			mu.Unlock()
			return extensioncontract.BridgesMessagesIngestResult{
				SessionID: "sess-1",
				RoutingKey: bridgepkg.RoutingKey{
					Scope:            envelope.Scope,
					WorkspaceID:      envelope.WorkspaceID,
					BridgeInstanceID: envelope.BridgeInstanceID,
					GroupID:          envelope.GroupID,
					ThreadID:         envelope.ThreadID,
				},
			}, nil
		},
	)

	if err := hostPeer.Call(
		context.Background(),
		"initialize",
		testInitializeRequest(now, managedRuntime),
		nil,
	); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	waitForCondition(t, func() bool {
		return runtimeWithSession.currentSession() != nil
	})
	runtimeWithSession.mu.Lock()
	runtimeWithSession.routes["brg-1"] = resolvedInstanceConfig{
		instanceID: "brg-1",
		dedup:      bridgesdk.NewDedupCache(time.Minute, 10),
	}
	runtimeWithSession.mu.Unlock()

	batch := bridgesdk.InboundBatch{
		Items: []bridgepkg.InboundMessageEnvelope{
			{
				BridgeInstanceID:  "brg-1",
				Scope:             bridgepkg.ScopeWorkspace,
				WorkspaceID:       "ws-telegram",
				GroupID:           "-100777",
				ThreadID:          "654",
				PlatformMessageID: "321",
				ReceivedAt:        now,
				Content:           bridgepkg.MessageContent{Text: "hello"},
				EventFamily:       bridgepkg.InboundEventFamilyMessage,
				IdempotencyKey:    "telegram:brg-1:1",
			},
			{
				BridgeInstanceID:  "brg-1",
				Scope:             bridgepkg.ScopeWorkspace,
				WorkspaceID:       "ws-telegram",
				GroupID:           "-100777",
				ThreadID:          "654",
				PlatformMessageID: "322",
				ReceivedAt:        now,
				Content:           bridgepkg.MessageContent{Text: "world"},
				EventFamily:       bridgepkg.InboundEventFamilyMessage,
				IdempotencyKey:    "telegram:brg-1:2",
			},
		},
	}
	if err := runtimeWithSession.dispatchInboundBatch(context.Background(), "brg-1", batch); err != nil {
		t.Fatalf("dispatchInboundBatch() error = %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if got, want := len(ingested), 1; got != want {
		t.Fatalf("len(ingested) = %d, want %d", got, want)
	}
	if got, want := ingested[0].Content.Text, "hello\nworld"; got != want {
		t.Fatalf("ingested[0].Content.Text = %q, want %q", got, want)
	}
	ingests := waitForJSONLinesFile[ingestMarker](
		t,
		env.ingestPath,
		func(items []ingestMarker) bool { return len(items) >= 1 },
	)
	if got, want := ingests[len(ingests)-1].Envelope.Content.Text, "hello\nworld"; got != want {
		t.Fatalf("ingest marker text = %q, want %q", got, want)
	}
}

func TestRunRejectsUnsupportedCommand(t *testing.T) {
	t.Parallel()

	if err := run([]string{"bad"}, strings.NewReader(""), io.Discard, io.Discard); err == nil {
		t.Fatal("run(unsupported) error = nil, want non-nil")
	}
}

func TestRunServeReturnsOnEOF(t *testing.T) {
	t.Parallel()

	done := make(chan error, 1)
	go func() {
		done <- run([]string{"serve"}, strings.NewReader(""), io.Discard, io.Discard)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run(serve) error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run(serve) did not return before timeout")
	}
}

type fakeTelegramAPI struct {
	methods       []string
	nextMessageID int64
}

func (f *fakeTelegramAPI) GetMe(context.Context) (*telegramBotIdentity, error) {
	f.methods = append(f.methods, "getMe")
	return &telegramBotIdentity{ID: 1, Username: "aghbot"}, nil
}

func (f *fakeTelegramAPI) SendMessage(
	_ context.Context,
	_ telegramSendMessageRequest,
) (*telegramSentMessage, error) {
	f.methods = append(f.methods, "sendMessage")
	return &telegramSentMessage{MessageID: f.nextMessageID}, nil
}

func (f *fakeTelegramAPI) EditMessageText(
	_ context.Context,
	_ telegramEditMessageTextRequest,
) error {
	f.methods = append(f.methods, "editMessageText")
	return nil
}

func (f *fakeTelegramAPI) DeleteMessage(_ context.Context, _ telegramDeleteMessageRequest) error {
	f.methods = append(f.methods, "deleteMessage")
	return nil
}

type fakeTelegramAPIError struct {
	err error
}

func (f fakeTelegramAPIError) GetMe(context.Context) (*telegramBotIdentity, error) {
	return nil, f.err
}

func (f fakeTelegramAPIError) SendMessage(
	context.Context,
	telegramSendMessageRequest,
) (*telegramSentMessage, error) {
	return nil, f.err
}

func (f fakeTelegramAPIError) EditMessageText(
	context.Context,
	telegramEditMessageTextRequest,
) error {
	return f.err
}

func (f fakeTelegramAPIError) DeleteMessage(context.Context, telegramDeleteMessageRequest) error {
	return f.err
}

type telegramAPIServer struct {
	server        *httptest.Server
	mu            sync.Mutex
	calls         []telegramAPICall
	nextMessageID int64
}

type telegramAPICall struct {
	Method string
	Body   map[string]any
}

func newTelegramAPIServer(t *testing.T) *telegramAPIServer {
	t.Helper()

	srv := &telegramAPIServer{nextMessageID: 700}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := filepath.Base(r.URL.Path)
		body := map[string]any{}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		srv.mu.Lock()
		srv.calls = append(srv.calls, telegramAPICall{Method: method, Body: body})
		srv.mu.Unlock()

		switch method {
		case "getMe":
			writeTelegramAPIResponse(t, w, map[string]any{"id": 1, "username": "aghbot"})
		case "sendMessage":
			srv.mu.Lock()
			messageID := srv.nextMessageID
			srv.nextMessageID++
			srv.mu.Unlock()
			writeTelegramAPIResponse(t, w, map[string]any{"message_id": messageID})
		case "editMessageText", "deleteMessage":
			writeTelegramAPIResponse(t, w, true)
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ok":          false,
				"error_code":  http.StatusNotFound,
				"description": "unknown method",
			})
		}
	}))
	srv.server = server
	t.Cleanup(server.Close)
	return srv
}

func (s *telegramAPIServer) URL() string {
	return s.server.URL
}

func (s *telegramAPIServer) Calls() []telegramAPICall {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := make([]telegramAPICall, len(s.calls))
	copy(cloned, s.calls)
	return cloned
}

func writeTelegramAPIResponse(t *testing.T, w http.ResponseWriter, result any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"result": result,
	}); err != nil {
		t.Fatalf("json.NewEncoder().Encode() error = %v", err)
	}
}

func newRuntimePeerPair(t *testing.T) (*telegramProvider, *bridgesdk.Peer, func()) {
	t.Helper()

	hostConn, runtimeConn := net.Pipe()
	runtime, err := newTelegramProvider(io.Discard)
	if err != nil {
		t.Fatalf("newTelegramProvider() error = %v", err)
	}

	hostPeer := bridgesdk.NewPeer(hostConn, hostConn)
	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 2)
	go func() { errCh <- runtime.serve(runtimeConn, runtimeConn) }()
	go func() { errCh <- hostPeer.Serve(ctx) }()

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			cancel()
			runtime.stop()
			runtime.mu.RLock()
			server := runtime.server
			listener := runtime.listener
			runtime.mu.RUnlock()
			if listener != nil {
				_ = listener.Close()
			}
			if server != nil {
				shutdownCtx, shutdownCancel := context.WithTimeout(
					context.Background(),
					2*time.Second,
				)
				_ = server.Shutdown(shutdownCtx)
				_ = server.Close()
				shutdownCancel()
			}
			_ = hostConn.Close()
			_ = runtimeConn.Close()
			for range 2 {
				err := <-errCh
				if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, net.ErrClosed) {
					continue
				}
				if strings.Contains(err.Error(), "closed") {
					continue
				}
				t.Fatalf("runtime peer serve error = %v", err)
			}
			runtime.wg.Wait()
		})
	}

	return runtime, hostPeer, cleanup
}

func mustHandle(t *testing.T, peer *bridgesdk.Peer, method string, handler bridgesdk.RPCHandler) {
	t.Helper()
	if err := peer.Handle(method, handler); err != nil {
		t.Fatalf("peer.Handle(%q) error = %v", method, err)
	}
}

func testBridgeRuntime(
	now time.Time,
	instanceID string,
) subprocess.InitializeBridgeManagedInstance {
	return subprocess.InitializeBridgeManagedInstance{
		Instance: bridgepkg.BridgeInstance{
			ID:            instanceID,
			Scope:         bridgepkg.ScopeWorkspace,
			WorkspaceID:   "ws-telegram",
			Platform:      "telegram",
			ExtensionName: "telegram",
			DisplayName:   "Telegram",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{
				IncludePeer:   true,
				IncludeThread: true,
				IncludeGroup:  true,
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		BoundSecrets: []subprocess.InitializeBridgeBoundSecret{
			{BindingName: "bot_token", Kind: "token", Value: "telegram-token"},
			{BindingName: "webhook_secret", Kind: "token", Value: "top-secret"},
		},
	}
}

func testInitializeRequest(
	_ time.Time,
	managed ...subprocess.InitializeBridgeManagedInstance,
) subprocess.InitializeRequest {
	return subprocess.InitializeRequest{
		ProtocolVersion:          "1",
		SupportedProtocolVersion: []string{"1"},
		AGHVersion:               "0.5.0",
		SessionNonce:             "nonce-test",
		Extension: subprocess.InitializeExtension{
			Name:       "telegram",
			Version:    "0.1.0",
			SourceTier: "user",
		},
		Capabilities: subprocess.InitializeCapabilities{
			Provides: []string{"bridge.adapter"},
			GrantedActions: []extensionprotocol.HostAPIMethod{
				extensionprotocol.HostAPIMethodBridgesInstancesList,
				extensionprotocol.HostAPIMethodBridgesInstancesGet,
				extensionprotocol.HostAPIMethodBridgesInstancesReportState,
				extensionprotocol.HostAPIMethodBridgesMessagesIngest,
			},
			GrantedSecurity: []string{"bridge.read", "bridge.write"},
		},
		Methods: subprocess.InitializeMethods{
			ExtensionServices: []string{"bridges/deliver", "health_check", "shutdown"},
		},
		Runtime: subprocess.InitializeRuntime{
			HealthCheckIntervalMS: 30_000,
			HealthCheckTimeoutMS:  5_000,
			ShutdownTimeoutMS:     5_000,
			DefaultHookTimeoutMS:  5_000,
			Bridge: &subprocess.InitializeBridgeRuntime{
				RuntimeVersion:   subprocess.InitializeBridgeRuntimeVersion1,
				Provider:         "telegram",
				Platform:         "telegram",
				ManagedInstances: managed,
			},
		},
	}
}

func testDeliveryRequest(
	instanceID string,
	deliveryID string,
	seq int64,
	eventType string,
	final bool,
) bridgepkg.DeliveryRequest {
	return bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       deliveryID,
			BridgeInstanceID: instanceID,
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-telegram",
				BridgeInstanceID: instanceID,
				PeerID:           "peer-1",
				ThreadID:         "thread-1",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: instanceID,
				PeerID:           "peer-1",
				ThreadID:         "thread-1",
				Mode:             bridgepkg.DeliveryModeReply,
			},
			Seq:       seq,
			EventType: eventType,
			Content:   bridgepkg.MessageContent{Text: "hello"},
			Final:     final,
		},
	}
}

func testDeleteRequest(
	instanceID string,
	deliveryID string,
	seq int64,
	remoteMessageID string,
) bridgepkg.DeliveryRequest {
	req := testDeliveryRequest(instanceID, deliveryID, seq, bridgepkg.DeliveryEventTypeDelete, true)
	req.Event.Operation = bridgepkg.DeliveryOperationDelete
	req.Event.Reference = &bridgepkg.DeliveryMessageReference{RemoteMessageID: remoteMessageID}
	req.Event.Content = bridgepkg.MessageContent{}
	return req
}

func telegramWebhookPayload() string {
	return `{"update_id":1001,"message":{"message_id":9,"date":1775866800,"chat":{"id":12345,"type":"private"},"from":{"id":7,"username":"alice","first_name":"Alice"},"text":"hello"}}`
}

func setProviderTestEnv(t *testing.T) markerEnv {
	t.Helper()

	root := filepath.Join(t.TempDir(), "markers")
	env := markerEnv{
		handshakePath: filepath.Join(root, "handshake.json"),
		ownershipPath: filepath.Join(root, "ownership.json"),
		statePath:     filepath.Join(root, "state.jsonl"),
		deliveryPath:  filepath.Join(root, "delivery.jsonl"),
		ingestPath:    filepath.Join(root, "ingest.jsonl"),
		startsPath:    filepath.Join(root, "starts.log"),
		shutdownPath:  filepath.Join(root, "shutdown.log"),
		crashOncePath: filepath.Join(root, "crash-once.json"),
	}

	t.Setenv(adapterHandshakeEnv, env.handshakePath)
	t.Setenv(adapterOwnershipEnv, env.ownershipPath)
	t.Setenv(adapterStateEnv, env.statePath)
	t.Setenv(adapterDeliveryEnv, env.deliveryPath)
	t.Setenv(adapterIngestEnv, env.ingestPath)
	t.Setenv(adapterStartsEnv, env.startsPath)
	t.Setenv(adapterShutdownEnv, env.shutdownPath)
	t.Setenv(adapterCrashOnceEnv, "")

	return env
}

func reserveListenAddr(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("127.0.0.1:%d", testutil.FreeTCPPort(t))
}

func waitForNonEmptyLines(t *testing.T, path string) []string {
	t.Helper()

	var lines []string
	waitForCondition(t, func() bool {
		payload, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		lines = nonEmptyLines(string(payload))
		return len(lines) > 0
	})
	return lines
}

func waitForJSONFile[T any](t *testing.T, path string) T {
	t.Helper()

	var item T
	waitForCondition(t, func() bool {
		payload, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		return json.Unmarshal(payload, &item) == nil
	})
	return item
}

func waitForJSONLinesFile[T any](t *testing.T, path string, predicate func([]T) bool) []T {
	t.Helper()

	var items []T
	waitForCondition(t, func() bool {
		payload, err := os.ReadFile(path)
		if err != nil {
			return false
		}
		lines := nonEmptyLines(string(payload))
		decoded := make([]T, 0, len(lines))
		for _, line := range lines {
			var item T
			if err := json.Unmarshal([]byte(line), &item); err != nil {
				return false
			}
			decoded = append(decoded, item)
		}
		items = decoded
		return predicate(items)
	})
	return items
}

func waitForCondition(t *testing.T, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition did not succeed before timeout")
}

func nonEmptyLines(input string) []string {
	lines := strings.Split(input, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return filtered
}
