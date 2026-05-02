package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	aghdaemon "github.com/pedronauck/agh/internal/daemon"
	"github.com/pedronauck/agh/internal/procutil"
)

type stubRunner struct {
	ran bool
}

func (s *stubRunner) Run(context.Context) error {
	s.ran = true
	return nil
}

func TestCommandPathsAndHelpers(t *testing.T) {
	t.Parallel()

	statusSession := SessionRecord{
		ID:            "sess-1",
		AgentName:     "coder",
		WorkspaceID:   "ws-1",
		WorkspacePath: "/workspace/project",
		State:         "active",
		CreatedAt:     fixedTestNow,
		UpdatedAt:     fixedTestNow,
	}
	statusSessionHealth := SessionHealthRecord{
		SessionID:       "sess-1",
		AgentName:       "coder",
		WorkspaceID:     "ws-1",
		State:           "idle",
		Health:          "healthy",
		Attachable:      true,
		EligibleForWake: true,
		UpdatedAt:       fixedTestNow,
	}
	tempDir := t.TempDir()
	soulBodyPath := filepath.Join(tempDir, "SOUL.md")
	if err := os.WriteFile(soulBodyPath, []byte("# Soul\n\nStay precise.\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(SOUL.md) error = %v", err)
	}
	heartbeatBodyPath := filepath.Join(tempDir, "HEARTBEAT.md")
	if err := os.WriteFile(heartbeatBodyPath, []byte("# Heartbeat\n\nCheck in.\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile(HEARTBEAT.md) error = %v", err)
	}

	getCalls := 0
	networkChannelsCalled := false
	getAgentSoulCalled := false
	putAgentSoulCalled := false
	deleteAgentSoulCalled := false
	rollbackAgentSoulCalled := false
	refreshSessionSoulCalled := false
	getAgentHeartbeatCalled := false
	putAgentHeartbeatCalled := false
	deleteAgentHeartbeatCalled := false
	rollbackAgentHeartbeatCalled := false
	getAgentHeartbeatStatusCalled := false
	client := &stubClient{
		getAgentFn: func(context.Context, string, AgentQuery) (AgentRecord, error) {
			return AgentRecord{Name: "coder", Provider: "fake", Prompt: "hi"}, nil
		},
		getAgentSoulFn: func(context.Context, string, AgentQuery) (AgentSoulRecord, error) {
			getAgentSoulCalled = true
			return AgentSoulRecord{AgentName: "coder", Enabled: true, Valid: true, ValidationStatus: "valid"}, nil
		},
		putAgentSoulFn: func(_ context.Context, _ string, request AgentSoulPutRequest) (AgentSoulMutationRecord, error) {
			putAgentSoulCalled = true
			return AgentSoulMutationRecord{
				Soul: AgentSoulRecord{
					AgentName:        "coder",
					Valid:            true,
					ValidationStatus: "valid",
					Digest:           request.ExpectedDigest,
				},
			}, nil
		},
		deleteAgentSoulFn: func(_ context.Context, _ string, request AgentSoulDeleteRequest) (AgentSoulMutationRecord, error) {
			deleteAgentSoulCalled = true
			return AgentSoulMutationRecord{
				Soul: AgentSoulRecord{
					AgentName:        "coder",
					Valid:            true,
					ValidationStatus: "valid",
					Digest:           request.ExpectedDigest,
				},
			}, nil
		},
		rollbackAgentSoulFn: func(
			_ context.Context,
			_ string,
			request AgentSoulRollbackRequest,
		) (AgentSoulMutationRecord, error) {
			rollbackAgentSoulCalled = true
			return AgentSoulMutationRecord{
				Soul: AgentSoulRecord{
					AgentName:        "coder",
					Valid:            true,
					ValidationStatus: "valid",
					Digest:           request.ExpectedDigest,
				},
			}, nil
		},
		refreshSessionSoulFn: func(context.Context, string, SessionSoulRefreshRequest) (AgentSoulRecord, error) {
			refreshSessionSoulCalled = true
			return AgentSoulRecord{AgentName: "coder", Enabled: true, Valid: true, ValidationStatus: "valid"}, nil
		},
		getAgentHeartbeatFn: func(context.Context, string, AgentQuery) (AgentHeartbeatRecord, error) {
			getAgentHeartbeatCalled = true
			return AgentHeartbeatRecord{AgentName: "coder", Enabled: true, Valid: true, ValidationStatus: "valid"}, nil
		},
		putAgentHeartbeatFn: func(
			_ context.Context,
			_ string,
			request AgentHeartbeatPutRequest,
		) (AgentHeartbeatMutationRecord, error) {
			putAgentHeartbeatCalled = true
			return AgentHeartbeatMutationRecord{
				Heartbeat: AgentHeartbeatRecord{
					AgentName: "coder", Valid: true, ValidationStatus: "valid", Digest: request.ExpectedDigest,
				},
			}, nil
		},
		deleteAgentHeartbeatFn: func(
			_ context.Context,
			_ string,
			request AgentHeartbeatDeleteRequest,
		) (AgentHeartbeatMutationRecord, error) {
			deleteAgentHeartbeatCalled = true
			return AgentHeartbeatMutationRecord{
				Heartbeat: AgentHeartbeatRecord{
					AgentName: "coder", Valid: true, ValidationStatus: "valid", Digest: request.ExpectedDigest,
				},
			}, nil
		},
		rollbackAgentHeartbeatFn: func(
			_ context.Context,
			_ string,
			request AgentHeartbeatRollbackRequest,
		) (AgentHeartbeatMutationRecord, error) {
			rollbackAgentHeartbeatCalled = true
			return AgentHeartbeatMutationRecord{
				Heartbeat: AgentHeartbeatRecord{
					AgentName: "coder", Valid: true, ValidationStatus: "valid", Digest: request.ExpectedDigest,
				},
			}, nil
		},
		getAgentHeartbeatStatusFn: func(
			context.Context,
			string,
			AgentHeartbeatStatusRequest,
		) (AgentHeartbeatStatusRecord, error) {
			getAgentHeartbeatStatusCalled = true
			return AgentHeartbeatStatusRecord{
				AgentName:        "coder",
				Enabled:          true,
				Valid:            true,
				ValidationStatus: "valid",
			}, nil
		},
		networkStatusFn: func(context.Context) (NetworkStatusRecord, error) {
			return NetworkStatusRecord{Enabled: true, Status: "running"}, nil
		},
		networkPeersFn: func(_ context.Context, query NetworkPeersQuery) ([]NetworkPeerRecord, error) {
			if query.Channel != "builders" {
				t.Fatalf("NetworkPeers() query = %#v, want builders channel", query)
			}
			return []NetworkPeerRecord{{PeerID: "reviewer.sess-1", Channel: "builders"}}, nil
		},
		networkChannelsFn: func(context.Context) ([]NetworkChannelRecord, error) {
			networkChannelsCalled = true
			return []NetworkChannelRecord{{Channel: "builders", PeerCount: 1}}, nil
		},
		networkSendFn: func(_ context.Context, request NetworkSendRequest) (NetworkSendRecord, error) {
			if request.SessionID != "sess-1" || request.Channel != "builders" || request.Kind != "say" ||
				string(request.Body) != `{"text":"hello"}` {
				t.Fatalf("NetworkSend() request = %#v, want session/channel/kind/body", request)
			}
			return NetworkSendRecord{ID: "msg-1", SessionID: "sess-1", Channel: "builders", Kind: "say"}, nil
		},
		networkInboxFn: func(_ context.Context, sessionID string) ([]NetworkEnvelopeRecord, error) {
			if sessionID != "sess-1" {
				t.Fatalf("NetworkInbox() sessionID = %q, want sess-1", sessionID)
			}
			return []NetworkEnvelopeRecord{
				{ID: "msg-1", Kind: "say", Channel: "builders", From: "reviewer.sess-1"},
			}, nil
		},
		observeEventsFn: func(context.Context, ObserveEventQuery) ([]ObserveEventRecord, error) {
			return []ObserveEventRecord{
				{ID: "sum-1", SessionID: "sess-1", Type: "done", AgentName: "coder", Timestamp: fixedTestNow},
			}, nil
		},
		streamObserveEventsFn: func(_ context.Context, _ ObserveEventQuery, _ string, handler SSEHandler) error {
			return handler(
				SSEEvent{
					Event: "done",
					Data: mustJSON(
						t,
						ObserveEventRecord{
							ID:        "sum-1",
							SessionID: "sess-1",
							Type:      "done",
							AgentName: "coder",
							Timestamp: fixedTestNow,
						},
					),
				},
			)
		},
		observeHealthFn: func(context.Context) (HealthStatus, error) {
			return HealthStatus{Status: "ok", UptimeSeconds: 10}, nil
		},
		getSessionFn: func(context.Context, string) (SessionRecord, error) {
			getCalls++
			if getCalls == 1 {
				return statusSession, nil
			}
			stopped := statusSession
			stopped.State = "stopped"
			return stopped, nil
		},
		getSessionHealthFn: func(context.Context, string) (SessionHealthRecord, error) {
			return statusSessionHealth, nil
		},
		getSessionStatusFn: func(context.Context, string) (SessionStatusRecord, error) {
			return SessionStatusRecord{
				SessionID:       "sess-1",
				AgentName:       "coder",
				WorkspaceID:     "ws-1",
				State:           "idle",
				Health:          "healthy",
				Attachable:      true,
				EligibleForWake: true,
				UpdatedAt:       fixedTestNow,
			}, nil
		},
		inspectSessionFn: func(context.Context, string, SessionInspectQuery) (SessionInspectRecord, error) {
			return SessionInspectRecord{SessionID: "sess-1", Health: statusSessionHealth}, nil
		},
		resumeSessionFn: func(context.Context, string) (SessionRecord, error) {
			return statusSession, nil
		},
		streamSessionFn: func(_ context.Context, _ string, _ SessionEventQuery, _ string, handler SSEHandler) error {
			return handler(SSEEvent{Event: "session_stopped"})
		},
		sessionHistoryFn: func(context.Context, string, SessionEventQuery) ([]TurnHistoryRecord, error) {
			return []TurnHistoryRecord{{TurnID: "turn-1"}}, nil
		},
		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
			return DaemonStatus{Status: "running", PID: 10, StartedAt: fixedTestNow}, nil
		},
		getBridgeFn: func(context.Context, string) (BridgeRecord, error) {
			return BridgeRecord{
				ID:            "brg-1",
				Scope:         "global",
				Platform:      "telegram",
				ExtensionName: "ext-telegram",
				DisplayName:   "Support",
				Enabled:       true,
				Status:        "ready",
			}, nil
		},
		bridgeRoutesFn: func(context.Context, string) ([]BridgeRouteRecord, error) {
			return []BridgeRouteRecord{
				{
					RoutingKeyHash:   "hash-1",
					Scope:            "global",
					BridgeInstanceID: "brg-1",
					PeerID:           "peer-1",
					SessionID:        "sess-1",
					AgentName:        "coder",
					LastActivityAt:   fixedTestNow,
				},
			}, nil
		},
		testBridgeDeliveryFn: func(context.Context, string, BridgeTestDeliveryRequest) (BridgeTestDeliveryRecord, error) {
			return BridgeTestDeliveryRecord{
				Status:         "resolved",
				DeliveryTarget: DeliveryTargetRecord{BridgeInstanceID: "brg-1", PeerID: "peer-1", Mode: "reply"},
			}, nil
		},
	}
	deps := newTestDeps(t, client)
	runner := &stubRunner{}
	deps.newDaemon = func() (daemonRunner, error) { return runner, nil }

	tests := [][]string{
		{"agent", "info", "coder", "-o", "json"},
		{"agent", "soul", "inspect", "coder", "-o", "json"},
		{
			"agent",
			"soul",
			"write",
			"coder",
			"--file",
			soulBodyPath,
			"--expected-digest",
			"sha256:soul-old",
			"-o",
			"json",
		},
		{"agent", "soul", "delete", "coder", "--expected-digest", "sha256:soul-old", "-o", "json"},
		{
			"agent",
			"soul",
			"rollback",
			"coder",
			"--revision-id",
			"rev-soul-1",
			"--expected-digest",
			"sha256:soul-old",
			"-o",
			"json",
		},
		{"agent", "heartbeat", "inspect", "coder", "-o", "json"},
		{
			"agent",
			"heartbeat",
			"write",
			"coder",
			"--file",
			heartbeatBodyPath,
			"--expected-digest",
			"sha256:hb-old",
			"-o",
			"json",
		},
		{"agent", "heartbeat", "delete", "coder", "--expected-digest", "sha256:hb-old", "-o", "json"},
		{
			"agent",
			"heartbeat",
			"rollback",
			"coder",
			"--revision-id",
			"rev-hb-1",
			"--expected-digest",
			"sha256:hb-old",
			"-o",
			"json",
		},
		{"agent", "heartbeat", "status", "coder", "-o", "json"},
		{"network", "status", "-o", "json"},
		{"network", "peers", "builders", "-o", "json"},
		{"network", "channels", "-o", "json"},
		{
			"network",
			"send",
			"--session",
			"sess-1",
			"--channel",
			"builders",
			"--kind",
			"say",
			"--body",
			`{"text":"hello"}`,
			"-o",
			"json",
		},
		{"network", "inbox", "--session", "sess-1", "-o", "json"},
		{"observe", "events", "-o", "json"},
		{"observe", "events", "--follow", "-o", "json"},
		{"observe", "health", "-o", "json"},
		{"bridge", "get", "brg-1", "-o", "json"},
		{"bridge", "routes", "brg-1", "-o", "json"},
		{"bridge", "test-delivery", "brg-1", "--peer-id", "peer-1", "--mode", "reply", "-o", "json"},
		{"session", "soul", "refresh", "sess-1", "--expected-digest", "sha256:old", "-o", "json"},
		{"session", "health", "sess-1", "-o", "json"},
		{"session", "status", "sess-1", "-o", "json"},
		{"session", "inspect", "sess-1", "-o", "json"},
		{"session", "resume", "sess-1", "-o", "json"},
		{"session", "wait", "sess-1", "-o", "json"},
		{"session", "history", "sess-1", "-o", "json"},
		{"daemon", "status", "-o", "json"},
	}

	for _, args := range tests {
		if _, _, err := executeRootCommand(t, deps, args...); err != nil {
			t.Fatalf("executeRootCommand(%v) error = %v", args, err)
		}
	}

	if _, _, err := executeRootCommand(t, deps, "daemon", "start", "--foreground"); err != nil {
		t.Fatalf("daemon start --foreground error = %v", err)
	}
	if !networkChannelsCalled {
		t.Fatal("NetworkChannels() was not called")
	}
	if !getAgentSoulCalled || !putAgentSoulCalled || !deleteAgentSoulCalled || !rollbackAgentSoulCalled {
		t.Fatalf(
			"soul command routing flags = inspect:%v write:%v delete:%v rollback:%v, want all true",
			getAgentSoulCalled,
			putAgentSoulCalled,
			deleteAgentSoulCalled,
			rollbackAgentSoulCalled,
		)
	}
	if !getAgentHeartbeatCalled || !putAgentHeartbeatCalled || !deleteAgentHeartbeatCalled ||
		!rollbackAgentHeartbeatCalled || !getAgentHeartbeatStatusCalled {
		t.Fatalf(
			"heartbeat command routing flags = inspect:%v write:%v delete:%v rollback:%v status:%v, want all true",
			getAgentHeartbeatCalled,
			putAgentHeartbeatCalled,
			deleteAgentHeartbeatCalled,
			rollbackAgentHeartbeatCalled,
			getAgentHeartbeatStatusCalled,
		)
	}
	if !refreshSessionSoulCalled {
		t.Fatal("RefreshSessionSoul() was not called")
	}
	if !runner.ran {
		t.Fatal("daemon runner did not execute")
	}

	if wd, err := currentWorkingDirectory(deps); err != nil || wd != "/workspace/project" {
		t.Fatalf("currentWorkingDirectory() = %q, %v", wd, err)
	}

	if err := procutil.Signal(os.Getpid(), syscall.Signal(0)); err != nil {
		t.Fatalf("procutil.Signal(os.Getpid(), 0) error = %v", err)
	}
	if !procutil.Alive(os.Getpid()) {
		t.Fatal("procutil.Alive(os.Getpid()) = false, want true")
	}
}

func TestExecuteContextVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	code := ExecuteContext(context.Background(), []string{"version", "-o", "json"}, &stdout, &bytes.Buffer{})
	if code != 0 {
		t.Fatalf("ExecuteContext(version) code = %d, want 0", code)
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal(version) error = %v", err)
	}
	if _, ok := payload["Version"]; !ok {
		t.Fatalf("version payload = %#v, want Version field", payload)
	}
}

func TestDaemonStatusFallbackStartingAndStopped(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{
		daemonStatusFn: func(context.Context) (DaemonStatus, error) {
			return DaemonStatus{}, os.ErrNotExist
		},
	})
	info := aghdaemon.Info{PID: 42, StartedAt: fixedTestNow}
	deps.readDaemonInfo = func(string) (aghdaemon.Info, error) { return info, nil }
	deps.processAlive = func(pid int) bool { return pid == 42 }

	runtime, err := loadRuntimeContext(deps)
	if err != nil {
		t.Fatalf("loadRuntimeContext() error = %v", err)
	}

	status, err := daemonStatusFromDeps(context.Background(), deps, runtime)
	if err != nil {
		t.Fatalf("daemonStatusFromDeps(starting) error = %v", err)
	}
	if status.Status != "starting" {
		t.Fatalf("starting status = %q, want %q", status.Status, "starting")
	}

	deps.processAlive = func(int) bool { return false }
	status, err = daemonStatusFromDeps(context.Background(), deps, runtime)
	if err != nil {
		t.Fatalf("daemonStatusFromDeps(stopped) error = %v", err)
	}
	if status.Status != "stopped" {
		t.Fatalf("stopped status = %q, want %q", status.Status, "stopped")
	}
	if status.Network != nil {
		t.Fatalf("stopped network = %#v, want nil", status.Network)
	}
}

func TestWriteCommandOutputErrors(t *testing.T) {
	t.Parallel()

	if _, _, err := executeRootCommand(
		t,
		newTestDeps(t, &stubClient{}),
		"version",
		"-o",
		"bogus",
	); err == nil ||
		!strings.Contains(err.Error(), "invalid output format") {
		t.Fatalf("invalid output error = %v, want invalid output format", err)
	}
}

func TestDaemonStartRejectsNilDetachedProcess(t *testing.T) {
	t.Parallel()

	deps := newTestDeps(t, &stubClient{})
	deps.spawnDetached = func(context.Context, aghconfig.HomePaths) (daemonProcess, error) {
		return nil, nil
	}

	if _, _, err := executeRootCommand(
		t,
		deps,
		"daemon",
		"start",
		"-o",
		"json",
	); err == nil ||
		!strings.Contains(err.Error(), "detached daemon process is required") {
		t.Fatalf("daemon start nil detached process error = %v, want detached daemon process is required", err)
	}
}
