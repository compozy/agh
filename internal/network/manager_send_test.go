package network

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	aghconfig "github.com/compozy/agh/internal/config"
	"github.com/compozy/agh/internal/network/participation"
	"github.com/compozy/agh/internal/session"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

func TestManagerSendCommitsBeforeDispatch(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 14, 4, 0, 0, 0, time.UTC)
	acceptance := &managerAcceptanceStub{}
	notifier := &managerWakeNotifierStub{}
	manager, err := NewManager(
		t.Context(),
		aghconfig.DefaultNetworkConfig(),
		"",
		nil,
		WithManagerLogger(discardManagerLogger()),
		WithManagerClock(func() time.Time { return now }),
		WithManagerAuditWriter(managerAuditWriterStub{}),
		WithManagerAcceptanceStore(acceptance),
		WithManagerWakeNotifier(notifier),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})

	joinManagerSendParticipant(t, manager, "sess-sender", "sender.sess-abc")
	joinManagerSendParticipant(t, manager, "sess-recipient-b", "reviewer.sess-bbb")
	joinManagerSendParticipant(t, manager, "sess-recipient-a", "reviewer.sess-aaa")

	acceptance.handle = func(req store.AcceptNetworkMessageRequest, call int) (store.AcceptNetworkMessageResult, error) {
		switch call {
		case 1:
			notify := make([]store.CommittedNetworkNotification, 0, len(req.Admissions))
			for index, admission := range req.Admissions {
				notify = append(notify, store.CommittedNetworkNotification{
					RecipientSessionID: admission.RecipientSessionID,
					TaskRunID:          admission.TaskRunID,
					AcceptanceSeq:      int64(index + 1),
				})
			}
			return store.AcceptNetworkMessageResult{
				AcceptanceSeq: 1,
				Dispositions:  slices.Clone(req.Dispositions),
				Notify:        notify,
			}, nil
		case 2:
			return store.AcceptNetworkMessageResult{
				AcceptanceSeq: 1,
				Duplicate:     true,
				Dispositions:  slices.Clone(req.Dispositions),
			}, nil
		default:
			return store.AcceptNetworkMessageResult{}, errors.New("accept transaction failed")
		}
	}

	request := managerThreadSendRequest("msg-manager-commit-1")
	messageID, err := manager.Send(testutil.Context(t), request)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if messageID != "msg-manager-commit-1" {
		t.Fatalf("Send() message id = %q, want committed id", messageID)
	}
	first := acceptance.request(t, 0)
	wantRecipients := []string{"sess-recipient-a", "sess-recipient-b"}
	if got := managerDispositionRecipients(first.Dispositions); !slices.Equal(got, wantRecipients) {
		t.Fatalf("accepted disposition recipients = %#v, want pre-message snapshot %#v", got, wantRecipients)
	}
	if got := managerAdmissionRecipients(first.Admissions); !slices.Equal(got, wantRecipients) {
		t.Fatalf("accepted admission recipients = %#v, want %#v", got, wantRecipients)
	}
	if got := notifier.recipients(); !slices.Equal(got, wantRecipients) {
		t.Fatalf("wake notification order = %#v, want committed order %#v", got, wantRecipients)
	}

	if _, err := manager.Send(testutil.Context(t), request); err != nil {
		t.Fatalf("Send(duplicate) error = %v", err)
	}
	if got := notifier.recipients(); !slices.Equal(got, wantRecipients) {
		t.Fatalf("wake notifications after duplicate = %#v, want no second dispatch", got)
	}

	failed := managerThreadSendRequest("msg-manager-commit-failed")
	if _, err := manager.Send(testutil.Context(t), failed); err == nil {
		t.Fatal("Send(accept failure) error = nil, want typed failure")
	}
	if got := notifier.recipients(); !slices.Equal(got, wantRecipients) {
		t.Fatalf("wake notifications after failed accept = %#v, want committed notifications only", got)
	}
	stats := manager.stats.snapshot()
	if stats.MessagesSent != 1 || stats.MessagesReceived != 2 || stats.MessagesDelivered != 2 {
		t.Fatalf(
			"post-commit message stats = sent:%d received:%d delivered:%d, want 1/2/2",
			stats.MessagesSent,
			stats.MessagesReceived,
			stats.MessagesDelivered,
		)
	}
}

func TestManagerWakeAdmissionEligibility(t *testing.T) {
	t.Parallel()

	manager, err := NewManager(
		t.Context(),
		aghconfig.DefaultNetworkConfig(),
		"",
		nil,
		WithManagerLogger(discardManagerLogger()),
		WithManagerAuditWriter(managerAuditWriterStub{}),
		WithManagerAcceptanceStore(&managerAcceptanceStub{}),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Shutdown(context.Background()); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	})
	joinManagerSendParticipant(t, manager, "sess-recipient", "reviewer.sess-target")

	tests := []struct {
		name      string
		kind      Kind
		direct    bool
		mention   bool
		eligible  bool
		addressed bool
		trigger   string
	}{
		{
			name: "direct say", kind: KindSay, direct: true,
			eligible: true, addressed: true, trigger: store.NetworkWakeTriggerDirect,
		},
		{
			name: "mentioned say", kind: KindSay, mention: true,
			eligible: true, addressed: true, trigger: store.NetworkWakeTriggerMention,
		},
		{name: "unaddressed say", kind: KindSay, eligible: true},
		{name: "greet control", kind: KindGreet, direct: true},
		{name: "whois control", kind: KindWhois, direct: true},
		{name: "capability status", kind: KindCapability, direct: true},
		{name: "receipt control", kind: KindReceipt, direct: true},
		{name: "trace control", kind: KindTrace, direct: true},
	}
	for _, test := range tests {
		t.Run("Should classify "+test.name, func(t *testing.T) {
			envelope := Envelope{Kind: test.kind}
			if test.direct {
				target := "reviewer.sess-target"
				envelope.To = &target
			}
			if test.mention {
				envelope.Mentions = []string{"reviewer.sess-target"}
			}
			admissions := manager.wakeAdmissions([]Delivery{{
				SessionID: "sess-recipient",
				PeerID:    "reviewer.sess-target",
				Envelope:  envelope,
			}}, "root-eligibility", 0)
			if len(admissions) != 1 {
				t.Fatalf("wakeAdmissions() = %#v, want one recipient decision", admissions)
			}
			admission := admissions[0]
			if admission.Eligible != test.eligible || admission.Addressed != test.addressed ||
				admission.Trigger != test.trigger {
				t.Fatalf("wake admission = %#v, want eligible=%t addressed=%t trigger=%q",
					admission,
					test.eligible,
					test.addressed,
					test.trigger,
				)
			}
			if test.addressed {
				if admission.WakeID == "" || admission.TaskRunID == "" {
					t.Fatalf("addressed wake IDs = (%q, %q), want allocated IDs", admission.WakeID, admission.TaskRunID)
				}
				return
			}
			if admission.WakeID != "" || admission.TaskRunID != "" {
				t.Fatalf("non-addressed wake IDs = (%q, %q), want none", admission.WakeID, admission.TaskRunID)
			}
		})
	}
}

func TestManagerSendRejectsRawCredentialsBeforeAcceptance(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	acceptance := &managerAcceptanceStub{}
	manager, err := NewManager(
		t.Context(),
		aghconfig.DefaultNetworkConfig(),
		"",
		nil,
		WithManagerLogger(slog.New(slog.NewJSONHandler(&logs, nil))),
		WithManagerAuditWriter(managerAuditWriterStub{}),
		WithManagerAcceptanceStore(acceptance),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	t.Cleanup(func() {
		if shutdownErr := manager.Shutdown(context.Background()); shutdownErr != nil {
			t.Fatalf("Shutdown() error = %v", shutdownErr)
		}
	})
	joinManagerSendParticipant(t, manager, "sess-sender", "sender.sess-abc")
	logs.Reset()

	rawToken := "agh_claim_MANAGER_SECURITY_123"
	tests := []struct {
		name string
		body json.RawMessage
		ext  ExtensionMap
	}{
		{
			name: "body",
			body: json.RawMessage(`{"text":"unsafe","nested":{"claim_token":"` + rawToken + `"}}`),
		},
		{
			name: "extension metadata",
			body: json.RawMessage(`{"text":"unsafe ext"}`),
			ext: ExtensionMap{
				"agh.metadata": json.RawMessage(`{"claim_token":"` + rawToken + `"}`),
			},
		},
	}
	for _, test := range tests {
		t.Run("Should reject raw claim token in "+test.name, func(t *testing.T) {
			request := managerThreadSendRequest("msg-security-" + strings.ReplaceAll(test.name, " ", "-"))
			request.Body = test.body
			request.Ext = test.ext
			if _, sendErr := manager.Send(testutil.Context(t), request); !errors.Is(sendErr, ErrInvalidBody) {
				t.Fatalf("Send() error = %v, want ErrInvalidBody", sendErr)
			}
		})
	}

	if got := acceptance.callCount(); got != 0 {
		t.Fatalf("AcceptNetworkMessage calls = %d, want 0", got)
	}
	logOutput := logs.String()
	if strings.Contains(logOutput, rawToken) {
		t.Fatalf("pre-persistence logs leaked raw claim token: %s", logOutput)
	}
	if got := strings.Count(logOutput, "network.message.rejected_pre_persistence"); got != len(tests) {
		t.Fatalf("pre-persistence rejection log count = %d, want %d; logs=%s", got, len(tests), logOutput)
	}
	for line := range strings.SplitSeq(strings.TrimSpace(logOutput), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("json.Unmarshal(log) error = %v; line=%s", err, line)
		}
		digest, ok := record["payload_sha256"].(string)
		if !ok {
			t.Fatalf("rejection log missing payload_sha256: %#v", record)
		}
		decoded, err := hex.DecodeString(digest)
		if err != nil || len(decoded) != 32 {
			t.Fatalf("payload_sha256 = %q, want 32-byte SHA-256; error=%v", digest, err)
		}
	}
}

func joinManagerSendParticipant(t *testing.T, manager *Manager, sessionID string, peerID string) {
	t.Helper()
	spec := managerSendLiveSpec()
	if err := manager.JoinChannel(testutil.Context(t), session.NetworkPeerJoin{
		SessionID:            sessionID,
		PeerID:               peerID,
		WorkspaceID:          testWorkspaceID,
		Channel:              "builders",
		OwnerKey:             "session:" + sessionID,
		NetworkParticipation: spec,
	}); err != nil {
		t.Fatalf("JoinChannel(%q) error = %v", sessionID, err)
	}
}

func managerSendLiveSpec() participation.Spec {
	defaults := aghconfig.DefaultNetworkConfig().Live.Defaults
	return participation.Spec{
		Version: participation.SpecVersion, Mode: participation.ModeLive,
		WorkspaceID: testWorkspaceID, ChannelStrategy: participation.StrategyNamed,
		ChannelID: "builders", Source: participation.SourceExplicitRequest,
		Bounds: participation.Bounds{
			MaxWakes: defaults.MaxWakes, MaxWakeWallTime: defaults.MaxWakeWallTime,
			MaxTotalWallTime: defaults.MaxTotalWallTime, MaxInputTokens: defaults.MaxInputTokens,
			MaxOutputTokens: defaults.MaxOutputTokens, MaxWakeDepth: defaults.MaxWakeDepth,
			CoalesceWindow: defaults.CoalesceWindow,
		},
	}
}

func managerThreadSendRequest(messageID string) SendRequest {
	surface := SurfaceThread
	threadID := "thread_manager_commit"
	return SendRequest{
		SessionID: "sess-sender", WorkspaceID: testWorkspaceID, Channel: "builders",
		Surface: &surface, ThreadID: &threadID, Kind: KindSay,
		Body: json.RawMessage(`{"text":"review this"}`), ID: &messageID,
	}
}

func managerDispositionRecipients(values []store.NetworkMessageDisposition) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value.Decision == store.NetworkDispositionDeliver {
			result = append(result, value.RecipientSessionID)
		}
	}
	return result
}

func managerAdmissionRecipients(values []store.NetworkWakeAdmissionInput) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.RecipientSessionID)
	}
	return result
}

type managerAcceptanceStub struct {
	mu       sync.Mutex
	requests []store.AcceptNetworkMessageRequest
	handle   func(store.AcceptNetworkMessageRequest, int) (store.AcceptNetworkMessageResult, error)
}

func (s *managerAcceptanceStub) AcceptNetworkMessage(
	_ context.Context,
	req store.AcceptNetworkMessageRequest,
) (store.AcceptNetworkMessageResult, error) {
	s.mu.Lock()
	s.requests = append(s.requests, req)
	call := len(s.requests)
	handle := s.handle
	s.mu.Unlock()
	if handle == nil {
		return store.AcceptNetworkMessageResult{}, nil
	}
	return handle(req, call)
}

func (*managerAcceptanceStub) SettleNetworkWake(
	context.Context,
	store.WakeReservation,
	store.NetworkWakeOutcome,
) error {
	return nil
}

func (s *managerAcceptanceStub) request(t *testing.T, index int) store.AcceptNetworkMessageRequest {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if index < 0 || index >= len(s.requests) {
		t.Fatalf("acceptance request index = %d, calls = %d", index, len(s.requests))
	}
	return s.requests[index]
}

func (s *managerAcceptanceStub) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

type managerWakeNotifierStub struct {
	mu            sync.Mutex
	notifications []store.CommittedNetworkNotification
}

func (s *managerWakeNotifierStub) NotifyNetworkWake(notification store.CommittedNetworkNotification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifications = append(s.notifications, notification)
}

func (s *managerWakeNotifierStub) recipients() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]string, 0, len(s.notifications))
	for _, notification := range s.notifications {
		result = append(result, notification.RecipientSessionID)
	}
	return result
}

type managerAuditWriterStub struct{}

func (managerAuditWriterStub) RecordSent(context.Context, string, Envelope) error { return nil }
func (managerAuditWriterStub) RecordReceived(context.Context, string, Envelope) error {
	return nil
}
func (managerAuditWriterStub) RecordRejected(context.Context, string, Envelope, string) error {
	return nil
}
func (managerAuditWriterStub) RecordDelivered(context.Context, string, Envelope) error { return nil }
