package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
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

	"github.com/golang-jwt/jwt/v5"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	extensionprotocol "github.com/pedronauck/agh/internal/extension/protocol"
	"github.com/pedronauck/agh/internal/subprocess"
)

func TestMapTeamsActivityFamiliesAndDMPolicy(t *testing.T) {
	t.Parallel()

	cfg := resolvedInstanceConfig{
		instanceID: "brg-teams",
		managed: &subprocess.InitializeBridgeManagedInstance{
			Instance: bridgepkg.BridgeInstance{
				ID:          "brg-teams",
				Scope:       bridgepkg.ScopeWorkspace,
				WorkspaceID: "ws-teams",
			},
		},
		serviceURL: teamsDefaultServiceURL,
	}

	messageActivity := teamsActivity{
		Type:       "message",
		ID:         "activity-1",
		Text:       "<at>Bot</at> Need a summary",
		Timestamp:  "2026-04-15T18:05:00Z",
		ServiceURL: teamsDefaultServiceURL,
		ChannelID:  "msteams",
		From:       teamsChannelAccount{ID: "29:user-1", Name: "Alice Example"},
		Recipient:  teamsChannelAccount{ID: "28:bot", Name: "Bridge Bot"},
		Conversation: teamsConversation{
			ID:               "19:channel@thread.tacv2;messageid=activity-1",
			ConversationType: "channel",
			TenantID:         "11111111-2222-3333-4444-555555555555",
		},
		Attachments: []teamsAttachment{{
			ContentType: "image/png",
			ContentURL:  "https://example.test/image.png",
			Name:        "image.png",
		}},
	}
	items, err := mapTeamsActivity(messageActivity, cfg, time.Time{})
	if err != nil {
		t.Fatalf("mapTeamsActivity(message) error = %v", err)
	}
	if got, want := len(items), 1; got != want {
		t.Fatalf("len(items) = %d, want %d", got, want)
	}
	message := items[0].Envelope
	if got, want := message.EventFamily, bridgepkg.InboundEventFamilyMessage; got != want {
		t.Fatalf("message.EventFamily = %q, want %q", got, want)
	}
	if got, want := message.GroupID, "19:channel@thread.tacv2"; got != want {
		t.Fatalf("message.GroupID = %q, want %q", got, want)
	}
	if got, want := len(message.Attachments), 1; got != want {
		t.Fatalf("len(message.Attachments) = %d, want %d", got, want)
	}
	if _, err := decodeTeamsThreadID(message.ThreadID); err != nil {
		t.Fatalf("decodeTeamsThreadID(message.ThreadID) error = %v", err)
	}

	actionActivity := teamsActivity{
		Type:       "message",
		ID:         "activity-2",
		Timestamp:  "2026-04-15T18:05:01Z",
		ServiceURL: teamsDefaultServiceURL,
		From:       teamsChannelAccount{ID: "29:user-1", Name: "Alice Example"},
		Recipient:  teamsChannelAccount{ID: "28:bot", Name: "Bridge Bot"},
		Conversation: teamsConversation{
			ID:       "a:direct-conversation",
			TenantID: "11111111-2222-3333-4444-555555555555",
		},
		Value: json.RawMessage(`{"actionId":"approve","value":"yes"}`),
	}
	items, err = mapTeamsActivity(actionActivity, cfg, time.Time{})
	if err != nil {
		t.Fatalf("mapTeamsActivity(message action) error = %v", err)
	}
	if got, want := items[0].Envelope.EventFamily, bridgepkg.InboundEventFamilyAction; got != want {
		t.Fatalf("items[0].Envelope.EventFamily = %q, want %q", got, want)
	}
	if got, want := items[0].Envelope.Action.ActionID, "approve"; got != want {
		t.Fatalf("items[0].Envelope.Action.ActionID = %q, want %q", got, want)
	}
	if got, want := items[0].Envelope.PeerID, "a:direct-conversation"; got != want {
		t.Fatalf("items[0].Envelope.PeerID = %q, want %q", got, want)
	}

	invokeActivity := actionActivity
	invokeActivity.Type = "invoke"
	invokeActivity.ID = "activity-3"
	invokeActivity.Value = json.RawMessage(
		`{"action":{"data":{"actionId":"escalate","value":"high"}}}`,
	)
	items, err = mapTeamsActivity(invokeActivity, cfg, time.Time{})
	if err != nil {
		t.Fatalf("mapTeamsActivity(invoke action) error = %v", err)
	}
	if got, want := items[0].Envelope.Action.ActionID, "escalate"; got != want {
		t.Fatalf("items[0].Envelope.Action.ActionID = %q, want %q", got, want)
	}

	reactionActivity := teamsActivity{
		Type:       "messageReaction",
		ID:         "activity-4",
		Timestamp:  "2026-04-15T18:05:02Z",
		ServiceURL: teamsDefaultServiceURL,
		From:       teamsChannelAccount{ID: "29:user-1", Name: "Alice Example"},
		Conversation: teamsConversation{
			ID:               "19:channel@thread.tacv2;messageid=activity-1",
			ConversationType: "channel",
		},
		ReactionsAdded:   []teamsMessageReaction{{Type: "like"}},
		ReactionsRemoved: []teamsMessageReaction{{Type: "sad"}},
	}
	items, err = mapTeamsActivity(reactionActivity, cfg, time.Time{})
	if err != nil {
		t.Fatalf("mapTeamsActivity(reaction) error = %v", err)
	}
	if got, want := len(items), 2; got != want {
		t.Fatalf("len(reaction items) = %d, want %d", got, want)
	}
	if got, want := items[0].Envelope.Reaction.Added, true; got != want {
		t.Fatalf("items[0].Envelope.Reaction.Added = %t, want %t", got, want)
	}
	if got, want := items[1].Envelope.Reaction.Added, false; got != want {
		t.Fatalf("items[1].Envelope.Reaction.Added = %t, want %t", got, want)
	}

	user := teamsUserIdentity{
		ID:          "29:user-1",
		Username:    "alice example",
		DisplayName: "Alice Example",
	}
	if !allowTeamsDirectMessage(
		resolvedInstanceConfig{dmPolicy: bridgepkg.BridgeDMPolicyOpen},
		user,
		true,
	) {
		t.Fatal("allowTeamsDirectMessage(open) = false, want true")
	}
	if !allowTeamsDirectMessage(resolvedInstanceConfig{
		dmPolicy:       bridgepkg.BridgeDMPolicyAllowlist,
		allowUserIDs:   map[string]struct{}{"29:user-1": {}},
		allowUsernames: map[string]struct{}{"alice example": {}},
	}, user, true) {
		t.Fatal("allowTeamsDirectMessage(allowlist) = false, want true")
	}
	if !allowTeamsDirectMessage(resolvedInstanceConfig{
		dmPolicy:        bridgepkg.BridgeDMPolicyPairing,
		pairedUsernames: map[string]struct{}{"alice example": {}},
	}, user, true) {
		t.Fatal("allowTeamsDirectMessage(pairing) = false, want true")
	}
	if allowTeamsDirectMessage(
		resolvedInstanceConfig{dmPolicy: bridgepkg.BridgeDMPolicyAllowlist},
		user,
		true,
	) {
		t.Fatal("allowTeamsDirectMessage(rejected) = true, want false")
	}
}

func TestExecuteTeamsDeliveryConversationAndProactiveDM(t *testing.T) {
	t.Parallel()

	api := &fakeTeamsAPI{
		createConversationID: "a:created-conversation",
		nextActivityID:       700,
	}
	cfg := resolvedInstanceConfig{
		instanceID:  "brg-teams",
		serviceURL:  "https://smba.trafficmanager.net/teams/",
		appID:       "app-id",
		appTenantID: "11111111-2222-3333-4444-555555555555",
	}
	threadID := encodeTeamsThreadID(teamsThreadRef{
		ConversationID: "19:channel@thread.tacv2;messageid=activity-parent",
		ServiceURL:     "https://smba.trafficmanager.net/teams/",
	})
	startReq := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "delivery-1",
			BridgeInstanceID: "brg-teams",
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-teams",
				BridgeInstanceID: "brg-teams",
				GroupID:          "19:channel@thread.tacv2",
				ThreadID:         threadID,
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: "brg-teams",
				GroupID:          "19:channel@thread.tacv2",
				ThreadID:         threadID,
				Mode:             bridgepkg.DeliveryModeReply,
			},
			Seq:       1,
			EventType: bridgepkg.DeliveryEventTypeStart,
			Content:   bridgepkg.MessageContent{Text: "hello"},
		},
	}

	startAck, state, err := executeTeamsDelivery(
		context.Background(),
		api,
		cfg,
		startReq,
		deliveryState{},
		func(string, string) (teamsUserContext, bool) {
			return teamsUserContext{}, false
		},
	)
	if err != nil {
		t.Fatalf("executeTeamsDelivery(start) error = %v", err)
	}
	if got, want := len(api.sendCalls), 1; got != want {
		t.Fatalf("len(api.sendCalls) = %d, want %d", got, want)
	}
	if got, want := api.sendCalls[0].ReplyToID, "activity-parent"; got != want {
		t.Fatalf("api.sendCalls[0].ReplyToID = %q, want %q", got, want)
	}

	finalReq := startReq
	finalReq.Event.Seq = 2
	finalReq.Event.EventType = bridgepkg.DeliveryEventTypeFinal
	finalReq.Event.Final = true
	finalReq.Event.Content.Text = "hello world"
	finalAck, state, err := executeTeamsDelivery(
		context.Background(),
		api,
		cfg,
		finalReq,
		state,
		func(string, string) (teamsUserContext, bool) {
			return teamsUserContext{}, false
		},
	)
	if err != nil {
		t.Fatalf("executeTeamsDelivery(final) error = %v", err)
	}
	if got, want := len(api.updateCalls), 1; got != want {
		t.Fatalf("len(api.updateCalls) = %d, want %d", got, want)
	}
	if got, want := finalAck.ReplaceRemoteMessageID, startAck.RemoteMessageID; got != want {
		t.Fatalf("finalAck.ReplaceRemoteMessageID = %q, want %q", got, want)
	}

	deleteReq := finalReq
	deleteReq.Event.Seq = 3
	deleteReq.Event.EventType = bridgepkg.DeliveryEventTypeDelete
	deleteReq.Event.Operation = bridgepkg.DeliveryOperationDelete
	deleteReq.Event.Reference = &bridgepkg.DeliveryMessageReference{
		RemoteMessageID: finalAck.RemoteMessageID,
	}
	deleteReq.Event.Content.Text = ""
	_, _, err = executeTeamsDelivery(
		context.Background(),
		api,
		cfg,
		deleteReq,
		state,
		func(string, string) (teamsUserContext, bool) {
			return teamsUserContext{}, false
		},
	)
	if err != nil {
		t.Fatalf("executeTeamsDelivery(delete) error = %v", err)
	}
	if got, want := len(api.deleteCalls), 1; got != want {
		t.Fatalf("len(api.deleteCalls) = %d, want %d", got, want)
	}

	proactiveReq := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "delivery-2",
			BridgeInstanceID: "brg-teams",
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-teams",
				BridgeInstanceID: "brg-teams",
				PeerID:           "29:user-2",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: "brg-teams",
				PeerID:           "29:user-2",
				Mode:             bridgepkg.DeliveryModeDirectSend,
			},
			Seq:       1,
			EventType: bridgepkg.DeliveryEventTypeStart,
			Content:   bridgepkg.MessageContent{Text: "ping"},
		},
	}
	_, _, err = executeTeamsDelivery(
		context.Background(),
		api,
		cfg,
		proactiveReq,
		deliveryState{},
		func(instanceID string, userID string) (teamsUserContext, bool) {
			if instanceID != "brg-teams" || userID != "29:user-2" {
				return teamsUserContext{}, false
			}
			return teamsUserContext{
				ServiceURL: "https://smba.trafficmanager.net/teams/",
				TenantID:   "11111111-2222-3333-4444-555555555555",
			}, true
		},
	)
	if err != nil {
		t.Fatalf("executeTeamsDelivery(proactive) error = %v", err)
	}
	if got, want := len(api.createCalls), 1; got != want {
		t.Fatalf("len(api.createCalls) = %d, want %d", got, want)
	}

	resumeReq := proactiveReq
	resumeReq.Event.EventType = bridgepkg.DeliveryEventTypeResume
	resumeReq.Event.Resume = &bridgepkg.DeliveryResumeState{
		LatestEventType: bridgepkg.DeliveryEventTypeFinal,
	}
	resumeReq.Snapshot = &bridgepkg.DeliverySnapshot{
		DeliveryID:       "delivery-2",
		SessionID:        "sess-1",
		TurnID:           "turn-1",
		BridgeInstanceID: "brg-teams",
		RoutingKey:       proactiveReq.Event.RoutingKey,
		DeliveryTarget:   proactiveReq.Event.DeliveryTarget,
		LatestSeq:        1,
		LastSentSeq:      1,
		LastAckedSeq:     1,
		LatestEventType:  bridgepkg.DeliveryEventTypeFinal,
		CurrentContent:   bridgepkg.MessageContent{Text: "ping"},
		RemoteMessageID:  startAck.RemoteMessageID,
		Final:            true,
		UpdatedAt:        time.Date(2026, 4, 15, 18, 10, 0, 0, time.UTC),
	}
	resumeAck, _, err := executeTeamsDelivery(
		context.Background(),
		api,
		cfg,
		resumeReq,
		deliveryState{},
		func(string, string) (teamsUserContext, bool) {
			return teamsUserContext{}, false
		},
	)
	if err != nil {
		t.Fatalf("executeTeamsDelivery(resume) error = %v", err)
	}
	if got, want := resumeAck.RemoteMessageID, startAck.RemoteMessageID; got != want {
		t.Fatalf("resumeAck.RemoteMessageID = %q, want %q", got, want)
	}
}

func TestResolveInstanceConfigAndDetermineInitialState(t *testing.T) {
	mock := newTeamsProviderServer(t, teamsProviderServerConfig{})
	env := setProviderTestEnv(t)
	_ = env
	listenAddr := reserveListenAddr(t)
	t.Setenv(teamsListenAddrEnv, listenAddr)
	t.Setenv(teamsOpenIDMetadataURLEnv, mock.MetadataURL())
	t.Setenv(teamsOAuthTokenURLEnvName(), mock.TokenURL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 18, 0, 0, 0, time.UTC)
	managed := testTeamsManagedInstance(t, now, "brg-1", map[string]any{
		"service_url": mock.ServiceURL(),
		"webhook": map[string]any{
			"listen_addr": listenAddr,
			"path":        "teams",
		},
		"auth": map[string]any{
			"openid_metadata_url": mock.MetadataURL(),
			"token_url":           mock.TokenURL(),
		},
		"batching": map[string]any{
			"delay_ms":        5,
			"split_delay_ms":  7,
			"split_threshold": 2,
		},
		"dm": map[string]any{
			"allow_user_ids":   []string{"29:user-1"},
			"allow_usernames":  []string{"Alice Example"},
			"paired_usernames": []string{"Bob Example"},
		},
	})

	mustHandleLifecycle(t, hostPeer, managed)
	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	waitForCondition(t, func() bool {
		runtime.mu.RLock()
		defer runtime.mu.RUnlock()
		return runtime.server != nil && strings.TrimSpace(runtime.serverAddr) != ""
	})

	session := runtime.currentSession()
	if session == nil {
		t.Fatal("runtime.currentSession() = nil, want session")
	}

	cfg := runtime.resolveInstanceConfig(session, managed)
	if cfg.configError != nil {
		t.Fatalf("resolveInstanceConfig() configError = %v, want nil", cfg.configError)
	}
	defer cfg.batcher.Close()
	if got, want := cfg.serviceURL, mock.ServiceURL(); got != want {
		t.Fatalf("cfg.serviceURL = %q, want %q", got, want)
	}
	if got, want := cfg.openIDMetadataURL, mock.MetadataURL(); got != want {
		t.Fatalf("cfg.openIDMetadataURL = %q, want %q", got, want)
	}
	if got, want := cfg.tokenURL, mock.TokenURL(); got != want {
		t.Fatalf("cfg.tokenURL = %q, want %q", got, want)
	}
	if got, want := cfg.appTenantID, "11111111-2222-3333-4444-555555555555"; got != want {
		t.Fatalf("cfg.appTenantID = %q, want %q", got, want)
	}
	if cfg.batcher == nil {
		t.Fatal("cfg.batcher = nil, want non-nil")
	}

	status, degradation, err := runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID:        "bad-config",
			configError:       errors.New("bad config"),
			serviceURL:        mock.ServiceURL(),
			openIDMetadataURL: mock.MetadataURL(),
			tokenURL:          mock.TokenURL(),
		},
	)
	if err == nil {
		t.Fatal("determineInitialState(configError) error = nil, want non-nil")
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
		resolvedInstanceConfig{
			instanceID:        "missing-auth",
			serviceURL:        mock.ServiceURL(),
			openIDMetadataURL: mock.MetadataURL(),
			tokenURL:          mock.TokenURL(),
		},
	)
	if err == nil {
		t.Fatal("determineInitialState(missing auth) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusAuthRequired; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil || degradation.Reason != bridgepkg.BridgeDegradationReasonAuthFailed {
		t.Fatalf("degradation = %#v, want auth failed", degradation)
	}

	runtime.apiFactory = func(resolvedInstanceConfig) teamsAPI {
		return &fakeTeamsAPI{validateErr: &bridgesdk.AuthError{Err: errors.New("bad token")}}
	}
	status, degradation, err = runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID:        "bad-auth",
			serviceURL:        mock.ServiceURL(),
			openIDMetadataURL: mock.MetadataURL(),
			tokenURL:          mock.TokenURL(),
			appID:             "app-id",
			appPassword:       "app-password",
		},
	)
	if err == nil {
		t.Fatal("determineInitialState(auth error) error = nil, want non-nil")
	}
	if got, want := status, bridgepkg.BridgeStatusAuthRequired; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation == nil || degradation.Reason != bridgepkg.BridgeDegradationReasonAuthFailed {
		t.Fatalf("degradation = %#v, want auth failed classification", degradation)
	}

	runtime.apiFactory = func(resolvedInstanceConfig) teamsAPI {
		return &fakeTeamsAPI{}
	}
	status, degradation, err = runtime.determineInitialState(
		context.Background(),
		resolvedInstanceConfig{
			instanceID:        "ready",
			serviceURL:        mock.ServiceURL(),
			openIDMetadataURL: mock.MetadataURL(),
			tokenURL:          mock.TokenURL(),
			appID:             "app-id",
			appPassword:       "app-password",
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
}

func TestRuntimeInitializeStartsServerAndWritesMarkers(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mock := newTeamsProviderServer(t, teamsProviderServerConfig{})
	t.Setenv(teamsListenAddrEnv, listenAddr)
	t.Setenv(teamsOpenIDMetadataURLEnv, mock.MetadataURL())
	t.Setenv(teamsOAuthTokenURLEnvName(), mock.TokenURL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 18, 20, 0, 0, time.UTC)
	managed := []subprocess.InitializeBridgeManagedInstance{
		testTeamsManagedInstance(
			t,
			now,
			"brg-1",
			map[string]any{
				"service_url": mock.ServiceURL(),
				"auth": map[string]any{
					"openid_metadata_url": mock.MetadataURL(),
					"token_url":           mock.TokenURL(),
				},
			},
		),
		testTeamsManagedInstance(
			t,
			now,
			"brg-2",
			map[string]any{
				"service_url": mock.ServiceURL(),
				"auth": map[string]any{
					"openid_metadata_url": mock.MetadataURL(),
					"token_url":           mock.TokenURL(),
				},
			},
		),
	}
	mustHandleLifecycle(t, hostPeer, managed...)

	if err := hostPeer.Call(
		context.Background(),
		"initialize",
		testInitializeRequest(now, managed...),
		nil,
	); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}

	handshake := waitForJSONFile[initializeMarker](t, env.handshakePath)
	if got, want := handshake.Request.Runtime.Bridge.Provider, "teams"; got != want {
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

func TestWebhookAuthorizationRejectsInvalidTokenAndIngestsActivities(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mock := newTeamsProviderServer(t, teamsProviderServerConfig{})
	t.Setenv(teamsListenAddrEnv, listenAddr)
	t.Setenv(teamsOpenIDMetadataURLEnv, mock.MetadataURL())
	t.Setenv(teamsOAuthTokenURLEnvName(), mock.TokenURL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 18, 25, 0, 0, time.UTC)
	managed := testTeamsManagedInstance(t, now, "brg-1", map[string]any{
		"service_url": mock.ServiceURL(),
		"auth": map[string]any{
			"openid_metadata_url": mock.MetadataURL(),
			"token_url":           mock.TokenURL(),
		},
	})
	mustHandleLifecycle(t, hostPeer, managed)

	var ingested []bridgepkg.InboundMessageEnvelope
	var mu sync.Mutex
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
					GroupID:          envelope.GroupID,
					ThreadID:         envelope.ThreadID,
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
	webhookURL := "http://" + serverAddr + "/teams/brg-1"

	invalidReq, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		webhookURL,
		strings.NewReader(teamsMessageWebhook(mock.ServiceURL(), "Need a summary")),
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(invalid) error = %v", err)
	}
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.Header.Set("Authorization", "Bearer bad-token")
	invalidResp, err := http.DefaultClient.Do(invalidReq)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(invalid) error = %v", err)
	}
	_ = invalidResp.Body.Close()
	if got, want := invalidResp.StatusCode, http.StatusUnauthorized; got != want {
		t.Fatalf("invalid webhook status = %d, want %d", got, want)
	}

	postTeamsWebhook(
		t,
		mock,
		webhookURL,
		"app-id",
		teamsMessageWebhook(mock.ServiceURL(), "Need a summary"),
	)
	postTeamsWebhook(t, mock, webhookURL, "app-id", teamsInvokeWebhook(mock.ServiceURL()))
	postTeamsWebhook(t, mock, webhookURL, "app-id", teamsReactionWebhook(mock.ServiceURL()))

	ingests := waitForJSONLinesFile[ingestMarker](
		t,
		env.ingestPath,
		func(items []ingestMarker) bool {
			return len(items) >= 4
		},
	)
	if got, want := ingests[0].Envelope.EventFamily, bridgepkg.InboundEventFamilyMessage; got != want {
		t.Fatalf("ingests[0].Envelope.EventFamily = %q, want %q", got, want)
	}
	families := map[bridgepkg.InboundEventFamily]int{}
	for _, item := range ingests {
		families[item.Envelope.EventFamily]++
	}
	if families[bridgepkg.InboundEventFamilyMessage] == 0 ||
		families[bridgepkg.InboundEventFamilyAction] == 0 ||
		families[bridgepkg.InboundEventFamilyReaction] == 0 {
		t.Fatalf("families = %#v, want message/action/reaction coverage", families)
	}
	mu.Lock()
	if got, want := len(ingested), len(ingests); got != want {
		t.Fatalf("len(ingested) = %d, want %d", got, want)
	}
	mu.Unlock()
}

func TestRuntimeDeliveriesCallTeamsAPI(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mock := newTeamsProviderServer(t, teamsProviderServerConfig{})
	t.Setenv(teamsListenAddrEnv, listenAddr)
	t.Setenv(teamsOpenIDMetadataURLEnv, mock.MetadataURL())
	t.Setenv(teamsOAuthTokenURLEnvName(), mock.TokenURL())

	_, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 18, 30, 0, 0, time.UTC)
	managed := testTeamsManagedInstance(t, now, "brg-1", map[string]any{
		"service_url": mock.ServiceURL(),
		"auth": map[string]any{
			"openid_metadata_url": mock.MetadataURL(),
			"token_url":           mock.TokenURL(),
		},
	})
	mustHandleLifecycle(t, hostPeer, managed)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}

	threadID := encodeTeamsThreadID(teamsThreadRef{
		ConversationID: "19:channel@thread.tacv2;messageid=activity-parent",
		ServiceURL:     mock.ServiceURL(),
	})
	startReq := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "delivery-1",
			BridgeInstanceID: "brg-1",
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-teams",
				BridgeInstanceID: "brg-1",
				GroupID:          "19:channel@thread.tacv2",
				ThreadID:         threadID,
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: "brg-1",
				GroupID:          "19:channel@thread.tacv2",
				ThreadID:         threadID,
				Mode:             bridgepkg.DeliveryModeReply,
			},
			Seq:       1,
			EventType: bridgepkg.DeliveryEventTypeStart,
			Content:   bridgepkg.MessageContent{Text: "hello"},
		},
	}
	var startAck bridgepkg.DeliveryAck
	if err := hostPeer.Call(context.Background(), "bridges/deliver", startReq, &startAck); err != nil {
		t.Fatalf("hostPeer.Call(start delivery) error = %v", err)
	}
	finalReq := startReq
	finalReq.Event.Seq = 2
	finalReq.Event.EventType = bridgepkg.DeliveryEventTypeFinal
	finalReq.Event.Final = true
	finalReq.Event.Content.Text = "hello world"
	var finalAck bridgepkg.DeliveryAck
	if err := hostPeer.Call(context.Background(), "bridges/deliver", finalReq, &finalAck); err != nil {
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
	if got, want := finalAck.ReplaceRemoteMessageID, startAck.RemoteMessageID; got != want {
		t.Fatalf("finalAck.ReplaceRemoteMessageID = %q, want %q", got, want)
	}
	if got, want := len(mock.APICalls()), 2; got < want {
		t.Fatalf("len(mock.APICalls()) = %d, want at least %d", got, want)
	}
}

func TestClassifyTeamsHTTPErrorAndHelpers(t *testing.T) {
	t.Parallel()

	rate := classifyTeamsHTTPError(http.StatusTooManyRequests, "5", "slow down")
	var rateErr *bridgesdk.RateLimitError
	if !errors.As(rate, &rateErr) {
		t.Fatalf("classifyTeamsHTTPError(rate) = %T, want *RateLimitError", rate)
	}
	if got, want := rateErr.RetryAfter, 5*time.Second; got != want {
		t.Fatalf("rateErr.RetryAfter = %s, want %s", got, want)
	}

	auth := classifyTeamsHTTPError(http.StatusUnauthorized, "", "bad token")
	var authErr *bridgesdk.AuthError
	if !errors.As(auth, &authErr) {
		t.Fatalf("classifyTeamsHTTPError(auth) = %T, want *AuthError", auth)
	}

	if !looksLikeTeamsUserID("29:user-1") {
		t.Fatal("looksLikeTeamsUserID(29:user-1) = false, want true")
	}
	if looksLikeTeamsUserID("19:channel@thread.tacv2") {
		t.Fatal("looksLikeTeamsUserID(channel) = true, want false")
	}
	if !looksLikeTenantID("11111111-2222-3333-4444-555555555555") {
		t.Fatal("looksLikeTenantID(valid uuid) = false, want true")
	}
	if !validTeamsServiceURL("http://127.0.0.1:3000") {
		t.Fatal("validTeamsServiceURL(loopback http) = false, want true")
	}
	if validTeamsServiceURL("http://example.test") {
		t.Fatal("validTeamsServiceURL(http) = true, want false")
	}
}

func TestProviderHelperStateAndShutdown(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)

	runtime, err := newTeamsProvider(io.Discard)
	if err != nil {
		t.Fatalf("newTeamsProvider() error = %v", err)
	}
	if err := runtime.startServer(listenAddr); err != nil {
		t.Fatalf("startServer() error = %v", err)
	}
	if err := runtime.healthCheck(); err != nil {
		t.Fatalf("healthCheck() error = %v, want nil", err)
	}

	runtime.setLastError(errors.New("boom"))
	if err := runtime.healthCheck(); err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("healthCheck() error = %v, want boom", err)
	}
	runtime.clearLastError()
	if err := runtime.healthCheck(); err != nil {
		t.Fatalf("healthCheck() after clear error = %v, want nil", err)
	}

	runtime.storeUserContext("brg-1", teamsActivity{
		From:       teamsChannelAccount{ID: "29:user-1"},
		ServiceURL: "https://service.test",
		Conversation: teamsConversation{
			TenantID: "tenant-1",
		},
	})
	if ctx, ok := runtime.userContext("brg-1", "29:user-1"); !ok || ctx.TenantID != "tenant-1" {
		t.Fatalf("userContext() = (%#v, %t), want tenant-1", ctx, ok)
	}
	if _, ok := runtime.userContext("brg-1", "missing"); ok {
		t.Fatal("userContext(missing) = true, want false")
	}

	degradation := &bridgepkg.BridgeDegradation{
		Reason:  bridgepkg.BridgeDegradationReasonAuthFailed,
		Message: "bad auth",
	}
	cloned := cloneDegradation(degradation)
	if cloned == nil || cloned == degradation || cloned.Message != degradation.Message {
		t.Fatalf("cloneDegradation() = %#v, want independent clone of %#v", cloned, degradation)
	}
	if cloneDegradation(nil) != nil {
		t.Fatal("cloneDegradation(nil) != nil")
	}

	if !isNotInitializedRPCError(subprocess.NewRPCError(rpcCodeNotInitialized, "not ready", nil)) {
		t.Fatal("isNotInitializedRPCError() = false, want true")
	}
	if isNotInitializedRPCError(errors.New("boom")) {
		t.Fatal("isNotInitializedRPCError(non-rpc) = true, want false")
	}

	done := make(chan struct{})
	runtime.wg.Go(func() {
		<-runtime.stopCh
		close(done)
	})

	if err := runtime.handleShutdown(
		context.Background(),
		nil,
		subprocess.ShutdownRequest{DeadlineMS: 250},
	); err != nil {
		t.Fatalf("handleShutdown() error = %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not close stopCh before timeout")
	}

	waitForCondition(t, func() bool {
		data, err := os.ReadFile(env.shutdownPath)
		return err == nil && strings.Contains(string(data), "pid=")
	})
}

func TestRetryHostCallCoverage(t *testing.T) {
	runtime, err := newTeamsProvider(io.Discard)
	if err != nil {
		t.Fatalf("newTeamsProvider() error = %v", err)
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
		t.Fatalf("retryHostCall(success after retries) error = %v", err)
	}
	if got, want := attempts, 3; got != want {
		t.Fatalf("attempts = %d, want %d", got, want)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	err = runtime.retryHostCall(canceled, func(context.Context) error {
		return subprocess.NewRPCError(rpcCodeNotInitialized, "not ready", nil)
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("retryHostCall(canceled) error = %v, want context.Canceled", err)
	}

	stopped, err := newTeamsProvider(io.Discard)
	if err != nil {
		t.Fatalf("newTeamsProvider(stopped) error = %v", err)
	}
	stopped.stop()
	expected := subprocess.NewRPCError(rpcCodeNotInitialized, "not ready", nil)
	err = stopped.retryHostCall(context.Background(), func(context.Context) error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("retryHostCall(stopped) error = %v, want %v", err, expected)
	}

	permanent := errors.New("permanent")
	err = runtime.retryHostCall(context.Background(), func(context.Context) error {
		return permanent
	})
	if !errors.Is(err, permanent) {
		t.Fatalf("retryHostCall(permanent) error = %v, want %v", err, permanent)
	}
}

func TestDispatchInboundBatchAndEnvelopeCoverage(t *testing.T) {
	env := setProviderTestEnv(t)
	_ = env
	listenAddr := reserveListenAddr(t)
	mock := newTeamsProviderServer(t, teamsProviderServerConfig{})
	t.Setenv(teamsListenAddrEnv, listenAddr)
	t.Setenv(teamsOpenIDMetadataURLEnv, mock.MetadataURL())
	t.Setenv(teamsOAuthTokenURLEnvName(), mock.TokenURL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 19, 20, 0, 0, time.UTC)
	managed := testTeamsManagedInstance(t, now, "brg-1", map[string]any{
		"service_url": mock.ServiceURL(),
		"auth": map[string]any{
			"openid_metadata_url": mock.MetadataURL(),
			"token_url":           mock.TokenURL(),
		},
	})
	mustHandleLifecycle(t, hostPeer, managed)

	var ingested []bridgepkg.InboundMessageEnvelope
	var mu sync.Mutex
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
					GroupID:          envelope.GroupID,
					ThreadID:         envelope.ThreadID,
				},
			}, nil
		},
	)

	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	waitForCondition(t, func() bool {
		_, err := runtime.configForInstance("brg-1")
		return err == nil
	})

	envelope := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID: "brg-1",
		Scope:            bridgepkg.ScopeWorkspace,
		WorkspaceID:      "ws-teams",
		GroupID:          "19:channel@thread.tacv2",
		ThreadID: encodeTeamsThreadID(
			teamsThreadRef{
				ConversationID: "19:channel@thread.tacv2;messageid=activity-1",
				ServiceURL:     mock.ServiceURL(),
			},
		),
		PlatformMessageID: "activity-1",
		ReceivedAt:        now,
		Sender: bridgepkg.MessageSender{
			ID:          "29:user-1",
			Username:    "alice",
			DisplayName: "Alice",
		},
		Content:        bridgepkg.MessageContent{Text: "first"},
		EventFamily:    bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey: "idem-1",
	}

	if err := runtime.dispatchInboundBatch(context.Background(), "brg-1", bridgesdk.InboundBatch{}); err != nil {
		t.Fatalf("dispatchInboundBatch(empty) error = %v", err)
	}
	if err := runtime.dispatchInboundBatch(context.Background(), "brg-1", bridgesdk.InboundBatch{
		Items: []bridgepkg.InboundMessageEnvelope{
			envelope,
			func() bridgepkg.InboundMessageEnvelope {
				clone := envelope
				clone.PlatformMessageID = "activity-2"
				clone.Content.Text = "second"
				clone.IdempotencyKey = "idem-2"
				return clone
			}(),
		},
	}); err != nil {
		t.Fatalf("dispatchInboundBatch(merged) error = %v", err)
	}

	waitForCondition(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(ingested) >= 1
	})
	mu.Lock()
	got := ingested[len(ingested)-1]
	mu.Unlock()
	if got.Content.Text != "first\nsecond" {
		t.Fatalf("merged text = %q, want %q", got.Content.Text, "first\nsecond")
	}
	if !strings.Contains(got.IdempotencyKey, ":batch:2") {
		t.Fatalf("merged idempotency key = %q, want batch suffix", got.IdempotencyKey)
	}

	uninitialized, err := newTeamsProvider(io.Discard)
	if err != nil {
		t.Fatalf("newTeamsProvider(uninitialized) error = %v", err)
	}
	if err := uninitialized.dispatchInboundEnvelope(
		context.Background(),
		"brg-1",
		envelope,
	); err == nil ||
		!strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("dispatchInboundEnvelope(uninitialized) error = %v, want not initialized", err)
	}
}

func TestBotClientCoverageAndWebhookGuards(t *testing.T) {
	env := setProviderTestEnv(t)
	listenAddr := reserveListenAddr(t)
	mock := newTeamsProviderServer(t, teamsProviderServerConfig{})
	t.Setenv(teamsListenAddrEnv, listenAddr)
	t.Setenv(teamsOpenIDMetadataURLEnv, mock.MetadataURL())
	t.Setenv(teamsOAuthTokenURLEnvName(), mock.TokenURL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 19, 30, 0, 0, time.UTC)
	managed := testTeamsManagedInstance(t, now, "brg-1", map[string]any{
		"service_url": mock.ServiceURL(),
		"auth": map[string]any{
			"openid_metadata_url": mock.MetadataURL(),
			"token_url":           mock.TokenURL(),
		},
	})
	mustHandleLifecycle(t, hostPeer, managed)
	if err := hostPeer.Call(context.Background(), "initialize", testInitializeRequest(now, managed), nil); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	waitForCondition(t, func() bool {
		runtime.mu.RLock()
		defer runtime.mu.RUnlock()
		return strings.TrimSpace(runtime.serverAddr) != ""
	})

	webhookURL := fmt.Sprintf("http://%s/teams/%s", listenAddr, managed.Instance.ID)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodGet,
		webhookURL,
		http.NoBody,
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(GET) error = %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(GET) error = %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if got, want := resp.StatusCode, http.StatusMethodNotAllowed; got != want {
		t.Fatalf("GET webhook status = %d, want %d", got, want)
	}

	req, err = http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://"+listenAddr+"/unknown",
		strings.NewReader(`{}`),
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(unknown) error = %v", err)
	}
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("http.DefaultClient.Do(unknown) error = %v", err)
	}
	_, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if got, want := resp.StatusCode, http.StatusNotFound; got != want {
		t.Fatalf("unknown path status = %d, want %d", got, want)
	}

	cfg := resolvedInstanceConfig{
		appID:             "app-id",
		appPassword:       "app-password",
		serviceURL:        mock.ServiceURL(),
		tokenURL:          mock.TokenURL(),
		openIDMetadataURL: mock.MetadataURL(),
	}
	client := &teamsBotClient{cfg: cfg, httpClient: http.DefaultClient}
	created, err := client.CreateConversation(
		context.Background(),
		mock.ServiceURL(),
		teamsCreateConversationRequest{
			Bot:      teamsChannelAccount{ID: "28:bot"},
			IsGroup:  false,
			Members:  []teamsChannelAccount{{ID: "29:user-1"}},
			TenantID: "11111111-2222-3333-4444-555555555555",
		},
	)
	if err != nil {
		t.Fatalf("CreateConversation() error = %v", err)
	}
	if got, want := created.ID, "a:created-conversation"; got != want {
		t.Fatalf("CreateConversation().ID = %q, want %q", got, want)
	}
	if err := client.DeleteActivity(
		context.Background(),
		mock.ServiceURL(),
		"conversation-1",
		"activity-1",
	); err != nil {
		t.Fatalf("DeleteActivity() error = %v", err)
	}

	if got, want := readResponseBody(io.NopCloser(strings.NewReader(" body "))), "body"; got != want {
		t.Fatalf("readResponseBody() = %q, want %q", got, want)
	}

	calls := mock.APICalls()
	if len(calls) == 0 {
		t.Fatal("mock.APICalls() = 0, want recorded bot client calls")
	}
	if len(calls) < 2 {
		t.Fatalf("len(mock.APICalls()) = %d, want at least 2", len(calls))
	}

	waitForCondition(t, func() bool {
		data, err := os.ReadFile(env.startsPath)
		return err == nil && strings.Contains(string(data), "listen=")
	})
}

func TestHandleBridgesDeliverCoverageAndRunCommand(t *testing.T) {
	env := setProviderTestEnv(t)
	_ = env
	listenAddr := reserveListenAddr(t)
	mock := newTeamsProviderServer(t, teamsProviderServerConfig{})
	t.Setenv(teamsListenAddrEnv, listenAddr)
	t.Setenv(teamsOpenIDMetadataURLEnv, mock.MetadataURL())
	t.Setenv(teamsOAuthTokenURLEnvName(), mock.TokenURL())

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 19, 40, 0, 0, time.UTC)
	managed := testTeamsManagedInstance(t, now, "brg-1", map[string]any{
		"service_url": mock.ServiceURL(),
		"auth": map[string]any{
			"openid_metadata_url": mock.MetadataURL(),
			"token_url":           mock.TokenURL(),
		},
	})
	mustHandleLifecycle(t, hostPeer, managed)
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
	waitForCondition(t, func() bool {
		_, err := runtime.configForInstance("brg-1")
		return err == nil && runtime.currentSession() != nil
	})

	api := &fakeTeamsAPI{nextActivityID: 800}
	runtime.apiFactory = func(resolvedInstanceConfig) teamsAPI { return api }
	session := runtime.currentSession()
	if session == nil {
		t.Fatal("runtime.currentSession() = nil, want session")
	}

	threadID := encodeTeamsThreadID(teamsThreadRef{
		ConversationID: "19:channel@thread.tacv2;messageid=activity-parent",
		ServiceURL:     mock.ServiceURL(),
	})
	req := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "delivery-1",
			BridgeInstanceID: "brg-1",
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-teams",
				BridgeInstanceID: "brg-1",
				GroupID:          "19:channel@thread.tacv2",
				ThreadID:         threadID,
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: "brg-1",
				GroupID:          "19:channel@thread.tacv2",
				ThreadID:         threadID,
				Mode:             bridgepkg.DeliveryModeReply,
			},
			Seq:       1,
			EventType: bridgepkg.DeliveryEventTypeStart,
			Content:   bridgepkg.MessageContent{Text: "hello"},
		},
	}

	ack, err := runtime.handleBridgesDeliver(context.Background(), session, req)
	if err != nil {
		t.Fatalf("handleBridgesDeliver(success) error = %v", err)
	}
	if strings.TrimSpace(ack.RemoteMessageID) == "" {
		t.Fatal("handleBridgesDeliver(success) remote message id = empty, want non-empty")
	}
	if got, want := len(api.sendCalls), 1; got != want {
		t.Fatalf("len(api.sendCalls) = %d, want %d", got, want)
	}

	badReq := req
	badReq.Event.BridgeInstanceID = "missing"
	_, err = runtime.handleBridgesDeliver(context.Background(), session, badReq)
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf(
			"handleBridgesDeliver(missing instance) error = %v, want missing instance error",
			err,
		)
	}

	if err := run(
		[]string{"unknown"},
		strings.NewReader(""),
		io.Discard,
		io.Discard,
	); err == nil ||
		!strings.Contains(err.Error(), "unsupported command") {
		t.Fatalf("run(unknown) error = %v, want unsupported command", err)
	}
}

func TestTeamsOpenIDAndAuthHelperCoverage(t *testing.T) {
	t.Parallel()

	if _, err := fetchTeamsOpenIDMetadata(context.Background(), ""); err == nil {
		t.Fatal("fetchTeamsOpenIDMetadata(empty) error = nil, want non-nil")
	}

	metadataServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/metadata":
				_ = json.NewEncoder(w).
					Encode(map[string]any{"issuer": "https://api.botframework.com"})
			case "/jwks":
				_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{}})
			default:
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": "missing"})
			}
		}),
	)
	defer metadataServer.Close()

	if _, err := fetchTeamsOpenIDMetadata(
		context.Background(),
		metadataServer.URL+"/metadata",
	); err == nil ||
		!strings.Contains(err.Error(), "jwks_uri") {
		t.Fatalf("fetchTeamsOpenIDMetadata(missing jwks_uri) error = %v, want jwks_uri error", err)
	}
	if _, err := fetchTeamsJWKS(
		context.Background(),
		metadataServer.URL+"/jwks",
	); err == nil ||
		!strings.Contains(err.Error(), "omitted signing keys") {
		t.Fatalf("fetchTeamsJWKS(empty keys) error = %v, want empty key error", err)
	}

	keys := teamsJWKS{Keys: []teamsJWK{{Kid: "known"}}}
	if _, err := keys.keyByID("missing"); err == nil ||
		!strings.Contains(err.Error(), "not found") {
		t.Fatalf("keyByID(missing) error = %v, want not found", err)
	}
	if err := (teamsJWK{Endorsements: []string{"other"}}).validateEndorsement(
		"msteams",
	); err == nil ||
		!strings.Contains(err.Error(), "not endorsed") {
		t.Fatalf("validateEndorsement() error = %v, want endorsement error", err)
	}
	if _, err := (teamsJWK{
		Kty: "RSA",
		N:   base64.RawURLEncoding.EncodeToString([]byte{1}),
		E:   base64.RawURLEncoding.EncodeToString([]byte{0}),
	}).publicKey(); err == nil || !strings.Contains(err.Error(), "exponent") {
		t.Fatalf("publicKey(invalid exponent) error = %v, want exponent error", err)
	}

	mock := newTeamsProviderServer(t, teamsProviderServerConfig{})
	cfg := resolvedInstanceConfig{
		appID:             "app-id",
		serviceURL:        mock.ServiceURL(),
		openIDMetadataURL: mock.MetadataURL(),
	}
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://teams.test/webhook",
		strings.NewReader(teamsMessageWebhook(mock.ServiceURL(), "Need a summary")),
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}
	if err := verifyTeamsAuthorization(
		context.Background(),
		req,
		[]byte(teamsMessageWebhook(mock.ServiceURL(), "Need a summary")),
		cfg,
	); err == nil ||
		!strings.Contains(err.Error(), "bearer authorization") {
		t.Fatalf(
			"verifyTeamsAuthorization(missing auth) error = %v, want bearer authorization error",
			err,
		)
	}

	req, err = http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"http://teams.test/webhook",
		strings.NewReader(teamsMessageWebhook(mock.ServiceURL(), "Need a summary")),
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(mismatch) error = %v", err)
	}
	req.Header.Set(
		"Authorization",
		"Bearer "+mock.SignedToken(t, "app-id", "https://elsewhere.test"),
	)
	if err := verifyTeamsAuthorization(
		context.Background(),
		req,
		[]byte(teamsMessageWebhook(mock.ServiceURL(), "Need a summary")),
		cfg,
	); err == nil ||
		!strings.Contains(err.Error(), "did not match") {
		t.Fatalf(
			"verifyTeamsAuthorization(service url mismatch) error = %v, want mismatch error",
			err,
		)
	}
}

func TestReconcileInstanceConfigCoverage(t *testing.T) {
	t.Setenv(teamsListenAddrEnv, "")

	runtime, hostPeer, cleanup := newRuntimePeerPair(t)
	defer cleanup()

	now := time.Date(2026, 4, 15, 19, 50, 0, 0, time.UTC)
	managed := testTeamsManagedInstance(t, now, "brg-1", map[string]any{
		"service_url": teamsDefaultServiceURL,
		"webhook": map[string]any{
			"path": "shared",
		},
	})
	managed2 := testTeamsManagedInstance(t, now, "brg-2", map[string]any{
		"service_url": teamsDefaultServiceURL,
		"webhook": map[string]any{
			"path": "shared",
		},
	})
	mustHandleLifecycle(t, hostPeer, managed, managed2)
	if err := hostPeer.Call(
		context.Background(),
		"initialize",
		testInitializeRequest(now, managed, managed2),
		nil,
	); err != nil {
		t.Fatalf("hostPeer.Call(initialize) error = %v", err)
	}
	session := runtime.currentSession()
	if session == nil {
		t.Fatal("runtime.currentSession() = nil, want session")
	}

	configs := runtime.reconcileInstanceConfigs(
		context.Background(),
		session,
		[]subprocess.InitializeBridgeManagedInstance{managed, managed2},
	)
	if got, want := len(configs), 2; got != want {
		t.Fatalf("len(configs) = %d, want %d", got, want)
	}
	if configs[0].configError == nil ||
		!strings.Contains(configs[0].configError.Error(), "listen address") {
		t.Fatalf("configs[0].configError = %v, want listen address error", configs[0].configError)
	}
	if configs[1].configError == nil ||
		!strings.Contains(configs[1].configError.Error(), "shared") {
		t.Fatalf(
			"configs[1].configError = %v, want shared path or listen error",
			configs[1].configError,
		)
	}

	badJSON := managed
	badJSON.Instance.ID = "bad-json"
	badJSON.Instance.ProviderConfig = json.RawMessage("{")
	resolved := runtime.resolveInstanceConfig(session, badJSON)
	if resolved.configError == nil ||
		!strings.Contains(resolved.configError.Error(), "decode provider_config") {
		t.Fatalf(
			"resolveInstanceConfig(bad json) error = %v, want decode provider_config error",
			resolved.configError,
		)
	}
}

func TestMarkerHelperCoverage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	linesPath := filepath.Join(dir, "markers", "lines.log")
	jsonPath := filepath.Join(dir, "markers", "state.jsonl")
	filePath := filepath.Join(dir, "markers", "data.json")
	crashPath := filepath.Join(dir, "markers", "crash-once")

	if err := appendMarkerLine("", "ignored"); err != nil {
		t.Fatalf("appendMarkerLine(empty) error = %v", err)
	}
	if err := appendMarkerLine(linesPath, " first "); err != nil {
		t.Fatalf("appendMarkerLine(first) error = %v", err)
	}
	if err := appendMarkerLine(linesPath, "second"); err != nil {
		t.Fatalf("appendMarkerLine(second) error = %v", err)
	}
	data, err := os.ReadFile(linesPath)
	if err != nil {
		t.Fatalf("os.ReadFile(linesPath) error = %v", err)
	}
	if got, want := strings.TrimSpace(string(data)), "first\nsecond"; got != want {
		t.Fatalf("marker lines = %q, want %q", got, want)
	}

	if err := appendJSONLine(jsonPath, map[string]any{"ok": true}); err != nil {
		t.Fatalf("appendJSONLine() error = %v", err)
	}
	if err := writeJSONFile(filePath, map[string]any{"ok": true}); err != nil {
		t.Fatalf("writeJSONFile() error = %v", err)
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("os.Stat(filePath) error = %v", err)
	}

	var stderr bytes.Buffer
	reportSideEffectError(&stderr, "write marker", errors.New("boom"))
	if got := stderr.String(); !strings.Contains(got, "write marker") ||
		!strings.Contains(got, "boom") {
		t.Fatalf("reportSideEffectError() wrote %q, want action and error text", got)
	}
	reportSideEffectError(&stderr, "noop", nil)
	reportSideEffectError(nil, "noop", errors.New("ignored"))

	if shouldCrashOnce("") {
		t.Fatal("shouldCrashOnce(empty) = true, want false")
	}
	if !shouldCrashOnce(crashPath) {
		t.Fatal("shouldCrashOnce(missing) = false, want true")
	}
	if err := os.WriteFile(crashPath, []byte("done"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(crashPath) error = %v", err)
	}
	if shouldCrashOnce(crashPath) {
		t.Fatal("shouldCrashOnce(existing) = true, want false")
	}
}

func TestRunServeCoverage(t *testing.T) {
	t.Parallel()

	if err := runServe(strings.NewReader(""), io.Discard, io.Discard); err != nil {
		t.Fatalf("runServe(empty stdin) error = %v", err)
	}
	if err := run(nil, strings.NewReader(""), io.Discard, io.Discard); err != nil {
		t.Fatalf("run(default serve) error = %v", err)
	}
}

func TestHandleWebhookRequestCoverage(t *testing.T) {
	t.Parallel()

	runtime, err := newTeamsProvider(io.Discard)
	if err != nil {
		t.Fatalf("newTeamsProvider() error = %v", err)
	}

	cfg := resolvedInstanceConfig{
		instanceID: "brg-1",
		dedup:      bridgesdk.NewDedupCache(5*time.Minute, 16),
	}

	recorder := httptest.NewRecorder()
	err = runtime.handleWebhookRequest(recorder, cfg, bridgesdk.WebhookRequest{
		Body:       []byte("{"),
		ReceivedAt: time.Now().UTC(),
	})
	var httpErr *bridgesdk.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("handleWebhookRequest(invalid json) error = %v, want bad request http error", err)
	}

	recorder = httptest.NewRecorder()
	err = runtime.handleWebhookRequest(recorder, cfg, bridgesdk.WebhookRequest{
		Body: []byte(
			`{"type":"conversationUpdate","from":{"id":"29:user-1"},"serviceUrl":"https://service.test","conversation":{"tenantId":"tenant-1"}}`,
		),
	})
	if err != nil {
		t.Fatalf("handleWebhookRequest(ignored activity) error = %v", err)
	}
	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("handleWebhookRequest(ignored activity) status = %d, want %d", got, want)
	}
}

func TestRemoteMessageReferenceHelpers(t *testing.T) {
	t.Parallel()

	encoded := encodeRemoteMessageID(teamsRemoteMessageRef{
		ConversationID: "conversation-1",
		ServiceURL:     "https://service.test/",
		ActivityID:     "activity-1",
	})
	decoded, err := decodeRemoteMessageID(encoded)
	if err != nil {
		t.Fatalf("decodeRemoteMessageID(valid) error = %v", err)
	}
	if decoded.ServiceURL != "https://service.test" {
		t.Fatalf(
			"decodeRemoteMessageID().ServiceURL = %q, want %q",
			decoded.ServiceURL,
			"https://service.test",
		)
	}
	if _, err := decodeRemoteMessageID(
		base64.RawURLEncoding.EncodeToString(
			[]byte(`{"conversation_id":"","service_url":"https://service.test","activity_id":"activity-1"}`),
		),
	); err == nil {
		t.Fatal("decodeRemoteMessageID(incomplete) error = nil, want non-nil")
	}

	if got, want := referenceRemoteMessageID(
		&bridgepkg.DeliveryMessageReference{RemoteMessageID: " remote-id "},
	), "remote-id"; got != want {
		t.Fatalf("referenceRemoteMessageID() = %q, want %q", got, want)
	}
	if referenceRemoteMessageID(nil) != "" {
		t.Fatal("referenceRemoteMessageID(nil) != empty string")
	}
}

type fakeTeamsAPI struct {
	createConversationID string
	nextActivityID       int
	validateErr          error

	createCalls []teamsCreateConversationRequest
	sendCalls   []teamsSendCall
	updateCalls []teamsUpdateCall
	deleteCalls []teamsDeleteCall
}

type teamsSendCall struct {
	ServiceURL     string
	ConversationID string
	ReplyToID      string
	Activity       teamsOutboundActivity
}

type teamsUpdateCall struct {
	ServiceURL     string
	ConversationID string
	ActivityID     string
	Activity       teamsOutboundActivity
}

type teamsDeleteCall struct {
	ServiceURL     string
	ConversationID string
	ActivityID     string
}

func (f *fakeTeamsAPI) ValidateAuth(context.Context) error {
	return f.validateErr
}

func (f *fakeTeamsAPI) CreateConversation(
	_ context.Context,
	_ string,
	req teamsCreateConversationRequest,
) (*teamsConversationResourceResponse, error) {
	f.createCalls = append(f.createCalls, req)
	return &teamsConversationResourceResponse{ID: f.createConversationID}, nil
}

func (f *fakeTeamsAPI) SendActivity(
	_ context.Context,
	serviceURL string,
	conversationID string,
	replyToID string,
	activity teamsOutboundActivity,
) (*teamsResourceResponse, error) {
	f.sendCalls = append(f.sendCalls, teamsSendCall{
		ServiceURL:     serviceURL,
		ConversationID: conversationID,
		ReplyToID:      replyToID,
		Activity:       activity,
	})
	id := fmt.Sprintf("activity-%d", f.nextActivityID)
	f.nextActivityID++
	return &teamsResourceResponse{ID: id}, nil
}

func (f *fakeTeamsAPI) UpdateActivity(
	_ context.Context,
	serviceURL string,
	conversationID string,
	activityID string,
	activity teamsOutboundActivity,
) error {
	f.updateCalls = append(f.updateCalls, teamsUpdateCall{
		ServiceURL:     serviceURL,
		ConversationID: conversationID,
		ActivityID:     activityID,
		Activity:       activity,
	})
	return nil
}

func (f *fakeTeamsAPI) DeleteActivity(
	_ context.Context,
	serviceURL string,
	conversationID string,
	activityID string,
) error {
	f.deleteCalls = append(f.deleteCalls, teamsDeleteCall{
		ServiceURL:     serviceURL,
		ConversationID: conversationID,
		ActivityID:     activityID,
	})
	return nil
}

type teamsProviderServerConfig struct{}

type teamsProviderServer struct {
	server     *httptest.Server
	privateKey *rsa.PrivateKey
	keyID      string

	mu       sync.Mutex
	apiCalls []teamsProviderAPICall
	nextID   int
}

type teamsProviderAPICall struct {
	Method string
	Path   string
	Body   map[string]any
}

func newTeamsProviderServer(t *testing.T, _ teamsProviderServerConfig) *teamsProviderServer {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	srv := &teamsProviderServer{privateKey: privateKey, keyID: "teams-test-key", nextID: 700}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/openid/.well-known/openidconfiguration":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"issuer":   "https://api.botframework.com",
				"jwks_uri": srv.server.URL + "/openid/keys",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/openid/keys":
			pub := privateKey.PublicKey
			_ = json.NewEncoder(w).Encode(map[string]any{
				"keys": []map[string]any{{
					"kty":          "RSA",
					"kid":          srv.keyID,
					"x5t":          srv.keyID,
					"n":            base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
					"e":            base64.RawURLEncoding.EncodeToString(bigEndianExponent(pub.E)),
					"endorsements": []string{"msteams"},
				}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/oauth2/v2.0/token":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"token_type":   "Bearer",
				"expires_in":   3600,
				"access_token": "bot-access-token",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v3/conversations":
			body := decodeJSONBody(r.Body)
			srv.recordCall(r.Method, r.URL.Path, body)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "a:created-conversation"})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/activities"):
			body := decodeJSONBody(r.Body)
			srv.recordCall(r.Method, r.URL.Path, body)
			srv.mu.Lock()
			id := fmt.Sprintf("activity-%d", srv.nextID)
			srv.nextID++
			srv.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/activities/"):
			srv.recordCall(r.Method, r.URL.Path, decodeJSONBody(r.Body))
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "updated"})
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/activities/"):
			srv.recordCall(r.Method, r.URL.Path, map[string]any{})
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": "unknown path"})
		}
	}))
	srv.server = server
	t.Cleanup(server.Close)
	return srv
}

func (s *teamsProviderServer) recordCall(method string, path string, body map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.apiCalls = append(s.apiCalls, teamsProviderAPICall{Method: method, Path: path, Body: body})
}

func (s *teamsProviderServer) ServiceURL() string {
	return s.server.URL
}

func (s *teamsProviderServer) MetadataURL() string {
	return s.server.URL + "/openid/.well-known/openidconfiguration"
}

func (s *teamsProviderServer) TokenURL() string {
	return s.server.URL + "/oauth2/v2.0/token"
}

func (s *teamsProviderServer) APICalls() []teamsProviderAPICall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]teamsProviderAPICall, len(s.apiCalls))
	copy(out, s.apiCalls)
	return out
}

func (s *teamsProviderServer) SignedToken(t *testing.T, appID string, serviceURL string) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, teamsAuthClaims{
		ServiceURL: serviceURL,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://api.botframework.com",
			Audience:  []string{appID},
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().UTC().Add(-time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
	})
	token.Header["kid"] = s.keyID
	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		t.Fatalf("token.SignedString() error = %v", err)
	}
	return signed
}

func decodeJSONBody(body io.ReadCloser) map[string]any {
	defer func() { _ = body.Close() }()
	out := map[string]any{}
	_ = json.NewDecoder(body).Decode(&out)
	return out
}

func bigEndianExponent(e int) []byte {
	if e == 0 {
		return []byte{0}
	}
	buf := make([]byte, 0, 4)
	for e > 0 {
		buf = append([]byte{byte(e & 0xff)}, buf...)
		e >>= 8
	}
	return buf
}

func postTeamsWebhook(
	t *testing.T,
	server *teamsProviderServer,
	webhookURL string,
	appID string,
	payload string,
) {
	t.Helper()

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		webhookURL,
		strings.NewReader(payload),
	)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext() error = %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+server.SignedToken(t, appID, server.ServiceURL()))

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return
		}
		if resp.StatusCode == http.StatusNotFound ||
			resp.StatusCode == http.StatusServiceUnavailable {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		t.Fatalf("webhook status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	t.Fatalf("webhook %s did not become ready before timeout", webhookURL)
}

func teamsMessageWebhook(serviceURL string, text string) string {
	return fmt.Sprintf(
		`{"type":"message","id":"activity-1","channelId":"msteams","serviceUrl":%q,"timestamp":"2026-04-15T18:25:00Z","text":%q,"from":{"id":"29:user-1","name":"Alice Example"},"recipient":{"id":"28:bot","name":"Bridge Bot"},"conversation":{"id":"19:channel@thread.tacv2;messageid=activity-1","conversationType":"channel","tenantId":"11111111-2222-3333-4444-555555555555"}}`,
		serviceURL,
		text,
	)
}

func teamsInvokeWebhook(serviceURL string) string {
	return fmt.Sprintf(
		`{"type":"invoke","id":"activity-2","channelId":"msteams","serviceUrl":%q,"timestamp":"2026-04-15T18:25:01Z","from":{"id":"29:user-1","name":"Alice Example"},"recipient":{"id":"28:bot","name":"Bridge Bot"},"conversation":{"id":"a:direct-conversation","tenantId":"11111111-2222-3333-4444-555555555555"},"value":{"action":{"data":{"actionId":"approve","value":"yes"}}}}`,
		serviceURL,
	)
}

func teamsReactionWebhook(serviceURL string) string {
	return fmt.Sprintf(
		`{"type":"messageReaction","id":"activity-3","channelId":"msteams","serviceUrl":%q,"timestamp":"2026-04-15T18:25:02Z","from":{"id":"29:user-1","name":"Alice Example"},"conversation":{"id":"19:channel@thread.tacv2;messageid=activity-1","conversationType":"channel","tenantId":"11111111-2222-3333-4444-555555555555"},"reactionsAdded":[{"type":"like"}],"reactionsRemoved":[{"type":"sad"}]}`,
		serviceURL,
	)
}

func newRuntimePeerPair(t *testing.T) (*teamsProvider, *bridgesdk.Peer, func()) {
	t.Helper()

	hostConn, runtimeConn := net.Pipe()
	runtime, err := newTeamsProvider(io.Discard)
	if err != nil {
		t.Fatalf("newTeamsProvider() error = %v", err)
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
			runtime.mu.RUnlock()
			if server != nil {
				shutdownCtx, shutdownCancel := context.WithTimeout(
					context.Background(),
					2*time.Second,
				)
				_ = server.Shutdown(shutdownCtx)
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

func mustHandleLifecycle(
	t *testing.T,
	peer *bridgesdk.Peer,
	managed ...subprocess.InitializeBridgeManagedInstance,
) {
	t.Helper()

	mustHandle(
		t,
		peer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesList),
		func(context.Context, json.RawMessage) (any, error) {
			instances := make([]bridgepkg.BridgeInstance, 0, len(managed))
			for _, item := range managed {
				instances = append(instances, item.Instance)
			}
			return instances, nil
		},
	)
	mustHandle(
		t,
		peer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesGet),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgeInstanceTargetParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			for _, item := range managed {
				if item.Instance.ID == payload.BridgeInstanceID {
					return item.Instance, nil
				}
			}
			return nil, errors.New("unexpected instance")
		},
	)
	mustHandle(
		t,
		peer,
		string(extensionprotocol.HostAPIMethodBridgesInstancesReportState),
		func(_ context.Context, params json.RawMessage) (any, error) {
			var payload extensioncontract.BridgesInstancesReportStateParams
			if err := json.Unmarshal(params, &payload); err != nil {
				return nil, err
			}
			for _, item := range managed {
				if item.Instance.ID == payload.BridgeInstanceID {
					instance := item.Instance
					instance.Status = payload.Status
					instance.Degradation = payload.Degradation
					return instance, nil
				}
			}
			return nil, errors.New("unexpected state instance")
		},
	)
}

func testTeamsManagedInstance(
	t *testing.T,
	now time.Time,
	instanceID string,
	providerConfig map[string]any,
) subprocess.InitializeBridgeManagedInstance {
	encodedProviderConfig, err := json.Marshal(providerConfig)
	if err != nil {
		t.Fatalf("json.Marshal(providerConfig) error = %v", err)
	}
	return subprocess.InitializeBridgeManagedInstance{
		Instance: bridgepkg.BridgeInstance{
			ID:            instanceID,
			Scope:         bridgepkg.ScopeWorkspace,
			WorkspaceID:   "ws-teams",
			Platform:      "teams",
			ExtensionName: "teams",
			DisplayName:   "Teams",
			Enabled:       true,
			Status:        bridgepkg.BridgeStatusReady,
			RoutingPolicy: bridgepkg.RoutingPolicy{
				IncludePeer:   true,
				IncludeGroup:  true,
				IncludeThread: true,
			},
			ProviderConfig: encodedProviderConfig,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
		BoundSecrets: []subprocess.InitializeBridgeBoundSecret{
			{BindingName: "app_id", Kind: "token", Value: "app-id"},
			{BindingName: "app_password", Kind: "token", Value: "app-password"},
			{
				BindingName: "app_tenant_id",
				Kind:        "token",
				Value:       "11111111-2222-3333-4444-555555555555",
			},
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
			Name:       "teams",
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
				Provider:         "teams",
				Platform:         "teams",
				ManagedInstances: managed,
			},
		},
	}
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

	var listenConfig net.ListenConfig
	ln, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("ListenConfig.Listen() error = %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("ln.Close() error = %v", err)
	}
	return addr
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
