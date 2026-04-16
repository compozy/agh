package session

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

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/pedronauck/agh/internal/acp"
	aghconfig "github.com/pedronauck/agh/internal/config"
	hookspkg "github.com/pedronauck/agh/internal/hooks"
	skillspkg "github.com/pedronauck/agh/internal/skills"
	"github.com/pedronauck/agh/internal/skills/bundled"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/sessiondb"
	"github.com/pedronauck/agh/internal/testutil"
	workspacepkg "github.com/pedronauck/agh/internal/workspace"
)

func TestCreateOpensStoreRegistersSessionAndActivates(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "primary",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
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
	if got := session.Info().WorkspaceID; got != h.workspaceID {
		t.Fatalf("session workspace id = %q, want %q", got, h.workspaceID)
	}
	if meta := readMeta(t, session.MetaPath()); meta.WorkspaceID != h.workspaceID {
		t.Fatalf("meta workspace id = %q, want %q", meta.WorkspaceID, h.workspaceID)
	}
	if got := h.driver.startCalls[0].Cwd; got != h.workspace {
		t.Fatalf("start cwd = %q, want %q", got, h.workspace)
	}
	if got := session.Info().Type; got != SessionTypeUser {
		t.Fatalf("Create() type = %q, want %q", got, SessionTypeUser)
	}
	if got := session.Info().Channel; got != "" {
		t.Fatalf("Create() channel = %q, want empty", got)
	}
	if meta := readMeta(t, session.MetaPath()); meta.SessionType != string(SessionTypeUser) {
		t.Fatalf("meta session type = %q, want %q", meta.SessionType, SessionTypeUser)
	}
	if meta := readMeta(t, session.MetaPath()); meta.Channel != "" {
		t.Fatalf("meta channel = %q, want empty", meta.Channel)
	}
	if got := len(h.resolver.resolveCalls); got != 1 {
		t.Fatalf("resolver Resolve() calls = %d, want 1", got)
	}
	if got := h.resolver.resolveCalls[0]; got != h.workspaceID {
		t.Fatalf("resolver Resolve() arg = %q, want %q", got, h.workspaceID)
	}
	if got := len(h.resolver.resolveOrRegisterCalls); got != 0 {
		t.Fatalf("resolver ResolveOrRegister() calls = %d, want 0", got)
	}
}

func TestCreateNotifiesSessionCreationBeforeImmediateExit(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.driver.startHook = func(opts acp.StartOpts, sequence int) (*fakeProcess, error) {
		proc := newFakeProcess(opts.AgentName, opts.Command, opts.Cwd, fmt.Sprintf("acp-%d", sequence))
		proc.exit()
		return proc, nil
	}

	session := createSession(t, h)
	waitForCondition(t, "stop notification after immediate exit", func() bool {
		return h.notifier.stoppedCount() == 1
	})

	got := h.notifier.notificationOrder()
	want := []string{"created:" + session.ID, "stopped:" + session.ID}
	if !testutil.EqualStringSlices(got, want) {
		t.Fatalf("notification order = %#v, want %#v", got, want)
	}

	meta := readMeta(t, session.MetaPath())
	if meta.StopReason == nil {
		t.Fatal("meta.StopReason = nil, want non-nil")
	}
	if *meta.StopReason != store.StopCompleted {
		t.Fatalf("meta.StopReason = %q, want %q", *meta.StopReason, store.StopCompleted)
	}
}

func TestCreateWithWorkspacePathUsesResolveOrRegister(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	workspacePath := filepath.Join(t.TempDir(), "path-workspace")
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		t.Fatalf("MkdirAll(path workspace) error = %v", err)
	}

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName:     "coder",
		Name:          "path-session",
		WorkspacePath: workspacePath,
	})
	if err != nil {
		t.Fatalf("Create(workspace path) error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if got := len(h.resolver.resolveCalls); got != 0 {
		t.Fatalf("resolver Resolve() calls = %d, want 0", got)
	}
	if got := len(h.resolver.resolveOrRegisterCalls); got != 1 {
		t.Fatalf("resolver ResolveOrRegister() calls = %d, want 1", got)
	}
	if got, want := h.resolver.resolveOrRegisterCalls[0], normalizeResolverPath(workspacePath); got != want {
		t.Fatalf("resolver ResolveOrRegister() arg = %q, want %q", got, want)
	}
	if got, want := session.Info().Workspace, normalizeResolverPath(workspacePath); got != want {
		t.Fatalf("session workspace = %q, want %q", got, want)
	}
	if !strings.HasPrefix(session.Info().WorkspaceID, "ws-auto-") {
		t.Fatalf("session workspace id = %q, want ws-auto-*", session.Info().WorkspaceID)
	}
	if meta := readMeta(t, session.MetaPath()); meta.WorkspaceID != session.Info().WorkspaceID {
		t.Fatalf("meta workspace id = %q, want %q", meta.WorkspaceID, session.Info().WorkspaceID)
	}
}

func TestStopTransitionsToStoppedAndNotifies(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
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
	if meta.StopReason == nil {
		t.Fatal("meta.StopReason = nil, want non-nil")
	}
	if *meta.StopReason != store.StopUserCanceled {
		t.Fatalf("meta.StopReason = %q, want %q", *meta.StopReason, store.StopUserCanceled)
	}
	if got := session.Info().StopReason; got != store.StopUserCanceled {
		t.Fatalf("session.Info().StopReason = %q, want %q", got, store.StopUserCanceled)
	}

	events := readStoredEvents(t, session)
	stopEvent := storedEventByType(t, events, EventTypeSessionStopped)
	stopPayload := decodeStoredEventPayload(t, stopEvent)
	if got, want := stopPayload["stop_reason"], string(store.StopUserCanceled); got != want {
		t.Fatalf("session_stopped stop_reason = %v, want %q", got, want)
	}
}

func TestResumeLoadsMetaAndPassesStoredACPSessionID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	originalACP := session.Info().ACPSessionID

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
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
	if got := resumed.Info().StopReason; got != "" {
		t.Fatalf("resumed stop reason = %q, want empty", got)
	}
	if got := resumed.Info().StopDetail; got != "" {
		t.Fatalf("resumed stop detail = %q, want empty", got)
	}
}

func TestCreateAndResumePreserveChannel(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "networked",
		Workspace: h.workspaceID,
		Channel:   "builders",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got := session.Info().Channel; got != "builders" {
		t.Fatalf("Create() channel = %q, want %q", got, "builders")
	}
	if meta := readMeta(t, session.MetaPath()); meta.Channel != "builders" {
		t.Fatalf("meta channel = %q, want %q", meta.Channel, "builders")
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	stopped, err := h.manager.Status(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Status(stopped) error = %v", err)
	}
	if got := stopped.Channel; got != "builders" {
		t.Fatalf("Status(stopped).Channel = %q, want %q", got, "builders")
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	if got := resumed.Info().Channel; got != "builders" {
		t.Fatalf("Resume() channel = %q, want %q", got, "builders")
	}
	if meta := readMeta(t, resumed.MetaPath()); meta.Channel != "builders" {
		t.Fatalf("resumed meta channel = %q, want %q", meta.Channel, "builders")
	}
}

func TestCreateResumeAndStopInvokeLateBoundNetworkPeerLifecycle(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	lifecycle := newFakeNetworkPeerLifecycle()
	h.manager.SetNetworkPeerLifecycle(lifecycle)

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "networked",
		Workspace: h.workspaceID,
		Channel:   "builders",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got := lifecycle.joinCount(); got != 1 {
		t.Fatalf("join calls after Create() = %d, want 1", got)
	}
	firstJoin := lifecycle.joinCall(0)
	if got, want := firstJoin.sessionID, session.ID; got != want {
		t.Fatalf("first join session_id = %q, want %q", got, want)
	}
	if got, want := firstJoin.peerID, "coder."+session.ID; got != want {
		t.Fatalf("first join peer_id = %q, want %q", got, want)
	}
	if got, want := firstJoin.channel, "builders"; got != want {
		t.Fatalf("first join channel = %q, want %q", got, want)
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if got := lifecycle.leaveCount(); got != 1 {
		t.Fatalf("leave calls after Stop() = %d, want 1", got)
	}
	if got, want := lifecycle.leaveCall(0), session.ID; got != want {
		t.Fatalf("first leave session_id = %q, want %q", got, want)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	if got := lifecycle.joinCount(); got != 2 {
		t.Fatalf("join calls after Resume() = %d, want 2", got)
	}
	secondJoin := lifecycle.joinCall(1)
	if got, want := secondJoin.sessionID, resumed.ID; got != want {
		t.Fatalf("second join session_id = %q, want %q", got, want)
	}
	if got, want := secondJoin.peerID, "coder."+resumed.ID; got != want {
		t.Fatalf("second join peer_id = %q, want %q", got, want)
	}
	if got, want := secondJoin.channel, "builders"; got != want {
		t.Fatalf("second join channel = %q, want %q", got, want)
	}

	if err := h.manager.Stop(testutil.Context(t), resumed.ID); err != nil {
		t.Fatalf("Stop(resumed) error = %v", err)
	}
	if got := lifecycle.leaveCount(); got != 2 {
		t.Fatalf("leave calls after resumed Stop() = %d, want 2", got)
	}
	if got, want := lifecycle.leaveCall(1), resumed.ID; got != want {
		t.Fatalf("second leave session_id = %q, want %q", got, want)
	}
}

func TestResumeRepairsIncompleteStartAndStartsFreshACPClient(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	originalACP := session.Info().ACPSessionID

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	meta := readMeta(t, session.MetaPath())
	meta.State = string(StateStarting)
	meta.StopReason = nil
	meta.StopDetail = ""
	meta.ACPSessionID = stringPointer(originalACP)
	if err := store.WriteSessionMeta(session.MetaPath(), meta); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume(incomplete start) error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	if got := h.driver.startCalls[1].ResumeSessionID; got != "" {
		t.Fatalf("resume start ResumeSessionID = %q, want empty for repaired start", got)
	}
	if got := resumed.Info().ACPSessionID; got == "" || got == originalACP {
		t.Fatalf("resumed ACPSessionID = %q, want fresh ACP session id distinct from %q", got, originalACP)
	}
	if got := resumed.Info().State; got != StateActive {
		t.Fatalf("resumed state = %q, want %q", got, StateActive)
	}
	if got := resumed.Info().StopReason; got != "" {
		t.Fatalf("resumed stop reason = %q, want empty", got)
	}
	if got := resumed.Info().StopDetail; got != "" {
		t.Fatalf("resumed stop detail = %q, want empty", got)
	}
}

func TestResumePreservesCrashStopClassificationFromRepairedMetadata(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	meta := readMeta(t, session.MetaPath())
	meta.State = string(StateActive)
	meta.StopReason = nil
	meta.StopDetail = ""
	if err := store.WriteSessionMeta(session.MetaPath(), meta); err != nil {
		t.Fatalf("WriteSessionMeta() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	if got := resumed.Info().State; got != StateActive {
		t.Fatalf("resumed state = %q, want %q", got, StateActive)
	}
	if got := resumed.Info().StopReason; got != store.StopAgentCrashed {
		t.Fatalf("resumed stop reason = %q, want %q", got, store.StopAgentCrashed)
	}
	if got := resumed.Info().StopDetail; got != resumeStopDetailAgentCrashed {
		t.Fatalf("resumed stop detail = %q, want %q", got, resumeStopDetailAgentCrashed)
	}

	repaired := readMeta(t, resumed.MetaPath())
	if repaired.StopReason == nil {
		t.Fatal("meta.StopReason = nil, want non-nil")
	}
	if *repaired.StopReason != store.StopAgentCrashed {
		t.Fatalf("meta.StopReason = %q, want %q", *repaired.StopReason, store.StopAgentCrashed)
	}
	if got := repaired.StopDetail; got != resumeStopDetailAgentCrashed {
		t.Fatalf("meta.StopDetail = %q, want %q", got, resumeStopDetailAgentCrashed)
	}
}

func TestResumeFallsBackToFreshStartWhenStoredACPSessionIsMissing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	originalACP := session.Info().ACPSessionID

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	h.driver.startHook = func(opts acp.StartOpts, sequence int) (*fakeProcess, error) {
		if opts.ResumeSessionID != "" {
			return nil, fmt.Errorf(
				"%w: load session %q for %q: %w",
				acp.ErrLoadSessionFailed,
				opts.ResumeSessionID,
				opts.AgentName,
				&acpsdk.RequestError{
					Code:    -32002,
					Message: "Resource not found: " + opts.ResumeSessionID,
				},
			)
		}
		return newFakeProcess(opts.AgentName, opts.Command, opts.Cwd, fmt.Sprintf("acp-new-%d", sequence)), nil
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume(missing ACP session) error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	if got := h.driver.startCalls[1].ResumeSessionID; got != originalACP {
		t.Fatalf("first resume start ResumeSessionID = %q, want %q", got, originalACP)
	}
	if got := h.driver.startCalls[2].ResumeSessionID; got != "" {
		t.Fatalf("fallback resume start ResumeSessionID = %q, want empty", got)
	}
	if got := resumed.Info().ACPSessionID; got == "" || got == originalACP {
		t.Fatalf("resumed ACPSessionID = %q, want fresh ACP session id distinct from %q", got, originalACP)
	}
	if got := resumed.Info().State; got != StateActive {
		t.Fatalf("resumed state = %q, want %q", got, StateActive)
	}
	if meta := readMeta(t, session.MetaPath()); meta.State != string(StateActive) {
		t.Fatalf("meta state after fallback resume = %q, want %q", meta.State, StateActive)
	}
}

func TestResumeFailureRestoresStoppedMetadata(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	originalACP := session.Info().ACPSessionID

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	metaBefore := readMeta(t, session.MetaPath())
	h.driver.startHook = func(_ acp.StartOpts, _ int) (*fakeProcess, error) {
		return nil, errors.New("start failed")
	}

	if _, err := h.manager.Resume(testutil.Context(t), session.ID); err == nil {
		t.Fatal("Resume(generic failure) error = nil, want non-nil")
	}

	metaAfter := readMeta(t, session.MetaPath())
	if got := metaAfter.State; got != string(StateStopped) {
		t.Fatalf("meta state after failed resume = %q, want %q", got, StateStopped)
	}
	if got := derefString(metaAfter.ACPSessionID); got != originalACP {
		t.Fatalf("meta ACPSessionID after failed resume = %q, want %q", got, originalACP)
	}
	assertOptionalStopReasonEqual(t, metaAfter.StopReason, metaBefore.StopReason)
	if got := metaAfter.StopDetail; got != metaBefore.StopDetail {
		t.Fatalf("meta stop detail after failed resume = %q, want %q", got, metaBefore.StopDetail)
	}
}

func TestActivateAndWatchUpdatesStateAndStartsWatcher(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	sessionDir := filepath.Join(h.homePaths.SessionsDir, "sess-helper")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(sessionDir) error = %v", err)
	}

	dbPath := store.SessionDBFile(sessionDir)
	recorder, err := sessiondb.OpenSessionDB(testutil.Context(t), "sess-helper", dbPath)
	if err != nil {
		t.Fatalf("OpenSessionDB() error = %v", err)
	}

	session := &Session{
		ID:          "sess-helper",
		Name:        "helper",
		AgentName:   "coder",
		WorkspaceID: h.workspaceID,
		Workspace:   h.workspace,
		Type:        SessionTypeUser,
		State:       StateStarting,
		CreatedAt:   time.Date(2026, 4, 6, 23, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 6, 23, 0, 0, 0, time.UTC),
		sessionDir:  sessionDir,
		metaPath:    store.SessionMetaFile(sessionDir),
		dbPath:      dbPath,
		recorder:    recorder,
	}

	if err := h.manager.reserve(session.ID, h.cfg.Limits.MaxSessions); err != nil {
		t.Fatalf("reserve() error = %v", err)
	}

	proc, err := h.driver.Start(testutil.Context(t), acp.StartOpts{
		AgentName: "coder",
		Command:   "fake-agent",
		Cwd:       h.workspace,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := h.manager.activateAndWatch(
		testutil.Context(t),
		session,
		proc,
		aghconfig.ResolvedAgent{Name: "coder"},
		hookspkg.HookSessionPostCreate,
		false,
	); err != nil {
		t.Fatalf("activateAndWatch() error = %v", err)
	}

	if got := session.Info().State; got != StateActive {
		t.Fatalf("session state = %q, want %q", got, StateActive)
	}
	if got := session.Info().ACPSessionID; got != proc.SessionID {
		t.Fatalf("session ACPSessionID = %q, want %q", got, proc.SessionID)
	}
	if got, ok := h.manager.Get(session.ID); !ok || got != session {
		t.Fatalf("Get(%q) = (%v, %v), want active session", session.ID, got, ok)
	}
	if got := h.notifier.createdCount(); got != 1 {
		t.Fatalf("created notifications = %d, want 1", got)
	}
	if meta := readMeta(t, session.MetaPath()); meta.State != string(StateActive) {
		t.Fatalf("meta state = %q, want %q", meta.State, StateActive)
	}

	h.driver.lastProcess().exit()
	waitForCondition(t, "session watcher finalization", func() bool {
		_, ok := h.manager.Get(session.ID)
		return !ok && h.notifier.stoppedCount() == 1
	})
}

func TestResumeFailsWhenWorkspaceCannotBeResolved(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	h.resolver.resolveErr = workspacepkg.ErrWorkspaceNotFound
	if _, err := h.manager.Resume(testutil.Context(t), session.ID); err == nil {
		t.Fatal("Resume(missing workspace) error = nil, want non-nil")
	} else if !errors.Is(err, workspacepkg.ErrWorkspaceNotFound) {
		t.Fatalf("Resume(missing workspace) error = %v, want ErrWorkspaceNotFound", err)
	}
}

func TestActivateAndWatchRollsBackOnMetaWriteFailure(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	sessionDir := filepath.Join(t.TempDir(), "session")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(sessionDir) error = %v", err)
	}
	blockingPath := filepath.Join(sessionDir, "blocked-parent")
	if err := os.WriteFile(blockingPath, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("WriteFile(blockingPath) error = %v", err)
	}

	recorder, err := h.manager.openStore(testutil.Context(t), "sess-rollback", filepath.Join(sessionDir, "events.db"))
	if err != nil {
		t.Fatalf("openStore() error = %v", err)
	}
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = recorder.Close(closeCtx)
	})

	session := &Session{
		ID:          "sess-rollback",
		AgentName:   "coder",
		WorkspaceID: h.workspaceID,
		Workspace:   h.workspace,
		Type:        SessionTypeUser,
		State:       StateStarting,
		CreatedAt:   time.Date(2026, 4, 6, 23, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 6, 23, 0, 0, 0, time.UTC),
		sessionDir:  sessionDir,
		metaPath:    filepath.Join(blockingPath, "session.json"),
		dbPath:      filepath.Join(sessionDir, "events.db"),
		recorder:    recorder,
	}

	proc, err := h.driver.Start(testutil.Context(t), acp.StartOpts{
		AgentName: "coder",
		Command:   "fake-agent",
		Cwd:       h.workspace,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := h.manager.activateAndWatch(
		testutil.Context(t),
		session,
		proc,
		aghconfig.ResolvedAgent{Name: "coder"},
		hookspkg.HookSessionPostCreate,
		false,
	); err == nil {
		t.Fatal("activateAndWatch() error = nil, want non-nil")
	}
	if _, ok := h.manager.Get(session.ID); ok {
		t.Fatalf("Get(%q) = active session, want rollback", session.ID)
	}
	if got := session.Info().State; got != StateStarting {
		t.Fatalf("session state after rollback = %q, want %q", got, StateStarting)
	}
	if got := session.processHandle(); got != nil {
		t.Fatalf("session process after rollback = %#v, want nil", got)
	}
	if h.driver.stopCalls != 1 {
		t.Fatalf("driver stop calls = %d, want 1", h.driver.stopCalls)
	}
}

func TestCleanupFailedStartRemovesSessionDir(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	recorder := &fakeEventRecorder{}
	proc, err := h.driver.Start(testutil.Context(t), acp.StartOpts{AgentName: "coder"})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	sessionDir := filepath.Join(t.TempDir(), "failed-session")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(sessionDir) error = %v", err)
	}

	if err := h.manager.cleanupFailedStart(sessionDir, recorder, proc); err != nil {
		t.Fatalf("cleanupFailedStart(with dir) error = %v", err)
	}
	if h.driver.stopCalls != 1 {
		t.Fatalf("driver stop calls = %d, want 1", h.driver.stopCalls)
	}
	if recorder.closeCalls != 1 {
		t.Fatalf("recorder close calls = %d, want 1", recorder.closeCalls)
	}
	if _, err := os.Stat(sessionDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(sessionDir) error = %v, want os.ErrNotExist", err)
	}
}

func TestPumpPromptReturnsWhenContextIsCanceledWhileWaitingForSource(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	source := make(chan acp.AgentEvent)
	out := make(chan acp.AgentEvent)
	ctx, cancel := context.WithCancel(testutil.Context(t))

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.manager.pumpPrompt(ctx, nil, newPromptTurnDispatchState(nil, "turn-1", TurnSourceUser, ""), source, out)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("pumpPrompt() did not return after context cancellation")
	}

	select {
	case _, ok := <-out:
		if ok {
			t.Fatal("pumpPrompt() output channel remained open after cancellation")
		}
	default:
		t.Fatal("pumpPrompt() did not close output channel")
	}
}

func TestCleanupFailedStartWithoutSessionDirSkipsRemoval(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	recorder := &fakeEventRecorder{}
	proc, err := h.driver.Start(testutil.Context(t), acp.StartOpts{AgentName: "coder"})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := h.manager.cleanupFailedStart("", recorder, proc); err != nil {
		t.Fatalf("cleanupFailedStart(without dir) error = %v", err)
	}
	if h.driver.stopCalls != 1 {
		t.Fatalf("driver stop calls = %d, want 1", h.driver.stopCalls)
	}
	if recorder.closeCalls != 1 {
		t.Fatalf("recorder close calls = %d, want 1", recorder.closeCalls)
	}
}

func TestPromptStreamsToRecorderAndNotifier(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
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

	stored, err := session.recorderHandle().Query(testutil.Context(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("stored events = %d, want 3", len(stored))
	}
	if got := stored[0].Type; got != acp.EventTypeUserMessage {
		t.Fatalf("first stored event type = %q, want %q", got, acp.EventTypeUserMessage)
	}
	if got := h.notifier.eventCount(session.ID); got != 3 {
		t.Fatalf("notifier events = %d, want 3", got)
	}
}

func TestPromptPersistsUserMessageBeforeDriverPrompt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	var storedBeforePrompt []store.SessionEvent
	h.driver.promptHook = func(_ *fakeProcess, _ acp.PromptRequest) (<-chan acp.AgentEvent, error) {
		events, err := session.recorderHandle().Query(testutil.Context(t), store.EventQuery{})
		if err != nil {
			return nil, err
		}
		storedBeforePrompt = events

		ch := make(chan acp.AgentEvent)
		close(ch)
		return ch, nil
	}

	eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "remember me")
	if err != nil {
		t.Fatalf("Prompt() error = %v", err)
	}
	for event := range eventsCh {
		_ = event
	}

	if len(storedBeforePrompt) != 1 {
		t.Fatalf("storedBeforePrompt = %d events, want 1", len(storedBeforePrompt))
	}
	if got := storedBeforePrompt[0].Type; got != acp.EventTypeUserMessage {
		t.Fatalf("storedBeforePrompt[0].Type = %q, want %q", got, acp.EventTypeUserMessage)
	}
	if !strings.Contains(storedBeforePrompt[0].Content, `"text":"remember me"`) {
		t.Fatalf("stored user_message content = %s", storedBeforePrompt[0].Content)
	}
}

func TestPromptWithOptsTracksTurnSourceAndClearsAfterPrompt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	seenSources := make([]TurnSource, 0, 2)
	h.driver.promptHook = func(_ *fakeProcess, _ acp.PromptRequest) (<-chan acp.AgentEvent, error) {
		seenSources = append(seenSources, session.CurrentTurnSource())

		ch := make(chan acp.AgentEvent)
		close(ch)
		return ch, nil
	}

	firstEvents, err := h.manager.PromptWithOpts(testutil.Context(t), session.ID, PromptOpts{
		Message: "user prompt",
	})
	if err != nil {
		t.Fatalf("PromptWithOpts(user) error = %v", err)
	}
	_ = collectEvents(t, firstEvents)

	secondEvents, err := h.manager.PromptNetwork(testutil.Context(t), session.ID, "network prompt")
	if err != nil {
		t.Fatalf("PromptNetwork() error = %v", err)
	}
	_ = collectEvents(t, secondEvents)

	if !slices.Equal(seenSources, []TurnSource{TurnSourceUser, TurnSourceNetwork}) {
		t.Fatalf("seen turn sources = %#v, want %#v", seenSources, []TurnSource{TurnSourceUser, TurnSourceNetwork})
	}
	if got := session.CurrentTurnSource(); got != "" {
		t.Fatalf("CurrentTurnSource() after prompts = %q, want empty", got)
	}
}

func TestApprovePermissionRoutesToActiveSession(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
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

	err := h.manager.ApprovePermission(testutil.Context(t), session.ID, acp.ApproveRequest{
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
	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	err := h.manager.ApprovePermission(testutil.Context(t), session.ID, acp.ApproveRequest{
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
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	testCases := []struct {
		name    string
		hookErr error
		wantErr error
	}{
		{
			name:    "ShouldMapNotFound",
			hookErr: acp.ErrPendingPermissionNotFound,
			wantErr: ErrPendingPermissionNotFound,
		},
		{
			name:    "ShouldMapConflict",
			hookErr: acp.ErrPendingPermissionConflict,
			wantErr: ErrPendingPermissionConflict,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			session := createSession(t, h)
			t.Cleanup(func() {
				_ = h.manager.Stop(testutil.Context(t), session.ID)
			})

			h.driver.approveHook = func(*fakeProcess, acp.ApproveRequest) error {
				return tc.hookErr
			}
			err := h.manager.ApprovePermission(testutil.Context(t), session.ID, acp.ApproveRequest{
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
	if meta.StopReason == nil {
		t.Fatal("meta.StopReason = nil, want non-nil")
	}
	if *meta.StopReason != store.StopAgentCrashed {
		t.Fatalf("meta.StopReason = %q, want %q", *meta.StopReason, store.StopAgentCrashed)
	}

	events := readStoredEvents(t, session)
	if !containsEventType(events, acp.EventTypeError) {
		t.Fatalf("stored events missing crash error: %#v", events)
	}
	stopEvent := storedEventByType(t, events, EventTypeSessionStopped)
	stopPayload := decodeStoredEventPayload(t, stopEvent)
	if got, want := stopPayload["stop_reason"], string(store.StopAgentCrashed); got != want {
		t.Fatalf("session_stopped stop_reason = %v, want %q", got, want)
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
		stopDone <- h.manager.Stop(testutil.Context(t), session.ID)
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

	reopened, err := sessiondb.OpenSessionDB(testutil.Context(t), session.ID, session.DBPath())
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	defer func() {
		_ = reopened.Close(testutil.Context(t))
	}()

	events, err := reopened.Query(testutil.Context(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query(reopened) error = %v", err)
	}
	if got := countEventType(events, EventTypeSessionStopped); got != 1 {
		t.Fatalf("countEventType(session_stopped) = %d, want 1", got)
	}
	meta := readMeta(t, session.MetaPath())
	if meta.StopReason == nil {
		t.Fatal("meta.StopReason = nil, want non-nil")
	}
	if *meta.StopReason != store.StopUserCanceled {
		t.Fatalf("meta.StopReason = %q, want %q", *meta.StopReason, store.StopUserCanceled)
	}
}

func TestStopWaitsForProcessDoneAfterSuccessfulDriverStop(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	release := make(chan struct{})
	h.driver.stopHook = func(proc *fakeProcess) error {
		go func() {
			<-release
			proc.exit()
		}()
		return nil
	}

	stopCtx, cancel := context.WithTimeout(testutil.Context(t), time.Second)
	defer cancel()

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- h.manager.Stop(stopCtx, session.ID)
	}()

	select {
	case err := <-stopDone:
		t.Fatalf("Stop() completed early with %v, want it blocked on process exit", err)
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	if err := <-stopDone; err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	waitForCondition(t, "stopped session finalization", func() bool {
		_, ok := h.manager.Get(session.ID)
		return !ok && readMeta(t, session.MetaPath()).State == string(StateStopped)
	})
}

func TestPromptSerializesSetupAgainstConcurrentStop(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	session := createSession(t, h)

	promptEntered := make(chan struct{})
	releasePrompt := make(chan struct{})
	h.driver.promptHook = func(_ *fakeProcess, _ acp.PromptRequest) (<-chan acp.AgentEvent, error) {
		close(promptEntered)
		<-releasePrompt
		events := make(chan acp.AgentEvent)
		close(events)
		return events, nil
	}

	promptDone := make(chan error, 1)
	go func() {
		eventsCh, err := h.manager.Prompt(testutil.Context(t), session.ID, "hello")
		if err != nil {
			promptDone <- err
			return
		}
		for event := range eventsCh {
			_ = event
		}
		promptDone <- nil
	}()

	<-promptEntered

	stopDone := make(chan error, 1)
	go func() {
		stopDone <- h.manager.Stop(testutil.Context(t), session.ID)
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
		_ = h.manager.Stop(testutil.Context(t), first.ID)
		_ = h.manager.Stop(testutil.Context(t), second.ID)
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
	readers.Go(func() {
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
	})

	const total = 8
	var workers sync.WaitGroup
	for i := range total {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()

			session, err := h.manager.Create(testutil.Context(t), CreateOpts{
				AgentName: "coder",
				Name:      fmt.Sprintf("session-%d", index),
				Workspace: h.workspaceID,
			})
			if err != nil {
				t.Errorf("Create(%d) error = %v", index, err)
				return
			}
			if _, ok := h.manager.Get(session.ID); !ok {
				t.Errorf("Get(%q) = missing after Create()", session.ID)
			}
			if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
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
		_ = h.manager.Stop(testutil.Context(t), first.ID)
	})

	_, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
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
	skillRegistry := newFakeSkillRegistry()
	h.cfg.MCPServers = []aghconfig.MCPServer{
		{Name: "global", Command: "global-command"},
	}
	h.cfg.Providers["claude"] = aghconfig.ProviderConfig{
		Command: "provider-command",
		MCPServers: []aghconfig.MCPServer{
			{Name: "base", Command: "base-command", Args: []string{"--base"}},
			{Name: "override", Command: "provider-override"},
		},
	}
	h.resolver.upsert(&workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      h.workspaceID,
			RootDir: h.workspace,
			Name:    h.workspaceName,
		},
		Config: h.cfg,
		Agents: []aghconfig.AgentDef{{
			Name:     "coder",
			Provider: "claude",
			Prompt:   "You are helpful.",
			MCPServers: []aghconfig.MCPServer{
				{Name: "override", Command: "agent-override", Args: []string{"--agent"}},
				{Name: "extra", Command: "extra-command"},
			},
		}},
	})
	skillRegistry.setSkills(h.workspaceID, []*skillspkg.Skill{
		{
			Source: skillspkg.SourceUser,
			Meta:   skillspkg.SkillMeta{Name: "skill-mcp"},
			MCPServers: []skillspkg.MCPServerDecl{
				{Name: "override", Command: "skill-override", Args: []string{"--skill"}},
				{Name: "skill-extra", Command: "skill-extra-command"},
			},
		},
	})
	h.manager = newManagerWithHarness(
		t,
		h,
		WithSkillRegistry(skillRegistry),
		WithMCPResolver(skillspkg.NewMCPResolver(aghconfig.SkillsConfig{}, nil)),
	)

	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	got := h.driver.startCalls[0].MCPServers
	if len(got) != 5 {
		t.Fatalf("start MCPServers = %#v, want 5 entries", got)
	}
	if got[0].Name != "global" || got[0].Command != "global-command" {
		t.Fatalf("global MCP server = %#v", got[0])
	}
	if got[1].Name != "base" || got[1].Command != "base-command" {
		t.Fatalf("base MCP server = %#v", got[1])
	}
	if got[2].Name != "override" || got[2].Command != "skill-override" {
		t.Fatalf("override MCP server = %#v", got[2])
	}
	if got[3].Name != "extra" || got[3].Command != "extra-command" {
		t.Fatalf("extra MCP server = %#v", got[3])
	}
	if got[4].Name != "skill-extra" || got[4].Command != "skill-extra-command" {
		t.Fatalf("skill-extra MCP server = %#v", got[4])
	}
	if got := skillRegistry.callCount(); got != 1 {
		t.Fatalf("skill registry call count = %d, want 1", got)
	}
	if got := skillRegistry.call(0).ID; got != h.workspaceID {
		t.Fatalf("skill registry workspace id = %q, want %q", got, h.workspaceID)
	}
}

func TestResumePassesMergedSkillMCPServers(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	skillRegistry := newFakeSkillRegistry()
	skillRegistry.setSkills(h.workspaceID, []*skillspkg.Skill{
		{
			Source: skillspkg.SourceUser,
			Meta:   skillspkg.SkillMeta{Name: "resume-skill"},
			MCPServers: []skillspkg.MCPServerDecl{
				{Name: "resume-extra", Command: "resume-extra-command"},
			},
		},
	})
	h.manager = newManagerWithHarness(
		t,
		h,
		WithSkillRegistry(skillRegistry),
		WithMCPResolver(skillspkg.NewMCPResolver(aghconfig.SkillsConfig{}, nil)),
	)

	session := createSession(t, h)
	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	got := h.driver.startCalls[1].MCPServers
	if len(got) != 1 {
		t.Fatalf("resume start MCPServers = %#v, want 1 entry", got)
	}
	if got[0].Name != "resume-extra" || got[0].Command != "resume-extra-command" {
		t.Fatalf("resume MCP server = %#v", got[0])
	}
	if got := skillRegistry.callCount(); got != 2 {
		t.Fatalf("skill registry call count after resume = %d, want 2", got)
	}
}

func TestCreateBlocksMarketplaceSkillMCPServersWithoutConsent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	skillRegistry := newFakeSkillRegistry()
	skillRegistry.setSkills(h.workspaceID, []*skillspkg.Skill{
		{
			Source: skillspkg.SourceMarketplace,
			Meta:   skillspkg.SkillMeta{Name: "market-skill"},
			MCPServers: []skillspkg.MCPServerDecl{
				{Name: "market-extra", Command: "market-extra-command"},
			},
		},
	})
	h.manager = newManagerWithHarness(
		t,
		h,
		WithSkillRegistry(skillRegistry),
		WithMCPResolver(skillspkg.NewMCPResolver(aghconfig.SkillsConfig{}, nil)),
	)

	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if got := h.driver.startCalls[0].MCPServers; len(got) != 0 {
		t.Fatalf("start MCPServers = %#v, want marketplace skill MCP blocked", got)
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
	h.manager = newManagerWithHarness(
		t,
		h,
		WithPromptAssembler(
			promptAssemblerFunc(
				func(_ context.Context, agent aghconfig.AgentDef, workspace *workspacepkg.ResolvedWorkspace) (string, error) {
					called = true
					gotWorkspace = workspace.RootDir
					gotAgentName = agent.Name
					gotAgentPrompt = agent.Prompt
					return agent.Prompt + "\n\nmemory block", nil
				},
			),
		),
	)

	session := createSession(t, h)
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
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
	if got := h.driver.startCalls[0].SystemPrompt; got != "You are a coding assistant.\n\nmemory block" {
		t.Fatalf("start system prompt = %q, want assembled prompt", got)
	}
}

func TestCreateWithChannelAppendsBundledNetworkSkillAfterPromptAssembly(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	networkSkill, err := bundled.LoadContent(networkSkillName)
	if err != nil {
		t.Fatalf("LoadContent(%q) error = %v", networkSkillName, err)
	}

	h.manager = newManagerWithHarness(
		t,
		h,
		WithPromptAssembler(
			promptAssemblerFunc(
				func(_ context.Context, agent aghconfig.AgentDef, workspace *workspacepkg.ResolvedWorkspace) (string, error) {
					if got, want := workspace.RootDir, h.workspace; got != want {
						t.Fatalf("assembler workspace = %q, want %q", got, want)
					}
					return agent.Prompt + "\n\nmemory block", nil
				},
			),
		),
	)

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "networked",
		Workspace: h.workspaceID,
		Channel:   "builders",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	wantPrompt := "You are a coding assistant.\n\nmemory block\n\n" + networkSkill
	if got := h.driver.startCalls[0].SystemPrompt; got != wantPrompt {
		t.Fatalf("start system prompt = %q, want %q", got, wantPrompt)
	}
	if got := strings.Count(h.driver.startCalls[0].SystemPrompt, networkSkill); got != 1 {
		t.Fatalf("network skill occurrences = %d, want 1", got)
	}
}

func TestCreateUsesRawPromptWhenAssemblerIsNil(t *testing.T) {
	t.Parallel()

	h := newHarness(t, WithPromptAssembler(nil))

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if got := h.driver.startCalls[0].SystemPrompt; got != "You are a coding assistant." {
		t.Fatalf("start system prompt = %q, want raw agent prompt", got)
	}
}

func TestCreateWithoutChannelDoesNotAppendBundledNetworkSkill(t *testing.T) {
	t.Parallel()

	h := newHarness(t, WithPromptAssembler(nil))
	networkSkill, err := bundled.LoadContent(networkSkillName)
	if err != nil {
		t.Fatalf("LoadContent(%q) error = %v", networkSkillName, err)
	}

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if got := h.driver.startCalls[0].SystemPrompt; got != "You are a coding assistant." {
		t.Fatalf("start system prompt = %q, want raw agent prompt", got)
	}
	if strings.Contains(h.driver.startCalls[0].SystemPrompt, networkSkill) {
		t.Fatalf("start system prompt unexpectedly contains bundled network skill")
	}
}

func TestResumeWithChannelReinjectsBundledNetworkSkillOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t, WithPromptAssembler(nil))
	networkSkill, err := bundled.LoadContent(networkSkillName)
	if err != nil {
		t.Fatalf("LoadContent(%q) error = %v", networkSkillName, err)
	}

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "networked",
		Workspace: h.workspaceID,
		Channel:   "builders",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got := strings.Count(h.driver.startCalls[0].SystemPrompt, networkSkill); got != 1 {
		t.Fatalf("create prompt network skill occurrences = %d, want 1", got)
	}

	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	wantPrompt := "You are a coding assistant.\n\n" + networkSkill
	if got := h.driver.startCalls[1].SystemPrompt; got != wantPrompt {
		t.Fatalf("resume system prompt = %q, want %q", got, wantPrompt)
	}
	if got := strings.Count(h.driver.startCalls[1].SystemPrompt, networkSkill); got != 1 {
		t.Fatalf("resume prompt network skill occurrences = %d, want 1", got)
	}
}

func TestCreateWithChannelInjectsNetworkSessionEnv(t *testing.T) {
	t.Parallel()

	h := newHarness(t, WithPromptAssembler(nil))

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "networked",
		Workspace: h.workspaceID,
		Channel:   "builders",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	env := h.driver.startCalls[0].Env
	if got, ok := lookupEnvValue(env, "AGH_SESSION_ID"); !ok || got != session.ID {
		t.Fatalf("AGH_SESSION_ID = %q, %v, want %q", got, ok, session.ID)
	}
	if got, ok := lookupEnvValue(env, "AGH_SESSION_CHANNEL"); !ok || got != "builders" {
		t.Fatalf("AGH_SESSION_CHANNEL = %q, %v, want %q", got, ok, "builders")
	}
	if got, ok := lookupEnvValue(env, "AGH_PEER_ID"); !ok || got != "coder."+session.ID {
		t.Fatalf("AGH_PEER_ID = %q, %v, want %q", got, ok, "coder."+session.ID)
	}
}

func TestCreateWithoutChannelOmitsNetworkChannelEnv(t *testing.T) {
	t.Parallel()

	h := newHarness(t, WithPromptAssembler(nil))

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	env := h.driver.startCalls[0].Env
	if got, ok := lookupEnvValue(env, "AGH_SESSION_ID"); !ok || got != session.ID {
		t.Fatalf("AGH_SESSION_ID = %q, %v, want %q", got, ok, session.ID)
	}
	if got, ok := lookupEnvValue(env, "AGH_SESSION_CHANNEL"); ok {
		t.Fatalf("AGH_SESSION_CHANNEL = %q, want unset", got)
	}
	if got, ok := lookupEnvValue(env, "AGH_PEER_ID"); ok {
		t.Fatalf("AGH_PEER_ID = %q, want unset", got)
	}
}

func TestResumeWithChannelReinjectsNetworkSessionEnv(t *testing.T) {
	t.Parallel()

	h := newHarness(t, WithPromptAssembler(nil))

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "networked",
		Workspace: h.workspaceID,
		Channel:   "builders",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := h.manager.Stop(testutil.Context(t), session.ID); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	resumed, err := h.manager.Resume(testutil.Context(t), session.ID)
	if err != nil {
		t.Fatalf("Resume() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), resumed.ID)
	})

	env := h.driver.startCalls[1].Env
	if got, ok := lookupEnvValue(env, "AGH_SESSION_ID"); !ok || got != resumed.ID {
		t.Fatalf("AGH_SESSION_ID = %q, %v, want %q", got, ok, resumed.ID)
	}
	if got, ok := lookupEnvValue(env, "AGH_SESSION_CHANNEL"); !ok || got != "builders" {
		t.Fatalf("AGH_SESSION_CHANNEL = %q, %v, want %q", got, ok, "builders")
	}
	if got, ok := lookupEnvValue(env, "AGH_PEER_ID"); !ok || got != "coder."+resumed.ID {
		t.Fatalf("AGH_PEER_ID = %q, %v, want %q", got, ok, "coder."+resumed.ID)
	}
}

func TestCreateAppliesDreamPermissionsOverride(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.cfg.Permissions.Mode = aghconfig.PermissionModeDenyAll
	h.resolver.upsert(&workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      h.workspaceID,
			RootDir: h.workspace,
			Name:    h.workspaceName,
		},
		Config: h.cfg,
		Agents: []aghconfig.AgentDef{
			{
				Name:     aghconfig.DefaultAgentName,
				Provider: "claude",
				Prompt:   "You are a coding assistant.",
			},
			{
				Name:     "coder",
				Provider: "claude",
				Prompt:   "You are a coding assistant.",
			},
		},
	})
	h.manager = newManagerWithHarness(t, h)

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
		Type:      SessionTypeDream,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if got := h.driver.startCalls[0].Permissions; got != aghconfig.PermissionModeApproveAll {
		t.Fatalf("start permissions = %q, want %q", got, aghconfig.PermissionModeApproveAll)
	}
}

func TestCreateUsesConfiguredPermissionsForUserSessions(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.cfg.Permissions.Mode = aghconfig.PermissionModeDenyAll
	h.resolver.upsert(&workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      h.workspaceID,
			RootDir: h.workspace,
			Name:    h.workspaceName,
		},
		Config: h.cfg,
		Agents: []aghconfig.AgentDef{
			{
				Name:     aghconfig.DefaultAgentName,
				Provider: "claude",
				Prompt:   "You are a coding assistant.",
			},
			{
				Name:     "coder",
				Provider: "claude",
				Prompt:   "You are a coding assistant.",
			},
		},
	})
	h.manager = newManagerWithHarness(t, h)

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Workspace: h.workspaceID,
		Type:      SessionTypeUser,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		_ = h.manager.Stop(testutil.Context(t), session.ID)
	})

	if got := h.driver.startCalls[0].Permissions; got != aghconfig.PermissionModeDenyAll {
		t.Fatalf("start permissions = %q, want %q", got, aghconfig.PermissionModeDenyAll)
	}
}

func TestACPDriverAdapterErrorPaths(t *testing.T) {
	t.Parallel()

	adapter := NewACPDriverAdapter(acp.New())
	if _, err := adapter.Prompt(testutil.Context(t), &AgentProcess{}, acp.PromptRequest{}); err == nil {
		t.Fatal("Prompt(unsupported process) error = nil, want non-nil")
	}
	if err := adapter.Stop(testutil.Context(t), &AgentProcess{}); err == nil {
		t.Fatal("Stop(unsupported process) error = nil, want non-nil")
	}
}

type harness struct {
	manager       *Manager
	driver        *fakeDriver
	notifier      *fakeNotifier
	resolver      *fakeWorkspaceResolver
	cfg           aghconfig.Config
	homePaths     aghconfig.HomePaths
	workspace     string
	workspaceID   string
	workspaceName string
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
		driver:        newFakeDriver(),
		notifier:      newFakeNotifier(),
		cfg:           aghconfig.DefaultWithHome(homePaths),
		homePaths:     homePaths,
		workspace:     workspace,
		workspaceID:   "ws-primary",
		workspaceName: "workspace",
	}
	h.resolver = newFakeWorkspaceResolver(&workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      h.workspaceID,
			RootDir: h.workspace,
			Name:    h.workspaceName,
		},
		Config: h.cfg,
		Agents: []aghconfig.AgentDef{
			{
				Name:     aghconfig.DefaultAgentName,
				Provider: "claude",
				Prompt:   "You are a coding assistant.",
			},
			{
				Name:     "coder",
				Provider: "claude",
				Prompt:   "You are a coding assistant.",
			},
		},
	})
	h.manager = newManagerWithHarness(t, h, extraOpts...)
	return h
}

func newManagerWithHarness(t *testing.T, h *harness, extraOpts ...Option) *Manager {
	t.Helper()

	opts := []Option{
		WithHomePaths(h.homePaths),
		WithDriver(h.driver),
		WithNotifier(h.notifier),
		WithWorkspaceResolver(h.resolver),
		WithStore(func(ctx context.Context, sessionID string, path string) (EventRecorder, error) {
			return sessiondb.OpenSessionDB(ctx, sessionID, path)
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

	session, err := h.manager.Create(testutil.Context(t), CreateOpts{
		AgentName: "coder",
		Name:      "session",
		Workspace: h.workspaceID,
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

func readStoredEvents(t *testing.T, session *Session) []store.SessionEvent {
	t.Helper()

	reopened, err := sessiondb.OpenSessionDB(testutil.Context(t), session.ID, session.DBPath())
	if err != nil {
		t.Fatalf("OpenSessionDB(reopen) error = %v", err)
	}
	defer func() {
		if err := reopened.Close(testutil.Context(t)); err != nil {
			t.Fatalf("Close(reopened) error = %v", err)
		}
	}()

	events, err := reopened.Query(testutil.Context(t), store.EventQuery{})
	if err != nil {
		t.Fatalf("Query(reopened) error = %v", err)
	}
	return events
}

func storedEventByType(t *testing.T, events []store.SessionEvent, want string) store.SessionEvent {
	t.Helper()

	for _, event := range events {
		if event.Type == want {
			return event
		}
	}

	t.Fatalf("stored event type %q not found", want)
	return store.SessionEvent{}
}

func decodeStoredEventPayload(t *testing.T, event store.SessionEvent) map[string]any {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal([]byte(event.Content), &payload); err != nil {
		t.Fatalf("json.Unmarshal(event.Content) error = %v", err)
	}
	return payload
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

func sequentialIDGenerator(prefix string) IDGenerator {
	var counter atomic.Int64
	return func() string {
		return fmt.Sprintf("%s-%d", prefix, counter.Add(1))
	}
}

type promptAssemblerFunc func(context.Context, aghconfig.AgentDef, *workspacepkg.ResolvedWorkspace) (string, error)

func (fn promptAssemblerFunc) Assemble(
	ctx context.Context,
	agent aghconfig.AgentDef,
	workspace *workspacepkg.ResolvedWorkspace,
) (string, error) {
	return fn(ctx, agent, workspace)
}

type fakeNotifier struct {
	mu      sync.Mutex
	created []*Info
	stopped []*Info
	events  map[string][]acp.AgentEvent
	order   []string
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
	n.order = append(n.order, "created:"+session.ID)
}

func (n *fakeNotifier) OnSessionStopped(_ context.Context, session *Session) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.stopped = append(n.stopped, session.Info())
	n.order = append(n.order, "stopped:"+session.ID)
}

func (n *fakeNotifier) OnAgentEvent(_ context.Context, sessionID string, event any) {
	n.mu.Lock()
	defer n.mu.Unlock()
	agentEvent, ok := event.(acp.AgentEvent)
	if !ok {
		return
	}
	n.events[sessionID] = append(n.events[sessionID], agentEvent)
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

func (n *fakeNotifier) notificationOrder() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]string(nil), n.order...)
}

type fakeNetworkPeerLifecycle struct {
	mu    sync.Mutex
	joins []fakeNetworkJoinCall
	left  []string
}

type fakeNetworkJoinCall struct {
	sessionID string
	peerID    string
	channel   string
}

func newFakeNetworkPeerLifecycle() *fakeNetworkPeerLifecycle {
	return &fakeNetworkPeerLifecycle{}
}

func (f *fakeNetworkPeerLifecycle) JoinChannel(
	_ context.Context,
	sessionID string,
	peerID string,
	channel string,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.joins = append(f.joins, fakeNetworkJoinCall{
		sessionID: sessionID,
		peerID:    peerID,
		channel:   channel,
	})
	return nil
}

func (f *fakeNetworkPeerLifecycle) LeaveChannel(_ context.Context, sessionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.left = append(f.left, sessionID)
	return nil
}

func (f *fakeNetworkPeerLifecycle) joinCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.joins)
}

func (f *fakeNetworkPeerLifecycle) joinCall(index int) fakeNetworkJoinCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.joins[index]
}

func (f *fakeNetworkPeerLifecycle) leaveCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.left)
}

func (f *fakeNetworkPeerLifecycle) leaveCall(index int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.left[index]
}

type fakeEventRecorder struct {
	closeCalls int
}

func (r *fakeEventRecorder) Record(context.Context, store.SessionEvent) error {
	return nil
}

func (r *fakeEventRecorder) RecordTokenUsage(context.Context, store.TokenUsage) error {
	return nil
}

func (r *fakeEventRecorder) Query(context.Context, store.EventQuery) ([]store.SessionEvent, error) {
	return nil, nil
}

func (r *fakeEventRecorder) History(context.Context, store.EventQuery) ([]store.TurnHistory, error) {
	return nil, nil
}

func (r *fakeEventRecorder) Close(context.Context) error {
	r.closeCalls++
	return nil
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

type fakeWorkspaceResolver struct {
	mu                     sync.Mutex
	byRef                  map[string]workspacepkg.ResolvedWorkspace
	byPath                 map[string]workspacepkg.ResolvedWorkspace
	resolveCalls           []string
	resolveOrRegisterCalls []string
	resolveErr             error
	resolveOrRegisterErr   error
	resolveHook            func(context.Context, string) (workspacepkg.ResolvedWorkspace, error)
	resolveOrRegisterHook  func(context.Context, string) (workspacepkg.ResolvedWorkspace, error)
	autoRegisterConfig     aghconfig.Config
	autoRegisterAgents     []aghconfig.AgentDef
	nextID                 int
}

type fakeSkillRegistry struct {
	mu                sync.Mutex
	skillsByWorkspace map[string][]*skillspkg.Skill
	calls             []workspacepkg.ResolvedWorkspace
	err               error
}

func newFakeWorkspaceResolver(resolved *workspacepkg.ResolvedWorkspace) *fakeWorkspaceResolver {
	r := &fakeWorkspaceResolver{
		byRef:              make(map[string]workspacepkg.ResolvedWorkspace),
		byPath:             make(map[string]workspacepkg.ResolvedWorkspace),
		autoRegisterConfig: resolved.Config,
		autoRegisterAgents: append([]aghconfig.AgentDef(nil), resolved.Agents...),
	}
	r.upsert(resolved)
	return r
}

func newFakeSkillRegistry() *fakeSkillRegistry {
	return &fakeSkillRegistry{
		skillsByWorkspace: make(map[string][]*skillspkg.Skill),
	}
}

func (r *fakeSkillRegistry) ForWorkspace(
	_ context.Context,
	resolved *workspacepkg.ResolvedWorkspace,
) ([]*skillspkg.Skill, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if resolved != nil {
		r.calls = append(r.calls, cloneResolvedWorkspaceForTests(resolved))
	}
	if r.err != nil {
		return nil, r.err
	}
	if resolved == nil {
		return nil, nil
	}

	skills := r.skillsByWorkspace[resolved.ID]
	return append([]*skillspkg.Skill(nil), skills...), nil
}

func (r *fakeSkillRegistry) setSkills(workspaceID string, skills []*skillspkg.Skill) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.skillsByWorkspace[strings.TrimSpace(workspaceID)] = append([]*skillspkg.Skill(nil), skills...)
}

func (r *fakeSkillRegistry) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *fakeSkillRegistry) call(index int) workspacepkg.ResolvedWorkspace {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneResolvedWorkspaceForTests(&r.calls[index])
}

func (r *fakeWorkspaceResolver) Resolve(ctx context.Context, idOrPath string) (workspacepkg.ResolvedWorkspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ref := strings.TrimSpace(idOrPath)
	r.resolveCalls = append(r.resolveCalls, ref)
	if r.resolveHook != nil {
		return r.resolveHook(ctx, ref)
	}
	if r.resolveErr != nil {
		return workspacepkg.ResolvedWorkspace{}, r.resolveErr
	}
	if resolved, ok := r.byRef[ref]; ok {
		return cloneResolvedWorkspaceForTests(&resolved), nil
	}
	if resolved, ok := r.byPath[normalizeResolverPath(ref)]; ok {
		return cloneResolvedWorkspaceForTests(&resolved), nil
	}
	return workspacepkg.ResolvedWorkspace{}, workspacepkg.ErrWorkspaceNotFound
}

func (r *fakeWorkspaceResolver) ResolveOrRegister(
	ctx context.Context,
	path string,
) (workspacepkg.ResolvedWorkspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	target := normalizeResolverPath(path)
	r.resolveOrRegisterCalls = append(r.resolveOrRegisterCalls, target)
	if r.resolveOrRegisterHook != nil {
		return r.resolveOrRegisterHook(ctx, target)
	}
	if r.resolveOrRegisterErr != nil {
		return workspacepkg.ResolvedWorkspace{}, r.resolveOrRegisterErr
	}
	if resolved, ok := r.byPath[target]; ok {
		return cloneResolvedWorkspaceForTests(&resolved), nil
	}

	r.nextID++
	resolved := workspacepkg.ResolvedWorkspace{
		Workspace: workspacepkg.Workspace{
			ID:      fmt.Sprintf("ws-auto-%d", r.nextID),
			RootDir: target,
			Name:    filepath.Base(target),
		},
		Config: r.autoRegisterConfig,
		Agents: append([]aghconfig.AgentDef(nil), r.autoRegisterAgents...),
	}
	r.upsert(&resolved)
	return cloneResolvedWorkspaceForTests(&resolved), nil
}

func (r *fakeWorkspaceResolver) upsert(resolved *workspacepkg.ResolvedWorkspace) {
	cloned := cloneResolvedWorkspaceForTests(resolved)
	r.byRef[cloned.ID] = cloned
	if name := strings.TrimSpace(cloned.Name); name != "" {
		r.byRef[name] = cloned
	}
	if path := normalizeResolverPath(cloned.RootDir); path != "" {
		cloned.RootDir = path
		r.byPath[path] = cloned
	}
}

func normalizeResolverPath(path string) string {
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

func cloneResolvedWorkspaceForTests(src *workspacepkg.ResolvedWorkspace) workspacepkg.ResolvedWorkspace {
	if src == nil {
		return workspacepkg.ResolvedWorkspace{}
	}
	dst := *src
	dst.AdditionalDirs = append([]string(nil), src.AdditionalDirs...)
	dst.Agents = append([]aghconfig.AgentDef(nil), src.Agents...)
	dst.Skills = append([]workspacepkg.SkillPath(nil), src.Skills...)
	return dst
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
	copied.AdditionalDirs = append([]string(nil), opts.AdditionalDirs...)
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

func (d *fakeDriver) Prompt(
	_ context.Context,
	proc *AgentProcess,
	req acp.PromptRequest,
) (<-chan acp.AgentEvent, error) {
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

func lookupEnvValue(env []string, key string) (string, bool) {
	for _, entry := range env {
		existingKey, value, ok := strings.Cut(entry, "=")
		if ok && existingKey == key {
			return value, true
		}
	}
	return "", false
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
		Caps: acp.Caps{
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
