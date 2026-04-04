package session

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/store"
)

func TestCreateCleansUpOnStartFailure(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	recorder := &stubRecorder{}
	h.driver.startHook = func(opts acp.StartOpts, sequence int) (*fakeProcess, error) {
		return nil, errors.New("start failed")
	}
	h.manager = newManagerWithHarness(t, h,
		WithStore(func(context.Context, string, string) (EventRecorder, error) {
			return recorder, nil
		}),
	)

	_, err := h.manager.Create(testContext(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspace,
	})
	if err == nil {
		t.Fatal("Create() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "start failed") {
		t.Fatalf("Create() error = %v, want start failure", err)
	}
	if recorder.closeCalls != 1 {
		t.Fatalf("recorder close calls = %d, want 1", recorder.closeCalls)
	}
	if _, statErr := os.Stat(filepath.Join(h.homePaths.SessionsDir, "sess-1")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("session directory still exists after failed create: %v", statErr)
	}
	if len(h.manager.List()) != 0 {
		t.Fatalf("List() after failed create = %d sessions, want 0", len(h.manager.List()))
	}
}

func TestCreateErrorBranches(t *testing.T) {
	t.Parallel()

	t.Run("blank agent name", func(t *testing.T) {
		h := newHarness(t)
		if _, err := h.manager.Create(testContext(t), CreateOpts{Workspace: h.workspace}); err == nil {
			t.Fatal("Create(blank agent) error = nil, want non-nil")
		}
	})

	t.Run("empty generated session id", func(t *testing.T) {
		h := newHarness(t, WithSessionIDGenerator(func() string { return "" }))
		if _, err := h.manager.Create(testContext(t), CreateOpts{
			AgentName: "coder",
			Workspace: h.workspace,
		}); err == nil {
			t.Fatal("Create(empty session id) error = nil, want non-nil")
		}
	})

	t.Run("store open failure", func(t *testing.T) {
		h := newHarness(t)
		h.manager = newManagerWithHarness(t, h, WithStore(func(context.Context, string, string) (EventRecorder, error) {
			return nil, errors.New("open failed")
		}))
		if _, err := h.manager.Create(testContext(t), CreateOpts{
			AgentName: "coder",
			Workspace: h.workspace,
		}); err == nil {
			t.Fatal("Create(store open failure) error = nil, want non-nil")
		}
	})
}

func TestCreateWithNilPromptAssemblerIsSafe(t *testing.T) {
	t.Parallel()

	h := newHarness(t, WithPromptAssembler(nil))

	session, err := h.manager.Create(testContext(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspace,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), session.ID)
	})

	if got := session.Info().Type; got != SessionTypeUser {
		t.Fatalf("session type = %q, want %q", got, SessionTypeUser)
	}
	if got := h.driver.startCalls[0].SystemPrompt; got != "You are a coding assistant." {
		t.Fatalf("start system prompt = %q, want raw agent prompt", got)
	}
}

func TestResumeCleansUpOnStartFailure(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	if err := h.manager.Stop(testContext(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	recorder := &stubRecorder{}
	h.driver.startHook = func(opts acp.StartOpts, sequence int) (*fakeProcess, error) {
		return nil, errors.New("resume failed")
	}
	h.manager = newManagerWithHarness(t, h,
		WithStore(func(context.Context, string, string) (EventRecorder, error) {
			return recorder, nil
		}),
	)

	_, err := h.manager.Resume(testContext(t), session.ID)
	if err == nil {
		t.Fatal("Resume() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "resume failed") {
		t.Fatalf("Resume() error = %v, want resume failure", err)
	}
	if recorder.closeCalls != 1 {
		t.Fatalf("recorder close calls = %d, want 1", recorder.closeCalls)
	}
}

func TestResumeErrorBranches(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	if _, err := h.manager.Resume(testContext(t), ""); err == nil {
		t.Fatal("Resume(blank id) error = nil, want non-nil")
	}
	if _, err := h.manager.Resume(testContext(t), "missing"); err == nil {
		t.Fatal("Resume(missing meta) error = nil, want non-nil")
	}
}

func TestPromptErrorPaths(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	if _, err := h.manager.Prompt(testContext(t), session.ID, "   "); err == nil {
		t.Fatal("Prompt(empty) error = nil, want non-nil")
	}
	if _, err := h.manager.Prompt(testContext(t), "missing", "hello"); err == nil {
		t.Fatal("Prompt(missing) error = nil, want non-nil")
	}
	if err := h.manager.Stop(testContext(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if _, err := h.manager.Prompt(testContext(t), session.ID, "after-stop"); err == nil {
		t.Fatal("Prompt(stopped) error = nil, want non-nil")
	}

	h = newHarness(t)
	session = createSession(t, h)
	session.clearProcess(time.Now().UTC())
	if _, err := h.manager.Prompt(testContext(t), session.ID, "missing-process"); err == nil {
		t.Fatal("Prompt(missing process) error = nil, want non-nil")
	}
}

func TestResumeReturnsExistingActiveSession(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), session.ID)
	})

	resumed, err := h.manager.Resume(testContext(t), session.ID)
	if err != nil {
		t.Fatalf("Resume(active) error = %v", err)
	}
	if resumed != session {
		t.Fatalf("Resume(active) returned %p, want %p", resumed, session)
	}
}

func TestNewManagerOptionsAndValidation(t *testing.T) {
	t.Parallel()

	homePaths, err := aghconfig.ResolveHomePathsFrom(t.TempDir())
	if err != nil {
		t.Fatalf("ResolveHomePathsFrom() error = %v", err)
	}

	now := time.Date(2026, 4, 3, 15, 0, 0, 0, time.UTC)
	cfg := aghconfig.DefaultWithHome(homePaths)
	manager, err := NewManager(
		WithHomePaths(homePaths),
		WithDriver(newFakeDriver()),
		WithNotifier(newFakeNotifier()),
		WithConfigLoader(func(workspace string) (aghconfig.Config, error) {
			if workspace != "/tmp/workspace" {
				t.Fatalf("workspace = %q, want /tmp/workspace", workspace)
			}
			return cfg, nil
		}),
		WithNow(func() time.Time { return now }),
		WithPromptBufferSize(7),
	)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}
	if got := manager.now(); !got.Equal(now) {
		t.Fatalf("now() = %s, want %s", got, now)
	}
	if got := manager.promptBufSize; got != 7 {
		t.Fatalf("promptBufSize = %d, want 7", got)
	}
	if _, err := manager.loadConfig("/tmp/workspace"); err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	defaultManager, err := NewManager(WithHomePaths(homePaths))
	if err != nil {
		t.Fatalf("NewManager(defaults) error = %v", err)
	}
	if defaultManager.driver == nil {
		t.Fatal("defaultManager.driver = nil, want default ACP adapter")
	}

	_, err = NewManager(
		WithHomePaths(homePaths),
		WithDriver(nil),
	)
	if err == nil {
		t.Fatal("NewManager(nil driver) error = nil, want non-nil")
	}

	manager, err = NewManager(
		WithHomePaths(homePaths),
		WithDriver(newFakeDriver()),
		WithLogger(nil),
		WithNotifier(nil),
		WithConfigLoader(func(string) (aghconfig.Config, error) { return cfg, nil }),
		WithAgentLoader(staticAgentLoader(aghconfig.AgentDef{Provider: "claude", Prompt: "hi"})),
		WithStore(func(context.Context, string, string) (EventRecorder, error) { return &stubRecorder{}, nil }),
		WithNow(nil),
		WithSessionIDGenerator(nil),
		WithTurnIDGenerator(nil),
		WithPromptBufferSize(0),
	)
	if err != nil {
		t.Fatalf("NewManager(normalized defaults) error = %v", err)
	}
	if manager.logger == nil || manager.now == nil || manager.newSessionID == nil || manager.newTurnID == nil {
		t.Fatal("NewManager() failed to restore default dependencies after nil overrides")
	}
	if manager.notifier != nil {
		t.Fatal("NewManager() restored a notifier default, want nil")
	}
	if manager.promptBufSize != defaultPromptBufferSize {
		t.Fatalf("promptBufSize = %d, want default %d", manager.promptBufSize, defaultPromptBufferSize)
	}

	testCases := []struct {
		name string
		opts []Option
	}{
		{
			name: "missing config loader",
			opts: []Option{
				WithHomePaths(homePaths),
				WithDriver(newFakeDriver()),
				WithConfigLoader(nil),
			},
		},
		{
			name: "missing agent loader",
			opts: []Option{
				WithHomePaths(homePaths),
				WithDriver(newFakeDriver()),
				WithConfig(cfg),
				WithAgentLoader(nil),
			},
		},
		{
			name: "missing store opener",
			opts: []Option{
				WithHomePaths(homePaths),
				WithDriver(newFakeDriver()),
				WithConfig(cfg),
				WithAgentLoader(staticAgentLoader(aghconfig.AgentDef{Provider: "claude", Prompt: "hi"})),
				WithStore(nil),
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewManager(tc.opts...); err == nil {
				t.Fatalf("NewManager(%s) error = nil, want non-nil", tc.name)
			}
		})
	}
}

func TestHelperFunctionsAndUtilities(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	got, err := resolveWorkspace(dir)
	if err != nil {
		t.Fatalf("resolveWorkspace(dir) error = %v", err)
	}
	if got != dir {
		t.Fatalf("resolveWorkspace(dir) = %q, want %q", got, dir)
	}

	filePath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := resolveWorkspace(filePath); err == nil {
		t.Fatal("resolveWorkspace(file) error = nil, want non-nil")
	}

	done := make(chan struct{})
	close(done)
	if !isProcessDone(&AgentProcess{done: done}) {
		t.Fatal("isProcessDone(closed) = false, want true")
	}
	if !isProcessDone(nil) {
		t.Fatal("isProcessDone(nil) = false, want true")
	}

	if got := derefString(nil); got != "" {
		t.Fatalf("derefString(nil) = %q, want empty", got)
	}
	if got := *stringPointer("value"); got != "value" {
		t.Fatalf("stringPointer(value) = %q, want value", got)
	}
	if stringPointer("   ") != nil {
		t.Fatal("stringPointer(blank) = non-nil, want nil")
	}
	if got := newID("sess"); !strings.HasPrefix(got, "sess-") {
		t.Fatalf("newID(sess) = %q, want prefixed id", got)
	}
	if got := newID(""); got == "" {
		t.Fatal("newID(\"\") = empty, want non-empty")
	}

	if got := (maxSessionsReachedError{active: 1, limit: 2}).Error(); !strings.Contains(got, "1/2") {
		t.Fatalf("maxSessionsReachedError.Error() = %q", got)
	}
}

func TestCreateWithBlankWorkspaceUsesCurrentDir(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session, err := h.manager.Create(testContext(t), CreateOpts{
		AgentName: "coder",
		Workspace: "",
	})
	if err != nil {
		t.Fatalf("Create(blank workspace) error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testContext(t), session.ID)
	})

	if got, err := os.Getwd(); err != nil {
		t.Fatalf("Getwd() error = %v", err)
	} else if session.Info().Workspace != got {
		t.Fatalf("session workspace = %q, want %q", session.Info().Workspace, got)
	}
}

func TestMarshalAgentEvent(t *testing.T) {
	t.Parallel()

	rawPayload := json.RawMessage(`{"raw":true}`)
	raw, err := marshalAgentEvent(acp.AgentEvent{Raw: rawPayload})
	if err != nil {
		t.Fatalf("marshalAgentEvent(raw) error = %v", err)
	}
	if raw != string(rawPayload) {
		t.Fatalf("marshalAgentEvent(raw) = %q, want %q", raw, string(rawPayload))
	}

	totalTokens := int64(4)
	payload, err := marshalAgentEvent(acp.AgentEvent{
		Type:      acp.EventTypeDone,
		SessionID: "acp-1",
		TurnID:    "turn-1",
		Timestamp: time.Now().UTC(),
		Text:      "done",
		Error:     "none",
		Usage: &acp.TokenUsage{
			TurnID:      "turn-1",
			TotalTokens: &totalTokens,
			Timestamp:   time.Now().UTC(),
		},
	})
	if err != nil {
		t.Fatalf("marshalAgentEvent(structured) error = %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}
	if decoded["type"] != acp.EventTypeDone {
		t.Fatalf("decoded[type] = %v, want %q", decoded["type"], acp.EventTypeDone)
	}
	if decoded["text"] != "done" {
		t.Fatalf("decoded[text] = %v, want %q", decoded["text"], "done")
	}
}

func TestAgentProcessHelpersAndAdapterUtilities(t *testing.T) {
	t.Parallel()

	wrapped := wrapACPProcess(&acp.AgentProcess{
		PID:       99,
		AgentName: "coder",
		Command:   "fake",
		Cwd:       "/tmp",
		SessionID: "acp-1",
		Caps:      acp.ACPCaps{SupportsLoadSession: true},
		StartedAt: time.Now().UTC(),
	})
	if wrapped == nil || wrapped.PID != 99 || wrapped.SessionID != "acp-1" {
		t.Fatalf("wrapACPProcess() = %#v", wrapped)
	}
	if wrapACPProcess(nil) != nil {
		t.Fatal("wrapACPProcess(nil) = non-nil, want nil")
	}

	manager := &Manager{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if got := manager.sessionLogger(nil); got == nil {
		t.Fatal("sessionLogger(nil) = nil, want logger")
	}
	if got := manager.sessionLogger(&Session{ID: "sess-1", AgentName: "coder"}); got == nil {
		t.Fatal("sessionLogger(session) = nil, want logger")
	}

	adapter := NewACPDriverAdapter(acp.New())
	if _, err := adapter.nativeProcess(&AgentProcess{}); err == nil {
		t.Fatal("nativeProcess(unsupported process) error = nil, want non-nil")
	}

	if _, err := adapter.Start(context.Background(), acp.StartOpts{}); err == nil {
		t.Fatal("Start(invalid opts) error = nil, want non-nil")
	}
	if err := adapter.Cancel(context.Background(), wrapACPProcess(&acp.AgentProcess{})); err == nil {
		t.Fatal("Cancel(empty session id) error = nil, want non-nil")
	}
}

func TestSessionDirAccessorAndStopWithoutProcess(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	if session.SessionDir() == "" {
		t.Fatal("SessionDir() = empty, want path")
	}

	session.clearProcess(time.Now().UTC())
	if err := h.manager.Stop(testContext(t), session.ID); err != nil {
		t.Fatalf("Stop(no process) error = %v", err)
	}
	if readMeta(t, session.MetaPath()).State != string(StateStopped) {
		t.Fatalf("meta state after no-process stop = %q, want %q", readMeta(t, session.MetaPath()).State, StateStopped)
	}
}

func TestTransitionToSameStateOnlyUpdatesTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	session := &Session{
		ID:        "sess-1",
		AgentName: "coder",
		Workspace: t.TempDir(),
		State:     StateActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	later := now.Add(time.Second)
	if err := session.transition(StateActive, later); err != nil {
		t.Fatalf("transition(same state) error = %v", err)
	}
	if got := session.Info().UpdatedAt; !got.Equal(later) {
		t.Fatalf("UpdatedAt = %s, want %s", got, later)
	}
}

type stubRecorder struct {
	closeCalls int
}

func (s *stubRecorder) Record(context.Context, store.SessionEvent) error {
	return nil
}

func (s *stubRecorder) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return nil
}

func (s *stubRecorder) Query(context.Context, store.EventQuery) ([]store.SessionEvent, error) {
	return nil, nil
}

func (s *stubRecorder) History(context.Context, store.EventQuery) ([]store.TurnHistory, error) {
	return nil, nil
}

func (s *stubRecorder) Close(context.Context) error {
	s.closeCalls++
	return nil
}
