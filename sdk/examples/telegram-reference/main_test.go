package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/subprocess"
)

func TestMapTelegramUpdateToInboundEnvelope(t *testing.T) {
	timestamp := time.Date(2026, 4, 11, 4, 30, 0, 0, time.UTC)
	bridgeRuntime := testBridgeRuntime(timestamp)

	envelope, err := mapTelegramUpdate(telegramUpdate{
		UpdateID: 9001,
		Message: &telegramMessage{
			MessageID:       321,
			MessageThreadID: 654,
			Date:            timestamp.Unix(),
			Chat: telegramChat{
				ID:    777,
				Type:  "supergroup",
				Title: "ops",
			},
			From: telegramUser{
				ID:        888,
				Username:  "alice",
				FirstName: "Alice",
				LastName:  "Example",
			},
			Text: "  Need a summary  ",
		},
	}, bridgeRuntime, func() time.Time {
		return timestamp.Add(2 * time.Hour)
	})
	if err != nil {
		t.Fatalf("mapTelegramUpdate() error = %v", err)
	}

	if got, want := envelope.BridgeInstanceID, bridgeRuntime.Instance.ID; got != want {
		t.Fatalf("envelope.BridgeInstanceID = %q, want %q", got, want)
	}
	if got, want := envelope.Scope, bridgeRuntime.Instance.Scope; got != want {
		t.Fatalf("envelope.Scope = %q, want %q", got, want)
	}
	if got, want := envelope.WorkspaceID, bridgeRuntime.Instance.WorkspaceID; got != want {
		t.Fatalf("envelope.WorkspaceID = %q, want %q", got, want)
	}
	if got, want := envelope.PeerID, "777"; got != want {
		t.Fatalf("envelope.PeerID = %q, want %q", got, want)
	}
	if got, want := envelope.ThreadID, "654"; got != want {
		t.Fatalf("envelope.ThreadID = %q, want %q", got, want)
	}
	if got, want := envelope.PlatformMessageID, "321"; got != want {
		t.Fatalf("envelope.PlatformMessageID = %q, want %q", got, want)
	}
	if got, want := envelope.ReceivedAt, timestamp; !got.Equal(want) {
		t.Fatalf("envelope.ReceivedAt = %s, want %s", got.Format(time.RFC3339Nano), want.Format(time.RFC3339Nano))
	}
	if got, want := envelope.Sender.ID, "888"; got != want {
		t.Fatalf("envelope.Sender.ID = %q, want %q", got, want)
	}
	if got, want := envelope.Sender.Username, "alice"; got != want {
		t.Fatalf("envelope.Sender.Username = %q, want %q", got, want)
	}
	if got, want := envelope.Sender.DisplayName, "Alice Example"; got != want {
		t.Fatalf("envelope.Sender.DisplayName = %q, want %q", got, want)
	}
	if got, want := envelope.Content.Text, "Need a summary"; got != want {
		t.Fatalf("envelope.Content.Text = %q, want %q", got, want)
	}
	if got, want := envelope.IdempotencyKey, "telegram:brg-telegram-reference:9001"; got != want {
		t.Fatalf("envelope.IdempotencyKey = %q, want %q", got, want)
	}
}

func TestBoundSecretValueReadsOnlyBoundLaunchCredentials(t *testing.T) {
	bridgeRuntime := testBridgeRuntime(time.Date(2026, 4, 11, 4, 45, 0, 0, time.UTC))
	bridgeRuntime.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "bot_token", Kind: "token", Value: "  telegram-token  "},
		{BindingName: "webhook_secret", Kind: "token", Value: "webhook-secret"},
	}

	value, ok := boundSecretValue(bridgeRuntime, " bot_token ")
	if !ok {
		t.Fatal("boundSecretValue(bot_token) ok = false, want true")
	}
	if got, want := value, "telegram-token"; got != want {
		t.Fatalf("boundSecretValue(bot_token) = %q, want %q", got, want)
	}

	if got, ok := boundSecretValue(bridgeRuntime, "runtime/vault/read"); ok || got != "" {
		t.Fatalf("boundSecretValue(runtime/vault/read) = (%q, %t), want empty/false", got, ok)
	}
	if got, ok := boundSecretValue(bridgeRuntime, "missing"); ok || got != "" {
		t.Fatalf("boundSecretValue(missing) = (%q, %t), want empty/false", got, ok)
	}
}

func TestAckDeliveryPreservesOrderedRemoteAndReplacementIDs(t *testing.T) {
	runtime := newTelegramReferenceRuntime(io.Discard, nil)

	startAck, err := runtime.ackDelivery(testDeliveryRequest("delivery-1", 1, bridgepkg.DeliveryEventTypeStart, false))
	if err != nil {
		t.Fatalf("ackDelivery(start) error = %v", err)
	}
	if got, want := startAck.RemoteMessageID, "telegram:delivery-1:1"; got != want {
		t.Fatalf("start ack remote_message_id = %q, want %q", got, want)
	}
	if got := startAck.ReplaceRemoteMessageID; got != "" {
		t.Fatalf("start ack replace_remote_message_id = %q, want empty", got)
	}

	deltaAck, err := runtime.ackDelivery(testDeliveryRequest("delivery-1", 2, bridgepkg.DeliveryEventTypeDelta, false))
	if err != nil {
		t.Fatalf("ackDelivery(delta) error = %v", err)
	}
	if got, want := deltaAck.RemoteMessageID, "telegram:delivery-1:2"; got != want {
		t.Fatalf("delta ack remote_message_id = %q, want %q", got, want)
	}
	if got, want := deltaAck.ReplaceRemoteMessageID, startAck.RemoteMessageID; got != want {
		t.Fatalf("delta ack replace_remote_message_id = %q, want %q", got, want)
	}

	finalAck, err := runtime.ackDelivery(testDeliveryRequest("delivery-1", 3, bridgepkg.DeliveryEventTypeFinal, true))
	if err != nil {
		t.Fatalf("ackDelivery(final) error = %v", err)
	}
	if got, want := finalAck.RemoteMessageID, "telegram:delivery-1:3"; got != want {
		t.Fatalf("final ack remote_message_id = %q, want %q", got, want)
	}
	if got, want := finalAck.ReplaceRemoteMessageID, deltaAck.RemoteMessageID; got != want {
		t.Fatalf("final ack replace_remote_message_id = %q, want %q", got, want)
	}
}

func TestAckDeliveryRejectsOutOfOrderSequence(t *testing.T) {
	runtime := newTelegramReferenceRuntime(io.Discard, nil)

	if _, err := runtime.ackDelivery(testDeliveryRequest("delivery-2", 1, bridgepkg.DeliveryEventTypeStart, false)); err != nil {
		t.Fatalf("ackDelivery(start) error = %v", err)
	}
	if _, err := runtime.ackDelivery(testDeliveryRequest("delivery-2", 1, bridgepkg.DeliveryEventTypeDelta, false)); err == nil {
		t.Fatal("ackDelivery(out-of-order) error = nil, want failure")
	}
}

func TestRunRejectsUnsupportedCommand(t *testing.T) {
	err := run([]string{"bogus"}, strings.NewReader(""), io.Discard, io.Discard)
	if err == nil {
		t.Fatal("run(unsupported) error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "unsupported command") {
		t.Fatalf("run(unsupported) error = %v, want unsupported command", err)
	}
}

func TestRunServeReturnsOnEOFAndWritesStartMarker(t *testing.T) {
	env := setAdapterTestEnv(t)

	if err := run(nil, strings.NewReader(""), io.Discard, io.Discard); err != nil {
		t.Fatalf("run(serve) error = %v", err)
	}

	lines := waitForNonEmptyLines(t, env.startsPath)
	if len(lines) == 0 || !strings.Contains(lines[0], "pid=") {
		t.Fatalf("start marker lines = %#v, want pid entry", lines)
	}
}

func TestRPCPeerCallRoundTripAndErrors(t *testing.T) {
	client, server, cleanup := newRPCPeerPair(t)
	defer cleanup()

	server.handle("echo", func(params json.RawMessage) (any, error) {
		var payload struct {
			Value string `json:"value"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, err
		}
		return map[string]string{"value": payload.Value + "!"}, nil
	})
	server.handle("denied", func(json.RawMessage) (any, error) {
		return nil, &runtimeRPCError{Code: -32001, Message: "denied"}
	})
	server.handle("explode", func(json.RawMessage) (any, error) {
		return nil, errors.New("boom")
	})

	var echo struct {
		Value string `json:"value"`
	}
	if err := client.call(context.Background(), "echo", map[string]string{"value": "hi"}, &echo); err != nil {
		t.Fatalf("peer.call(echo) error = %v", err)
	}
	if got, want := echo.Value, "hi!"; got != want {
		t.Fatalf("peer.call(echo) value = %q, want %q", got, want)
	}

	if err := client.call(context.Background(), "denied", nil, nil); err == nil {
		t.Fatal("peer.call(denied) error = nil, want failure")
	} else if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("peer.call(denied) error = %v, want denied", err)
	}

	if err := client.call(context.Background(), "explode", nil, nil); err == nil {
		t.Fatal("peer.call(explode) error = nil, want failure")
	} else if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("peer.call(explode) error = %v, want boom", err)
	}

	if err := client.call(context.Background(), "missing", nil, nil); err == nil {
		t.Fatal("peer.call(missing) error = nil, want failure")
	} else if !strings.Contains(err.Error(), "Method not found") {
		t.Fatalf("peer.call(missing) error = %v, want method not found", err)
	}
}

func TestHandleInitializeReportsReadyAndShutdown(t *testing.T) {
	env := setAdapterTestEnv(t)
	client, server, cleanup := newRPCPeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 11, 7, 0, 0, 0, time.UTC)
	instance := testBridgeRuntime(now).Instance
	var reportedStatuses []bridgepkg.BridgeStatus
	server.handle(string(extensionprotocol.HostAPIMethodBridgesInstancesGet), func(json.RawMessage) (any, error) {
		return instance, nil
	})
	server.handle(string(extensionprotocol.HostAPIMethodBridgesInstancesReportState), func(params json.RawMessage) (any, error) {
		var payload extensioncontract.BridgesInstancesReportStateParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, err
		}
		instance.Status = payload.Status
		reportedStatuses = append(reportedStatuses, payload.Status)
		return instance, nil
	})

	runtime := newTelegramReferenceRuntime(nil, client)
	result, err := runtime.handleInitialize(mustRawJSON(testInitializeRequest(now, true)))
	if err != nil {
		t.Fatalf("handleInitialize() error = %v", err)
	}

	response, ok := result.(subprocess.InitializeResponse)
	if !ok {
		t.Fatalf("handleInitialize() result type = %T, want subprocess.InitializeResponse", result)
	}
	if !response.Supports.HealthCheck {
		t.Fatal("initialize response health support = false, want true")
	}

	states := waitForJSONLinesFile[stateMarker](t, env.statePath, func(items []stateMarker) bool {
		return len(items) > 0
	})
	if got, want := states[len(states)-1].Status.Normalize(), bridgepkg.BridgeStatusReady; got != want {
		t.Fatalf("last state status = %q, want %q", got, want)
	}
	if got, want := len(reportedStatuses), 1; got != want {
		t.Fatalf("reported status count = %d, want %d", got, want)
	}
	if got, want := reportedStatuses[0].Normalize(), bridgepkg.BridgeStatusReady; got != want {
		t.Fatalf("reported status = %q, want %q", got, want)
	}

	handshake := waitForJSONFile[initializeMarker](t, env.handshakePath)
	managed, err := handshake.Request.Runtime.Bridge.SingleManagedInstance()
	if err != nil {
		t.Fatalf("handshake.Request.Runtime.Bridge.SingleManagedInstance() error = %v", err)
	}
	if got, want := managed.Instance.ID, instance.ID; got != want {
		t.Fatalf("handshake runtime instance id = %q, want %q", got, want)
	}
	instanceMarker := waitForJSONFile[bridgepkg.BridgeInstance](t, env.instancePath)
	if got, want := instanceMarker.ID, instance.ID; got != want {
		t.Fatalf("instance marker id = %q, want %q", got, want)
	}

	healthValue, err := runtime.handleHealthCheck(nil)
	if err != nil {
		t.Fatalf("handleHealthCheck() error = %v", err)
	}
	health := healthValue.(subprocess.HealthCheckResponse)
	if !health.Healthy {
		t.Fatalf("health.Healthy = false, want true (message=%q)", health.Message)
	}

	if got, ok := runtime.sessionSnapshot().boundSecret["bot_token"]; !ok || got.Value != "telegram-bot-token" {
		t.Fatalf("sessionSnapshot().boundSecret[bot_token] = %#v, want injected bot token", got)
	}

	shutdownValue, err := runtime.handleShutdown(mustRawJSON(subprocess.ShutdownRequest{DeadlineMS: 50}))
	if err != nil {
		t.Fatalf("handleShutdown() error = %v", err)
	}
	shutdown := shutdownValue.(subprocess.ShutdownResponse)
	if !shutdown.Acknowledged {
		t.Fatal("shutdown.Acknowledged = false, want true")
	}
	if lines := waitForNonEmptyLines(t, env.shutdownPath); len(lines) == 0 {
		t.Fatal("shutdown marker lines = empty, want pid entry")
	}
}

func TestHandleInitializeAuthRequiredAndPollInboundUpdates(t *testing.T) {
	env := setAdapterTestEnv(t)
	client, server, cleanup := newRPCPeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 11, 7, 5, 0, 0, time.UTC)
	instance := testBridgeRuntime(now).Instance
	var ingestCalls atomic.Int64
	server.handle(string(extensionprotocol.HostAPIMethodBridgesInstancesGet), func(json.RawMessage) (any, error) {
		return instance, nil
	})
	server.handle(string(extensionprotocol.HostAPIMethodBridgesInstancesReportState), func(params json.RawMessage) (any, error) {
		var payload extensioncontract.BridgesInstancesReportStateParams
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, err
		}
		instance.Status = payload.Status
		return instance, nil
	})
	server.handle(string(extensionprotocol.HostAPIMethodBridgesMessagesIngest), func(params json.RawMessage) (any, error) {
		if ingestCalls.Add(1) == 1 {
			return nil, &runtimeRPCError{Code: rpcCodeNotInitialized, Message: "Not initialized"}
		}
		var envelope bridgepkg.InboundMessageEnvelope
		if err := json.Unmarshal(params, &envelope); err != nil {
			return nil, err
		}
		return extensioncontract.BridgesMessagesIngestResult{
			SessionID:    "sess-1",
			RouteCreated: true,
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            envelope.Scope,
				WorkspaceID:      envelope.WorkspaceID,
				BridgeInstanceID: envelope.BridgeInstanceID,
				PeerID:           envelope.PeerID,
				ThreadID:         envelope.ThreadID,
			},
		}, nil
	})

	runtime := newTelegramReferenceRuntime(io.Discard, client)
	if _, err := runtime.handleInitialize(mustRawJSON(testInitializeRequest(now, false))); err != nil {
		t.Fatalf("handleInitialize() error = %v", err)
	}

	states := waitForJSONLinesFile[stateMarker](t, env.statePath, func(items []stateMarker) bool {
		return len(items) > 0
	})
	if got, want := states[len(states)-1].Status.Normalize(), bridgepkg.BridgeStatusAuthRequired; got != want {
		t.Fatalf("last state status = %q, want %q", got, want)
	}

	update := telegramUpdate{
		UpdateID: 9002,
		Message: &telegramMessage{
			MessageID:       654,
			MessageThreadID: 987,
			Date:            now.Unix(),
			Chat:            telegramChat{ID: 777},
			From:            telegramUser{ID: 888, Username: "alice"},
			Caption:         "caption fallback",
		},
	}
	if err := appendJSONLine(env.updatesPath, update); err != nil {
		t.Fatalf("appendJSONLine(update) error = %v", err)
	}

	ingests := waitForJSONLinesFile[ingestMarker](t, env.ingestPath, func(items []ingestMarker) bool {
		return len(items) > 0 && strings.TrimSpace(items[len(items)-1].Result.SessionID) != ""
	})
	if got, want := ingests[len(ingests)-1].Envelope.Content.Text, "caption fallback"; got != want {
		t.Fatalf("ingest envelope text = %q, want %q", got, want)
	}
	if got := ingestCalls.Load(); got < 2 {
		t.Fatalf("ingest host call attempts = %d, want retry after not initialized", got)
	}

	if _, err := runtime.handleShutdown(mustRawJSON(subprocess.ShutdownRequest{DeadlineMS: 50})); err != nil {
		t.Fatalf("handleShutdown() error = %v", err)
	}
}

func TestHandleBridgesDeliverRecordsAckAndErrors(t *testing.T) {
	env := setAdapterTestEnv(t)
	runtime := newTelegramReferenceRuntime(io.Discard, nil)
	runtime.initialized = true

	result, err := runtime.handleBridgesDeliver(mustRawJSON(testDeliveryRequest("delivery-3", 1, bridgepkg.DeliveryEventTypeStart, false)))
	if err != nil {
		t.Fatalf("handleBridgesDeliver(start) error = %v", err)
	}
	ack := result.(bridgepkg.DeliveryAck)
	if got, want := ack.RemoteMessageID, "telegram:delivery-3:1"; got != want {
		t.Fatalf("delivery ack remote_message_id = %q, want %q", got, want)
	}

	if _, err := runtime.handleBridgesDeliver(mustRawJSON(testDeliveryRequest("delivery-3", 1, bridgepkg.DeliveryEventTypeDelta, false))); err == nil {
		t.Fatal("handleBridgesDeliver(out-of-order) error = nil, want failure")
	}

	records := waitForJSONLinesFile[deliveryMarker](t, env.deliveryPath, func(items []deliveryMarker) bool {
		return len(items) >= 2
	})
	if records[0].Ack == nil {
		t.Fatal("first delivery marker ack = nil, want ack")
	}
	if records[1].Ack != nil || strings.TrimSpace(records[1].Error) == "" {
		t.Fatalf("second delivery marker = %#v, want recorded error without ack", records[1])
	}

	healthValue, err := runtime.handleHealthCheck(nil)
	if err != nil {
		t.Fatalf("handleHealthCheck() error = %v", err)
	}
	health := healthValue.(subprocess.HealthCheckResponse)
	if health.Healthy {
		t.Fatalf("health.Healthy = true, want false after delivery error (message=%q)", health.Message)
	}
	runtime.clearLastError()
	healthValue, err = runtime.handleHealthCheck(nil)
	if err != nil {
		t.Fatalf("handleHealthCheck() after clear error = %v", err)
	}
	if !healthValue.(subprocess.HealthCheckResponse).Healthy {
		t.Fatal("health after clearLastError = unhealthy, want healthy")
	}
}

func TestUtilityHelpers(t *testing.T) {
	if _, err := mapTelegramUpdate(telegramUpdate{}, testBridgeRuntime(time.Now().UTC()), nil); err == nil {
		t.Fatal("mapTelegramUpdate(nil message) error = nil, want failure")
	}
	if got := indexBoundSecrets(nil); got != nil {
		t.Fatalf("indexBoundSecrets(nil) = %#v, want nil", got)
	}
	if got, want := optionalTelegramID(0), ""; got != want {
		t.Fatalf("optionalTelegramID(0) = %q, want empty", got)
	}
	if !isNotInitializedRPCError(&runtimeRPCError{Code: rpcCodeNotInitialized, Message: "Not initialized"}) {
		t.Fatal("isNotInitializedRPCError() = false, want true")
	}
	if isNotInitializedRPCError(errors.New("boom")) {
		t.Fatal("isNotInitializedRPCError(non-rpc) = true, want false")
	}

	target := filepath.Join(t.TempDir(), "crash-once.json")
	if !shouldCrashOnce(target) {
		t.Fatal("shouldCrashOnce(missing file) = false, want true")
	}
	if err := os.WriteFile(target, []byte("ok"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", target, err)
	}
	if shouldCrashOnce(target) {
		t.Fatal("shouldCrashOnce(existing file) = true, want false")
	}

	markerPath := filepath.Join(t.TempDir(), "markers", "lines.log")
	if err := appendMarkerLine(markerPath, " hello "); err != nil {
		t.Fatalf("appendMarkerLine() error = %v", err)
	}
	if got, want := waitForNonEmptyLines(t, markerPath)[0], "hello"; got != want {
		t.Fatalf("marker line = %q, want %q", got, want)
	}

	jsonlPath := filepath.Join(t.TempDir(), "markers", "data.jsonl")
	if err := appendJSONLine(jsonlPath, map[string]string{"hello": "world"}); err != nil {
		t.Fatalf("appendJSONLine() error = %v", err)
	}
	if got := waitForNonEmptyLines(t, jsonlPath); len(got) != 1 || !strings.Contains(got[0], `"hello":"world"`) {
		t.Fatalf("jsonl lines = %#v, want encoded payload", got)
	}

	jsonPath := filepath.Join(t.TempDir(), "markers", "state.json")
	if err := writeJSONFile(jsonPath, map[string]string{"status": "ready"}); err != nil {
		t.Fatalf("writeJSONFile() error = %v", err)
	}
	if got := waitForNonEmptyLines(t, jsonPath); len(got) != 1 || !strings.Contains(got[0], `"status":"ready"`) {
		t.Fatalf("json file lines = %#v, want encoded payload", got)
	}

	if got := string(mustRawJSON(map[string]string{"key": "value"})); !strings.Contains(got, `"key":"value"`) {
		t.Fatalf("mustRawJSON() = %q, want encoded payload", got)
	}
	if got, want := string(bytesTrim([]byte("  hello \n"))), "hello"; got != want {
		t.Fatalf("bytesTrim() = %q, want %q", got, want)
	}
	lines := nonEmptyLines("\n one \n\n two \n")
	if got, want := strings.Join(lines, ","), "one,two"; got != want {
		t.Fatalf("nonEmptyLines() = %q, want %q", got, want)
	}
}

func testBridgeRuntime(now time.Time) subprocess.InitializeBridgeManagedInstance {
	return subprocess.InitializeBridgeManagedInstance{
		Instance: bridgepkg.BridgeInstance{
			ID:            "brg-telegram-reference",
			Scope:         bridgepkg.ScopeWorkspace,
			WorkspaceID:   "ws-telegram",
			Platform:      "telegram",
			ExtensionName: "telegram-reference",
			DisplayName:   "Telegram Reference",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true, IncludeThread: true},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
}

func testInitializeRequest(now time.Time, includeBotToken bool) subprocess.InitializeRequest {
	bridgeRuntime := testBridgeRuntime(now)
	if includeBotToken {
		bridgeRuntime.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
			{BindingName: "bot_token", Kind: "token", Value: "telegram-bot-token"},
		}
	}
	return subprocess.InitializeRequest{
		ProtocolVersion:          "1",
		SupportedProtocolVersion: []string{"1"},
		AGHVersion:               "0.5.0",
		Extension: subprocess.InitializeExtension{
			Name:       "telegram-reference",
			Version:    "0.1.0",
			SourceTier: "user",
		},
		Capabilities: subprocess.InitializeCapabilities{
			Provides: []string{"bridge.adapter"},
			GrantedActions: []extensionprotocol.HostAPIMethod{
				extensionprotocol.HostAPIMethodBridgesInstancesGet,
				extensionprotocol.HostAPIMethodBridgesInstancesReportState,
				extensionprotocol.HostAPIMethodBridgesMessagesIngest,
			},
			GrantedSecurity: []string{"bridge.read", "bridge.write"},
		},
		Runtime: subprocess.InitializeRuntime{
			HealthCheckIntervalMS: 30_000,
			HealthCheckTimeoutMS:  5_000,
			ShutdownTimeoutMS:     5_000,
			DefaultHookTimeoutMS:  5_000,
			Bridge: &subprocess.InitializeBridgeRuntime{
				RuntimeVersion: subprocess.InitializeBridgeRuntimeVersion1,
				Provider:       "telegram-reference",
				Platform:       bridgeRuntime.Instance.Platform,
				ManagedInstances: []subprocess.InitializeBridgeManagedInstance{
					bridgeRuntime,
				},
			},
		},
	}
}

func testDeliveryRequest(deliveryID string, seq int64, eventType string, final bool) bridgepkg.DeliveryRequest {
	return bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       deliveryID,
			BridgeInstanceID: "brg-telegram-reference",
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-telegram",
				BridgeInstanceID: "brg-telegram-reference",
				PeerID:           "peer-1",
				ThreadID:         "thread-1",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: "brg-telegram-reference",
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

func setAdapterTestEnv(t *testing.T) adapterEnv {
	t.Helper()

	root := filepath.Join(t.TempDir(), "markers")
	env := adapterEnv{
		handshakePath: filepath.Join(root, "handshake.json"),
		instancePath:  filepath.Join(root, "instance.json"),
		statePath:     filepath.Join(root, "state.jsonl"),
		deliveryPath:  filepath.Join(root, "delivery.jsonl"),
		ingestPath:    filepath.Join(root, "ingest.jsonl"),
		updatesPath:   filepath.Join(root, "updates.jsonl"),
		startsPath:    filepath.Join(root, "starts.log"),
		shutdownPath:  filepath.Join(root, "shutdown.log"),
		crashOncePath: filepath.Join(root, "crash-once.json"),
	}

	t.Setenv(adapterHandshakeEnv, env.handshakePath)
	t.Setenv(adapterInstanceEnv, env.instancePath)
	t.Setenv(adapterStateEnv, env.statePath)
	t.Setenv(adapterDeliveryEnv, env.deliveryPath)
	t.Setenv(adapterIngestEnv, env.ingestPath)
	t.Setenv(adapterUpdatesEnv, env.updatesPath)
	t.Setenv(adapterStartsEnv, env.startsPath)
	t.Setenv(adapterShutdownEnv, env.shutdownPath)
	t.Setenv(adapterCrashOnceEnv, "")

	return env
}

func newRPCPeerPair(t *testing.T) (*rpcPeer, *rpcPeer, func()) {
	t.Helper()

	adapterInput, hostOutput := io.Pipe()
	hostInput, adapterOutput := io.Pipe()

	client := newRPCPeer(adapterInput, adapterOutput)
	server := newRPCPeer(hostInput, hostOutput)

	errCh := make(chan error, 2)
	go func() { errCh <- client.serve() }()
	go func() { errCh <- server.serve() }()

	var once sync.Once
	cleanup := func() {
		once.Do(func() {
			_ = adapterOutput.Close()
			_ = hostOutput.Close()
			_ = adapterInput.Close()
			_ = hostInput.Close()
			for i := 0; i < 2; i++ {
				if err := <-errCh; err != nil {
					if errors.Is(err, io.ErrClosedPipe) || strings.Contains(err.Error(), "read/write on closed pipe") {
						continue
					}
					t.Fatalf("rpc peer serve error = %v", err)
				}
			}
		})
	}

	return client, server, cleanup
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

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition did not succeed before timeout")
}
