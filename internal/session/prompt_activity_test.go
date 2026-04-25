package session

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestPromptActivitySupervisorReportPersistsHeartbeatWithoutEvent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, WithNow(func() time.Time { return now }))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	supervisor := newPromptActivitySupervisor(
		testutil.Context(t),
		h.manager,
		session,
		newPromptTurnDispatchState(session, "turn-activity", TurnSourceUser, "hello"),
		testSupervisionConfig(),
	)
	supervisor.report(acp.PromptActivityReport{
		Timestamp: now.Add(5 * time.Second),
		Kind:      "agent_waiting",
		Detail:    "waiting for provider",
	})

	meta := readMeta(t, session.MetaPath())
	if meta.Liveness == nil || meta.Liveness.Activity == nil {
		t.Fatal("meta.Liveness.Activity = nil, want persisted activity")
	}
	if got, want := meta.Liveness.Activity.LastActivityKind, "agent_waiting"; got != want {
		t.Fatalf("activity kind = %q, want %q", got, want)
	}
	if got, want := meta.Liveness.Activity.LastActivityDetail, "waiting for provider"; got != want {
		t.Fatalf("activity detail = %q, want %q", got, want)
	}
	if meta.Liveness.Activity.LastActivityAt == nil ||
		!meta.Liveness.Activity.LastActivityAt.Equal(now.Add(5*time.Second)) {
		t.Fatalf("activity LastActivityAt = %#v, want heartbeat timestamp", meta.Liveness.Activity.LastActivityAt)
	}

	select {
	case event := <-supervisor.eventsChannel():
		t.Fatalf("unexpected runtime event from heartbeat-only report: %#v", event)
	default:
	}
}

func TestPromptActivitySupervisorProgressIsPersistedThroughPromptPump(t *testing.T) {
	h := newHarness(t,
		WithSessionSupervision(aghconfig.SessionSupervisionConfig{
			ActivityHeartbeatInterval: time.Millisecond,
			ProgressNotifyInterval:    time.Millisecond,
			InactivityWarningAfter:    0,
			InactivityTimeout:         0,
			TimeoutCancelGrace:        time.Second,
		}),
	)
	session := createSession(t, h)
	source := make(chan acp.AgentEvent)
	h.driver.promptHook = func(_ *fakeProcess, _ acp.PromptRequest) (<-chan acp.AgentEvent, error) {
		return source, nil
	}

	ctx, cancel := context.WithCancel(testutil.Context(t))
	events, err := h.manager.Prompt(ctx, session.ID, "long running")
	if err != nil {
		cancel()
		t.Fatalf("Prompt() error = %v", err)
	}

	progress := waitForPromptEvent(t, events, acp.EventTypeRuntimeProgress)
	if !strings.Contains(progress.Text, "Still working") {
		t.Fatalf("runtime progress text = %q, want Still working", progress.Text)
	}
	if progress.Runtime == nil {
		t.Fatal("runtime progress Runtime = nil, want activity payload")
	}

	cancel()
	drainPromptEvents(t, events)
	stored := readStoredEvents(t, session)
	if !storedEventsContainType(stored, acp.EventTypeRuntimeProgress) {
		t.Fatalf("stored events = %#v, want runtime_progress persisted", stored)
	}

	meta := readMeta(t, session.MetaPath())
	if meta.Liveness != nil && meta.Liveness.Activity != nil {
		t.Fatalf("meta activity after prompt cancellation = %#v, want cleared", meta.Liveness.Activity)
	}
}

func TestPromptActivitySupervisorWarningEmitsOnce(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, WithNow(func() time.Time { return now }))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	config := testSupervisionConfig()
	config.InactivityWarningAfter = time.Minute
	supervisor := newPromptActivitySupervisor(
		testutil.Context(t),
		h.manager,
		session,
		newPromptTurnDispatchState(session, "turn-warning", TurnSourceUser, "hello"),
		config,
	)
	supervisor.touch(now, runtimeActivityKindPromptStarted, "prompt started")
	supervisor.evaluate(now.Add(2 * time.Minute))

	warning := readRuntimeEvent(t, supervisor.eventsChannel())
	if got, want := warning.Type, acp.EventTypeRuntimeWarning; got != want {
		t.Fatalf("warning event type = %q, want %q", got, want)
	}
	if warning.Runtime == nil || warning.Runtime.IdleSeconds < 60 {
		t.Fatalf("warning runtime = %#v, want idle activity payload", warning.Runtime)
	}

	supervisor.evaluate(now.Add(3 * time.Minute))
	select {
	case event := <-supervisor.eventsChannel():
		t.Fatalf("unexpected second warning event: %#v", event)
	default:
	}
}

func TestPromptActivityRuntimeEventDoesNotClearStallState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, WithNow(func() time.Time { return now }))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	supervisor := newPromptActivitySupervisor(
		testutil.Context(t),
		h.manager,
		session,
		newPromptTurnDispatchState(session, "turn-stalled", TurnSourceUser, "hello"),
		testSupervisionConfig(),
	)
	supervisor.touch(now, runtimeActivityKindPromptStarted, "prompt started")
	session.markRuntimeStalled(store.SessionStallReasonActivityTimeout, now.Add(time.Second))

	supervisor.emitRuntimeEvent(acp.EventTypeRuntimeWarning, "Runtime activity timed out.", now.Add(2*time.Second))

	meta := readMeta(t, session.MetaPath())
	if meta.Liveness == nil {
		t.Fatal("meta.Liveness = nil, want preserved stalled liveness")
	}
	if got, want := meta.Liveness.StallState, store.SessionStallStateDetected; got != want {
		t.Fatalf("meta.Liveness.StallState = %q, want %q", got, want)
	}
	if got, want := meta.Liveness.StallReason, store.SessionStallReasonActivityTimeout; got != want {
		t.Fatalf("meta.Liveness.StallReason = %q, want %q", got, want)
	}
	if meta.Liveness.Activity == nil {
		t.Fatal("meta.Liveness.Activity = nil, want runtime activity preserved")
	}
}

func TestPromptActivitySupervisorTimeoutCancelsThenStopsSession(t *testing.T) {
	now := time.Date(2026, 4, 24, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, WithNow(func() time.Time { return now }))
	session := createSession(t, h)
	session.setCurrentTurnSource(TurnSourceUser)
	session.setCurrentPromptMeta(acp.PromptMeta{TurnSource: acp.PromptTurnSourceUser})

	config := testSupervisionConfig()
	config.InactivityTimeout = time.Second
	config.TimeoutCancelGrace = 200 * time.Millisecond
	supervisor := newPromptActivitySupervisor(
		testutil.Context(t),
		h.manager,
		session,
		newPromptTurnDispatchState(session, "turn-timeout", TurnSourceUser, "hello"),
		config,
	)
	supervisor.touch(now, runtimeActivityKindPromptStarted, "prompt started")
	supervisor.evaluate(now.Add(2 * time.Second))

	if got := h.driver.cancelCalls; got != 1 {
		t.Fatalf("driver cancel calls = %d, want 1", got)
	}
	if got := h.driver.stopCalls; got != 1 {
		t.Fatalf("driver stop calls = %d, want 1", got)
	}
	meta := readMeta(t, session.MetaPath())
	if meta.StopReason == nil || *meta.StopReason != store.StopTimeout {
		t.Fatalf("meta.StopReason = %#v, want %q", meta.StopReason, store.StopTimeout)
	}
	if meta.Liveness == nil || meta.Liveness.Activity != nil {
		t.Fatalf("meta.Liveness = %#v, want cleared activity after forced stop", meta.Liveness)
	}
}

func TestPromptActivitySupervisorTimeoutStopDeadline(t *testing.T) {
	t.Parallel()

	defaultGrace := aghconfig.DefaultSessionSupervisionConfig().TimeoutCancelGrace
	testCases := []struct {
		name       string
		supervisor *promptActivitySupervisor
		want       time.Duration
	}{
		{
			name:       "Should use default timeout cancel grace for nil supervisor",
			supervisor: nil,
			want:       defaultGrace,
		},
		{
			name: "Should use default timeout cancel grace for zero configured grace",
			supervisor: &promptActivitySupervisor{
				config: aghconfig.SessionSupervisionConfig{},
			},
			want: defaultGrace,
		},
		{
			name: "Should use default timeout cancel grace for negative configured grace",
			supervisor: &promptActivitySupervisor{
				config: aghconfig.SessionSupervisionConfig{
					TimeoutCancelGrace: -time.Millisecond,
				},
			},
			want: defaultGrace,
		},
		{
			name: "Should use configured timeout cancel grace for forced stop deadline",
			supervisor: &promptActivitySupervisor{
				config: aghconfig.SessionSupervisionConfig{
					TimeoutCancelGrace: 42 * time.Millisecond,
				},
			},
			want: 42 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := tc.supervisor.timeoutStopDeadline(); got != tc.want {
				t.Fatalf("timeoutStopDeadline() = %s, want %s", got, tc.want)
			}
		})
	}
}

func testSupervisionConfig() aghconfig.SessionSupervisionConfig {
	return aghconfig.SessionSupervisionConfig{
		ActivityHeartbeatInterval: time.Hour,
		ProgressNotifyInterval:    0,
		InactivityWarningAfter:    0,
		InactivityTimeout:         0,
		TimeoutCancelGrace:        time.Second,
	}
}

func waitForPromptEvent(t *testing.T, events <-chan acp.AgentEvent, eventType string) acp.AgentEvent {
	t.Helper()

	deadline := time.After(time.Second)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatalf("prompt events closed before %s", eventType)
			}
			if event.Type == eventType {
				return event
			}
		case <-deadline:
			t.Fatalf("timed out waiting for prompt event %s", eventType)
		}
	}
}

func readRuntimeEvent(t *testing.T, events <-chan acp.AgentEvent) acp.AgentEvent {
	t.Helper()

	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for runtime event")
	}
	return acp.AgentEvent{}
}

func drainPromptEvents(t *testing.T, events <-chan acp.AgentEvent) {
	t.Helper()

	for {
		select {
		case _, ok := <-events:
			if !ok {
				return
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for prompt event channel to close")
		}
	}
}

func storedEventsContainType(events []store.SessionEvent, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
