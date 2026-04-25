package observe

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/session"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
	"github.com/pedronauck/agh/internal/version"
	aghworkspace "github.com/pedronauck/agh/internal/workspace"
)

func TestOnSessionCreatedRegistersSessionInGlobalDB(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-created", session.StateActive, h.workspace, h.now)

	h.observer.OnSessionCreated(testutil.Context(t), sess)

	sessions, err := h.observer.registry.ListSessions(testutil.Context(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].ID != "sess-created" || sessions[0].State != string(session.StateActive) {
		t.Fatalf("sessions[0] = %#v", sessions[0])
	}
}

func TestOnSessionStoppedUpdatesSessionStateToStopped(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-stopped", session.StateActive, h.workspace, h.now)

	h.observer.OnSessionCreated(testutil.Context(t), sess)
	sess.State = session.StateStopped
	sess.UpdatedAt = h.now.Add(2 * time.Minute)
	h.observer.OnSessionStopped(testutil.Context(t), sess)

	sessions, err := h.observer.registry.ListSessions(testutil.Context(t), store.SessionListQuery{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if got, want := len(sessions), 1; got != want {
		t.Fatalf("len(sessions) = %d, want %d", got, want)
	}
	if sessions[0].State != string(session.StateStopped) {
		t.Fatalf("sessions[0].State = %q, want %q", sessions[0].State, session.StateStopped)
	}
}

func TestOnAgentEventWritesEventSummaryToGlobalDB(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-summary", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)

	h.observer.OnAgentEvent(testutil.Context(t), sess.ID, acp.AgentEvent{
		Type:      "agent_message",
		TurnID:    "turn-1",
		Timestamp: h.now.Add(time.Minute),
		Text:      "assistant replied with the requested diff",
	})

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if events[0].Summary != "assistant replied with the requested diff" {
		t.Fatalf("events[0].Summary = %q", events[0].Summary)
	}
}

func TestSweepRetentionUsesConfiguredCutoff(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.retention = RetentionConfig{
		Enabled:       true,
		RetentionDays: 7,
		SweepInterval: time.Hour,
	}
	h.observer.setRetentionHealth(h.observer.initialRetentionHealth())

	sess := newSession("sess-retention", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)

	healthNow := h.now.Add(time.Hour)
	cutoff := healthNow.AddDate(0, 0, -7)
	h.recordEvent(t, sess.ID, "agent_message", cutoff.Add(-time.Nanosecond), "old event")
	h.recordEvent(t, sess.ID, "agent_message", cutoff, "cutoff event")
	h.recordEvent(t, sess.ID, "agent_message", cutoff.Add(time.Nanosecond), "fresh event")

	health, err := h.observer.SweepRetention(testutil.Context(t))
	if err != nil {
		t.Fatalf("SweepRetention() error = %v", err)
	}
	if !health.Enabled || health.RetentionDays != 7 || health.LastSweepStatus != retentionSweepStatusOK {
		t.Fatalf("SweepRetention() health = %#v, want enabled ok seven-day retention", health)
	}
	if health.LastCutoffAt == nil || !health.LastCutoffAt.Equal(cutoff) {
		t.Fatalf("SweepRetention().LastCutoffAt = %#v, want %s", health.LastCutoffAt, cutoff)
	}
	if health.DeletedEventSummaries != 1 {
		t.Fatalf("SweepRetention().DeletedEventSummaries = %d, want 1", health.DeletedEventSummaries)
	}

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("len(events) = %d, want %d: %#v", got, want, events)
	}
	if events[0].Summary != "cutoff event" || events[1].Summary != "fresh event" {
		t.Fatalf("events after retention = %#v, want cutoff and fresh events", events)
	}
}

func TestSweepRetentionDisabledKeepsHistory(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.retention = RetentionConfig{
		Enabled:       false,
		RetentionDays: 0,
		SweepInterval: time.Hour,
	}
	h.observer.setRetentionHealth(h.observer.initialRetentionHealth())

	sess := newSession("sess-keep-history", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)
	h.recordEvent(t, sess.ID, "agent_message", h.now.AddDate(-1, 0, 0), "old retained event")

	health, err := h.observer.SweepRetention(testutil.Context(t))
	if err != nil {
		t.Fatalf("SweepRetention(disabled) error = %v", err)
	}
	if health.Enabled || health.LastSweepStatus != retentionSweepStatusDisabled {
		t.Fatalf("SweepRetention(disabled) health = %#v, want disabled status", health)
	}

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("len(events) = %d, want %d: %#v", got, want, events)
	}
}

func TestOnAgentEventUpdatesTokenStatsWithNullableValues(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-usage", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)

	outputTokens := int64(4)
	totalTokens := int64(4)
	h.observer.OnAgentEvent(testutil.Context(t), sess.ID, acp.AgentEvent{
		Type:      "done",
		TurnID:    "turn-usage",
		Timestamp: h.now.Add(time.Minute),
		Usage: &acp.TokenUsage{
			TurnID:       "turn-usage",
			OutputTokens: &outputTokens,
			TotalTokens:  &totalTokens,
			Timestamp:    h.now.Add(time.Minute),
		},
	})

	stats, err := h.observer.QueryTokenStats(testutil.Context(t), store.TokenStatsQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryTokenStats() error = %v", err)
	}
	if got, want := len(stats), 1; got != want {
		t.Fatalf("len(stats) = %d, want %d", got, want)
	}
	if stats[0].InputTokens != nil {
		t.Fatalf("InputTokens = %#v, want nil", stats[0].InputTokens)
	}
	if stats[0].OutputTokens == nil || *stats[0].OutputTokens != 4 {
		t.Fatalf("OutputTokens = %#v, want 4", stats[0].OutputTokens)
	}
	if stats[0].TurnCount != 1 {
		t.Fatalf("TurnCount = %d, want 1", stats[0].TurnCount)
	}
}

func TestOnAgentEventWritesPermissionLog(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-permission", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)

	h.observer.OnAgentEvent(testutil.Context(t), sess.ID, acp.AgentEvent{
		Type:      "permission",
		TurnID:    "turn-perm",
		Timestamp: h.now.Add(time.Minute),
		Action:    "session/request_permission",
		Resource:  filepath.Join(h.workspace, "secret.txt"),
		Decision:  "allow",
	})

	entries, err := h.observer.QueryPermissionLog(testutil.Context(t), store.PermissionLogQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryPermissionLog() error = %v", err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("len(entries) = %d, want %d", got, want)
	}
	if entries[0].PolicyUsed != "approve-all" {
		t.Fatalf("entries[0].PolicyUsed = %q, want approve-all", entries[0].PolicyUsed)
	}
}

func TestOnAgentEventSkipsUnknownSession(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.OnAgentEvent(testutil.Context(t), "missing", acp.AgentEvent{
		Type:      "agent_message",
		TurnID:    "turn-1",
		Timestamp: h.now,
		Text:      "ignored",
	})

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("len(events) = %d, want 0", len(events))
	}
}

func TestNotifierLifecycleWritesThroughObserver(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-nil-ctx", session.StateActive, h.workspace, h.now)

	h.observer.OnSessionCreated(testutil.Context(t), sess)
	h.observer.OnAgentEvent(testutil.Context(t), sess.ID, acp.AgentEvent{
		Type:      "tool_result",
		TurnID:    "turn-nil-ctx",
		Timestamp: h.now.Add(time.Minute),
		Title:     "ls",
	})
	sess.State = session.StateStopped
	h.observer.OnSessionStopped(testutil.Context(t), sess)

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
}

func TestOnAgentEventGuardBranches(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.OnAgentEvent(testutil.Context(t), "", acp.AgentEvent{
		Type:      "agent_message",
		TurnID:    "turn-empty-session",
		Timestamp: h.now,
	})

	sess := newSession("sess-empty-type", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)
	h.observer.OnAgentEvent(testutil.Context(t), sess.ID, acp.AgentEvent{
		TurnID:    "turn-empty-type",
		Timestamp: h.now,
	})

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("len(events) = %d, want 0", len(events))
	}
}

func TestOnAgentEventPermissionWithoutResolvedPolicySkipsAudit(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.observer.resolvePermissionMode = func(context.Context, string, string) (string, error) {
		return "", nil
	}

	sess := newSession("sess-no-policy", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)
	h.observer.OnAgentEvent(testutil.Context(t), sess.ID, acp.AgentEvent{
		Type:      "permission",
		TurnID:    "turn-no-policy",
		Timestamp: h.now.Add(time.Minute),
		Action:    "session/request_permission",
		Resource:  filepath.Join(h.workspace, "secret.txt"),
		Decision:  "deny",
	})

	entries, err := h.observer.QueryPermissionLog(testutil.Context(t), store.PermissionLogQuery{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("QueryPermissionLog() error = %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("len(entries) = %d, want 0", len(entries))
	}
}

func TestQueryEventsFilterBySessionID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sessA := newSession("sess-a", session.StateActive, h.workspace, h.now)
	sessB := newSession("sess-b", session.StateActive, h.workspace, h.now.Add(time.Minute))
	h.observer.OnSessionCreated(testutil.Context(t), sessA)
	h.observer.OnSessionCreated(testutil.Context(t), sessB)

	h.recordEvent(t, sessA.ID, "agent_message", h.now.Add(time.Minute), "a-1")
	h.recordEvent(t, sessB.ID, "agent_message", h.now.Add(2*time.Minute), "b-1")

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{SessionID: sessB.ID})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if events[0].SessionID != sessB.ID {
		t.Fatalf("events[0].SessionID = %q, want %q", events[0].SessionID, sessB.ID)
	}
}

func TestQueryEventsReturnsHarnessLifecycleSummaries(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-harness-observe", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)

	base := h.now.Add(3 * time.Minute)
	summaries := []store.EventSummary{
		{
			SessionID: sess.ID,
			Type:      "harness.context_resolved",
			AgentName: sess.AgentName,
			Summary:   "surface=startup sections=memory|skills|network",
			Timestamp: base,
		},
		{
			SessionID: sess.ID,
			Type:      "harness.section_selected",
			AgentName: sess.AgentName,
			Summary:   "selected=memory|skills|network count=3",
			Timestamp: base.Add(time.Second),
		},
		{
			SessionID: sess.ID,
			Type:      "harness.augmenter_applied",
			AgentName: sess.AgentName,
			Summary:   "augmenter=durable_memory outcome=blank",
			Timestamp: base.Add(2 * time.Second),
		},
	}
	for _, summary := range summaries {
		if err := h.observer.registry.WriteEventSummary(testutil.Context(t), summary); err != nil {
			t.Fatalf("WriteEventSummary(%q) error = %v", summary.Type, err)
		}
	}

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{
		SessionID: sess.ID,
		Limit:     3,
	})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 3; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if got, want := events[0].Type, "harness.context_resolved"; got != want {
		t.Fatalf("events[0].Type = %q, want %q", got, want)
	}
	if got, want := events[2].Type, "harness.augmenter_applied"; got != want {
		t.Fatalf("events[2].Type = %q, want %q", got, want)
	}
	if got, want := events[2].Summary, summaries[2].Summary; got != want {
		t.Fatalf("events[2].Summary = %q, want %q", got, want)
	}
}

func TestQueryEventsFilterByEventType(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-type", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)

	h.recordEvent(t, sess.ID, "agent_message", h.now.Add(time.Minute), "msg")
	h.recordEvent(t, sess.ID, "tool_call", h.now.Add(2*time.Minute), "tool")

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{Type: "tool_call"})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if events[0].Type != "tool_call" {
		t.Fatalf("events[0].Type = %q, want tool_call", events[0].Type)
	}
}

func TestQueryEventsFilterByTimeRange(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-since", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)

	oldTs := h.now.Add(time.Minute)
	newTs := h.now.Add(3 * time.Minute)
	h.recordEvent(t, sess.ID, "agent_message", oldTs, "old")
	h.recordEvent(t, sess.ID, "agent_message", newTs, "new")

	events, err := h.observer.QueryEvents(
		testutil.Context(t),
		store.EventSummaryQuery{Since: h.now.Add(2 * time.Minute)},
	)
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if events[0].Summary != "new" {
		t.Fatalf("events[0].Summary = %q, want new", events[0].Summary)
	}
}

func TestQueryEventsLimitReturnsMostRecentRowsInAscendingOrder(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sess := newSession("sess-limit", session.StateActive, h.workspace, h.now)
	h.observer.OnSessionCreated(testutil.Context(t), sess)

	h.recordEvent(t, sess.ID, "agent_message", h.now.Add(time.Minute), "one")
	h.recordEvent(t, sess.ID, "agent_message", h.now.Add(2*time.Minute), "two")
	h.recordEvent(t, sess.ID, "agent_message", h.now.Add(3*time.Minute), "three")

	events, err := h.observer.QueryEvents(testutil.Context(t), store.EventSummaryQuery{Limit: 2})
	if err != nil {
		t.Fatalf("QueryEvents() error = %v", err)
	}
	if got, want := len(events), 2; got != want {
		t.Fatalf("len(events) = %d, want %d", got, want)
	}
	if events[0].Summary != "two" || events[1].Summary != "three" {
		t.Fatalf("events = %#v, want [two three]", events)
	}
}

func TestHealthReturnsCorrectActiveCounts(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	lastActivityAt := h.now.Add(30 * time.Minute)
	turnStartedAt := h.now.Add(20 * time.Minute)
	h.source.sessions = []*session.Info{
		{
			ID:        "sess-active-1",
			AgentName: "coder",
			State:     session.StateActive,
			Liveness: &store.SessionLivenessMeta{
				Activity: &store.SessionActivityMeta{
					TurnID:             "turn-active",
					TurnSource:         "user",
					TurnStartedAt:      &turnStartedAt,
					LastActivityAt:     &lastActivityAt,
					LastActivityKind:   "agent_waiting",
					LastActivityDetail: "waiting for provider",
					CurrentTool:        "delegate_task",
				},
			},
		},
		{ID: "sess-active-2", AgentName: "coder", State: session.StateStopping},
		{ID: "sess-stopped", State: session.StateStopped},
	}

	health, err := h.observer.Health(testutil.Context(t))
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if health.ActiveSessions != 2 || health.ActiveAgents != 1 {
		t.Fatalf("Health() = %#v, want 2 active sessions and 1 active agent", health)
	}
	if health.UptimeSeconds != 3600 {
		t.Fatalf("Health().UptimeSeconds = %d, want 3600", health.UptimeSeconds)
	}
	if health.Version != "1.2.3" {
		t.Fatalf("Health().Version = %q, want 1.2.3", health.Version)
	}
	if health.Persistence.Status != "ok" ||
		health.Persistence.GlobalDBSizeBytes != health.GlobalDBSizeBytes ||
		health.Persistence.SessionDBSizeBytes != health.SessionDBSizeBytes {
		t.Fatalf("Health().Persistence = %#v, want ok status with DB sizes", health.Persistence)
	}
	if health.Retention.LastSweepStatus != retentionSweepStatusDisabled {
		t.Fatalf("Health().Retention = %#v, want disabled default retention state", health.Retention)
	}
	if got, want := len(health.Activities), 1; got != want {
		t.Fatalf("len(Health().Activities) = %d, want %d: %#v", got, want, health.Activities)
	}
	activity := health.Activities[0]
	if activity.SessionID != "sess-active-1" || activity.Status != "active" || activity.CurrentTool != "delegate_task" {
		t.Fatalf("Health().Activities[0] = %#v, want active delegate_task activity", activity)
	}
	if got, want := activity.IdleSeconds, int64(30*60); got != want {
		t.Fatalf("Health().Activities[0].IdleSeconds = %d, want %d", got, want)
	}
}

func TestHealthIncludesLifecycleFailuresAndAgentProbes(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	ctx := testutil.Context(t)
	if err := h.registry.RegisterSession(ctx, store.SessionInfo{
		ID:          "sess-crash",
		Name:        "Crashed",
		AgentName:   "coder",
		Provider:    "claude",
		WorkspaceID: h.workspaceID,
		State:       string(session.StateStopped),
		StopReason:  store.StopAgentCrashed,
		Failure: &store.SessionFailure{
			Kind:            store.FailureProcess,
			Summary:         "provider crashed token=super-secret",
			CrashBundlePath: "/tmp/crash-token=super-secret.json",
		},
		CreatedAt: h.now,
		UpdatedAt: h.now.Add(2 * time.Minute),
	}); err != nil {
		t.Fatalf("RegisterSession(crash) error = %v", err)
	}
	if err := h.registry.RegisterSession(ctx, store.SessionInfo{
		ID:          "sess-protocol",
		Name:        "Protocol",
		AgentName:   "reviewer",
		Provider:    "codex",
		WorkspaceID: h.workspaceID,
		State:       string(session.StateStopped),
		StopReason:  store.StopError,
		Failure: &store.SessionFailure{
			Kind:    store.FailureProtocol,
			Summary: "bad ACP frame",
		},
		CreatedAt: h.now,
		UpdatedAt: h.now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("RegisterSession(protocol) error = %v", err)
	}
	h.observer.agentProbeSource = func(context.Context) ([]acp.ProbeTarget, error) {
		return []acp.ProbeTarget{{
			AgentName: "coder",
			Provider:  "claude",
			Command:   `missing-agent --api-key=super-secret "unterminated`,
		}}, nil
	}

	health, err := h.observer.Health(ctx)
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if got, want := health.Status, "degraded"; got != want {
		t.Fatalf("Health().Status = %q, want %q", got, want)
	}
	if health.Failures.Status != "degraded" ||
		health.Failures.Total != 2 ||
		health.Failures.ByKind[store.FailureProcess] != 1 ||
		health.Failures.ByKind[store.FailureProtocol] != 1 {
		t.Fatalf("Health().Failures = %#v, want classified lifecycle failures", health.Failures)
	}
	if got, want := len(health.Failures.Recent), 2; got != want {
		t.Fatalf("len(Health().Failures.Recent) = %d, want %d", got, want)
	}
	recent := health.Failures.Recent[0]
	if recent.SessionID != "sess-crash" || recent.FailureKind != store.FailureProcess {
		t.Fatalf("Health().Failures.Recent[0] = %#v, want most recent process failure", recent)
	}
	if strings.Contains(recent.Summary, "super-secret") ||
		strings.Contains(recent.CrashBundlePath, "super-secret") {
		t.Fatalf("Health().Failures.Recent[0] = %#v, want redacted diagnostics", recent)
	}
	if !strings.Contains(recent.Summary, "[REDACTED]") ||
		!strings.Contains(recent.CrashBundlePath, "[REDACTED]") {
		t.Fatalf("Health().Failures.Recent[0] = %#v, want redacted markers", recent)
	}
	if got, want := len(health.AgentProbes), 1; got != want {
		t.Fatalf("len(Health().AgentProbes) = %d, want %d", got, want)
	}
	probe := health.AgentProbes[0]
	if probe.Status != acp.ProbeStatusInvalid {
		t.Fatalf("Health().AgentProbes[0] = %#v, want invalid probe", probe)
	}
	if strings.Contains(probe.Command, "super-secret") || strings.Contains(probe.Error, "super-secret") {
		t.Fatalf("Health().AgentProbes[0] = %#v, want redacted probe command and error", probe)
	}
}

func TestHealthStatusDegradesForLifecycleFailures(t *testing.T) {
	t.Parallel()

	t.Run("ShouldDegradeTopLevelStatusWhenOnlyFailureHealthIsDegraded", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		ctx := testutil.Context(t)
		if err := h.registry.RegisterSession(ctx, store.SessionInfo{
			ID:          "sess-failure-only",
			Name:        "Failure Only",
			AgentName:   "coder",
			Provider:    "codex",
			WorkspaceID: h.workspaceID,
			State:       string(session.StateStopped),
			StopReason:  store.StopError,
			Failure: &store.SessionFailure{
				Kind:    store.FailureProtocol,
				Summary: "bad frame",
			},
			CreatedAt: h.now,
			UpdatedAt: h.now,
		}); err != nil {
			t.Fatalf("RegisterSession(failure) error = %v", err)
		}

		health, err := h.observer.Health(ctx)
		if err != nil {
			t.Fatalf("Health() error = %v", err)
		}
		if got, want := health.Failures.Status, observeHealthStatusDegraded; got != want {
			t.Fatalf("Health().Failures.Status = %q, want %q", got, want)
		}
		if got, want := health.Status, observeHealthStatusDegraded; got != want {
			t.Fatalf("Health().Status = %q, want %q", got, want)
		}
	})
}

type harness struct {
	observer    *Observer
	registry    *globaldb.GlobalDB
	bridges     *observeBridgeSource
	home        aghconfig.HomePaths
	source      *stubSessionSource
	now         time.Time
	workspaceID string
	workspace   string
}

const observerWorkspaceID = "ws-observe-workspace"

type stubSessionSource struct {
	sessions []*session.Info
}

type observeBridgeSource struct {
	*bridgepkg.Service
	broker *bridgepkg.Broker
}

func (s *stubSessionSource) List() []*session.Info {
	return s.sessions
}

func (s *observeBridgeSource) DeliveryMetrics() map[string]bridgepkg.BridgeDeliveryMetrics {
	if s == nil || s.broker == nil {
		return nil
	}
	return s.broker.DeliveryMetrics()
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	home, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(home); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	registry, err := globaldb.OpenGlobalDB(testutil.Context(t), home.DatabaseFile)
	if err != nil {
		t.Fatalf("OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := registry.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	})

	now := time.Date(2026, 4, 3, 18, 0, 0, 0, time.UTC)
	source := &stubSessionSource{}
	bridges := &observeBridgeSource{
		Service: bridgepkg.NewRegistry(registry, bridgepkg.WithNow(func() time.Time { return now })),
		broker:  bridgepkg.NewBroker(nil, bridgepkg.WithDeliveryBrokerNow(func() time.Time { return now })),
	}
	t.Cleanup(bridges.broker.Close)
	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("MkdirAll(workspace) error = %v", err)
	}
	cfg := aghconfig.DefaultWithHome(home)
	workspaceResolver := &fakeObserveWorkspaceResolver{
		expectedRef: observerWorkspaceID,
		resolved: aghworkspace.ResolvedWorkspace{
			Workspace: aghworkspace.Workspace{
				ID:      observerWorkspaceID,
				RootDir: workspace,
				Name:    "observe-workspace",
			},
			Config: cfg,
			Agents: []aghconfig.AgentDef{{
				Name:     "coder",
				Provider: "claude",
				Prompt:   "You are a coding assistant.",
			}},
		},
	}
	if err := registry.InsertWorkspace(testutil.Context(t), aghworkspace.Workspace{
		ID:        observerWorkspaceID,
		RootDir:   workspace,
		Name:      "observe-workspace",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("InsertWorkspace() error = %v", err)
	}

	observer, err := New(testutil.Context(t),
		WithRegistry(registry),
		WithHomePaths(home),
		WithSessionSource(source),
		WithBridgeSource(bridges),
		WithWorkspaceResolver(workspaceResolver),
		WithPermissionModeResolver(func(_ context.Context, agentName, workspaceID string) (string, error) {
			if strings.TrimSpace(agentName) == "" || strings.TrimSpace(workspaceID) == "" {
				return "", context.Canceled
			}
			return "approve-all", nil
		}),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		WithNow(func() time.Time { return now.Add(time.Hour) }),
		WithStartTime(now),
		WithVersionSource(func() version.Info {
			return version.Info{Version: "1.2.3"}
		}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return &harness{
		observer:    observer,
		registry:    registry,
		bridges:     bridges,
		home:        home,
		source:      source,
		now:         now,
		workspaceID: observerWorkspaceID,
		workspace:   workspace,
	}
}

func (h *harness) recordEvent(t *testing.T, sessionID string, eventType string, timestamp time.Time, text string) {
	t.Helper()

	h.observer.OnAgentEvent(testutil.Context(t), sessionID, acp.AgentEvent{
		Type:      eventType,
		TurnID:    "turn-" + strings.ReplaceAll(text, " ", "-"),
		Timestamp: timestamp,
		Text:      text,
	})
}

func newSession(id string, state session.State, workspace string, now time.Time) *session.Session {
	return &session.Session{
		ID:           id,
		Name:         strings.ToUpper(id),
		AgentName:    "coder",
		Provider:     "claude",
		WorkspaceID:  observerWorkspaceID,
		Workspace:    workspace,
		State:        state,
		ACPSessionID: "acp-" + id,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
