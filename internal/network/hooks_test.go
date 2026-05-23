package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	hookspkg "github.com/compozy/agh/internal/hooks"
	"github.com/compozy/agh/internal/store"
)

func TestManagerDispatchesNetworkHooksAfterConversationCommit(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	conversations := &recordingConversationStore{
		result: store.NetworkConversationWriteResult{
			MessageID:          "msg-hook",
			ConversationOpened: true,
			WorkOpened:         true,
			WorkTransitioned:   true,
			WorkState:          store.NetworkWorkStateCompleted,
			LastActivityAt:     fixedNow,
		},
	}
	dispatcher := &recordingNetworkHookDispatcher{
		commitObserved: func() bool {
			return conversations.entry(0).MessageID == "msg-hook"
		},
	}
	manager := &Manager{
		logger:        discardManagerLogger(),
		now:           func() time.Time { return fixedNow },
		conversations: conversations,
		hooks:         dispatcher,
		stats:         newRuntimeStats(),
	}

	result, wrote, err := manager.writeConversationMessage(
		context.Background(),
		"sess-coder",
		AuditDirectionSent,
		networkHookTestEnvelope(t),
	)
	if err != nil {
		t.Fatalf("writeConversationMessage() error = %v", err)
	}
	if !wrote || result.Duplicate {
		t.Fatalf(
			"writeConversationMessage() wrote=%v duplicate=%v, want committed non-duplicate",
			wrote,
			result.Duplicate,
		)
	}

	gotEvents := dispatcher.events()
	wantEvents := []hookspkg.HookEvent{
		hookspkg.HookNetworkThreadOpened,
		hookspkg.HookNetworkMessagePersisted,
		hookspkg.HookNetworkWorkOpened,
		hookspkg.HookNetworkWorkTransitioned,
		hookspkg.HookNetworkWorkClosed,
	}
	if !reflect.DeepEqual(gotEvents, wantEvents) {
		t.Fatalf("network hook events = %#v, want %#v", gotEvents, wantEvents)
	}
	if dispatcher.commitFailures() != 0 {
		t.Fatalf("network hooks dispatched before commit %d times", dispatcher.commitFailures())
	}
	for _, call := range dispatcher.callsSnapshot() {
		if call.payload.Event != call.event ||
			call.payload.MessageID != "msg-hook" ||
			call.payload.WorkID != "work_hooks_01" ||
			call.payload.TraceID != "trace-hooks-01" ||
			call.payload.CausationID != "cause-hooks-01" {
			t.Fatalf("hook payload missing stable dedupe fields: %#v", call.payload)
		}
		encoded, marshalErr := json.Marshal(call.payload)
		if marshalErr != nil {
			t.Fatalf("json.Marshal(network payload) error = %v", marshalErr)
		}
		if strings.Contains(string(encoded), "body") || strings.Contains(string(encoded), "text") {
			t.Fatalf("network hook payload leaked body/text fields: %s", encoded)
		}
	}

	stats := manager.stats.snapshot()
	if stats.OpenThreads != 1 || stats.ConversationMessages != 1 || stats.WorkTransitions != 1 {
		t.Fatalf("stats = %#v, want thread/message/transition counters", stats)
	}
	if stats.OpenWorkItems != 0 {
		t.Fatalf("OpenWorkItems = %d, want 0 after terminal transition", stats.OpenWorkItems)
	}
}

func TestManagerNetworkHookFailureDoesNotRollbackCommittedMessage(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, 5, 5, 12, 5, 0, 0, time.UTC)
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelInfo}))
	conversations := &recordingConversationStore{
		result: store.NetworkConversationWriteResult{
			MessageID:          "msg-hook",
			ConversationOpened: true,
			WorkState:          store.NetworkWorkStateSubmitted,
			LastActivityAt:     fixedNow,
		},
	}
	dispatchErr := errors.New("observer offline")
	manager := &Manager{
		logger:        logger,
		now:           func() time.Time { return fixedNow },
		conversations: conversations,
		hooks:         &recordingNetworkHookDispatcher{err: dispatchErr},
		stats:         newRuntimeStats(),
	}

	_, wrote, err := manager.writeConversationMessage(
		context.Background(),
		"sess-coder",
		AuditDirectionSent,
		networkHookTestEnvelope(t),
	)
	if err != nil {
		t.Fatalf("writeConversationMessage() error = %v, want nil despite hook failure", err)
	}
	if !wrote {
		t.Fatal("writeConversationMessage() wrote = false, want true")
	}
	if got := conversations.entry(0).MessageID; got != "msg-hook" {
		t.Fatalf("committed message ID = %q, want msg-hook", got)
	}
	if got := manager.stats.snapshot().ConversationMessages; got != 1 {
		t.Fatalf("ConversationMessages = %d, want 1", got)
	}
	logOutput := logs.String()
	for _, want := range []string{
		"network.hook.dispatch_failed",
		"event=network.thread.opened",
		"message_id=msg-hook",
		"work_id=work_hooks_01",
		"trace_id=trace-hooks-01",
		"observer offline",
	} {
		if !strings.Contains(logOutput, want) {
			t.Fatalf("logs missing %q:\n%s", want, logOutput)
		}
	}
}

func TestManagerSkipsNetworkHooksForDuplicateConversationWrites(t *testing.T) {
	t.Parallel()

	conversations := &recordingConversationStore{
		result: store.NetworkConversationWriteResult{
			MessageID: "msg-hook",
			Duplicate: true,
		},
	}
	dispatcher := &recordingNetworkHookDispatcher{}
	manager := &Manager{
		logger:        discardManagerLogger(),
		now:           func() time.Time { return time.Date(2026, 5, 5, 12, 10, 0, 0, time.UTC) },
		conversations: conversations,
		hooks:         dispatcher,
		stats:         newRuntimeStats(),
	}

	_, wrote, err := manager.writeConversationMessage(
		context.Background(),
		"sess-coder",
		AuditDirectionSent,
		networkHookTestEnvelope(t),
	)
	if err != nil {
		t.Fatalf("writeConversationMessage() error = %v", err)
	}
	if !wrote {
		t.Fatal("writeConversationMessage() wrote = false, want true")
	}
	if got := dispatcher.events(); len(got) != 0 {
		t.Fatalf("duplicate network hook events = %#v, want none", got)
	}
	if got := manager.stats.snapshot().ConversationMessages; got != 0 {
		t.Fatalf("duplicate ConversationMessages = %d, want 0", got)
	}
}

func TestManagerDispatchesPeerLifecycleHooksFromJoinAndLeave(t *testing.T) {
	t.Parallel()

	t.Run("Should emit peer joined and left only from network lifecycle calls", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		fixedNow := time.Date(2026, 5, 21, 13, 30, 0, 0, time.UTC)
		dispatcher := &recordingNetworkHookDispatcher{}
		manager, err := NewManager(
			ctx,
			testManagerConfig(),
			newFakeDeliveryPrompter(),
			filepath.Join(t.TempDir(), "network.audit"),
			nil,
			WithManagerLogger(discardManagerLogger()),
			WithManagerClock(func() time.Time { return fixedNow }),
			WithManagerHookDispatcher(dispatcher),
		)
		if err != nil {
			t.Fatalf("NewManager() error = %v", err)
		}
		t.Cleanup(func() {
			if err := manager.Shutdown(context.Background()); err != nil {
				t.Fatalf("Shutdown() error = %v", err)
			}
		})

		join := testJoinRequest("sess-local", "reviewer.sess-local", "builders")
		if err := manager.JoinChannel(ctx, join); err != nil {
			t.Fatalf("JoinChannel() error = %v", err)
		}
		if err := manager.LeaveChannel(ctx, join.SessionID); err != nil {
			t.Fatalf("LeaveChannel() error = %v", err)
		}

		calls := dispatcher.lifecycleCalls()
		if got, want := len(calls), 2; got != want {
			t.Fatalf("peer lifecycle hook count = %d, want %d: %#v", got, want, calls)
		}
		if calls[0].event != hookspkg.HookNetworkPeerJoined || calls[1].event != hookspkg.HookNetworkPeerLeft {
			t.Fatalf("peer lifecycle events = %#v, want joined then left", dispatcher.events())
		}
		for _, call := range calls {
			if call.payload.PeerID != join.PeerID ||
				call.payload.PeerFrom != join.PeerID ||
				call.payload.WorkspaceID != join.WorkspaceID ||
				call.payload.Channel != join.Channel {
				t.Fatalf("peer lifecycle payload = %#v, want joined peer metadata", call.payload)
			}
		}
	})
}

func TestManagerListPeersSweepsExpiredRemoteAndDispatchesLeftOnce(t *testing.T) {
	t.Parallel()

	t.Run("Should emit one peer-left hook when a list snapshot sweeps an expired remote", func(t *testing.T) {
		t.Parallel()

		startedAt := time.Date(2026, 5, 21, 14, 30, 0, 0, time.UTC)
		current := startedAt
		registry, err := NewPeerRegistry(10*time.Second, WithPeerRegistryClock(func() time.Time { return current }))
		if err != nil {
			t.Fatalf("NewPeerRegistry() error = %v", err)
		}
		remote := mustPeerCard(t, "coder.sess-expiring")
		if _, stored, refreshErr := registry.RefreshRemote(
			testWorkspaceID,
			"builders",
			remote,
			startedAt,
		); refreshErr != nil {
			t.Fatalf("RefreshRemote() error = %v", refreshErr)
		} else if !stored {
			t.Fatal("RefreshRemote() stored = false, want true")
		}

		dispatcher := &recordingNetworkHookDispatcher{}
		manager := &Manager{
			logger: discardManagerLogger(),
			now: func() time.Time {
				return current
			},
			config: testManagerConfig(),
			peers:  registry,
			hooks:  dispatcher,
		}

		current = startedAt.Add(21 * time.Second)
		peers, listErr := manager.ListPeers(context.Background(), testWorkspaceID, "builders")
		if listErr != nil {
			t.Fatalf("ListPeers() error = %v", listErr)
		}
		if len(peers) != 0 {
			t.Fatalf("ListPeers() = %#v, want expired remote omitted", peers)
		}
		wantEvents := []hookspkg.HookEvent{hookspkg.HookNetworkPeerLeft}
		if got := dispatcher.events(); !reflect.DeepEqual(got, wantEvents) {
			t.Fatalf("events after expiry sweep = %#v, want %#v", got, wantEvents)
		}

		if _, listErr = manager.ListPeers(context.Background(), testWorkspaceID, "builders"); listErr != nil {
			t.Fatalf("ListPeers(second) error = %v", listErr)
		}
		if got := dispatcher.events(); !reflect.DeepEqual(got, wantEvents) {
			t.Fatalf("events after second sweep = %#v, want still %#v", got, wantEvents)
		}
	})
}

func TestNetworkMetricLabelsExcludeHighCardinalityIDs(t *testing.T) {
	t.Parallel()

	stats := newRuntimeStats()
	stats.recordConversationWrite(
		store.NetworkConversationMessage{
			MessageID:   "msg_high",
			SessionID:   "sess-coder",
			Channel:     "builders",
			Surface:     store.NetworkSurfaceDirect,
			DirectID:    "direct_99401d24bee62651d189e5a561785466",
			Direction:   AuditDirectionReceived,
			PeerFrom:    "coder.sess-abc",
			PeerTo:      "reviewer.sess-xyz",
			Kind:        string(KindSay),
			WorkID:      "work_high",
			TraceID:     "trace-high",
			CausationID: "cause-high",
			Body:        mustRawJSON(t, SayBody{Text: "safe"}),
			Timestamp:   time.Date(2026, 5, 5, 12, 15, 0, 0, time.UTC),
		},
		store.NetworkConversationWriteResult{
			MessageID:          "msg_high",
			ConversationOpened: true,
			WorkOpened:         true,
			WorkState:          store.NetworkWorkStateWorking,
		},
	)
	snapshot := stats.snapshot()
	if snapshot.OpenDirectRooms != 1 || snapshot.OpenWorkItems != 1 || snapshot.DirectResolves != 1 {
		t.Fatalf("snapshot counters = %#v, want direct room, work item, and direct resolve", snapshot)
	}
	for _, sample := range snapshot.Metrics {
		for _, forbidden := range []string{
			"thread_id",
			"direct_id",
			"work_id",
			"message_id",
			"trace_id",
			"causation_id",
		} {
			if _, ok := sample.Labels[forbidden]; ok {
				t.Fatalf("%s labels include high-cardinality %q: %#v", sample.Name, forbidden, sample.Labels)
			}
		}
	}
	names := metricSampleNames(snapshot.Metrics)
	for _, want := range []string{
		"network_conversation_messages_total",
		"network_direct_rooms_open_total",
		"network_work_open_total",
		"network_open_work_items",
		"network_direct_resolve_total",
	} {
		if !slices.Contains(names, want) {
			t.Fatalf("metric names = %#v, want %q", names, want)
		}
	}
}

func TestDeliveryQueueDepthMetricUsesLowCardinalityLabels(t *testing.T) {
	t.Parallel()

	queue := newInboundQueue(4)
	queue.enqueue(networkHookTestEnvelope(t), time.Date(2026, 5, 5, 12, 20, 0, 0, time.UTC), true)
	coordinator := &deliveryCoordinator{
		queues: map[string]*inboundQueue{"sess-coder": queue},
	}
	samples := coordinator.queueDepthMetrics()
	if len(samples) != 1 {
		t.Fatalf("queue depth samples = %#v, want one sample", samples)
	}
	sample := samples[0]
	if sample.Name != "network_delivery_queue_depth" || sample.Value != 1 {
		t.Fatalf("queue depth sample = %#v, want value 1", sample)
	}
	if got, want := sample.Labels["channel"], "builders"; got != want {
		t.Fatalf("queue depth channel label = %q, want %q", got, want)
	}
	if got, want := sample.Labels["surface"], "thread"; got != want {
		t.Fatalf("queue depth surface label = %q, want %q", got, want)
	}
	if _, ok := sample.Labels["message_id"]; ok {
		t.Fatalf("queue depth labels include message_id: %#v", sample.Labels)
	}
}

type networkHookCall struct {
	event   hookspkg.HookEvent
	payload hookspkg.NetworkPayload
}

type recordingNetworkHookDispatcher struct {
	mu             sync.Mutex
	calls          []networkHookCall
	err            error
	commitObserved func() bool
	commitMisses   int
}

func (d *recordingNetworkHookDispatcher) DispatchNetworkPeerJoined(
	_ context.Context,
	payload hookspkg.NetworkPeerJoinedPayload,
) (hookspkg.NetworkPeerJoinedPayload, error) {
	return payload, d.record(hookspkg.HookNetworkPeerJoined, payload)
}

func (d *recordingNetworkHookDispatcher) DispatchNetworkPeerLeft(
	_ context.Context,
	payload hookspkg.NetworkPeerLeftPayload,
) (hookspkg.NetworkPeerLeftPayload, error) {
	return payload, d.record(hookspkg.HookNetworkPeerLeft, payload)
}

func (d *recordingNetworkHookDispatcher) DispatchNetworkThreadOpened(
	_ context.Context,
	payload hookspkg.NetworkThreadOpenedPayload,
) (hookspkg.NetworkThreadOpenedPayload, error) {
	return payload, d.record(hookspkg.HookNetworkThreadOpened, payload)
}

func (d *recordingNetworkHookDispatcher) DispatchNetworkDirectRoomOpened(
	_ context.Context,
	payload hookspkg.NetworkDirectRoomOpenedPayload,
) (hookspkg.NetworkDirectRoomOpenedPayload, error) {
	return payload, d.record(hookspkg.HookNetworkDirectRoomOpened, payload)
}

func (d *recordingNetworkHookDispatcher) DispatchNetworkMessagePersisted(
	_ context.Context,
	payload hookspkg.NetworkMessagePersistedPayload,
) (hookspkg.NetworkMessagePersistedPayload, error) {
	return payload, d.record(hookspkg.HookNetworkMessagePersisted, payload)
}

func (d *recordingNetworkHookDispatcher) DispatchNetworkWorkOpened(
	_ context.Context,
	payload hookspkg.NetworkWorkOpenedPayload,
) (hookspkg.NetworkWorkOpenedPayload, error) {
	return payload, d.record(hookspkg.HookNetworkWorkOpened, payload)
}

func (d *recordingNetworkHookDispatcher) DispatchNetworkWorkTransitioned(
	_ context.Context,
	payload hookspkg.NetworkWorkTransitionedPayload,
) (hookspkg.NetworkWorkTransitionedPayload, error) {
	return payload, d.record(hookspkg.HookNetworkWorkTransitioned, payload)
}

func (d *recordingNetworkHookDispatcher) DispatchNetworkWorkClosed(
	_ context.Context,
	payload hookspkg.NetworkWorkClosedPayload,
) (hookspkg.NetworkWorkClosedPayload, error) {
	return payload, d.record(hookspkg.HookNetworkWorkClosed, payload)
}

func (d *recordingNetworkHookDispatcher) record(event hookspkg.HookEvent, payload hookspkg.NetworkPayload) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.commitObserved != nil && !d.commitObserved() {
		d.commitMisses++
	}
	d.calls = append(d.calls, networkHookCall{event: event, payload: payload})
	return d.err
}

func (d *recordingNetworkHookDispatcher) events() []hookspkg.HookEvent {
	d.mu.Lock()
	defer d.mu.Unlock()
	events := make([]hookspkg.HookEvent, 0, len(d.calls))
	for _, call := range d.calls {
		events = append(events, call.event)
	}
	return events
}

func (d *recordingNetworkHookDispatcher) callsSnapshot() []networkHookCall {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]networkHookCall(nil), d.calls...)
}

func (d *recordingNetworkHookDispatcher) lifecycleCalls() []networkHookCall {
	d.mu.Lock()
	defer d.mu.Unlock()
	calls := make([]networkHookCall, 0, len(d.calls))
	for _, call := range d.calls {
		if call.event == hookspkg.HookNetworkPeerJoined || call.event == hookspkg.HookNetworkPeerLeft {
			calls = append(calls, call)
		}
	}
	return calls
}

func (d *recordingNetworkHookDispatcher) commitFailures() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.commitMisses
}

func networkHookTestEnvelope(t *testing.T) Envelope {
	t.Helper()
	return Envelope{
		Protocol:    ProtocolV0,
		WorkspaceID: testWorkspaceID,
		ID:          "msg-hook",
		Kind:        KindSay,
		Channel:     "builders",
		Surface:     new(SurfaceThread),
		ThreadID:    new("thread_hooks_01"),
		From:        "coder.sess-abc",
		WorkID:      new("work_hooks_01"),
		TraceID:     new("trace-hooks-01"),
		CausationID: new("cause-hooks-01"),
		TS:          time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC).Unix(),
		Body:        mustRawJSON(t, SayBody{Text: "safe hook payload"}),
	}
}

func metricSampleNames(samples []MetricSample) []string {
	seen := make(map[string]struct{}, len(samples))
	names := make([]string, 0, len(samples))
	for _, sample := range samples {
		if _, ok := seen[sample.Name]; ok {
			continue
		}
		seen[sample.Name] = struct{}{}
		names = append(names, sample.Name)
	}
	return names
}
