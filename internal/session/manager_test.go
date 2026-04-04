package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
)

func TestCreateOpensStoreRegistersSessionAndActivates(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	session, err := h.manager.Create(testContext(t), CreateOpts{
		AgentName: "coder",
		Name:      "primary",
		Workspace: h.workspace,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), session.ID)
	})

	if got := session.Info().State; got != StateActive {
		t.Fatalf("Create() state = %q, want %q", got, StateActive)
	}
	if got, ok := h.manager.Get(session.ID); !ok || got != session {
		t.Fatalf("Get(%q) = (%v, %v), want created session", session.ID, got, ok)
	}
	if got := h.notifier.createdCount(); got != 1 {
		t.Fatalf("created notifications = %d, want 1", got)
	}
	if meta := readMeta(t, session.MetaPath()); meta.State != string(StateActive) {
		t.Fatalf("meta state = %q, want %q", meta.State, StateActive)
	}
	if got := h.driver.startCalls[0].Cwd; got != h.workspace {
		t.Fatalf("start cwd = %q, want %q", got, h.workspace)
	}
	if got := session.Info().Type; got != SessionTypeUser {
		t.Fatalf("Create() type = %q, want %q", got, SessionTypeUser)
	}
	if meta := readMeta(t, session.MetaPath()); meta.SessionType != string(SessionTypeUser) {
		t.Fatalf("meta session type = %q, want %q", meta.SessionType, SessionTypeUser)
	}
}

func TestStopTransitionsToStoppedAndNotifies(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	if err := h.manager.Stop(testContext(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if _, ok := h.manager.Get(session.ID); ok {
		t.Fatalf("Get(%q) after Stop() = found, want missing", session.ID)
	}
	if got := h.notifier.stoppedCount(); got != 1 {
		t.Fatalf("stopped notifications = %d, want 1", got)
	}
	meta := readMeta(t, session.MetaPath())
	if meta.State != string(StateStopped) {
		t.Fatalf("meta state = %q, want %q", meta.State, StateStopped)
	}

	reopened, err := store.OpenSessionDB(testContext(t), session.ID, session.DBPath())
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	if err := reopened.Close(testContext(t)); err != nil {
		t.Fatalf("Close(reopened) error = %v", err)
	}
}

func TestResumeLoadsMetaAndPassesStoredACPSessionID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	originalACP := session.Info().ACPSessionID

	if err := h.manager.Stop(testContext(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	resumed, err := h.manager.Resume(testContext(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), resumed.ID)
	})

	if got := h.driver.startCalls[1].ResumeSessionID; got != originalACP {
		t.Fatalf("resume start ResumeSessionID = %q, want %q", got, originalACP)
	}
	if got := resumed.Info().ACPSessionID; got != originalACP {
		t.Fatalf("resumed ACPSessionID = %q, want %q", got, originalACP)
	}
	if got := resumed.Info().State; got != StateActive {
		t.Fatalf("resumed state = %q, want %q", got, StateActive)
	}
}

func TestResumeFallbackUpdatesACPSessionID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	if err := h.manager.Stop(testContext(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	h.driver.fallbackOnResume = true
	resumed, err := h.manager.Resume(testContext(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), resumed.ID)
	})

	if got, want := resumed.Info().ACPSessionID, "acp-new-2"; got != want {
		t.Fatalf("resumed ACPSessionID = %q, want %q", got, want)
	}
}

func TestPromptStreamsToRecorderAndNotifier(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), session.ID)
	})

	eventsCh, err := h.manager.Prompt(testContext(t), session.ID, "hello")
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	events := collectEvents(t, eventsCh)
	if len(events) != 2 {
		t.Fatalf("Prompt() events = %d, want 2", len(events))
	}
	if events[0].Type != acp.EventTypeAgentMessage {
		t.Fatalf("first event type = %q, want %q", events[0].Type, acp.EventTypeAgentMessage)
	}
	if events[1].Type != acp.EventTypeDone {
		t.Fatalf("second event type = %q, want %q", events[1].Type, acp.EventTypeDone)
	}

	stored, err := session.recorderHandle().Query(testContext(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(stored) != 2 {
		t.Fatalf("stored events = %d, want 2", len(stored))
	}
	if got := h.notifier.eventCount(session.ID); got != 2 {
		t.Fatalf("notifier events = %d, want 2", got)
	}
}

func TestApprovePermissionRoutesToActiveSession(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), session.ID)
	})

	var (
		gotReq sessionApproveCapture
		called bool
	)
	h.driver.approveHook = func(proc *fakeProcess, req acp.ApproveRequest) error {
		called = true
		gotReq = sessionApproveCapture{
			SessionID: proc.handle.SessionID,
			RequestID: req.RequestID,
			TurnID:    req.TurnID,
			Decision:  req.Decision,
		}
		return nil
	}

	err := h.manager.ApprovePermission(testContext(t), session.ID, acp.ApproveRequest{
		RequestID: "req-1",
		TurnID:    "turn-1",
		Decision:  "allow-once",
	})
	if err != nil {
		t.Fatalf("ApprovePermission() error = %v", err)
	}
	if !called {
		t.Fatal("ApprovePermission() did not reach the active session process")
	}
	if gotReq.RequestID != "req-1" || gotReq.TurnID != "turn-1" || gotReq.Decision != "allow-once" {
		t.Fatalf("approve request = %#v", gotReq)
	}
}

func TestApprovePermissionReturnsNotActiveForStoppedSession(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	if err := h.manager.Stop(testContext(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	err := h.manager.ApprovePermission(testContext(t), session.ID, acp.ApproveRequest{
		RequestID: "req-1",
		Decision:  "allow-once",
	})
	if !errors.Is(err, ErrSessionNotActive) {
		t.Fatalf("ApprovePermission(stopped) error = %v, want ErrSessionNotActive", err)
	}
}

func TestApprovePermissionMapsPendingLookupErrors(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), session.ID)
	})

	testCases := []struct {
		name    string
		hookErr error
		wantErr error
	}{
		{
			name:    "not found",
			hookErr: acp.ErrPendingPermissionNotFound,
			wantErr: ErrPendingPermissionNotFound,
		},
		{
			name:    "conflict",
			hookErr: acp.ErrPendingPermissionConflict,
			wantErr: ErrPendingPermissionConflict,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			h.driver.approveHook = func(*fakeProcess, acp.ApproveRequest) error {
				return tc.hookErr
			}
			err := h.manager.ApprovePermission(testContext(t), session.ID, acp.ApproveRequest{
				RequestID: "req-1",
				Decision:  "allow-once",
			})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ApprovePermission() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestAgentCrashTransitionsToStoppedAndNotifies(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	h.driver.lastProcess().crash(errors.New("boom"), "stderr trace")

	waitForCondition(t, "session stopped after crash", func() bool {
		_, ok := h.manager.Get(session.ID)
		return !ok && h.notifier.stoppedCount() == 1
	})

	meta := readMeta(t, session.MetaPath())
	if meta.State != string(StateStopped) {
		t.Fatalf("meta state = %q, want %q", meta.State, StateStopped)
	}

	reopened, err := store.OpenSessionDB(testContext(t), session.ID, session.DBPath())
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	defer func() {
		_ = reopened.Close(testContext(t))
	}()

	events, err := reopened.Query(testContext(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query(reopened) error = %v", err)
	}
	if !containsEventType(events, acp.EventTypeError) {
		t.Fatalf("stored events missing crash error: %#v", events)
	}
}

func TestStopAndProcessExitFinalizeOnlyOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	proceed := make(chan struct{})
	h.driver.stopHook = func(proc *fakeProcess) error {
		proc.crash(errors.New("boom"), "stderr trace")
		<-proceed
		return nil
	}

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- h.manager.Stop(testContext(t), session.ID)
	}()

	waitForCondition(t, "stop notification", func() bool {
		return h.notifier.stoppedCount() == 1
	})
	close(proceed)

	if err := <-stopDone; err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if got := h.notifier.stoppedCount(); got != 1 {
		t.Fatalf("stopped notifications = %d, want 1", got)
	}

	reopened, err := store.OpenSessionDB(testContext(t), session.ID, session.DBPath())
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	defer func() {
		_ = reopened.Close(testContext(t))
	}()

	events, err := reopened.Query(testContext(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query(reopened) error = %v", err)
	}
	if got := countEventType(events, EventTypeSessionStopped); got != 1 {
		t.Fatalf("countEventType(session_stopped) = %d, want 1", got)
	}
}

func TestPromptSerializesSetupAgainstConcurrentStop(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	promptEntered := make(chan struct{})
	releasePrompt := make(chan struct{})
	h.driver.promptHook = func(proc *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
		close(promptEntered)
		<-releasePrompt
		events := make(chan acp.AgentEvent)
		close(events)
		return events, nil
	}

	promptDone := make(chan error, 1)
	go func() {
		eventsCh, err := h.manager.Prompt(testContext(t), session.ID, "hello")
		if err != nil {
			promptDone <- err
			return
		}
		for range eventsCh {
		}
		promptDone <- nil
	}()

	<-promptEntered

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- h.manager.Stop(testContext(t), session.ID)
	}()

	select {
	case err := <-stopDone:
		t.Fatalf("Stop() returned before prompt setup finished: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(releasePrompt)

	if err := <-promptDone; err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	if err := <-stopDone; err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestNormalizeEventSetsTimestampOnlyWhenZero(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	now := h.manager.now()

	normalized := h.manager.normalizeEvent(session, "turn-1", acp.AgentEvent{})
	if normalized.Timestamp.IsZero() {
		t.Fatal("normalizeEvent() left zero timestamp")
	}

	explicit := time.Date(2026, 4, 3, 14, 0, 0, 0, time.UTC)
	preserved := h.manager.normalizeEvent(session, "turn-1", acp.AgentEvent{Timestamp: explicit})
	if !preserved.Timestamp.Equal(explicit) {
		t.Fatalf("normalizeEvent() timestamp = %v, want %v", preserved.Timestamp, explicit)
	}
	if normalized.Timestamp.Before(now) {
		t.Fatalf("normalizeEvent() timestamp = %v, want >= %v", normalized.Timestamp, now)
	}
}

func TestListAndGet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := createSession(t, h)
	second := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), first.ID)
		_ = h.manager.Stop(testContext(t), second.ID)
	})

	list := h.manager.List()
	if len(list) != 2 {
		t.Fatalf("List() = %d sessions, want 2", len(list))
	}
	if list[0].ID != first.ID || list[1].ID != second.ID {
		t.Fatalf("List() ids = [%s %s], want [%s %s]", list[0].ID, list[1].ID, first.ID, second.ID)
	}
	if _, ok := h.manager.Get("missing"); ok {
		t.Fatal("Get(missing) = found, want missing")
	}
}

func TestConcurrentCreateStopGet(t *testing.T) {
	h := newHarness(t, WithMaxSessions(32))

	done := make(chan struct{})
	var readers sync.WaitGroup
	readers.Add(1)
	go func() {
		defer readers.Done()
		for {
			select {
			case <-done:
				return
			default:
				_ = h.manager.List()
				for _, info := range h.manager.List() {
					h.manager.Get(info.ID)
				}
			}
		}
	}()

	const total = 8
	var workers sync.WaitGroup
	for i := 0; i < total; i++ {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()

			session, err := h.manager.Create(testContext(t), CreateOpts{
				AgentName: "coder",
				Name:      fmt.Sprintf("session-%d", index),
				Workspace: h.workspace,
			})
			if err != nil {
				t.Errorf("Create(%d) error = %v", index, err)
				return
			}
			if _, ok := h.manager.Get(session.ID); !ok {
				t.Errorf("Get(%q) = missing after Create()", session.ID)
			}
			if err := h.manager.Stop(testContext(t), session.ID); err != nil {
				t.Errorf("Stop(%q) error = %v", session.ID, err)
			}
		}(i)
	}

	workers.Wait()
	close(done)
	readers.Wait()

	if list := h.manager.List(); len(list) != 0 {
		t.Fatalf("List() after concurrent stop = %d, want 0", len(list))
	}
}

func TestCreateEnforcesMaxSessions(t *testing.T) {
	t.Parallel()

	h := newHarness(t, WithMaxSessions(1))
	first := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), first.ID)
	})

	_, err := h.manager.Create(testContext(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspace,
	})
	if err == nil {
		t.Fatal("Create(second) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrMaxSessionsReached) {
		t.Fatalf("Create(second) error = %v, want ErrMaxSessionsReached", err)
	}
}

func TestCreatePassesMergedMCPServers(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.cfg.Providers["claude"] = aghconfig.ProviderConfig{
		Command: "provider-command",
		MCPServers: []aghconfig.MCPServer{
			{Name: "base", Command: "base-command", Args: []string{"--base"}},
			{Name: "override", Command: "provider-override"},
		},
	}
	h.manager = newManagerWithHarness(t, h,
		WithConfig(h.cfg),
		WithAgentLoader(staticAgentLoader(aghconfig.AgentDef{
			Provider: "claude",
			Prompt:   "You are helpful.",
			MCPServers: []aghconfig.MCPServer{
				{Name: "override", Command: "agent-override", Args: []string{"--agent"}},
				{Name: "extra", Command: "extra-command"},
			},
		})),
	)

	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), session.ID)
	})

	got := h.driver.startCalls[0].MCPServers
	if len(got) != 3 {
		t.Fatalf("start MCPServers = %#v, want 3 entries", got)
	}
	if got[0].Name != "base" || got[0].Command != "base-command" {
		t.Fatalf("base MCP server = %#v", got[0])
	}
	if got[1].Name != "override" || got[1].Command != "agent-override" {
		t.Fatalf("override MCP server = %#v", got[1])
	}
	if got[2].Name != "extra" || got[2].Command != "extra-command" {
		t.Fatalf("extra MCP server = %#v", got[2])
	}
}

func TestCreateInvokesPromptAssemblerWhenConfigured(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	var (
		called         bool
		gotWorkspace   string
		gotAgentName   string
		gotAgentPrompt string
	)
	h.manager = newManagerWithHarness(t, h, WithPromptAssembler(promptAssemblerFunc(func(_ context.Context, agent aghconfig.AgentDef, workspace string) (string, error) {
		called = true
		gotWorkspace = workspace
		gotAgentName = agent.Name
		gotAgentPrompt = agent.Prompt
		return agent.Prompt + "\n\nmemory block", nil
	})))

	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), session.ID)
	})

	if !called {
		t.Fatal("Create() did not invoke the configured prompt assembler")
	}
	if gotWorkspace != h.workspace {
		t.Fatalf("assembler workspace = %q, want %q", gotWorkspace, h.workspace)
	}
	if gotAgentName != "coder" {
		t.Fatalf("assembler agent name = %q, want %q", gotAgentName, "coder")
	}
	if gotAgentPrompt != "You are a coding assistant." {
		t.Fatalf("assembler prompt = %q, want original agent prompt", gotAgentPrompt)
	}
}

func TestACPDriverAdapterErrorPaths(t *testing.T) {
	t.Parallel()

	adapter := NewACPDriverAdapter(acp.New())
	if _, err := adapter.Prompt(testContext(t), &AgentProcess{}, acp.PromptRequest{}); err == nil {
		t.Fatal("Prompt(unsupported process) error = nil, want non-nil")
	}
	if err := adapter.Stop(testContext(t), &AgentProcess{}); err == nil {
		t.Fatal("Stop(unsupported process) error = nil, want non-nil")
	}
}

type harness struct {
	manager   *Manager
	driver    *fakeDriver
	notifier  *fakeNotifier
	cfg       aghconfig.Config
	homePaths aghconfig.HomePaths
	workspace string
}

func newHarness(t *testing.T, extraOpts ...Option) *harness {
	t.Helper()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}
	if err := aghconfig.EnsureHomeLayout(homePaths); err != nil {
		t.Fatalf("EnsureHomeLayout() error = %v", err)
	}

	workspace := filepath.Join(homePaths.HomeDir, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatalf("MkdirAll(workspace) error = %v", err)
	}

	h := &harness{
		driver:    newFakeDriver(),
		notifier:  newFakeNotifier(),
		cfg:       aghconfig.DefaultWithHome(homePaths),
		homePaths: homePaths,
		workspace: workspace,
	}
	h.manager = newManagerWithHarness(t, h, extraOpts...)
	return h
}

func newManagerWithHarness(t *testing.T, h *harness, extraOpts ...Option) *Manager {
	t.Helper()

	opts := []Option{
		WithHomePaths(h.homePaths),
		WithDriver(h.driver),
		WithNotifier(h.notifier),
		WithConfig(h.cfg),
		WithAgentLoader(staticAgentLoader(aghconfig.AgentDef{
			Provider: "claude",
			Prompt:   "You are a coding assistant.",
		})),
		WithStore(func(ctx context.Context, sessionID string, path string) (EventRecorder, error) {
			return store.OpenSessionDB(ctx, sessionID, path)
		}),
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		WithSessionIDGenerator(sequentialIDGenerator("sess")),
		WithTurnIDGenerator(sequentialIDGenerator("turn")),
	}
	opts = append(opts, extraOpts...)

	manager, err := NewManager(opts...)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	return manager
}

func createSession(t *testing.T, h *harness) *Session {
	t.Helper()

	session, err := h.manager.Create(testContext(t), CreateOpts{
		AgentName: "coder",
		Name:      "session",
		Workspace: h.workspace,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return session
}

func readMeta(t *testing.T, path string) store.SessionMeta {
	t.Helper()

	meta, err := store.ReadSessionMeta(path)
	if err != nil {
		t.Fatalf("ReadSessionMeta(%q) error = %v", path, err)
	}
	return meta
}

func collectEvents(t *testing.T, eventsCh <-chan acp.AgentEvent) []acp.AgentEvent {
	t.Helper()

	events := make([]acp.AgentEvent, 0, 4)
	for event := range eventsCh {
		events = append(events, event)
	}
	return events
}

func containsEventType(events []store.SessionEvent, want string) bool {
	for _, event := range events {
		if event.Type == want {
			return true
		}
	}
	return false
}

func countEventType(events []store.SessionEvent, want string) int {
	count := 0
	for _, event := range events {
		if event.Type == want {
			count++
		}
	}
	return count
}

func waitForCondition(t *testing.T, label string, fn func() bool) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", label)
}

func testContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func sequentialIDGenerator(prefix string) IDGenerator {
	var counter atomic.Int64
	return func() string {
		return fmt.Sprintf("%s-%d", prefix, counter.Add(1))
	}
}

func staticAgentLoader(agent aghconfig.AgentDef) AgentLoader {
	return func(name string, _ aghconfig.HomePaths) (aghconfig.AgentDef, error) {
		copied := agent
		copied.Name = name
		if copied.Provider == "" {
			copied.Provider = "claude"
		}
		if copied.Prompt == "" {
			copied.Prompt = "You are helpful."
		}
		return copied, nil
	}
}

type promptAssemblerFunc func(context.Context, aghconfig.AgentDef, string) (string, error)

func (fn promptAssemblerFunc) Assemble(ctx context.Context, agent aghconfig.AgentDef, workspace string) (string, error) {
	return fn(ctx, agent, workspace)
}

type fakeNotifier struct {
	mu      sync.Mutex
	created []*SessionInfo
	stopped []*SessionInfo
	events  map[string][]acp.AgentEvent
}

func newFakeNotifier() *fakeNotifier {
	return &fakeNotifier{
		events: make(map[string][]acp.AgentEvent),
	}
}

func (n *fakeNotifier) OnSessionCreated(_ context.Context, session *Session) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.created = append(n.created, session.Info())
}

func (n *fakeNotifier) OnSessionStopped(_ context.Context, session *Session) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.stopped = append(n.stopped, session.Info())
}

func (n *fakeNotifier) OnAgentEvent(_ context.Context, sessionID string, event acp.AgentEvent) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events[sessionID] = append(n.events[sessionID], event)
}

func (n *fakeNotifier) createdCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.created)
}

func (n *fakeNotifier) stoppedCount() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.stopped)
}

func (n *fakeNotifier) eventCount(sessionID string) int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.events[sessionID])
}

type fakeDriver struct {
	mu               sync.Mutex
	startCalls       []acp.StartOpts
	promptCalls      []acp.PromptRequest
	stopCalls        int
	cancelCalls      int
	processes        map[*AgentProcess]*fakeProcess
	lastProc         *fakeProcess
	promptHook       func(proc *fakeProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error)
	approveHook      func(proc *fakeProcess, req acp.ApproveRequest) error
	stopHook         func(proc *fakeProcess) error
	startHook        func(opts acp.StartOpts, sequence int) (*fakeProcess, error)
	fallbackOnResume bool
}

func newFakeDriver() *fakeDriver {
	return &fakeDriver{
		processes: make(map[*AgentProcess]*fakeProcess),
	}
}

func (d *fakeDriver) Start(_ context.Context, opts acp.StartOpts) (*AgentProcess, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	copied := opts
	copied.Env = append([]string(nil), opts.Env...)
	copied.MCPServers = append([]aghconfig.MCPServer(nil), opts.MCPServers...)
	d.startCalls = append(d.startCalls, copied)

	sequence := len(d.startCalls)
	var proc *fakeProcess
	var err error
	if d.startHook != nil {
		proc, err = d.startHook(copied, sequence)
	} else {
		sessionID := fmt.Sprintf("acp-%d", sequence)
		if copied.ResumeSessionID != "" {
			if d.fallbackOnResume {
				sessionID = fmt.Sprintf("acp-new-%d", sequence)
			} else {
				sessionID = copied.ResumeSessionID
			}
		}
		proc = newFakeProcess(copied.AgentName, copied.Command, copied.Cwd, sessionID)
	}
	if err != nil {
		return nil, err
	}

	proc.handle.approvePermissionFn = func(ctx context.Context, req acp.ApproveRequest) error {
		if err := ctx.Err(); err != nil {
			return err
		}

		d.mu.Lock()
		hook := d.approveHook
		d.mu.Unlock()

		if hook != nil {
			return hook(proc, req)
		}
		return nil
	}

	d.processes[proc.handle] = proc
	d.lastProc = proc
	return proc.handle, nil
}

func (d *fakeDriver) Prompt(_ context.Context, proc *AgentProcess, req acp.PromptRequest) (<-chan acp.AgentEvent, error) {
	d.mu.Lock()
	fakeProc := d.processes[proc]
	d.promptCalls = append(d.promptCalls, req)
	hook := d.promptHook
	d.mu.Unlock()

	if fakeProc == nil {
		return nil, errors.New("test: unknown fake process")
	}
	if hook != nil {
		return hook(fakeProc, req)
	}

	totalTokens := int64(9)
	events := make(chan acp.AgentEvent, 2)
	go func() {
		defer close(events)
		ts := time.Now().UTC()
		events <- acp.AgentEvent{
			Type:      acp.EventTypeAgentMessage,
			SessionID: fakeProc.handle.SessionID,
			TurnID:    req.TurnID,
			Timestamp: ts,
			Text:      "reply",
		}
		events <- acp.AgentEvent{
			Type:       acp.EventTypeDone,
			SessionID:  fakeProc.handle.SessionID,
			TurnID:     req.TurnID,
			Timestamp:  ts,
			StopReason: "end_turn",
			Usage: &acp.TokenUsage{
				TurnID:      req.TurnID,
				TotalTokens: &totalTokens,
				Timestamp:   ts,
			},
		}
	}()
	return events, nil
}

func (d *fakeDriver) Cancel(_ context.Context, _ *AgentProcess) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cancelCalls++
	return nil
}

func (d *fakeDriver) Stop(_ context.Context, proc *AgentProcess) error {
	d.mu.Lock()
	fakeProc := d.processes[proc]
	d.stopCalls++
	hook := d.stopHook
	d.mu.Unlock()

	if fakeProc == nil {
		return errors.New("test: unknown fake process")
	}
	if hook != nil {
		return hook(fakeProc)
	}
	fakeProc.exit()
	return nil
}

func (d *fakeDriver) lastProcess() *fakeProcess {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastProc
}

type fakeProcess struct {
	mu      sync.Mutex
	done    chan struct{}
	closed  bool
	waitErr error
	stderr  string
	handle  *AgentProcess
}

type sessionApproveCapture struct {
	SessionID string
	RequestID string
	TurnID    string
	Decision  string
}

func newFakeProcess(agentName string, command string, cwd string, sessionID string) *fakeProcess {
	proc := &fakeProcess{
		done: make(chan struct{}),
	}
	proc.handle = &AgentProcess{
		PID:       1,
		AgentName: agentName,
		Command:   command,
		Cwd:       cwd,
		SessionID: sessionID,
		Caps: acp.ACPCaps{
			SupportsLoadSession: true,
			SupportedModes:      []string{"chat"},
			SupportedModels:     []string{"gpt-4o"},
		},
		StartedAt: time.Now().UTC(),
		done:      proc.done,
		waitFn:    proc.wait,
		stderrFn:  proc.stderrOutput,
	}
	return proc
}

func (p *fakeProcess) wait() error {
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.waitErr
}

func (p *fakeProcess) stderrOutput() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.stderr
}

func (p *fakeProcess) exit() {
	p.finish(nil, "")
}

func (p *fakeProcess) crash(err error, stderr string) {
	p.finish(err, stderr)
}

func (p *fakeProcess) finish(err error, stderr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.waitErr = err
	p.stderr = stderr
	if !p.closed {
		p.closed = true
		close(p.done)
	}
}
