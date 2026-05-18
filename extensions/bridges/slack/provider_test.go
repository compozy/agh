package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/subprocess"
)

func TestMapSlackMessageEventRoutingAndAttachments(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-slack")

	direct, ignored, err := mapSlackMessageEvent(slackMessageEvent{
		Type:        "message",
		Channel:     "D123",
		ChannelType: "im",
		User:        "U123",
		Username:    "Alice",
		Text:        " hello ",
		ThreadTS:    "1775866800.500000",
		TS:          "1775866800.100000",
		Files: []slackFile{{
			ID:         "F1",
			Name:       "report.txt",
			MIMEType:   "text/plain",
			URLPrivate: "https://files.example/F1",
		}},
	}, managed, now, "Ev1", "T1", now.Unix())
	if err != nil {
		t.Fatalf("mapSlackMessageEvent(direct) error = %v", err)
	}
	if ignored {
		t.Fatal("mapSlackMessageEvent(direct) ignored = true, want false")
	}
	if got, want := direct.Envelope.PeerID, "D123"; got != want {
		t.Fatalf("direct.Envelope.PeerID = %q, want %q", got, want)
	}
	if got, want := direct.Envelope.ThreadID, "1775866800.500000"; got != want {
		t.Fatalf("direct.Envelope.ThreadID = %q, want %q", got, want)
	}
	if got, want := direct.Envelope.Attachments[0].ID, "F1"; got != want {
		t.Fatalf("direct attachment id = %q, want %q", got, want)
	}

	channel, ignored, err := mapSlackMessageEvent(slackMessageEvent{
		Type:        "message",
		Channel:     "C777",
		ChannelType: "channel",
		User:        "U777",
		Username:    "bob",
		Text:        " need summary ",
		TS:          "1775866801.100000",
	}, managed, now, "Ev2", "T1", now.Unix())
	if err != nil {
		t.Fatalf("mapSlackMessageEvent(channel) error = %v", err)
	}
	if ignored {
		t.Fatal("mapSlackMessageEvent(channel) ignored = true, want false")
	}
	if got, want := channel.Envelope.GroupID, "C777"; got != want {
		t.Fatalf("channel.Envelope.GroupID = %q, want %q", got, want)
	}
	if got, want := channel.Envelope.ThreadID, "1775866801.100000"; got != want {
		t.Fatalf("channel.Envelope.ThreadID = %q, want %q", got, want)
	}
	if got, want := channel.Envelope.Content.Text, "need summary"; got != want {
		t.Fatalf("channel.Envelope.Content.Text = %q, want %q", got, want)
	}
}

func TestMapSlackSlashCommandStableTargetIdentity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 1, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-slack")

	mapped, err := mapSlackSlashCommand(url.Values{
		"command":      []string{"/agh"},
		"text":         []string{"summarize this"},
		"user_id":      []string{"u123"},
		"user_name":    []string{"Alice"},
		"channel_id":   []string{"C123"},
		"channel_name": []string{"general"},
		"team_id":      []string{"T123"},
		"trigger_id":   []string{"1337.42"},
		"response_url": []string{"https://hooks.slack.test/cmd"},
	}, managed, now)
	if err != nil {
		t.Fatalf("mapSlackSlashCommand() error = %v", err)
	}
	if got, want := mapped.Envelope.EventFamily, bridgepkg.InboundEventFamilyCommand; got != want {
		t.Fatalf("EventFamily = %q, want %q", got, want)
	}
	if got, want := mapped.Envelope.GroupID, "C123"; got != want {
		t.Fatalf("GroupID = %q, want %q", got, want)
	}
	if got, want := mapped.Envelope.Command.Command, "/agh"; got != want {
		t.Fatalf("Command.Command = %q, want %q", got, want)
	}
	if got, want := mapped.Envelope.Command.TriggerID, "1337.42"; got != want {
		t.Fatalf("Command.TriggerID = %q, want %q", got, want)
	}
	if got, want := mapped.Envelope.IdempotencyKey, "1337.42"; got != want {
		t.Fatalf("IdempotencyKey = %q, want %q", got, want)
	}

	direct, err := mapSlackSlashCommand(url.Values{
		"command":      []string{"/agh"},
		"user_id":      []string{"U999"},
		"user_name":    []string{"Bob"},
		"channel_id":   []string{"D999"},
		"channel_name": []string{"directmessage"},
	}, managed, now)
	if err != nil {
		t.Fatalf("mapSlackSlashCommand(direct) error = %v", err)
	}
	if got, want := direct.Envelope.PeerID, "D999"; got != want {
		t.Fatalf("direct.PeerID = %q, want %q", got, want)
	}
	if got := direct.Envelope.GroupID; got != "" {
		t.Fatalf("direct.GroupID = %q, want empty", got)
	}
}

func TestMapSlackBlockActionsPreserveIdentifiers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 2, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-slack")
	payload := slackBlockActionsPayload{
		Type:        "block_actions",
		ResponseURL: "https://hooks.slack.test/action",
		TriggerID:   "trigger-1",
	}
	payload.Channel.ID = "C123"
	payload.Container.ChannelID = "C123"
	payload.Container.MessageTS = "1775866802.100000"
	payload.Container.ThreadTS = "1775866802.000000"
	payload.User.ID = "U123"
	payload.User.Username = "alice"
	payload.Actions = []slackBlockAction{{
		Type:     "button",
		ActionID: "approve",
		BlockID:  "primary",
		Value:    "yes",
		ActionTS: "1775866802.200000",
	}}

	mapped, err := mapSlackBlockActions(payload, managed, now)
	if err != nil {
		t.Fatalf("mapSlackBlockActions() error = %v", err)
	}
	if got, want := len(mapped), 1; got != want {
		t.Fatalf("len(mapped) = %d, want %d", got, want)
	}
	envelope := mapped[0].Envelope
	if got, want := envelope.EventFamily, bridgepkg.InboundEventFamilyAction; got != want {
		t.Fatalf("EventFamily = %q, want %q", got, want)
	}
	if got, want := envelope.GroupID, "C123"; got != want {
		t.Fatalf("GroupID = %q, want %q", got, want)
	}
	if got, want := envelope.ThreadID, "1775866802.000000"; got != want {
		t.Fatalf("ThreadID = %q, want %q", got, want)
	}
	if got, want := envelope.Action.ActionID, "approve"; got != want {
		t.Fatalf("ActionID = %q, want %q", got, want)
	}
	if got, want := envelope.Action.MessageID, "1775866802.100000"; got != want {
		t.Fatalf("MessageID = %q, want %q", got, want)
	}
	if got, want := envelope.Action.Value, "yes"; got != want {
		t.Fatalf("Value = %q, want %q", got, want)
	}
	if got, want := envelope.IdempotencyKey, "1775866802.200000"; got != want {
		t.Fatalf("IdempotencyKey = %q, want %q", got, want)
	}
	if !strings.Contains(string(envelope.ProviderMetadata), `"response_url":"https://hooks.slack.test/action"`) {
		t.Fatalf("ProviderMetadata = %s, want response_url", envelope.ProviderMetadata)
	}
}

func TestMapSlackReactionEventAndRejectMalformed(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 3, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-slack")

	mapped, err := mapSlackReactionEvent(slackReactionEvent{
		Type:     "reaction_added",
		User:     "U555",
		Reaction: "thumbsup",
		EventTS:  "1775866803.100000",
		Item: slackReactionItem{
			Type:    "message",
			Channel: "C555",
			TS:      "1775866803.000000",
		},
	}, managed, now, "Ev555", "T555")
	if err != nil {
		t.Fatalf("mapSlackReactionEvent(valid) error = %v", err)
	}
	if got, want := mapped.Envelope.EventFamily, bridgepkg.InboundEventFamilyReaction; got != want {
		t.Fatalf("EventFamily = %q, want %q", got, want)
	}
	if got, want := mapped.Envelope.Reaction.Emoji, ":thumbsup:"; got != want {
		t.Fatalf("Reaction.Emoji = %q, want %q", got, want)
	}
	if !mapped.Envelope.Reaction.Added {
		t.Fatal("Reaction.Added = false, want true")
	}
	if got, want := mapped.Envelope.GroupID, "C555"; got != want {
		t.Fatalf("GroupID = %q, want %q", got, want)
	}

	if _, err := mapSlackReactionEvent(slackReactionEvent{
		Type:     "reaction_added",
		User:     "U1",
		Reaction: "eyes",
		Item: slackReactionItem{
			Type: "file",
			TS:   "1775866803.000000",
		},
	}, managed, now, "", ""); err == nil {
		t.Fatal("mapSlackReactionEvent(malformed) error = nil, want non-nil")
	}
}

func TestAllowSlackDirectMessagePolicies(t *testing.T) {
	t.Parallel()

	user := slackUserIdentity{ID: "U123", Username: "alice"}

	if !allowSlackDirectMessage(resolvedInstanceConfig{dmPolicy: bridgepkg.BridgeDMPolicyOpen}, user, true) {
		t.Fatal("allowSlackDirectMessage(open) = false, want true")
	}
	if !allowSlackDirectMessage(resolvedInstanceConfig{
		dmPolicy:       bridgepkg.BridgeDMPolicyAllowlist,
		allowUserIDs:   map[string]struct{}{"U123": {}},
		allowUsernames: map[string]struct{}{"bob": {}},
	}, user, true) {
		t.Fatal("allowSlackDirectMessage(allowlist by id) = false, want true")
	}
	if !allowSlackDirectMessage(resolvedInstanceConfig{
		dmPolicy:        bridgepkg.BridgeDMPolicyPairing,
		pairedUsernames: map[string]struct{}{"alice": {}},
	}, user, true) {
		t.Fatal("allowSlackDirectMessage(pairing by username) = false, want true")
	}
	if allowSlackDirectMessage(resolvedInstanceConfig{dmPolicy: bridgepkg.BridgeDMPolicyAllowlist}, user, true) {
		t.Fatal("allowSlackDirectMessage(rejected direct) = true, want false")
	}
	if !allowSlackDirectMessage(resolvedInstanceConfig{dmPolicy: bridgepkg.BridgeDMPolicyAllowlist}, user, false) {
		t.Fatal("allowSlackDirectMessage(non-direct) = false, want true")
	}
}

func TestVerifySlackSignature(t *testing.T) {
	t.Parallel()

	body := []byte(`{"type":"event_callback"}`)
	now := time.Date(2026, 4, 15, 12, 4, 0, 0, time.UTC)
	timestamp := strconv.FormatInt(now.Unix(), 10)
	signature := slackSignature("top-secret", timestamp, body)

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://example.com/slack/brg-1",
		bytes.NewReader(body),
	)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", signature)

	if err := verifySlackSignature(context.Background(), req, body, "top-secret", now); err != nil {
		t.Fatalf("verifySlackSignature(valid) error = %v", err)
	}
	if err := verifySlackSignature(context.Background(), req, body, "wrong", now); err == nil {
		t.Fatal("verifySlackSignature(invalid secret) error = nil, want non-nil")
	}
	if err := verifySlackSignature(context.Background(), req, body, "top-secret", now.Add(10*time.Minute)); err == nil {
		t.Fatal("verifySlackSignature(stale) error = nil, want non-nil")
	}
}

func TestVerifySlackSignatureValidationErrors(t *testing.T) {
	t.Parallel()

	body := []byte(`{}`)
	now := time.Date(2026, 4, 15, 12, 4, 30, 0, time.UTC)

	if err := verifySlackSignature(context.Background(), nil, body, "top-secret", now); err == nil {
		t.Fatal("verifySlackSignature(nil request) error = nil, want non-nil")
	}
	if err := verifySlackSignature(
		context.Background(),
		httptest.NewRequestWithContext(
			context.Background(),
			http.MethodPost,
			"http://example.com",
			bytes.NewReader(body),
		),
		body,
		"",
		now,
	); err == nil {
		t.Fatal("verifySlackSignature(empty secret) error = nil, want non-nil")
	}

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://example.com",
		bytes.NewReader(body),
	)
	if err := verifySlackSignature(context.Background(), req, body, "top-secret", now); err == nil {
		t.Fatal("verifySlackSignature(missing headers) error = nil, want non-nil")
	}

	req.Header.Set("X-Slack-Request-Timestamp", "not-a-number")
	req.Header.Set("X-Slack-Signature", "v0=signature")
	if err := verifySlackSignature(context.Background(), req, body, "top-secret", now); err == nil {
		t.Fatal("verifySlackSignature(invalid timestamp) error = nil, want non-nil")
	}
}

func TestExecuteDeliveryPostEditDeleteAndResume(t *testing.T) {
	t.Parallel()

	api := &fakeSlackAPI{nextTS: "1775866805.100000"}

	startReq := testDeliveryRequest("brg-slack", "delivery-1", 1, bridgepkg.DeliveryEventTypeStart, false)
	startAck, state, err := executeDelivery(context.Background(), api, startReq, deliveryState{})
	if err != nil {
		t.Fatalf("executeDelivery(start) error = %v", err)
	}
	if got, want := startAck.RemoteMessageID, "C123:1775866805.100000"; got != want {
		t.Fatalf("startAck.RemoteMessageID = %q, want %q", got, want)
	}

	finalReq := testDeliveryRequest("brg-slack", "delivery-1", 2, bridgepkg.DeliveryEventTypeFinal, true)
	finalAck, state, err := executeDelivery(context.Background(), api, finalReq, state)
	if err != nil {
		t.Fatalf("executeDelivery(final) error = %v", err)
	}
	if got, want := finalAck.ReplaceRemoteMessageID, startAck.RemoteMessageID; got != want {
		t.Fatalf("finalAck.ReplaceRemoteMessageID = %q, want %q", got, want)
	}

	deleteReq := testDeleteRequest("brg-slack", "delivery-1", 3, finalAck.RemoteMessageID)
	deleteAck, _, err := executeDelivery(context.Background(), api, deleteReq, state)
	if err != nil {
		t.Fatalf("executeDelivery(delete) error = %v", err)
	}
	if got, want := deleteAck.RemoteMessageID, finalAck.RemoteMessageID; got != want {
		t.Fatalf("deleteAck.RemoteMessageID = %q, want %q", got, want)
	}
	if got, want := strings.Join(api.methods, ","), "chat.postMessage,chat.update,chat.delete"; got != want {
		t.Fatalf("api methods = %q, want %q", got, want)
	}

	resumeAPI := &fakeSlackAPI{nextTS: "1775866806.100000"}
	resumeReq := testDeliveryRequest("brg-slack", "delivery-2", 1, bridgepkg.DeliveryEventTypeResume, false)
	resumeReq.Event.Resume = &bridgepkg.DeliveryResumeState{LatestEventType: bridgepkg.DeliveryEventTypeFinal}
	resumeReq.Snapshot = &bridgepkg.DeliverySnapshot{
		DeliveryID:       "delivery-2",
		SessionID:        "sess-1",
		TurnID:           "turn-1",
		BridgeInstanceID: "brg-slack",
		RoutingKey:       resumeReq.Event.RoutingKey,
		DeliveryTarget:   resumeReq.Event.DeliveryTarget,
		LatestSeq:        1,
		LatestEventType:  bridgepkg.DeliveryEventTypeFinal,
		CurrentContent:   bridgepkg.MessageContent{Text: "hello"},
		Final:            true,
		UpdatedAt:        time.Date(2026, 4, 15, 12, 5, 0, 0, time.UTC),
	}
	resumeAck, _, err := executeDelivery(context.Background(), resumeAPI, resumeReq, deliveryState{})
	if err != nil {
		t.Fatalf("executeDelivery(resume without remote) error = %v", err)
	}
	if got, want := resumeAck.RemoteMessageID, "C123:1775866806.100000"; got != want {
		t.Fatalf("resumeAck.RemoteMessageID = %q, want %q", got, want)
	}
}

func TestRuntimeInitializeStartsWebhookServerAndWritesMarkers(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newSlackAPIServer(t)
	t.Setenv(slackListenAddrEnv, listenAddr)
	t.Setenv(slackAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 0, 0, 0, time.UTC)
	runtime.now = func() time.Time { return now }
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
	if got, want := handshake.Request.Runtime.Bridge.Provider, "slack"; got != want {
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

func TestHandleShutdownWritesMarker(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newSlackAPIServer(t)
	t.Setenv(slackListenAddrEnv, listenAddr)
	t.Setenv(slackAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 2, 0, 0, time.UTC)
	runtime.now = func() time.Time { return now }
	managed := testBridgeRuntime(now, "brg-1")

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

func TestHandleJSONWebhookChallengeAndReaction(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newSlackAPIServer(t)
	t.Setenv(slackListenAddrEnv, listenAddr)
	t.Setenv(slackAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 3, 0, 0, time.UTC)
	runtime.now = func() time.Time { return now }
	managed := testBridgeRuntime(now, "brg-1")

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
			return extensioncontract.BridgesMessagesIngestResult{SessionID: "sess-1"}, nil
		},
	)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	cfg, err := runtime.waitForInstanceConfig("brg-1", time.Second)
	if err != nil {
		t.Fatalf("waitForInstanceConfig() error = %v", err)
	}

	challengeRecorder := httptest.NewRecorder()
	if err := runtime.handleJSONWebhook(context.Background(), challengeRecorder, cfg, bridgesdk.WebhookRequest{
		Body:       []byte(`{"type":"url_verification","challenge":"abc123"}`),
		ReceivedAt: now,
	}); err != nil {
		t.Fatalf("handleJSONWebhook(challenge) error = %v", err)
	}
	if got, want := challengeRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("challenge status = %d, want %d", got, want)
	}
	if !strings.Contains(strings.TrimSpace(challengeRecorder.Body.String()), `"challenge":"abc123"`) {
		t.Fatalf("challenge body = %q, want challenge json", challengeRecorder.Body.String())
	}

	reactionRecorder := httptest.NewRecorder()
	if err := runtime.handleJSONWebhook(context.Background(), reactionRecorder, cfg, bridgesdk.WebhookRequest{
		Body: []byte(
			`{"type":"event_callback","team_id":"T1","event_id":"EvReaction","event":{"type":"reaction_added","user":"U1","reaction":"eyes","event_ts":"1775866803.100000","item":{"type":"message","channel":"C123","ts":"1775866803.000000"}}}`,
		),
		ReceivedAt: now,
	}); err != nil {
		t.Fatalf("handleJSONWebhook(reaction) error = %v", err)
	}
	if got, want := reactionRecorder.Code, http.StatusOK; got != want {
		t.Fatalf("reaction status = %d, want %d", got, want)
	}
	ingests := waitForJSONLinesFile[ingestMarker](
		t,
		env.ingestPath,
		func(items []ingestMarker) bool { return len(items) >= 1 },
	)
	if got, want := ingests[len(ingests)-1].Envelope.EventFamily, bridgepkg.InboundEventFamilyReaction; got != want {
		t.Fatalf("reaction ingest family = %q, want %q", got, want)
	}
}

func TestWebhookIngressRejectsInvalidSignatureAndIngestsMessage(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newSlackAPIServer(t)
	t.Setenv(slackListenAddrEnv, listenAddr)
	t.Setenv(slackAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 5, 0, 0, time.UTC)
	runtime.now = func() time.Time { return now }
	managed := testBridgeRuntime(now, "brg-1")
	managed.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "bot_token", Kind: "token", Value: "xoxb-slack-token"},
		{BindingName: "signing_secret", Kind: "token", Value: "top-secret"},
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
	webhookURL := "http://" + serverAddr + "/slack/brg-1"
	body := []byte(slackMessageWebhookPayload())
	timestamp := strconv.FormatInt(now.Unix(), 10)

	invalidReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		webhookURL,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("http.NewRequest(invalid) error = %v", err)
	}
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.Header.Set("X-Slack-Request-Timestamp", timestamp)
	invalidReq.Header.Set("X-Slack-Signature", "v0=invalid")
	invalidResp, err := http.DefaultClient.Do(invalidReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(invalid) error = %v", err)
	}
	defer func() {
		_ = invalidResp.Body.Close()
	}()
	if got, want := invalidResp.StatusCode, http.StatusUnauthorized; got != want {
		t.Fatalf("invalid webhook status = %d, want %d", got, want)
	}

	validReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		webhookURL,
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("http.NewRequest(valid) error = %v", err)
	}
	validReq.Header.Set("Content-Type", "application/json")
	validReq.Header.Set("X-Slack-Request-Timestamp", timestamp)
	validReq.Header.Set("X-Slack-Signature", slackSignature("top-secret", timestamp, body))
	validResp, err := http.DefaultClient.Do(validReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(valid) error = %v", err)
	}
	defer func() {
		_ = validResp.Body.Close()
	}()
	if got, want := validResp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("valid webhook status = %d, want %d", got, want)
	}

	ingests := waitForJSONLinesFile[ingestMarker](t, env.ingestPath, func(items []ingestMarker) bool {
		return len(items) == 1 && strings.TrimSpace(items[0].Result.SessionID) != ""
	})
	if got, want := ingests[0].Envelope.GroupID, "C123"; got != want {
		t.Fatalf("ingest envelope group id = %q, want %q", got, want)
	}
	mu.Lock()
	if got, want := len(ingested), 1; got != want {
		t.Fatalf("len(ingested) = %d, want %d", got, want)
	}
	mu.Unlock()
}

func TestWebhookIngressHandlesSlashCommandAndBlockActions(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newSlackAPIServer(t)
	t.Setenv(slackListenAddrEnv, listenAddr)
	t.Setenv(slackAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 10, 0, 0, time.UTC)
	runtime.now = func() time.Time { return now }
	managed := testBridgeRuntime(now, "brg-1")
	managed.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "bot_token", Kind: "token", Value: "xoxb-slack-token"},
		{BindingName: "signing_secret", Kind: "token", Value: "top-secret"},
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
	mustHandle(
		t,
		hostPeer,
		string(extensionprotocol.HostAPIMethodBridgesMessagesIngest),
		func(_ context.Context, params json.RawMessage) (any, error) {
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
	webhookURL := "http://" + serverAddr + "/slack/brg-1"

	commandBody := []byte(
		"token=t&team_id=T1&channel_id=C123&channel_name=general&user_id=U123&user_name=alice&command=%2Fagh&text=hello&trigger_id=1337.42",
	)
	postSignedSlackForm(t, webhookURL, "top-secret", now, commandBody)

	actionBody := []byte("payload=" + url.QueryEscape(slackBlockActionsPayloadJSON()))
	postSignedSlackForm(t, webhookURL, "top-secret", now, actionBody)

	ingests := waitForJSONLinesFile[ingestMarker](
		t,
		env.ingestPath,
		func(items []ingestMarker) bool { return len(items) >= 2 },
	)
	if got, want := ingests[0].Envelope.EventFamily, bridgepkg.InboundEventFamilyCommand; got != want {
		t.Fatalf("ingests[0].Envelope.EventFamily = %q, want %q", got, want)
	}
	if got, want := ingests[1].Envelope.EventFamily, bridgepkg.InboundEventFamilyAction; got != want {
		t.Fatalf("ingests[1].Envelope.EventFamily = %q, want %q", got, want)
	}
}

func TestWebhookRetriesAfterTransientIngestFailure(t *testing.T) {
	testCases := []struct {
		name    string
		wantKey string
		invoke  func(context.Context, *slackProvider, resolvedInstanceConfig, time.Time) (*httptest.ResponseRecorder, error)
	}{
		{
			name:    "ShouldRetrySlashCommandAfterTransientIngestFailure",
			wantKey: "1337.42",
			invoke: func(
				ctx context.Context,
				runtime *slackProvider,
				cfg resolvedInstanceConfig,
				now time.Time,
			) (*httptest.ResponseRecorder, error) {
				recorder := httptest.NewRecorder()
				commandBody := []byte(
					"token=t&team_id=T1&channel_id=C123&channel_name=general&user_id=U123&user_name=alice&command=%2Fagh&text=hello&trigger_id=1337.42",
				)
				return recorder, runtime.handleFormWebhook(ctx, recorder, cfg, bridgesdk.WebhookRequest{
					Body:       commandBody,
					ReceivedAt: now,
				})
			},
		},
		{
			name:    "ShouldRetryBlockActionsAfterTransientIngestFailure",
			wantKey: "1775866802.200000",
			invoke: func(
				ctx context.Context,
				runtime *slackProvider,
				cfg resolvedInstanceConfig,
				now time.Time,
			) (*httptest.ResponseRecorder, error) {
				recorder := httptest.NewRecorder()
				body := []byte("payload=" + url.QueryEscape(slackBlockActionsPayloadJSON()))
				return recorder, runtime.handleFormWebhook(ctx, recorder, cfg, bridgesdk.WebhookRequest{
					Body:       body,
					ReceivedAt: now,
				})
			},
		},
		{
			name:    "ShouldRetryJSONMessageAfterTransientIngestFailure",
			wantKey: "EvMessage",
			invoke: func(
				ctx context.Context,
				runtime *slackProvider,
				cfg resolvedInstanceConfig,
				now time.Time,
			) (*httptest.ResponseRecorder, error) {
				recorder := httptest.NewRecorder()
				return recorder, runtime.handleJSONWebhook(ctx, recorder, cfg, bridgesdk.WebhookRequest{
					Body:       []byte(slackMessageWebhookPayload()),
					ReceivedAt: now,
				})
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			env := setProviderTestEnv(t)

			runtime, hostPeer, cleanup := newRuntimePeerPair(t)
			defer cleanup()

			now := time.Date(2026, 4, 15, 13, 12, 0, 0, time.UTC)
			runtime.now = func() time.Time { return now }
			managed := testBridgeRuntime(now, "brg-1")

			var mu sync.Mutex
			attempts := make(map[string]int)

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
					attempts[envelope.IdempotencyKey]++
					attempt := attempts[envelope.IdempotencyKey]
					mu.Unlock()
					if attempt == 1 {
						return nil, errors.New("transient ingest failure")
					}
					return extensioncontract.BridgesMessagesIngestResult{SessionID: "sess-1"}, nil
				},
			)

			if err := hostPeer.Call(
				context.Background(),
				"initialize",
				testInitializeRequest(now, managed),
				nil,
			); err != nil {
				t.Fatalf("hostPeer.Call(initialize) error = %v", err)
			}
			cfg, err := runtime.waitForInstanceConfig("brg-1", time.Second)
			if err != nil {
				t.Fatalf("waitForInstanceConfig() error = %v", err)
			}

			_, firstErr := tt.invoke(context.Background(), runtime, cfg, now)
			var httpErr *bridgesdk.HTTPError
			if !errors.As(firstErr, &httpErr) {
				t.Fatalf("first invoke error type = %T, want *bridgesdk.HTTPError", firstErr)
			}
			if got, want := httpErr.StatusCode, http.StatusInternalServerError; got != want {
				t.Fatalf("first invoke status = %d, want %d", got, want)
			}

			recorder, err := tt.invoke(context.Background(), runtime, cfg, now)
			if err != nil {
				t.Fatalf("second invoke error = %v", err)
			}
			if got, want := recorder.Code, http.StatusOK; got != want {
				t.Fatalf("second invoke status = %d, want %d", got, want)
			}

			mu.Lock()
			attemptCount := attempts[tt.wantKey]
			mu.Unlock()
			if got, want := attemptCount, 2; got != want {
				t.Fatalf("attempts[%q] = %d, want %d", tt.wantKey, got, want)
			}
			if !cfg.dedup.Seen(tt.wantKey) {
				t.Fatalf("cfg.dedup.Seen(%q) = false, want true", tt.wantKey)
			}

			ingests := waitForJSONLinesFile[ingestMarker](t, env.ingestPath, func(items []ingestMarker) bool {
				return len(items) >= 2
			})
			if got, want := ingests[0].Envelope.IdempotencyKey, tt.wantKey; got != want {
				t.Fatalf("ingests[0].Envelope.IdempotencyKey = %q, want %q", got, want)
			}
			if got := strings.TrimSpace(ingests[0].Error); got == "" {
				t.Fatal("ingests[0].Error = empty, want transient failure")
			}
			if got, want := ingests[1].Envelope.IdempotencyKey, tt.wantKey; got != want {
				t.Fatalf("ingests[1].Envelope.IdempotencyKey = %q, want %q", got, want)
			}
			if got := strings.TrimSpace(ingests[1].Error); got != "" {
				t.Fatalf("ingests[1].Error = %q, want empty", got)
			}
			if got, want := ingests[1].Result.SessionID, "sess-1"; got != want {
				t.Fatalf("ingests[1].Result.SessionID = %q, want %q", got, want)
			}
		})
	}
}

func TestRuntimeDeliveriesCallSlackAPI(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newSlackAPIServer(t)
	t.Setenv(slackListenAddrEnv, listenAddr)
	t.Setenv(slackAPIBaseEnv, mockAPI.URL())

	_, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 15, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	managed.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "bot_token", Kind: "token", Value: "xoxb-slack-token"},
		{BindingName: "signing_secret", Kind: "token", Value: "top-secret"},
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
	if err := hostPeer.Call(
		context.Background(),
		"bridges/deliver",
		testDeliveryRequest("brg-1", "delivery-1", 2, bridgepkg.DeliveryEventTypeFinal, true),
		&ack,
	); err != nil {
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
		t.Fatalf("len(mockAPI calls) = %d, want %d (auth + post + update)", got, want)
	}
	if got, want := calls[0].Method, "auth.test"; got != want {
		t.Fatalf("calls[0].Method = %q, want %q", got, want)
	}
	if got, want := calls[1].Method, "chat.postMessage"; got != want {
		t.Fatalf("calls[1].Method = %q, want %q", got, want)
	}
	if got, want := calls[2].Method, "chat.update"; got != want {
		t.Fatalf("calls[2].Method = %q, want %q", got, want)
	}
}

func TestHandleFormWebhookRejectsMissingPayload(t *testing.T) {
	t.Parallel()

	runtime, err := newSlackProvider(io.Discard)
	if err != nil {
		t.Fatalf("newSlackProvider() error = %v", err)
	}
	recorder := httptest.NewRecorder()
	err = runtime.handleFormWebhook(
		context.Background(),
		recorder,
		resolvedInstanceConfig{},
		bridgesdk.WebhookRequest{
			Body:       []byte("payload="),
			ReceivedAt: time.Now().UTC(),
		},
	)
	var httpErr *bridgesdk.HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("handleFormWebhook() error type = %T, want *bridgesdk.HTTPError", err)
	}
	if got, want := httpErr.StatusCode, http.StatusBadRequest; got != want {
		t.Fatalf("httpErr.StatusCode = %d, want %d", got, want)
	}
}

func TestHandleBridgesDeliverErrorPaths(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newSlackAPIServer(t)
	t.Setenv(slackListenAddrEnv, listenAddr)
	t.Setenv(slackAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 18, 0, 0, time.UTC)
	runtime.now = func() time.Time { return now }
	managed := testBridgeRuntime(now, "brg-1")

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
			instance.Degradation = payload.Degradation
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

	if _, err := runtime.handleBridgesDeliver(
		context.Background(),
		session,
		testDeliveryRequest("missing", "delivery-x", 1, bridgepkg.DeliveryEventTypeStart, false),
	); err == nil {
		t.Fatal("handleBridgesDeliver(missing instance) error = nil, want non-nil")
	}

	runtime.apiFactory = func(resolvedInstanceConfig) slackAPI {
		return fakeSlackAPIError{err: &bridgesdk.AuthError{Err: errors.New("invalid_auth")}}
	}
	if _, err := runtime.handleBridgesDeliver(
		context.Background(),
		session,
		testDeliveryRequest("brg-1", "delivery-y", 1, bridgepkg.DeliveryEventTypeStart, false),
	); err == nil {
		t.Fatal("handleBridgesDeliver(auth failure) error = nil, want non-nil")
	}
	lines := waitForJSONLinesFile[deliveryMarker](
		t,
		env.deliveryPath,
		func(items []deliveryMarker) bool { return len(items) >= 2 },
	)
	if lines[0].Error == "" || lines[1].Error == "" {
		t.Fatalf("delivery errors = %#v, want recorded marker failures", lines)
	}
}

func TestDispatchInboundBatchMergesContent(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mockAPI := newSlackAPIServer(t)
	t.Setenv(slackListenAddrEnv, listenAddr)
	t.Setenv(slackAPIBaseEnv, mockAPI.URL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 13, 20, 0, 0, time.UTC)
	runtime.now = func() time.Time { return now }
	managed := testBridgeRuntime(now, "brg-1")

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
			return extensioncontract.BridgesMessagesIngestResult{SessionID: "sess-1"}, nil
		},
	)

	managed.Instance.ProviderConfig = []byte(
		`{"webhook":{"listen_addr":"127.0.0.1:9999","path":"/slack/brg-1"},"batching":{"delay_ms":5,"split_delay_ms":5,"split_threshold":2}}`,
	)
	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}

	cfg, err := runtime.waitForInstanceConfig("brg-1", time.Second)
	if err != nil {
		t.Fatalf("waitForInstanceConfig() error = %v", err)
	}
	if cfg.batcher == nil {
		t.Fatal("cfg.batcher = nil, want batcher")
	}
	if err := runtime.dispatchInboundBatch(context.Background(), "brg-1", bridgesdk.InboundBatch{
		Items: []bridgepkg.InboundMessageEnvelope{
			{
				BridgeInstanceID:  "brg-1",
				Scope:             bridgepkg.ScopeWorkspace,
				WorkspaceID:       "ws-slack",
				GroupID:           "C123",
				ThreadID:          "thread-1",
				PlatformMessageID: "m1",
				ReceivedAt:        now,
				Sender:            bridgepkg.MessageSender{ID: "U1"},
				Content:           bridgepkg.MessageContent{Text: "hello"},
				EventFamily:       bridgepkg.InboundEventFamilyMessage,
				IdempotencyKey:    "k1",
			},
			{
				BridgeInstanceID:  "brg-1",
				Scope:             bridgepkg.ScopeWorkspace,
				WorkspaceID:       "ws-slack",
				GroupID:           "C123",
				ThreadID:          "thread-1",
				PlatformMessageID: "m2",
				ReceivedAt:        now,
				Sender:            bridgepkg.MessageSender{ID: "U1"},
				Content:           bridgepkg.MessageContent{Text: "world"},
				EventFamily:       bridgepkg.InboundEventFamilyMessage,
				IdempotencyKey:    "k2",
			},
		},
	}); err != nil {
		t.Fatalf("dispatchInboundBatch() error = %v", err)
	}
	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(ingested) == 1
	})
	mu.Lock()
	defer mu.Unlock()
	if got, want := ingested[0].Content.Text, "hello\nworld"; got != want {
		t.Fatalf("merged content = %q, want %q", got, want)
	}
	if got, want := ingested[0].IdempotencyKey, "k1:batch:2"; got != want {
		t.Fatalf("merged idempotency key = %q, want %q", got, want)
	}
	if got, want := len(
		waitForJSONLinesFile[ingestMarker](
			t,
			env.ingestPath,
			func(items []ingestMarker) bool { return len(items) == 1 },
		),
	), 1; got != want {
		t.Fatalf("ingest markers = %d, want %d", got, want)
	}
}

func TestStopClosesBatchersWithoutProviderLockDeadlock(t *testing.T) {
	runtime, err := newSlackProvider(io.Discard)
	if err != nil {
		t.Fatalf("newSlackProvider() error = %v", err)
	}

	dispatchStarted := make(chan struct{})
	allowLookup := make(chan struct{})
	dispatchDone := make(chan struct{})

	batcher, err := bridgesdk.NewInboundBatcher(bridgesdk.InboundBatcherConfig{
		Context: context.Background(),
		Delay:   5 * time.Millisecond,
		Dispatch: func(context.Context, bridgesdk.InboundBatch) error {
			close(dispatchStarted)
			<-allowLookup
			_, lookupErr := runtime.configForInstance("brg-1")
			close(dispatchDone)
			return lookupErr
		},
		Now: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		t.Fatalf("NewInboundBatcher() error = %v", err)
	}

	runtime.routes = map[string]resolvedInstanceConfig{
		"brg-1": {
			instanceID:  "brg-1",
			webhookPath: "/slack/brg-1",
			batcher:     batcher,
		},
	}

	if err := batcher.Enqueue(bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  "brg-1",
		Scope:             bridgepkg.ScopeWorkspace,
		WorkspaceID:       "ws-slack",
		GroupID:           "C123",
		ThreadID:          "thread-1",
		PlatformMessageID: "m-1",
		ReceivedAt:        time.Now().UTC(),
		Sender:            bridgepkg.MessageSender{ID: "U1"},
		Content:           bridgepkg.MessageContent{Text: "hello"},
		EventFamily:       bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey:    "slack-stop-deadlock",
	}); err != nil {
		t.Fatalf("batcher.Enqueue() error = %v", err)
	}

	select {
	case <-dispatchStarted:
	case <-time.After(time.Second):
		t.Fatal("dispatch did not start before timeout")
	}

	stopDone := make(chan struct{})
	go func() {
		runtime.stop()
		close(stopDone)
	}()

	close(allowLookup)

	select {
	case <-dispatchDone:
	case <-time.After(time.Second):
		t.Fatal("dispatch remained blocked during stop")
	}

	select {
	case <-stopDone:
	case <-time.After(time.Second):
		t.Fatal("stop() remained blocked while closing batchers")
	}
}

func TestResolveInstanceConfigAndHelperNormalization(t *testing.T) {
	env := setProviderTestEnv(t)
	_ = env

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 14, 0, 0, 0, time.UTC)
	managed := testBridgeRuntime(now, "brg-1")
	managed.Instance.DMPolicy = bridgepkg.BridgeDMPolicyPairing
	configured := managed
	configured.Instance.ProviderConfig = []byte(`{
		"api_base_url":"https://slack-gov.example/api/",
		"webhook":{"listen_addr":"127.0.0.1:9999","path":"slack"},
		"batching":{"delay_ms":5,"split_delay_ms":7,"split_threshold":2},
		"dm":{"allow_user_ids":[" u123 "],"allow_usernames":["@Alice"],"paired_usernames":["Bob"]}
	}`)
	configured.BoundSecrets = []subprocess.InitializeBridgeBoundSecret{
		{BindingName: "bot_token", Kind: "token", Value: "xoxb-slack-token"},
		{BindingName: "signing_secret", Kind: "token", Value: "top-secret"},
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

	cfg := runtime.resolveInstanceConfig(session, configured)
	if cfg.configError != nil {
		t.Fatalf("resolveInstanceConfig() configError = %v, want nil", cfg.configError)
	}
	defer cfg.batcher.Close()

	if got, want := cfg.apiBaseURL, "https://slack-gov.example/api"; got != want {
		t.Fatalf("cfg.apiBaseURL = %q, want %q", got, want)
	}
	if got, want := cfg.listenAddr, "127.0.0.1:9999"; got != want {
		t.Fatalf("cfg.listenAddr = %q, want %q", got, want)
	}
	if got, want := cfg.webhookPath, "/slack"; got != want {
		t.Fatalf("cfg.webhookPath = %q, want %q", got, want)
	}
	if got, want := cfg.botToken, "xoxb-slack-token"; got != want {
		t.Fatalf("cfg.botToken = %q, want %q", got, want)
	}
	if got, want := cfg.signingSecret, "top-secret"; got != want {
		t.Fatalf("cfg.signingSecret = %q, want %q", got, want)
	}
	if cfg.batcher == nil {
		t.Fatal("cfg.batcher = nil, want batcher")
	}
	if _, ok := cfg.allowUserIDs["U123"]; !ok {
		t.Fatalf("cfg.allowUserIDs = %#v, want normalized user id", cfg.allowUserIDs)
	}
	if _, ok := cfg.allowUsernames["alice"]; !ok {
		t.Fatalf("cfg.allowUsernames = %#v, want normalized username", cfg.allowUsernames)
	}
	if _, ok := cfg.pairedUsernames["bob"]; !ok {
		t.Fatalf("cfg.pairedUsernames = %#v, want normalized username", cfg.pairedUsernames)
	}
	if got, want := normalizeWebhookPath("slack"), "/slack"; got != want {
		t.Fatalf("normalizeWebhookPath() = %q, want %q", got, want)
	}
	if got, want := normalizeURL("https://example.com/api/"), "https://example.com/api"; got != want {
		t.Fatalf("normalizeURL() = %q, want %q", got, want)
	}

	bad := configured
	bad.Instance.ProviderConfig = []byte("{")
	if cfg := runtime.resolveInstanceConfig(session, bad); cfg.configError == nil {
		t.Fatal("resolveInstanceConfig(bad json) configError = nil, want non-nil")
	}
}

func TestSlackWebhookPathConflictsDegradeAllRoutes(t *testing.T) {
	t.Parallel()

	configs := make([]resolvedInstanceConfig, 0, 2)
	usedPaths := make(map[string]int)

	first := resolvedInstanceConfig{instanceID: "brg-1", webhookPath: "/slack"}
	applySlackWebhookPathConflict(&first, usedPaths, configs)
	configs = append(configs, first)

	second := resolvedInstanceConfig{instanceID: "brg-2", webhookPath: "/slack"}
	applySlackWebhookPathConflict(&second, usedPaths, configs)
	configs = append(configs, second)

	if configs[0].configError == nil {
		t.Fatal("first duplicate configError = nil, want conflict degradation")
	}
	if configs[1].configError == nil {
		t.Fatal("second duplicate configError = nil, want conflict degradation")
	}

	runtime, err := newSlackProvider(io.Discard)
	if err != nil {
		t.Fatalf("newSlackProvider() error = %v", err)
	}
	runtime.routes = buildSlackRouteMap(configs)

	if _, ok := runtime.configForPath("/slack"); ok {
		t.Fatal("configForPath(/slack) returned conflicted config, want deterministic rejection")
	}
}

func TestDetermineInitialStateRetryAndHealthHelpers(t *testing.T) {
	t.Parallel()

	runtime, err := newSlackProvider(io.Discard)
	if err != nil {
		t.Fatalf("newSlackProvider() error = %v", err)
	}

	badConfig := errors.New("bad config")
	status, degradation, err := runtime.determineInitialState(context.Background(), resolvedInstanceConfig{
		instanceID:  "cfg-err",
		configError: badConfig,
	})
	if !errors.Is(err, badConfig) {
		t.Fatalf("determineInitialState(configError) error = %v, want %v", err, badConfig)
	}
	if got, want := status, bridgepkg.BridgeStatusDegraded; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if got, want := degradation.Reason, bridgepkg.BridgeDegradationReasonTenantConfigInvalid; got != want {
		t.Fatalf("degradation reason = %q, want %q", got, want)
	}

	runtime.apiFactory = func(cfg resolvedInstanceConfig) slackAPI {
		switch cfg.instanceID {
		case "auth":
			return fakeSlackAPIError{err: &bridgesdk.AuthError{Err: errors.New("invalid_auth")}}
		case "transient":
			return fakeSlackAPIError{err: &bridgesdk.TransientError{Err: errors.New("unavailable")}}
		default:
			return &fakeSlackAPI{}
		}
	}

	status, degradation, err = runtime.determineInitialState(context.Background(), resolvedInstanceConfig{
		instanceID:    "auth",
		botToken:      "xoxb",
		signingSecret: "secret",
	})
	if err == nil {
		t.Fatal("determineInitialState(auth) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusAuthRequired; got != want {
		t.Fatalf("auth status = %q, want %q", got, want)
	}
	if got, want := degradation.Reason, bridgepkg.BridgeDegradationReasonAuthFailed; got != want {
		t.Fatalf("auth degradation = %q, want %q", got, want)
	}

	status, degradation, err = runtime.determineInitialState(context.Background(), resolvedInstanceConfig{
		instanceID:    "transient",
		botToken:      "xoxb",
		signingSecret: "secret",
	})
	if err == nil {
		t.Fatal("determineInitialState(transient) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusDegraded; got != want {
		t.Fatalf("transient status = %q, want %q", got, want)
	}
	if got, want := degradation.Reason, bridgepkg.BridgeDegradationReasonProviderTimeout; got != want {
		t.Fatalf("transient degradation reason = %q, want %q", got, want)
	}

	runtime.setLastError(errors.New("boom"))
	if err := runtime.healthCheck(); err == nil {
		t.Fatal("healthCheck() error = nil, want non-nil")
	}
	runtime.clearLastError()
	if err := runtime.healthCheck(); err != nil {
		t.Fatalf("healthCheck(clear) error = %v", err)
	}
}

func TestRetryHostCallRetriesNotInitialized(t *testing.T) {
	t.Parallel()

	runtime, err := newSlackProvider(io.Discard)
	if err != nil {
		t.Fatalf("newSlackProvider() error = %v", err)
	}
	attempts := 0
	err = runtime.retryHostCall(context.Background(), func(context.Context) error {
		attempts++
		if attempts < 3 {
			return subprocess.NewRPCError(rpcCodeNotInitialized, "not ready", nil)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retryHostCall() error = %v", err)
	}
	if got, want := attempts, 3; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}
}

func TestRetryHostCallHonorsContextAndStop(t *testing.T) {
	t.Parallel()

	runtime, err := newSlackProvider(io.Discard)
	if err != nil {
		t.Fatalf("newSlackProvider() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := runtime.retryHostCall(ctx, func(context.Context) error {
		return subprocess.NewRPCError(rpcCodeNotInitialized, "not ready", nil)
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("retryHostCall(context canceled) error = %v, want %v", err, context.Canceled)
	}

	runtime.stop()
	stopErr := subprocess.NewRPCError(rpcCodeNotInitialized, "still stopping", nil)
	if err := runtime.retryHostCall(context.Background(), func(context.Context) error {
		return stopErr
	}); !errors.Is(err, stopErr) {
		t.Fatalf("retryHostCall(stop) error = %v, want %v", err, stopErr)
	}
}

func TestClassifySlackAPIErrorAndDeleteMessage(t *testing.T) {
	t.Parallel()

	rateErr := classifySlackAPIError(http.StatusTooManyRequests, "ratelimited", 3*time.Second)
	var typedRateErr *bridgesdk.RateLimitError
	if !errors.As(rateErr, &typedRateErr) {
		t.Fatalf("rateErr type = %T, want *bridgesdk.RateLimitError", rateErr)
	}

	authErr := classifySlackAPIError(http.StatusUnauthorized, "invalid_auth", 0)
	var typedAuthErr *bridgesdk.AuthError
	if !errors.As(authErr, &typedAuthErr) {
		t.Fatalf("authErr type = %T, want *bridgesdk.AuthError", authErr)
	}

	transientErr := classifySlackAPIError(http.StatusServiceUnavailable, "service_unavailable", 0)
	var typedTransientErr *bridgesdk.TransientError
	if !errors.As(transientErr, &typedTransientErr) {
		t.Fatalf("transientErr type = %T, want *bridgesdk.TransientError", transientErr)
	}

	permanentErr := classifySlackAPIError(0, "unknown_problem", 0)
	var typedPermanentErr *bridgesdk.PermanentError
	if !errors.As(permanentErr, &typedPermanentErr) {
		t.Fatalf("permanentErr type = %T, want *bridgesdk.PermanentError", permanentErr)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := strings.TrimPrefix(r.URL.Path, "/"), "chat.delete"; got != want {
			t.Fatalf("method path = %q, want %q", got, want)
		}
		writeSlackAPIResponse(t, w, map[string]any{})
	}))
	defer server.Close()

	client := &slackBotClient{
		baseURL:    server.URL,
		botToken:   "xoxb-token",
		httpClient: &http.Client{Timeout: time.Second},
	}
	if err := client.DeleteMessage(
		context.Background(),
		slackDeleteMessageRequest{Channel: "C123", TS: "1775866808.100000"},
	); err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	if got, want := parseRetryAfter("3"), 3*time.Second; got != want {
		t.Fatalf("parseRetryAfter() = %v, want %v", got, want)
	}
	if got, want := maxInt(0, 500), 500; got != want {
		t.Fatalf("maxInt() = %d, want %d", got, want)
	}
}

func TestSlackBotClientCallBranches(t *testing.T) {
	t.Parallel()

	if err := ((*slackBotClient)(nil)).call(
		context.Background(),
		"chat.postMessage",
		map[string]any{},
		nil,
	); err == nil {
		t.Fatal("nil client call error = nil, want non-nil")
	}

	client := &slackBotClient{baseURL: "http://example.com", botToken: "xoxb"}
	if err := client.call(context.Background(), "chat.postMessage", func() {}, nil); err == nil {
		t.Fatal("marshal failure error = nil, want non-nil")
	}

	badURLClient := &slackBotClient{baseURL: "://bad-url", botToken: "xoxb"}
	if err := badURLClient.call(context.Background(), "chat.postMessage", map[string]any{}, nil); err == nil {
		t.Fatal("bad URL error = nil, want non-nil")
	}

	t.Run("Should rate limited", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			writeSlackAPIResponse(t, w, map[string]any{"ok": false, "error": "ratelimited"})
		}))
		defer server.Close()

		client := &slackBotClient{baseURL: server.URL, botToken: "xoxb", httpClient: &http.Client{Timeout: time.Second}}
		err := client.call(context.Background(), "chat.postMessage", map[string]any{"channel": "C1"}, nil)
		var rateErr *bridgesdk.RateLimitError
		if !errors.As(err, &rateErr) {
			t.Fatalf("rate limited error type = %T, want *bridgesdk.RateLimitError", err)
		}
		if got, want := rateErr.RetryAfter, 7*time.Second; got != want {
			t.Fatalf("RetryAfter = %v, want %v", got, want)
		}
	})

	t.Run("Should decode response failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer server.Close()

		client := &slackBotClient{baseURL: server.URL, botToken: "xoxb", httpClient: &http.Client{Timeout: time.Second}}
		if err := client.call(context.Background(), "auth.test", map[string]any{}, nil); err == nil {
			t.Fatal("decode response error = nil, want non-nil")
		}
	})

	t.Run("Should api error classification", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeSlackAPIResponse(t, w, map[string]any{"ok": false, "error": "missing_scope"})
		}))
		defer server.Close()

		client := &slackBotClient{baseURL: server.URL, botToken: "xoxb", httpClient: &http.Client{Timeout: time.Second}}
		err := client.call(context.Background(), "chat.postMessage", map[string]any{"channel": "C1"}, nil)
		var authErr *bridgesdk.AuthError
		if !errors.As(err, &authErr) {
			t.Fatalf("api error type = %T, want *bridgesdk.AuthError", err)
		}
	})

	t.Run("Should decode result failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeSlackAPIResponse(t, w, map[string]any{"ok": true, "ts": "1775866808.100000"})
		}))
		defer server.Close()

		client := &slackBotClient{baseURL: server.URL, botToken: "xoxb", httpClient: &http.Client{Timeout: time.Second}}
		if err := client.call(
			context.Background(),
			"chat.postMessage",
			map[string]any{"channel": "C1"},
			make(chan int),
		); err == nil {
			t.Fatal("decode result error = nil, want non-nil")
		}
	})

	t.Run("Should wrapper success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch strings.TrimPrefix(r.URL.Path, "/") {
			case "auth.test":
				writeSlackAPIResponse(t, w, map[string]any{"ok": true, "bot_id": "B1", "user_id": "U1"})
			case "chat.postMessage":
				writeSlackAPIResponse(t, w, map[string]any{"ok": true, "ts": "1775866808.200000"})
			default:
				t.Fatalf("unexpected method path %q", r.URL.Path)
			}
		}))
		defer server.Close()

		client := &slackBotClient{baseURL: server.URL, botToken: "xoxb", httpClient: &http.Client{Timeout: time.Second}}
		auth, err := client.AuthTest(context.Background())
		if err != nil {
			t.Fatalf("AuthTest() error = %v", err)
		}
		if got, want := auth.BotID, "B1"; got != want {
			t.Fatalf("auth.BotID = %q, want %q", got, want)
		}
		posted, err := client.PostMessage(context.Background(), slackPostMessageRequest{Channel: "C1", Text: "hello"})
		if err != nil {
			t.Fatalf("PostMessage() error = %v", err)
		}
		if got, want := posted.TS, "1775866808.200000"; got != want {
			t.Fatalf("posted.TS = %q, want %q", got, want)
		}
	})
}

func TestProviderHelperEdges(t *testing.T) {
	t.Parallel()

	if !isIgnoredSlackMessageEvent(slackMessageEvent{Type: "message"}) {
		t.Fatal("isIgnoredSlackMessageEvent(no user) = false, want true")
	}
	if !isIgnoredSlackMessageEvent(slackMessageEvent{Type: "message", User: "U1", BotID: "B1"}) {
		t.Fatal("isIgnoredSlackMessageEvent(bot) = false, want true")
	}
	if !isIgnoredSlackMessageEvent(slackMessageEvent{Type: "message", User: "U1", Subtype: "channel_join"}) {
		t.Fatal("isIgnoredSlackMessageEvent(channel_join) = false, want true")
	}
	if isIgnoredSlackMessageEvent(slackMessageEvent{Type: "message", User: "U1", Text: "hello"}) {
		t.Fatal("isIgnoredSlackMessageEvent(normal message) = true, want false")
	}

	if _, err := parseSlackTimestamp(""); err == nil {
		t.Fatal("parseSlackTimestamp(empty) error = nil, want non-nil")
	}
	if _, err := parseSlackTimestamp("nope"); err == nil {
		t.Fatal("parseSlackTimestamp(invalid) error = nil, want non-nil")
	}
	if got, want := normalizeSlackEmoji(""), ""; got != want {
		t.Fatalf("normalizeSlackEmoji(empty) = %q, want %q", got, want)
	}
	if got, want := normalizeURL(""), ""; got != want {
		t.Fatalf("normalizeURL(empty) = %q, want %q", got, want)
	}
	if got, want := referenceRemoteMessageID(nil), ""; got != want {
		t.Fatalf("referenceRemoteMessageID(nil) = %q, want %q", got, want)
	}
}

func TestRunRejectsUnsupportedCommand(t *testing.T) {
	t.Parallel()

	if err := run([]string{"unknown"}, nil, io.Discard, io.Discard); err == nil {
		t.Fatal("run(unknown) error = nil, want non-nil")
	}
}

type fakeSlackAPI struct {
	methods []string
	nextTS  string
}

func (f fakeSlackAPI) AuthTest(context.Context) (*slackAuthIdentity, error) {
	return &slackAuthIdentity{UserID: "U_BOT", BotID: "B_BOT"}, nil
}

func (f *fakeSlackAPI) PostMessage(_ context.Context, _ slackPostMessageRequest) (*slackPostedMessage, error) {
	f.methods = append(f.methods, "chat.postMessage")
	if strings.TrimSpace(f.nextTS) == "" {
		f.nextTS = "1775866805.100000"
	}
	return &slackPostedMessage{TS: f.nextTS}, nil
}

func (f *fakeSlackAPI) UpdateMessage(context.Context, slackUpdateMessageRequest) error {
	f.methods = append(f.methods, "chat.update")
	return nil
}

func (f *fakeSlackAPI) DeleteMessage(context.Context, slackDeleteMessageRequest) error {
	f.methods = append(f.methods, "chat.delete")
	return nil
}

type fakeSlackAPIError struct {
	err error
}

func (f fakeSlackAPIError) AuthTest(context.Context) (*slackAuthIdentity, error) {
	return nil, f.err
}

func (f fakeSlackAPIError) PostMessage(context.Context, slackPostMessageRequest) (*slackPostedMessage, error) {
	return nil, f.err
}

func (f fakeSlackAPIError) UpdateMessage(context.Context, slackUpdateMessageRequest) error {
	return f.err
}

func (f fakeSlackAPIError) DeleteMessage(context.Context, slackDeleteMessageRequest) error {
	return f.err
}

type slackAPIServer struct {
	mu     sync.Mutex
	server *httptest.Server
	calls  []slackAPICall
	nextTS string
}

type slackAPICall struct {
	Method string
	Body   map[string]any
}

func newSlackAPIServer(t *testing.T) *slackAPIServer {
	t.Helper()

	srv := &slackAPIServer{nextTS: "1775866808.100000"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := strings.TrimPrefix(r.URL.Path, "/")
		body := make(map[string]any)
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			t.Fatalf("json.NewDecoder().Decode() error = %v", err)
		}
		srv.mu.Lock()
		srv.calls = append(srv.calls, slackAPICall{Method: method, Body: body})
		srv.mu.Unlock()

		switch method {
		case "auth.test":
			writeSlackAPIResponse(t, w, map[string]any{"user_id": "U_BOT", "bot_id": "B_BOT"})
		case "chat.postMessage":
			writeSlackAPIResponse(t, w, map[string]any{"ts": srv.nextTS})
		case "chat.update":
			writeSlackAPIResponse(t, w, map[string]any{"ts": body["ts"]})
		case "chat.delete":
			writeSlackAPIResponse(t, w, map[string]any{})
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "unknown_method"})
		}
	}))
	srv.server = server
	t.Cleanup(server.Close)
	return srv
}

func (s *slackAPIServer) URL() string {
	return s.server.URL
}

func (s *slackAPIServer) Calls() []slackAPICall {
	s.mu.Lock()
	defer s.mu.Unlock()
	cloned := make([]slackAPICall, len(s.calls))
	copy(cloned, s.calls)
	return cloned
}

func writeSlackAPIResponse(t *testing.T, w http.ResponseWriter, result any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	payload := map[string]any{"ok": true}
	switch typed := result.(type) {
	case map[string]any:
		maps.Copy(payload, typed)
	default:
		raw, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		payload["ok"] = true
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("json.NewEncoder().Encode() error = %v", err)
	}
}

func newRuntimePeerPair(t *testing.T) (*slackProvider, *bridgesdk.Peer, func()) {
	t.Helper()

	hostConn, runtimeConn := net.Pipe()
	runtime, err := newSlackProvider(io.Discard)
	if err != nil {
		t.Fatalf("newSlackProvider() error = %v", err)
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
			listener := runtime.serverListener
			runtime.mu.RUnlock()
			if listener != nil {
				_ = listener.Close()
			}
			if server != nil {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
				if err := server.Shutdown(shutdownCtx); err != nil {
					_ = server.Close()
				}
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

func testBridgeRuntime(now time.Time, instanceID string) subprocess.InitializeBridgeManagedInstance {
	return subprocess.InitializeBridgeManagedInstance{
		Instance: bridgepkg.BridgeInstance{
			ID:            instanceID,
			Scope:         bridgepkg.ScopeWorkspace,
			WorkspaceID:   "ws-slack",
			Platform:      "slack",
			ExtensionName: "slack",
			DisplayName:   "Slack",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true, IncludeThread: true, IncludeGroup: true},
			CreatedAt:     now,
			UpdatedAt:     now,
		},
		BoundSecrets: []subprocess.InitializeBridgeBoundSecret{
			{BindingName: "bot_token", Kind: "token", Value: "xoxb-slack-token"},
			{BindingName: "signing_secret", Kind: "token", Value: "top-secret"},
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
			Name:       "slack",
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
				Provider:         "slack",
				Platform:         "slack",
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
				WorkspaceID:      "ws-slack",
				BridgeInstanceID: instanceID,
				GroupID:          "C123",
				ThreadID:         "1775866805.000000",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: instanceID,
				GroupID:          "C123",
				ThreadID:         "1775866805.000000",
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

func slackMessageWebhookPayload() string {
	return `{"type":"event_callback","team_id":"T123","event_id":"EvMessage","event_time":1775866800,"event":{"type":"message","channel":"C123","channel_type":"channel","user":"U123","username":"alice","text":"hello","ts":"1775866800.100000"}}`
}

func slackBlockActionsPayloadJSON() string {
	return `{"type":"block_actions","trigger_id":"trigger-1","response_url":"https://hooks.slack.test/action","channel":{"id":"C123"},"container":{"type":"message","channel_id":"C123","message_ts":"1775866802.100000","thread_ts":"1775866802.000000"},"message":{"ts":"1775866802.100000","thread_ts":"1775866802.000000"},"user":{"id":"U123","username":"alice"},"actions":[{"type":"button","action_id":"approve","value":"yes","action_ts":"1775866802.200000"}]}`
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

	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("ln.Close() error = %v", err)
	}
	return addr
}

func postSignedSlackForm(t *testing.T, webhookURL string, secret string, now time.Time, body []byte) {
	t.Helper()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	timestamp := strconv.FormatInt(now.Unix(), 10)
	req.Header.Set("X-Slack-Request-Timestamp", timestamp)
	req.Header.Set("X-Slack-Signature", slackSignature(secret, timestamp, body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do() error = %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("form webhook status = %d, want %d", got, want)
	}
}

func slackSignature(secret string, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(slackSignatureVersion + ":" + strings.TrimSpace(timestamp) + ":"))
	_, _ = mac.Write(body)
	return slackSignatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))
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
