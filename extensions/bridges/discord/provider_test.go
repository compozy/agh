package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	"github.com/pedronauck/agh/internal/bridgesdk"
	extensioncontract "github.com/pedronauck/agh/internal/extension/contract"
	"github.com/pedronauck/agh/internal/subprocess"
)

func TestVerifyDiscordSignatureRejectsInvalidPublicKeySignatures(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	body := []byte(`{"hello":"discord"}`)
	timestamp := "1775866800"
	message := append([]byte(timestamp), body...)
	signature := ed25519.Sign(priv, message)

	req := httptest.NewRequest(http.MethodPost, "/discord/brg-1", nil)
	req.Header.Set("X-Signature-Timestamp", timestamp)
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(signature))

	now := time.Unix(1775866800, 0).UTC()
	if err := verifyDiscordSignature(context.Background(), req, body, hex.EncodeToString(pub), now); err != nil {
		t.Fatalf("verifyDiscordSignature(valid) error = %v", err)
	}

	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(ed25519.Sign(priv, []byte("wrong"+string(body)))))
	if err := verifyDiscordSignature(context.Background(), req, body, hex.EncodeToString(pub), now); err == nil {
		t.Fatal("verifyDiscordSignature(invalid) error = nil, want non-nil")
	}
}

func TestMapDiscordMessageEventRoutingAndAttachments(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	managed := testDiscordManagedInstance("brg-discord")

	direct, ignored, err := mapDiscordMessageEvent(discordMessageEvent{
		ID:          "msg-dm-1",
		ChannelID:   "dm-1",
		ChannelType: discordChannelTypeDM,
		Content:     " hello ",
		Timestamp:   now.Format(time.RFC3339Nano),
		Author: discordUser{
			ID:       "user-1",
			Username: "alice",
		},
		Attachments: []discordAttachment{{
			ID:          "att-1",
			Filename:    "report.txt",
			ContentType: "text/plain",
			URL:         "https://cdn.test/report.txt",
		}},
	}, managed, now, "evt-msg-1")
	if err != nil {
		t.Fatalf("mapDiscordMessageEvent(direct) error = %v", err)
	}
	if ignored {
		t.Fatal("mapDiscordMessageEvent(direct) ignored = true, want false")
	}
	if got, want := direct.Envelope.PeerID, "dm-1"; got != want {
		t.Fatalf("PeerID = %q, want %q", got, want)
	}
	if got, want := direct.Envelope.Attachments[0].ID, "att-1"; got != want {
		t.Fatalf("attachment id = %q, want %q", got, want)
	}

	threaded, ignored, err := mapDiscordMessageEvent(discordMessageEvent{
		ID:          "msg-thread-1",
		ChannelID:   "thread-1",
		GuildID:     "guild-1",
		ParentID:    "channel-1",
		ChannelType: discordChannelTypePublicThread,
		Content:     " need summary ",
		Timestamp:   now.Format(time.RFC3339Nano),
		Author: discordUser{
			ID:       "user-2",
			Username: "bob",
		},
	}, managed, now, "evt-msg-2")
	if err != nil {
		t.Fatalf("mapDiscordMessageEvent(threaded) error = %v", err)
	}
	if ignored {
		t.Fatal("mapDiscordMessageEvent(threaded) ignored = true, want false")
	}
	if got, want := threaded.Envelope.GroupID, "channel-1"; got != want {
		t.Fatalf("GroupID = %q, want %q", got, want)
	}
	if got, want := threaded.Envelope.ThreadID, "thread-1"; got != want {
		t.Fatalf("ThreadID = %q, want %q", got, want)
	}
	if got, want := threaded.Envelope.Content.Text, "need summary"; got != want {
		t.Fatalf("Content.Text = %q, want %q", got, want)
	}
}

func TestMapDiscordInteractionPayloadsStableTargetIdentity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 1, 0, 0, time.UTC)
	managed := testDiscordManagedInstance("brg-discord")

	command, err := mapDiscordInteractionCommand(discordInteraction{
		ID:        "ixn-cmd-1",
		Type:      discordInteractionTypeApplicationCommand,
		Token:     "token-cmd-1",
		GuildID:   "guild-1",
		ChannelID: "thread-1",
		Channel: &discordInteractionChannel{
			ID:       "thread-1",
			Type:     discordChannelTypePublicThread,
			ParentID: "channel-1",
		},
		Member: &discordInteractionMember{
			User: &discordUser{
				ID:         "user-1",
				Username:   "alice",
				GlobalName: "Alice",
			},
		},
		Data: &discordInteractionData{
			Name: "agh",
			Options: []discordInteractionOption{{
				Name: "summarize",
				Type: discordApplicationCommandOptionTypeSubcommand,
				Options: []discordInteractionOption{{
					Name:  "topic",
					Value: "release notes",
				}},
			}},
		},
	}, managed, now)
	if err != nil {
		t.Fatalf("mapDiscordInteractionCommand() error = %v", err)
	}
	if got, want := command.Envelope.EventFamily, bridgepkg.InboundEventFamilyCommand; got != want {
		t.Fatalf("EventFamily = %q, want %q", got, want)
	}
	if got, want := command.Envelope.GroupID, "channel-1"; got != want {
		t.Fatalf("GroupID = %q, want %q", got, want)
	}
	if got, want := command.Envelope.ThreadID, "thread-1"; got != want {
		t.Fatalf("ThreadID = %q, want %q", got, want)
	}
	if got, want := command.Envelope.Command.Command, "/agh summarize"; got != want {
		t.Fatalf("Command.Command = %q, want %q", got, want)
	}
	if got, want := command.Envelope.Command.Text, "release notes"; got != want {
		t.Fatalf("Command.Text = %q, want %q", got, want)
	}
	if got, want := command.Envelope.IdempotencyKey, "ixn-cmd-1"; got != want {
		t.Fatalf("IdempotencyKey = %q, want %q", got, want)
	}

	action, err := mapDiscordInteractionAction(discordInteraction{
		ID:        "ixn-action-1",
		Type:      discordInteractionTypeMessageComponent,
		Token:     "token-action-1",
		ChannelID: "dm-1",
		Channel: &discordInteractionChannel{
			ID:   "dm-1",
			Type: discordChannelTypeDM,
		},
		User: &discordUser{
			ID:       "user-2",
			Username: "bob",
		},
		Message: &discordInteractionMessage{ID: "msg-1"},
		Data: &discordInteractionData{
			CustomID:      "approve",
			ComponentType: 2,
			Values:        []string{"yes"},
		},
	}, managed, now)
	if err != nil {
		t.Fatalf("mapDiscordInteractionAction() error = %v", err)
	}
	if got, want := action.Envelope.EventFamily, bridgepkg.InboundEventFamilyAction; got != want {
		t.Fatalf("EventFamily = %q, want %q", got, want)
	}
	if got, want := action.Envelope.PeerID, "dm-1"; got != want {
		t.Fatalf("PeerID = %q, want %q", got, want)
	}
	if got, want := action.Envelope.Action.ActionID, "approve"; got != want {
		t.Fatalf("ActionID = %q, want %q", got, want)
	}
	if got, want := action.Envelope.Action.MessageID, "msg-1"; got != want {
		t.Fatalf("MessageID = %q, want %q", got, want)
	}
	if got, want := action.Envelope.Action.Value, "yes"; got != want {
		t.Fatalf("Value = %q, want %q", got, want)
	}
}

func TestMapDiscordReactionPayloads(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 3, 0, 0, time.UTC)
	managed := testDiscordManagedInstance("brg-discord")

	mapped, err := mapDiscordReactionEvent(discordReactionEvent{
		ChannelID:   "thread-1",
		GuildID:     "guild-1",
		ParentID:    "channel-1",
		ChannelType: discordChannelTypePublicThread,
		MessageID:   "msg-1",
		UserID:      "user-1",
		Emoji:       discordEmoji{Name: "thumbsup"},
		Timestamp:   now.Format(time.RFC3339Nano),
	}, managed, now, "evt-reaction-1", "MESSAGE_REACTION_ADD")
	if err != nil {
		t.Fatalf("mapDiscordReactionEvent(valid) error = %v", err)
	}
	if got, want := mapped.Envelope.EventFamily, bridgepkg.InboundEventFamilyReaction; got != want {
		t.Fatalf("EventFamily = %q, want %q", got, want)
	}
	if got, want := mapped.Envelope.GroupID, "channel-1"; got != want {
		t.Fatalf("GroupID = %q, want %q", got, want)
	}
	if got, want := mapped.Envelope.ThreadID, "thread-1"; got != want {
		t.Fatalf("ThreadID = %q, want %q", got, want)
	}
	if got, want := mapped.Envelope.Reaction.Emoji, ":thumbsup:"; got != want {
		t.Fatalf("Reaction.Emoji = %q, want %q", got, want)
	}
	if !mapped.Envelope.Reaction.Added {
		t.Fatal("Reaction.Added = false, want true")
	}

	if _, err := mapDiscordReactionEvent(discordReactionEvent{
		ChannelID: "channel-1",
		UserID:    "user-1",
	}, managed, now, "evt-bad", "MESSAGE_REACTION_ADD"); err == nil {
		t.Fatal("mapDiscordReactionEvent(malformed) error = nil, want non-nil")
	}
}

func TestExecuteDiscordDeliveryValidatesEditAndDeleteOperations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	target := bridgepkg.DeliveryTarget{
		BridgeInstanceID: "brg-discord",
		GroupID:          "channel-1",
		ThreadID:         "thread-1",
		Mode:             bridgepkg.DeliveryModeReply,
	}
	routing := bridgepkg.RoutingKey{
		Scope:            bridgepkg.ScopeWorkspace,
		WorkspaceID:      "ws-1",
		BridgeInstanceID: "brg-discord",
		GroupID:          "channel-1",
		ThreadID:         "thread-1",
	}

	postReq := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "del-1",
			BridgeInstanceID: "brg-discord",
			RoutingKey:       routing,
			DeliveryTarget:   target,
			Seq:              1,
			EventType:        bridgepkg.DeliveryEventTypeStart,
			Final:            false,
			Content:          bridgepkg.MessageContent{Text: "hello"},
		},
	}
	editReq := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "del-1",
			BridgeInstanceID: "brg-discord",
			RoutingKey:       routing,
			DeliveryTarget:   target,
			Seq:              2,
			EventType:        bridgepkg.DeliveryEventTypeDelta,
			Content:          bridgepkg.MessageContent{Text: "hello world"},
			Operation:        bridgepkg.DeliveryOperationEdit,
			Reference:        &bridgepkg.DeliveryMessageReference{RemoteMessageID: "thread-1:msg-1"},
		},
	}
	invalidEditReq := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "del-1",
			BridgeInstanceID: "brg-discord",
			RoutingKey:       routing,
			DeliveryTarget:   target,
			Seq:              2,
			EventType:        bridgepkg.DeliveryEventTypeDelta,
			Content:          bridgepkg.MessageContent{Text: "hello world"},
			Operation:        bridgepkg.DeliveryOperationEdit,
		},
	}
	deleteReq := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "del-1",
			BridgeInstanceID: "brg-discord",
			RoutingKey:       routing,
			DeliveryTarget:   target,
			Seq:              3,
			EventType:        bridgepkg.DeliveryEventTypeDelete,
			Final:            true,
			Operation:        bridgepkg.DeliveryOperationDelete,
			Reference:        &bridgepkg.DeliveryMessageReference{RemoteMessageID: "thread-1:msg-1"},
		},
	}

	api := &discordAPIFake{postedMessageID: "msg-1"}
	ack, state, err := executeDiscordDelivery(ctx, api, postReq, deliveryState{})
	if err != nil {
		t.Fatalf("executeDiscordDelivery(post) error = %v", err)
	}
	if got, want := ack.RemoteMessageID, "thread-1:msg-1"; got != want {
		t.Fatalf("RemoteMessageID = %q, want %q", got, want)
	}
	if got, want := api.postRequests[0].ChannelID, "thread-1"; got != want {
		t.Fatalf("post channel = %q, want %q", got, want)
	}

	if _, _, err := executeDiscordDelivery(ctx, api, invalidEditReq, deliveryState{}); err == nil {
		t.Fatal("executeDiscordDelivery(edit without state) error = nil, want non-nil")
	}

	if _, state, err = executeDiscordDelivery(ctx, api, editReq, state); err != nil {
		t.Fatalf("executeDiscordDelivery(edit with state) error = %v", err)
	}
	if got, want := api.updateRequests[0].MessageID, "msg-1"; got != want {
		t.Fatalf("update message id = %q, want %q", got, want)
	}

	if _, _, err := executeDiscordDelivery(ctx, api, deleteReq, state); err != nil {
		t.Fatalf("executeDiscordDelivery(delete) error = %v", err)
	}
	if got, want := api.deleteRequests[0].MessageID, "msg-1"; got != want {
		t.Fatalf("delete message id = %q, want %q", got, want)
	}
}

func TestHandleInteractionWebhookAcknowledgesImmediately(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 15, 12, 4, 0, 0, time.UTC)
	provider, err := newDiscordProvider(nil)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	provider.now = func() time.Time { return now }
	provider.mu.Lock()
	provider.session = nil
	provider.routes["brg-discord"] = resolvedInstanceConfig{
		instanceID: "brg-discord",
		managed:    testDiscordManagedInstance("brg-discord"),
		dedup:      bridgesdk.NewDedupCache(time.Minute, 16),
		dmPolicy:   bridgepkg.BridgeDMPolicyOpen,
	}
	provider.mu.Unlock()

	recorder := httptest.NewRecorder()
	err = provider.handleInteractionWebhook(recorder, resolvedInstanceConfig{
		instanceID: "brg-discord",
		managed:    testDiscordManagedInstance("brg-discord"),
		dedup:      bridgesdk.NewDedupCache(time.Minute, 16),
		dmPolicy:   bridgepkg.BridgeDMPolicyOpen,
	}, bridgepkgToWebhookRequest(t, discordInteraction{
		ID:        "ixn-cmd-1",
		Type:      discordInteractionTypeApplicationCommand,
		Token:     "token-cmd-1",
		ChannelID: "dm-1",
		Channel:   &discordInteractionChannel{ID: "dm-1", Type: discordChannelTypeDM},
		User:      &discordUser{ID: "user-1", Username: "alice"},
		Data:      &discordInteractionData{Name: "agh"},
	}, now))
	if err != nil {
		t.Fatalf("handleInteractionWebhook() error = %v", err)
	}
	if got, want := recorder.Code, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}
	if body := strings.TrimSpace(recorder.Body.String()); body != `{"type":5}` {
		t.Fatalf("body = %s, want {\"type\":5}", body)
	}
}

func TestDiscordBotClientRoutesRequestsAndClassifiesFailures(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.Method+" "+r.URL.Path)
		mu.Unlock()

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/users/@me":
			_ = json.NewEncoder(w).Encode(discordBotIdentity{ID: "bot-1", Username: "agh"})
		case r.Method == http.MethodPost && r.URL.Path == "/channels/thread-1/messages":
			_ = json.NewEncoder(w).Encode(discordPostedMessage{ID: "msg-1"})
		case r.Method == http.MethodPatch && r.URL.Path == "/channels/thread-1/messages/msg-1":
			_ = json.NewEncoder(w).Encode(discordPostedMessage{ID: "msg-1"})
		case r.Method == http.MethodDelete && r.URL.Path == "/channels/thread-1/messages/msg-1":
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/channels/thread-2/messages":
			w.Header().Set("Retry-After", "2")
			http.Error(w, "too many requests", http.StatusTooManyRequests)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &discordBotClient{
		baseURL:    server.URL,
		botToken:   "discord-bot-token",
		httpClient: server.Client(),
	}

	if _, err := client.GetBotUser(context.Background()); err != nil {
		t.Fatalf("GetBotUser() error = %v", err)
	}
	if _, err := client.PostMessage(context.Background(), discordPostMessageRequest{
		ChannelID: "thread-1",
		Content:   "hello",
	}); err != nil {
		t.Fatalf("PostMessage() error = %v", err)
	}
	if err := client.UpdateMessage(context.Background(), discordUpdateMessageRequest{
		ChannelID: "thread-1",
		MessageID: "msg-1",
		Content:   "hello world",
	}); err != nil {
		t.Fatalf("UpdateMessage() error = %v", err)
	}
	if err := client.DeleteMessage(context.Background(), discordDeleteMessageRequest{
		ChannelID: "thread-1",
		MessageID: "msg-1",
	}); err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	if _, err := client.PostMessage(context.Background(), discordPostMessageRequest{
		ChannelID: "thread-2",
		Content:   "slow",
	}); err == nil {
		t.Fatal("PostMessage(rate limited) error = nil, want non-nil")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(paths) < 5 {
		t.Fatalf("len(paths) = %d, want at least 5", len(paths))
	}
}

func TestServeWebhookHTTPHandlesSignedMessageWebhookWithBatching(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	now := time.Now().UTC()
	deliveries := make(chan bridgepkg.InboundMessageEnvelope, 1)
	batcher, err := bridgesdk.NewInboundBatcher(bridgesdk.InboundBatcherConfig{
		Context: context.Background(),
		Delay:   time.Millisecond,
		Dispatch: func(_ context.Context, batch bridgesdk.InboundBatch) error {
			deliveries <- batch.Items[0]
			return nil
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewInboundBatcher() error = %v", err)
	}
	defer batcher.Close()

	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	provider.now = func() time.Time { return now }
	provider.mu.Lock()
	provider.routes["brg-discord"] = resolvedInstanceConfig{
		instanceID:      "brg-discord",
		managed:         testDiscordManagedInstance("brg-discord"),
		webhookPath:     "/discord/brg-discord",
		publicKey:       hex.EncodeToString(pub),
		dedup:           bridgesdk.NewDedupCache(time.Minute, 16),
		rateLimiter:     bridgesdk.NewFixedWindowRateLimiter(10, time.Minute),
		inFlightLimiter: bridgesdk.NewInFlightLimiter(4),
		batcher:         batcher,
		dmPolicy:        bridgepkg.BridgeDMPolicyOpen,
	}
	provider.mu.Unlock()

	payload := map[string]any{
		"type": 1,
		"event": map[string]any{
			"id":        "evt-msg-1",
			"type":      "MESSAGE_CREATE",
			"timestamp": now.Format(time.RFC3339Nano),
			"data": map[string]any{
				"id":           "msg-1",
				"channel_id":   "thread-1",
				"guild_id":     "guild-1",
				"parent_id":    "channel-1",
				"channel_type": 11,
				"content":      "Need a summary",
				"timestamp":    now.Format(time.RFC3339Nano),
				"author": map[string]any{
					"id":       "user-1",
					"username": "alice",
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	timestamp := strconv.FormatInt(now.Unix(), 10)
	signature := ed25519.Sign(priv, append([]byte(timestamp), body...))

	req := httptest.NewRequest(http.MethodPost, "http://discord.test/discord/brg-discord", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature-Timestamp", timestamp)
	req.Header.Set("X-Signature-Ed25519", hex.EncodeToString(signature))
	recorder := httptest.NewRecorder()

	provider.serveWebhookHTTP(recorder, req)

	if got, want := recorder.Code, http.StatusNoContent; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	select {
	case envelope := <-deliveries:
		if got, want := envelope.GroupID, "channel-1"; got != want {
			t.Fatalf("GroupID = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for batched envelope")
	}
}

func TestDetermineInitialStateAndLifecycleHelpers(t *testing.T) {
	t.Parallel()

	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	provider.apiFactory = func(resolvedInstanceConfig) discordAPI {
		return &discordAPIFake{postedMessageID: "msg-1"}
	}

	status, degradation, err := provider.determineInitialState(context.Background(), resolvedInstanceConfig{})
	if err == nil || status != bridgepkg.BridgeStatusAuthRequired || degradation == nil {
		t.Fatalf("determineInitialState(missing token) = (%q, %#v, %v), want auth_required with error", status, degradation, err)
	}

	status, degradation, err = provider.determineInitialState(context.Background(), resolvedInstanceConfig{
		botToken:  "discord-bot-token",
		publicKey: "bad",
	})
	if err == nil || status != bridgepkg.BridgeStatusAuthRequired || degradation == nil {
		t.Fatalf("determineInitialState(invalid key) = (%q, %#v, %v), want auth_required with error", status, degradation, err)
	}

	status, degradation, err = provider.determineInitialState(context.Background(), resolvedInstanceConfig{
		botToken:      "discord-bot-token",
		publicKey:     hex.EncodeToString(pub),
		applicationID: "bot-1",
	})
	if err != nil {
		t.Fatalf("determineInitialState(valid) error = %v", err)
	}
	if got, want := status, bridgepkg.BridgeStatusReady; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if degradation != nil {
		t.Fatalf("degradation = %#v, want nil", degradation)
	}

	status, degradation, err = provider.determineInitialState(context.Background(), resolvedInstanceConfig{
		botToken:      "discord-bot-token",
		publicKey:     hex.EncodeToString(pub),
		applicationID: "other-bot",
	})
	if err == nil || status != bridgepkg.BridgeStatusDegraded || degradation == nil {
		t.Fatalf("determineInitialState(mismatch) = (%q, %#v, %v), want degraded with error", status, degradation, err)
	}

	provider.setLastError(errors.New("boom"))
	if err := provider.healthCheck(); err == nil {
		t.Fatal("healthCheck() error = nil, want non-nil")
	}
	provider.clearLastError()
	if err := provider.healthCheck(); err != nil {
		t.Fatalf("healthCheck() error = %v, want nil", err)
	}
}

func TestHandleInitializeAfterInitializeRetryAndShutdownHelpers(t *testing.T) {
	t.Parallel()

	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}

	if err := provider.handleInitialize(context.Background(), &bridgesdk.Session{}); err != nil {
		t.Fatalf("handleInitialize() error = %v", err)
	}

	attempts := 0
	err = provider.retryHostCall(context.Background(), func(context.Context) error {
		attempts++
		if attempts == 1 {
			return rpcCodeErr{}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("retryHostCall() error = %v", err)
	}

	if err := provider.handleShutdown(context.Background(), nil, subprocess.ShutdownRequest{DeadlineMS: 10}); err != nil {
		t.Fatalf("handleShutdown() error = %v", err)
	}
	provider.stop()
}

func TestHandleEventAndInteractionWebhookBranches(t *testing.T) {
	t.Parallel()

	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	defer func() {
		provider.stop()
		provider.wg.Wait()
	}()
	provider.apiFactory = func(resolvedInstanceConfig) discordAPI {
		return &discordAPIFake{postedMessageID: "msg-1"}
	}
	var mu sync.Mutex
	ingests := make([]bridgepkg.InboundMessageEnvelope, 0)
	cfg := resolvedInstanceConfig{
		instanceID: "brg-discord",
		managed:    testDiscordManagedInstance("brg-discord"),
		publicKey:  hex.EncodeToString(pub),
		dedup:      bridgesdk.NewDedupCache(time.Minute, 16),
		dmPolicy:   bridgepkg.BridgeDMPolicyOpen,
	}
	provider.mu.Lock()
	provider.session = injectedDiscordSession(t,
		bridgesdk.NewHostAPIClientFromCall(func(_ context.Context, method string, params any, result any) error {
			if method == "bridges/messages/ingest" {
				mu.Lock()
				ingests = append(ingests, params.(bridgepkg.InboundMessageEnvelope))
				mu.Unlock()
				target := result.(*extensioncontract.BridgesMessagesIngestResult)
				*target = extensioncontract.BridgesMessagesIngestResult{SessionID: "sess-1"}
				return nil
			}
			if method == "bridges/instances/report_state" {
				target := result.(*bridgepkg.BridgeInstance)
				*target = testDiscordManagedInstance("brg-discord").Instance
				return nil
			}
			return nil
		}),
		bridgesdk.NewInstanceCache(&subprocess.InitializeBridgeRuntime{}),
	)
	provider.routes["brg-discord"] = cfg
	provider.mu.Unlock()

	recorder := httptest.NewRecorder()
	if err := provider.handleEventWebhook(recorder, nil, cfg, bridgepkgToWebhookRequest(t, map[string]any{"type": 0}, time.Now().UTC())); err != nil {
		t.Fatalf("handleEventWebhook(ping) error = %v", err)
	}
	if got, want := recorder.Code, http.StatusNoContent; got != want {
		t.Fatalf("ping status = %d, want %d", got, want)
	}

	recorder = httptest.NewRecorder()
	if err := provider.handleEventWebhook(recorder, nil, cfg, bridgepkgToWebhookRequest(t, map[string]any{
		"type": 1,
		"event": map[string]any{
			"id":        "evt-reaction-1",
			"type":      "MESSAGE_REACTION_ADD",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"data": map[string]any{
				"channel_id": "dm-1",
				"message_id": "msg-1",
				"user_id":    "user-1",
				"emoji": map[string]any{
					"name": "thumbsup",
				},
			},
		},
	}, time.Now().UTC())); err != nil {
		t.Fatalf("handleEventWebhook(reaction) error = %v", err)
	}
	if got, want := recorder.Code, http.StatusNoContent; got != want {
		t.Fatalf("reaction status = %d, want %d", got, want)
	}
	mu.Lock()
	reactionIngests := len(ingests)
	mu.Unlock()
	if got, want := reactionIngests, 1; got != want {
		t.Fatalf("reaction ingests = %d, want %d", got, want)
	}

	recorder = httptest.NewRecorder()
	blockedCfg := cfg
	blockedCfg.dedup = bridgesdk.NewDedupCache(time.Minute, 16)
	blockedCfg.dmPolicy = bridgepkg.BridgeDMPolicyAllowlist
	if err := provider.handleEventWebhook(recorder, nil, blockedCfg, bridgepkgToWebhookRequest(t, map[string]any{
		"type": 1,
		"event": map[string]any{
			"id":        "evt-reaction-blocked",
			"type":      "MESSAGE_REACTION_ADD",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"data": map[string]any{
				"channel_id": "dm-2",
				"message_id": "msg-2",
				"user_id":    "user-blocked",
				"emoji": map[string]any{
					"name": "thumbsup",
				},
			},
		},
	}, time.Now().UTC())); err != nil {
		t.Fatalf("handleEventWebhook(blocked reaction) error = %v", err)
	}
	if got, want := recorder.Code, http.StatusNoContent; got != want {
		t.Fatalf("blocked reaction status = %d, want %d", got, want)
	}
	mu.Lock()
	blockedReactionIngests := len(ingests)
	mu.Unlock()
	if got, want := blockedReactionIngests, 1; got != want {
		t.Fatalf("blocked reaction ingests = %d, want %d", got, want)
	}

	recorder = httptest.NewRecorder()
	if err := provider.handleInteractionWebhook(recorder, cfg, bridgepkgToWebhookRequest(t, discordInteraction{ID: "ixn-ping", Type: discordInteractionTypePing}, time.Now().UTC())); err != nil {
		t.Fatalf("handleInteractionWebhook(ping) error = %v", err)
	}
	if got, want := strings.TrimSpace(recorder.Body.String()), `{"type":1}`; got != want {
		t.Fatalf("ping body = %s, want %s", got, want)
	}

	recorder = httptest.NewRecorder()
	if err := provider.handleInteractionWebhook(recorder, cfg, bridgepkgToWebhookRequest(t, discordInteraction{
		ID:        "ixn-action-1",
		Type:      discordInteractionTypeMessageComponent,
		Token:     "ixn-token-1",
		ChannelID: "dm-1",
		Channel:   &discordInteractionChannel{ID: "dm-1", Type: discordChannelTypeDM},
		User:      &discordUser{ID: "user-1", Username: "alice"},
		Message:   &discordInteractionMessage{ID: "msg-1"},
		Data:      &discordInteractionData{CustomID: "approve"},
	}, time.Now().UTC())); err != nil {
		t.Fatalf("handleInteractionWebhook(action) error = %v", err)
	}
	if got, want := strings.TrimSpace(recorder.Body.String()), `{"type":6}`; got != want {
		t.Fatalf("action body = %s, want %s", got, want)
	}
}

func TestAllowDiscordDirectMessagePoliciesAndUtilityHelpers(t *testing.T) {
	t.Parallel()

	user := discordUserIdentity{ID: "user-1", Username: "alice"}
	if !allowDiscordDirectMessage(resolvedInstanceConfig{dmPolicy: bridgepkg.BridgeDMPolicyOpen}, user, true) {
		t.Fatal("allowDiscordDirectMessage(open) = false, want true")
	}
	if allowDiscordDirectMessage(resolvedInstanceConfig{
		dmPolicy:     bridgepkg.BridgeDMPolicyAllowlist,
		allowUserIDs: buildDiscordIDSet([]string{"other"}),
	}, user, true) {
		t.Fatal("allowDiscordDirectMessage(allowlist mismatch) = true, want false")
	}
	if !allowDiscordDirectMessage(resolvedInstanceConfig{
		dmPolicy:       bridgepkg.BridgeDMPolicyPairing,
		pairedUserIDs:  buildDiscordIDSet([]string{"user-1"}),
		allowUsernames: buildDiscordUsernameSet([]string{"alice"}),
	}, user, true) {
		t.Fatal("allowDiscordDirectMessage(pairing) = false, want true")
	}
	if got := rawDiscordEmoji(discordEmoji{Name: "thumbsup", ID: "123"}); got != "thumbsup:123" {
		t.Fatalf("rawDiscordEmoji() = %q, want thumbsup:123", got)
	}
	if got := normalizeDiscordEmoji(discordEmoji{Name: "wave", ID: "123"}); got != "<:wave:123>" {
		t.Fatalf("normalizeDiscordEmoji(custom) = %q, want <:wave:123>", got)
	}
	if got := parseRetryAfter("2"); got != 2*time.Second {
		t.Fatalf("parseRetryAfter() = %s, want 2s", got)
	}
	if got := cloneDegradation(&bridgepkg.BridgeDegradation{Reason: bridgepkg.BridgeDegradationReasonAuthFailed}); got == nil {
		t.Fatal("cloneDegradation() = nil, want non-nil")
	}
}

func TestHandleBridgesDeliverFailureAndMainHelpers(t *testing.T) {
	t.Parallel()

	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	provider.apiFactory = func(resolvedInstanceConfig) discordAPI {
		return &discordAPIFake{postErr: &bridgesdk.HTTPError{StatusCode: http.StatusTooManyRequests, Message: "slow down"}}
	}
	provider.mu.Lock()
	provider.routes["brg-discord"] = resolvedInstanceConfig{
		instanceID: "brg-discord",
		managed:    testDiscordManagedInstance("brg-discord"),
		dedup:      bridgesdk.NewDedupCache(time.Minute, 16),
	}
	provider.mu.Unlock()

	session := injectedDiscordSession(t,
		bridgesdk.NewHostAPIClientFromCall(func(_ context.Context, method string, params any, result any) error {
			if method == "bridges/instances/report_state" {
				target := result.(*bridgepkg.BridgeInstance)
				updated := testDiscordManagedInstance("brg-discord").Instance
				updated.Status = params.(extensioncontract.BridgesInstancesReportStateParams).Status
				*target = updated
			}
			return nil
		}),
		bridgesdk.NewInstanceCache(&subprocess.InitializeBridgeRuntime{}),
	)

	req := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "del-err-1",
			BridgeInstanceID: "brg-discord",
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-1",
				BridgeInstanceID: "brg-discord",
				GroupID:          "channel-1",
				ThreadID:         "thread-1",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: "brg-discord",
				GroupID:          "channel-1",
				ThreadID:         "thread-1",
				Mode:             bridgepkg.DeliveryModeReply,
			},
			Seq:       1,
			EventType: bridgepkg.DeliveryEventTypeStart,
			Content:   bridgepkg.MessageContent{Text: "hello"},
		},
	}
	if _, err := provider.handleBridgesDeliver(context.Background(), session, req); err == nil {
		t.Fatal("handleBridgesDeliver(rate limited) error = nil, want non-nil")
	}

	if err := run([]string{"bad"}, bytes.NewBuffer(nil), &bytes.Buffer{}, io.Discard); err == nil {
		t.Fatal("run(bad) error = nil, want non-nil")
	}

	done := make(chan error, 1)
	go func() {
		done <- run(nil, bytes.NewBuffer(nil), &bytes.Buffer{}, io.Discard)
	}()
	select {
	case <-time.After(time.Second):
		t.Fatal("run(nil) timed out, want serve path to exit")
	case <-done:
	}
}

func TestAfterInitializeSuccessAndParsingBranches(t *testing.T) {
	t.Parallel()

	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	cfg := discordProviderConfig{
		APIBaseURL: "https://discord.test/api/",
	}
	cfg.Webhook.ListenAddr = "127.0.0.1:0"
	cfg.Webhook.Path = "/discord/brg-discord"
	rawConfig, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	instance := bridgepkg.BridgeInstance{
		ID:             "brg-discord",
		Scope:          bridgepkg.ScopeWorkspace,
		WorkspaceID:    "ws-1",
		ProviderConfig: rawConfig,
	}
	managed := subprocess.InitializeBridgeManagedInstance{
		Instance: instance,
		BoundSecrets: []subprocess.InitializeBridgeBoundSecret{
			{BindingName: "bot_token", Value: "discord-bot-token"},
			{BindingName: "public_key", Value: hex.EncodeToString(pub)},
		},
	}
	session := injectedDiscordSession(t,
		bridgesdk.NewHostAPIClientFromCall(func(_ context.Context, method string, params any, result any) error {
			switch method {
			case "bridges/instances/list":
				target := result.(*[]bridgepkg.BridgeInstance)
				*target = []bridgepkg.BridgeInstance{instance}
			case "bridges/instances/get":
				target := result.(*bridgepkg.BridgeInstance)
				*target = instance
			case "bridges/instances/report_state":
				target := result.(*bridgepkg.BridgeInstance)
				updated := instance
				updated.Status = params.(extensioncontract.BridgesInstancesReportStateParams).Status
				*target = updated
			default:
				return nil
			}
			return nil
		}),
		bridgesdk.NewInstanceCache(&subprocess.InitializeBridgeRuntime{
			RuntimeVersion: subprocess.InitializeBridgeRuntimeVersion1,
			Provider:       "discord",
			Platform:       "discord",
			ManagedInstances: []subprocess.InitializeBridgeManagedInstance{
				managed,
			},
		}),
	)
	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	tmpDir := t.TempDir()
	provider.env = markerEnv{
		handshakePath: filepath.Join(tmpDir, "handshake.json"),
		ownershipPath: filepath.Join(tmpDir, "ownership.json"),
		statePath:     filepath.Join(tmpDir, "state.jsonl"),
		startsPath:    filepath.Join(tmpDir, "starts.log"),
	}
	provider.apiFactory = func(resolvedInstanceConfig) discordAPI {
		return &discordAPIFake{postedMessageID: "msg-1"}
	}
	provider.afterInitialize(session)
	defer func() {
		if provider.server != nil {
			_ = provider.server.Close()
		}
	}()

	if _, err := os.Stat(provider.env.ownershipPath); err != nil {
		t.Fatalf("ownership marker missing: %v", err)
	}
	if _, err := os.Stat(provider.env.statePath); err != nil {
		t.Fatalf("state marker missing: %v", err)
	}

	if got := parseDiscordReceivedAt("", time.Unix(1, 0).UTC()); !got.Equal(time.Unix(1, 0).UTC()) {
		t.Fatalf("parseDiscordReceivedAt(empty) = %s, want fallback", got)
	}
	if got := parseDiscordReceivedAt("1775866800", time.Time{}); got.Unix() != 1775866800 {
		t.Fatalf("parseDiscordReceivedAt(unix) = %d, want 1775866800", got.Unix())
	}
}

func TestResolveInstanceConfigErrorBranchesAndServerGuards(t *testing.T) {
	t.Parallel()

	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}

	badManaged := subprocess.InitializeBridgeManagedInstance{
		Instance: bridgepkg.BridgeInstance{
			ID:             "brg-bad",
			Scope:          bridgepkg.ScopeWorkspace,
			WorkspaceID:    "ws-1",
			ProviderConfig: []byte(`{`),
		},
	}
	if cfg := provider.resolveInstanceConfig(&bridgesdk.Session{}, badManaged); cfg.configError == nil {
		t.Fatal("resolveInstanceConfig(invalid json) configError = nil, want non-nil")
	}

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/discord/missing", nil)
	provider.serveWebhookHTTP(recorder, req)
	if got, want := recorder.Code, http.StatusNotFound; got != want {
		t.Fatalf("serveWebhookHTTP(not found) status = %d, want %d", got, want)
	}
}

func TestAdditionalDiscordProviderBranches(t *testing.T) {
	t.Parallel()

	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	provider.apiFactory = func(resolvedInstanceConfig) discordAPI {
		return &discordAPIGetBotUserErrorFake{err: context.DeadlineExceeded}
	}

	status, degradation, err := provider.determineInitialState(context.Background(), resolvedInstanceConfig{
		configError: errors.New("bad config"),
	})
	if err == nil || status != bridgepkg.BridgeStatusDegraded || degradation == nil {
		t.Fatalf("determineInitialState(configError) = (%q, %#v, %v), want degraded with error", status, degradation, err)
	}

	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	status, degradation, err = provider.determineInitialState(context.Background(), resolvedInstanceConfig{
		botToken:  "discord-bot-token",
		publicKey: hex.EncodeToString(pub),
	})
	if err == nil || status != bridgepkg.BridgeStatusDegraded || degradation == nil {
		t.Fatalf("determineInitialState(timeout) = (%q, %#v, %v), want degraded with error", status, degradation, err)
	}

	if _, err := provider.waitForInstanceConfig("missing", 20*time.Millisecond); err == nil {
		t.Fatal("waitForInstanceConfig(missing) error = nil, want non-nil")
	}
	if isNotInitializedRPCError(errors.New("not initialized")) {
		t.Fatal("isNotInitializedRPCError(string only) = true, want false")
	}
	if !isNotInitializedRPCError(rpcCodeErr{}) {
		t.Fatal("isNotInitializedRPCError(rpc code) = false, want true")
	}

	if got := parseDiscordReceivedAt(time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC).Format(time.RFC3339), time.Time{}); got.IsZero() {
		t.Fatal("parseDiscordReceivedAt(rfc3339) = zero, want parsed time")
	}

	if !shouldPostDiscordMessage(bridgepkg.DeliveryEvent{EventType: bridgepkg.DeliveryEventTypeResume}, deliveryState{}, bridgepkg.DeliveryRequest{}) {
		t.Fatal("shouldPostDiscordMessage(resume without snapshot) = false, want true")
	}

	recorder := httptest.NewRecorder()
	if err := provider.handleWebhookRequest(recorder, nil, resolvedInstanceConfig{
		instanceID: "brg-discord",
		managed:    testDiscordManagedInstance("brg-discord"),
		dedup:      bridgesdk.NewDedupCache(time.Minute, 16),
		dmPolicy:   bridgepkg.BridgeDMPolicyOpen,
	}, bridgepkgToWebhookRequest(t, map[string]any{
		"type":  2,
		"token": "ixn-token-1",
	}, time.Now().UTC())); err == nil {
		t.Fatal("handleWebhookRequest(invalid interaction) error = nil, want non-nil")
	}
}

func TestDiscordWebhookAndHelperErrorBranches(t *testing.T) {
	t.Parallel()

	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	cfg := resolvedInstanceConfig{
		instanceID: "brg-discord",
		managed:    testDiscordManagedInstance("brg-discord"),
		dedup:      bridgesdk.NewDedupCache(time.Minute, 16),
		dmPolicy:   bridgepkg.BridgeDMPolicyOpen,
	}

	recorder := httptest.NewRecorder()
	err = provider.handleWebhookRequest(recorder, nil, cfg, bridgesdk.WebhookRequest{
		Body:       []byte("{"),
		ReceivedAt: time.Now().UTC(),
	})
	var httpErr *bridgesdk.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("handleWebhookRequest(invalid json) error = %v, want bad request http error", err)
	}

	recorder = httptest.NewRecorder()
	err = provider.handleInteractionWebhook(recorder, cfg, bridgepkgToWebhookRequest(t, discordInteraction{
		ID:    "ixn-unsupported-1",
		Type:  999,
		Token: "ixn-token-1",
	}, time.Now().UTC()))
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("handleInteractionWebhook(unsupported) error = %v, want bad request http error", err)
	}

	recorder = httptest.NewRecorder()
	if err := provider.handleEventWebhook(recorder, nil, cfg, bridgepkgToWebhookRequest(t, map[string]any{"type": 1}, time.Now().UTC())); err != nil {
		t.Fatalf("handleEventWebhook(missing event) error = %v", err)
	}
	if got, want := recorder.Code, http.StatusNoContent; got != want {
		t.Fatalf("handleEventWebhook(missing event) status = %d, want %d", got, want)
	}

	if got := normalizeWebhookPath("discord/brg-discord"); got != "/discord/brg-discord" {
		t.Fatalf("normalizeWebhookPath() = %q, want /discord/brg-discord", got)
	}
	if got := normalizeURL(" https://discord.test/api/ "); got != "https://discord.test/api" {
		t.Fatalf("normalizeURL() = %q, want https://discord.test/api", got)
	}
	if got := referenceRemoteMessageID(nil); got != "" {
		t.Fatalf("referenceRemoteMessageID(nil) = %q, want empty", got)
	}
	if got := referenceRemoteMessageID(&bridgepkg.DeliveryMessageReference{RemoteMessageID: " channel-1:msg-1 "}); got != "channel-1:msg-1" {
		t.Fatalf("referenceRemoteMessageID(value) = %q, want channel-1:msg-1", got)
	}
	if _, _, err := decodeRemoteMessageID("broken"); err == nil {
		t.Fatal("decodeRemoteMessageID(invalid) error = nil, want non-nil")
	}
	channelID, messageID, err := decodeRemoteMessageID(" channel-1 : msg-1 ")
	if err != nil {
		t.Fatalf("decodeRemoteMessageID(valid) error = %v", err)
	}
	if channelID != "channel-1" || messageID != "msg-1" {
		t.Fatalf("decodeRemoteMessageID(valid) = (%q, %q), want (channel-1, msg-1)", channelID, messageID)
	}
	if got := readResponseBody(nil); got != "" {
		t.Fatalf("readResponseBody(nil) = %q, want empty", got)
	}
	if got := readResponseBody(discordErrorReader{}); got != "" {
		t.Fatalf("readResponseBody(error) = %q, want empty", got)
	}
	if got := parseRetryAfter("0"); got != 0 {
		t.Fatalf("parseRetryAfter(0) = %s, want 0", got)
	}
	if got := parseRetryAfter("bad"); got != 0 {
		t.Fatalf("parseRetryAfter(bad) = %s, want 0", got)
	}
}

func TestHandleDiscordEventWebhookUsesRequestContext(t *testing.T) {
	t.Parallel()

	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}

	managed := testDiscordManagedInstance("brg-discord")
	runtime := &subprocess.InitializeBridgeRuntime{
		RuntimeVersion: subprocess.InitializeBridgeRuntimeVersion1,
		Provider:       "discord",
		Platform:       "discord",
		ManagedInstances: []subprocess.InitializeBridgeManagedInstance{
			managed,
		},
	}
	provider.mu.Lock()
	provider.session = injectedDiscordSession(t,
		bridgesdk.NewHostAPIClientFromCall(func(ctx context.Context, method string, params any, result any) error {
			if method != "bridges/messages/ingest" {
				return fmt.Errorf("unexpected host api method %q", method)
			}
			if !errors.Is(ctx.Err(), context.Canceled) {
				t.Fatalf("host call ctx.Err() = %v, want context.Canceled", ctx.Err())
			}
			return context.Canceled
		}),
		bridgesdk.NewInstanceCache(runtime),
	)
	provider.mu.Unlock()

	cfg := resolvedInstanceConfig{
		instanceID: "brg-discord",
		managed:    managed,
		dedup:      bridgesdk.NewDedupCache(time.Minute, 16),
		dmPolicy:   bridgepkg.BridgeDMPolicyOpen,
	}
	now := time.Date(2026, 4, 15, 22, 0, 0, 0, time.UTC)
	payload := map[string]any{
		"type": 1,
		"event": map[string]any{
			"id":        "evt-msg-ctx",
			"type":      "MESSAGE_CREATE",
			"timestamp": now.Format(time.RFC3339Nano),
			"data": map[string]any{
				"id":           "msg-ctx",
				"channel_id":   "dm-1",
				"channel_type": 1,
				"content":      "Need context",
				"timestamp":    now.Format(time.RFC3339Nano),
				"author": map[string]any{
					"id":       "user-ctx",
					"username": "alice",
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	recorder := httptest.NewRecorder()
	err = provider.handleEventWebhook(
		recorder,
		httptest.NewRequest(http.MethodPost, "http://discord.test/discord/brg-discord", nil).WithContext(ctx),
		cfg,
		bridgepkgToWebhookRequest(t, payload, now),
	)
	var httpErr *bridgesdk.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("handleEventWebhook(canceled context) error = %v, want HTTP 500", err)
	}
}

func TestReconcileConfigMarkerAndFileHelpers(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	provider.apiFactory = func(resolvedInstanceConfig) discordAPI {
		return &discordAPIFake{postedMessageID: "msg-1"}
	}

	cfg := discordProviderConfig{
		APIBaseURL:    "https://tenant.example.invalid/api/",
		ApplicationID: "bot-1",
	}
	cfg.Webhook.ListenAddr = "127.0.0.1:0"
	cfg.Webhook.Path = "/discord/brg-discord"
	rawConfig, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	managed := subprocess.InitializeBridgeManagedInstance{
		Instance: bridgepkg.BridgeInstance{
			ID:             "brg-discord",
			Scope:          bridgepkg.ScopeWorkspace,
			WorkspaceID:    "ws-1",
			DMPolicy:       bridgepkg.BridgeDMPolicyOpen,
			ProviderConfig: rawConfig,
		},
	}

	configs, err := provider.reconcileInstanceConfigs(context.Background(), &bridgesdk.Session{}, []subprocess.InitializeBridgeManagedInstance{managed})
	if err != nil {
		t.Fatalf("reconcileInstanceConfigs() error = %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("len(configs) = %d, want 1", len(configs))
	}
	if got, want := configs[0].apiBaseURL, discordDefaultAPIBaseURL; got != want {
		t.Fatalf("apiBaseURL = %q, want %q", got, want)
	}
	if provider.server == nil {
		t.Fatal("provider.server = nil, want configured webhook server")
	}
	if got, want := provider.server.ReadHeaderTimeout, discordWebhookReadHeaderTimeout; got != want {
		t.Fatalf("ReadHeaderTimeout = %s, want %s", got, want)
	}
	if got, want := provider.server.IdleTimeout, discordWebhookIdleTimeout; got != want {
		t.Fatalf("IdleTimeout = %s, want %s", got, want)
	}
	if _, err := provider.waitForInstanceConfig("brg-discord", time.Second); err != nil {
		t.Fatalf("waitForInstanceConfig() error = %v", err)
	}
	if _, ok := provider.configForPath("/discord/brg-discord"); !ok {
		t.Fatal("configForPath() ok = false, want true")
	}

	linePath := filepath.Join(tmpDir, "markers", "line.log")
	if err := appendMarkerLine(linePath, "hello"); err != nil {
		t.Fatalf("appendMarkerLine() error = %v", err)
	}
	jsonLinePath := filepath.Join(tmpDir, "markers", "items.jsonl")
	if err := appendJSONLine(jsonLinePath, map[string]string{"hello": "world"}); err != nil {
		t.Fatalf("appendJSONLine() error = %v", err)
	}
	jsonFilePath := filepath.Join(tmpDir, "markers", "state.json")
	if err := writeJSONFile(jsonFilePath, map[string]string{"hello": "world"}); err != nil {
		t.Fatalf("writeJSONFile() error = %v", err)
	}
	crashPath := filepath.Join(tmpDir, "markers", "crash.json")
	if !shouldCrashOnce(crashPath) {
		t.Fatal("shouldCrashOnce(missing) = false, want true")
	}
	if err := os.WriteFile(crashPath, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if shouldCrashOnce(crashPath) {
		t.Fatal("shouldCrashOnce(existing) = true, want false")
	}
}

func TestProviderHostAPIFlowWithInjectedSession(t *testing.T) {
	t.Parallel()

	pub, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	cfg := discordProviderConfig{
		APIBaseURL:    "https://discord.test/api/",
		ApplicationID: "bot-1",
	}
	cfg.Webhook.ListenAddr = "127.0.0.1:0"
	cfg.Webhook.Path = "/discord/brg-discord"
	cfg.Batching.DelayMS = 1
	cfg.DM.AllowUserIDs = []string{"user-allow"}
	rawConfig, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	instance := bridgepkg.BridgeInstance{
		ID:             "brg-discord",
		Scope:          bridgepkg.ScopeWorkspace,
		WorkspaceID:    "ws-1",
		DMPolicy:       bridgepkg.BridgeDMPolicyAllowlist,
		ProviderConfig: rawConfig,
	}
	managed := subprocess.InitializeBridgeManagedInstance{
		Instance: instance,
		BoundSecrets: []subprocess.InitializeBridgeBoundSecret{
			{BindingName: "bot_token", Value: "discord-bot-token"},
			{BindingName: "public_key", Value: hex.EncodeToString(pub)},
		},
	}
	runtime := &subprocess.InitializeBridgeRuntime{
		RuntimeVersion: subprocess.InitializeBridgeRuntimeVersion1,
		Provider:       "discord",
		Platform:       "discord",
		ManagedInstances: []subprocess.InitializeBridgeManagedInstance{
			managed,
		},
	}

	var mu sync.Mutex
	ingests := make([]bridgepkg.InboundMessageEnvelope, 0)
	reportedStates := make([]extensioncontract.BridgesInstancesReportStateParams, 0)
	session := injectedDiscordSession(t,
		bridgesdk.NewHostAPIClientFromCall(func(_ context.Context, method string, params any, result any) error {
			mu.Lock()
			defer mu.Unlock()
			switch method {
			case "bridges/instances/list":
				target := result.(*[]bridgepkg.BridgeInstance)
				*target = []bridgepkg.BridgeInstance{instance}
				return nil
			case "bridges/instances/get":
				target := result.(*bridgepkg.BridgeInstance)
				*target = instance
				return nil
			case "bridges/instances/report_state":
				reportedStates = append(reportedStates, params.(extensioncontract.BridgesInstancesReportStateParams))
				target := result.(*bridgepkg.BridgeInstance)
				updated := instance
				updated.Status = params.(extensioncontract.BridgesInstancesReportStateParams).Status
				updated.Degradation = params.(extensioncontract.BridgesInstancesReportStateParams).Degradation
				*target = updated
				return nil
			case "bridges/messages/ingest":
				ingests = append(ingests, params.(bridgepkg.InboundMessageEnvelope))
				target := result.(*extensioncontract.BridgesMessagesIngestResult)
				*target = extensioncontract.BridgesMessagesIngestResult{SessionID: "sess-1"}
				return nil
			default:
				return fmt.Errorf("unexpected host api method %q", method)
			}
		}),
		bridgesdk.NewInstanceCache(runtime),
	)

	provider, err := newDiscordProvider(io.Discard)
	if err != nil {
		t.Fatalf("newDiscordProvider() error = %v", err)
	}
	provider.apiFactory = func(resolvedInstanceConfig) discordAPI {
		return &discordAPIFake{postedMessageID: "msg-1"}
	}

	if listed, err := provider.syncOwnedInstances(context.Background(), session); err != nil || len(listed) != 1 {
		t.Fatalf("syncOwnedInstances() = (%d, %v), want (1, nil)", len(listed), err)
	}
	if fetched, err := provider.getOwnedInstance(context.Background(), session, instance.ID); err != nil || fetched == nil || fetched.ID != instance.ID {
		t.Fatalf("getOwnedInstance() = (%#v, %v), want instance and nil", fetched, err)
	}

	resolved := provider.resolveInstanceConfig(session, managed)
	if got, want := resolved.botToken, "discord-bot-token"; got != want {
		t.Fatalf("botToken = %q, want %q", got, want)
	}
	if got, want := resolved.publicKey, hex.EncodeToString(pub); got != want {
		t.Fatalf("publicKey = %q, want %q", got, want)
	}
	if got, want := resolved.applicationID, "bot-1"; got != want {
		t.Fatalf("applicationID = %q, want %q", got, want)
	}

	configs, err := provider.reconcileInstanceConfigs(context.Background(), session, []subprocess.InitializeBridgeManagedInstance{managed})
	if err != nil {
		t.Fatalf("reconcileInstanceConfigs() error = %v", err)
	}
	defer func() {
		if provider.server != nil {
			_ = provider.server.Close()
		}
		provider.stop()
	}()
	if len(configs) != 1 {
		t.Fatalf("len(configs) = %d, want 1", len(configs))
	}

	provider.mu.Lock()
	provider.session = session
	provider.mu.Unlock()

	message := bridgepkg.InboundMessageEnvelope{
		BridgeInstanceID:  instance.ID,
		Scope:             instance.Scope,
		WorkspaceID:       instance.WorkspaceID,
		GroupID:           "channel-1",
		ThreadID:          "thread-1",
		PlatformMessageID: "msg-in-1",
		ReceivedAt:        time.Now().UTC(),
		Sender:            bridgepkg.MessageSender{ID: "user-allow"},
		Content:           bridgepkg.MessageContent{Text: "hello"},
		EventFamily:       bridgepkg.InboundEventFamilyMessage,
		IdempotencyKey:    "idem-1",
	}
	if err := provider.dispatchInboundEnvelope(context.Background(), instance.ID, message); err != nil {
		t.Fatalf("dispatchInboundEnvelope() error = %v", err)
	}
	if len(ingests) != 1 {
		t.Fatalf("len(ingests) = %d, want 1", len(ingests))
	}

	if err := provider.dispatchInboundBatch(context.Background(), instance.ID, bridgesdk.InboundBatch{
		Items: []bridgepkg.InboundMessageEnvelope{message, {
			BridgeInstanceID:  instance.ID,
			Scope:             instance.Scope,
			WorkspaceID:       instance.WorkspaceID,
			GroupID:           "channel-1",
			ThreadID:          "thread-1",
			PlatformMessageID: "msg-in-2",
			ReceivedAt:        time.Now().UTC(),
			Sender:            bridgepkg.MessageSender{ID: "user-allow"},
			Content:           bridgepkg.MessageContent{Text: "world"},
			EventFamily:       bridgepkg.InboundEventFamilyMessage,
			IdempotencyKey:    "idem-2",
		}},
	}); err != nil {
		t.Fatalf("dispatchInboundBatch() error = %v", err)
	}
	if len(ingests) != 2 {
		t.Fatalf("len(ingests) = %d, want 2", len(ingests))
	}
	if got, want := ingests[1].Content.Text, "hello\nworld"; got != want {
		t.Fatalf("batched text = %q, want %q", got, want)
	}

	req := bridgepkg.DeliveryRequest{
		Event: bridgepkg.DeliveryEvent{
			DeliveryID:       "del-1",
			BridgeInstanceID: instance.ID,
			RoutingKey: bridgepkg.RoutingKey{
				Scope:            bridgepkg.ScopeWorkspace,
				WorkspaceID:      "ws-1",
				BridgeInstanceID: instance.ID,
				GroupID:          "channel-1",
				ThreadID:         "thread-1",
			},
			DeliveryTarget: bridgepkg.DeliveryTarget{
				BridgeInstanceID: instance.ID,
				GroupID:          "channel-1",
				ThreadID:         "thread-1",
				Mode:             bridgepkg.DeliveryModeReply,
			},
			Seq:       1,
			EventType: bridgepkg.DeliveryEventTypeStart,
			Content:   bridgepkg.MessageContent{Text: "hello"},
		},
	}
	ack, err := provider.handleBridgesDeliver(context.Background(), session, req)
	if err != nil {
		t.Fatalf("handleBridgesDeliver() error = %v", err)
	}
	if got, want := ack.RemoteMessageID, "thread-1:msg-1"; got != want {
		t.Fatalf("ack.RemoteMessageID = %q, want %q", got, want)
	}
	if got := provider.deliveryState(instance.ID, "del-1").RemoteMessageID; got == "" {
		t.Fatal("deliveryState().RemoteMessageID = empty, want stored value")
	}

	if len(reportedStates) == 0 {
		t.Fatal("reportedStates = 0, want at least one state report")
	}
}

type discordAPIFake struct {
	postedMessageID string
	postErr         error
	updateErr       error
	deleteErr       error

	postRequests   []discordPostMessageRequest
	updateRequests []discordUpdateMessageRequest
	deleteRequests []discordDeleteMessageRequest
}

func (f *discordAPIFake) GetBotUser(context.Context) (*discordBotIdentity, error) {
	return &discordBotIdentity{ID: "bot-1", Username: "agh"}, nil
}

func (f *discordAPIFake) PostMessage(_ context.Context, req discordPostMessageRequest) (*discordPostedMessage, error) {
	if f.postErr != nil {
		return nil, f.postErr
	}
	f.postRequests = append(f.postRequests, req)
	return &discordPostedMessage{ID: f.postedMessageID}, nil
}

func (f *discordAPIFake) UpdateMessage(_ context.Context, req discordUpdateMessageRequest) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updateRequests = append(f.updateRequests, req)
	return nil
}

func (f *discordAPIFake) DeleteMessage(_ context.Context, req discordDeleteMessageRequest) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleteRequests = append(f.deleteRequests, req)
	return nil
}

func testDiscordManagedInstance(id string) subprocess.InitializeBridgeManagedInstance {
	return subprocess.InitializeBridgeManagedInstance{
		Instance: bridgepkg.BridgeInstance{
			ID:          id,
			Scope:       bridgepkg.ScopeWorkspace,
			WorkspaceID: "ws-1",
			DMPolicy:    bridgepkg.BridgeDMPolicyOpen,
		},
	}
}

func bridgepkgToWebhookRequest(t *testing.T, payload any, receivedAt time.Time) bridgesdk.WebhookRequest {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return bridgesdk.WebhookRequest{
		Body:       body,
		ReceivedAt: receivedAt,
	}
}

type rpcCodeErr struct{}

func (rpcCodeErr) Error() string { return "not initialized" }

func (rpcCodeErr) Code() int { return rpcCodeNotInitialized }

type discordAPIGetBotUserErrorFake struct {
	err error
}

func (f *discordAPIGetBotUserErrorFake) GetBotUser(context.Context) (*discordBotIdentity, error) {
	return nil, f.err
}

func (f *discordAPIGetBotUserErrorFake) PostMessage(context.Context, discordPostMessageRequest) (*discordPostedMessage, error) {
	return nil, f.err
}

func (f *discordAPIGetBotUserErrorFake) UpdateMessage(context.Context, discordUpdateMessageRequest) error {
	return f.err
}

func (f *discordAPIGetBotUserErrorFake) DeleteMessage(context.Context, discordDeleteMessageRequest) error {
	return f.err
}

type discordErrorReader struct{}

func (discordErrorReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func injectedDiscordSession(t *testing.T, host *bridgesdk.HostAPIClient, cache *bridgesdk.InstanceCache) *bridgesdk.Session {
	t.Helper()

	session := &bridgesdk.Session{}
	sessionValue := reflect.ValueOf(session).Elem()
	setUnexportedField(t, sessionValue.FieldByName("host"), host)
	setUnexportedField(t, sessionValue.FieldByName("cache"), cache)
	setUnexportedField(t, sessionValue.FieldByName("now"), func() time.Time { return time.Now().UTC() })
	return session
}

func setUnexportedField(t *testing.T, field reflect.Value, value any) {
	t.Helper()

	if !field.IsValid() {
		t.Fatal("setUnexportedField() received invalid field")
	}
	replacement := reflect.ValueOf(value)
	if !replacement.Type().AssignableTo(field.Type()) {
		t.Fatalf("setUnexportedField() type mismatch: got %s want %s", replacement.Type(), field.Type())
	}
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(replacement)
}
