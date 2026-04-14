package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	apicontract "github.com/pedronauck/agh/internal/api/contract"
	automationpkg "github.com/pedronauck/agh/internal/automation"
	bridgepkg "github.com/pedronauck/agh/internal/bridges"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/extension/protocol"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	"github.com/pedronauck/agh/internal/memory"
	observepkg "github.com/pedronauck/agh/internal/observe"
	"github.com/pedronauck/agh/internal/session"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/store/sessiondb"
	"github.com/pedronauck/agh/internal/subprocess"
	taskpkg "github.com/pedronauck/agh/internal/task"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestHostAPIHandlerSessionsListReturnsAuthorizedSessions(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-allowed", []string{"sessions/list"}, []string{"session.read"})

	sess := env.createSession(t)
	result, err := env.call(t, "ext-allowed", "sessions/list", map[string]string{"workspace": env.workspaceID})
	if err != nil {
		t.Fatalf("Handle(sessions/list) error = %v", err)
	}

	var sessionsList []hostAPISessionSummary
	decodeResult(t, result, &sessionsList)
	if len(sessionsList) != 1 {
		t.Fatalf("sessions/list len = %d, want 1", len(sessionsList))
	}
	if sessionsList[0].ID != sess.ID {
		t.Fatalf("sessions/list[0].ID = %q, want %q", sessionsList[0].ID, sess.ID)
	}
	if sessionsList[0].Agent != "coder" {
		t.Fatalf("sessions/list[0].Agent = %q, want coder", sessionsList[0].Agent)
	}
}

func TestHostAPIHandlerSessionsListReturnsCapabilityDeniedWithoutSessionRead(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-denied", nil, nil)

	_, err := env.call(t, "ext-denied", "sessions/list", nil)
	assertCapabilityDenied(t, err, "sessions/list")
}

func TestHostAPIHandlerSessionsCreateReturnsSessionID(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-create", []string{"sessions/create"}, []string{"session.write"})

	result, err := env.call(t, "ext-create", "sessions/create", map[string]string{
		"agent":     "coder",
		"workspace": env.workspaceID,
	})
	if err != nil {
		t.Fatalf("Handle(sessions/create) error = %v", err)
	}

	var created hostAPISessionCreateResult
	decodeResult(t, result, &created)
	if created.SessionID == "" {
		t.Fatal("sessions/create session_id = empty, want non-empty")
	}

	info, err := env.sessions.Status(testutil.Context(t), created.SessionID)
	if err != nil {
		t.Fatalf("sessions.Status(%q) error = %v", created.SessionID, err)
	}
	if info.State != session.StateActive {
		t.Fatalf("created session state = %q, want %q", info.State, session.StateActive)
	}
}

func TestHostAPIHandlerSessionsCreateReturnsCapabilityDeniedWithoutSessionWrite(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-denied", nil, nil)

	_, err := env.call(t, "ext-denied", "sessions/create", map[string]string{
		"agent":     "coder",
		"workspace": env.workspaceID,
	})
	assertCapabilityDenied(t, err, "sessions/create")
}

func TestHostAPIHandlerSessionsPromptReturnsTurnIDAndPersistsEvents(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-prompt", []string{"sessions/prompt"}, []string{"session.write"})

	sess := env.createSession(t)
	result, err := env.call(t, "ext-prompt", "sessions/prompt", map[string]string{
		"session_id": sess.ID,
		"message":    "hello from extension",
	})
	if err != nil {
		t.Fatalf("Handle(sessions/prompt) error = %v", err)
	}

	var prompt hostAPISessionPromptResult
	decodeResult(t, result, &prompt)
	if prompt.TurnID == "" {
		t.Fatal("sessions/prompt turn_id = empty, want non-empty")
	}

	events, err := env.sessions.Events(testutil.Context(t), sess.ID, store.EventQuery{TurnID: prompt.TurnID})
	if err != nil {
		t.Fatalf("sessions.Events(%q) error = %v", sess.ID, err)
	}
	if len(events) == 0 {
		t.Fatal("sessions events = empty, want prompt events")
	}
	if events[0].TurnID != prompt.TurnID {
		t.Fatalf("events[0].TurnID = %q, want %q", events[0].TurnID, prompt.TurnID)
	}
}

func TestHostAPIHandlerSessionsStopStopsSession(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-stop", []string{"sessions/stop"}, []string{"session.write"})

	sess := env.createSession(t)
	if _, err := env.call(t, "ext-stop", "sessions/stop", map[string]string{"session_id": sess.ID}); err != nil {
		t.Fatalf("Handle(sessions/stop) error = %v", err)
	}

	info, err := env.sessions.Status(testutil.Context(t), sess.ID)
	if err != nil {
		t.Fatalf("sessions.Status(%q) error = %v", sess.ID, err)
	}
	if info.State != session.StateStopped {
		t.Fatalf("stopped session state = %q, want %q", info.State, session.StateStopped)
	}
}

func TestHostAPIHandlerSessionsStatusReturnsAuthorizedState(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-status", []string{"sessions/status"}, []string{"session.read"})

	sess := env.createSession(t)
	result, err := env.call(t, "ext-status", "sessions/status", map[string]string{"session_id": sess.ID})
	if err != nil {
		t.Fatalf("Handle(sessions/status) error = %v", err)
	}

	var status hostAPISessionStatus
	decodeResult(t, result, &status)
	if status.SessionID != sess.ID {
		t.Fatalf("sessions/status session_id = %q, want %q", status.SessionID, sess.ID)
	}
	if status.State != session.StateActive {
		t.Fatalf("sessions/status state = %q, want %q", status.State, session.StateActive)
	}
}

func TestHostAPIHandlerSessionsEventsSupportsSinceFilter(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-events", []string{"sessions/events", "sessions/prompt"}, []string{"session.read", "session.write"})

	sess := env.createSession(t)
	if _, err := env.call(t, "ext-events", "sessions/events", map[string]any{
		"session_id": sess.ID,
		"limit":      10,
	}); err != nil {
		t.Fatalf("Handle(sessions/events baseline) error = %v", err)
	}

	since := env.currentTime().Add(-time.Second).Format(time.RFC3339Nano)
	if _, err := env.submitPrompt(t, "ext-events", sess.ID, "show me the timeline"); err != nil {
		t.Fatalf("submitPrompt() error = %v", err)
	}

	result, err := env.call(t, "ext-events", "sessions/events", map[string]any{
		"session_id": sess.ID,
		"since":      since,
		"limit":      10,
	})
	if err != nil {
		t.Fatalf("Handle(sessions/events) error = %v", err)
	}

	var events []hostAPISessionEvent
	decodeResult(t, result, &events)
	if len(events) == 0 {
		t.Fatal("sessions/events len = 0, want prompt-related events")
	}
}

func TestHostAPIHandlerSessionsMethodsRequireConfiguredManager(t *testing.T) {
	t.Parallel()

	checker := &CapabilityChecker{}
	checker.Register("ext-sessions", SourceUser, &Manifest{
		Actions: ActionsConfig{Requires: []string{"sessions/stop", "sessions/status", "sessions/events"}},
		Security: SecurityConfig{
			Capabilities: []string{"session.read", "session.write"},
		},
	})

	handler := NewHostAPIHandler(
		nil,
		nil,
		nil,
		nil,
		WithHostAPICapabilityChecker(checker),
		WithHostAPIRateLimit(1000, 1000),
	)

	tests := []struct {
		name   string
		method string
		params any
	}{
		{
			name:   "ShouldRejectStopWithoutManager",
			method: "sessions/stop",
			params: map[string]any{"session_id": "sess-1"},
		},
		{
			name:   "ShouldRejectStatusWithoutManager",
			method: "sessions/status",
			params: map[string]any{"session_id": "sess-1"},
		},
		{
			name:   "ShouldRejectEventsWithoutManager",
			method: "sessions/events",
			params: map[string]any{"session_id": "sess-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := marshalParams(tt.params)
			if err != nil {
				t.Fatalf("marshalParams() error = %v", err)
			}

			_, err = handler.Handle(testutil.Context(t), "ext-sessions", tt.method, params)
			assertErrorContains(t, err, "session manager is not configured")
		})
	}
}

func TestHostAPIHandlerMemoryStorePersistsContentWithTags(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-memory", []string{"memory/store"}, []string{"memory.write"})

	if _, err := env.call(t, "ext-memory", "memory/store", map[string]any{
		"key":     "deploy-script",
		"content": "The deploy script lives at /scripts/deploy.sh",
		"tags":    []string{"project-knowledge", "reference"},
	}); err != nil {
		t.Fatalf("Handle(memory/store) error = %v", err)
	}

	content, err := env.memory.Read(memory.ScopeGlobal, "deploy-script.md")
	if err != nil {
		t.Fatalf("memory.Read() error = %v", err)
	}
	if !strings.Contains(string(content), "/scripts/deploy.sh") {
		t.Fatalf("stored content = %q, want deploy path", string(content))
	}
	if !strings.Contains(string(content), "agh-tags: project-knowledge, reference") {
		t.Fatalf("stored content = %q, want persisted tag comment", string(content))
	}
}

func TestHostAPIHandlerMemoryRecallReturnsRankedMatches(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-memory", []string{"memory/store", "memory/recall"}, []string{"memory.write", "memory.read"})

	if _, err := env.call(t, "ext-memory", "memory/store", map[string]any{
		"key":     "deploy-script",
		"content": "The deploy script lives at /scripts/deploy.sh",
		"tags":    []string{"reference"},
	}); err != nil {
		t.Fatalf("Handle(memory/store) error = %v", err)
	}

	result, err := env.call(t, "ext-memory", "memory/recall", map[string]any{
		"query": "where is the deploy script",
		"limit": 5,
	})
	if err != nil {
		t.Fatalf("Handle(memory/recall) error = %v", err)
	}

	var entries []hostAPIMemoryRecallEntry
	decodeResult(t, result, &entries)
	if len(entries) == 0 {
		t.Fatal("memory/recall entries = 0, want at least one match")
	}
	if !strings.Contains(entries[0].Content, "deploy.sh") {
		t.Fatalf("memory/recall first content = %q, want deploy.sh", entries[0].Content)
	}
	if entries[0].Score <= 0 {
		t.Fatalf("memory/recall first score = %f, want > 0", entries[0].Score)
	}
}

func TestHostAPIHandlerMemoryRecallRequiresConfiguredStore(t *testing.T) {
	t.Parallel()

	checker := &CapabilityChecker{}
	checker.Register("ext-memory", SourceUser, &Manifest{
		Actions: ActionsConfig{Requires: []string{"memory/recall"}},
		Security: SecurityConfig{
			Capabilities: []string{"memory.read"},
		},
	})

	handler := NewHostAPIHandler(
		nil,
		nil,
		nil,
		nil,
		WithHostAPICapabilityChecker(checker),
		WithHostAPIRateLimit(1000, 1000),
	)

	params, err := marshalParams(map[string]any{"query": "needle"})
	if err != nil {
		t.Fatalf("marshalParams() error = %v", err)
	}

	_, err = handler.Handle(testutil.Context(t), "ext-memory", "memory/recall", params)
	assertErrorContains(t, err, "memory store is not configured")
}

func TestHostAPIHandlerMemoryForgetRemovesEntries(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-memory", []string{"memory/store", "memory/forget"}, []string{"memory.write"})

	if _, err := env.call(t, "ext-memory", "memory/store", map[string]any{
		"key":     "scratch",
		"content": "temporary note",
	}); err != nil {
		t.Fatalf("Handle(memory/store) error = %v", err)
	}
	if _, err := env.call(t, "ext-memory", "memory/forget", map[string]any{"key": "scratch"}); err != nil {
		t.Fatalf("Handle(memory/forget) error = %v", err)
	}

	if _, err := env.memory.Read(memory.ScopeGlobal, "scratch.md"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("memory.Read() error = %v, want os.ErrNotExist", err)
	}
}

func TestHostAPIHandlerObserveHealthReturnsSnapshot(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-observe", []string{"observe/health"}, []string{"observe.read"})

	env.createSession(t)
	result, err := env.call(t, "ext-observe", "observe/health", nil)
	if err != nil {
		t.Fatalf("Handle(observe/health) error = %v", err)
	}

	var health observepkg.Health
	decodeResult(t, result, &health)
	if health.ActiveSessions != 1 {
		t.Fatalf("observe/health active_sessions = %d, want 1", health.ActiveSessions)
	}
	if health.Status != "ok" {
		t.Fatalf("observe/health status = %q, want ok", health.Status)
	}
}

func TestHostAPIHandlerObserveEventsReturnsFilteredEventsWithSince(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-observe", []string{"sessions/prompt", "observe/events"}, []string{"session.write", "observe.read"})

	sess := env.createSession(t)
	since := env.currentTime().Add(-time.Second).Format(time.RFC3339Nano)
	if _, err := env.submitPrompt(t, "ext-observe", sess.ID, "collect observe event"); err != nil {
		t.Fatalf("submitPrompt() error = %v", err)
	}

	result, err := env.call(t, "ext-observe", "observe/events", map[string]any{
		"session_id": sess.ID,
		"since":      since,
		"limit":      20,
	})
	if err != nil {
		t.Fatalf("Handle(observe/events) error = %v", err)
	}

	var events []hostAPISessionEvent
	decodeResult(t, result, &events)
	if len(events) == 0 {
		t.Fatal("observe/events len = 0, want at least one event")
	}
}

func TestHostAPIHandlerSkillsListReturnsWorkspaceSkills(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-skills", []string{"skills/list"}, []string{"skills.read"})

	result, err := env.call(t, "ext-skills", "skills/list", map[string]any{"workspace": env.workspaceID})
	if err != nil {
		t.Fatalf("Handle(skills/list) error = %v", err)
	}

	var listed []hostAPISkillSummary
	decodeResult(t, result, &listed)
	if len(listed) == 0 {
		t.Fatal("skills/list len = 0, want workspace skill")
	}
	if listed[0].Name != "workspace-review" {
		t.Fatalf("skills/list[0].Name = %q, want workspace-review", listed[0].Name)
	}
}

func TestHostAPIHandlerBridgesMessagesIngestRejectsInvalidPayloads(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("telegram-adapter", []string{"bridges/messages/ingest"}, []string{"bridge.write"})

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-ingest-invalid",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	ctx := env.bridgeContext(t, instance)

	tests := []struct {
		name       string
		params     map[string]any
		wantText   string
		wantCode   int
		promptWant int
	}{
		{
			name: "MissingBridgeInstanceID",
			params: map[string]any{
				"scope":               instance.Scope,
				"workspace_id":        instance.WorkspaceID,
				"peer_id":             "peer-1",
				"platform_message_id": "msg-1",
				"received_at":         env.currentTime().Format(time.RFC3339Nano),
				"idempotency_key":     "idem-1",
				"content":             map[string]any{"text": "hello"},
			},
			wantText:   "bridge instance id",
			wantCode:   HostAPIInvalidParamsCode,
			promptWant: 0,
		},
		{
			name: "MissingPolicyRequiredPeer",
			params: map[string]any{
				"bridge_instance_id":  instance.ID,
				"scope":               instance.Scope,
				"workspace_id":        instance.WorkspaceID,
				"platform_message_id": "msg-2",
				"received_at":         env.currentTime().Format(time.RFC3339Nano),
				"idempotency_key":     "idem-2",
				"content":             map[string]any{"text": "hello"},
			},
			wantText:   "routing policy requires peer id",
			wantCode:   HostAPIInvalidParamsCode,
			promptWant: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := env.callWithContext(t, ctx, "telegram-adapter", "bridges/messages/ingest", tt.params)
			assertRPCErrorCode(t, err, tt.wantCode)
			assertErrorContains(t, err, tt.wantText)
			if got := env.driver.promptCount(); got != tt.promptWant {
				t.Fatalf("driver.promptCount() = %d, want %d", got, tt.promptWant)
			}
		})
	}
}

func TestHostAPIHandlerBridgesMessagesIngestRejectsDisabledOrUnknownInstances(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("telegram-adapter", []string{"bridges/messages/ingest"}, []string{"bridge.write"})

	disabled := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-ingest-disabled",
		Enabled:       false,
		Status:        bridgepkg.BridgeStatusDisabled,
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	disabledCtx := env.bridgeContext(t, disabled)

	_, err := env.callWithContext(t, disabledCtx, "telegram-adapter", "bridges/messages/ingest", map[string]any{
		"bridge_instance_id":  disabled.ID,
		"scope":               disabled.Scope,
		"workspace_id":        disabled.WorkspaceID,
		"peer_id":             "peer-1",
		"platform_message_id": "msg-disabled",
		"received_at":         env.currentTime().Format(time.RFC3339Nano),
		"idempotency_key":     "idem-disabled",
		"content":             map[string]any{"text": "hello"},
	})
	assertRPCErrorCode(t, err, HostAPIUnavailableCode)
	assertErrorContains(t, err, "disabled")
	if got := env.driver.promptCount(); got != 0 {
		t.Fatalf("driver.promptCount() = %d, want 0", got)
	}

	ready := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-ingest-ready",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	readyCtx := env.bridgeContext(t, ready)

	_, err = env.callWithContext(t, readyCtx, "telegram-adapter", "bridges/messages/ingest", map[string]any{
		"bridge_instance_id":  "brg-missing",
		"scope":               ready.Scope,
		"workspace_id":        ready.WorkspaceID,
		"peer_id":             "peer-1",
		"platform_message_id": "msg-missing",
		"received_at":         env.currentTime().Format(time.RFC3339Nano),
		"idempotency_key":     "idem-missing",
		"content":             map[string]any{"text": "hello"},
	})
	assertRPCErrorCode(t, err, HostAPINotFoundCode)
	if got := env.driver.promptCount(); got != 0 {
		t.Fatalf("driver.promptCount() after unknown instance = %d, want 0", got)
	}
}

func TestHostAPIHandlerBridgesMessagesIngestSuppressesDuplicateWebhookRetries(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("telegram-adapter", []string{"bridges/messages/ingest"}, []string{"bridge.write"})

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-ingest-dedup",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	ctx := env.bridgeContext(t, instance)
	params := map[string]any{
		"bridge_instance_id":  instance.ID,
		"scope":               instance.Scope,
		"workspace_id":        instance.WorkspaceID,
		"peer_id":             "peer-1",
		"platform_message_id": "msg-dedup",
		"received_at":         env.currentTime().Format(time.RFC3339Nano),
		"idempotency_key":     "idem-dedup",
		"content":             map[string]any{"text": "hello"},
	}

	first, err := env.callWithContext(t, ctx, "telegram-adapter", "bridges/messages/ingest", params)
	if err != nil {
		t.Fatalf("first ingest error = %v", err)
	}
	var firstResult hostAPIBridgesMessagesIngestResult
	decodeResult(t, first, &firstResult)

	firstRoute, err := env.bridges.ResolveRoute(testutil.Context(t), firstResult.RoutingKey)
	if err != nil {
		t.Fatalf("bridges.ResolveRoute(first) error = %v", err)
	}

	env.advanceTime(5 * time.Minute)

	second, err := env.callWithContext(t, ctx, "telegram-adapter", "bridges/messages/ingest", params)
	if err != nil {
		t.Fatalf("duplicate ingest error = %v", err)
	}
	var secondResult hostAPIBridgesMessagesIngestResult
	decodeResult(t, second, &secondResult)

	secondRoute, err := env.bridges.ResolveRoute(testutil.Context(t), secondResult.RoutingKey)
	if err != nil {
		t.Fatalf("bridges.ResolveRoute(second) error = %v", err)
	}

	routes, err := env.bridges.ListRoutes(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("bridges.ListRoutes() error = %v", err)
	}
	if got := len(routes); got != 1 {
		t.Fatalf("len(routes) = %d, want 1", got)
	}
	if got := env.driver.promptCount(); got != 1 {
		t.Fatalf("driver.promptCount() = %d, want 1", got)
	}
	if secondResult.SessionID != firstResult.SessionID {
		t.Fatalf("duplicate session_id = %q, want %q", secondResult.SessionID, firstResult.SessionID)
	}
	if !secondRoute.UpdatedAt.Equal(firstRoute.UpdatedAt) {
		t.Fatalf("duplicate retry updated route from %s to %s", firstRoute.UpdatedAt, secondRoute.UpdatedAt)
	}
}

func TestHostAPIHandlerBridgesInstancesReportStateRejectsInvalidUpdates(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("telegram-adapter", []string{"bridges/instances/report_state"}, []string{"bridge.write"})

	ready := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-report-state-ready",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	readyCtx := env.bridgeContext(t, ready)

	_, err := env.callWithContext(t, readyCtx, "telegram-adapter", "bridges/instances/report_state", map[string]any{
		"status": "disabled",
	})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "operator-controlled")

	_, err = env.callWithContext(t, readyCtx, "telegram-adapter", "bridges/instances/report_state", map[string]any{
		"status": "bogus",
	})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "unsupported bridge status")
}

func TestHostAPIHandlerBridgesInstancesGetRejectsMismatchedRuntimeOwnership(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("telegram-adapter", []string{"bridges/instances/get"}, []string{"bridge.read"})

	other := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-other-owner",
		ExtensionName: "discord-adapter",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	ctx := env.bridgeContext(t, other)

	_, err := env.callWithContext(t, ctx, "telegram-adapter", "bridges/instances/get", nil)
	assertRPCErrorCode(t, err, HostAPINotFoundCode)
}

func TestHostAPIHandlerMethodHandlersExposeBridgeRuntimeAwareInstanceLookup(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("telegram-adapter", []string{"bridges/instances/get"}, []string{"bridge.read"})

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-method-handler",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})

	handlers := env.handler.MethodHandlers()
	handler, ok := handlers["bridges/instances/get"]
	if !ok {
		t.Fatal("MethodHandlers()[bridges/instances/get] = missing, want handler")
	}

	ctx := withHostAPIExtensionName(env.bridgeContext(t, instance), "telegram-adapter")
	result, err := handler(ctx, nil)
	if err != nil {
		t.Fatalf("MethodHandlers()[bridges/instances/get]() error = %v", err)
	}

	var loaded hostAPIBridgeInstance
	decodeResult(t, result, &loaded)
	if loaded.ID != instance.ID {
		t.Fatalf("loaded.ID = %q, want %q", loaded.ID, instance.ID)
	}
}

func TestHostAPIHandlerBridgesMessagesIngestConcurrentSameRoutingKeyCreatesOneSessionAndRoute(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.useSessionsWithoutObserver(t)
	env.grant("telegram-adapter", []string{"bridges/messages/ingest"}, []string{"bridge.write"})

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-ingest-concurrent",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	ctx := env.bridgeContext(t, instance)

	type ingestResult struct {
		result hostAPIBridgesMessagesIngestResult
		err    error
	}

	results := make([]ingestResult, 2)
	var wg sync.WaitGroup
	for idx := range results {
		idx := idx
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := env.callWithContext(t, ctx, "telegram-adapter", "bridges/messages/ingest", map[string]any{
				"bridge_instance_id":  instance.ID,
				"scope":               instance.Scope,
				"workspace_id":        instance.WorkspaceID,
				"peer_id":             "peer-1",
				"platform_message_id": fmt.Sprintf("msg-%d", idx),
				"received_at":         env.currentTime().Format(time.RFC3339Nano),
				"idempotency_key":     fmt.Sprintf("idem-%d", idx),
				"content":             map[string]any{"text": fmt.Sprintf("hello-%d", idx)},
			})
			if err != nil {
				results[idx].err = err
				return
			}
			decodeResult(t, res, &results[idx].result)
		}()
	}
	wg.Wait()

	for idx, result := range results {
		if result.err != nil {
			t.Fatalf("ingest[%d] error = %v", idx, result.err)
		}
	}

	routes, err := env.bridges.ListRoutes(testutil.Context(t), instance.ID)
	if err != nil {
		t.Fatalf("bridges.ListRoutes() error = %v", err)
	}
	if got := len(routes); got != 1 {
		t.Fatalf("len(routes) = %d, want 1", got)
	}

	sessions, err := env.sessions.ListAll(testutil.Context(t))
	if err != nil {
		t.Fatalf("sessions.ListAll() error = %v", err)
	}
	if got := len(sessions); got != 1 {
		t.Fatalf("len(sessions) = %d, want 1", got)
	}
	if results[0].result.SessionID != results[1].result.SessionID {
		t.Fatalf("session IDs = %q and %q, want same session", results[0].result.SessionID, results[1].result.SessionID)
	}
	if got := env.driver.promptCount(); got != 2 {
		t.Fatalf("driver.promptCount() = %d, want 2", got)
	}
}

func TestHostAPIHandlerBridgesMessagesIngestRebindsStaleRouteToReplacementSession(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("telegram-adapter", []string{"bridges/messages/ingest"}, []string{"bridge.write"})

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-ingest-rebind",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	ctx := env.bridgeContext(t, instance)

	key, err := env.bridges.BuildRoutingKey(testutil.Context(t), bridgepkg.RoutingKey{
		BridgeInstanceID: instance.ID,
		Scope:            instance.Scope,
		WorkspaceID:      instance.WorkspaceID,
		PeerID:           "peer-1",
	})
	if err != nil {
		t.Fatalf("bridges.BuildRoutingKey() error = %v", err)
	}
	if _, err := env.bridges.UpsertRoute(testutil.Context(t), bridgepkg.BridgeRoute{
		Scope:            key.Scope,
		WorkspaceID:      key.WorkspaceID,
		BridgeInstanceID: key.BridgeInstanceID,
		PeerID:           key.PeerID,
		SessionID:        "missing-session",
		AgentName:        "coder",
	}); err != nil {
		t.Fatalf("bridges.UpsertRoute() error = %v", err)
	}

	result, err := env.callWithContext(t, ctx, "telegram-adapter", "bridges/messages/ingest", map[string]any{
		"bridge_instance_id":  instance.ID,
		"scope":               instance.Scope,
		"workspace_id":        instance.WorkspaceID,
		"peer_id":             "peer-1",
		"platform_message_id": "msg-rebind",
		"received_at":         env.currentTime().Format(time.RFC3339Nano),
		"idempotency_key":     "idem-rebind",
		"content":             map[string]any{"text": "hello"},
	})
	if err != nil {
		t.Fatalf("Handle(bridges/messages/ingest) error = %v", err)
	}

	var ingest hostAPIBridgesMessagesIngestResult
	decodeResult(t, result, &ingest)
	if ingest.SessionID == "missing-session" {
		t.Fatal("ingest session_id = missing-session, want replacement session")
	}

	route, err := env.bridges.ResolveRoute(testutil.Context(t), key)
	if err != nil {
		t.Fatalf("bridges.ResolveRoute() error = %v", err)
	}
	if route.SessionID != ingest.SessionID {
		t.Fatalf("route.SessionID = %q, want %q", route.SessionID, ingest.SessionID)
	}
	if got := env.driver.promptCount(); got != 1 {
		t.Fatalf("driver.promptCount() = %d, want 1", got)
	}
}

func TestHostAPIHandlerBridgesMessagesIngestExpiredDedupAllowsReingest(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("telegram-adapter", []string{"bridges/messages/ingest"}, []string{"bridge.write"})

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-ingest-expiry",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	ctx := env.bridgeContext(t, instance)
	params := map[string]any{
		"bridge_instance_id":  instance.ID,
		"scope":               instance.Scope,
		"workspace_id":        instance.WorkspaceID,
		"peer_id":             "peer-1",
		"platform_message_id": "msg-expiry",
		"received_at":         env.currentTime().Format(time.RFC3339Nano),
		"idempotency_key":     "idem-expiry",
		"content":             map[string]any{"text": "hello"},
	}

	if _, err := env.callWithContext(t, ctx, "telegram-adapter", "bridges/messages/ingest", params); err != nil {
		t.Fatalf("first ingest error = %v", err)
	}
	if got := env.driver.promptCount(); got != 1 {
		t.Fatalf("driver.promptCount() after first ingest = %d, want 1", got)
	}

	env.advanceTime(20 * time.Minute)
	if _, err := env.registry.GetBridgeIngestDedup(testutil.Context(t), "idem-expiry", env.currentTime()); !errors.Is(err, bridgepkg.ErrIngestDedupRecordNotFound) {
		t.Fatalf("GetBridgeIngestDedup(expired) error = %v, want ErrIngestDedupRecordNotFound", err)
	}

	if _, err := env.callWithContext(t, ctx, "telegram-adapter", "bridges/messages/ingest", params); err != nil {
		t.Fatalf("second ingest after expiry error = %v", err)
	}
	if got := env.driver.promptCount(); got != 2 {
		t.Fatalf("driver.promptCount() after reingest = %d, want 2", got)
	}

	if _, err := env.registry.GetBridgeIngestDedup(testutil.Context(t), "idem-expiry", env.currentTime()); err != nil {
		t.Fatalf("GetBridgeIngestDedup(refreshed) error = %v", err)
	}
}

func TestHostAPIHandlerBridgesMessagesIngestRegistersPromptDelivery(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("telegram-adapter", []string{"bridges/messages/ingest"}, []string{"bridge.write"})

	broker := &recordingPromptDeliveryBroker{}
	env.handler = NewHostAPIHandler(
		env.sessions,
		env.memory,
		env.observer,
		env.skills,
		WithHostAPICapabilityChecker(env.checker),
		WithHostAPIWorkspaceResolver(env.workspaces),
		WithHostAPIBridgeRegistry(env.bridges),
		WithHostAPIBridgeDedupStore(env.registry),
		WithHostAPIDeliveryBroker(broker),
		WithHostAPINow(func() time.Time { return env.currentTime() }),
		WithHostAPIBridgeIngressConfig(15*time.Minute, time.Minute),
		WithHostAPIRateLimit(1000, 1000),
	)

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-ingest-register",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	ctx := env.bridgeContext(t, instance)
	params := map[string]any{
		"bridge_instance_id":  instance.ID,
		"scope":               instance.Scope,
		"workspace_id":        instance.WorkspaceID,
		"peer_id":             "peer-1",
		"platform_message_id": "msg-register",
		"received_at":         env.currentTime().Format(time.RFC3339Nano),
		"idempotency_key":     "idem-register",
		"content":             map[string]any{"text": "hello"},
	}

	if _, err := env.callWithContext(t, ctx, "telegram-adapter", "bridges/messages/ingest", params); err != nil {
		t.Fatalf("Handle(bridges/messages/ingest) error = %v", err)
	}

	regs := broker.snapshotRegistrations()
	if len(regs) != 1 {
		t.Fatalf("len(prompt delivery registrations) = %d, want 1", len(regs))
	}
	reg := regs[0]
	if reg.SessionID == "" {
		t.Fatal("registration session id = empty, want non-empty")
	}
	if reg.TurnID == "" {
		t.Fatal("registration turn id = empty, want non-empty")
	}
	if got, want := reg.ExtensionName, instance.ExtensionName; got != want {
		t.Fatalf("registration extension = %q, want %q", got, want)
	}
	if got, want := reg.RoutingKey.BridgeInstanceID, instance.ID; got != want {
		t.Fatalf("registration routing key instance = %q, want %q", got, want)
	}
	if got, want := reg.RoutingKey.PeerID, "peer-1"; got != want {
		t.Fatalf("registration routing key peer = %q, want %q", got, want)
	}
	if got, want := reg.DeliveryTarget.Mode, bridgepkg.DeliveryModeReply; got != want {
		t.Fatalf("registration delivery mode = %q, want %q", got, want)
	}

	eventTypes := make([]string, 0, len(reg.SeedEvents))
	for _, event := range reg.SeedEvents {
		eventTypes = append(eventTypes, event.Type)
	}
	if !slices.Contains(eventTypes, acp.EventTypeUserMessage) {
		t.Fatalf("registration seed event types = %#v, want user_message from prompt boundary seed", eventTypes)
	}
}

func TestHostAPIHandlerRegisterPromptDeliveryReplaysStoredPromptEvents(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("delivery-replayer", []string{"sessions/prompt"}, []string{"session.write"})

	broker := &recordingPromptDeliveryBroker{}
	env.handler = NewHostAPIHandler(
		env.sessions,
		env.memory,
		env.observer,
		env.skills,
		WithHostAPICapabilityChecker(env.checker),
		WithHostAPIWorkspaceResolver(env.workspaces),
		WithHostAPIBridgeRegistry(env.bridges),
		WithHostAPIBridgeDedupStore(env.registry),
		WithHostAPIDeliveryBroker(broker),
		WithHostAPINow(func() time.Time { return env.currentTime() }),
		WithHostAPIBridgeIngressConfig(15*time.Minute, time.Minute),
		WithHostAPIRateLimit(1000, 1000),
	)

	sess := env.createSession(t)
	prompt, err := env.submitPrompt(t, "delivery-replayer", sess.ID, "replay me")
	if err != nil {
		t.Fatalf("submitPrompt() error = %v", err)
	}

	var promptEvents []store.SessionEvent
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		promptEvents, err = env.sessions.Events(testutil.Context(t), sess.ID, store.EventQuery{TurnID: prompt.TurnID})
		if err != nil {
			t.Fatalf("sessions.Events(%q) error = %v", sess.ID, err)
		}
		hasDone := false
		for _, storedEvent := range promptEvents {
			if strings.TrimSpace(storedEvent.Type) == acp.EventTypeDone {
				hasDone = true
				break
			}
		}
		if hasDone {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	instance := env.createBridgeInstance(t, bridgepkg.CreateInstanceRequest{
		ID:            "brg-register-replay",
		RoutingPolicy: bridgepkg.RoutingPolicy{IncludePeer: true},
	})
	routingKey, err := env.bridges.BuildRoutingKey(testutil.Context(t), bridgepkg.RoutingKey{
		Scope:            instance.Scope,
		WorkspaceID:      instance.WorkspaceID,
		BridgeInstanceID: instance.ID,
		PeerID:           "peer-1",
	})
	if err != nil {
		t.Fatalf("BuildRoutingKey() error = %v", err)
	}

	if err := env.handler.registerPromptDelivery(testutil.Context(t), *instance, routingKey, sess.ID, hostAPIPromptSubmission{
		TurnID: prompt.TurnID,
		SeedEvents: []bridgepkg.DeliveryProjectionEvent{{
			Type:      acp.EventTypeUserMessage,
			TurnID:    prompt.TurnID,
			Timestamp: env.currentTime(),
			Text:      "replay me",
		}},
	}); err != nil {
		t.Fatalf("registerPromptDelivery() error = %v", err)
	}

	projected := broker.snapshotProjectedEvents()
	projectedTypes := make([]string, 0, len(projected))
	for _, event := range projected {
		projectedTypes = append(projectedTypes, event.Type)
	}
	if !slices.Contains(projectedTypes, acp.EventTypeAgentMessage) {
		t.Fatalf("projected event types = %#v, want agent_message replay", projectedTypes)
	}
	if !slices.Contains(projectedTypes, acp.EventTypeDone) {
		t.Fatalf("projected event types = %#v, want done replay", projectedTypes)
	}
}

func TestBridgeHostAPIHelpersMapErrorsAndFormatInboundMetadata(t *testing.T) {
	t.Parallel()

	attachmentSummary := summarizeInboundAttachment(bridgepkg.MessageAttachment{
		ID:       "att-1",
		Name:     "report.pdf",
		MIMEType: "application/pdf",
		URL:      "https://example.com/report.pdf",
	})
	if !strings.Contains(attachmentSummary, "report.pdf") || !strings.Contains(attachmentSummary, "application/pdf") {
		t.Fatalf("summarizeInboundAttachment() = %q, want attachment name and mime type", attachmentSummary)
	}

	prompt := renderInboundMessagePrompt(bridgepkg.InboundMessageEnvelope{
		PlatformMessageID: "msg-1",
		ReceivedAt:        time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC),
		PeerID:            "peer-1",
		Sender:            bridgepkg.MessageSender{DisplayName: "Alice", Username: "alice"},
		Content:           bridgepkg.MessageContent{},
		Attachments: []bridgepkg.MessageAttachment{{
			Name:     "report.pdf",
			MIMEType: "application/pdf",
		}},
	})
	if !strings.Contains(prompt, "[No text body]") || !strings.Contains(prompt, "Attachments:") {
		t.Fatalf("renderInboundMessagePrompt() = %q, want attachment block and empty-body marker", prompt)
	}

	assertRPCErrorCode(t, mapBridgeLookupError("brg-1", bridgepkg.ErrBridgeInstanceNotFound), HostAPINotFoundCode)
	assertRPCErrorCode(t, mapBridgeRouteError("brg-1", bridgepkg.ErrBridgeInstanceUnavailable), HostAPIUnavailableCode)
	assertRPCErrorCode(t, mapBridgeStateUpdateError("brg-1", bridgepkg.ErrInvalidBridgeStateTransition), HostAPIInvalidParamsCode)

	env := newHostAPITestEnv(t)
	if err := env.handler.stopBridgeSession(testutil.Context(t), "missing-session"); err != nil {
		t.Fatalf("stopBridgeSession(missing) error = %v, want nil", err)
	}
}

func TestHostAPIHandlerUnknownMethodReturnsMethodNotFound(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	_, err := env.call(t, "ext-any", "sessions/missing", nil)
	assertRPCErrorCode(t, err, HostAPIMethodNotFoundCode)
}

func TestHostAPIHandlerRateLimitExceededReturnsRetryAfter(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-rate", []string{"observe/health"}, []string{"observe.read"})

	handler := NewHostAPIHandler(
		env.sessions,
		env.memory,
		env.observer,
		env.skills,
		WithHostAPICapabilityChecker(env.checker),
		WithHostAPIWorkspaceResolver(env.workspaces),
		WithHostAPINow(func() time.Time { return env.currentTime() }),
		WithHostAPIRateLimit(1, 1),
	)

	if _, err := handler.Handle(testutil.Context(t), "ext-rate", "observe/health", nil); err != nil {
		t.Fatalf("first Handle(observe/health) error = %v, want nil", err)
	}
	_, err := handler.Handle(testutil.Context(t), "ext-rate", "observe/health", nil)
	assertRPCErrorCode(t, err, HostAPIRateLimitedCode)

	data := decodeRPCData(t, err)
	if _, ok := data["retry_after_ms"]; !ok {
		t.Fatalf("rate limit data = %#v, want retry_after_ms", data)
	}
	if got := data["scope"]; got != "host_api.observe/health" {
		t.Fatalf("rate limit scope = %v, want host_api.observe/health", got)
	}
}

func TestHostAPIHandlerRateLimitUsesConfiguredClockRegardlessOfOptionOrder(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-rate", []string{"observe/health"}, []string{"observe.read"})

	handler := NewHostAPIHandler(
		env.sessions,
		env.memory,
		env.observer,
		env.skills,
		WithHostAPICapabilityChecker(env.checker),
		WithHostAPIWorkspaceResolver(env.workspaces),
		WithHostAPIRateLimit(1, 1),
		WithHostAPINow(func() time.Time { return env.currentTime() }),
	)

	if _, err := handler.Handle(testutil.Context(t), "ext-rate", "observe/health", nil); err != nil {
		t.Fatalf("first Handle(observe/health) error = %v, want nil", err)
	}

	env.advanceTime(2 * time.Second)
	if _, err := handler.Handle(testutil.Context(t), "ext-rate", "observe/health", nil); err != nil {
		t.Fatalf("second Handle(observe/health) error = %v, want nil after refill from injected clock", err)
	}
}

func TestHostAPIHandlerCapabilityErrorsCarryMethodAndRequiredCapabilities(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-denied", nil, nil)

	tests := []struct {
		method string
		params any
	}{
		{method: "sessions/list", params: nil},
		{method: "sessions/create", params: map[string]any{"agent": "coder", "workspace": env.workspaceID}},
		{method: "sessions/prompt", params: map[string]any{"session_id": "sess-1", "message": "hello"}},
		{method: "sessions/stop", params: map[string]any{"session_id": "sess-1"}},
		{method: "sessions/status", params: map[string]any{"session_id": "sess-1"}},
		{method: "sessions/events", params: map[string]any{"session_id": "sess-1"}},
		{method: "memory/recall", params: map[string]any{"query": "needle"}},
		{method: "memory/store", params: map[string]any{"key": "note", "content": "body"}},
		{method: "memory/forget", params: map[string]any{"key": "note"}},
		{method: "observe/health", params: nil},
		{method: "observe/events", params: map[string]any{"limit": 1}},
		{method: "skills/list", params: map[string]any{"workspace": env.workspaceID}},
		{method: "automation/jobs", params: map[string]any{"scope": "workspace", "workspace_id": env.workspaceID}},
		{method: "automation/jobs/create", params: map[string]any{
			"name":         "host-api-job",
			"scope":        "workspace",
			"workspace_id": env.workspaceID,
			"agent_name":   "coder",
			"prompt":       "do work",
			"schedule": map[string]any{
				"mode":     "every",
				"interval": "5m",
			},
		}},
		{method: "automation/triggers/fire", params: map[string]any{
			"event":        "ext.github.push",
			"scope":        "workspace",
			"workspace_id": env.workspaceID,
		}},
		{method: "bridges/messages/ingest", params: map[string]any{
			"bridge_instance_id":  "brg-1",
			"scope":               "workspace",
			"workspace_id":        env.workspaceID,
			"peer_id":             "peer-1",
			"platform_message_id": "msg-1",
			"received_at":         env.currentTime().Format(time.RFC3339Nano),
			"idempotency_key":     "idem-1",
		}},
		{method: "bridges/instances/get", params: nil},
		{method: "bridges/instances/report_state", params: map[string]any{"status": "ready"}},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			t.Parallel()

			_, err := env.call(t, "ext-denied", tt.method, tt.params)
			assertCapabilityDenied(t, err, tt.method)
		})
	}
}

func TestManagerWrapHostHandlerInjectsExtensionNameForHostAPIHandler(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-wrapped", []string{"observe/health"}, []string{"observe.read"})

	manager := NewManager(nil, WithCapabilityChecker(env.checker))
	wrapped := manager.wrapHostHandler("ext-wrapped", "observe/health", nil, env.handler.HandleMethod("observe/health"))

	result, err := wrapped(testutil.Context(t), nil)
	if err != nil {
		t.Fatalf("wrapHostHandler(observe/health) error = %v", err)
	}

	var health observepkg.Health
	decodeResult(t, result, &health)
	if health.Status != "ok" {
		t.Fatalf("wrapped observe/health status = %q, want ok", health.Status)
	}
}

func TestHostAPIHandlerAutomationTriggerFireRejectsNonExtensionEvent(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-automation", []string{"automation/triggers/fire"}, []string{"automation.write"})

	_, err := env.call(t, "ext-automation", "automation/triggers/fire", map[string]any{
		"event": "session.stopped",
		"scope": "workspace",
		"payload": map[string]any{
			"session_id": "sess-1",
		},
		"workspace_id": env.workspaceID,
	})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	data := decodeRPCData(t, err)
	if got := data["error"]; got != `trigger_fire.event must start with "ext."` {
		t.Fatalf("rpc data error = %#v, want ext prefix validation", got)
	}
}

func TestHostAPIHandlerAutomationJobCRUDAndRunQueries(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant(
		"ext-automation",
		[]string{
			"automation/jobs",
			"automation/jobs/get",
			"automation/jobs/create",
			"automation/jobs/update",
			"automation/jobs/delete",
			"automation/jobs/trigger",
			"automation/jobs/runs",
			"automation/runs",
		},
		[]string{"automation.read", "automation.write"},
	)

	createResult, err := env.call(t, "ext-automation", "automation/jobs/create", map[string]any{
		"name":         "host-api-job",
		"scope":        "workspace",
		"workspace_id": env.workspace.RootDir,
		"agent_name":   "coder",
		"prompt":       "Original host API job prompt",
		"schedule": map[string]any{
			"mode":     "every",
			"interval": "5m",
		},
	})
	if err != nil {
		t.Fatalf("Handle(automation/jobs/create) error = %v", err)
	}

	var created automationpkg.Job
	decodeResult(t, createResult, &created)
	if got, want := created.WorkspaceID, env.workspaceID; got != want {
		t.Fatalf("created workspace_id = %q, want %q", got, want)
	}

	listResult, err := env.call(t, "ext-automation", "automation/jobs", map[string]any{
		"scope":        "workspace",
		"workspace_id": env.workspace.RootDir,
		"enabled":      true,
	})
	if err != nil {
		t.Fatalf("Handle(automation/jobs) error = %v", err)
	}
	var listed []automationpkg.Job
	decodeResult(t, listResult, &listed)
	if got, want := len(listed), 1; got != want {
		t.Fatalf("len(automation/jobs) = %d, want %d", got, want)
	}

	getResult, err := env.call(t, "ext-automation", "automation/jobs/get", map[string]any{"id": created.ID})
	if err != nil {
		t.Fatalf("Handle(automation/jobs/get) error = %v", err)
	}
	var fetched automationpkg.Job
	decodeResult(t, getResult, &fetched)
	if got, want := fetched.ID, created.ID; got != want {
		t.Fatalf("automation/jobs/get id = %q, want %q", got, want)
	}

	updateResult, err := env.call(t, "ext-automation", "automation/jobs/update", map[string]any{
		"id":           created.ID,
		"workspace_id": env.workspace.RootDir,
		"prompt":       "Updated host API job prompt",
		"schedule": map[string]any{
			"mode":     "every",
			"interval": "15m",
		},
	})
	if err != nil {
		t.Fatalf("Handle(automation/jobs/update) error = %v", err)
	}
	var updated automationpkg.Job
	decodeResult(t, updateResult, &updated)
	if got, want := updated.Prompt, "Updated host API job prompt"; got != want {
		t.Fatalf("updated prompt = %q, want %q", got, want)
	}
	if updated.Schedule == nil || updated.Schedule.Interval != "15m" {
		t.Fatalf("updated schedule = %#v, want interval 15m", updated.Schedule)
	}

	triggerResult, err := env.call(t, "ext-automation", "automation/jobs/trigger", map[string]any{"id": created.ID})
	if err != nil {
		t.Fatalf("Handle(automation/jobs/trigger) error = %v", err)
	}
	var run automationpkg.Run
	decodeResult(t, triggerResult, &run)
	if got, want := run.JobID, created.ID; got != want {
		t.Fatalf("triggered run job_id = %q, want %q", got, want)
	}

	runsByJobResult, err := env.call(t, "ext-automation", "automation/jobs/runs", map[string]any{"id": created.ID})
	if err != nil {
		t.Fatalf("Handle(automation/jobs/runs) error = %v", err)
	}
	var runsByJob []automationpkg.Run
	decodeResult(t, runsByJobResult, &runsByJob)
	if got, want := len(runsByJob), 1; got != want {
		t.Fatalf("len(automation/jobs/runs) = %d, want %d", got, want)
	}

	allRunsResult, err := env.call(t, "ext-automation", "automation/runs", map[string]any{"job_id": created.ID})
	if err != nil {
		t.Fatalf("Handle(automation/runs) error = %v", err)
	}
	var allRuns []automationpkg.Run
	decodeResult(t, allRunsResult, &allRuns)
	if got, want := len(allRuns), 1; got != want {
		t.Fatalf("len(automation/runs) = %d, want %d", got, want)
	}

	if _, err := env.call(t, "ext-automation", "automation/jobs/delete", map[string]any{"id": created.ID}); err != nil {
		t.Fatalf("Handle(automation/jobs/delete) error = %v", err)
	}
}

func TestHostAPIHandlerAutomationTriggerCRUDAndConfigGuardrails(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant(
		"ext-automation",
		[]string{
			"automation/triggers",
			"automation/triggers/get",
			"automation/triggers/create",
			"automation/triggers/update",
			"automation/triggers/delete",
			"automation/triggers/runs",
			"automation/triggers/fire",
			"automation/jobs/delete",
			"automation/jobs/update",
		},
		[]string{"automation.read", "automation.write"},
	)

	createResult, err := env.call(t, "ext-automation", "automation/triggers/create", map[string]any{
		"name":         "host-api-trigger",
		"scope":        "workspace",
		"workspace_id": env.workspace.RootDir,
		"agent_name":   "coder",
		"event":        "ext.github.push",
		"prompt":       `Review {{ index .Data "repo" }}`,
		"filter": map[string]string{
			"data.repo": "acme/api",
		},
	})
	if err != nil {
		t.Fatalf("Handle(automation/triggers/create) error = %v", err)
	}

	var created automationpkg.Trigger
	decodeResult(t, createResult, &created)
	if got, want := created.WorkspaceID, env.workspaceID; got != want {
		t.Fatalf("created trigger workspace_id = %q, want %q", got, want)
	}

	listResult, err := env.call(t, "ext-automation", "automation/triggers", map[string]any{
		"scope":        "workspace",
		"workspace_id": env.workspace.RootDir,
		"event":        "ext.github.push",
		"enabled":      true,
	})
	if err != nil {
		t.Fatalf("Handle(automation/triggers) error = %v", err)
	}
	var listed []automationpkg.Trigger
	decodeResult(t, listResult, &listed)
	if got, want := len(listed), 1; got != want {
		t.Fatalf("len(automation/triggers) = %d, want %d", got, want)
	}

	getResult, err := env.call(t, "ext-automation", "automation/triggers/get", map[string]any{"id": created.ID})
	if err != nil {
		t.Fatalf("Handle(automation/triggers/get) error = %v", err)
	}
	var fetched automationpkg.Trigger
	decodeResult(t, getResult, &fetched)
	if got, want := fetched.ID, created.ID; got != want {
		t.Fatalf("automation/triggers/get id = %q, want %q", got, want)
	}

	updateResult, err := env.call(t, "ext-automation", "automation/triggers/update", map[string]any{
		"id":           created.ID,
		"workspace_id": env.workspace.RootDir,
		"prompt":       `Updated {{ index .Data "repo" }}`,
		"filter": map[string]string{
			"data.repo": "acme/api",
		},
	})
	if err != nil {
		t.Fatalf("Handle(automation/triggers/update) error = %v", err)
	}
	var updated automationpkg.Trigger
	decodeResult(t, updateResult, &updated)
	if got, want := updated.Prompt, `Updated {{ index .Data "repo" }}`; got != want {
		t.Fatalf("updated trigger prompt = %q, want %q", got, want)
	}

	fireResult, err := env.call(t, "ext-automation", "automation/triggers/fire", map[string]any{
		"event":        "ext.github.push",
		"scope":        "workspace",
		"workspace_id": env.workspaceID,
		"payload": map[string]any{
			"repo": "acme/api",
		},
	})
	if err != nil {
		t.Fatalf("Handle(automation/triggers/fire) error = %v", err)
	}
	var fire automationpkg.TriggerResult
	decodeResult(t, fireResult, &fire)
	if got, want := fire.Matched, 1; got != want {
		t.Fatalf("automation/triggers/fire matched = %d, want %d", got, want)
	}

	runsResult, err := env.call(t, "ext-automation", "automation/triggers/runs", map[string]any{"id": created.ID})
	if err != nil {
		t.Fatalf("Handle(automation/triggers/runs) error = %v", err)
	}
	var triggerRuns []automationpkg.Run
	decodeResult(t, runsResult, &triggerRuns)
	if got, want := len(triggerRuns), 1; got != want {
		t.Fatalf("len(automation/triggers/runs) = %d, want %d", got, want)
	}

	configJob, err := env.registry.CreateJob(testutil.Context(t), automationpkg.Job{
		ID:          "job-config-host-api",
		Scope:       automationpkg.AutomationScopeWorkspace,
		Name:        "config-host-api-job",
		AgentName:   "coder",
		WorkspaceID: env.workspaceID,
		Prompt:      "Config-backed prompt",
		Schedule: &automationpkg.ScheduleSpec{
			Mode:     automationpkg.ScheduleModeEvery,
			Interval: "1h",
		},
		Enabled:   true,
		Retry:     automationpkg.DefaultRetryConfig(),
		FireLimit: automationpkg.DefaultFireLimitConfig(),
		Source:    automationpkg.JobSourceConfig,
	})
	if err != nil {
		t.Fatalf("CreateJob(config) error = %v", err)
	}
	if _, err := env.call(t, "ext-automation", "automation/jobs/update", map[string]any{
		"id":      configJob.ID,
		"enabled": false,
	}); err != nil {
		t.Fatalf("Handle(automation/jobs/update enabled-only) error = %v", err)
	}
	_, err = env.call(t, "ext-automation", "automation/jobs/update", map[string]any{
		"id":     configJob.ID,
		"prompt": "should fail",
	})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	_, err = env.call(t, "ext-automation", "automation/jobs/delete", map[string]any{"id": configJob.ID})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)

	configTrigger, err := env.registry.CreateTrigger(testutil.Context(t), automationpkg.Trigger{
		ID:          "trigger-config-host-api",
		Scope:       automationpkg.AutomationScopeWorkspace,
		Name:        "config-host-api-trigger",
		AgentName:   "coder",
		WorkspaceID: env.workspaceID,
		Prompt:      `Config {{ index .Data "repo" }}`,
		Event:       "ext.github.push",
		Enabled:     true,
		Retry:       automationpkg.DefaultRetryConfig(),
		FireLimit:   automationpkg.DefaultFireLimitConfig(),
		Source:      automationpkg.JobSourceConfig,
	})
	if err != nil {
		t.Fatalf("CreateTrigger(config) error = %v", err)
	}
	if _, err := env.call(t, "ext-automation", "automation/triggers/update", map[string]any{
		"id":      configTrigger.ID,
		"enabled": false,
	}); err != nil {
		t.Fatalf("Handle(automation/triggers/update enabled-only) error = %v", err)
	}
	_, err = env.call(t, "ext-automation", "automation/triggers/update", map[string]any{
		"id":     configTrigger.ID,
		"prompt": "should fail",
	})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	_, err = env.call(t, "ext-automation", "automation/triggers/delete", map[string]any{"id": configTrigger.ID})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)

	if _, err := env.call(t, "ext-automation", "automation/triggers/delete", map[string]any{"id": created.ID}); err != nil {
		t.Fatalf("Handle(automation/triggers/delete) error = %v", err)
	}
}

func TestDescribeExtensionProjectsHealthAndState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)
	payload := DescribeExtension(&Extension{
		Manifest: &Manifest{
			Capabilities: CapabilitiesConfig{Provides: []string{"runtime"}},
			Actions:      ActionsConfig{Requires: []string{"automation/jobs"}},
			Subprocess:   SubprocessConfig{Command: "ext-runtime"},
		},
		Info: ExtensionInfo{
			Name:    "ext-runtime",
			Version: "1.0.0",
			Enabled: true,
			Source:  SourceUser,
			Capabilities: CapabilitiesConfig{
				Provides: []string{"runtime"},
			},
			Actions: ActionsConfig{Requires: []string{"automation/jobs"}},
		},
		Status: ExtensionStatus{
			Active:        true,
			Healthy:       true,
			Registered:    true,
			PID:           42,
			LastStartedAt: now.Add(-5 * time.Minute),
		},
	}, true, now)

	if got, want := payload.Type, "subprocess"; got != want {
		t.Fatalf("DescribeExtension() type = %q, want %q", got, want)
	}
	if got, want := payload.State, "active"; got != want {
		t.Fatalf("DescribeExtension() state = %q, want %q", got, want)
	}
	if got, want := payload.Health, "healthy"; got != want {
		t.Fatalf("DescribeExtension() health = %q, want %q", got, want)
	}
	if payload.UptimeSeconds <= 0 {
		t.Fatalf("DescribeExtension() uptime_seconds = %d, want positive", payload.UptimeSeconds)
	}

	disabled := DescribeExtension(&Extension{
		Info: ExtensionInfo{
			Name:    "ext-disabled",
			Version: "1.0.0",
			Enabled: false,
			Source:  SourceWorkspace,
		},
		Status: ExtensionStatus{Registered: true},
	}, false, now)
	if got, want := disabled.Type, "resource"; got != want {
		t.Fatalf("DescribeExtension(disabled) type = %q, want %q", got, want)
	}
	if got, want := disabled.State, "disabled"; got != want {
		t.Fatalf("DescribeExtension(disabled) state = %q, want %q", got, want)
	}
	if got, want := disabled.Health, "unknown"; got != want {
		t.Fatalf("DescribeExtension(disabled) health = %q, want %q", got, want)
	}

	registered := DescribeExtension(&Extension{
		Info: ExtensionInfo{
			Name:    "ext-registered",
			Version: "1.0.0",
			Enabled: true,
			Source:  SourceUser,
		},
		Status: ExtensionStatus{
			Registered: true,
		},
	}, true, now)
	if got, want := registered.State, "registered"; got != want {
		t.Fatalf("DescribeExtension(registered) state = %q, want %q", got, want)
	}
	if got, want := registered.Health, "healthy"; got != want {
		t.Fatalf("DescribeExtension(registered) health = %q, want %q", got, want)
	}

	unhealthy := DescribeExtension(&Extension{
		Manifest: &Manifest{
			Capabilities: CapabilitiesConfig{Provides: []string{"runtime"}},
			Subprocess:   SubprocessConfig{Command: "ext-runtime"},
		},
		Info: ExtensionInfo{
			Name:    "ext-unhealthy",
			Version: "1.0.0",
			Enabled: true,
			Source:  SourceUser,
			Capabilities: CapabilitiesConfig{
				Provides: []string{"runtime"},
			},
		},
		Status: ExtensionStatus{
			LastError: "boom",
		},
	}, true, now)
	if got, want := unhealthy.State, "error"; got != want {
		t.Fatalf("DescribeExtension(unhealthy) state = %q, want %q", got, want)
	}
	if got, want := unhealthy.Health, "unhealthy"; got != want {
		t.Fatalf("DescribeExtension(unhealthy) health = %q, want %q", got, want)
	}

	enabled := DescribeExtension(&Extension{
		Info: ExtensionInfo{
			Name:    "ext-enabled",
			Version: "1.0.0",
			Enabled: true,
			Source:  SourceUser,
		},
	}, false, now)
	if got, want := enabled.State, "enabled"; got != want {
		t.Fatalf("DescribeExtension(enabled daemon stopped) state = %q, want %q", got, want)
	}
}

func TestHostAPIHandlerAutomationGetterAndMethodHandlers(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	handler := NewHostAPIHandler(
		env.sessions,
		env.memory,
		env.observer,
		env.skills,
		WithHostAPICapabilityChecker(env.checker),
		WithHostAPIWorkspaceResolver(env.workspaces),
		WithHostAPIAutomationGetter(func() HostAPIAutomationManager {
			return env.automation
		}),
	)

	handlers := handler.MethodHandlers()
	if _, ok := handlers[string(protocol.HostAPIMethodAutomationJobs)]; !ok {
		t.Fatal("MethodHandlers() missing automation/jobs handler")
	}

	env.checker.Register("ext-automation", SourceUser, &Manifest{
		Actions: ActionsConfig{Requires: []string{"automation/jobs"}},
		Security: SecurityConfig{
			Capabilities: []string{"automation.read"},
		},
	})

	result, err := handler.Handle(testutil.Context(t), "ext-automation", "automation/jobs", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Handle(automation/jobs via getter) error = %v", err)
	}

	var jobs []automationpkg.Job
	decodeResult(t, result, &jobs)
	if jobs == nil {
		t.Fatal("automation/jobs result = nil, want empty slice")
	}
}

func TestHostAPIHandlerTaskOperationsRequireCapabilities(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-denied", nil, nil)

	tests := []struct {
		name   string
		method string
		params map[string]any
	}{
		{
			name:   "CreateDenied",
			method: "tasks/create",
			params: map[string]any{"scope": taskpkg.ScopeGlobal, "title": "Denied create"},
		},
		{
			name:   "UpdateDenied",
			method: "tasks/update",
			params: map[string]any{"id": "task-denied", "title": "Denied update"},
		},
		{
			name:   "RunStartDenied",
			method: "tasks/runs/start",
			params: map[string]any{"id": "run-denied", "idempotency_key": "idem-denied"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := env.call(t, "ext-denied", tt.method, tt.params)
			assertCapabilityDenied(t, err, tt.method)
		})
	}
}

func TestHostAPIHandlerTasksCreateUsesTrustedExtensionIdentity(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-tasks", []string{"tasks/create"}, []string{"task.write"})

	result, err := env.call(t, "ext-tasks", "tasks/create", map[string]any{
		"scope": taskpkg.ScopeGlobal,
		"title": "Trusted extension task",
		"created_by": map[string]any{
			"kind": "human",
			"ref":  "spoofed-user",
		},
		"origin": map[string]any{
			"kind": "cli",
			"ref":  "spoofed-origin",
		},
	})
	if err != nil {
		t.Fatalf("Handle(tasks/create) error = %v", err)
	}

	var created apicontract.TaskPayload
	decodeResult(t, result, &created)
	stored, err := env.registry.GetTask(testutil.Context(t), created.ID)
	if err != nil {
		t.Fatalf("registry.GetTask(%q) error = %v", created.ID, err)
	}
	if got, want := stored.CreatedBy.Kind, taskpkg.ActorKindExtension; got != want {
		t.Fatalf("stored.CreatedBy.Kind = %q, want %q", got, want)
	}
	if got, want := stored.CreatedBy.Ref, "ext-tasks"; got != want {
		t.Fatalf("stored.CreatedBy.Ref = %q, want %q", got, want)
	}
	if got, want := stored.Origin.Kind, taskpkg.OriginKindExtension; got != want {
		t.Fatalf("stored.Origin.Kind = %q, want %q", got, want)
	}
	if got, want := stored.Origin.Ref, "ext-tasks"; got != want {
		t.Fatalf("stored.Origin.Ref = %q, want %q", got, want)
	}
}

func TestHostAPIHandlerTaskRunStartRespectsManagerTransitions(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant(
		"ext-tasks",
		[]string{"tasks/create", "tasks/runs/enqueue", "tasks/runs/start"},
		[]string{"task.write"},
	)

	createResult, err := env.call(t, "ext-tasks", "tasks/create", map[string]any{
		"scope":     taskpkg.ScopeWorkspace,
		"title":     "Lifecycle guard task",
		"workspace": env.workspaceID,
	})
	if err != nil {
		t.Fatalf("Handle(tasks/create) error = %v", err)
	}

	var created apicontract.TaskPayload
	decodeResult(t, createResult, &created)

	enqueueResult, err := env.call(t, "ext-tasks", "tasks/runs/enqueue", map[string]any{
		"task_id":         created.ID,
		"idempotency_key": "enqueue-guard",
	})
	if err != nil {
		t.Fatalf("Handle(tasks/runs/enqueue) error = %v", err)
	}

	var run apicontract.TaskRunPayload
	decodeResult(t, enqueueResult, &run)

	_, err = env.call(t, "ext-tasks", "tasks/runs/start", map[string]any{
		"id":              run.ID,
		"idempotency_key": "start-guard",
	})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "invalid status transition")
}

func TestHostAPIHandlerTasksListAndGetReturnFilteredDetail(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-reader", []string{"tasks", "tasks/get"}, []string{"task.read"})

	actor := mustExtensionTaskActorContext(t, "seed-writer")
	parent, err := env.tasks.CreateTask(testutil.Context(t), taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: env.workspaceID,
		Title:       "Parent task",
		Owner: &taskpkg.Ownership{
			Kind: taskpkg.OwnerKindExtension,
			Ref:  "ops",
		},
		NetworkChannel: "tasks_ops",
	}, actor)
	if err != nil {
		t.Fatalf("tasks.CreateTask(parent) error = %v", err)
	}

	child, err := env.tasks.CreateChildTask(testutil.Context(t), parent.ID, taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: env.workspaceID,
		Title:       "Filtered child",
		Owner: &taskpkg.Ownership{
			Kind: taskpkg.OwnerKindExtension,
			Ref:  "ops",
		},
		NetworkChannel: "tasks_ops",
	}, actor)
	if err != nil {
		t.Fatalf("tasks.CreateChildTask(filtered) error = %v", err)
	}

	if _, err := env.tasks.CreateChildTask(testutil.Context(t), parent.ID, taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: env.workspaceID,
		Title:       "Other child",
		Owner: &taskpkg.Ownership{
			Kind: taskpkg.OwnerKindPool,
			Ref:  "backlog",
		},
		NetworkChannel: "tasks_other",
	}, actor); err != nil {
		t.Fatalf("tasks.CreateChildTask(other) error = %v", err)
	}

	blocker, err := env.tasks.CreateTask(testutil.Context(t), taskpkg.CreateTask{
		Scope:       taskpkg.ScopeWorkspace,
		WorkspaceID: env.workspaceID,
		Title:       "Blocking task",
	}, actor)
	if err != nil {
		t.Fatalf("tasks.CreateTask(blocker) error = %v", err)
	}
	if err := env.tasks.AddDependency(testutil.Context(t), taskpkg.AddDependency{
		TaskID:          child.ID,
		DependsOnTaskID: blocker.ID,
		Kind:            taskpkg.DependencyKindBlocks,
	}, actor); err != nil {
		t.Fatalf("tasks.AddDependency() error = %v", err)
	}

	run, err := env.tasks.EnqueueRun(testutil.Context(t), taskpkg.EnqueueRun{
		TaskID:         child.ID,
		IdempotencyKey: "seed-list-detail",
	}, actor)
	if err != nil {
		t.Fatalf("tasks.EnqueueRun() error = %v", err)
	}

	listResult, err := env.call(t, "ext-reader", "tasks", map[string]any{
		"scope":           taskpkg.ScopeWorkspace,
		"workspace":       env.workspaceID,
		"owner_kind":      taskpkg.OwnerKindExtension,
		"owner_ref":       "ops",
		"parent_task_id":  parent.ID,
		"network_channel": "tasks_ops",
		"limit":           1,
	})
	if err != nil {
		t.Fatalf("Handle(tasks) error = %v", err)
	}

	var listed []apicontract.TaskSummaryPayload
	decodeResult(t, listResult, &listed)
	if got, want := len(listed), 1; got != want {
		t.Fatalf("len(tasks) = %d, want %d", got, want)
	}
	if got, want := listed[0].ID, child.ID; got != want {
		t.Fatalf("tasks[0].ID = %q, want %q", got, want)
	}
	if listed[0].Owner == nil {
		t.Fatal("tasks[0].Owner = nil, want extension owner")
	}
	if got, want := listed[0].Owner.Ref, "ops"; got != want {
		t.Fatalf("tasks[0].Owner.Ref = %q, want %q", got, want)
	}

	getResult, err := env.call(t, "ext-reader", "tasks/get", map[string]any{"id": child.ID})
	if err != nil {
		t.Fatalf("Handle(tasks/get) error = %v", err)
	}

	var detail apicontract.TaskDetailPayload
	decodeResult(t, getResult, &detail)
	if got, want := detail.Task.ID, child.ID; got != want {
		t.Fatalf("tasks/get.task.id = %q, want %q", got, want)
	}
	if got, want := len(detail.Dependencies), 1; got != want {
		t.Fatalf("len(tasks/get.dependencies) = %d, want %d", got, want)
	}
	if got, want := detail.Dependencies[0].DependsOnTaskID, blocker.ID; got != want {
		t.Fatalf("tasks/get.dependencies[0].depends_on_task_id = %q, want %q", got, want)
	}
	if got, want := len(detail.Runs), 1; got != want {
		t.Fatalf("len(tasks/get.runs) = %d, want %d", got, want)
	}
	if got, want := detail.Runs[0].ID, run.ID; got != want {
		t.Fatalf("tasks/get.runs[0].id = %q, want %q", got, want)
	}
	if len(detail.Events) == 0 {
		t.Fatal("tasks/get.events = 0, want audit events")
	}
}

func TestHostAPIHandlerTasksUpdateAndCancelMutateTask(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant("ext-writer", []string{"tasks/create", "tasks/update", "tasks/cancel"}, []string{"task.write"})

	createResult, err := env.call(t, "ext-writer", "tasks/create", map[string]any{
		"scope":           taskpkg.ScopeWorkspace,
		"workspace":       env.workspaceID,
		"title":           "Original title",
		"description":     "Original description",
		"network_channel": "tasks_initial",
		"owner": map[string]any{
			"kind": taskpkg.OwnerKindPool,
			"ref":  "triage",
		},
		"metadata": map[string]any{"phase": "initial"},
	})
	if err != nil {
		t.Fatalf("Handle(tasks/create) error = %v", err)
	}

	var created apicontract.TaskPayload
	decodeResult(t, createResult, &created)

	updateResult, err := env.call(t, "ext-writer", "tasks/update", map[string]any{
		"id":              created.ID,
		"title":           " Updated title ",
		"description":     " Updated description ",
		"network_channel": "tasks_updated",
		"owner": map[string]any{
			"kind": taskpkg.OwnerKindExtension,
			"ref":  "ext-writer",
		},
		"metadata": map[string]any{"phase": "updated"},
	})
	if err != nil {
		t.Fatalf("Handle(tasks/update) error = %v", err)
	}

	var updated apicontract.TaskPayload
	decodeResult(t, updateResult, &updated)
	if got, want := updated.Title, "Updated title"; got != want {
		t.Fatalf("tasks/update title = %q, want %q", got, want)
	}
	if got, want := updated.Description, "Updated description"; got != want {
		t.Fatalf("tasks/update description = %q, want %q", got, want)
	}
	if got, want := updated.NetworkChannel, "tasks_updated"; got != want {
		t.Fatalf("tasks/update network_channel = %q, want %q", got, want)
	}
	if updated.Owner == nil {
		t.Fatal("tasks/update owner = nil, want extension owner")
	}
	if got, want := updated.Owner.Ref, "ext-writer"; got != want {
		t.Fatalf("tasks/update owner.ref = %q, want %q", got, want)
	}
	if !strings.Contains(string(updated.Metadata), `"updated"`) {
		t.Fatalf("tasks/update metadata = %s, want updated marker", string(updated.Metadata))
	}

	clearOwnerResult, err := env.call(t, "ext-writer", "tasks/update", map[string]any{
		"id":          created.ID,
		"clear_owner": true,
	})
	if err != nil {
		t.Fatalf("Handle(tasks/update clear_owner) error = %v", err)
	}

	var cleared apicontract.TaskPayload
	decodeResult(t, clearOwnerResult, &cleared)
	if cleared.Owner != nil {
		t.Fatalf("tasks/update clear_owner owner = %#v, want nil", cleared.Owner)
	}

	cancelResult, err := env.call(t, "ext-writer", "tasks/cancel", map[string]any{
		"id":     created.ID,
		"reason": " user requested ",
		"metadata": map[string]any{
			"source": "host-api",
		},
	})
	if err != nil {
		t.Fatalf("Handle(tasks/cancel) error = %v", err)
	}

	var cancelled apicontract.TaskPayload
	decodeResult(t, cancelResult, &cancelled)
	if got, want := cancelled.Status, taskpkg.TaskStatusCancelled; got != want {
		t.Fatalf("tasks/cancel status = %q, want %q", got, want)
	}
	if cancelled.ClosedAt.IsZero() {
		t.Fatal("tasks/cancel closed_at = zero, want terminal timestamp")
	}

	stored, err := env.registry.GetTask(testutil.Context(t), created.ID)
	if err != nil {
		t.Fatalf("registry.GetTask(%q) error = %v", created.ID, err)
	}
	if got, want := stored.Status, taskpkg.TaskStatusCancelled; got != want {
		t.Fatalf("stored.Status = %q, want %q", got, want)
	}
	if stored.Owner != nil {
		t.Fatalf("stored.Owner = %#v, want nil after clear_owner", stored.Owner)
	}
}

func TestHostAPIHandlerTaskRunLifecycleOperationsAndFiltering(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant(
		"ext-runs",
		[]string{
			"tasks/create",
			"tasks/runs",
			"tasks/runs/enqueue",
			"tasks/runs/claim",
			"tasks/runs/attach_session",
			"tasks/runs/start",
			"tasks/runs/complete",
			"tasks/runs/fail",
			"tasks/runs/cancel",
		},
		[]string{"task.read", "task.write"},
	)

	createTask := func(title string) apicontract.TaskPayload {
		t.Helper()

		result, err := env.call(t, "ext-runs", "tasks/create", map[string]any{
			"scope":     taskpkg.ScopeWorkspace,
			"workspace": env.workspaceID,
			"title":     title,
		})
		if err != nil {
			t.Fatalf("Handle(tasks/create %q) error = %v", title, err)
		}
		var task apicontract.TaskPayload
		decodeResult(t, result, &task)
		return task
	}

	enqueueRun := func(taskID string, idempotencyKey string) apicontract.TaskRunPayload {
		t.Helper()

		result, err := env.call(t, "ext-runs", "tasks/runs/enqueue", map[string]any{
			"task_id":         taskID,
			"idempotency_key": idempotencyKey,
		})
		if err != nil {
			t.Fatalf("Handle(tasks/runs/enqueue %q) error = %v", taskID, err)
		}
		var run apicontract.TaskRunPayload
		decodeResult(t, result, &run)
		return run
	}

	claimRun := func(runID string, idempotencyKey string) apicontract.TaskRunPayload {
		t.Helper()

		result, err := env.call(t, "ext-runs", "tasks/runs/claim", map[string]any{
			"id":              runID,
			"idempotency_key": idempotencyKey,
		})
		if err != nil {
			t.Fatalf("Handle(tasks/runs/claim %q) error = %v", runID, err)
		}
		var run apicontract.TaskRunPayload
		decodeResult(t, result, &run)
		return run
	}

	completedTask := createTask("Completed run task")
	completedQueued := enqueueRun(completedTask.ID, "enqueue-complete")
	completedClaimed := claimRun(completedQueued.ID, "claim-complete")
	if got, want := completedClaimed.Status, taskpkg.TaskRunStatusClaimed; got != want {
		t.Fatalf("tasks/runs/claim status = %q, want %q", got, want)
	}
	if completedClaimed.ClaimedBy == nil {
		t.Fatal("tasks/runs/claim claimed_by = nil, want extension actor")
	}

	boundSession := env.createSession(t)
	attachResult, err := env.call(t, "ext-runs", "tasks/runs/attach_session", map[string]any{
		"id":         completedQueued.ID,
		"session_id": boundSession.ID,
	})
	if err != nil {
		t.Fatalf("Handle(tasks/runs/attach_session) error = %v", err)
	}

	var attached apicontract.TaskRunPayload
	decodeResult(t, attachResult, &attached)
	if got, want := attached.SessionID, boundSession.ID; got != want {
		t.Fatalf("tasks/runs/attach_session session_id = %q, want %q", got, want)
	}

	startResult, err := env.call(t, "ext-runs", "tasks/runs/start", map[string]any{
		"id":              completedQueued.ID,
		"idempotency_key": "start-complete",
	})
	if err != nil {
		t.Fatalf("Handle(tasks/runs/start) error = %v", err)
	}

	var started apicontract.TaskRunPayload
	decodeResult(t, startResult, &started)
	if got, want := started.Status, taskpkg.TaskRunStatusRunning; got != want {
		t.Fatalf("tasks/runs/start status = %q, want %q", got, want)
	}

	completeResult, err := env.call(t, "ext-runs", "tasks/runs/complete", map[string]any{
		"id":     completedQueued.ID,
		"result": map[string]any{"ok": true},
	})
	if err != nil {
		t.Fatalf("Handle(tasks/runs/complete) error = %v", err)
	}

	var completed apicontract.TaskRunPayload
	decodeResult(t, completeResult, &completed)
	if got, want := completed.Status, taskpkg.TaskRunStatusCompleted; got != want {
		t.Fatalf("tasks/runs/complete status = %q, want %q", got, want)
	}
	if !strings.Contains(string(completed.Result), `"ok":true`) {
		t.Fatalf("tasks/runs/complete result = %s, want ok marker", string(completed.Result))
	}

	failedTask := createTask("Failed run task")
	failedQueued := enqueueRun(failedTask.ID, "enqueue-fail")
	_ = claimRun(failedQueued.ID, "claim-fail")
	_, err = env.call(t, "ext-runs", "tasks/runs/start", map[string]any{
		"id":              failedQueued.ID,
		"idempotency_key": "start-fail",
	})
	if err != nil {
		t.Fatalf("Handle(tasks/runs/start fail path) error = %v", err)
	}
	failResult, err := env.call(t, "ext-runs", "tasks/runs/fail", map[string]any{
		"id":    failedQueued.ID,
		"error": " execution failed ",
		"metadata": map[string]any{
			"retryable": false,
		},
	})
	if err != nil {
		t.Fatalf("Handle(tasks/runs/fail) error = %v", err)
	}

	var failed apicontract.TaskRunPayload
	decodeResult(t, failResult, &failed)
	if got, want := failed.Status, taskpkg.TaskRunStatusFailed; got != want {
		t.Fatalf("tasks/runs/fail status = %q, want %q", got, want)
	}
	if got, want := failed.Error, "execution failed"; got != want {
		t.Fatalf("tasks/runs/fail error = %q, want %q", got, want)
	}

	cancelledTask := createTask("Cancelled run task")
	cancelledQueued := enqueueRun(cancelledTask.ID, "enqueue-cancel")
	_ = claimRun(cancelledQueued.ID, "claim-cancel")
	cancelRunResult, err := env.call(t, "ext-runs", "tasks/runs/cancel", map[string]any{
		"id":     cancelledQueued.ID,
		"reason": " no longer needed ",
		"metadata": map[string]any{
			"source": "extension",
		},
	})
	if err != nil {
		t.Fatalf("Handle(tasks/runs/cancel) error = %v", err)
	}

	var cancelled apicontract.TaskRunPayload
	decodeResult(t, cancelRunResult, &cancelled)
	if got, want := cancelled.Status, taskpkg.TaskRunStatusCancelled; got != want {
		t.Fatalf("tasks/runs/cancel status = %q, want %q", got, want)
	}

	runsResult, err := env.call(t, "ext-runs", "tasks/runs", map[string]any{
		"id":         completedTask.ID,
		"status":     taskpkg.TaskRunStatusCompleted,
		"session_id": boundSession.ID,
		"limit":      1,
	})
	if err != nil {
		t.Fatalf("Handle(tasks/runs) error = %v", err)
	}

	var filtered []apicontract.TaskRunPayload
	decodeResult(t, runsResult, &filtered)
	if got, want := len(filtered), 1; got != want {
		t.Fatalf("len(tasks/runs) = %d, want %d", got, want)
	}
	if got, want := filtered[0].ID, completedQueued.ID; got != want {
		t.Fatalf("tasks/runs[0].id = %q, want %q", got, want)
	}
	if got, want := filtered[0].SessionID, boundSession.ID; got != want {
		t.Fatalf("tasks/runs[0].session_id = %q, want %q", got, want)
	}
}

func TestHostAPIHandlerTaskMethodsValidateInputsAndConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("MissingManager", func(t *testing.T) {
		t.Parallel()

		checker := &CapabilityChecker{}
		checker.Register("ext-tasks", SourceUser, &Manifest{
			Actions: ActionsConfig{Requires: []string{"tasks", "tasks/get", "tasks/runs"}},
			Security: SecurityConfig{
				Capabilities: []string{"task.read"},
			},
		})

		handler := NewHostAPIHandler(
			nil,
			nil,
			nil,
			nil,
			WithHostAPICapabilityChecker(checker),
			WithHostAPIRateLimit(1000, 1000),
		)

		tests := []struct {
			name   string
			method string
			params map[string]any
		}{
			{name: "List", method: "tasks", params: map[string]any{}},
			{name: "Get", method: "tasks/get", params: map[string]any{"id": "task-1"}},
			{name: "Runs", method: "tasks/runs", params: map[string]any{"id": "task-1"}},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				params, err := marshalParams(tt.params)
				if err != nil {
					t.Fatalf("marshalParams() error = %v", err)
				}

				_, err = handler.Handle(testutil.Context(t), "ext-tasks", tt.method, params)
				assertErrorContains(t, err, "task manager is not configured")
			})
		}
	})

	t.Run("InvalidInputs", func(t *testing.T) {
		t.Parallel()

		env := newHostAPITestEnv(t)
		env.grant(
			"ext-tasks",
			[]string{"tasks", "tasks/create", "tasks/update", "tasks/runs/attach_session"},
			[]string{"task.read", "task.write"},
		)

		tests := []struct {
			name     string
			method   string
			params   map[string]any
			wantCode int
			wantText string
		}{
			{
				name:   "UnknownWorkspace",
				method: "tasks/create",
				params: map[string]any{
					"scope":     taskpkg.ScopeWorkspace,
					"workspace": "ws-missing",
					"title":     "Missing workspace task",
				},
				wantCode: HostAPINotFoundCode,
				wantText: "workspace",
			},
			{
				name:   "InvalidListChannel",
				method: "tasks",
				params: map[string]any{
					"network_channel": "not valid",
				},
				wantCode: HostAPIInvalidParamsCode,
				wantText: "task_query.network_channel",
			},
			{
				name:     "UpdateRequiresChanges",
				method:   "tasks/update",
				params:   map[string]any{"id": "task-1"},
				wantCode: HostAPIInvalidParamsCode,
				wantText: "at least one mutable field",
			},
			{
				name:     "AttachRequiresSessionID",
				method:   "tasks/runs/attach_session",
				params:   map[string]any{"id": "run-1"},
				wantCode: HostAPIInvalidParamsCode,
				wantText: "session_id is required",
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				_, err := env.call(t, "ext-tasks", tt.method, tt.params)
				assertRPCErrorCode(t, err, tt.wantCode)
				assertErrorContains(t, err, tt.wantText)
			})
		}
	})
}

func TestHostAPIHandlerTaskMethodsRequireIdentifiers(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant(
		"ext-ids",
		[]string{
			"tasks/get",
			"tasks/update",
			"tasks/cancel",
			"tasks/runs",
			"tasks/runs/enqueue",
			"tasks/runs/claim",
			"tasks/runs/start",
			"tasks/runs/complete",
			"tasks/runs/fail",
			"tasks/runs/cancel",
		},
		[]string{"task.read", "task.write"},
	)

	tests := []struct {
		name     string
		method   string
		params   map[string]any
		wantText string
	}{
		{name: "Get", method: "tasks/get", params: map[string]any{}, wantText: "id is required"},
		{name: "Update", method: "tasks/update", params: map[string]any{"title": "rename"}, wantText: "id is required"},
		{name: "Cancel", method: "tasks/cancel", params: map[string]any{"reason": "stop"}, wantText: "id is required"},
		{name: "RunsList", method: "tasks/runs", params: map[string]any{}, wantText: "id is required"},
		{name: "RunEnqueue", method: "tasks/runs/enqueue", params: map[string]any{"idempotency_key": "idem"}, wantText: "task_id is required"},
		{name: "RunClaim", method: "tasks/runs/claim", params: map[string]any{"idempotency_key": "idem"}, wantText: "id is required"},
		{name: "RunStart", method: "tasks/runs/start", params: map[string]any{"idempotency_key": "idem"}, wantText: "id is required"},
		{name: "RunComplete", method: "tasks/runs/complete", params: map[string]any{"result": map[string]any{"ok": true}}, wantText: "id is required"},
		{name: "RunFail", method: "tasks/runs/fail", params: map[string]any{"error": "boom"}, wantText: "id is required"},
		{name: "RunCancel", method: "tasks/runs/cancel", params: map[string]any{"reason": "cancel"}, wantText: "id is required"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := env.call(t, "ext-ids", tt.method, tt.params)
			assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
			assertErrorContains(t, err, tt.wantText)
		})
	}
}

func TestHostAPIHandlerTaskMethodsReturnNotFoundForMissingRecords(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant(
		"ext-missing",
		[]string{
			"tasks/get",
			"tasks/update",
			"tasks/cancel",
			"tasks/runs",
			"tasks/runs/claim",
			"tasks/runs/start",
			"tasks/runs/attach_session",
			"tasks/runs/complete",
			"tasks/runs/fail",
			"tasks/runs/cancel",
		},
		[]string{"task.read", "task.write"},
	)

	tests := []struct {
		name     string
		method   string
		params   map[string]any
		wantText string
	}{
		{name: "GetTask", method: "tasks/get", params: map[string]any{"id": "task-missing"}, wantText: "task not found"},
		{name: "UpdateTask", method: "tasks/update", params: map[string]any{"id": "task-missing", "title": "rename"}, wantText: "task not found"},
		{name: "CancelTask", method: "tasks/cancel", params: map[string]any{"id": "task-missing"}, wantText: "task not found"},
		{name: "ListRuns", method: "tasks/runs", params: map[string]any{"id": "task-missing"}, wantText: "task not found"},
		{name: "ClaimRun", method: "tasks/runs/claim", params: map[string]any{"id": "run-missing", "idempotency_key": "idem"}, wantText: "task run not found"},
		{name: "StartRun", method: "tasks/runs/start", params: map[string]any{"id": "run-missing", "idempotency_key": "idem"}, wantText: "task run not found"},
		{name: "AttachRun", method: "tasks/runs/attach_session", params: map[string]any{"id": "run-missing", "session_id": "sess-missing"}, wantText: "task run not found"},
		{name: "CompleteRun", method: "tasks/runs/complete", params: map[string]any{"id": "run-missing", "result": map[string]any{"ok": true}}, wantText: "task run not found"},
		{name: "FailRun", method: "tasks/runs/fail", params: map[string]any{"id": "run-missing", "error": "boom"}, wantText: "task run not found"},
		{name: "CancelRun", method: "tasks/runs/cancel", params: map[string]any{"id": "run-missing"}, wantText: "task run not found"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := env.call(t, "ext-missing", tt.method, tt.params)
			assertRPCErrorCode(t, err, HostAPINotFoundCode)
			assertErrorContains(t, err, tt.wantText)
		})
	}
}

func TestMapTaskRPCErrorTranslatesKnownErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		resource string
		id       string
		err      error
		wantCode int
		wantText string
		wantNil  bool
		wantSame bool
	}{
		{name: "Nil", err: nil, wantNil: true},
		{name: "WorkspaceNotFound", resource: "workspace", id: "ws-1", err: workspacepkg.ErrWorkspaceNotFound, wantCode: HostAPINotFoundCode, wantText: "workspace not found"},
		{name: "TaskNotFound", resource: "task", id: "task-1", err: taskpkg.ErrTaskNotFound, wantCode: HostAPINotFoundCode, wantText: "task not found"},
		{name: "RunNotFound", resource: "task_run", id: "run-1", err: taskpkg.ErrTaskRunNotFound, wantCode: HostAPINotFoundCode, wantText: "task run not found"},
		{name: "DependencyNotFound", resource: "task_dependency", id: "dep-1", err: taskpkg.ErrTaskDependencyNotFound, wantCode: HostAPINotFoundCode, wantText: "task dependency not found"},
		{name: "PermissionDenied", resource: "task", id: "task-1", err: taskpkg.ErrPermissionDenied, wantCode: HostAPIInvalidParamsCode, wantText: "permission denied"},
		{name: "StaleNetworkChannel", resource: "task_run", id: "run-1", err: taskpkg.ErrStaleNetworkChannel, wantCode: HostAPIInvalidParamsCode, wantText: "stale network channel"},
		{name: "Passthrough", resource: "task", id: "task-1", err: errors.New("boom"), wantSame: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mapped := mapTaskRPCError(tt.resource, tt.id, tt.err)
			if tt.wantNil {
				if mapped != nil {
					t.Fatalf("mapTaskRPCError() = %v, want nil", mapped)
				}
				return
			}
			if tt.wantSame {
				if !errors.Is(mapped, tt.err) {
					t.Fatalf("mapTaskRPCError() = %v, want same error %v", mapped, tt.err)
				}
				return
			}

			assertRPCErrorCode(t, mapped, tt.wantCode)
			assertErrorContains(t, mapped, tt.wantText)
		})
	}
}

func TestHostAPITaskHelpersHandleZeroAndUnavailableCases(t *testing.T) {
	t.Parallel()

	var nilHandler *HostAPIHandler
	_, err := nilHandler.taskManager()
	assertErrorContains(t, err, "host api handler is required")

	_, err = (&HostAPIHandler{}).taskManager()
	assertErrorContains(t, err, "task manager is not configured")

	env := newHostAPITestEnv(t)
	raw, err := marshalParams(map[string]any{
		"scope": taskpkg.ScopeGlobal,
		"title": "No context task",
	})
	if err != nil {
		t.Fatalf("marshalParams() error = %v", err)
	}

	_, err = env.handler.handleTasksCreate(testutil.Context(t), raw)
	assertRPCErrorCode(t, err, HostAPIUnavailableCode)
	assertErrorContains(t, err, "extension name is not available")

	zeroTask := taskPayloadFromTask(nil)
	if zeroTask.ID != "" {
		t.Fatalf("taskPayloadFromTask(nil).ID = %q, want empty", zeroTask.ID)
	}

	zeroRun := taskRunPayloadFromRun(nil)
	if zeroRun.ID != "" {
		t.Fatalf("taskRunPayloadFromRun(nil).ID = %q, want empty", zeroRun.ID)
	}

	zeroDetail := taskDetailPayloadFromView(nil)
	if zeroDetail.Task.ID != "" {
		t.Fatalf("taskDetailPayloadFromView(nil).Task.ID = %q, want empty", zeroDetail.Task.ID)
	}

	filtered := filterTaskRuns([]taskpkg.TaskRun{
		{ID: "run-1", Status: taskpkg.TaskRunStatusRunning, SessionID: "sess-1"},
		{ID: "run-2", Status: taskpkg.TaskRunStatusCompleted, SessionID: "sess-2"},
		{ID: "run-3", Status: taskpkg.TaskRunStatusCompleted, SessionID: "sess-1"},
	}, taskpkg.TaskRunQuery{
		Status:    taskpkg.TaskRunStatusCompleted,
		SessionID: "sess-1",
		Limit:     1,
	})
	if got, want := len(filtered), 1; got != want {
		t.Fatalf("len(filterTaskRuns) = %d, want %d", got, want)
	}
	if got, want := filtered[0].ID, "run-3"; got != want {
		t.Fatalf("filterTaskRuns()[0].ID = %q, want %q", got, want)
	}
}

func TestHostAPIHandlerTaskMethodsRejectInvalidPayloadCombinations(t *testing.T) {
	t.Parallel()

	env := newHostAPITestEnv(t)
	env.grant(
		"ext-invalid",
		[]string{"tasks/create", "tasks/update", "tasks/runs/enqueue"},
		[]string{"task.write"},
	)

	_, err := env.call(t, "ext-invalid", "tasks/create", map[string]any{
		"scope":           taskpkg.ScopeGlobal,
		"title":           "Invalid channel task",
		"network_channel": "not valid",
	})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "create_task.network_channel")

	createResult, err := env.call(t, "ext-invalid", "tasks/create", map[string]any{
		"scope":     taskpkg.ScopeWorkspace,
		"workspace": env.workspaceID,
		"title":     "Mutable task",
	})
	if err != nil {
		t.Fatalf("Handle(tasks/create mutable task) error = %v", err)
	}

	var created apicontract.TaskPayload
	decodeResult(t, createResult, &created)

	_, err = env.call(t, "ext-invalid", "tasks/update", map[string]any{
		"id":          created.ID,
		"owner":       map[string]any{"kind": taskpkg.OwnerKindPool, "ref": "triage"},
		"clear_owner": true,
	})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "cannot both be set")

	_, err = env.call(t, "ext-invalid", "tasks/runs/enqueue", map[string]any{
		"task_id":         created.ID,
		"idempotency_key": "idem-invalid-channel",
		"network_channel": "not valid",
	})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "enqueue_run.network_channel")
}

func TestHostAPITaskRequestHelpersRejectInvalidPayloads(t *testing.T) {
	t.Parallel()

	oversizedMetadata := json.RawMessage(fmt.Sprintf("%q", strings.Repeat("m", taskpkg.MaxPayloadBytes+1)))
	oversizedResult := json.RawMessage(fmt.Sprintf("%q", strings.Repeat("r", taskpkg.MaxResultBytes+1)))

	_, err := cancelTaskFromRequest(apicontract.CancelTaskRequest{Metadata: oversizedMetadata})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "cancel_task.metadata")

	_, err = completeTaskRunFromRequest(apicontract.CompleteTaskRunRequest{Result: oversizedResult})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "run_result.value")

	_, err = failTaskRunFromRequest(apicontract.FailTaskRunRequest{})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "run_failure.error")

	_, err = cancelTaskRunFromRequest(apicontract.CancelTaskRunRequest{Metadata: oversizedMetadata})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "cancel_run.metadata")

	_, err = taskRunQueryFromParams(apicontract.TaskRunListQuery{Limit: -1})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "task_run_query.limit")

	env := newHostAPITestEnv(t)
	_, err = env.handler.taskQueryFromParams(testutil.Context(t), hostAPITasksParams{Limit: -1})
	assertRPCErrorCode(t, err, HostAPIInvalidParamsCode)
	assertErrorContains(t, err, "task_query.limit")
}

type hostAPITestEnv struct {
	nowMu       sync.RWMutex
	now         time.Time
	homePaths   aghconfig.HomePaths
	workspaceID string
	workspace   workspacepkg.ResolvedWorkspace
	registry    *globaldb.GlobalDB
	bridges     *bridgepkg.Service
	sessions    *session.Manager
	automation  HostAPIAutomationManager
	tasks       taskpkg.Manager
	observer    *observepkg.Observer
	memory      *memory.Store
	skills      *skillspkg.Registry
	workspaces  *hostAPIFakeWorkspaceResolver
	driver      *hostAPIFakeDriver
	checker     *CapabilityChecker
	handler     *HostAPIHandler
}

type hostAPITestEnvConfig struct {
	hooks *hookspkg.Hooks
}

type hostAPITestEnvOption func(*hostAPITestEnvConfig)

type hostAPITestTaskSessionExecutor struct {
	sessions            *session.Manager
	globalWorkspacePath string
}

func mustExtensionTaskActorContext(t testing.TB, extensionName string) taskpkg.ActorContext {
	t.Helper()

	actor, err := taskpkg.DeriveExtensionActorContext(extensionName, "")
	if err != nil {
		t.Fatalf("DeriveExtensionActorContext(%q) error = %v", extensionName, err)
	}
	return actor
}

func (e *hostAPITestTaskSessionExecutor) StartTaskSession(ctx context.Context, spec taskpkg.StartTaskSession) (*taskpkg.SessionRef, error) {
	if ctx == nil {
		return nil, errors.New("extension: host api test task start context is required")
	}

	opts := session.CreateOpts{
		AgentName: "coder",
		Name:      "task:" + strings.TrimSpace(spec.Task.Title),
		Channel:   strings.TrimSpace(spec.Run.NetworkChannel),
		Type:      session.SessionTypeSystem,
	}
	switch spec.Task.Scope.Normalize() {
	case taskpkg.ScopeWorkspace:
		opts.Workspace = strings.TrimSpace(spec.Task.WorkspaceID)
	case taskpkg.ScopeGlobal:
		opts.WorkspacePath = strings.TrimSpace(e.globalWorkspacePath)
	default:
		return nil, fmt.Errorf("%w: unsupported task scope %q", taskpkg.ErrValidation, spec.Task.Scope)
	}

	created, err := e.sessions.Create(ctx, opts)
	if err != nil {
		return nil, err
	}
	info := created.Info()
	if info == nil {
		return nil, fmt.Errorf("%w: task session create returned nil session info", taskpkg.ErrValidation)
	}
	return &taskpkg.SessionRef{
		SessionID:   info.ID,
		WorkspaceID: info.WorkspaceID,
		StartedAt:   info.CreatedAt,
	}, nil
}

func (e *hostAPITestTaskSessionExecutor) AttachTaskSession(ctx context.Context, _ string, sessionID string) (*taskpkg.SessionRef, error) {
	if ctx == nil {
		return nil, errors.New("extension: host api test task attach context is required")
	}

	info, err := e.sessions.Status(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return nil, err
	}
	if info == nil || info.State != session.StateActive {
		return nil, fmt.Errorf("%w: session %q is not active", taskpkg.ErrSessionAttachNotAllowed, strings.TrimSpace(sessionID))
	}
	return &taskpkg.SessionRef{
		SessionID:   info.ID,
		WorkspaceID: info.WorkspaceID,
		StartedAt:   info.CreatedAt,
	}, nil
}

func (e *hostAPITestTaskSessionExecutor) RequestTaskStop(ctx context.Context, sessionID string, _ taskpkg.StopReason) error {
	return e.sessions.RequestStopWithCause(ctx, strings.TrimSpace(sessionID), session.CauseUserRequested, "task cancellation")
}

func (e *hostAPITestTaskSessionExecutor) ForceTaskStop(ctx context.Context, sessionID string, _ taskpkg.StopReason) error {
	return e.sessions.StopWithCause(ctx, strings.TrimSpace(sessionID), session.CauseUserRequested, "task cancellation")
}

func newHostAPITestEnv(t *testing.T, opts ...hostAPITestEnvOption) *hostAPITestEnv {
	t.Helper()

	cfg := hostAPITestEnvConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	homePaths, err := aghconfig.ResolveHomePathsFrom(filepath.Join(t.TempDir(), "home"))
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	workspaceRoot := filepath.Join(t.TempDir(), "workspace")
	skillDir := filepath.Join(workspaceRoot, aghconfig.DirName, aghconfig.SkillsDirName, "workspace-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(skillDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(`---
name: workspace-review
description: Review workspace changes
---

Review the workspace changes carefully.
`), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}

	baseNow := time.Date(2026, 4, 10, 18, 0, 0, 0, time.UTC)
	env := &hostAPITestEnv{now: baseNow, homePaths: homePaths}
	resolvedWorkspace := workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      "ws-host-api",
			RootDir: workspaceRoot,
			Name:    "host-api-workspace",
		},
		Config: aghconfig.Config{
			Defaults: aghconfig.DefaultsConfig{Agent: "coder"},
			Providers: map[string]aghconfig.ProviderConfig{
				"fake": {Command: "fake-agent"},
			},
			Permissions: aghconfig.PermissionsConfig{Mode: aghconfig.PermissionModeApproveAll},
		},
		Agents: []aghconfig.AgentDef{{
			Name:        "coder",
			Provider:    "fake",
			Permissions: string(aghconfig.PermissionModeApproveAll),
			Prompt:      "You are a reliable coder.",
		}},
		Skills: []workspacepkg.SkillPath{{
			Dir:    skillDir,
			Source: "workspace",
		}},
		ResolvedAt: baseNow,
	}

	workspaces := newHostAPIFakeWorkspaceResolver(resolvedWorkspace)
	driver := newHostAPIFakeDriver(baseNow)
	source := &hostAPISessionSource{}
	registry, err := globaldb.OpenGlobalDB(testutil.Context(t), homePaths.DatabaseFile)
	if err != nil {
		t.Fatalf("globaldb.OpenGlobalDB() error = %v", err)
	}
	if err := registry.InsertWorkspace(testutil.Context(t), resolvedWorkspace.Workspace); err != nil {
		t.Fatalf("registry.InsertWorkspace() error = %v", err)
	}
	bridgeRegistry := bridgepkg.NewRegistry(registry, bridgepkg.WithNow(func() time.Time { return env.currentTime() }))

	observer, err := observepkg.New(testutil.Context(t),
		observepkg.WithRegistry(registry),
		observepkg.WithHomePaths(homePaths),
		observepkg.WithSessionSource(source),
		observepkg.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		observepkg.WithNow(func() time.Time { return env.currentTime().Add(time.Hour) }),
		observepkg.WithStartTime(baseNow),
	)
	if err != nil {
		_ = registry.Close(testutil.Context(t))
		t.Fatalf("observe.New() error = %v", err)
	}
	t.Cleanup(func() {
		if err := observer.Close(testutil.Context(t)); err != nil {
			t.Fatalf("observer.Close() error = %v", err)
		}
	})

	sessions, err := session.NewManager(
		session.WithHomePaths(homePaths),
		session.WithDriver(driver),
		session.WithNotifier(observer),
		session.WithWorkspaceResolver(workspaces),
		session.WithStore(func(ctx context.Context, sessionID string, path string) (session.EventRecorder, error) {
			return storeSessionDB(ctx, sessionID, path)
		}),
		session.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		session.WithNow(func() time.Time { return env.currentTime() }),
		session.WithSessionIDGenerator(sequentialSessionIDGenerator("sess")),
		session.WithTurnIDGenerator(sequentialSessionIDGenerator("turn")),
	)
	if err != nil {
		t.Fatalf("session.NewManager() error = %v", err)
	}
	source.manager = sessions

	memoryStore := memory.NewStore(homePaths.MemoryDir)
	if err := memoryStore.EnsureDirs(); err != nil {
		t.Fatalf("memory.EnsureDirs() error = %v", err)
	}

	skillsRegistry := skillspkg.NewRegistry(skillspkg.RegistryConfig{}, skillspkg.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	checker := &CapabilityChecker{}
	automationOpts := []automationpkg.Option{
		automationpkg.WithStore(registry),
		automationpkg.WithSessions(sessions),
		automationpkg.WithWorkspaceResolver(workspaces),
		automationpkg.WithConfig(aghconfig.AutomationConfig{
			Timezone:          automationpkg.DefaultTimezone,
			MaxConcurrentJobs: automationpkg.DefaultMaxConcurrentJobs,
			DefaultFireLimit:  automationpkg.DefaultFireLimitConfig(),
		}),
		automationpkg.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		automationpkg.WithGlobalWorkspacePath(homePaths.HomeDir),
	}
	if cfg.hooks != nil {
		automationOpts = append(automationOpts, automationpkg.WithHooks(cfg.hooks))
	}
	automationManager, err := automationpkg.New(automationOpts...)
	if err != nil {
		t.Fatalf("automation.New() error = %v", err)
	}
	if err := automationManager.Start(testutil.Context(t)); err != nil {
		t.Fatalf("automation.Start() error = %v", err)
	}
	t.Cleanup(func() {
		if err := automationManager.Shutdown(testutil.Context(t)); err != nil {
			t.Fatalf("automation.Shutdown() error = %v", err)
		}
	})

	taskManager, err := taskpkg.NewManager(
		taskpkg.WithStore(registry),
		taskpkg.WithSessionExecutor(&hostAPITestTaskSessionExecutor{
			sessions:            sessions,
			globalWorkspacePath: homePaths.HomeDir,
		}),
		taskpkg.WithManagerNow(func() time.Time { return env.currentTime() }),
	)
	if err != nil {
		t.Fatalf("task.NewManager() error = %v", err)
	}

	handler := NewHostAPIHandler(
		sessions,
		memoryStore,
		observer,
		skillsRegistry,
		WithHostAPIAutomationManager(automationManager),
		WithHostAPITaskManager(taskManager),
		WithHostAPICapabilityChecker(checker),
		WithHostAPIWorkspaceResolver(workspaces),
		WithHostAPIBridgeRegistry(bridgeRegistry),
		WithHostAPIBridgeDedupStore(registry),
		WithHostAPINow(func() time.Time { return env.currentTime() }),
		WithHostAPIBridgeIngressConfig(15*time.Minute, time.Minute),
		WithHostAPIRateLimit(1000, 1000),
	)

	env.workspaceID = resolvedWorkspace.ID
	env.workspace = resolvedWorkspace
	env.registry = registry
	env.bridges = bridgeRegistry
	env.sessions = sessions
	env.automation = automationManager
	env.tasks = taskManager
	env.observer = observer
	env.memory = memoryStore
	env.skills = skillsRegistry
	env.workspaces = workspaces
	env.driver = driver
	env.checker = checker
	env.handler = handler
	return env
}

func (e *hostAPITestEnv) grant(extName string, actions []string, security []string) {
	e.checker.Register(extName, SourceUser, &Manifest{
		Actions:  ActionsConfig{Requires: append([]string(nil), actions...)},
		Security: SecurityConfig{Capabilities: append([]string(nil), security...)},
	})
}

func (e *hostAPITestEnv) currentTime() time.Time {
	e.nowMu.RLock()
	defer e.nowMu.RUnlock()
	return e.now
}

func (e *hostAPITestEnv) advanceTime(delta time.Duration) time.Time {
	e.nowMu.Lock()
	defer e.nowMu.Unlock()
	e.now = e.now.Add(delta)
	return e.now
}

func (e *hostAPITestEnv) call(t testing.TB, extName string, method string, params any) (any, error) {
	t.Helper()

	eRaw, err := marshalParams(params)
	if err != nil {
		return nil, err
	}
	return e.handler.Handle(testutil.Context(t), extName, method, eRaw)
}

func (e *hostAPITestEnv) callWithContext(t testing.TB, ctx context.Context, extName string, method string, params any) (any, error) {
	t.Helper()

	eRaw, err := marshalParams(params)
	if err != nil {
		return nil, err
	}
	return e.handler.Handle(ctx, extName, method, eRaw)
}

func (e *hostAPITestEnv) bridgeContext(t testing.TB, instance *bridgepkg.BridgeInstance) context.Context {
	t.Helper()

	if instance == nil {
		t.Fatal("bridge instance = nil, want non-nil")
		return testutil.Context(t)
	}

	return withHostAPIBridgeRuntime(testutil.Context(t), &subprocess.InitializeBridgeRuntime{
		Instance: *instance,
	})
}

func (e *hostAPITestEnv) submitPrompt(t testing.TB, extName string, sessionID string, message string) (hostAPISessionPromptResult, error) {
	t.Helper()

	result, err := e.call(t, extName, "sessions/prompt", map[string]string{
		"session_id": sessionID,
		"message":    message,
	})
	if err != nil {
		return hostAPISessionPromptResult{}, err
	}
	var prompt hostAPISessionPromptResult
	decodeResult(t, result, &prompt)
	return prompt, nil
}

func (e *hostAPITestEnv) createSession(t *testing.T) *session.Session {
	t.Helper()

	sess, err := e.sessions.Create(testutil.Context(t), session.CreateOpts{
		AgentName: "coder",
		Workspace: e.workspace.ID,
		Type:      session.SessionTypeSystem,
	})
	if err != nil {
		t.Fatalf("sessions.Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = e.sessions.Stop(testutil.Context(t), sess.ID)
	})
	return sess
}

func (e *hostAPITestEnv) createBridgeInstance(t *testing.T, req bridgepkg.CreateInstanceRequest) *bridgepkg.BridgeInstance {
	t.Helper()

	if req.Scope == "" {
		req.Scope = bridgepkg.ScopeWorkspace
	}
	if req.WorkspaceID == "" && req.Scope == bridgepkg.ScopeWorkspace {
		req.WorkspaceID = e.workspaceID
	}
	if req.Platform == "" {
		req.Platform = "telegram"
	}
	if req.ExtensionName == "" {
		req.ExtensionName = "telegram-adapter"
	}
	if req.DisplayName == "" {
		req.DisplayName = "Telegram Test"
	}
	if !req.Enabled && req.Status == "" {
		req.Enabled = true
	}
	if req.Status == "" {
		req.Status = bridgepkg.BridgeStatusReady
	}

	instance, err := e.bridges.CreateInstance(testutil.Context(t), req)
	if err != nil {
		t.Fatalf("bridges.CreateInstance() error = %v", err)
	}
	return instance
}

func (e *hostAPITestEnv) useSessionsWithoutObserver(t *testing.T) {
	t.Helper()

	sessions, err := session.NewManager(
		session.WithHomePaths(e.homePaths),
		session.WithDriver(e.driver),
		session.WithWorkspaceResolver(e.workspaces),
		session.WithStore(func(ctx context.Context, sessionID string, path string) (session.EventRecorder, error) {
			return storeSessionDB(ctx, sessionID, path)
		}),
		session.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		session.WithNow(func() time.Time { return e.currentTime() }),
		session.WithSessionIDGenerator(sequentialSessionIDGenerator("sess")),
		session.WithTurnIDGenerator(sequentialSessionIDGenerator("turn")),
	)
	if err != nil {
		t.Fatalf("session.NewManager(without observer) error = %v", err)
	}

	taskManager, err := taskpkg.NewManager(
		taskpkg.WithStore(e.registry),
		taskpkg.WithSessionExecutor(&hostAPITestTaskSessionExecutor{
			sessions:            sessions,
			globalWorkspacePath: e.homePaths.HomeDir,
		}),
		taskpkg.WithManagerNow(func() time.Time { return e.currentTime() }),
	)
	if err != nil {
		t.Fatalf("task.NewManager(without observer) error = %v", err)
	}

	e.sessions = sessions
	e.tasks = taskManager
	e.handler = NewHostAPIHandler(
		e.sessions,
		e.memory,
		nil,
		e.skills,
		WithHostAPITaskManager(e.tasks),
		WithHostAPICapabilityChecker(e.checker),
		WithHostAPIWorkspaceResolver(e.workspaces),
		WithHostAPIBridgeRegistry(e.bridges),
		WithHostAPIBridgeDedupStore(e.registry),
		WithHostAPINow(func() time.Time { return e.currentTime() }),
		WithHostAPIBridgeIngressConfig(15*time.Minute, time.Minute),
		WithHostAPIRateLimit(1000, 1000),
	)
}

type hostAPISessionSource struct {
	manager *session.Manager
}

func (s *hostAPISessionSource) List() []*session.SessionInfo {
	if s == nil || s.manager == nil {
		return nil
	}
	return s.manager.List()
}

type hostAPIFakeWorkspaceResolver struct {
	mu       sync.Mutex
	resolved map[string]workspacepkg.ResolvedWorkspace
}

type recordingPromptDeliveryBroker struct {
	mu            sync.Mutex
	registrations []bridgepkg.PromptDeliveryRegistration
	projected     []bridgepkg.DeliveryProjectionEvent
}

func (b *recordingPromptDeliveryBroker) RegisterPromptDelivery(
	_ context.Context,
	reg bridgepkg.PromptDeliveryRegistration,
) (*bridgepkg.DeliverySnapshot, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	cloned := reg
	if len(cloned.SeedEvents) > 0 {
		cloned.SeedEvents = append([]bridgepkg.DeliveryProjectionEvent(nil), cloned.SeedEvents...)
	}
	b.registrations = append(b.registrations, cloned)
	return &bridgepkg.DeliverySnapshot{
		DeliveryID:       "del-test",
		SessionID:        reg.SessionID,
		TurnID:           reg.TurnID,
		BridgeInstanceID: reg.RoutingKey.BridgeInstanceID,
		RoutingKey:       reg.RoutingKey,
		DeliveryTarget:   reg.DeliveryTarget,
		LatestEventType:  bridgepkg.DeliveryEventTypeStart,
		UpdatedAt:        time.Now().UTC(),
	}, nil
}

func (b *recordingPromptDeliveryBroker) ProjectEvent(
	_ context.Context,
	_ string,
	event bridgepkg.DeliveryProjectionEvent,
) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.projected = append(b.projected, event)
	return nil
}

func (b *recordingPromptDeliveryBroker) snapshotRegistrations() []bridgepkg.PromptDeliveryRegistration {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]bridgepkg.PromptDeliveryRegistration, 0, len(b.registrations))
	for _, reg := range b.registrations {
		cloned := reg
		if len(cloned.SeedEvents) > 0 {
			cloned.SeedEvents = append([]bridgepkg.DeliveryProjectionEvent(nil), cloned.SeedEvents...)
		}
		out = append(out, cloned)
	}
	return out
}

func (b *recordingPromptDeliveryBroker) snapshotProjectedEvents() []bridgepkg.DeliveryProjectionEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	out := make([]bridgepkg.DeliveryProjectionEvent, 0, len(b.projected))
	out = append(out, b.projected...)
	return out
}

func newHostAPIFakeWorkspaceResolver(workspace workspacepkg.ResolvedWorkspace) *hostAPIFakeWorkspaceResolver {
	resolver := &hostAPIFakeWorkspaceResolver{
		resolved: make(map[string]workspacepkg.ResolvedWorkspace),
	}
	resolver.upsert(workspace)
	return resolver
}

func (r *hostAPIFakeWorkspaceResolver) Resolve(ctx context.Context, idOrPath string) (workspacepkg.ResolvedWorkspace, error) {
	if err := ctx.Err(); err != nil {
		return workspacepkg.ResolvedWorkspace{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if resolved, ok := r.resolved[strings.TrimSpace(idOrPath)]; ok {
		return cloneResolvedWorkspaceForHostAPITests(resolved), nil
	}
	if resolved, ok := r.resolved[normalizeHostAPIPath(idOrPath)]; ok {
		return cloneResolvedWorkspaceForHostAPITests(resolved), nil
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (r *hostAPIFakeWorkspaceResolver) ResolveOrRegister(ctx context.Context, path string) (workspacepkg.ResolvedWorkspace, error) {
	return r.Resolve(ctx, path)
}

func (r *hostAPIFakeWorkspaceResolver) upsert(workspace workspacepkg.ResolvedWorkspace) {
	cloned := cloneResolvedWorkspaceForHostAPITests(workspace)
	r.resolved[cloned.ID] = cloned
	if name := strings.TrimSpace(cloned.Name); name != "" {
		r.resolved[name] = cloned
	}
	if root := normalizeHostAPIPath(cloned.RootDir); root != "" {
		r.resolved[root] = cloned
	}
}

func cloneResolvedWorkspaceForHostAPITests(src workspacepkg.ResolvedWorkspace) workspacepkg.ResolvedWorkspace {
	dst := src
	dst.AdditionalDirs = append([]string(nil), src.AdditionalDirs...)
	dst.Agents = append([]aghconfig.AgentDef(nil), src.Agents...)
	dst.Skills = append([]workspacepkg.SkillPath(nil), src.Skills...)
	return dst
}

func normalizeHostAPIPath(path string) string {
	target := strings.TrimSpace(path)
	if target == "" {
		return ""
	}
	absPath, err := filepath.Abs(target)
	if err != nil {
		return filepath.Clean(target)
	}
	return filepath.Clean(absPath)
}

type hostAPIFakeDriver struct {
	mu        sync.Mutex
	now       time.Time
	processes map[*session.AgentProcess]*hostAPIFakeProcess
	promptLog []acp.PromptRequest
	prompts   []acp.PromptRequest
	startSeq  atomic.Int64
}

type hostAPIFakeProcess struct {
	done sync.Once
	ch   chan struct{}
}

func newHostAPIFakeDriver(now time.Time) *hostAPIFakeDriver {
	return &hostAPIFakeDriver{
		now:       now,
		processes: make(map[*session.AgentProcess]*hostAPIFakeProcess),
	}
}

func (d *hostAPIFakeDriver) Start(_ context.Context, opts acp.StartOpts) (*session.AgentProcess, error) {
	seq := d.startSeq.Add(1)
	procState := &hostAPIFakeProcess{ch: make(chan struct{})}
	proc := session.NewAgentProcess(session.AgentProcessOptions{
		PID:       int(seq),
		AgentName: opts.AgentName,
		Command:   opts.Command,
		Cwd:       opts.Cwd,
		SessionID: fmt.Sprintf("acp-%d", seq),
		StartedAt: d.now.Add(time.Duration(seq) * time.Millisecond),
		Done:      procState.ch,
		Wait: func() error {
			<-procState.ch
			return nil
		},
	})

	d.mu.Lock()
	d.processes[proc] = procState
	d.mu.Unlock()
	return proc, nil
}

func (d *hostAPIFakeDriver) Prompt(_ context.Context, _ *session.AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	d.mu.Lock()
	d.promptLog = append(d.promptLog, req)
	d.mu.Unlock()

	d.mu.Lock()
	d.prompts = append(d.prompts, req)
	d.mu.Unlock()

	events := make(chan acp.AgentEvent, 2)
	go func() {
		defer close(events)
		events <- acp.AgentEvent{
			Type:      acp.EventTypeAgentMessage,
			TurnID:    req.TurnID,
			Timestamp: time.Now().UTC(),
			Text:      "ack: " + req.Message,
		}
		events <- acp.AgentEvent{
			Type:      acp.EventTypeDone,
			TurnID:    req.TurnID,
			Timestamp: time.Now().UTC(),
		}
	}()
	return events, nil
}

func (d *hostAPIFakeDriver) Cancel(context.Context, *session.AgentProcess) error {
	return nil
}

func (d *hostAPIFakeDriver) Stop(_ context.Context, proc *session.AgentProcess) error {
	d.mu.Lock()
	state := d.processes[proc]
	d.mu.Unlock()
	if state == nil {
		return nil
	}
	state.done.Do(func() { close(state.ch) })
	return nil
}

func (d *hostAPIFakeDriver) promptCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.prompts)
}

func storeSessionDB(ctx context.Context, sessionID string, path string) (session.EventRecorder, error) {
	return sessiondbOpen(ctx, sessionID, path)
}

func sessiondbOpen(ctx context.Context, sessionID string, path string) (session.EventRecorder, error) {
	return sessiondb.OpenSessionDB(ctx, sessionID, path)
}

func sequentialSessionIDGenerator(prefix string) session.IDGenerator {
	var counter atomic.Int64
	return func() string {
		return fmt.Sprintf("%s-%d", prefix, counter.Add(1))
	}
}

func marshalParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

func decodeResult(t testing.TB, result any, target any) {
	t.Helper()

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal(result) error = %v", err)
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		t.Fatalf("json.Unmarshal(result) error = %v", err)
	}
}

func assertCapabilityDenied(t testing.TB, err error, wantMethod string) {
	t.Helper()

	assertRPCErrorCode(t, err, CapabilityDeniedCode)
	data := decodeRPCData(t, err)
	if got := data["method"]; got != wantMethod {
		t.Fatalf("rpc data method = %v, want %q", got, wantMethod)
	}
	required, ok := data["required"].([]any)
	if !ok || len(required) == 0 {
		t.Fatalf("rpc data required = %#v, want non-empty slice", data["required"])
	}
}

func assertRPCErrorCode(t testing.TB, err error, want int) {
	t.Helper()

	var rpcErr *subprocess.RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error type = %T, want *subprocess.RPCError", err)
	}
	if rpcErr.Code != want {
		t.Fatalf("rpc error code = %d, want %d", rpcErr.Code, want)
	}
}

func assertErrorContains(t testing.TB, err error, fragment string) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want containing %q", fragment)
	}
	if strings.Contains(err.Error(), fragment) {
		return
	}

	data := decodeRPCData(t, err)
	if raw, ok := data["error"].(string); ok && strings.Contains(raw, fragment) {
		return
	}
	t.Fatalf("error = %q with data %#v, want containing %q", err.Error(), data, fragment)
}

func decodeRPCData(t testing.TB, err error) map[string]any {
	t.Helper()

	var rpcErr *subprocess.RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatalf("error type = %T, want *subprocess.RPCError", err)
	}

	var data map[string]any
	if len(rpcErr.Data) == 0 {
		return data
	}
	if unmarshalErr := json.Unmarshal(rpcErr.Data, &data); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal(rpcErr.Data) error = %v", unmarshalErr)
	}
	return data
}
