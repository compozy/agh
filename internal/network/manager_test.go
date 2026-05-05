package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	sessionpkg "github.com/pedronauck/agh/internal/session"
)

func nilTestContext() context.Context {
	var ctx context.Context
	return ctx
}

func testJoinRequest(
	sessionID string,
	peerID string,
	channel string,
	capabilities ...sessionpkg.NetworkPeerCapability,
) sessionpkg.NetworkPeerJoin {
	return sessionpkg.NetworkPeerJoin{
		SessionID:    sessionID,
		PeerID:       peerID,
		Channel:      channel,
		Capabilities: append([]sessionpkg.NetworkPeerCapability(nil), capabilities...),
	}
}

func waitForCondition(ctx context.Context, t *testing.T, condition func() bool, description string) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s: %v", description, ctx.Err())
		case <-ticker.C:
		}
	}
}

func TestNewManagerRequiresEnabledConfigAndPrompter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ctx      context.Context
		cfg      aghconfig.NetworkConfig
		prompter deliveryPrompter
		wantErr  string
	}{
		{
			name: "Should reject disabled config",
			ctx:  context.Background(),
			cfg: func() aghconfig.NetworkConfig {
				cfg := testManagerConfig()
				cfg.Enabled = false
				return cfg
			}(),
			prompter: newFakeDeliveryPrompter(),
			wantErr:  "enabled network config is required",
		},
		{
			name:     "Should reject nil prompter",
			ctx:      context.Background(),
			cfg:      testManagerConfig(),
			prompter: nil,
			wantErr:  "session prompter is required",
		},
		{
			name:     "Should reject nil context",
			ctx:      nilTestContext(),
			cfg:      testManagerConfig(),
			prompter: newFakeDeliveryPrompter(),
			wantErr:  "manager context is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewManager(tc.ctx, tc.cfg, tc.prompter, filepath.Join(t.TempDir(), "audit.jsonl"), nil)
			if err == nil {
				t.Fatalf("NewManager() error = nil, want %q", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("NewManager() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestNewManagerReportsRollbackShutdownFailures(t *testing.T) {
	t.Parallel()

	t.Run("Should report rollback shutdown failures", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := NewManager(
			ctx,
			testManagerConfig(),
			newFakeDeliveryPrompter(),
			"",
			nil,
			WithManagerLogger(discardManagerLogger()),
		)
		if err == nil {
			t.Fatal("NewManager() error = nil, want rollback failure")
		}
		if !strings.Contains(err.Error(), "audit sink is required") {
			t.Fatalf("NewManager() error = %v, want audit sink failure", err)
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("NewManager() error = %v, want wrapped context cancellation from rollback shutdown", err)
		}
	})
}

func TestManagerJoinSendStatusAndLeave(t *testing.T) {
	t.Parallel()

	newManagerHarness := func(t *testing.T) (context.Context, *Manager, *fakeDeliveryPrompter) {
		t.Helper()

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		prompter := newFakeDeliveryPrompter()
		manager, err := NewManager(
			ctx,
			testManagerConfig(),
			prompter,
			filepath.Join(t.TempDir(), "network.audit"),
			nil,
			WithManagerLogger(discardManagerLogger()),
		)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
		})

		return ctx, manager, prompter
	}

	t.Run("Should report initial running status", func(t *testing.T) {
		t.Parallel()

		ctx, manager, _ := newManagerHarness(t)

		status, err := manager.Status(ctx)
		if err != nil {
			t.Fatalf("Status(initial) error = %v", err)
		}
		if !status.Enabled || status.Status != StatusRunning || status.ListenerPort <= 0 {
			t.Fatalf("Status(initial) = %#v, want enabled running listener", status)
		}
	})

	t.Run("Should join send and drain deliveries", func(t *testing.T) {
		t.Parallel()

		ctx, manager, prompter := newManagerHarness(t)
		if err := manager.JoinChannel(ctx, testJoinRequest("sess-a", "coder.sess-a", "builders")); err != nil {
			t.Fatalf("JoinChannel() error = %v", err)
		}
		if err := manager.JoinChannel(ctx, testJoinRequest("sess-b", "reviewer.sess-b", "builders")); err != nil {
			t.Fatalf("JoinChannel(second peer) error = %v", err)
		}

		status, err := manager.Status(ctx)
		if err != nil {
			t.Fatalf("Status(joined) error = %v", err)
		}
		if status.LocalPeers != 2 || status.Channels != 1 {
			t.Fatalf("Status(joined) = %#v, want 2 local peers and 1 channel", status)
		}

		id, err := manager.Send(ctx, withTestConversation(SendRequest{
			SessionID: "sess-a",
			Channel:   "builders",
			Kind:      KindSay,
			Body:      mustRawJSON(t, map[string]any{"text": "hello builders"}),
		}))
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		if strings.TrimSpace(id) == "" {
			t.Fatal("Send() id = empty, want generated message id")
		}

		prompter.waitForCalls(t, 1)
		call := prompter.call(0)
		if got, want := call.sessionID, "sess-b"; got != want {
			t.Fatalf("prompt session id = %q, want %q", got, want)
		}
		if !strings.Contains(call.message, "hello builders") {
			t.Fatalf("prompt message = %q, want rendered network preview", call.message)
		}
		prompter.finishCall(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
		manager.deliveries.wait()

		inbox, err := manager.Inbox(ctx, "sess-a")
		if err != nil {
			t.Fatalf("Inbox() error = %v", err)
		}
		if len(inbox) != 0 {
			t.Fatalf("Inbox() = %#v, want empty after immediate delivery", inbox)
		}
	})

	t.Run("Should leave channel idempotently and clear status", func(t *testing.T) {
		t.Parallel()

		ctx, manager, _ := newManagerHarness(t)
		if err := manager.JoinChannel(ctx, testJoinRequest("sess-a", "coder.sess-a", "builders")); err != nil {
			t.Fatalf("JoinChannel() error = %v", err)
		}
		if err := manager.LeaveChannel(ctx, "sess-a"); err != nil {
			t.Fatalf("LeaveChannel() error = %v", err)
		}
		if err := manager.LeaveChannel(ctx, "sess-a"); err != nil {
			t.Fatalf("LeaveChannel(repeated) error = %v, want nil", err)
		}

		status, err := manager.Status(ctx)
		if err != nil {
			t.Fatalf("Status(left) error = %v", err)
		}
		if status.LocalPeers != 0 || status.Channels != 0 {
			t.Fatalf("Status(left) = %#v, want zero local peers and channels", status)
		}
	})
}

func TestPrepareJoinLocalPeerUsesCapabilityAwareRuntimeInput(t *testing.T) {
	t.Parallel()

	peers, err := NewPeerRegistry(time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	now := time.Date(2026, 4, 19, 3, 0, 0, 0, time.UTC)
	manager := &Manager{
		peers:    peers,
		sessions: make(map[string]*managedSession),
		now: func() time.Time {
			return now
		},
	}

	t.Run("Should build local peer card from capability-aware runtime input", func(t *testing.T) {
		capabilities := []sessionpkg.NetworkPeerCapability{{
			ID:                "review-pr",
			Summary:           "Review pull requests",
			Outcome:           "Actionable review findings",
			ContextNeeded:     []string{"pull request link"},
			ArtifactsExpected: []string{"review summary"},
		}}
		local, alreadyJoined, err := manager.prepareJoinLocalPeer(context.Background(), joinChannelRequest{
			sessionID:    "sess-capabilities",
			peerID:       "reviewer.sess-capabilities",
			channel:      "builders",
			capabilities: capabilities,
		})
		if err != nil {
			t.Fatalf("prepareJoinLocalPeer() error = %v", err)
		}
		if alreadyJoined {
			t.Fatal("prepareJoinLocalPeer() alreadyJoined = true, want false")
		}
		if got, want := local.PeerCard.Capabilities, []string{"review-pr"}; !slices.Equal(got, want) {
			t.Fatalf("local peer capabilities = %#v, want %#v", got, want)
		}
		if got, want := local.PeerCard.ArtifactsSupported, []string{"capability"}; !slices.Equal(got, want) {
			t.Fatalf("local peer artifacts_supported = %#v, want %#v", got, want)
		}
		if !reflect.DeepEqual(local.CapabilityCatalog, capabilities) {
			t.Fatalf("local capability catalog = %#v, want %#v", local.CapabilityCatalog, capabilities)
		}
		if got := decodeCapabilityBriefPayload(
			t,
			local.PeerCard.Ext[capabilityBriefExtKey],
		); !slices.Equal(
			got,
			[]capabilityBrief{{
				ID:      "review-pr",
				Summary: "Review pull requests",
			}},
		) {
			t.Fatalf("local peer capability brief = %#v, want review-pr brief entry", got)
		}

		stored, ok := manager.peers.LocalBySession("sess-capabilities")
		if !ok {
			t.Fatal("LocalBySession() = missing, want registered local peer")
		}
		if got, want := stored.PeerCard.Capabilities, []string{"review-pr"}; !slices.Equal(got, want) {
			t.Fatalf("stored peer capabilities = %#v, want %#v", got, want)
		}
		if got, want := stored.PeerCard.ArtifactsSupported, []string{"capability"}; !slices.Equal(got, want) {
			t.Fatalf("stored peer artifacts_supported = %#v, want %#v", got, want)
		}
		if !reflect.DeepEqual(stored.CapabilityCatalog, capabilities) {
			t.Fatalf("stored capability catalog = %#v, want %#v", stored.CapabilityCatalog, capabilities)
		}
		if got := decodeCapabilityBriefPayload(
			t,
			stored.PeerCard.Ext[capabilityBriefExtKey],
		); !slices.Equal(
			got,
			[]capabilityBrief{{
				ID:      "review-pr",
				Summary: "Review pull requests",
			}},
		) {
			t.Fatalf("stored peer capability brief = %#v, want review-pr brief entry", got)
		}
	})

	t.Run("Should keep empty capability projection non-nil when runtime supplied none", func(t *testing.T) {
		local, alreadyJoined, err := manager.prepareJoinLocalPeer(context.Background(), joinChannelRequest{
			sessionID:    "sess-empty-capabilities",
			peerID:       "reviewer.sess-empty-capabilities",
			channel:      "builders",
			capabilities: []sessionpkg.NetworkPeerCapability{},
		})
		if err != nil {
			t.Fatalf("prepareJoinLocalPeer() error = %v", err)
		}
		if alreadyJoined {
			t.Fatal("prepareJoinLocalPeer() alreadyJoined = true, want false")
		}
		if local.PeerCard.Capabilities == nil {
			t.Fatal("local peer capabilities = nil, want empty-but-valid slice")
		}
		if got := len(local.PeerCard.Capabilities); got != 0 {
			t.Fatalf("local peer capabilities len = %d, want 0", got)
		}
		if got, want := local.PeerCard.ArtifactsSupported, []string{"capability"}; !slices.Equal(got, want) {
			t.Fatalf("local peer artifacts_supported = %#v, want %#v", got, want)
		}
		if local.PeerCard.Ext != nil && local.PeerCard.Ext[capabilityBriefExtKey] != nil {
			t.Fatalf("local peer ext = %#v, want omitted capability brief key", local.PeerCard.Ext)
		}
		if local.CapabilityCatalog == nil {
			t.Fatal("local capability catalog = nil, want deterministic empty slice")
		}
		if got := len(local.CapabilityCatalog); got != 0 {
			t.Fatalf("local capability catalog len = %d, want 0", got)
		}
	})
}

func TestManagerQueuesBusyDeliveriesTracksDisconnectsAndShutsDownIdempotently(t *testing.T) {
	t.Parallel()

	newBusyManagerHarness := func(t *testing.T) (context.Context, *Manager, *fakeDeliveryPrompter) {
		t.Helper()

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		prompter := newFakeDeliveryPrompter()
		manager, err := NewManager(
			ctx,
			testManagerConfig(),
			prompter,
			filepath.Join(t.TempDir(), "network.audit"),
			nil,
			WithManagerLogger(discardManagerLogger()),
		)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
		})

		if err := manager.JoinChannel(
			ctx,
			testJoinRequest("sess-sender", "coder.sess-sender", "builders"),
		); err != nil {
			t.Fatalf("JoinChannel(sender) error = %v", err)
		}
		if err := manager.JoinChannel(ctx, testJoinRequest("sess-busy", "reviewer.sess-busy", "builders")); err != nil {
			t.Fatalf("JoinChannel(busy) error = %v", err)
		}

		return ctx, manager, prompter
	}

	t.Run("Should queue busy deliveries until turn end", func(t *testing.T) {
		t.Parallel()

		ctx, manager, prompter := newBusyManagerHarness(t)
		prompter.setPrompting("sess-busy", true)
		if _, err := manager.Send(ctx, withTestConversation(SendRequest{
			SessionID: "sess-sender",
			Channel:   "builders",
			Kind:      KindSay,
			Body:      mustRawJSON(t, map[string]any{"text": "queued while busy"}),
		})); err != nil {
			t.Fatalf("Send() error = %v", err)
		}

		waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Second)
		defer waitCancel()
		waitForCondition(waitCtx, t, func() bool {
			return manager.deliveries.queueDepth("sess-busy") == 1
		}, "queued busy delivery")

		inbox, err := manager.Inbox(ctx, "sess-busy")
		if err != nil {
			t.Fatalf("Inbox() error = %v", err)
		}
		if len(inbox) != 1 {
			t.Fatalf("Inbox() len = %d, want 1 queued envelope", len(inbox))
		}

		prompter.setPrompting("sess-busy", false)
		manager.OnTurnEnd("sess-busy")
		prompter.waitForCalls(t, 1)
		call := prompter.call(0)
		if !strings.Contains(call.message, "queued while busy") {
			t.Fatalf("prompt message after turn end = %q, want queued delivery preview", call.message)
		}
		prompter.finishCall(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: time.Now().UTC()})
		manager.deliveries.wait()
	})

	t.Run("Should track disconnect and reconnect status", func(t *testing.T) {
		t.Parallel()

		ctx, manager, _ := newBusyManagerHarness(t)

		manager.handleDisconnect(errors.New("transport lost"))
		status, err := manager.Status(ctx)
		if err != nil {
			t.Fatalf("Status(disconnected) error = %v", err)
		}
		if status.Status != StatusDisconnected {
			t.Fatalf("Status(disconnected) = %#v, want disconnected", status)
		}

		manager.handleReconnect()
		status, err = manager.Status(ctx)
		if err != nil {
			t.Fatalf("Status(reconnected) error = %v", err)
		}
		if status.Status != StatusRunning {
			t.Fatalf("Status(reconnected) = %#v, want running", status)
		}
	})

	t.Run("Should shut down idempotently", func(t *testing.T) {
		t.Parallel()

		_, manager, _ := newBusyManagerHarness(t)

		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown(repeated) error = %v, want nil", err)
		}
	})
}

func TestManagerWaitInboxWakesOnNewChannelMessage(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	prompter := newFakeDeliveryPrompter()
	manager, err := NewManager(
		ctx,
		testManagerConfig(),
		prompter,
		filepath.Join(t.TempDir(), "network.audit"),
		nil,
		WithManagerLogger(discardManagerLogger()),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	if err := manager.JoinChannel(ctx, testJoinRequest("sess-a", "coder.sess-a", "builders")); err != nil {
		t.Fatalf("JoinChannel(sess-a) error = %v", err)
	}
	if err := manager.JoinChannel(ctx, testJoinRequest("sess-b", "reviewer.sess-b", "builders")); err != nil {
		t.Fatalf("JoinChannel(sess-b) error = %v", err)
	}
	prompter.setPrompting("sess-a", true)
	prompter.setPrompting("sess-b", true)

	waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Second)
	defer waitCancel()
	resultCh := make(chan []Envelope, 1)
	errCh := make(chan error, 1)
	go func() {
		messages, waitErr := manager.WaitInbox(waitCtx, "sess-b", "builders")
		if waitErr != nil {
			errCh <- waitErr
			return
		}
		resultCh <- messages
	}()

	messageID, err := manager.Send(ctx, withTestConversation(SendRequest{
		SessionID: "sess-a",
		Channel:   "builders",
		Kind:      KindSay,
		Body:      mustRawJSON(t, map[string]any{"text": "wake reviewer"}),
	}))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	select {
	case err := <-errCh:
		t.Fatalf("WaitInbox() error = %v", err)
	case messages := <-resultCh:
		if len(messages) != 1 {
			t.Fatalf("WaitInbox() len = %d, want 1", len(messages))
		}
		if messages[0].ID != messageID || messages[0].Channel != "builders" {
			t.Fatalf("WaitInbox() message = %#v, want sent builders message %q", messages[0], messageID)
		}
	case <-waitCtx.Done():
		t.Fatalf("WaitInbox() did not wake before timeout: %v", waitCtx.Err())
	}
}

func TestManagerAuditsBusyQueueOverflowAsRejected(t *testing.T) {
	t.Run("ShouldAuditBusyQueueOverflowAsRejected", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		cfg := testManagerConfig()
		cfg.MaxQueueDepth = 1
		prompter := newFakeDeliveryPrompter()
		auditor := &recordingAuditWriter{}
		manager, err := NewManager(
			ctx,
			cfg,
			prompter,
			"",
			nil,
			WithManagerAuditWriter(auditor),
			WithManagerLogger(discardManagerLogger()),
		)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
		})

		if err := manager.JoinChannel(
			ctx,
			testJoinRequest("sess-sender", "coder.sess-sender", "builders"),
		); err != nil {
			t.Fatalf("JoinChannel(sender) error = %v", err)
		}
		if err := manager.JoinChannel(ctx, testJoinRequest("sess-busy", "reviewer.sess-busy", "builders")); err != nil {
			t.Fatalf("JoinChannel(busy) error = %v", err)
		}
		prompter.setPrompting("sess-busy", true)

		firstID := receiveTestEnvelope(t, manager, SendRequest{
			ID:        ptrString("msg-overflow-first"),
			SessionID: "sess-sender",
			Channel:   "builders",
			Kind:      KindSay,
			Body:      mustRawJSON(t, map[string]any{"text": "overflow first"}),
		})
		secondID := receiveTestEnvelope(t, manager, SendRequest{
			ID:        ptrString("msg-overflow-second"),
			SessionID: "sess-sender",
			Channel:   "builders",
			Kind:      KindSay,
			Body:      mustRawJSON(t, map[string]any{"text": "overflow second"}),
		})

		waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Second)
		defer waitCancel()
		waitForCondition(waitCtx, t, func() bool {
			inbox, inboxErr := manager.Inbox(ctx, "sess-busy")
			return inboxErr == nil && len(inbox) == 1 && inbox[0].ID == secondID
		}, "queue overflow to retain newest message")

		rejected := auditor.rejectedForMessage(firstID)
		if len(rejected) != 1 {
			t.Fatalf("rejected audit count for dropped message %q = %d, want 1", firstID, len(rejected))
		}
		if got, want := rejected[0].reason, "queue_overflow"; got != want {
			t.Fatalf("rejected audit reason = %q, want %q", got, want)
		}

		status, err := manager.Status(ctx)
		if err != nil {
			t.Fatalf("Status(after overflow) error = %v", err)
		}
		if got, want := status.MessagesRejected, int64(1); got != want {
			t.Fatalf("Status.MessagesRejected = %d, want %d", got, want)
		}
	})
}

func TestManagerRejectsBogusWhoisFloodWithoutResourceGrowth(t *testing.T) {
	t.Run("Should reject bogus whois flood while staying responsive", func(t *testing.T) {
		// This scenario captures process-wide goroutine and heap baselines, so it
		// must remain serial while other package tests are paused.
		ctx := t.Context()
		fixedNow := time.Date(2026, 5, 3, 9, 30, 0, 0, time.UTC)
		prompter := newFakeDeliveryPrompter()
		auditor := &recordingAuditWriter{}
		manager, err := NewManager(
			ctx,
			testManagerConfig(),
			prompter,
			filepath.Join(t.TempDir(), "network.audit"),
			nil,
			WithManagerLogger(discardManagerLogger()),
			WithManagerClock(func() time.Time { return fixedNow }),
			WithManagerAuditWriter(auditor),
		)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
		})

		if err := manager.JoinChannel(
			ctx,
			testJoinRequest("sess-sender", "coder.sess-sender", "builders"),
		); err != nil {
			t.Fatalf("JoinChannel(sender) error = %v", err)
		}
		if err := manager.JoinChannel(
			ctx,
			testJoinRequest("sess-receiver", "reviewer.sess-receiver", "builders"),
		); err != nil {
			t.Fatalf("JoinChannel(receiver) error = %v", err)
		}
		auditor.reset()

		runtime.GC()
		baselineGoroutines := runtime.NumGoroutine()
		var baselineMemory runtime.MemStats
		runtime.ReadMemStats(&baselineMemory)

		baselineP95 := p95Duration(collectStatusLatencies(ctx, t, manager, 25))

		const floodCount = 10_000
		bogusWhoisPayload, err := json.Marshal(Envelope{
			Protocol: ProtocolV0,
			ID:       "msg-bogus-whois",
			Kind:     KindWhois,
			Channel:  "builders",
			From:     "attacker.sess-bogus",
			TS:       fixedNow.Unix(),
			Body: mustRawJSON(t, map[string]any{
				"type": "bogus-peer-card-request",
			}),
		})
		if err != nil {
			t.Fatalf("json.Marshal(bogus whois envelope) error = %v", err)
		}

		started := make(chan struct{})
		done := make(chan struct{})
		go func() {
			close(started)
			defer close(done)
			for i := range floodCount {
				manager.handleInboundMessage(bogusWhoisPayload)
				if i%50 == 0 {
					time.Sleep(2 * time.Millisecond)
				}
			}
		}()

		<-started
		statusLatencies := collectStatusLatenciesUntil(done, func() time.Duration {
			start := time.Now()
			status, statusErr := manager.Status(ctx)
			if statusErr != nil {
				t.Fatalf("Status(during flood) error = %v", statusErr)
			}
			if status.Status != StatusRunning {
				t.Fatalf("Status(during flood).Status = %q, want %q", status.Status, StatusRunning)
			}
			return time.Since(start)
		})
		if len(statusLatencies) == 0 {
			t.Fatal("collected zero status latencies during flood")
		}

		cleanMessageID, err := manager.Send(ctx, withTestConversation(SendRequest{
			ID:        ptrString("msg-clean-during-flood"),
			SessionID: "sess-sender",
			Channel:   "builders",
			Kind:      KindSay,
			Body:      mustRawJSON(t, map[string]any{"text": "clean message during bogus flood"}),
		}))
		if err != nil {
			t.Fatalf("Send(clean during flood) error = %v", err)
		}
		prompter.waitForCalls(t, 1)
		call := prompter.call(0)
		if got, want := call.sessionID, "sess-receiver"; got != want {
			t.Fatalf("clean delivery session = %q, want %q", got, want)
		}
		if !strings.Contains(call.message, "clean message during bogus flood") {
			t.Fatalf("clean delivery prompt = %q, want clean flood message", call.message)
		}
		prompter.finishCall(0)
		manager.deliveries.wait()

		<-done
		waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Second)
		defer waitCancel()
		waitForCondition(waitCtx, t, func() bool {
			return auditor.rejectedCount() == floodCount
		}, "all bogus whois rejections to be audited")

		duringP95 := p95Duration(statusLatencies)
		if duringP95 > 200*time.Millisecond {
			t.Fatalf("Status() p95 during flood = %s, want <= 200ms", duringP95)
		}
		if duringP95 > baselineP95*2 && duringP95 > 10*time.Millisecond {
			t.Fatalf("Status() p95 during flood = %s, baseline p95 = %s, want within 2x", duringP95, baselineP95)
		}

		if got, want := auditor.countReceivedMessage(cleanMessageID), 1; got != want {
			t.Fatalf("clean received audit count = %d, want %d", got, want)
		}
		if got := prompter.callCount(); got != 1 {
			t.Fatalf("prompt calls after bogus flood = %d, want only the clean delivery", got)
		}

		runtime.GC()
		var afterMemory runtime.MemStats
		runtime.ReadMemStats(&afterMemory)
		var delta uint64
		if afterMemory.Alloc > baselineMemory.Alloc {
			delta = afterMemory.Alloc - baselineMemory.Alloc
		}
		if delta > 200*1024*1024 {
			t.Fatalf("heap allocation delta after flood = %d bytes, want < 200 MiB", delta)
		}
		waitForCondition(waitCtx, t, func() bool {
			return runtime.NumGoroutine() <= baselineGoroutines+8
		}, "goroutine count to return near baseline after bogus flood")
	})
}

func receiveTestEnvelope(t *testing.T, manager *Manager, req SendRequest) string {
	t.Helper()

	req = withTestConversation(req)
	requestID := ""
	if req.ID != nil {
		requestID = *req.ID
	}
	envelope, err := manager.router.buildEnvelope(req, time.Now().UTC())
	if err != nil {
		t.Fatalf("buildEnvelope(%q) error = %v", requestID, err)
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("json.Marshal(%q) error = %v", envelope.ID, err)
	}
	manager.handleInboundMessage(payload)
	return envelope.ID
}

func TestCleanupSubscriptionHelpers(t *testing.T) {
	t.Parallel()

	t.Run("ShouldIgnoreConnectionClosedUnsubscribeErrors", func(t *testing.T) {
		t.Parallel()

		if err := cleanupSubscription(
			func() error { return nats.ErrConnectionClosed },
			"network: unsubscribe direct subject for %q: %w",
			"sess-a",
		); err != nil {
			t.Fatalf("cleanupSubscription(connection closed) error = %v, want nil", err)
		}
	})

	t.Run("ShouldWrapDirectSubscriptionCleanupErrors", func(t *testing.T) {
		t.Parallel()

		stopErr := errors.New("unsubscribe failed")
		err := cleanupSubscription(
			func() error { return stopErr },
			"network: unsubscribe direct subject for %q: %w",
			"sess-a",
		)
		if !errors.Is(err, stopErr) {
			t.Fatalf("cleanupSubscription() error = %v, want wrapped unsubscribe failure", err)
		}
		if !strings.Contains(err.Error(), `unsubscribe direct subject for "sess-a"`) {
			t.Fatalf("cleanupSubscription() error = %v, want session context", err)
		}
	})

	t.Run("ShouldRollbackDuplicateBroadcastRefCountWhenCleanupFails", func(t *testing.T) {
		t.Parallel()

		runtime := &managedChannel{channel: "builders", refCount: 2}
		unsubscribeErr := errors.New("duplicate cleanup failed")

		err := cleanupDuplicateBroadcastSubscription("builders", runtime, func() error { return unsubscribeErr })
		if !errors.Is(err, unsubscribeErr) {
			t.Fatalf("cleanupDuplicateBroadcastSubscription() error = %v, want wrapped unsubscribe failure", err)
		}
		if got, want := runtime.refCount, 1; got != want {
			t.Fatalf("duplicate cleanup refCount = %d, want %d", got, want)
		}
		if !strings.Contains(err.Error(), `unsubscribe duplicate broadcast subject for "builders"`) {
			t.Fatalf("cleanupDuplicateBroadcastSubscription() error = %v, want channel context", err)
		}
	})
}

func TestReplayDeadlineClampsExplicitExpiryToReplayWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
	expiresAt := now.Add(10 * time.Minute).Unix()

	deadline := replayDeadline(Envelope{
		TS:        now.Unix(),
		ExpiresAt: &expiresAt,
	}, now, time.Minute)

	if got, want := deadline, now.Add(time.Minute).UTC(); !got.Equal(want) {
		t.Fatalf("replayDeadline(clamped) = %s, want %s", got, want)
	}
}

func TestManagerStatusTracksWorkflowMetricsAndStructuredLogs(t *testing.T) {
	t.Parallel()

	t.Run("Should track workflow metrics and structured logs", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		fixedNow := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
		var logs bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo}))
		prompter := newFakeDeliveryPrompter()
		cfg := testManagerConfig()
		cfg.GreetInterval = 3600

		manager, err := NewManager(
			ctx,
			cfg,
			prompter,
			filepath.Join(t.TempDir(), "network.audit"),
			nil,
			WithManagerLogger(logger),
			WithManagerClock(func() time.Time { return fixedNow }),
		)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		defer func() {
			if err := manager.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
		}()

		if err := manager.JoinChannel(ctx, testJoinRequest("sess-a", "reviewer.sess-a", "builders")); err != nil {
			t.Fatalf("JoinChannel() error = %v", err)
		}
		waitForCondition(ctx, t, func() bool {
			manager.router.mu.Lock()
			defer manager.router.mu.Unlock()
			return len(manager.router.seen) >= 1
		}, "first greet routing")

		if err := manager.JoinChannel(ctx, testJoinRequest("sess-b", "patcher.sess-b", "builders")); err != nil {
			t.Fatalf("JoinChannel(second peer) error = %v", err)
		}
		waitForCondition(ctx, t, func() bool {
			manager.router.mu.Lock()
			defer manager.router.mu.Unlock()
			return len(manager.router.seen) >= 2
		}, "second greet routing")

		_, err = manager.Send(ctx, withTestConversation(SendRequest{
			SessionID:   "sess-a",
			Channel:     "builders",
			Kind:        KindSay,
			Body:        mustRawJSON(t, map[string]any{"text": "hello builders"}),
			ReplyTo:     ptrString("msg-root"),
			TraceID:     ptrString("trace-1"),
			CausationID: ptrString("cause-1"),
			Ext: ExtensionMap{
				"agh.workflow_id":     mustRawJSON(t, "wf-1"),
				"agh.handoff_version": mustRawJSON(t, 3),
			},
		}))
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}

		prompter.waitForCalls(t, 1)
		prompter.finishCall(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: fixedNow})
		manager.deliveries.wait()

		status, err := manager.Status(ctx)
		if err != nil {
			t.Fatalf("Status() error = %v", err)
		}
		if status.MessagesSent != 3 || status.MessagesReceived != 2 || status.MessagesDelivered != 1 {
			t.Fatalf("status message counts = %#v, want sent=3 received=2 delivered=1", status)
		}
		if status.WorkflowTaggedEvents != 3 || status.HandoffTaggedEvents != 3 {
			t.Fatalf("status tagged counts = %#v, want workflow=3 handoff=3", status)
		}
		metricsByKind := make(map[Kind]KindMetric)
		for _, metric := range status.KindMetrics {
			metricsByKind[metric.Kind] = metric
		}
		if greet := metricsByKind[KindGreet]; greet.Sent != 2 || greet.Received != 1 || greet.Delivered != 0 {
			t.Fatalf("greet kind metrics = %#v, want sent=2 received=1 delivered=0", greet)
		}
		if say := metricsByKind[KindSay]; say.Sent != 1 || say.Received != 1 || say.Delivered != 1 {
			t.Fatalf("say kind metrics = %#v, want sent=1 received=1 delivered=1", say)
		}

		logOutput := logs.String()
		for _, want := range []string{
			"network.started",
			"network.message.sent",
			"network.message.delivered",
			"agh.workflow_id=wf-1",
			"agh.handoff_version=3",
			"reply_to=msg-root",
			"trace_id=trace-1",
			"causation_id=cause-1",
		} {
			if !strings.Contains(logOutput, want) {
				t.Fatalf("logs missing %q:\n%s", want, logOutput)
			}
		}
	})
}

func TestManagerDeliversLocalTraceLifecycleMessages(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	fixedNow := time.Date(2026, 4, 11, 18, 30, 0, 0, time.UTC)
	prompter := newFakeDeliveryPrompter()

	manager, err := NewManager(
		ctx,
		testManagerConfig(),
		prompter,
		filepath.Join(t.TempDir(), "network.audit"),
		nil,
		WithManagerLogger(discardManagerLogger()),
		WithManagerClock(func() time.Time { return fixedNow }),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	if err := manager.JoinChannel(ctx, testJoinRequest("sess-a", "reviewer.sess-a", "builders")); err != nil {
		t.Fatalf("JoinChannel(sess-a) error = %v", err)
	}
	if err := manager.JoinChannel(ctx, testJoinRequest("sess-b", "patcher.sess-b", "builders")); err != nil {
		t.Fatalf("JoinChannel(sess-b) error = %v", err)
	}

	directID, err := manager.Send(ctx, withTestConversation(SendRequest{
		SessionID: "sess-b",
		Channel:   "builders",
		Kind:      KindSay,
		To:        ptrString("reviewer.sess-a"),
		Body:      mustRawJSON(t, SayBody{Text: "I can take the migration fix."}),
		WorkID:    ptrString("int-local-trace"),
		TraceID:   ptrString("trace-local-trace"),
	}))
	if err != nil {
		t.Fatalf("Send(direct) error = %v", err)
	}

	prompter.waitForCalls(t, 1)
	directCall := prompter.call(0)
	if got, want := directCall.sessionID, "sess-a"; got != want {
		t.Fatalf("direct delivery session = %q, want %q", got, want)
	}
	if got, want := directCall.meta.Kind, string(KindSay); got != want {
		t.Fatalf("direct delivery kind = %q, want %q", got, want)
	}
	prompter.finishCall(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: fixedNow})
	manager.deliveries.wait()

	_, err = manager.Send(ctx, withTestConversation(SendRequest{
		SessionID: "sess-b",
		Channel:   "builders",
		Kind:      KindTrace,
		To:        ptrString("reviewer.sess-a"),
		Body: mustRawJSON(t, TraceBody{
			State:   WorkStateCompleted,
			Message: "Patch prepared and tests pass.",
			Result:  mustRawJSON(t, map[string]any{"summary": "migration fix applied"}),
		}),
		WorkID:      ptrString("int-local-trace"),
		ReplyTo:     &directID,
		TraceID:     ptrString("trace-local-trace"),
		CausationID: &directID,
	}))
	if err != nil {
		t.Fatalf("Send(trace) error = %v", err)
	}

	prompter.waitForCalls(t, 2)
	traceCall := prompter.call(1)
	if got, want := traceCall.sessionID, "sess-a"; got != want {
		t.Fatalf("trace delivery session = %q, want %q", got, want)
	}
	if got, want := traceCall.meta.Kind, string(KindTrace); got != want {
		t.Fatalf("trace delivery kind = %q, want %q", got, want)
	}
	prompter.finishCall(1, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: fixedNow})
	manager.deliveries.wait()

	status, err := manager.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	metricsByKind := make(map[Kind]KindMetric)
	for _, metric := range status.KindMetrics {
		metricsByKind[metric.Kind] = metric
	}

	if direct := metricsByKind[KindSay]; direct.Sent != 1 || direct.Received != 1 || direct.Delivered != 1 {
		t.Fatalf("direct kind metrics = %#v, want sent=1 received=1 delivered=1", direct)
	}
	if trace := metricsByKind[KindTrace]; trace.Sent != 1 || trace.Received != 1 || trace.Delivered != 1 {
		t.Fatalf("trace kind metrics = %#v, want sent=1 received=1 delivered=1", trace)
	}
}

func TestManagerShutdownTracksInterruptedInFlightMessages(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo}))
	prompter := newFakeDeliveryPrompter()

	manager, err := NewManager(
		ctx,
		testManagerConfig(),
		prompter,
		filepath.Join(t.TempDir(), "network.audit"),
		nil,
		WithManagerLogger(logger),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	if err := manager.JoinChannel(ctx, testJoinRequest("sess-sender", "coder.sess-sender", "builders")); err != nil {
		t.Fatalf("JoinChannel(sender) error = %v", err)
	}
	if err := manager.JoinChannel(ctx, testJoinRequest("sess-stop", "reviewer.sess-stop", "builders")); err != nil {
		t.Fatalf("JoinChannel(stop target) error = %v", err)
	}

	if _, err := manager.Send(ctx, withTestConversation(SendRequest{
		SessionID: "sess-sender",
		Channel:   "builders",
		Kind:      KindSay,
		Body:      mustRawJSON(t, map[string]any{"text": "hello before shutdown"}),
	})); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	prompter.waitForCalls(t, 1)

	status, err := manager.Status(ctx)
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.MessagesDelivered != 0 || status.DeliveryWorkers != 1 {
		t.Fatalf("status before shutdown = %#v, want delivered=0 workers=1", status)
	}

	if err := manager.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	logOutput := logs.String()
	for _, want := range []string{
		"network.message.delivery_interrupted",
		"pending_messages=1",
		"inflight_messages=1",
		"delivery_workers=1",
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("logs missing %q:\n%s", want, logOutput)
		}
	}
	if strings.Contains(logOutput, "network.message.delivered") {
		t.Fatalf("logs unexpectedly reported delivered message:\n%s", logOutput)
	}
}

func TestManagerListsPeersAndAuditsInboundRemoteDeliveries(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	fixedNow := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
	prompter := newFakeDeliveryPrompter()
	auditor := &recordingAuditWriter{}
	manager, err := NewManager(
		ctx,
		testManagerConfig(),
		prompter,
		filepath.Join(t.TempDir(), "network.audit"),
		nil,
		WithManagerLogger(discardManagerLogger()),
		WithManagerClock(func() time.Time { return fixedNow }),
		WithManagerAuditWriter(auditor),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	if err := manager.JoinChannel(ctx, testJoinRequest("sess-local", "reviewer.sess-local", "builders")); err != nil {
		t.Fatalf("JoinChannel() error = %v", err)
	}

	remoteCard, err := DefaultPeerCard("coder.sess-remote")
	if err != nil {
		t.Fatalf("DefaultPeerCard() error = %v", err)
	}
	greetPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg-greet-remote",
		Kind:     KindGreet,
		Channel:  "builders",
		From:     remoteCard.PeerID,
		TS:       fixedNow.Unix(),
		Body:     mustRawJSON(t, GreetBody{PeerCard: remoteCard, Summary: "remote hello"}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(greet envelope) error = %v", err)
	}
	manager.handleInboundMessage(greetPayload)
	auditor.reset()

	peers, err := manager.ListPeers(ctx, "builders")
	if err != nil {
		t.Fatalf("ListPeers() error = %v", err)
	}
	if len(peers) != 2 {
		t.Fatalf("ListPeers() len = %d, want 2 peers", len(peers))
	}

	var remoteSeen *time.Time
	for _, peer := range peers {
		if peer.PeerID == "coder.sess-remote" {
			remoteSeen = peer.LastSeen
		}
	}
	if remoteSeen == nil || !remoteSeen.Equal(fixedNow) {
		t.Fatalf("remote peer last_seen = %v, want %s", remoteSeen, fixedNow)
	}

	channels, err := manager.ListChannels(ctx)
	if err != nil {
		t.Fatalf("ListChannels() error = %v", err)
	}
	if len(channels) != 1 || channels[0].PeerCount != 2 {
		t.Fatalf("ListChannels() = %#v, want one channel with two peers", channels)
	}

	sayPayload, err := json.Marshal(withThreadSurface(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg-say-remote",
		Kind:     KindSay,
		Channel:  "builders",
		From:     remoteCard.PeerID,
		TS:       fixedNow.Unix(),
		Body:     mustRawJSON(t, map[string]any{"text": "remote delivery"}),
	}))
	if err != nil {
		t.Fatalf("json.Marshal(say envelope) error = %v", err)
	}
	manager.handleInboundMessage(sayPayload)

	prompter.waitForCalls(t, 1)
	call := prompter.call(0)
	if !strings.Contains(call.message, "remote delivery") {
		t.Fatalf("prompt message = %q, want inbound remote delivery preview", call.message)
	}
	prompter.finishCall(0, acp.AgentEvent{Type: acp.EventTypeDone, Timestamp: fixedNow})
	manager.deliveries.wait()

	if got, want := auditor.countReceivedMessage("msg-say-remote"), 1; got != want {
		t.Fatalf("received audit count for remote say = %d, want %d", got, want)
	}
	received := auditor.receivedForMessage("msg-say-remote")
	if got, want := received[0].sessionID, "sess-local"; got != want {
		t.Fatalf("received audit session id = %q, want %q", got, want)
	}
}

func TestManagerAuditsGeneratedGreetsAndControlReceivers(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	fixedNow := time.Date(2026, 4, 11, 18, 30, 0, 0, time.UTC)
	auditor := &recordingAuditWriter{}
	manager, err := NewManager(
		ctx,
		testManagerConfig(),
		newFakeDeliveryPrompter(),
		filepath.Join(t.TempDir(), "network.audit"),
		nil,
		WithManagerLogger(discardManagerLogger()),
		WithManagerClock(func() time.Time { return fixedNow }),
		WithManagerAuditWriter(auditor),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	for _, sessionID := range []string{"sess-a", "sess-b", "sess-c"} {
		if err := manager.JoinChannel(ctx, testJoinRequest(sessionID, "coder."+sessionID, "builders")); err != nil {
			t.Fatalf("JoinChannel(%q) error = %v", sessionID, err)
		}
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Second)
	defer waitCancel()
	waitForCondition(waitCtx, t, func() bool {
		return auditor.countSent(KindGreet) >= 3
	}, "initial greet sent audits")

	auditor.reset()
	manager.handleReconnect()
	if got, want := auditor.countSent(KindGreet), 3; got != want {
		t.Fatalf("handleReconnect greet sent audit count = %d, want %d", got, want)
	}
	waitForCondition(waitCtx, t, func() bool {
		return auditor.countReceived(KindGreet) >= 3
	}, "reconnect greet loopback audits")

	auditor.reset()
	remoteCard, err := DefaultPeerCard("reviewer.sess-remote")
	if err != nil {
		t.Fatalf("DefaultPeerCard() error = %v", err)
	}
	greetPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg-greet-control-audit",
		Kind:     KindGreet,
		Channel:  "builders",
		From:     remoteCard.PeerID,
		TS:       fixedNow.Unix(),
		Body:     mustRawJSON(t, GreetBody{PeerCard: remoteCard, Summary: "remote hello"}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(greet envelope) error = %v", err)
	}
	manager.handleInboundMessage(greetPayload)
	waitForCondition(waitCtx, t, func() bool {
		return auditor.countReceivedMessage("msg-greet-control-audit") >= 3
	}, "remote greet received audits")
	if got, want := auditor.countReceivedMessage("msg-greet-control-audit"), 3; got != want {
		t.Fatalf("greet received audit count = %d, want %d", got, want)
	}

	auditor.reset()
	whoisPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg-whois-control-audit",
		Kind:     KindWhois,
		Channel:  "builders",
		From:     remoteCard.PeerID,
		TS:       fixedNow.Unix(),
		Body: mustRawJSON(t, WhoisBody{
			Type:  WhoisTypeRequest,
			Query: "",
		}),
	})
	if err != nil {
		t.Fatalf("json.Marshal(whois envelope) error = %v", err)
	}
	manager.handleInboundMessage(whoisPayload)

	waitForCondition(waitCtx, t, func() bool {
		return auditor.countReceivedMessage("msg-whois-control-audit") >= 3
	}, "whois request received audits")
	if got, want := auditor.countReceivedMessage("msg-whois-control-audit"), 3; got != want {
		t.Fatalf("whois request received audit count = %d, want %d", got, want)
	}
	if got, want := auditor.countSent(KindWhois), 3; got != want {
		t.Fatalf("whois response sent audit count = %d, want %d", got, want)
	}
}

func TestManagerValidationAndNilGuards(t *testing.T) {
	t.Parallel()

	var nilManager *Manager
	if err := nilManager.Shutdown(context.Background()); err != nil {
		t.Fatalf("nil manager Shutdown() error = %v, want nil", err)
	}
	if _, err := nilManager.Status(context.Background()); err == nil {
		t.Fatal("nil manager Status() error = nil, want non-nil")
	}
	if _, err := nilManager.ListPeers(context.Background(), "builders"); err == nil {
		t.Fatal("nil manager ListPeers() error = nil, want non-nil")
	}
	if _, err := nilManager.ListChannels(context.Background()); err == nil {
		t.Fatal("nil manager ListChannels() error = nil, want non-nil")
	}
	if _, err := nilManager.Inbox(context.Background(), "sess"); err == nil {
		t.Fatal("nil manager Inbox() error = nil, want non-nil")
	}
	if err := nilManager.JoinChannel(context.Background(), testJoinRequest("sess", "peer", "builders")); err == nil {
		t.Fatal("nil manager JoinChannel() error = nil, want non-nil")
	}
	if err := nilManager.LeaveChannel(context.Background(), "sess"); err == nil {
		t.Fatal("nil manager LeaveChannel() error = nil, want non-nil")
	}
	if _, err := nilManager.Send(context.Background(), SendRequest{}); err == nil {
		t.Fatal("nil manager Send() error = nil, want non-nil")
	}

	ctx := t.Context()

	manager, err := NewManager(
		ctx,
		testManagerConfig(),
		newFakeDeliveryPrompter(),
		filepath.Join(t.TempDir(), "network.audit"),
		nil,
		WithManagerLogger(discardManagerLogger()),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	defer func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	}()

	if err := manager.JoinChannel(nilTestContext(), testJoinRequest("sess", "peer", "builders")); err == nil {
		t.Fatal("JoinChannel(nil ctx) error = nil, want non-nil")
	}
	if err := manager.LeaveChannel(nilTestContext(), "sess"); err == nil {
		t.Fatal("LeaveChannel(nil ctx) error = nil, want non-nil")
	}
	if _, err := manager.Send(nilTestContext(), SendRequest{}); err == nil {
		t.Fatal("Send(nil ctx) error = nil, want non-nil")
	}
	if _, err := manager.Status(nilTestContext()); err == nil {
		t.Fatal("Status(nil ctx) error = nil, want non-nil")
	}
	if _, err := manager.ListPeers(nilTestContext(), "builders"); err == nil {
		t.Fatal("ListPeers(nil ctx) error = nil, want non-nil")
	}
	if _, err := manager.ListChannels(nilTestContext()); err == nil {
		t.Fatal("ListChannels(nil ctx) error = nil, want non-nil")
	}
	if _, err := manager.Inbox(nilTestContext(), "sess"); err == nil {
		t.Fatal("Inbox(nil ctx) error = nil, want non-nil")
	}
	if err := manager.JoinChannel(context.Background(), testJoinRequest("", "peer", "builders")); err == nil {
		t.Fatal("JoinChannel(missing session) error = nil, want non-nil")
	}
	if err := manager.LeaveChannel(context.Background(), ""); err == nil {
		t.Fatal("LeaveChannel(missing session) error = nil, want non-nil")
	}

	cancelledCtx, canceled := context.WithCancel(context.Background())
	canceled()
	if _, err := manager.Status(cancelledCtx); err == nil {
		t.Fatal("Status(canceled ctx) error = nil, want non-nil")
	}

	if host, port := transportListener(nil); host != "" || port != 0 {
		t.Fatalf("transportListener(nil) = %q,%d want empty,0", host, port)
	}
}

func TestManagerAuditHelpersDelegateToWriter(t *testing.T) {
	t.Parallel()

	auditor := &recordingAuditWriter{}
	manager := &Manager{
		auditor: auditor,
		logger:  discardManagerLogger(),
	}
	envelope := Envelope{ID: "msg-audit", Kind: KindSay}

	manager.recordAuditSent(context.Background(), "sess-a", envelope)
	manager.recordAuditReceived(context.Background(), "sess-a", envelope)
	manager.recordAuditRejected(context.Background(), "sess-a", envelope, "busy")

	if len(auditor.sent) != 1 || len(auditor.received) != 1 || len(auditor.rejected) != 1 {
		t.Fatalf(
			"audit helper counts = sent:%d received:%d rejected:%d, want 1 each",
			len(auditor.sent),
			len(auditor.received),
			len(auditor.rejected),
		)
	}
}

func TestManagerRecordInboundAuditCapturesRejectedAndGeneratedEntries(t *testing.T) {
	t.Parallel()

	peers, err := NewPeerRegistry(time.Second)
	if err != nil {
		t.Fatalf("NewPeerRegistry() error = %v", err)
	}
	card, err := DefaultPeerCard("reviewer.sess-local")
	if err != nil {
		t.Fatalf("DefaultPeerCard() error = %v", err)
	}
	if _, err := peers.RegisterLocal("sess-local", "builders", card, time.Now().UTC()); err != nil {
		t.Fatalf("RegisterLocal() error = %v", err)
	}

	auditor := &recordingAuditWriter{}
	manager := &Manager{
		auditor: auditor,
		logger:  discardManagerLogger(),
		peers:   peers,
	}
	reason := ReasonCodeBusy
	manager.recordInboundAudit(RouteResult{
		Envelope: &Envelope{
			ID:      "msg-rejected",
			Kind:    KindSay,
			Channel: "builders",
			From:    "coder.sess-remote",
		},
		Rejected:   true,
		ReasonCode: &reason,
		Generated: []Envelope{{
			ID:      "msg-receipt",
			Kind:    KindReceipt,
			Channel: "builders",
			From:    "reviewer.sess-local",
		}},
	})

	if len(auditor.rejected) != 1 {
		t.Fatalf("rejected audit count = %d, want 1", len(auditor.rejected))
	}
	if len(auditor.sent) != 1 {
		t.Fatalf("generated sent audit count = %d, want 1", len(auditor.sent))
	}
}

func testManagerConfig() aghconfig.NetworkConfig {
	return aghconfig.NetworkConfig{
		Enabled:        true,
		DefaultChannel: "builders",
		Port:           -1,
		MaxPayload:     1 << 20,
		GreetInterval:  1,
		MaxReplayAge:   300,
		MaxQueueDepth:  8,
	}
}

func discardManagerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type recordingAuditWriter struct {
	mu        sync.Mutex
	sent      []auditCall
	received  []auditCall
	rejected  []auditCall
	delivered []auditCall
}

var _ AuditWriter = (*recordingAuditWriter)(nil)

type auditCall struct {
	sessionID string
	envelope  Envelope
	reason    string
}

func (w *recordingAuditWriter) RecordSent(_ context.Context, sessionID string, envelope Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sent = append(w.sent, auditCall{sessionID: sessionID, envelope: envelope})
	return nil
}

func (w *recordingAuditWriter) RecordReceived(_ context.Context, sessionID string, envelope Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.received = append(w.received, auditCall{sessionID: sessionID, envelope: envelope})
	return nil
}

func (w *recordingAuditWriter) RecordRejected(
	_ context.Context,
	sessionID string,
	envelope Envelope,
	reason string,
) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rejected = append(w.rejected, auditCall{sessionID: sessionID, envelope: envelope, reason: reason})
	return nil
}

func (w *recordingAuditWriter) RecordDelivered(_ context.Context, sessionID string, envelope Envelope) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.delivered = append(w.delivered, auditCall{sessionID: sessionID, envelope: envelope})
	return nil
}

func (w *recordingAuditWriter) countSent(kind Kind) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	count := 0
	for _, call := range w.sent {
		if call.envelope.Kind == kind {
			count++
		}
	}
	return count
}

func (w *recordingAuditWriter) countReceived(kind Kind) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	count := 0
	for _, call := range w.received {
		if call.envelope.Kind == kind {
			count++
		}
	}
	return count
}

func (w *recordingAuditWriter) countReceivedMessage(messageID string) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	count := 0
	for _, call := range w.received {
		if call.envelope.ID == messageID {
			count++
		}
	}
	return count
}

func (w *recordingAuditWriter) receivedForMessage(messageID string) []auditCall {
	w.mu.Lock()
	defer w.mu.Unlock()

	filtered := make([]auditCall, 0)
	for _, call := range w.received {
		if call.envelope.ID == messageID {
			filtered = append(filtered, call)
		}
	}
	return filtered
}

func (w *recordingAuditWriter) rejectedForMessage(messageID string) []auditCall {
	w.mu.Lock()
	defer w.mu.Unlock()

	filtered := make([]auditCall, 0)
	for _, call := range w.rejected {
		if call.envelope.ID == messageID {
			filtered = append(filtered, call)
		}
	}
	return filtered
}

func (w *recordingAuditWriter) rejectedCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.rejected)
}

func (w *recordingAuditWriter) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sent = nil
	w.received = nil
	w.rejected = nil
	w.delivered = nil
}

func collectStatusLatencies(ctx context.Context, t *testing.T, manager *Manager, count int) []time.Duration {
	t.Helper()

	latencies := make([]time.Duration, 0, count)
	for range count {
		start := time.Now()
		status, err := manager.Status(ctx)
		if err != nil {
			t.Fatalf("Status() error = %v", err)
		}
		if status.Status != StatusRunning {
			t.Fatalf("Status().Status = %q, want %q", status.Status, StatusRunning)
		}
		latencies = append(latencies, time.Since(start))
	}
	return latencies
}

func collectStatusLatenciesUntil(done <-chan struct{}, sample func() time.Duration) []time.Duration {
	latencies := make([]time.Duration, 0, 64)
	for {
		select {
		case <-done:
			return latencies
		default:
			latencies = append(latencies, sample())
			time.Sleep(time.Millisecond)
		}
	}
}

func p95Duration(values []time.Duration) time.Duration {
	if len(values) == 0 {
		return 0
	}
	ordered := append([]time.Duration(nil), values...)
	slices.Sort(ordered)
	index := max((len(ordered)*95+99)/100, 1)
	return ordered[index-1]
}
