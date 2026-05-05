package session

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/heartbeat"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/subprocess"
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

	supervisor.emitRuntimeEvent(acp.EventTypeRuntimeWarning, "Runtime activity timed out.", now.Add(2*time.Second), nil)

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

func TestPromptActivitySupervisorMarksUnhealthyProcessAsStalled(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	healthStore := newFakeSessionHealthStore()
	h := newHarness(t,
		WithNow(func() time.Time { return now }),
		WithSessionHealthStore(healthStore),
	)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	supervisor := newPromptActivitySupervisor(
		testutil.Context(t),
		h.manager,
		session,
		newPromptTurnDispatchState(session, "turn-unhealthy", TurnSourceUser, "hello"),
		testSupervisionConfig(),
	)
	supervisor.touch(now, runtimeActivityKindPromptStarted, "prompt started")

	proc := h.driver.lastProcess()
	if proc == nil {
		t.Fatal("lastProcess() = nil, want process")
	}
	proc.setHealth(subprocess.HealthState{
		Healthy:             false,
		LastCheckedAt:       now.Add(5 * time.Second),
		ConsecutiveFailures: 2,
		LastError:           "health_check: context deadline exceeded",
	})

	supervisor.evaluate(now.Add(6 * time.Second))

	meta := readMeta(t, session.MetaPath())
	if meta.Liveness == nil {
		t.Fatal("meta.Liveness = nil, want stalled liveness")
	}
	if got, want := meta.Liveness.StallState, store.SessionStallStateDetected; got != want {
		t.Fatalf("meta.Liveness.StallState = %q, want %q", got, want)
	}
	if got, want := meta.Liveness.StallReason, store.SessionStallReasonProcessUnhealthy; got != want {
		t.Fatalf("meta.Liveness.StallReason = %q, want %q", got, want)
	}
	if meta.Liveness.Activity == nil || meta.Liveness.Activity.LastActivityAt == nil ||
		!meta.Liveness.Activity.LastActivityAt.Equal(now) {
		t.Fatalf("meta.Liveness.Activity = %#v, want original prompt activity timestamp", meta.Liveness.Activity)
	}

	warning := readRuntimeEvent(t, supervisor.eventsChannel())
	if got, want := warning.Type, acp.EventTypeRuntimeWarning; got != want {
		t.Fatalf("warning event type = %q, want %q", got, want)
	}
	if !strings.Contains(warning.Text, "Runtime health check failed") {
		t.Fatalf("warning.Text = %q, want health failure text", warning.Text)
	}

	health, err := h.manager.storedSessionHealth(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("storedSessionHealth() error = %v", err)
	}
	if health.State != heartbeat.SessionHealthStatePrompting || health.Health != heartbeat.SessionHealthDegraded {
		t.Fatalf("storedSessionHealth() = %#v, want prompting/degraded", health)
	}
}

func TestPromptActivitySupervisorIgnoresSyntheticHeartbeatWhenProcessIsUnhealthy(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	healthStore := newFakeSessionHealthStore()
	h := newHarness(t,
		WithNow(func() time.Time { return now }),
		WithSessionHealthStore(healthStore),
	)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	supervisor := newPromptActivitySupervisor(
		testutil.Context(t),
		h.manager,
		session,
		newPromptTurnDispatchState(session, "turn-heartbeat", TurnSourceUser, "hello"),
		testSupervisionConfig(),
	)
	supervisor.touch(now, runtimeActivityKindPromptStarted, "prompt started")

	proc := h.driver.lastProcess()
	if proc == nil {
		t.Fatal("lastProcess() = nil, want process")
	}
	proc.setHealth(subprocess.HealthState{
		Healthy:             false,
		LastCheckedAt:       now.Add(5 * time.Second),
		ConsecutiveFailures: 2,
		LastError:           "health_check: context deadline exceeded",
	})

	supervisor.report(acp.PromptActivityReport{
		Timestamp: now.Add(10 * time.Second),
		Kind:      runtimeActivityKindAgentWaiting,
		Detail:    "waiting for provider",
	})

	meta := readMeta(t, session.MetaPath())
	if meta.Liveness == nil {
		t.Fatal("meta.Liveness = nil, want stalled liveness")
	}
	if got, want := meta.Liveness.StallReason, store.SessionStallReasonProcessUnhealthy; got != want {
		t.Fatalf("meta.Liveness.StallReason = %q, want %q", got, want)
	}
	if meta.Liveness.Activity == nil {
		t.Fatal("meta.Liveness.Activity = nil, want original activity preserved")
	}
	if got, want := meta.Liveness.Activity.LastActivityKind, runtimeActivityKindPromptStarted; got != want {
		t.Fatalf("meta.Liveness.Activity.LastActivityKind = %q, want %q", got, want)
	}
	if meta.Liveness.Activity.LastActivityAt == nil || !meta.Liveness.Activity.LastActivityAt.Equal(now) {
		t.Fatalf(
			"meta.Liveness.Activity.LastActivityAt = %#v, want original prompt activity timestamp",
			meta.Liveness.Activity.LastActivityAt,
		)
	}

	warning := readRuntimeEvent(t, supervisor.eventsChannel())
	if got, want := warning.Type, acp.EventTypeRuntimeWarning; got != want {
		t.Fatalf("warning event type = %q, want %q", got, want)
	}
}

func TestPromptActivitySupervisorIgnoresUnknownProcessHealthSnapshot(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, WithNow(func() time.Time { return now }))
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	supervisor := newPromptActivitySupervisor(
		testutil.Context(t),
		h.manager,
		session,
		newPromptTurnDispatchState(session, "turn-unknown-health", TurnSourceUser, "hello"),
		testSupervisionConfig(),
	)
	supervisor.touch(now, runtimeActivityKindPromptStarted, "prompt started")

	proc := h.driver.lastProcess()
	if proc == nil {
		t.Fatal("lastProcess() = nil, want process")
	}
	proc.setHealth(subprocess.HealthState{})

	supervisor.evaluate(now.Add(5 * time.Second))

	meta := readMeta(t, session.MetaPath())
	if meta.Liveness == nil {
		t.Fatal("meta.Liveness = nil, want liveness")
	}
	if got := meta.Liveness.StallState; got != "" {
		t.Fatalf("meta.Liveness.StallState = %q, want empty", got)
	}
	if got := meta.Liveness.StallReason; got != "" {
		t.Fatalf("meta.Liveness.StallReason = %q, want empty", got)
	}

	select {
	case event := <-supervisor.eventsChannel():
		t.Fatalf("unexpected runtime event for unknown health snapshot: %#v", event)
	default:
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

func TestPromptActivitySupervisorPromptDeadlineStopsWithDeadlineDetail(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	h := newHarness(t, WithNow(func() time.Time { return now }))
	session := createSession(t, h)
	session.setCurrentTurnSource(TurnSourceUser)
	session.setCurrentPromptMeta(acp.PromptMeta{TurnSource: acp.PromptTurnSourceUser})

	config := testSupervisionConfig()
	config.TimeoutCancelGrace = 200 * time.Millisecond
	supervisor := newPromptActivitySupervisor(
		testutil.Context(t),
		h.manager,
		session,
		newPromptTurnDispatchState(session, "turn-deadline", TurnSourceUser, "hello"),
		config,
	)
	deadline := now.Add(time.Second)
	supervisor.deadlineAt = &deadline
	supervisor.touch(now, runtimeActivityKindPromptStarted, "prompt started")

	done := make(chan struct{})
	go func() {
		defer close(done)
		supervisor.handlePromptDeadline(now.Add(2 * time.Second))
	}()

	warning := readRuntimeEvent(t, supervisor.eventsChannel())
	if got, want := warning.Type, acp.EventTypeRuntimeWarning; got != want {
		t.Fatalf("warning event type = %q, want %q", got, want)
	}
	if warning.Runtime == nil {
		t.Fatal("warning.Runtime = nil, want runtime activity payload")
	}
	if warning.Runtime.DeadlineAt == nil || !warning.Runtime.DeadlineAt.Equal(deadline) {
		t.Fatalf("warning.Runtime.DeadlineAt = %#v, want %s", warning.Runtime.DeadlineAt, deadline)
	}
	if got, want := warning.Runtime.ElapsedMS, int64(2000); got != want {
		t.Fatalf("warning.Runtime.ElapsedMS = %d, want %d", got, want)
	}

	var raw map[string]any
	if err := json.Unmarshal(warning.Raw, &raw); err != nil {
		t.Fatalf("json.Unmarshal(warning.Raw) error = %v", err)
	}
	if got, want := raw["deadline_at"], deadline.Format(time.RFC3339Nano); got != want {
		t.Fatalf("warning.Raw deadline_at = %#v, want %#v", got, want)
	}
	if got, want := raw["elapsed_ms"], float64(2000); got != want {
		t.Fatalf("warning.Raw elapsed_ms = %#v, want %#v", got, want)
	}
	if got := h.driver.cancelCalls; got != 0 {
		t.Fatalf("driver cancel calls before runtime warning ack = %d, want 0", got)
	}
	if got := h.driver.stopCalls; got != 0 {
		t.Fatalf("driver stop calls before runtime warning ack = %d, want 0", got)
	}

	supervisor.ackPromptDeadlineWarning(warning)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handlePromptDeadline() did not return after runtime warning ack")
	}

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
	if got, want := meta.StopDetail, store.SessionStallReasonPromptDeadlineExceeded; got != want {
		t.Fatalf("meta.StopDetail = %q, want %q", got, want)
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
