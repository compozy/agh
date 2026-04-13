package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

func nilTestContext() context.Context {
	var ctx context.Context
	return ctx
}

func waitForCondition(t *testing.T, ctx context.Context, condition func() bool, description string) {
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
		tc := tc
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

		_, err := NewManager(ctx, testManagerConfig(), newFakeDeliveryPrompter(), "", nil, WithManagerLogger(discardManagerLogger()))
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
		if err := manager.JoinChannel(ctx, "sess-a", "coder.sess-a", "builders"); err != nil {
			t.Fatalf("JoinChannel() error = %v", err)
		}

		status, err := manager.Status(ctx)
		if err != nil {
			t.Fatalf("Status(joined) error = %v", err)
		}
		if status.LocalPeers != 1 || status.Channels != 1 {
			t.Fatalf("Status(joined) = %#v, want 1 local peer and 1 channel", status)
		}

		id, err := manager.Send(ctx, SendRequest{
			SessionID: "sess-a",
			Channel:   "builders",
			Kind:      KindSay,
			Body:      mustRawJSON(t, map[string]any{"text": "hello builders"}),
		})
		if err != nil {
			t.Fatalf("Send() error = %v", err)
		}
		if strings.TrimSpace(id) == "" {
			t.Fatal("Send() id = empty, want generated message id")
		}

		prompter.waitForCalls(t, 1)
		call := prompter.call(0)
		if got, want := call.sessionID, "sess-a"; got != want {
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
		if err := manager.JoinChannel(ctx, "sess-a", "coder.sess-a", "builders"); err != nil {
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

		if err := manager.JoinChannel(ctx, "sess-busy", "reviewer.sess-busy", "builders"); err != nil {
			t.Fatalf("JoinChannel() error = %v", err)
		}

		return ctx, manager, prompter
	}

	t.Run("Should queue busy deliveries until turn end", func(t *testing.T) {
		t.Parallel()

		ctx, manager, prompter := newBusyManagerHarness(t)
		prompter.setPrompting("sess-busy", true)
		if _, err := manager.Send(ctx, SendRequest{
			SessionID: "sess-busy",
			Channel:   "builders",
			Kind:      KindSay,
			Body:      mustRawJSON(t, map[string]any{"text": "queued while busy"}),
		}); err != nil {
			t.Fatalf("Send() error = %v", err)
		}

		waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Second)
		defer waitCancel()
		waitForCondition(t, waitCtx, func() bool {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fixedNow := time.Date(2026, 4, 11, 18, 0, 0, 0, time.UTC)
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

	if err := manager.JoinChannel(ctx, "sess-a", "reviewer.sess-a", "builders"); err != nil {
		t.Fatalf("JoinChannel() error = %v", err)
	}

	_, err = manager.Send(ctx, SendRequest{
		SessionID:     "sess-a",
		Channel:       "builders",
		Kind:          KindSay,
		Body:          mustRawJSON(t, map[string]any{"text": "hello builders"}),
		ReplyTo:       ptrString("msg-root"),
		TraceID:       ptrString("trace-1"),
		CausationID:   ptrString("cause-1"),
		InteractionID: ptrString("int-1"),
		Ext: ExtensionMap{
			"agh.workflow_id":     mustRawJSON(t, "wf-1"),
			"agh.handoff_version": mustRawJSON(t, 3),
		},
	})
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
	if status.MessagesSent != 2 || status.MessagesReceived != 2 || status.MessagesDelivered != 1 {
		t.Fatalf("status message counts = %#v, want sent=2 received=2 delivered=1", status)
	}
	if status.WorkflowTaggedEvents != 3 || status.HandoffTaggedEvents != 3 {
		t.Fatalf("status tagged counts = %#v, want workflow=3 handoff=3", status)
	}
	metricsByKind := make(map[Kind]KindMetric)
	for _, metric := range status.KindMetrics {
		metricsByKind[metric.Kind] = metric
	}
	if greet := metricsByKind[KindGreet]; greet.Sent != 1 || greet.Received != 1 || greet.Delivered != 0 {
		t.Fatalf("greet kind metrics = %#v, want sent=1 received=1 delivered=0", greet)
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
}

func TestManagerShutdownTracksInterruptedInFlightMessages(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if err := manager.JoinChannel(ctx, "sess-stop", "reviewer.sess-stop", "builders"); err != nil {
		t.Fatalf("JoinChannel() error = %v", err)
	}

	if _, err := manager.Send(ctx, SendRequest{
		SessionID: "sess-stop",
		Channel:   "builders",
		Kind:      KindSay,
		Body:      mustRawJSON(t, map[string]any{"text": "hello before shutdown"}),
	}); err != nil {
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if err := manager.JoinChannel(ctx, "sess-local", "reviewer.sess-local", "builders"); err != nil {
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

	sayPayload, err := json.Marshal(Envelope{
		Protocol: ProtocolV0,
		ID:       "msg-say-remote",
		Kind:     KindSay,
		Channel:  "builders",
		From:     remoteCard.PeerID,
		TS:       fixedNow.Unix(),
		Body:     mustRawJSON(t, map[string]any{"text": "remote delivery"}),
	})
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		if err := manager.JoinChannel(ctx, sessionID, "coder."+sessionID, "builders"); err != nil {
			t.Fatalf("JoinChannel(%q) error = %v", sessionID, err)
		}
	}

	waitCtx, waitCancel := context.WithTimeout(ctx, 2*time.Second)
	defer waitCancel()
	waitForCondition(t, waitCtx, func() bool {
		return auditor.countSent(KindGreet) >= 3
	}, "initial greet sent audits")

	auditor.reset()
	manager.handleReconnect()
	if got, want := auditor.countSent(KindGreet), 3; got != want {
		t.Fatalf("handleReconnect greet sent audit count = %d, want %d", got, want)
	}
	waitForCondition(t, waitCtx, func() bool {
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
	waitForCondition(t, waitCtx, func() bool {
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

	waitForCondition(t, waitCtx, func() bool {
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
	if err := nilManager.JoinChannel(context.Background(), "sess", "peer", "builders"); err == nil {
		t.Fatal("nil manager JoinChannel() error = nil, want non-nil")
	}
	if err := nilManager.LeaveChannel(context.Background(), "sess"); err == nil {
		t.Fatal("nil manager LeaveChannel() error = nil, want non-nil")
	}
	if _, err := nilManager.Send(context.Background(), SendRequest{}); err == nil {
		t.Fatal("nil manager Send() error = nil, want non-nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if err := manager.JoinChannel(nilTestContext(), "sess", "peer", "builders"); err == nil {
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
	if err := manager.JoinChannel(context.Background(), "", "peer", "builders"); err == nil {
		t.Fatal("JoinChannel(missing session) error = nil, want non-nil")
	}
	if err := manager.LeaveChannel(context.Background(), ""); err == nil {
		t.Fatal("LeaveChannel(missing session) error = nil, want non-nil")
	}

	cancelledCtx, cancelled := context.WithCancel(context.Background())
	cancelled()
	if _, err := manager.Status(cancelledCtx); err == nil {
		t.Fatal("Status(cancelled ctx) error = nil, want non-nil")
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
		t.Fatalf("audit helper counts = sent:%d received:%d rejected:%d, want 1 each", len(auditor.sent), len(auditor.received), len(auditor.rejected))
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
	mu       sync.Mutex
	sent     []auditCall
	received []auditCall
	rejected []auditCall
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

func (w *recordingAuditWriter) RecordRejected(_ context.Context, sessionID string, envelope Envelope, reason string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rejected = append(w.rejected, auditCall{sessionID: sessionID, envelope: envelope, reason: reason})
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

func (w *recordingAuditWriter) reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sent = nil
	w.received = nil
	w.rejected = nil
}
